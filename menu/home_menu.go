package menu

import (
	"context"
	"log"
	"sync"
	"time"

	"gfx"
)

type home_selection_store struct {
	selection  int
	viewOffset int
}

type HomeSelectionMenu struct {
	ctx        context.Context
	animCtx    context.Context
	animCancel context.CancelFunc
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
	options    [][]string
	datastore  *home_selection_store
	stackIndex int
}

func (m *Menu) NewHomeSelectionMenu() MenuInstance {
	return &HomeSelectionMenu{
		parent: m,
		datastore: &home_selection_store{
			selection: 0,
		},
		options: [][]string{
			{"Phone book", "phone_book"},
			{"Messages", "messages"},
			{"Chat", "chat"},
			{"Call register", "call_register"},
			{"Settings", "settings"},
			{"Call divert", "call_divert"},
			{"Apps", "apps"},
			{"Calculator", "calculator"},
			{"Clock", "clock"},
			{"Reminders", "reminders"},
			{"Profiles", "profiles"},
			{"Tones", "tones"},
			{"SIM tools", "sim_tools"},
			{"Drawing", "drawing"},
			{"Radio", "radio"},
		},
	}
}

func (instance *HomeSelectionMenu) Label() string {
	return "Home Selection Menu"
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

func (instance *HomeSelectionMenu) Pause() {
	Pause(instance)
}

func (instance *HomeSelectionMenu) Stop() {
	Stop(instance)
}

func (instance *HomeSelectionMenu) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *HomeSelectionMenu) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *HomeSelectionMenu) Cleanup() {
	instance.datastore.selection = 0
	instance.datastore.viewOffset = 0
}

func (instance *HomeSelectionMenu) Save() {
	Save(instance.parent, instance, instance.datastore)
}

func (instance *HomeSelectionMenu) Load() {
	if loaded, ok := Load(instance.parent, instance); ok {
		instance.datastore = loaded.(*home_selection_store)
	}
}

func (instance *HomeSelectionMenu) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *HomeSelectionMenu) GetStackIndex() int {
	return instance.stackIndex
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
					instance.parent.Keypad.KeyLightsOn()

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
						if instance.datastore.selection == 0 {
							instance.datastore.selection = len(instance.options) - 1
						} else if instance.datastore.selection > 0 {
							instance.datastore.selection -= 1
						}
						instance.render()
					case 'D':
						instance.animCancel()
						instance.animCtx, instance.animCancel = context.WithCancel(instance.ctx)
						if instance.datastore.selection < len(instance.options)-1 {
							instance.datastore.selection += 1
						} else if instance.datastore.selection == len(instance.options)-1 {
							instance.datastore.selection = 0
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
							instance.datastore.selection = min(int(evt.Key-'0')-1, len(instance.options)-1)

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

func (instance *HomeSelectionMenu) render() {
	m := instance.parent
	display := m.Display
	label := instance.options[instance.datastore.selection][0]

	defer display.Unlock()
	display.Lock()

	display.Clear(display.Primary())
	m.RenderFooter("Select", true)

	font := display.Use_Font_Large_Bold()
	display.DrawTextAligned(40, 8, font, label, false, gfx.AlignCenter, gfx.AlignBelow)

	anim := instance.options[instance.datastore.selection][1]

	m.RenderSelectorScrollbar(len(instance.options), instance.datastore.selection, true)

	display.Render()
	go display.PlayAnimation(instance.animCtx, anim, 40, 36, gfx.AlignCenter, gfx.AlignAbove)
}

func (instance *HomeSelectionMenu) handle_selection() {
	go instance.parent.PlayKey()
	menuID := instance.options[instance.datastore.selection][1]
	log.Printf("Selected: %s", menuID)

	if menuID == "sim_tools" || menuID == "drawing" {
		// Not implemented yet
		go instance.parent.Pop()
		return
	}

	go instance.parent.PopToMenu(menuID)
}
