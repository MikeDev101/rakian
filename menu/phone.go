package menu

import (
	"context"
	"fmt"
	"modem"

	// "fmt"
	"log"
	"sync"
	"time"

	"lcd"
	"misc"
	"timers"
)

type PhoneMenu struct {
	ctx             context.Context
	configured      bool
	cancelFn        context.CancelFunc
	parent          *Menu
	wg              sync.WaitGroup
	batt_flash      bool
	data_flash      bool
	render_loop     *timers.ResettableTimer
	call            *modem.Call
	options_active  bool
	selector_active bool
}

func (m *Menu) NewPhoneMenu() *PhoneMenu {
	return &PhoneMenu{
		parent: m,
	}
}

func (instance *PhoneMenu) render() {
	m := instance.parent
	display := m.Display
	display.Clear(lcd.White)
	m.RenderStateCommon()

	display.DrawTextAligned(8, 10, display.Use_Font_Small_Bold(), instance.call.Number, false, lcd.AlignNone, lcd.AlignNone)
	display.DrawTextAligned(8, 18, display.Use_Font_Small_Plain(), instance.call.State, false, lcd.AlignNone, lcd.AlignNone)
	if instance.call.Active {
		d := time.Since(instance.call.StartTime)
		display.DrawTextAligned(8, 26, display.Use_Font_Tiny(), fmt.Sprintf("%02d:%02d:%02d", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60), false, lcd.AlignRight, lcd.AlignNone)
	}

	if instance.options_active {
		m.RenderFooter("Options", true)
	} else {
		m.RenderFooter("End", true)
	}
	display.Render()
}

func (instance *PhoneMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *PhoneMenu) ConfigureWithArgs(args ...any) {
	// Must be a *modem.Call
	call, ok := args[0].(*modem.Call)
	if !ok {
		panic("(*PhoneMenu).ConfigureWithArgs() Type error: argument must be a *modem.Call type")
	}
	instance.options_active = false
	instance.selector_active = false
	instance.call = call
	instance.Configure()
}

func (instance *PhoneMenu) Run() {
	m := instance.parent

	if !instance.configured {
		panic("Attempted to call (*PhoneMenu).Run() before (*PhoneMenu).Configure()!")
	}

	if !m.Phone.OK {
		panic("Attempted to call (*PhoneMenu).Run() in an illegal state - phone was not ready")
	}

	if instance.call == nil {
		panic("Attempted to call (*PhoneMenu).Run() in an illegal state - call was nil")
	}

	// Disable screensaver
	go m.Timers["screensaver"].Stop()
	misc.KeyLightsOn()

	// Main render loop
	instance.wg.Go(func() {
		for {
			select {
			case <-instance.ctx.Done():
				misc.KeyLightsOn()
				go m.Timers["keypad"].Restart()
				go m.Timers["screensaver"].Restart()
				return

			case <-time.After(100 * time.Millisecond):
				if m.Display.IsOn && !instance.selector_active {
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

					case 'P':
						go m.Push("power")
						return

					case 'S':
						if !instance.options_active {
							if m.Phone.OK {
								if err := m.Phone.HangupCall(instance.call); err == nil {
									go m.ToStart()
									return
								}
							}
							continue
						}

						var options [][]string

						switch instance.call.State {
						case "active":
							options = append(options, []string{"Hold"})
						case "held":
							options = append(options, []string{"Unhold"})
						}

						options = append(options, []string{"New call"})

						if instance.call.State == "incoming" || instance.call.State == "waiting" {
							options = append(options, []string{"Answer"})
							options = append(options, []string{"Reject"})
						}

						options = append(options, []string{"Swap"})
						options = append(options, []string{"End call"})
						options = append(options, []string{"Send DTMF"})
						options = append(options, []string{"Send string"})
						options = append(options, []string{"End all calls"})
						options = append(options, []string{"Phonebook"})

						instance.selector_active = true
						selection := m.ShowSelector(SelectorArgs{
							Options:      options,
							SelectorType: SELECTOR_MULTI_4,
							ButtonLabel:  "Select",
						}, instance.ctx)
						instance.selector_active = false

						if selection == nil || len(selection.SelectionPath) == 0 {
							instance.options_active = false
							instance.render()
							continue
						}

						switch selection.SelectionPath[0] {
						case "Hold":
							m.Phone.HoldCall(instance.call)
						case "Unhold":
							m.Phone.UnholdCall(instance.call)
						case "New call":
							go m.Push("dialer")
						case "Answer":
							m.Phone.AnswerCall(instance.call)
						case "Reject":
							m.Phone.HangupCall(instance.call)
							go m.ToStart()
							return
						case "Swap":
							// TODO: Implement swap
						case "End call":
							m.Phone.HangupCall(instance.call)
							go m.ToStart()
							return
						case "Send DTMF":
							// Already handled by keypad in main loop
						case "Send string":
							dtmf := m.InputPrompt(InputPromptArgs{Title: "DTMF", PhoneNumberOnly: true}, instance.ctx)
							if dtmf != "" {
								m.Phone.SendDTMF(instance.call, dtmf)
							}
						case "End all calls":
							m.Phone.HangupAll()
							go m.ToStart()
							return
						case "Phonebook":
							go m.Push("phonebook")
						}
						instance.options_active = false
						instance.render()

					case 'C':
						instance.options_active = !instance.options_active
						instance.render()

					case 'U':
					case 'D':
					default:
						go m.playDTMF(evt.Key)
						if m.Phone.OK {
							m.Phone.SendDTMF(instance.call, string(evt.Key))
						}
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
	instance.options_active = false
	instance.selector_active = false
}
