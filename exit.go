package main

import (
	"log"
	"misc"
	"os"
	"time"
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
