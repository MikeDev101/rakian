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

type PhoneMenu struct {
	ctx         context.Context
	configured  bool
	cancelFn    context.CancelFunc
	parent      *Menu
	wg          sync.WaitGroup
	batt_flash  bool
	data_flash  bool
	render_loop *timers.ResettableTimer
}

func (m *Menu) NewPhoneMenu() *PhoneMenu {
	return &PhoneMenu{
		parent: m,
	}
}

func (instance *PhoneMenu) render() {
	display := instance.parent.Display
	display.Clear(sh1107.Black)
	instance.parent.RenderStatusBar(&instance.batt_flash, &instance.data_flash)

	font := display.Use_Font8_Bold()
	display.DrawTextAligned(64, 105, font, "End", false, sh1107.AlignCenter, sh1107.AlignNone)
	display.DrawTextAligned(0, 65, font, instance.parent.Modem.CallState.Status, false, sh1107.AlignRight, sh1107.AlignNone)

	font = display.Use_Font16()
	display.DrawText(0, 45, font, instance.parent.Modem.CallState.PhoneNumber, false)

	if !instance.parent.Modem.CallState.StartTime.IsZero() {
		d := time.Since(instance.parent.Modem.CallState.StartTime)
		font = display.Use_Font_Time()
		display.DrawTextAligned(0, 80, font, fmt.Sprintf("%02d:%02d:%02d", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60), false, sh1107.AlignRight, sh1107.AlignNone)
	}

	display.Render()
}

func (instance *PhoneMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *PhoneMenu) ConfigureWithArgs(args ...any) {
	// Unused
	instance.Configure()
}

func (instance *PhoneMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*PhoneMenu).Run() before (*PhoneMenu).Configure()!")
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
						instance.parent.Modem.Hangup()
						return

					case 'U':
					case 'D':
					case 'C':
					default:
						instance.parent.Modem.EnterNumber(evt.Key)
					}
				}
			}
		}
	})
}

func (instance *PhoneMenu) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Phone menu pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *PhoneMenu) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Phone menu stop timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}
