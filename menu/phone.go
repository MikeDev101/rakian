package menu

import (
	"context"
	"fmt"
	"misc"
	"modem"
	"strings"

	"gfx"
	"log"
	"sync"
	"time"

	"timers"

	"github.com/maltegrosse/go-modemmanager"
)

type CallDisplayState struct {
	Label    string
	Bold     bool
	HideIcon bool
}

type phone_store struct {
	current_call      string
	queue             *sync.Map
	active_calls      []string
	call_display      map[string]CallDisplayState
	local_held        map[string]bool
	remote_held       map[string]bool
	user_unheld       map[string]bool
	state_change_time map[string]time.Time
	initial_outbound  map[string]bool
	dtmf_active       bool
	options_active    bool
	selector_active   bool
	show_calling      bool
	show_diverts      bool
	diverts_tstamp    time.Time
	blinker_tstamp    time.Time
	new_incoming      bool
}

type PhoneMenu struct {
	ctx          context.Context
	animCtx      context.Context
	animCancel   context.CancelFunc
	renderCtx    context.Context
	renderCancel context.CancelFunc
	configured   bool
	cancelFn     context.CancelFunc
	parent       *Menu
	wg           sync.WaitGroup
	batt_flash   bool
	data_flash   bool
	render_loop  *timers.ResettableTimer
	callsLock    sync.Mutex
	callEnded    chan *modem.Call
	datastore    *phone_store
	stackIndex   int
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
	instance.datastore.blinker_tstamp = time.Time{}
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
	instance.animCtx, instance.animCancel = context.WithCancel(instance.ctx)
	instance.renderCtx, instance.renderCancel = context.WithCancel(instance.ctx)
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
	if len(instance.datastore.active_calls) > 2 {
		instance.callsLock.Unlock()
		log.Printf("⚠️ Phone menu queue full (max 2 calls), ignoring new call: %s", call.ID)
		return
	} else if len(instance.datastore.active_calls) == 0 {
		instance.datastore.show_calling = true
		instance.datastore.blinker_tstamp = time.Now()
	}
	instance.callsLock.Unlock()

	// If we have an active call and this is a new outgoing call, hold the current one
	current := instance.getCurrentcall()
	if current != nil && current.State == modemmanager.MmCallStateActive {
		instance.setLocalHeld(current.ID, true)
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

	// Handle all phone events via subscription
	subKey := m.Phone.Subscribe(func(event modem.IMSEvent) error {
		switch event.Type {
		case modem.IMS_Call_Incoming:
			call, ok := event.Data.(*modem.Call)
			if !ok {
				return fmt.Errorf("invalid call data")
			}
			log.Printf("📞 New incoming call (phone menu event): %v", call)

			if len(m.Phone.GetCalls()) > 2 {
				return nil
			}

			instance.datastore.new_incoming = true
			instance.datastore.options_active = true

			instance.datastore.queue.Store(call.ID, call)

			instance.callsLock.Lock()
			if len(instance.datastore.active_calls) == 0 {
				instance.datastore.current_call = call.ID
			}
			alreadyActive := false
			for _, id := range instance.datastore.active_calls {
				if id == call.ID {
					alreadyActive = true
					break
				}
			}
			if !alreadyActive {
				instance.datastore.active_calls = append(instance.datastore.active_calls, call.ID)
			}
			instance.callsLock.Unlock()

			instance.spawnListener(call)

			if !instance.datastore.selector_active && !instance.datastore.dtmf_active {
				instance.render()
			}

			// Chime in for new call
			if m.Get("BeepOnly").(bool) || m.Get("CanRing").(bool) {
				go misc.PlayBeep(m.Player, instance.ctx)
			}

		case modem.IMS_Call_Connected, modem.IMS_Call_Terminated, modem.IMS_Call_Held, modem.IMS_Call_Unhheld, modem.IMS_Call_Ringing_Out, modem.IMS_Call_Ringing_In:
			call, ok := event.Data.(*modem.Call)
			if !ok {
				return fmt.Errorf("invalid call data")
			}
			if _, exists := instance.datastore.queue.Load(call.ID); exists {
				instance.handleCallEvent(call, event.Type)
				if !instance.datastore.selector_active && !instance.datastore.dtmf_active {
					instance.render()
				}
			}
		}
		return nil
	})

	instance.wg.Go(func() {
		<-instance.ctx.Done()
		m.Phone.Unsubscribe(subKey)
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

					// No more calls left, exit
					m.Set("PhoneActive", false)
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

				keypad_locked := m.Get("KeypadLocked").(bool)

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
						if keypad_locked {
							continue
						}
						go m.Push("power")
						return

					case 'S':
						if keypad_locked {
							go instance.parent.Push("keypad_unlock")
							return
						}

						if !instance.datastore.options_active {
							wasActive := current_call.State == modemmanager.MmCallStateActive
							m.Phone.HangupCall(current_call)

							if wasActive {
								// Swap to previous call if present
								queue := instance.getCurrentQueue()
								for _, call := range queue {
									if call.ID != current_call.ID && call.State == modemmanager.MmCallStateHeld {
										instance.setUserUnheld(call.ID, true)
										m.Phone.UnholdCall(call)
										instance.datastore.current_call = call.ID
										break
									}
								}
							}
							continue
						}

						switch current_call.State {
						case modemmanager.MmCallStateDialing, modemmanager.MmCallStateRingingOut:
							continue
						}

						var options [][]string

						if instance.datastore.new_incoming {
							options = append(options, []string{"Answer"})
							options = append(options, []string{"Decline"})
						}

						options = append(options, []string{"Lock Keypad"})

						if len(instance.getCurrentQueue()) == 1 {
							options = append(options, []string{"End"})
						} else {
							options = append(options, []string{"End active call"})
						}

						// TODO: don't show the "Swap" or "End all calls" options if there are incoming calls.
						if len(instance.getCurrentQueue()) > 1 && !instance.datastore.new_incoming {
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
						case "Lock Keypad":
							instance.datastore.options_active = false
							instance.renderCancel()
							m.Display.Lock()
							m.Display.Unlock()

							instance.animCancel()
							instance.animCtx, instance.animCancel = context.WithCancel(instance.ctx)
							m.Set("KeypadLocked", true)
							m.RenderAnimatedAlert("keypad_locked", instance.animCtx, []string{"Keypad", "locked"})
							go m.PlayAccepted()
							time.Sleep(time.Second * 2)
							instance.animCancel()

							instance.renderCtx, instance.renderCancel = context.WithCancel(instance.ctx)
						case "Mute", "Unmute":
							current_call.Mute = !current_call.Mute
							log.Printf("Set mute state of call to %v", current_call)
						case "Hold":
							instance.setLocalHeld(current_call.ID, true)
							m.Phone.HoldCall(current_call)
						case "Unhold":
							instance.setUserUnheld(current_call.ID, true)
							m.Phone.UnholdCall(current_call)
						case "New call":
							go m.Push("dialer")
						case "Answer":
							queue := instance.getCurrentQueue()
							var ringingCall *modem.Call
							for _, c := range queue {
								if c.State == modemmanager.MmCallStateRingingIn {
									ringingCall = c
									break
								}
							}

							if ringingCall != nil {
								if current_call.State == modemmanager.MmCallStateActive {
									instance.setLocalHeld(current_call.ID, true)
									m.Phone.HoldCall(current_call)
								}
								m.Phone.AnswerCall(ringingCall)
								instance.datastore.current_call = ringingCall.ID
							}
							instance.datastore.options_active = false
							instance.datastore.new_incoming = false
						case "Decline":
							queue := instance.getCurrentQueue()
							var ringingCall *modem.Call
							for _, c := range queue {
								if c.State == modemmanager.MmCallStateRingingIn {
									ringingCall = c
									break
								}
							}

							if ringingCall != nil {
								m.Phone.HangupCall(ringingCall)
							}
							instance.datastore.options_active = false
							instance.datastore.new_incoming = false
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
								instance.setLocalHeld(activeCall.ID, true)
								m.Phone.HoldCall(activeCall)
							}
							if heldCall != nil {
								instance.setUserUnheld(heldCall.ID, true)
								m.Phone.UnholdCall(heldCall)
								instance.datastore.current_call = heldCall.ID
							}
						case "End", "End active call":
							wasActive := current_call.State == modemmanager.MmCallStateActive
							m.Phone.HangupCall(current_call)

							if wasActive {
								// Swap to previous call if present
								queue := instance.getCurrentQueue()
								for _, call := range queue {
									if call.ID != current_call.ID && call.State == modemmanager.MmCallStateHeld {
										instance.setUserUnheld(call.ID, true)
										m.Phone.UnholdCall(call)
										instance.datastore.current_call = call.ID
										break
									}
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
									go m.Phone.SendDTMF(current_call, dtmf)
								}
							}()

						case "Send string":
							dtmf := m.InputPrompt(InputPromptArgs{Title: "DTMF", PhoneNumberOnly: true}, instance.ctx)
							if dtmf != "" {
								go m.Phone.SendDTMF(current_call, dtmf)
							}
						case "End all calls":
							m.Phone.HangupAll()
						case "Phone book":
							go m.Push("phone_book")
						}
						instance.render()

					case 'C':
						if keypad_locked {
							continue
						}

						switch current_call.State {
						case modemmanager.MmCallStateDialing, modemmanager.MmCallStateRingingOut:
							continue
						}
						instance.datastore.options_active = !instance.datastore.options_active
						instance.render()

					case 'U':
						// TODO: volume up
					case 'D':
						// TODO: volume down
					default:
						if keypad_locked {
							continue
						}
						go m.playDTMF(evt.Key)
						if m.Phone.OK() {
							go m.Phone.SendDTMF(current_call, string(evt.Key))
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

	select {
	case <-instance.renderCtx.Done():
		return
	default:
	}

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
		if (current_call.State == modemmanager.MmCallStateDialing ||
			current_call.State == modemmanager.MmCallStateRingingOut) &&
			len(queue) == 1 {

			dotCount := (time.Since(instance.datastore.blinker_tstamp).Milliseconds() / 500) % 4
			var dots strings.Builder
			for i := 0; i < int(dotCount); i++ {
				dots.WriteRune('.')
			}
			label := "Calling" + dots.String()

			font := display.Use_Font_Small_Bold()
			display.DrawTextAligned(8, 8, font, label, false, gfx.AlignNone, gfx.AlignNone)
			display.DrawTextWrapped(8, 18, 76, 38, display.Use_Font_Small_Plain(), current_call.Number, false, gfx.WrapRight, gfx.WrapUp)

			m.RenderFooter("End", true)
			display.Render()
			return
		}
		instance.datastore.show_calling = false

		if m.Get("DivertsActive").(bool) {
			instance.datastore.show_diverts = true
			instance.datastore.diverts_tstamp = time.Now()
		}
	}

	if instance.datastore.show_diverts {
		if time.Since(instance.datastore.diverts_tstamp) < 1*time.Second {
			font := display.Use_Font_Small_Bold()
			display.DrawTextWrapped(8, 8, 76, 38, font, "Note: You have active diverts", false, gfx.AlignRight, gfx.AlignBelow)
			if instance.datastore.options_active {
				m.RenderFooter("Options", true)
			} else {
				m.RenderFooter("End", true)
			}
			display.Render()
			return
		}
		instance.datastore.show_diverts = false
	}

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
		case modemmanager.MmCallStateWaiting, modemmanager.MmCallStateRingingIn, modemmanager.MmCallStateDialing, modemmanager.MmCallStateRingingOut:
			call_state = "call_preparing"
		default:
			call_state = "call_active"
		}

		// Draw Label
		label := fmt.Sprintf("Call %d", i+1)
		font := display.Use_Font_Small_Plain()
		hideIcon := false

		if call.State == modemmanager.MmCallStateRingingIn {
			bold := (time.Now().UnixMilli()/1000)%2 == 0
			hideIcon = false
			if bold {
				label = "Incoming"
				font = display.Use_Font_Small_Bold()
			} else {
				label = call.Number
				if label == "" {
					label = fmt.Sprintf("Call %d", i+1)
				}
				font = display.Use_Font_Small_Plain()
			}
		} else {
			instance.callsLock.Lock()
			if instance.datastore.call_display != nil {
				if st, ok := instance.datastore.call_display[call.ID]; ok {
					label = st.Label
					if st.Bold {
						font = display.Use_Font_Small_Bold()
					}
					hideIcon = st.HideIcon
				}
			}
			instance.callsLock.Unlock()
		}

		if call_state != "" && !hideIcon {
			display.DrawImageAligned(display.Use_Sprites()[call_state], 8, y, gfx.AlignNone, gfx.AlignBelow)
		}

		labelX := 18
		if hideIcon {
			labelX = 8
		}

		display.DrawTextAligned(labelX, y, font, label, false, gfx.AlignNone, gfx.AlignNone)
	}

	// Render footer
	if m.Get("KeypadLocked").(bool) {
		m.RenderFooter("Unlock", true)
	} else if instance.datastore.options_active {
		m.RenderFooter("Options", true)
	} else {
		m.RenderFooter("End", true)
	}
	display.Render()
}

