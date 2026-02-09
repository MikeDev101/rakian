package menu

import (
	"context"
	"log"
	"sync"
	"time"

	"misc"
	"sh1107"
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
}

func (m *Menu) NewDialerMenu() *DialerMenu {
	return &DialerMenu{
		parent:           m,
		lastAsteriskTime: time.Now(),
		pressStart:       make(map[rune]time.Time),
	}
}

func (instance *DialerMenu) render() {
	instance.parent.Display.Clear(sh1107.Black)
	instance.parent.Display.DrawText(0, 40, instance.parent.Display.Use_Font16(), instance.dial_number, false)
	instance.parent.Display.DrawTextAligned(64, 105, instance.parent.Display.Use_Font8_Bold(), "Call", false, sh1107.AlignCenter, sh1107.AlignNone)
	instance.parent.Display.Render()
}

func (instance *DialerMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *DialerMenu) ConfigureWithArgs(args ...any) {
	// Unused
	instance.Configure()
}

func (instance *DialerMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*DialerMenu).Run() before (*DialerMenu).Configure()!")
	}

	if instance.parent.Get("InitialKey") != ' ' {
		instance.dial_number = ""
		instance.dial_number += string(instance.parent.Get("InitialKey").(rune))
		instance.parent.Set("InitialKey", ' ')
	}
	instance.render()

	instance.wg.Add(1)
	defer instance.wg.Done()
	for {
		select {
		case <-instance.ctx.Done():
			return

		case evt := <-instance.parent.KeypadEvents:
			if evt.State {

				instance.parent.Timers["keypad"].Reset()
				instance.parent.Timers["oled"].Reset()
				instance.parent.Display.On()
				misc.KeyLightsOn()
				switch evt.Key {
				case '*':
					go instance.parent.PlayKey()
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
					go instance.parent.PlayKey()

					// Delete last key from dial_number
					runes := []rune(instance.dial_number)
					if len(runes) > 0 {
						instance.dial_number = string(runes[:len(runes)-1])
					}

					// Check if
					if len(runes) == 0 {
						go instance.parent.Pop()
						return
					} else {
						instance.render()
					}

				case 'P':
					go instance.parent.PlayKey()
					go instance.parent.Push("power")
					return

				case 'U':
					go instance.parent.PlayKey()
				case 'D':
					go instance.parent.PlayKey()
				case 'S':
					if len(instance.dial_number) == 0 {
						continue
					}

					if instance.parent.Modem == nil {
						instance.ExitWithAlert([]string{"No", "service!"})
						return

					} else if instance.parent.Modem.FlightMode {
						instance.ExitWithAlert([]string{"Airplane", "mode", "enabled."})
						return

					} else if !instance.parent.Modem.Connected {
						instance.ExitWithAlert([]string{"No", "service!"})
						return

					} else if !instance.parent.Modem.SimCardInserted {
						instance.ExitWithAlert([]string{"Insert a", "SIM card", "to continue."})
						return

					} else {
						go instance.parent.PlayKey()
						instance.parent.Modem.Dial(instance.dial_number)
						return
					}

				default:
					instance.dial_number += string(evt.Key)
					instance.render()
					go instance.parent.PlayKey()
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
}

func (instance *DialerMenu) ExitWithAlert(msg []string) {
	instance.parent.RenderAlert("prohibited", msg)
	go instance.parent.PlayAlert()
	timers.SleepWithContext(3*time.Second, instance.ctx)
	go instance.parent.Pop()
}
