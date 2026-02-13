package menu

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"misc"
	"sh1107"
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
	display.Clear(sh1107.Black)
	font := display.Use_Font16()

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
	display.DrawTextAligned(64, 55, font, clock_str, false, sh1107.AlignCenter, sh1107.AlignCenter)

	// Display battery
	font = display.Use_Font8_Normal()
	display.DrawTextAligned(64, 75, font, fmt.Sprintf("%d %%", instance.parent.Get("BatteryPercent").(int)), false, sh1107.AlignCenter, sh1107.AlignCenter)

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

	instance.parent.Display.SetBrightness(0.0)
	instance.parent.Timers["oled"].Stop()

	// Switch CPU and modem modes
	misc.SwitchToPowerSaveMode()
	instance.parent.Modem.SwitchToPowerSaveMode()

	// Start the screensaver loop
	instance.wg.Go(func() {
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
	})

	instance.wg.Go(func() {
		for {
			select {
			case <-instance.ctx.Done():
				misc.SwitchToNormalMode()
				return

			case evt := <-instance.parent.KeypadEvents:
				if evt.State {
					misc.SwitchToNormalMode()
					instance.parent.Timers["keypad"].Restart()
					instance.parent.Timers["oled"].Restart()
					instance.parent.Display.On()
					misc.KeyLightsOn()
					go instance.parent.PlayKey()
					go instance.parent.Pop()
					return
				}
			}
		}
	})
}

func (instance *Screensaver) Pause() {
	instance.cancelFn()
	instance.parent.Display.SetBrightness(1.0)
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Screensaver pause timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		instance.running = false
	}
}

func (instance *Screensaver) Stop() {
	instance.cancelFn()

	// Switch CPU and modem modes
	misc.SwitchToNormalMode()
	instance.parent.Modem.SwitchToNormalMode()

	// Restore brightness
	instance.parent.Display.SetBrightness(1.0)

	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Screensaver stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		instance.running = false
	}
}
