package menu

import (
	"context"
	"fmt"
	"modem"
	"strings"

	"gfx"
	"log"
	"sync"
	"time"

	"timers"

	"github.com/maltegrosse/go-modemmanager"
)

type phone_store struct {
	current_call    string
	queue           *sync.Map
	active_calls    []string
	dtmf_active     bool
	options_active  bool
	selector_active bool
	show_calling    bool
	dialing_tstamp  time.Time
}

type PhoneMenu struct {
	ctx         context.Context
	configured  bool
	cancelFn    context.CancelFunc
	parent      *Menu
	wg          sync.WaitGroup
	batt_flash  bool
	data_flash  bool
	render_loop *timers.ResettableTimer
	callsLock   sync.Mutex
	callEnded   chan *modem.Call
	datastore   *phone_store
	stackIndex  int
}

func (m *Menu) NewPhoneMenu() MenuInstance {
	return &PhoneMenu{
		parent:    m,
		callEnded: make(chan *modem.Call, 10),
		datastore: &phone_store{
			queue: &sync.Map{},
		},
	}
}

func (instance *PhoneMenu) Label() string {
	return "Phone Menu"
}

func (instance *PhoneMenu) Pause() {
	Pause(instance)
}

func (instance *PhoneMenu) Stop() {
	Stop(instance)
}

func (instance *PhoneMenu) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *PhoneMenu) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *PhoneMenu) Cleanup() {
	instance.datastore.current_call = ""
	instance.datastore.dtmf_active = false
	instance.datastore.options_active = false
	instance.datastore.selector_active = false
	instance.datastore.show_calling = false
	instance.datastore.dialing_tstamp = time.Time{}
	instance.configured = false
}

func (instance *PhoneMenu) Save() {
	Save(instance.parent, instance, instance.datastore)
}

func (instance *PhoneMenu) Load() {
	if loaded, ok := Load(instance.parent, instance); ok {
		instance.datastore = loaded.(*phone_store)

		// Respawn listeners for existing calls
		instance.callsLock.Lock()
		defer instance.callsLock.Unlock()
		for _, id := range instance.datastore.active_calls {
			if val, ok := instance.datastore.queue.Load(id); ok {
				call := val.(*modem.Call)
				instance.spawnListener(call)
			}
		}
	}
}

func (instance *PhoneMenu) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *PhoneMenu) GetStackIndex() int {
	return instance.stackIndex
}

func (instance *PhoneMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
	instance.Load()
}

func (instance *PhoneMenu) ConfigureWithArgs(args ...any) {
	// Must be a *modem.Call
	if !instance.configured {
		instance.Configure()
	}

	call, ok := args[0].(*modem.Call)
	if !ok {
		panic("(*PhoneMenu).ConfigureWithArgs() Type error: argument must be a *modem.Call type")
	}

	// Check if we are already tracking this call
	if _, exists := instance.datastore.queue.Load(call.ID); exists {
		instance.datastore.current_call = call.ID
		return
	}

	// Check queue limit
	instance.callsLock.Lock()
	if len(instance.datastore.active_calls) >= 2 {
		instance.callsLock.Unlock()
		log.Printf("⚠️ Phone menu queue full (max 2 calls), ignoring new call: %s", call.ID)
		return
	} else if len(instance.datastore.active_calls) == 0 {
		instance.datastore.show_calling = true
		instance.datastore.dialing_tstamp = time.Now()
	}
	instance.callsLock.Unlock()

	// If we have an active call and this is a new outgoing call, hold the current one
	current := instance.getCurrentcall()
	if current != nil && current.State == modemmanager.MmCallStateActive {
		instance.parent.Phone.HoldCall(current)
	}

	instance.callsLock.Lock()
	instance.datastore.active_calls = append(instance.datastore.active_calls, call.ID)
	instance.callsLock.Unlock()

	instance.datastore.queue.Store(call.ID, call)
	instance.datastore.current_call = call.ID

	instance.spawnListener(call)

	// Play ringing tone if dialing
	/* if call.State == modemmanager.MmCallStateDialing || call.State == modemmanager.MmCallStateRingingOut {
		go func() {
			ringCtx, ringCancel := context.WithCancel(context.Background())
			ringing := false

			defer ringCancel()
			for {
				select {
				case <-instance.ctx.Done():
					return
				case <-call.Ended:
					return
				case <-time.After(100 * time.Millisecond):
					if !ringing && call.State == modemmanager.MmCallStateRingingOut {
						go instance.parent.PlayRinging(ringCtx)
						ringing = true
					}
					if call.State == modemmanager.MmCallStateActive || call.State == modemmanager.MmCallStateTerminated {
						return
					}
				}
			}
		}()
	} */

	log.Printf("Phone configured with call %v", call)
}

