package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	// "hardware_keypad"

	"misc"
	"sim7600x"
	"simulator"

	// "pcd8544"

	"tones"

	"golang.org/x/sys/unix"
)

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

	/// Init go-networkmanager
	nm, wifi_device := InitNetworkManager()

	// Init db
	database := InitDB()

	/*
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
		defer rpio.SpiEnd(rpio.Spi0) */

	// Initialize software interfaces
	vio := simulator.New()
	display := vio.Display()

	lcd_powerdown := func() {
		display.Clear(display.Primary())
		display.Render()
		display.Off()
		display.Stop()
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
	phone := sim7600x.New(debug, database)
	// hardware_kp := hardware_keypad.New()
	rawKeypadEvents := vio.Run(ctx, debug)

	// Run main thread
	go main_thread(ctx, sigs, database, debug, rawKeypadEvents, display, phone, player, global_quit, nm, wifi_device)
	display.Start()
}
