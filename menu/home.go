package menu

import (
	"context"
	"log"
	"sync"
	"time"

	"lcd"
	"misc"
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

func (instance *HomeMenu) Label() string {
	return "Home Menu"
}

func (instance *HomeMenu) render() {
	m := instance.parent
	display := m.Display

	display.Clear(lcd.White)
	m.RenderStateCommon()

	if m.Phone.OK {
		if m.Phone.SOS {
			display.DrawTextAligned(42, 8, display.Use_Font_Small_Plain(), "SOS", false, lcd.AlignCenter, lcd.AlignBelow)
		} else if m.Phone.FlightMode {
			display.DrawTextAligned(42, 8, display.Use_Font_Small_Plain(), "Flight mode", false, lcd.AlignCenter, lcd.AlignBelow)
		} else {
			display.DrawTextAligned(42, 8, display.Use_Font_Small_Bold(), m.Phone.Carrier, false, lcd.AlignCenter, lcd.AlignBelow)
		}

		if m.Phone.Roaming {
			display.DrawTextAligned(42, 17, display.Use_Font_Small_Plain(), "Roaming", false, lcd.AlignCenter, lcd.AlignBelow)
		}
	}

	if m.Get("KeypadLocked").(bool) {
		m.RenderFooter("Unlock", true)
	} else {
		m.RenderFooter("Menu", true)
	}

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

	instance.render()

	// Main render loop
	instance.wg.Add(1)
	go func() {
		defer instance.wg.Done()
		for {
			select {
			case <-instance.ctx.Done():
				return

			case <-time.After(100 * time.Millisecond):
				if !instance.parent.Display.IsOn {
					continue
				}
				instance.render()
			}
		}
	}()

	// Input loop
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

				keypad_locked := instance.parent.Get("KeypadLocked").(bool)

				if evt.State {
					instance.parent.Timers["keypad"].Reset()
					instance.parent.Timers["screensaver"].Reset()
					instance.parent.Display.On()
					misc.KeyLightsOn()

					switch evt.Key {

					case 'P':
						if !keypad_locked {
							go instance.parent.PlayKey()
							go instance.parent.Push("power")
							return
						}
					case 'S':
						if keypad_locked {
							go instance.parent.PlayKey()
							go instance.parent.ToMenu("keypad_unlock")
							return
						}

						if !keypad_locked {
							go instance.parent.PlayKey()
							go instance.parent.Push("home_selection")
							return
						}
					case 'U':
						if !keypad_locked {
							go instance.parent.PlayKey()
							// TODO: cycle between different home menus
						}
					case 'D':
						if !keypad_locked {
							go instance.parent.PlayKey()
							// TODO: cycle between different home menus
						}
					case 'C':
						if !keypad_locked {
							go instance.parent.PlayKey()
						}
					default:
						if !keypad_locked {
							go instance.parent.PushWithArgs("dialer", evt.Key)
							return
						}
					}
				}
			}
		}
	}()
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
