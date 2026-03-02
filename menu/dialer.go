package menu

import (
	"context"
	"log"
	"slices"
	"sync"
	"time"

	"lcd"
	"misc"
	"timers"
)

type DialerMenu struct {
	ctx              context.Context
	configured       bool
	cancelFn         context.CancelFunc
	parent           *Menu
	wg               sync.WaitGroup
	dial_number      string
	lastAsteriskTime time.Time
	pressStart       map[rune]time.Time
	initialKey       rune
}

func (m *Menu) NewDialerMenu() *DialerMenu {
	return &DialerMenu{
		parent:           m,
		lastAsteriskTime: time.Now(),
		pressStart:       make(map[rune]time.Time),
	}
}

func (instance *DialerMenu) render() {
	m := instance.parent
	display := m.Display
	display.Clear(lcd.White)
	m.RenderStateCommon()
	display.DrawTextWrapped(8, 10, 80, 40, display.Use_Font_Large_Bold(), instance.dial_number, false, lcd.WrapRight, lcd.WrapUp)
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
		instance.dial_number = string(key)
		instance.initialKey = 0
		go m.playDTMF(key)
		if key == '*' {
			instance.lastAsteriskTime = time.Now()
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
			if evt.State {

				m.Timers["keypad"].Reset()
				m.Timers["screensaver"].Reset()
				m.Display.On()
				misc.KeyLightsOn()
				switch evt.Key {
				case '*':
					go m.playDTMF('*')
					now := time.Now()
					if now.Sub(instance.lastAsteriskTime) <= 750*time.Millisecond {
						// Replace last '*' with '+'
						runes := []rune(instance.dial_number)
						if len(runes) > 0 && runes[len(runes)-1] == '*' {
							runes[len(runes)-1] = '+'
							instance.dial_number = string(runes)
						} else {
							instance.dial_number += "*"
						}
					} else {
						instance.dial_number += "*"
					}
					instance.lastAsteriskTime = now
					instance.render()

				case 'C':
					go m.PlayKey()

					// Delete last key from dial_number
					runes := []rune(instance.dial_number)
					if len(runes) > 0 {
						instance.dial_number = string(runes[:len(runes)-1])
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
					if len(instance.dial_number) == 0 {
						continue
					}

					if !m.Phone.OK {
						instance.ExitWithAlert([]string{"No", "service!"})
						go m.PlayAlert()
						time.Sleep(3 * time.Second)
						go m.Pop()
						return
					}

					if m.Phone.FlightMode {
						instance.ExitWithAlert([]string{"Flight", "mode", "enabled."})
						go m.PlayAlert()
						time.Sleep(3 * time.Second)
						go m.Pop()
						return
					}

					// Ignore registration if the number is in the service/emergency numbers list
					if slices.Contains(m.Phone.EmergencyNumbers, instance.dial_number) {
						m.RenderAlert("", []string{"Attempting", "emergency", "call"})
						go m.PlayAlert()

						time.Sleep(3 * time.Second)

						session, err := m.Phone.PlaceCall(instance.dial_number)
						if err != nil {
							m.RenderAlert("prohibited", []string{"Call", "failed"})
							go m.PlayAlert()
							time.Sleep(3 * time.Second)
							go m.Pop()
							return
						}

						go m.ToMenuWithArgs("phone", session)
						return
					}

					if !m.Phone.Registered {
						instance.ExitWithAlert([]string{"No", "service!"})
						go m.PlayAlert()
						time.Sleep(3 * time.Second)
						go m.Pop()
						return
					}

					session, err := m.Phone.PlaceCall(instance.dial_number)
					if err != nil {
						m.RenderAlert("prohibited", []string{"Call", "failed"})
						go m.PlayAlert()
						time.Sleep(3 * time.Second)
						go m.Pop()
						return
					}

					go m.ToMenuWithArgs("phone", session)
					return

				default:
					if len(instance.dial_number) < 18 {
						instance.dial_number += string(evt.Key)
						instance.render()
					}
					go m.playDTMF(evt.Key)
				}
			}
		}
	}
}

func (instance *DialerMenu) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Dialer menu pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *DialerMenu) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Dialer menu stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		instance.cleanup()
	}
}

func (instance *DialerMenu) cleanup() {
	instance.dial_number = ""
	instance.pressStart = make(map[rune]time.Time)
	instance.initialKey = 0
}

func (instance *DialerMenu) ExitWithAlert(msg []string) {
	instance.parent.RenderAlert("prohibited", msg)
	go instance.parent.PlayAlert()
	timers.SleepWithContext(3*time.Second, instance.ctx)
	go instance.parent.Pop()
}
