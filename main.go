package main

import (
	"context"
	"gfx"
	"log"
	"os"
	"os/signal"
	"pcd8544"
	"syscall"
	"time"

	"db"
	"keypad"
	"menu"
	"misc"
	"modem"

	"github.com/stianeikeland/go-rpio/v4"
	"golang.org/x/sys/unix"

	"timers"
	"tones"

	"github.com/Wifx/gonetworkmanager/v3"
	"github.com/glebarez/sqlite"
	"github.com/godbus/dbus/v5"
	"gorm.io/gorm"
)

// go build -ldflags "-X 'main.DEBUG_MODE=false'" .
var DEBUG_MODE string = "true"
var FW_VERSION string = "0.1.20 (3.2.2026)"
var EXIT_MODE uint8 = 0 // 0 - none, 1 - shutdown, 2 - reboot, 3 - soft restart
const WLAN_DEVICE = "wlan0"
const DB_PATH = "/root/rakian/kvstore.db"

const (
	EXIT_SHUTDOWN = 1
	EXIT_REBOOT   = 2
	EXIT_RESTART  = 3
)

func exit() {
	if DEBUG_MODE == "true" {
		misc.EnablePowerbutton()
	}

	if r := recover(); r != nil {
		panic(r)
	}

	// DO NOT TOUCH
	if DEBUG_MODE == "true" {
		log.Println("👋 Goodbye")
		os.Exit(0)
	} else {
		time.Sleep(500 * time.Millisecond)
		switch EXIT_MODE {
		case EXIT_SHUTDOWN:
			misc.Shutdown()
		case EXIT_REBOOT:
			misc.HardReboot()
		case EXIT_RESTART:
			misc.SoftReboot()
		}
	}
}

