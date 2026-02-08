package menu

import (
	"context"
	"log"
	"sync"
	"time"

	"misc"
	"sh1107"
	"timers"
)

type RingMenu struct {
	ctx         context.Context
	configured  bool
	cancelFn    context.CancelFunc
	parent      *Menu
	wg          sync.WaitGroup
	batt_flash  bool
	render_loop *timers.ResettableTimer
}

func (m *Menu) NewRingMenu() *RingMenu {
	return &RingMenu{
		parent: m,
	}
}

func (instance *RingMenu) render() {
	display := instance.parent.Display
	display.Clear(sh1107.Black)
	instance.parent.RenderStatusBar(&instance.batt_flash)

	font := display.Use_Font8_Bold()
	display.DrawTextAligned(52, 105, font, "Answer", false, sh1107.AlignCenter, sh1107.AlignNone)
	display.DrawText(0, 60, font, instance.parent.Modem.CallState.Status, false)

	font = display.Use_Font16()
	display.DrawText(0, 40, font, instance.parent.Modem.CallState.PhoneNumber, false)

	display.Render()
}

func (instance *RingMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *RingMenu) ConfigureWithArgs(args ...any) {
	// Unused
	instance.Configure()
}

func (instance *RingMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*RingMenu).Run() before (*RingMenu).Configure()!")
	}

	if instance.parent.Get("CanVibrate").(bool) {
		instance.wg.Add(1)
		go func() {
			defer instance.wg.Done()
			for {
				select {
				case <-instance.ctx.Done():
					return
				default:
					misc.StartVibrate(instance.parent.Player, instance.ctx)
				}
			}
		}()
	}

	if instance.parent.Get("BeepOnly").(bool) {
		instance.wg.Add(1)
		go func() {
			defer instance.wg.Done()
			for {
				select {
				case <-instance.ctx.Done():
					return
				default:
					misc.PlayBeep(instance.parent.Player, instance.ctx)
				}
			}
		}()

	} else if instance.parent.Get("CanRing").(bool) {
		instance.wg.Add(1)
		go func() {
			defer instance.wg.Done()
			for {
				select {
				case <-instance.ctx.Done():
					return
				default:
					misc.PlayRingtone(instance.parent.Player, instance.ctx)
				}
			}
		}()
	}

	instance.wg.Add(1)
	go func() {
		defer instance.wg.Done()
		instance.render()
		for {
			select {
			case <-instance.ctx.Done():
				return
			case <-time.After(time.Second):
				if instance.parent.Display.IsOn {
					instance.render()
				}
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
						instance.parent.Modem.Answer()
						return

					case 'U':
					case 'D':
					case 'C':
						instance.parent.Modem.Hangup()
						return
					}
				}
			}
		}
	}()
}

func (instance *RingMenu) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Ring menu pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *RingMenu) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Ring menu stop timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}
