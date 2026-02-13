package menu

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"misc"
	"sh1107"
	"timers"
)

type HomeMenu struct {
	ctx         context.Context
	configured  bool
	cancelFn    context.CancelFunc
	parent      *Menu
	wg          sync.WaitGroup
	batt_flash  bool
	data_flash  bool
	render_loop *timers.ResettableTimer
}

func (m *Menu) NewHomeMenu() *HomeMenu {
	return &HomeMenu{
		parent: m,
	}
}

func (instance *HomeMenu) render() {
	display := instance.parent.Display

	display.Clear(sh1107.Black)

	// Render status bar
	instance.parent.RenderStatusBar(&instance.batt_flash, &instance.data_flash)

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

	clock_str := fmt.Sprintf("%2d:%02d %s", hour, now.Minute(), am_pm)

	// Trim leading spaces
	for clock_str[0] == ' ' {
		clock_str = clock_str[1:]
	}

	// Draw clock
	font := display.Use_Font16()
	display.DrawTextAligned(64, 55, font, clock_str, false, sh1107.AlignCenter, sh1107.AlignNone)

	// Draw carrier info
	font = display.Use_Font8_Normal()
	var carrier_label string
	if instance.parent.Modem == nil {
		carrier_label = "No service"

	} else if instance.parent.Modem.FlightMode {
		carrier_label = "Airplane mode"

	} else if !instance.parent.Modem.SimCardInserted {
		carrier_label = "Insert SIM card"

	} else {
		carrier_label = instance.parent.Modem.Carrier
	}
	display.DrawTextAligned(64, 75, font, carrier_label, false, sh1107.AlignCenter, sh1107.AlignNone)

	// Draw menu hint
	font = display.Use_Font8_Bold()
	display.DrawTextAligned(64, 105, font, "Menu", false, sh1107.AlignCenter, sh1107.AlignNone)

	display.Render()
}

func (instance *HomeMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *HomeMenu) ConfigureWithArgs(args ...any) {
	// Unused
	instance.Configure()
}

func (instance *HomeMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*HomeMenu).Run() before (*HomeMenu).Configure()!")
	}

	// Battery icon blinker
	instance.wg.Go(func() {
		for {
			select {
			case <-instance.ctx.Done():
				return

			case <-time.After(time.Second):
				instance.batt_flash = !instance.batt_flash
			}
		}
	})

	// Data icon blinker
	instance.wg.Go(func() {
		instance.render()
		for {
			select {
			case <-instance.ctx.Done():
				return

			case <-time.After(500 * time.Millisecond):
				instance.data_flash = !instance.data_flash
			}
		}
	})

	// Main render loop
	instance.wg.Go(func() {
		for {
			select {
			case <-instance.ctx.Done():
				return

			case <-time.After(100 * time.Millisecond):
				if instance.parent.Display.IsOn {
					instance.render()
				}
			}
		}
	})

	// Input loop
	instance.wg.Go(func() {
		for {
			select {
			case <-instance.ctx.Done():
				return

			case evt, ok := <-instance.parent.KeypadEvents:
				if !ok {
					return
				}

				if evt.State {

					instance.parent.Timers["keypad"].Reset()
					instance.parent.Timers["oled"].Reset()
					instance.parent.Display.On()
					misc.KeyLightsOn()
					go instance.parent.PlayKey()

					switch evt.Key {

					case 'P':
						go instance.parent.Push("power")
						return

					case 'S':
						go instance.parent.Push("home_selection")
						return
					case 'U':
						// TODO: cycle between different home menus
					case 'D':
						// TODO: cycle between different home menus
					case 'C':
					default:
						instance.parent.Set("InitialKey", evt.Key)
						go instance.parent.Push("dialer")
						return
					}
				}
			}
		}
	})
}

func (instance *HomeMenu) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Home menu pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *HomeMenu) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Home menu stop timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}