func main() {

	// Handle system exit
	defer exit()
	debug := (DEBUG_MODE == "true")
	if debug {
		misc.DisablePowerbutton()
	}

	// Setup crash logging in deploy mode
	if !debug {
		if f, err := os.OpenFile("crash.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err != nil {
			log.Println("⚠️ Failed to open crash log:", err)
		} else {
			// Redirect stderr to the file to capture panic stack traces
			// Note: This will also redirect all log.Println output to the file
			unix.Dup2(int(f.Fd()), int(os.Stderr.Fd()))
		}
	}

	// Configure main-level scoped values
	var BatteryChargingChan = make(chan bool, 1)
	var BatteryChargedChan = make(chan bool, 1)
	var VeryLowBattChan = make(chan bool, 1)
	var LowBattChan = make(chan bool, 1)
	var DeadBattChan = make(chan bool, 1)
	var chargingBattShown = false
	var fullBattShown = false
	var lastLowBattTime time.Time
	var lastVeryLowBattTime time.Time

	/* Create new instance of gonetworkmanager */
	nm, err := gonetworkmanager.NewNetworkManager()
	if err != nil {
		log.Println(err)
		return
	}

	devices, err := nm.GetDevices()
	if err != nil {
		log.Println(err)
		return
	}

	var wifi_device_raw dbus.ObjectPath
	for _, device := range devices {

		device_interface, err := device.GetPropertyInterface()
		if err != nil {
			panic(err)
		}

		if device_interface == WLAN_DEVICE {
			wifi_device_raw = device.GetPath()
			break
		}
	}

	if wifi_device_raw == "" {
		panic("No wifi device found")
	}

	wifi_device, err := gonetworkmanager.NewDeviceWireless(wifi_device_raw)
	if err != nil {
		panic(err)
	}

	// Failsafe
	enabled, _ := nm.GetPropertyWirelessEnabled()
	if !enabled {
		nm.SetPropertyWirelessEnabled(true)
	}

	// Init db
	database, err := gorm.Open(sqlite.Open(DB_PATH), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	database.AutoMigrate(&db.KVStore{})
	database.AutoMigrate(&db.Contacts{})
	database.AutoMigrate(&db.Messages{})
	database.AutoMigrate(&db.CallSessionEvents{})
	database.AutoMigrate(&db.CallStateEvents{})

	// Initialize rpio
	if err := rpio.Open(); err != nil {
		log.Fatalf("⚠️ Failed to initialize rpio: %v", err)
	}
	defer rpio.Close()

	// Define the control pins (DC, RST)
	dc := rpio.Pin(9)
	rst := rpio.Pin(7)

	// Initialize the display
	display := pcd8544.New(dc, rst)
	defer rpio.SpiEnd(rpio.Spi0)

	lcd_powerdown := func() {
		display.Clear(display.Primary())
		display.Render()
		display.Off()
	}
	defer lcd_powerdown()

	// Create a global context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	global_quit := func(mode uint8) {
		log.Println("👋 Global quit raised")
		EXIT_MODE = mode
		cancel()
	}

	// Create signal handlers for interrupts or shutdown requests
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	// Initialize components
	player := tones.New()
	phone := modem.Run(debug, database)
	hardware_kp := keypad.New_Hardware_Keypad()
	rawKeypadEvents := hardware_kp.Run(ctx, debug)

	// Initialize keypad and wrap events to detect long-press on power button
	keypadEvents := make(chan *keypad.KeypadEvent, 10)
	go func() {
		var powerTimer *time.Timer
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-rawKeypadEvents:
				if !ok {
					return
				}
				if evt.Key == 'P' {
					if evt.State {
						powerTimer = time.AfterFunc(5*time.Second, func() {
							global_quit(EXIT_SHUTDOWN)
						})
					} else if powerTimer != nil {
						powerTimer.Stop()
					}
				}
				select {
				case keypadEvents <- evt:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Boot logo
	logo, err := gfx.LoadSprite("sprites/logo.bmp")
	if err != nil {
		log.Fatalf("⚠️ Failed to load logo: %v", err)
	}

	display.On()
	misc.KeyLightsOn()

	// Load sprites
	display.Load_Sprites()

	// Load animations
	display.Load_Animations()

	// Initialize menu system
	menus := menu.Init(
		ctx,
		(debug),
		display,
		phone,
		player,
		global_quit,
		keypadEvents,
		database,
		nm,
		wifi_device,
	)

	// Setup global required keys
	menus.CreateOrLoadPersist("CanVibrate", true)
	menus.CreateOrLoadPersist("CanRing", true)
	menus.CreateOrLoadPersist("BeepOnly", false)
	menus.CreateOrLoadPersist("APN", "vzwinternet")

	// Play boot chime
	if menus.Get("CanRing").(bool) && !menus.Get("BeepOnly").(bool) {
		log.Println("🎵 Playing boot chime...")
		go misc.PlayBoot(player, ctx)
	}

	// Play boot animation
	display.PlayAnimation(ctx, "boot", 0, 0, gfx.AlignNone, gfx.AlignNone)
	time.Sleep(1 * time.Second)

	// Load fonts
	display.Load_Font_Tiny()
	display.Load_Font_Small_Bold()
	display.Load_Font_Small_Plain()
	display.Load_Font_Large_Bold()

	// Register menus
	menus.Register("power", menus.NewPowerMenu())
	menus.Register("home", menus.NewHomeMenu())
	menus.Register("home_selection", menus.NewHomeSelectionMenu())
	menus.Register("dialer", menus.NewDialerMenu())
	menus.Register("phone", menus.NewPhoneMenu())
	menus.Register("ring", menus.NewRingMenu())
	menus.Register("dummy", menus.NewDummyMenu())
	menus.Register("screensaver", menus.NewScreensaver())
	menus.Register("low_battery", menus.NewLowBatteryAlert())
	menus.Register("dead_battery", menus.NewDeadBatteryAlert())
	menus.Register("very_low_battery", menus.NewVeryLowBatteryAlert())
	menus.Register("battery_charging", menus.NewBatteryChargingAlert())
	menus.Register("battery_charged", menus.NewBatteryChargedAlert())
	menus.Register("calculator", menus.NewCalculatorMenu())
	menus.Register("settings", menus.NewSettingsMenu())
	menus.Register("phonebook", menus.NewPhonebookMenu())
	menus.Register("keypad_unlock", menus.NewKeypadUnlockMenu())

	// Set runtime keys
	menus.Set("DebugMode", (debug))
	menus.Set("FirmwareVersion", FW_VERSION)
	menus.Set("BatteryOK", true)
	menus.Set("BatteryVoltage", "")
	menus.Set("BatteryPercent", 0)
	menus.Set("BatteryScaledPercent", 0)
	menus.Set("BatteryCharging", false)
	menus.Set("BluetoothEnabled", false)
	menus.Set("KeypadLocked", false)

	// Set initial WiFi status values
	connected, ssid, strength, ipaddr := misc.GetWiFiStatus()
	menus.Set("WiFi_Connected", connected)
	menus.Set("WiFi_SSID", ssid)
	menus.Set("WiFi_Strength", strength)
	menus.Set("WiFi_IP", ipaddr)

	// Set initial data enabled state
	apn := menus.Get("APN").(string)
	menus.Set("WasDataEnabled", misc.CheckCellularData(apn))

	// Update WiFi state
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				connected, ssid, strength, ipaddr = misc.GetWiFiStatus()
				menus.Set("WiFi_Connected", connected)
				menus.Set("WiFi_SSID", ssid)
				menus.Set("WiFi_Strength", strength)
				menus.Set("WiFi_IP", ipaddr)
			}
		}
	}()

	// Set initial bluetooth state
	bt_enabled := misc.IsBluetoothEnabled()
	menus.Set("BluetoothEnabled", bt_enabled)

	// Update bluetooth state
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				menus.Set("BluetoothEnabled", misc.IsBluetoothEnabled())
			}
		}
	}()

	// Handle modem events
	if phone.OK {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return

				case call := <-phone.RingingChan:
					go menus.ToMenuWithArgs("ring", call)
					misc.KeyLightsOn()
					menus.Timers["keypad"].Restart()
					menus.Timers["screensaver"].Restart()
				}
			}
		}()
	}

	// Monitor Battery & Charging
	go func() {
		var lastChargingState bool
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				voltage, capacity, capacity_scaled, read_err := misc.GetBatteryStatus()

				if read_err != nil {
					menus.Set("BatteryOK", false)
					return
				}

				charging := misc.GetChargingStatus()

				menus.Set("BatteryOK", true)
				menus.Set("BatteryVoltage", voltage)
				menus.Set("BatteryPercent", capacity)
				menus.Set("BatteryScaledPercent", capacity_scaled)

				now := time.Now()

				if charging && !lastChargingState {
					lastChargingState = true
					if capacity != 100 {
						if !chargingBattShown {
							chargingBattShown = true
							log.Println("🪫 CHARGING")
							menus.Set("BatteryCharging", true)
							BatteryChargingChan <- true
						}
					}
				} else if !charging && lastChargingState {
					lastChargingState = false
					log.Println("🪫 DISCHARGING")
					menus.Set("BatteryCharging", false)
					if fullBattShown {
						fullBattShown = false
					}
					if chargingBattShown {
						chargingBattShown = false
					}

					// Reset lastLowBattTime and lastVeryLowBattTime
					now := time.Now()
					lastLowBattTime = now.Add(-10 * time.Minute)
					lastVeryLowBattTime = now.Add(-10 * time.Minute)
				}

				if capacity == 100 {
					if !fullBattShown && charging {
						fullBattShown = true
						log.Print("🪫 FULL BATTERY")
						menus.Set("BatteryCharging", false)
						BatteryChargedChan <- true
					}
					continue
				}

				if charging {
					if !chargingBattShown {
						chargingBattShown = true
						menus.Set("BatteryCharging", true)
					}
					continue
				}

				if capacity <= 1 {
					log.Print("🪫 BATTERY EMPTY")
					DeadBattChan <- true
					return

				} else if capacity <= 5 {
					if now.Sub(lastVeryLowBattTime) >= 10*time.Minute {
						lastVeryLowBattTime = now
						log.Print("🪫 VERY LOW BATTERY")
						VeryLowBattChan <- true
					}
				} else if capacity <= 25 {
					if now.Sub(lastLowBattTime) >= 10*time.Minute {
						lastLowBattTime = now
						log.Print("🪫 LOW BATTERY")
						LowBattChan <- true
					}
				}
			}
		}
	}()

	/*
		// Show alert if there's something wrong with the SIM state
		if modem != nil && !modem.SimCardInserted {
			// menus.RenderAlert("prohibited", []string{"No SIM", "card", "inserted."})
			if menus.Get("CanRing").(bool) || menus.Get("BeepOnly").(bool) {
				// go menus.PlayAlert()
			}
			time.Sleep(3 * time.Second)
		} */

	// Persist screen for a moment
	time.Sleep(time.Second)
	display.Clear(display.Primary())
	display.Render()

	// Configure timers
	menus.Timers["screensaver"] = timers.New(ctx, 30*time.Second, false, func() {
		menus.Push("screensaver")
	})
	menus.Timers["keypad"] = timers.New(ctx, 10*time.Second, false, func() {
		misc.KeyLightsOff()
	})

	// Run home menu
	menus.Push("home")

	// Configure power event handlers
	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case <-BatteryChargedChan:
				go menus.ToMenu("battery_charged")
				misc.KeyLightsOn()
				menus.Timers["keypad"].Restart()
				menus.Timers["screensaver"].Restart()

			case <-BatteryChargingChan:
				go menus.ToMenu("battery_charging")
				misc.KeyLightsOn()
				menus.Timers["keypad"].Restart()
				menus.Timers["screensaver"].Restart()

			case <-VeryLowBattChan:
				go menus.ToMenu("very_low_battery")
				misc.KeyLightsOn()
				menus.Timers["keypad"].Restart()
				menus.Timers["screensaver"].Restart()

			case <-LowBattChan:
				go menus.ToMenu("low_battery")
				misc.KeyLightsOn()
				menus.Timers["keypad"].Restart()
				menus.Timers["screensaver"].Restart()

			case <-DeadBattChan:
				misc.KeyLightsOn()
				menus.Timers["keypad"].Stop()
				menus.Timers["screensaver"].Stop()
				go menus.ToMenu("dead_battery")
			}
		}
	}()

	// Main block
	if debug {
		log.Println("🚀 Starting main() loop - press Ctrl+C to exit")
	}
	select {
	case <-sigs:
		log.Println("ℹ️ Interrupt detected, exiting")
	case <-ctx.Done():
	}

	// Wait for all contexts to close
	menus.Shutdown()
	if phone != nil {
		phone.HangupAll()
	}
	display.Clear(display.Primary())
	display.DrawImage(logo, 20, 70)
	display.Render()
	display.On()
	misc.KeyLightsOn()
	time.Sleep(500 * time.Millisecond)
	player.Stop()
	if debug {
		log.Println("🛑 End of main() reached")
	}
}
