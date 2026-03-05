package menu

import (
	"context"
	"log"
	"modem"
	"sync"
	"time"

	"gfx"
	"misc"
	"timers"
)

type RingMenu struct {
	ctx         context.Context
	configured  bool
	cancelFn    context.CancelFunc
	parent      *Menu
	wg          sync.WaitGroup
	batt_flash  bool
	data_flash  bool
	render_loop *timers.ResettableTimer
	ring_tstamp time.Time
	ring_shown  bool
	call        *modem.Call
	stackIndex  int
}

func (m *Menu) NewRingMenu() MenuInstance {
	return &RingMenu{
		parent: m,
	}
}

func (instance *RingMenu) Label() string {
	return "Ring Menu"
}

func (instance *RingMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *RingMenu) ConfigureWithArgs(args ...any) {
	if len(args) == 0 {
		panic("(*RingMenu).ConfigureWithArgs() requires a *modem.Call argument")
	}
	call, ok := args[0].(*modem.Call)
	if !ok {
		panic("(*RingMenu).ConfigureWithArgs() argument must be *modem.Call")
	}
	instance.call = call
	instance.Configure()
}

func (instance *RingMenu) Pause() {
	Pause(instance)
}

func (instance *RingMenu) Stop() {
	Stop(instance)
}

func (instance *RingMenu) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *RingMenu) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *RingMenu) Cleanup() {
	instance.parent.Timers["keypad"].Restart()
	instance.parent.Timers["screensaver"].Restart()
}

func (instance *RingMenu) Save() {}

func (instance *RingMenu) Load() {}

func (instance *RingMenu) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *RingMenu) GetStackIndex() int {
	return instance.stackIndex
}

func (instance *RingMenu) Run() {
	m := instance.parent
	if !instance.configured {
		panic("Attempted to call (*RingMenu).Run() before (*RingMenu).Configure()!")
	}

	if instance.call == nil {
		panic("Attempted to call (*RingMenu).Run() without a call!")
	}

	// Disable screensaver
	go m.Timers["screensaver"].Stop()
	m.Keypad.KeyLightsOn()

	if m.Get("BeepOnly").(bool) {
		instance.wg.Go(func() {
			for {
				select {
				case <-instance.ctx.Done():
					return
				default:
					misc.PlayBeep(m.Player, instance.ctx)
				}
			}
		})

	} else if m.Get("CanRing").(bool) {
		instance.wg.Go(func() {
			for {
				select {
				case <-instance.ctx.Done():
					return
				default:
					misc.PlayRingtone(m.Player, instance.ctx)
				}
			}
		})
	}

	// Main render loop
	instance.wg.Go(func() {
		for {
			select {
			case <-instance.ctx.Done():
				return

			case <-time.After(100 * time.Millisecond):
				instance.render()
			}
		}
	})

	// Input loop
	instance.wg.Go(func() {
		for {
			select {
			case <-instance.ctx.Done():
				return

			case <-instance.call.Ended:
				go m.Pop()
				return

			case evt, ok := <-m.KeypadEvents:
				if !ok {
					return
				}

				if evt.State {
					m.Timers["keypad"].Reset()
					m.Keypad.KeyLightsOn()
					switch evt.Key {
					case 'S':
						go m.PlayKey()
						if m.Phone.OK() {
							res := m.Phone.AnswerCall(instance.call)
							if res != nil {
								log.Printf("⚠️  Failed to answer call: %v", res)
							} else {
								idx := instance.GetStackIndex()
								if idx > 0 && m.GetMenuKeyAt(idx-1) == "phone" {
									go m.PopWithArgs(instance.call)
								} else {
									go m.ToMenuWithArgs("phone", instance.call)
								}
								return
							}
						}
					case 'C':
						go m.PlayKey()
						if m.Phone.OK() {
							res := m.Phone.HangupCall(instance.call)
							if res != nil {
								log.Printf("⚠️ Failed to hang up call: %v", res)
							} else {
								go m.Pop()
								return
							}
						}
					}
				}
			}
		}
	})
}

func (instance *RingMenu) render() {
	m := instance.parent
	display := m.Display

	defer display.Unlock()
	display.Lock()

	display.Clear(display.Primary())
	m.RenderStateCommon()

	display.DrawTextWrapped(8, 8, 78, 16, display.Use_Font_Small_Plain(), instance.call.Number, false, gfx.WrapRight, gfx.WrapUp)

	if time.Since(instance.ring_tstamp) > 500*time.Millisecond {
		instance.ring_tstamp = time.Now()
		instance.ring_shown = !instance.ring_shown
	}

	if instance.ring_shown {
		display.DrawTextAligned(8, 38, display.Use_Font_Small_Bold(), "calling", false, gfx.AlignNone, gfx.AlignAbove)
	}

	m.RenderFooter("Answer", true)
	display.Render()
}
