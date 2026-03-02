package menu

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"lcd"
	"misc"
)

type Screensaver struct {
	ctx        context.Context
	configured bool
	running    bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
}

func (m *Menu) NewScreensaver() *Screensaver {
	return &Screensaver{
		parent: m,
	}
}

func (instance *Screensaver) render() {
	display := instance.parent.Display
	display.Clear(lcd.White)
	font := display.Use_Font_Large_Bold()

	// Draw something
	// Read clock
	now := time.Now().In(time.Local)
	am_pm := "AM"
	if now.Hour() >= 12 {
		am_pm = "PM"
	}
	hour := now.Hour() % 12
	if hour == 0 {
		hour = 12
	}

	// Print clock
	clock_str := fmt.Sprintf("%2d:%02d %s", hour, now.Minute(), am_pm)

	// Trim leading spaces
	for clock_str[0] == ' ' {
		clock_str = clock_str[1:]
	}

	// Draw clock
	display.DrawTextAligned(42, 24, font, clock_str, false, lcd.AlignCenter, lcd.AlignCenter)

	display.Render()
}

func (instance *Screensaver) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *Screensaver) ConfigureWithArgs(args ...any) {
	// Unused
	instance.Configure()
}

func (instance *Screensaver) Run() {
	if !instance.configured {
		panic("Attempted to call (*Screensaver).Run() before (*Screensaver).Configure()!")
	}

	if instance.running {
		panic("Attempted to run multiple entries of (*Screensaver).Run()")
	}
	instance.running = true

	instance.render()

	// Calculate the initial delay to the next minute boundary
	now := time.Now()
	next := now.Truncate(time.Minute).Add(time.Minute)
	delay := next.Sub(now)
	if instance.parent.DebugMode {
		log.Printf("⏾ Initial screensaver delay: %s", delay)
	}

	instance.parent.Timers["screensaver"].Stop()

	// Switch modem mode
	/* if instance.parent.Modem != nil {
		instance.parent.Modem.SwitchToPowerSaveMode()
	} */

	// Start the screensaver loop
	instance.wg.Add(1)
	go func() {
		defer instance.wg.Done()
		for {
			select {
			case <-instance.ctx.Done():
				return
			case <-time.After(delay): // First delay for init

				// Reset the timer to be every minute
				delay = time.Minute

				// Render frame
				if instance.parent.DebugMode {
					log.Println("⏾ Screensaver timer raised")
				}
				instance.render()
			}
		}
	}()

	instance.wg.Add(1)
	go func() {
		defer instance.wg.Done()
		for {
			select {
			case <-instance.ctx.Done():
				return

			case evt := <-instance.parent.KeypadEvents:
				if evt.State {
					instance.parent.Timers["keypad"].Restart()
					instance.parent.Timers["screensaver"].Restart()
					instance.parent.Display.On()
					misc.KeyLightsOn()
					go instance.parent.PlayKey()
					go instance.parent.Pop()
					return
				}
			}
		}
	}()
}

func (instance *Screensaver) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Screensaver pause timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		instance.running = false
	}
}

func (instance *Screensaver) Stop() {
	instance.cancelFn()

	// Switch modem mode
	// if instance.parent.Modem != nil {
	// 	instance.parent.Modem.SwitchToNormalMode()
	//}

	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Screensaver stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		instance.running = false
	}
}