func (instance *PhoneMenu) setStateChangeTime(callID string, t time.Time) {
	instance.callsLock.Lock()
	defer instance.callsLock.Unlock()
	if instance.datastore.state_change_time == nil {
		instance.datastore.state_change_time = make(map[string]time.Time)
	}
	instance.datastore.state_change_time[callID] = t
}

func (instance *PhoneMenu) getStateChangeTime(callID string) time.Time {
	instance.callsLock.Lock()
	defer instance.callsLock.Unlock()
	if instance.datastore.state_change_time == nil {
		return time.Time{}
	}
	return instance.datastore.state_change_time[callID]
}

func (instance *PhoneMenu) setInitialOutbound(callID string, val bool) {
	instance.callsLock.Lock()
	defer instance.callsLock.Unlock()
	if instance.datastore.initial_outbound == nil {
		instance.datastore.initial_outbound = make(map[string]bool)
	}
	instance.datastore.initial_outbound[callID] = val
}

func (instance *PhoneMenu) isInitialOutbound(callID string) bool {
	instance.callsLock.Lock()
	defer instance.callsLock.Unlock()
	if instance.datastore.initial_outbound == nil {
		return false
	}
	return instance.datastore.initial_outbound[callID]
}

func (instance *PhoneMenu) updateCallDisplay(c *modem.Call) {
	callIndex := 0
	instance.callsLock.Lock()
	for i, id := range instance.datastore.active_calls {
		if id == c.ID {
			callIndex = i + 1
			break
		}
	}
	instance.callsLock.Unlock()

	if callIndex == 0 {
		return
	}

	baseLabel := fmt.Sprintf("Call %d", callIndex)
	var newLabel string
	var newFont bool
	var newHideIcon bool

	if c.State == modemmanager.MmCallStateRingingIn {
		newLabel = baseLabel
		newFont = false
		newHideIcon = false
	} else {
		newFont = false
		newHideIcon = false
		stateChangeTime := instance.getStateChangeTime(c.ID)
		isInitialOutbound := instance.isInitialOutbound(c.ID)

		if !stateChangeTime.IsZero() && time.Since(stateChangeTime) < 1500*time.Millisecond {
			switch c.State {
			case modemmanager.MmCallStateHeld:
				newFont = true
				newHideIcon = true
				if instance.isLocalHeld(c.ID) {
					newLabel = fmt.Sprintf("Call %d on hold", callIndex)
				} else {
					newLabel = "On hold"
				}
			case modemmanager.MmCallStateActive:
				if isInitialOutbound {
					newLabel = baseLabel
				} else {
					newFont = true
					newHideIcon = true
					if instance.isUserUnheld(c.ID) {
						if instance.isRemoteHeld(c.ID) {
							newLabel = "On hold"
						} else if instance.isLocalHeld(c.ID) {
							newLabel = fmt.Sprintf("Call %d active", callIndex)
						} else {
							newLabel = "Connected"
						}
					} else {
						newLabel = "Connected"
					}
				}
			default:
				newLabel = baseLabel
			}
		} else {
			if c.State == modemmanager.MmCallStateActive {
				if isInitialOutbound {
					instance.setInitialOutbound(c.ID, false)
				}
				if instance.isLocalHeld(c.ID) {
					instance.setLocalHeld(c.ID, false)
				}
				if instance.isUserUnheld(c.ID) {
					instance.setUserUnheld(c.ID, false)
				} else {
					instance.setRemoteHeld(c.ID, false)
				}
			}

			if c.State == modemmanager.MmCallStateHeld {
				if !instance.isLocalHeld(c.ID) {
					instance.setRemoteHeld(c.ID, true)
				}
			}
			newLabel = baseLabel
		}
	}

	instance.setCallDisplay(c.ID, CallDisplayState{
		Label:    newLabel,
		Bold:     newFont,
		HideIcon: newHideIcon,
	})
}

