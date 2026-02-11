package main

import (
	"context"
	"fmt"
	"image"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"db"
	"keypad"
	"menu"
	"misc"
	"phone"
	"sh1107"
	"timers"
	"tones"

	"github.com/Wifx/gonetworkmanager/v3"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// go build -ldflags "-X 'main.DEBUG_MODE=false'" .
var DEBUG_MODE string = "true"
var FW_VERSION string = "0.1.16 (2.11.2026)"
var EXIT_MODE uint8 = 0 // 0 - none, 1 - shutdown, 2 - reboot, 3 - soft restart
var SPRITE_LIST = []string{

	// Battery status sprites
	"battery/0",
	"battery/0_warn",
	"battery/1",
	"battery/2",
	"battery/3",
	"battery/4",
	"battery/5",
	"battery/6",
	"battery/7",
	"battery/8",
	"battery/9",
	"battery/10",
	"battery/unknown",

	// Cellular network sprites
	"cell/0",
	"cell/1",
	"cell/2",
	"cell/3",
	"cell/4",
	"cell/5",
	"cell/6",
	"cell/7",
	"cell/data_active",
	"cell/data_inactive",
	"cell/fault",
	"cell/locked",
	"cell/off",
	"cell/prohibit",
	"cell/sos",
	"cell/airplane",
	"cell/no_sim",

	// WiFi status sprites
	"wifi/0",
	"wifi/1",
	"wifi/2",
	"wifi/3",
	"wifi/4",
	"wifi/5",
	"wifi/6",
	"wifi/7",
	"wifi/connecting",
	"wifi/networks_found",
	"wifi/no_networks",
	"wifi/no_internet",

	// Home screen menu sprites
	"home/Calculator",
	"home/CallDivert",
	"home/CallRegister",
	"home/Clock",
	"home/Games",
	"home/Messages",
	"home/PhoneBook",
	"home/Settings",
	"home/Tones",

	// Misc sprites
	"alert",
	"ok",
	"prohibited",
	"low_battery",
	"very_low_battery",
	"dead_battery",
	"duck",
	"logo",
}

func exit() {
	// DO NOT TOUCH
	if DEBUG_MODE == "true" {
		log.Println("👋 Goodbye")
		os.Exit(0)
	} else {
		switch EXIT_MODE {
		case 1:
			misc.Shutdown()
		case 2:
			misc.HardReboot()
		case 3:
			misc.SoftReboot()
		}
	}
}

func main() {

	// Handle system exit
	defer exit()

	// Setup crash logging in deploy mode
	if DEBUG_MODE != "true" {
		if f, err := os.OpenFile("crash.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err != nil {
			log.Println("⚠️ Failed to open crash log:", err)
		} else {
			// Redirect stderr to the file to capture panic stack traces
			// Note: This will also redirect all log.Println output to the file
			unix.Dup2(int(f.Fd()), int(os.Stderr.Fd()))
		}
	}

	// Configure main-level scoped values
	var VeryLowBattChan = make(chan bool, 1)
	var LowBattChan = make(chan bool, 1)
	var DeadBattChan = make(chan bool, 1)
	var lastLowBattTime time.Time
	var lastVeryLowBattTime time.Time

	/* Create new instance of gonetworkmanager */
	nm, err := gonetworkmanager.NewNetworkManager()
	if err != nil {
		panic(err)
	}

	wifi_device, err := nm.GetDeviceByIpIface("wlan0")
	if err != nil {
		panic(err)
	}

	// Init db
	database, err := gorm.Open(sqlite.Open("/root/rakian/kvstore.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	database.AutoMigrate(&db.KVStore{})

	// Initialize the display
	display := sh1107.New(0x3c, 0, sh1107.UpsideDown, 128, 128)
	defer display.Close()

	if _, capacity, _, read_err := misc.GetBatteryStatus(); read_err == nil && capacity <= 1 {
		alert, err := sh1107.LoadSprite("sprites/battery_needs_charge.bmp")
		if err != nil {
			log.Fatalf("⚠️ Failed to load alert image: %v", err)
		}
		display.SetBrightness(100)
		display.Clear(sh1107.Black)
		display.DrawImageAligned(alert, 64, 74, sh1107.AlignCenter, sh1107.AlignCenter)
		display.Render()
		display.On()
		time.Sleep(5 * time.Second)
		EXIT_MODE = 1
		return
	} else if read_err != nil {
		// TODO: show diagnostic code
		log.Println(read_err)
		EXIT_MODE = 1
		return
	}

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
	keypadEvents := keypad.Run(ctx)
	modem := phone.Run()

	// Boot logo
	logo, err := sh1107.LoadSprite("sprites/logo.bmp")
	if err != nil {
		log.Fatalf("⚠️ Failed to load logo: %v", err)
	}
	display.SetBrightness(100)

	draw_logo := func() {
		display.Clear(sh1107.Black)
		display.DrawImage(logo, 20, 63)
		display.Render()
	}

	draw_logo()
	display.On()
	misc.KeyLightsOn()

	// Load sprites
	sprites := make(map[string]image.Image, len(SPRITE_LIST))
	for _, key := range SPRITE_LIST {

		loaded, err := sh1107.LoadSprite(fmt.Sprintf("sprites/%s.bmp", key))
		if err != nil {
			log.Fatalf("⚠️ Failed to load sprite %s: %v", key, err)
		}
		sprites[key] = loaded
	}

	// Initialize menu system
	menus := menu.Init(
		ctx,
		display,
		sprites,
		modem,
		player,
		global_quit,
		keypadEvents,
		database,
		nm,
		wifi_device,
	)

	// Setup global required keys
	menus.Set("DebugMode", (DEBUG_MODE == "true"))
	menus.Set("FirmwareVersion", FW_VERSION)
	menus.SetOrCreate("CanVibrate", false)
	menus.SetOrCreate("CanRing", false)
	menus.SetOrCreate("BeepOnly", false)
	menus.Set("InitialKey", ' ')
	menus.Set("BatteryOK", true)
	menus.Set("BatteryVoltage", "")
	menus.Set("BatteryPercent", 0)
	menus.Set("BatteryScaledPercent", 0)

	// Load fonts
	display.Load_Font_Time()
	display.Load_Font8_Bold()
	display.Load_Font8_Normal()
	display.Load_Font16()

	// Play boot chime
	if DEBUG_MODE != "true" {
		if menus.Get("CanRing").(bool) && !menus.Get("BeepOnly").(bool) {
			go misc.PlayBoot(player, ctx)
		}
		time.Sleep(2 * time.Second)
	}

	// Show version info
	if DEBUG_MODE == "true" {
		display.DrawTextAligned(64, 20, display.Use_Font8_Bold(), "DEBUG MODE", false, sh1107.AlignCenter, sh1107.AlignCenter)
	}
	display.DrawTextAligned(64, 82, display.Use_Font8_Normal(), "v"+FW_VERSION, false, sh1107.AlignCenter, sh1107.AlignCenter)
	display.Render()

	if DEBUG_MODE != "true" {
		time.Sleep(2 * time.Second)
	}

	// Failsafe
	enabled, _ := nm.GetPropertyWirelessEnabled()
	if !enabled {
		log.Println("WiFi was off, emergency re-enabling...")
		nm.SetPropertyWirelessEnabled(true)
		menus.RenderAlert("ok", []string{"WiFi", "failsafe", "triggered!"})
		go menus.PlayAlert()
		time.Sleep(5 * time.Second) // Give it a moment to breathe
	}

	// Set initial WiFi status values
	connected, ssid, strength, ipaddr := misc.GetWiFiStatus()
	menus.Set("WiFi_Connected", connected)
	menus.Set("WiFi_SSID", ssid)
	menus.Set("WiFi_Strength", strength)
	menus.Set("WiFi_IP", ipaddr)

	// Update WiFi state
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				connected, ssid, strength, ipaddr = misc.GetWiFiStatus()
				menus.Set("WiFi_Connected", connected)
				menus.Set("WiFi_SSID", ssid)
				menus.Set("WiFi_Strength", strength)
				menus.Set("WiFi_IP", ipaddr)
			}
		}
	}()

	// Handle modem events
	if modem != nil {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return

				case <-modem.RingingChan:
					go menus.ToMenu("ring")
					misc.KeyLightsOn()
					menus.Timers["keypad"].Stop()
					menus.Timers["oled"].Stop()

				case <-modem.CallStartChan:
					go menus.ToMenu("phone")
					menus.Timers["oled"].Restart()
					menus.Timers["keypad"].Restart()

				case <-modem.CallErrorChan:
					log.Println("⚠️ Call failed")
					go menus.RenderAlert("alert", []string{"Call", "failed."})
					menus.Timers["oled"].Restart()
					menus.Timers["keypad"].Restart()
					misc.KeyLightsOn()
					menus.PlayAlert()
					time.Sleep(2 * time.Second)
					modem.CallHandledChan <- true

				case <-modem.CallEndChan:
					go menus.ToStart()
					misc.KeyLightsOn()
					menus.Timers["oled"].Restart()
					menus.Timers["keypad"].Restart()
				}
			}
		}()
	}

	// Monitor Battery
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				voltage, capacity, capacity_scaled, read_err := misc.GetBatteryStatus()

				if read_err != nil {
					menus.Set("BatteryOK", false)
					return
				}

				menus.Set("BatteryOK", true)
				menus.Set("BatteryVoltage", voltage)
				menus.Set("BatteryPercent", capacity)
				menus.Set("BatteryScaledPercent", capacity_scaled)

				now := time.Now()

				if capacity <= 1 {
					log.Print("🪫 BATTERY EMPTY")
					DeadBattChan <- true
					return

				} else if capacity <= 5 {
					if now.Sub(lastVeryLowBattTime) >= 10*time.Minute {
						lastVeryLowBattTime = now
						log.Print("🪫 VERY LOW BATTERY")
						select {
						case VeryLowBattChan <- true:
						default:
						}
					}
				} else if capacity <= 25 {
					if now.Sub(lastLowBattTime) >= 10*time.Minute {
						lastLowBattTime = now
						log.Print("🪫 LOW BATTERY")
						select {
						case LowBattChan <- true:
						default:
						}
					}
				}
			}
		}
	}()

	// Show alert if there's something wrong with the SIM state
	if modem != nil && !modem.SimCardInserted {
		menus.RenderAlert("prohibited", []string{"No SIM", "card", "inserted."})
		time.Sleep(5 * time.Second)
	}

	// Persist screen for a moment
	time.Sleep(time.Second)
	display.Clear(sh1107.Black)
	display.Render()

	// Configure timers
	menus.Timers["oled"] = timers.New(ctx, 10*time.Second, false, func() {
		menus.Push("screensaver")
	})
	menus.Timers["keypad"] = timers.New(ctx, 5*time.Second, false, func() {
		misc.KeyLightsOff()
	})

	// Configure power event handlers
	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case <-VeryLowBattChan:
				go menus.ToMenu("very_low_battery")
				misc.KeyLightsOn()
				menus.Timers["keypad"].Restart()
				menus.Timers["oled"].Restart()

			case <-LowBattChan:
				go menus.ToMenu("low_battery")
				misc.KeyLightsOn()
				menus.Timers["keypad"].Restart()
				menus.Timers["oled"].Restart()

			case <-DeadBattChan:
				misc.KeyLightsOn()
				menus.Timers["keypad"].Stop()
				menus.Timers["oled"].Stop()
				go menus.ToMenu("dead_battery")
			}
		}
	}()

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
	menus.Register("calculator", menus.NewCalculatorMenu())
	menus.Register("selector", menus.NewSelector())
	menus.Register("settings", menus.NewSettingsMenu())
	menus.Register("phonebook", menus.NewPhonebookMenu())

	// Run home menu
	menus.Push("home")

	log.Println("Press CTRL+C to quit")
	select {
	case <-sigs:
		log.Println("Interrupt detected, exiting")
	case <-ctx.Done():
	}

	// Wait for all contexts to close
	menus.Shutdown()
	if modem != nil {
		modem.Hangup()
	}
	display.SetBrightness(0.0)
	display.Clear(sh1107.Black)
	display.DrawImage(logo, 20, 70)
	display.Render()
	display.On()
	misc.KeyLightsOn()
	time.Sleep(500 * time.Millisecond)
	player.Stop()
	log.Println("🛑 End of main() reached")
}
