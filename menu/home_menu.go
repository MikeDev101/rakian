package menu

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"misc"
	"sh1107"
)

type HomeSelectionMenu struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
	selection  int
	viewOffset int
	options    [][]string
}

func (m *Menu) NewHomeSelectionMenu() *HomeSelectionMenu {
	return &HomeSelectionMenu{
		parent:    m,
		selection: 0,
		options: [][]string{
			{"Phone Book", "home/PhoneBook"},
			{"Messages", "home/Messages"},
			{"Call Register", "home/CallRegister"},
			{"Settings", "home/Settings"},
			{"Call Divert", "home/CallDivert"},
			{"Applications", "home/Apps"},
			{"Calculator", "home/Calculator"},
			{"Clock", "home/Clock"},
			{"Tones", "home/Tones"},
		},
	}
}

func (instance *HomeSelectionMenu) render() {
	display := instance.parent.Display
	label := instance.options[instance.selection][0]
	sprite := instance.parent.Sprites[instance.options[instance.selection][1]]

	display.Clear(sh1107.Black)

	font := display.Use_Font8_Normal()
	display.DrawTextAligned(0, 20, font, "Home", false, sh1107.AlignRight, sh1107.AlignNone)

	display.SetColor(sh1107.White)
	display.SetLineWidth(1)
	display.DrawLine(0, 33, 127, 33)
	display.Stroke()

	font = display.Use_Font8_Bold()
	display.DrawTextAligned(64, 105, font, "Select", false, sh1107.AlignCenter, sh1107.AlignNone)
	display.DrawTextAligned(128, 20, font, fmt.Sprintf("%d", int(instance.selection+1)), false, sh1107.AlignLeft, sh1107.AlignNone)

	font = display.Use_Font16()
	display.DrawTextAligned(64, 40, font, label, false, sh1107.AlignCenter, sh1107.AlignCenter)
	display.DrawImageAligned(sprite, 64, 84, sh1107.AlignCenter, sh1107.AlignCenter)

	display.Render()
}

func (instance *HomeSelectionMenu) handle_selection() {
	go instance.parent.PlayKey()
	switch instance.selection {
	case 0: // Phone Book
		log.Println("Phone Book selected")
		go instance.parent.PopToMenu("phonebook")
	case 3: // Settings
		log.Println("Settings selected")
		go instance.parent.PopToMenu("settings")
	case 6: // Calculator
		log.Println("Calculator selected")
		go instance.parent.PopToMenu("calculator")
	default:
		// Generic handler
		go instance.parent.Pop()
	}
}

func (instance *HomeSelectionMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *HomeSelectionMenu) ConfigureWithArgs(args ...any) {
	// Unused
	instance.Configure()
}

func (instance *HomeSelectionMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*HomeSelectionMenu).Run() before (*HomeSelectionMenu).Configure()!")
	}

	instance.render()
	instance.wg.Add(1)
	go func() {
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
					case 'U':
						if instance.selection == 0 {
							instance.selection = len(instance.options) - 1
						} else if instance.selection > 0 {
							instance.selection -= 1
						}
						instance.render()
					case 'D':
						if instance.selection < len(instance.options)-1 {
							instance.selection += 1
						} else if instance.selection == len(instance.options)-1 {
							instance.selection = 0
						}
						instance.render()
					case 'S':
						go instance.handle_selection()
						return
					case 'C':
						go instance.parent.PlayKey()
						go instance.parent.Pop()
						return
					case 'P':
						go instance.parent.PlayKey()
						go instance.parent.Push("power")
						return
					default:
						// If key is a number in range of options, select it
						if evt.Key > '0' && evt.Key <= '9' {

							// Convert evt.Key to int
							instance.selection = min(int(evt.Key-'0')-1, len(instance.options)-1)

							// Handle
							go instance.handle_selection()
							return
						}
					}

					go instance.parent.PlayKey()
				}
			}
		}
	}()
}

func (instance *HomeSelectionMenu) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Home selection menu pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *HomeSelectionMenu) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Home selection menu stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		go instance.cleanup()
	}
}

func (instance *HomeSelectionMenu) cleanup() {
	instance.selection = 0
}