func (instance *PhoneMenu) handleCallEvent(c *modem.Call, eventType modem.IMSEventType) {
	if eventType == modem.IMS_Call_Terminated {
		go func() {
			if !instance.datastore.show_calling || !c.StartTime.IsZero() {
				time.Sleep(1 * time.Second)
			}
			select {
			case instance.callEnded <- c:
			case <-instance.ctx.Done():
			}
		}()
	} else {
		instance.setStateChangeTime(c.ID, time.Now())
		if c.State == modemmanager.MmCallStateDialing || c.State == modemmanager.MmCallStateRingingOut {
			instance.setInitialOutbound(c.ID, true)
		}
		instance.updateCallDisplay(c)
		time.AfterFunc(1500*time.Millisecond, func() {
			instance.updateCallDisplay(c)
		})
	}
}

func (instance *PhoneMenu) spawnListener(c *modem.Call) {
	if c.State == modemmanager.MmCallStateDialing {
		instance.datastore.blinker_tstamp = time.Now()
		instance.datastore.show_calling = true
	}
	instance.handleCallEvent(c, modem.IMS_Call_Ringing_In)
}

func (instance *PhoneMenu) setCallDisplay(callID string, state CallDisplayState) {
	instance.callsLock.Lock()
	defer instance.callsLock.Unlock()
	if instance.datastore.call_display == nil {
		instance.datastore.call_display = make(map[string]CallDisplayState)
	}
	instance.datastore.call_display[callID] = state
}

