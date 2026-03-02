package menu

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"lcd"
	"misc"
)

type HomeSelectionMenu struct {
	ctx        context.Context
	animCtx    context.Context
	animCancel context.CancelFunc
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
			{"Phonebook", "phonebook"},
			{"Messages", "messages"},
			{"Chats", "chats"},
			{"Call register", "call_register"},
			{"Settings", "settings"},
			{"Call divert", "call_divert"},
			{"Apps", "apps"},
			{"Calculator", "calculator"},
			{"Clock", "clock"},
			{"Notes", "notes"},
			{"Tones", "tones"},
			{"SIM tools", "sim_tools"},
			{"Drawing", "drawing"},
		},
	}
}

func (instance *HomeSelectionMenu) Label() string {
	return "Home Selection Menu"
}

func (instance *HomeSelectionMenu) render() {
	m := instance.parent
	display := m.Display
	label := instance.options[instance.selection][0]

	defer display.DrawLock.Unlock()
	display.DrawLock.Lock()

	display.Clear(lcd.White)
	m.RenderFooter("Select", true)

	font := display.Use_Font_Small_Bold()
	display.DrawTextAligned(84, 0, font, fmt.Sprintf("%d", int(instance.selection+1)), false, lcd.AlignLeft, lcd.AlignNone)

	font = display.Use_Font_Large_Bold()
	display.DrawTextAligned(40, 8, font, label, false, lcd.AlignCenter, lcd.AlignBelow)

	anim := instance.options[instance.selection][1]
	display.Render()
	go display.PlayAnimation(instance.animCtx, anim, 40, 36, lcd.AlignCenter, lcd.AlignAbove)
}

func (instance *HomeSelectionMenu) handle_selection() {
	go instance.parent.PlayKey()
	selection := instance.options[instance.selection][0]
	log.Printf("Selected: %s", selection)
	switch selection {
	case "Phonebook":
		go instance.parent.PopToMenu("phonebook")
	case "Settings":
		go instance.parent.PopToMenu("settings")
	case "Calculator":
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
	instance.animCtx, instance.animCancel = context.WithCancel(instance.ctx)
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
	instance.wg.Go(func() {
		for {
			select {
			case <-instance.ctx.Done():
				return
			case evt := <-instance.parent.KeypadEvents:
				if evt.State {

					instance.parent.Timers["keypad"].Reset()
					instance.parent.Timers["screensaver"].Reset()
					instance.parent.Display.On()
					misc.KeyLightsOn()

					switch evt.Key {
					case '*':
						instance.animCancel()
						instance.animCtx, instance.animCancel = context.WithCancel(instance.ctx)
						instance.parent.Set("KeypadLocked", true)
						instance.parent.RenderAnimatedAlert("keypad_locked", instance.animCtx, []string{"Keypad", "locked"})
						go instance.parent.PlayAccepted()
						time.Sleep(time.Second * 2)
						instance.animCancel()
						go instance.parent.Pop()
						return

					case 'U':
						instance.animCancel()
						instance.animCtx, instance.animCancel = context.WithCancel(instance.ctx)
						if instance.selection == 0 {
							instance.selection = len(instance.options) - 1
						} else if instance.selection > 0 {
							instance.selection -= 1
						}
						instance.render()
					case 'D':
						instance.animCancel()
						instance.animCtx, instance.animCancel = context.WithCancel(instance.ctx)
						if instance.selection < len(instance.options)-1 {
							instance.selection += 1
						} else if instance.selection == len(instance.options)-1 {
							instance.selection = 0
						}
						instance.render()
					case 'S':
						instance.animCancel()
						go instance.handle_selection()
						return
					case 'C':
						instance.animCancel()
						go instance.parent.PlayKey()
						go instance.parent.Pop()
						return
					case 'P':
						instance.animCancel()
						go instance.parent.PlayKey()
						go instance.parent.Push("power")
						return
					default:
						// If key is a number in range of options, select it
						if evt.Key > '0' && evt.Key <= '9' {
							instance.animCancel()

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
	})
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
