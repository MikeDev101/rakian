package menu

import (
	"context"
	"gfx"
	"log"
	"slices"
	"sync"
	"time"
)

type dialer_store struct {
	dial_number      string
	lastAsteriskTime time.Time
}

type DialerMenu struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
	pressStart map[rune]time.Time
	initialKey rune
	datastore  *dialer_store
	stackIndex int
}

func (m *Menu) NewDialerMenu() MenuInstance {
	return &DialerMenu{
		parent:     m,
		pressStart: make(map[rune]time.Time),
		datastore: &dialer_store{
			lastAsteriskTime: time.Now(),
		},
	}
}

func (instance *DialerMenu) Label() string {
	return "Dialer Menu"
}

func (instance *DialerMenu) render() {
	m := instance.parent
	display := m.Display
	defer display.Unlock()
	display.Lock()
	display.Clear(display.Primary())
	m.RenderStateCommon()
	display.DrawTextWrapped(8, 10, 78, 38, display.Use_Font_Large_Bold(), instance.datastore.dial_number, false, gfx.WrapRight, gfx.WrapUp)
	m.RenderFooter("Dial", true)
	display.Render()
}

func (instance *DialerMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *DialerMenu) ConfigureWithArgs(args ...any) {
	if len(args) > 0 {
		if key, ok := args[0].(rune); ok {
			instance.initialKey = key
		}
	}
	instance.Configure()
}

func (instance *DialerMenu) Run() {
	m := instance.parent
	if !instance.configured {
		panic("Attempted to call (*DialerMenu).Run() before (*DialerMenu).Configure()!")
	}

	if instance.initialKey != 0 {
		key := instance.initialKey
		instance.datastore.dial_number = string(key)
		instance.initialKey = 0
		go m.playDTMF(key)
		if key == '*' {
			instance.datastore.lastAsteriskTime = time.Now()
		}
	}
	instance.render()

	instance.wg.Add(1)
	defer instance.wg.Done()
	for {
		select {
		case <-instance.ctx.Done():
			return

		case evt := <-m.KeypadEvents:
			if evt.State { // Key Press

				m.Timers["keypad"].Reset()
				m.Timers["screensaver"].Reset()
				m.Display.On()
				instance.parent.Keypad.KeyLightsOn()
				switch evt.Key {
				case '*':
					go m.playDTMF('*')
					now := time.Now()
					if now.Sub(instance.datastore.lastAsteriskTime) <= 750*time.Millisecond {
						// Replace last '*' with '+'
						runes := []rune(instance.datastore.dial_number)
						if len(runes) > 0 && runes[len(runes)-1] == '*' {
							runes[len(runes)-1] = '+'
							instance.datastore.dial_number = string(runes)
						} else {
							instance.datastore.dial_number += "*"
						}
					} else {
						instance.datastore.dial_number += "*"
					}
					instance.datastore.lastAsteriskTime = now
					instance.render()

				case 'C':
					go m.PlayKey()

					// Delete last key from dial_number
					runes := []rune(instance.datastore.dial_number)
					if len(runes) > 0 {
						instance.datastore.dial_number = string(runes[:len(runes)-1])
					}

					// Check if
					if len(runes) == 0 {
						go m.Pop()
						return
					} else {
						instance.render()
					}

				case 'P':
					go m.PlayKey()
					go m.Push("power")
					return

				case 'U':
					go m.PlayKey()
				case 'D':
					go m.PlayKey()
				case 'S':
					if len(instance.datastore.dial_number) == 0 {
						continue
					}

					if !m.Phone.OK() {
						m.ExitWithAlert(instance.ctx, []string{"No", "service"})
						return
					}

					if m.Phone.IsFlightMode() {
						m.ExitWithAlert(instance.ctx, []string{"Flight", "mode", "enabled"})
						return
					}

					if !m.Phone.IsRegistered() {
						m.ExitWithAlert(instance.ctx, []string{"No", "service"})
						return
					}

					// Ignore registration if the number is in the service/emergency numbers list
					if slices.Contains(m.Phone.GetEmergencyNumbers(), instance.datastore.dial_number) {
						m.RenderAlert("", []string{"Attempting", "emergency", "call"})
						go m.PlayAlert()

						time.Sleep(3 * time.Second)

						session, err := m.Phone.PlaceCall(instance.datastore.dial_number)
						if err != nil {
							log.Printf("⚠️ Failed to place call: %v", err)
							m.RenderAlert("prohibited", []string{"Call", "failed"})
							go m.PlayAlert()
							time.Sleep(3 * time.Second)
							go m.Pop()
							return
						}

						idx := instance.GetStackIndex()
						if idx > 0 && m.GetMenuKeyAt(idx-1) == "phone" {
							go m.PopWithArgs(session)
						} else {
							go m.ToMenuWithArgs("phone", session)
						}
						return
					}

					if !m.Phone.IsRegistered() {
						m.ExitWithAlert(instance.ctx, []string{"No", "service!"})
						return
					}

					session, err := m.Phone.PlaceCall(instance.datastore.dial_number)
					if err != nil {
						log.Printf("⚠️ Failed to place call: %v", err)
						m.RenderAlert("prohibited", []string{"Call", "failed"})
						go m.PlayAlert()
						time.Sleep(3 * time.Second)
						go m.Pop()
						return
					}

					idx := instance.GetStackIndex()
					if idx > 0 && m.GetMenuKeyAt(idx-1) == "phone" {
						go m.PopWithArgs(session)
					} else {
						go m.ToMenuWithArgs("phone", session)
					}
					return

				default:
					if len(instance.datastore.dial_number) < 16 {
						instance.datastore.dial_number += string(evt.Key)
						instance.render()
						go m.playDTMF(evt.Key)
					}

				}
			} else { // Key Release
				go m.stopDTMF(evt.Key)
			}
		}
	}
}

func (instance *DialerMenu) Pause() {
	Pause(instance)
}

func (instance *DialerMenu) Stop() {
	Stop(instance)
}

func (instance *DialerMenu) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *DialerMenu) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *DialerMenu) Cleanup() {
	instance.datastore.dial_number = ""
	instance.datastore.lastAsteriskTime = time.Time{}
}

func (instance *DialerMenu) Save() {
	Save(instance.parent, instance, instance.datastore)
}

func (instance *DialerMenu) Load() {
	if loaded, ok := Load(instance.parent, instance); ok {
		instance.datastore = loaded.(*dialer_store)
	}
}

func (instance *DialerMenu) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *DialerMenu) GetStackIndex() int {
	return instance.stackIndex
}
