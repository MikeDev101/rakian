package menu

import (
	"context"
	"sync"
	"time"

	"gfx"
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
	stackIndex  int
}

func (m *Menu) NewHomeMenu() MenuInstance {
	return &HomeMenu{
		parent: m,
	}
}

func (instance *HomeMenu) Label() string {
	return "Home Menu"
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

func (instance *HomeMenu) Pause() {
	Pause(instance)
}

func (instance *HomeMenu) Stop() {
	Stop(instance)
}

func (instance *HomeMenu) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *HomeMenu) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *HomeMenu) Cleanup() {}

func (instance *HomeMenu) Save() {}

func (instance *HomeMenu) Load() {}

func (instance *HomeMenu) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *HomeMenu) GetStackIndex() int {
	return instance.stackIndex
}

func (instance *HomeMenu) render() {
	m := instance.parent
	display := m.Display

	defer display.Unlock()
	display.Lock()

	display.Clear(display.Primary())
	m.RenderStateCommon()
	m.RenderClock()

	if m.Phone.OK() {
		if m.Phone.IsSOS() {
			display.DrawTextAligned(42, 8, display.Use_Font_Small_Plain(), "SOS", false, gfx.AlignCenter, gfx.AlignBelow)
		} else if m.Phone.IsFlightMode() {
			display.DrawTextAligned(42, 8, display.Use_Font_Small_Plain(), "Flight mode", false, gfx.AlignCenter, gfx.AlignBelow)
		} else if !m.Phone.IsRegistered() {
			display.DrawTextAligned(42, 8, display.Use_Font_Small_Bold(), "No service", false, gfx.AlignCenter, gfx.AlignBelow)
		} else {
			display.DrawTextAligned(42, 8, display.Use_Font_Small_Bold(), m.Phone.GetCarrier(), false, gfx.AlignCenter, gfx.AlignBelow)
		}

		if m.Phone.IsRoaming() {
			display.DrawTextAligned(42, 17, display.Use_Font_Small_Plain(), "Roaming", false, gfx.AlignCenter, gfx.AlignBelow)
		}
	} else {
		display.DrawTextAligned(42, 8, display.Use_Font_Small_Plain(), "Modem error", false, gfx.AlignCenter, gfx.AlignBelow)
	}

	if m.Get("KeypadLocked").(bool) {
		display.DrawImageAligned(display.Use_Sprites()["keypad_locked"], 8, 0, gfx.AlignRight, gfx.AlignBelow)
		m.RenderFooter("Unlock", true)
	} else {
		m.RenderFooter("Menu", true)
	}

	display.Render()
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
				if !instance.parent.Display.IsOn() {
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
					instance.parent.Keypad.KeyLightsOn()

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
