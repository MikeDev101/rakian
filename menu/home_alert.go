package menu

import (
	"context"
	"sync"
	"time"

	"gfx"
	"misc"
	"timers"
)

type HomeAlert struct {
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

func (m *Menu) NewHomeAlert() MenuInstance {
	return &HomeAlert{
		parent: m,
	}
}

func (instance *HomeAlert) Label() string {
	return "Home Menu"
}

func (instance *HomeAlert) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *HomeAlert) ConfigureWithArgs(args ...any) {
	// Unused
	instance.Configure()
}

func (instance *HomeAlert) Pause() {
	Pause(instance)
}

func (instance *HomeAlert) Stop() {
	Stop(instance)
}

func (instance *HomeAlert) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *HomeAlert) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *HomeAlert) Cleanup() {}

func (instance *HomeAlert) Save() {}

func (instance *HomeAlert) Load() {}

func (instance *HomeAlert) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *HomeAlert) GetStackIndex() int {
	return instance.stackIndex
}

func (instance *HomeAlert) render() {
	m := instance.parent
	display := m.Display

	defer display.Unlock()
	display.Lock()

	display.Clear(display.Primary())
	m.RenderStateCommon()
	m.RenderClock()

	font := display.Use_Font_Small_Bold()

	// TODO: make this dynamic
	label := "1 message received"

	display.DrawTextWrapped(8, 8, 78, 38, font, label, false, gfx.AlignNone, gfx.AlignNone)

	if m.Get("KeypadLocked").(bool) {
		display.DrawImageAligned(display.Use_Sprites()["keypad_locked"], 8, 0, gfx.AlignRight, gfx.AlignBelow)
		m.RenderFooter("Unlock", true)
	} else {
		m.RenderFooter("Read", true)
	}

	display.Render()
}

func (instance *HomeAlert) Run() {
	m := instance.parent
	if !instance.configured {
		panic("Attempted to call (*HomeAlert).Run() before (*HomeAlert).Configure()!")
	}

	instance.render()

	if m.Get("BeepOnly").(bool) || m.Get("CanRing").(bool) {
		instance.wg.Go(func() {
			misc.PlayBeep(m.Player, instance.ctx)
		})
	}

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
						if !keypad_locked {
							go instance.parent.PlayKey()
							// TODO: jump to message inbox
							go instance.parent.Pop()
							// go instance.parent.ToMenu("keypad_unlock")
							return
						}
					case 'C':
						if !keypad_locked {
							go instance.parent.PlayKey()
							go instance.parent.Pop()
							// return to home screen
							return
						}
					default:
						if !keypad_locked {
							// stub
							go instance.parent.PlayKey()
							// go instance.parent.PushWithArgs("dialer", evt.Key)
						}
					}
				}
			}
		}
	}()
}