func (instance *PhoneMenu) Run() {
	m := instance.parent

	if !instance.configured {
		panic("Attempted to call (*PhoneMenu).Run() before (*PhoneMenu).Configure()!")
	}

	if !m.Phone.OK() {
		panic("Attempted to call (*PhoneMenu).Run() in an illegal state - phone was not ready")
	}

	// Disable screensaver
	m.Timers["screensaver"].Stop()

	// Main render loop
	instance.wg.Go(func() {
		for {
			select {
			case <-instance.ctx.Done():
				instance.parent.Keypad.KeyLightsOn()
				go m.Timers["keypad"].Restart()
				go m.Timers["screensaver"].Restart()
				return

			case <-time.After(100 * time.Millisecond):
				if m.Display.IsOn() && !instance.datastore.selector_active && !instance.datastore.dtmf_active {
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

			case endedCall := <-instance.callEnded:
				instance.datastore.queue.Delete(endedCall.ID)

				instance.callsLock.Lock()
				for i, id := range instance.datastore.active_calls {
					if id == endedCall.ID {
						instance.datastore.active_calls = append(instance.datastore.active_calls[:i], instance.datastore.active_calls[i+1:]...)
						break
					}
				}
				instance.callsLock.Unlock()

				// Check if we have any calls left
				queue := instance.getCurrentQueue()
				if len(queue) == 0 {
					go m.ToStart()
					return
				}

				// Switch to the next call
				nextCall := queue[0]
				instance.datastore.current_call = nextCall.ID

				// Auto-resume if held
				if nextCall.State == modemmanager.MmCallStateHeld {
					m.Phone.UnholdCall(nextCall)
				}

			case evt, ok := <-m.KeypadEvents:
				if !ok {
					return
				}

				if evt.State { // Key Press
					m.Timers["keypad"].Restart()
					m.Keypad.KeyLightsOn()

					current_call := instance.getCurrentcall()
					if current_call == nil {
						continue
					}

					if current_call.State == modemmanager.MmCallStateTerminated {
						continue
					}

					switch evt.Key {

					case 'P':
						go m.Push("power")
						return

					case 'S':
						if !instance.datastore.options_active {
							m.Phone.HangupCall(current_call)

							// Swap to previous call if present
							queue := instance.getCurrentQueue()
							for _, call := range queue {
								if call.ID != current_call.ID && call.State == modemmanager.MmCallStateHeld {
									m.Phone.UnholdCall(call)
									instance.datastore.current_call = call.ID
									break
								}
							}
							continue
						}

						switch current_call.State {
						case modemmanager.MmCallStateDialing, modemmanager.MmCallStateRingingOut:
							continue
						}

						var options [][]string

						options = append(options, []string{"End active call"})
						if len(instance.getCurrentQueue()) > 1 {
							options = append(options, []string{"Swap"})
							options = append(options, []string{"End all calls"})
						}

						switch current_call.State {
						case modemmanager.MmCallStateActive:
							options = append(options, []string{"Hold"})
						case modemmanager.MmCallStateHeld:
							options = append(options, []string{"Unhold"})
						}

						if len(instance.getCurrentQueue()) == 1 {
							options = append(options, []string{"New call"})
						}
						options = append(options, []string{"Send DTMF"})
						options = append(options, []string{"Phone book"})

						if !current_call.Mute {
							options = append(options, []string{"Mute"})
						} else {
							options = append(options, []string{"Unmute"})
						}

						custom_render_func := func() {
							m.Display.DrawImageAligned(m.Display.Use_Sprites()["phone"], 0, 0, gfx.AlignNone, gfx.AlignBelow)
						}

						instance.datastore.selector_active = true
						selection := m.ShowSelector(SelectorArgs{
							Options:      options,
							ShowTitle:    true,
							SelectorType: SELECTOR_MULTI_3,
							ButtonLabel:  "Select",
							CustomRender: custom_render_func,
						}, instance.ctx)
						instance.datastore.selector_active = false

						if selection == nil || len(selection.SelectionPath) == 0 {
							instance.render()
							continue
						}

						switch selection.SelectionPath[0] {
						case "Mute", "Unmute":
							current_call.Mute = !current_call.Mute
							log.Printf("Set mute state of call to %v", current_call)
						case "Hold":
							m.Phone.HoldCall(current_call)
						case "Unhold":
							m.Phone.UnholdCall(current_call)
						case "New call":
							go m.Push("dialer")
						case "Answer":
							m.Phone.AnswerCall(current_call)
						case "Decline":
							m.Phone.HangupCall(current_call)
						case "Swap":
							queue := instance.getCurrentQueue()
							var activeCall *modem.Call
							var heldCall *modem.Call

							for _, call := range queue {
								switch call.State {
								case modemmanager.MmCallStateActive:
									activeCall = call
								case modemmanager.MmCallStateHeld:
									if heldCall == nil {
										heldCall = call
									}
								}
							}

							if activeCall != nil {
								m.Phone.HoldCall(activeCall)
							}
							if heldCall != nil {
								m.Phone.UnholdCall(heldCall)
								instance.datastore.current_call = heldCall.ID
							}
						case "End active call":
							m.Phone.HangupCall(current_call)

							// Swap to previous call if present
							queue := instance.getCurrentQueue()
							for _, call := range queue {
								if call.ID != current_call.ID && call.State == modemmanager.MmCallStateHeld {
									m.Phone.UnholdCall(call)
									instance.datastore.current_call = call.ID
									break
								}
							}

						case "Send DTMF":
							func() {
								instance.datastore.dtmf_active = true
								m.Display.Lock()
								defer func() {
									instance.datastore.dtmf_active = false
									m.Display.Unlock()
								}()

								dtmf := m.InputPrompt(InputPromptArgs{Title: "DTMF", PhoneNumberOnly: true}, instance.ctx)
								if dtmf != "" {
									m.Phone.SendDTMF(current_call, dtmf)
								}
							}()

						case "Send string":
							dtmf := m.InputPrompt(InputPromptArgs{Title: "DTMF", PhoneNumberOnly: true}, instance.ctx)
							if dtmf != "" {
								m.Phone.SendDTMF(current_call, dtmf)
							}
						case "End all calls":
							m.Phone.HangupAll()
						case "Phone book":
							go m.Push("phone_book")
						}
						instance.render()

					case 'C':
						switch current_call.State {
						case modemmanager.MmCallStateDialing, modemmanager.MmCallStateRingingOut:
							continue
						}
						instance.datastore.options_active = !instance.datastore.options_active
						instance.render()

					case 'U':
					case 'D':
					default:
						go m.playDTMF(evt.Key)
						if m.Phone.OK() {
							m.Phone.SendDTMF(current_call, string(evt.Key))
						}
					}
				} else { // Key Release
					go m.stopDTMF(evt.Key)
				}
			}
		}
	})
}

func (instance *PhoneMenu) getCurrentcall() *modem.Call {
	instance.callsLock.Lock()
	defer instance.callsLock.Unlock()

	if instance.datastore.current_call == "" {
		if len(instance.datastore.active_calls) > 0 {
			instance.datastore.current_call = instance.datastore.active_calls[0]
		} else {
			return nil
		}
	}

	res, ok := instance.datastore.queue.Load(instance.datastore.current_call)
	if !ok {
		// Fallback if current_call is stale
		if len(instance.datastore.active_calls) > 0 {
			instance.datastore.current_call = instance.datastore.active_calls[0]
			res, ok = instance.datastore.queue.Load(instance.datastore.current_call)
			if !ok {
				return nil
			}
		} else {
			return nil
		}
	}
	return res.(*modem.Call)
}

func (instance *PhoneMenu) getCurrentQueue() []*modem.Call {
	instance.callsLock.Lock()
	defer instance.callsLock.Unlock()

	var res []*modem.Call
	for _, id := range instance.datastore.active_calls {
		if val, ok := instance.datastore.queue.Load(id); ok {
			res = append(res, val.(*modem.Call))
		}
	}

	return res
}

func (instance *PhoneMenu) render() {
	m := instance.parent

	if instance.datastore.dtmf_active {
		return
	}

	display := m.Display
	queue := instance.getCurrentQueue()
	current_call := instance.getCurrentcall()
	if len(queue) == 0 {
		return // Nothing to render
	}

	defer display.Unlock()
	display.Lock()

	// Render main screen
	display.Clear(display.Primary())
	m.RenderStateCommon()
	m.RenderClock()

	// Check for single dialing call
	if instance.datastore.show_calling {

		// Cycle dots while dialing
		switch current_call.State {
		case modemmanager.MmCallStateDialing, modemmanager.MmCallStateRingingOut:
			dotCount := (time.Since(instance.datastore.dialing_tstamp).Milliseconds() / 500) % 4
			var dots strings.Builder
			for i := 0; i < int(dotCount); i++ {
				dots.WriteRune('.')
			}
			label := "Calling" + dots.String()

			font := display.Use_Font_Small_Bold()
			display.DrawTextAligned(8, 10, font, label, false, gfx.AlignNone, gfx.AlignNone)
			display.DrawTextWrapped(8, 20, 76, 38, display.Use_Font_Small_Plain(), current_call.Number, false, gfx.WrapRight, gfx.WrapUp)

			m.RenderFooter("End", true)
			display.Render()
			return
		}
		instance.datastore.show_calling = false

		// TODO: if call diverts are active while ringing, show a note
	}

	display.DrawImageAligned(display.Use_Sprites()["phone"], 8, 0, gfx.AlignRight, gfx.AlignBelow)

	// Render up to 2 calls
	for i, call := range queue {
		if i >= 2 {
			break
		}

		y := 10 + (i * 12)

		// Determine icon
		var call_state string
		switch call.State {
		case modemmanager.MmCallStateHeld:
			call_state = "call_held"
		case modemmanager.MmCallStateTerminated:
			call_state = "call_ended"
		case modemmanager.MmCallStateWaiting:
			call_state = "call_preparing"
		default:
			call_state = "call_active"
		}

		if call_state != "" {
			display.DrawImageAligned(display.Use_Sprites()[call_state], 8, y, gfx.AlignNone, gfx.AlignBelow)
		}

		// Draw Label
		label := fmt.Sprintf("Call %d", i+1)
		display.DrawTextAligned(18, y, display.Use_Font_Small_Plain(), label, false, gfx.AlignNone, gfx.AlignNone)
	}

	// Render footer
	if instance.datastore.options_active {
		m.RenderFooter("Options", true)
	} else {
		m.RenderFooter("End", true)
	}
	display.Render()
}

func (instance *PhoneMenu) spawnListener(c *modem.Call) {
	instance.wg.Add(1)
	go func() {
		defer instance.wg.Done()

		// Initial state
		if c.State == modemmanager.MmCallStateDialing {
			instance.datastore.dialing_tstamp = time.Now()
			instance.datastore.show_calling = true
		}

		if c.State == modemmanager.MmCallStateTerminated {
			select {
			case instance.callEnded <- c:
			case <-instance.ctx.Done():
			}
			return
		}

		select {
		case <-instance.ctx.Done():
			return
		case <-c.Ended:
			// Wait a second before removing the call to allow the user to see the "Call ended" state
			if !instance.datastore.show_calling || !c.StartTime.IsZero() {
				timer := time.NewTimer(1 * time.Second)
				select {
				case <-instance.ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}

			select {
			case instance.callEnded <- c:
			case <-instance.ctx.Done():
			}
		}
	}()
}
