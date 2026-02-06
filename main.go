package main

import (
	"fmt"
	"image"
	"os"
	"os/signal"
	"syscall"
	"log"
	"time"
	"context"

	"phone"
	"menu"
	"tones"
	"keypad"
	"misc"
	"timers"
	"sh1107"
)

// go build -ldflags "-X 'main.DEBUG_MODE=false'" .
var DEBUG_MODE string = "true"
var FW_VERSION string = "0.1.12 (2.5.2026)"
var EXIT_MODE  uint8 = 0 // 0 - none, 1 - shutdown, 2 - reboot, 3 - soft restart

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

	// WiFi status sprites
	"wifi/0",
	"wifi/1",
	"wifi/2",
	"wifi/3",
	"wifi/4",
	"wifi/5",
	"wifi/6",
	"wifi/7",
	"wifi/off",
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
	
	// Configure main-level scoped values
	var VeryLowBattChan = make(chan bool, 1)
	var LowBattChan = make(chan bool, 1)
	var DeadBattChan = make(chan bool, 1)
	var lastLowBattTime   time.Time
	var lastVeryLowBattTime time.Time
	
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
	
	// Create a global contexr
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
	go misc.PlayBoot(player, ctx)
	
	display.Load_Font_Time()
	display.Load_Font8_Bold()
	display.Load_Font8_Normal()
	display.Load_Font16()
	
	if DEBUG_MODE == "true" {
		display.DrawTextAligned(64, 20, display.Use_Font8_Bold(), "DEBUG MODE", false, sh1107.AlignCenter, sh1107.AlignCenter)	
	}
	display.DrawTextAligned(64, 82, display.Use_Font8_Normal(), "v" + FW_VERSION, false, sh1107.AlignCenter, sh1107.AlignCenter)
	
	// Show loading bar while loading sprites
	display.DrawProgressBar(0, 110, 128, 10, 0)
	sprites := make(map[string]image.Image, len(SPRITE_LIST))
	for _, key := range SPRITE_LIST {
		
		loaded, err := sh1107.LoadSprite(fmt.Sprintf("sprites/%s.bmp", key))
		if err != nil {
			log.Fatalf("⚠️ Failed to load sprite %s: %v", key, err)
		}
		sprites[key] = loaded
	}
	display.DrawProgressBar(0, 110, 128, 10, 1)
	
	time.Sleep(time.Second)
	display.Clear(sh1107.Black)
	display.Render()
	
	// Initialize menu system
	menus := menu.Init(ctx, display, sprites, modem, player, global_quit, keypadEvents)
	
	// Configure storage
	misc.GlobalStorage = menus.GlobalStorage
	
	// Setup global required keys
	menus.GlobalStorage["CanVibrate"] = true
	menus.GlobalStorage["CanRing"] = true
	menus.GlobalStorage["BeepOnly"] = false
	menus.GlobalStorage["InitialKey"] = ' '
	menus.GlobalStorage["BatteryOK"] = true
	menus.GlobalStorage["BatteryVoltage"] = ""
	menus.GlobalStorage["BatteryPercent"] = 0
	menus.GlobalStorage["BatteryScaledPercent"] = 0
	
	// Set initial WiFi status values
	connected, ssid, strength, ipaddr := misc.GetWiFiStatus()
	menus.Set("WiFi_Connected", connected)
	menus.Set("WiFi_SSID", ssid)
	menus.Set("WiFi_Strength", strength)
	menus.Set("WiFi_IP", ipaddr)
	
	// Configure WiFi status thread
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
		go func () {
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
				
				case <-modem.CallEndChan:
					go func() {
						menus.Pop()
						menus.Pop()
					}()
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
					if now.Sub(lastVeryLowBattTime) >= 10 * time.Minute {
						lastVeryLowBattTime = now
						log.Print("🪫 VERY LOW BATTERY")
						select {
						case VeryLowBattChan <- true:
						default:
						}
					}
				} else if capacity <= 25 {
					if now.Sub(lastLowBattTime) >= 10 * time.Minute {
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
	
	// Configure power event handlers
	go func () {
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
	
	// Configure timers
	menus.Timers["oled"] = timers.New(ctx, 10 * time.Second, false, func() {
		menus.Push("screensaver")
	})
	menus.Timers["keypad"] = timers.New(ctx, 5 * time.Second, false, func() {
		misc.KeyLightsOff()
	})
	
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