func (instance *PhoneMenu) setLocalHeld(callID string, held bool) {
	instance.callsLock.Lock()
	defer instance.callsLock.Unlock()
	if instance.datastore.local_held == nil {
		instance.datastore.local_held = make(map[string]bool)
	}
	instance.datastore.local_held[callID] = held
}

func (instance *PhoneMenu) isLocalHeld(callID string) bool {
	instance.callsLock.Lock()
	defer instance.callsLock.Unlock()
	if instance.datastore.local_held == nil {
		return false
	}
	return instance.datastore.local_held[callID]
}

func (instance *PhoneMenu) setRemoteHeld(callID string, held bool) {
	instance.callsLock.Lock()
	defer instance.callsLock.Unlock()
	if instance.datastore.remote_held == nil {
		instance.datastore.remote_held = make(map[string]bool)
	}
	instance.datastore.remote_held[callID] = held
}

func (instance *PhoneMenu) isRemoteHeld(callID string) bool {
	instance.callsLock.Lock()
	defer instance.callsLock.Unlock()
	if instance.datastore.remote_held == nil {
		return false
	}
	return instance.datastore.remote_held[callID]
}

func (instance *PhoneMenu) setUserUnheld(callID string, unheld bool) {
	instance.callsLock.Lock()
	defer instance.callsLock.Unlock()
	if instance.datastore.user_unheld == nil {
		instance.datastore.user_unheld = make(map[string]bool)
	}
	instance.datastore.user_unheld[callID] = unheld
}

func (instance *PhoneMenu) isUserUnheld(callID string) bool {
	instance.callsLock.Lock()
	defer instance.callsLock.Unlock()
	if instance.datastore.user_unheld == nil {
		return false
	}
	return instance.datastore.user_unheld[callID]
}
