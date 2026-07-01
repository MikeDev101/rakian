package main

import (
	"context"
	"fmt"
	"gfx"
	"keypad"
	"log"
	"menu"
	"misc"
	"modem"
	"os"
	"time"
	"timers"
	"tones"

	"github.com/Wifx/gonetworkmanager/v3"
	"gorm.io/gorm"
)

func main_thread(
	ctx context.Context,
	sigs <-chan os.Signal,
	database *gorm.DB,
	debug bool,
	kp keypad.Keypad,
	rawKeypadEvents <-chan *keypad.KeypadEvent,
	display gfx.Driver, phone modem.Modem,
	player *tones.Tones,
	global_quit func(uint8), nm gonetworkmanager.NetworkManager,
	wifi_device gonetworkmanager.DeviceWireless) {

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
	kp.KeyLightsOn()

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
		kp,
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
	menus.CreateOrLoadPersist("DivertOnBusy", false)
	menus.CreateOrLoadPersist("DivertAlways", false)
	menus.CreateOrLoadPersist("DivertOnNoAnswer", false)

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
	menus.Register("power", menus.NewPowerMenu)
	menus.Register("home", menus.NewHomeMenu)
	menus.Register("home_alert", menus.NewHomeAlert)
	menus.Register("home_selection", menus.NewHomeSelectionMenu)
	menus.Register("dialer", menus.NewDialerMenu)
	menus.Register("phone", menus.NewPhoneMenu)
	menus.Register("ring", menus.NewRingMenu)
	menus.Register("dummy", menus.NewDummyMenu)
	menus.Register("screensaver", menus.NewScreensaver)
	menus.Register("low_battery", menus.NewLowBatteryAlert)
	menus.Register("dead_battery", menus.NewDeadBatteryAlert)
	menus.Register("very_low_battery", menus.NewVeryLowBatteryAlert)
	menus.Register("battery_charging", menus.NewBatteryChargingAlert)
	menus.Register("battery_charged", menus.NewBatteryChargedAlert)
	menus.Register("calculator", menus.NewCalculatorMenu)
	menus.Register("settings", menus.NewSettingsMenu)
	menus.Register("phone_book", menus.NewPhonebookMenu)
	menus.Register("keypad_unlock", menus.NewKeypadUnlockMenu)

	// Stub Menus
	menus.Register("messages", menus.NewMessagesMenu)
	menus.Register("chat", menus.NewChatMenu)
	menus.Register("call_register", menus.NewCallRegisterMenu)
	menus.Register("profiles", menus.NewProfilesMenu)
	menus.Register("call_divert", menus.NewCallDivertMenu)
	menus.Register("apps", menus.NewAppsMenu)
	menus.Register("clock", menus.NewClockMenu)
	menus.Register("reminders", menus.NewRemindersMenu)
	menus.Register("tones", menus.NewTonesMenu)

	// Set runtime keys
	menus.Set("DebugMode", (debug))
	menus.Set("FirmwareVersion", FW_VERSION)
	menus.Set("BatteryOK", true)
	menus.Set("BatteryPercent", 100)
	menus.Set("BatteryScaledPercent", 4)
	menus.Set("BatteryCharging", false)
	menus.Set("BluetoothEnabled", false)
	menus.Set("KeypadLocked", false)
	menus.Set("PhoneActive", false)

	// TODO: check if there are any unread messages or voicemails
	menus.Set("NewMessage", false)
	menus.Set("NewVoicemail", false)

	// Show if diverts are enabled
	if menus.Get("DivertAlways").(bool) || menus.Get("DivertOnBusy").(bool) || menus.Get("DivertOnNoAnswer").(bool) {
		menus.Set("DivertsActive", true)
	} else {
		menus.Set("DivertsActive", false)
	}

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

	// Handle modem events via subscription
	if phone.OK() {
		subKey := phone.Subscribe(func(event modem.IMSEvent) error {
			switch event.Type {
			case modem.IMS_Call_Incoming:
				call, ok := event.Data.(*modem.Call)
				if !ok {
					return fmt.Errorf("invalid call data for incoming call event")
				}

				if menus.Get("PhoneActive").(bool) {
					return nil
				}

				log.Printf("📞 New incoming call (main thread event): %v", call)

				if len(phone.GetCalls()) > 2 {
					return nil
				}

				kp.KeyLightsOn()
				menus.Timers["keypad"].Restart()
				menus.Timers["screensaver"].Restart()
				go menus.PushWithArgs("ring", call)

			case modem.IMS_New_Message:
				menus.Set("NewMessage", true)
				go menus.Push("home_alert")

			case modem.IMS_New_Voicemail:
				menus.Set("NewVoicemail", true)
			}
			return nil
		})

		go func() {
			<-ctx.Done()
			phone.Unsubscribe(subKey)
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
				_, capacity, capacity_scaled, read_err := misc.GetBatteryStatus()

				if read_err != nil {
					menus.Set("BatteryOK", false)
					return
				}

				charging := misc.GetChargingStatus()

				menus.Set("BatteryOK", true)
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

	// Show alert if there's something wrong with the SIM state
	if phone != nil && !phone.OK() {
		menus.RenderAlert("prohibited", []string{"No SIM", "card", "inserted."})
		if menus.Get("CanRing").(bool) || menus.Get("BeepOnly").(bool) {
			go menus.PlayAlert()
		}
		time.Sleep(3 * time.Second)
	}

	// Persist screen for a moment
	time.Sleep(time.Second)
	display.Clear(display.Primary())
	display.Render()

	// Configure timers
	menus.Timers["screensaver"] = timers.New(ctx, 30*time.Second, false, func() {
		menus.Push("screensaver")
	})
	menus.Timers["keypad"] = timers.New(ctx, 10*time.Second, false, func() {
		kp.KeyLightsOff()
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
				kp.KeyLightsOn()
				menus.Timers["keypad"].Restart()
				menus.Timers["screensaver"].Restart()

			case <-BatteryChargingChan:
				go menus.ToMenu("battery_charging")
				kp.KeyLightsOn()
				menus.Timers["keypad"].Restart()
				menus.Timers["screensaver"].Restart()

			case <-VeryLowBattChan:
				go menus.ToMenu("very_low_battery")
				kp.KeyLightsOn()
				menus.Timers["keypad"].Restart()
				menus.Timers["screensaver"].Restart()

			case <-LowBattChan:
				go menus.ToMenu("low_battery")
				kp.KeyLightsOn()
				menus.Timers["keypad"].Restart()
				menus.Timers["screensaver"].Restart()

			case <-DeadBattChan:
				kp.KeyLightsOn()
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
		phone.Stop()
	}
	display.Clear(display.Primary())
	display.DrawImage(logo, 20, 70)
	display.Render()
	display.On()
	kp.KeyLightsOn()
	time.Sleep(500 * time.Millisecond)
	player.Stop()
	if debug {
		log.Println("🛑 End of main() reached")
	}
}
