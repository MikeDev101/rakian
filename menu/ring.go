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
	call        *modem.Call
}

func (m *Menu) NewRingMenu() *RingMenu {
	return &RingMenu{
		parent: m,
	}
}

func (instance *RingMenu) Label() string {
	return "Ring Menu"
}

func (instance *RingMenu) render() {
	m := instance.parent
	display := m.Display

	defer display.Unlock()
	display.Lock()

	display.Clear(display.Primary())
	m.RenderStateCommon()

	display.DrawTextAligned(8, 10, display.Use_Font_Small_Bold(), instance.call.Number, false, gfx.AlignNone, gfx.AlignNone)

	m.RenderFooter("Answer", true)
	display.Render()
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

func (instance *RingMenu) Run() {
	m := instance.parent
	if !instance.configured {
		panic("Attempted to call (*RingMenu).Run() before (*RingMenu).Configure()!")
	}

	if instance.call == nil {
		panic("Attempted to call (*RingMenu).Run() without a call!")
	}

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
				go m.ToStart()
				return

			case evt, ok := <-m.KeypadEvents:
				if !ok {
					return
				}

				if evt.State {
					m.Timers["keypad"].Reset()
					m.Timers["screensaver"].Reset()
					m.Display.On()
					misc.KeyLightsOn()
					switch evt.Key {
					case 'S':
						go m.PlayKey()
						if m.Phone.OK {
							res := m.Phone.AnswerCall(instance.call)
							if res != nil {
								log.Printf("⚠️ Failed to answer call: %v", res)
							} else {
								go m.ToMenuWithArgs("phone", instance.call)
								return
							}
						}
					case 'C':
						go m.PlayKey()
						if m.Phone.OK {
							res := m.Phone.HangupCall(instance.call)
							if res != nil {
								log.Printf("⚠️ Failed to hang up call: %v", res)
							} else {
								go m.ToStart()
								return
							}
						}
					}
				}
			}
		}
	})
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
