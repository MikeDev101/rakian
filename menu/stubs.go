package menu

import (
	"context"
	"sync"
)

type StubMenu struct {
	ctx          context.Context
	configured   bool
	cancelFn     context.CancelFunc
	parent       *Menu
	wg           sync.WaitGroup
	default_args SelectorArgs
	stackIndex   int
}

func (instance *StubMenu) Label() string {
	return instance.default_args.Title + " Stub"
}

func (instance *StubMenu) Configure() {
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *StubMenu) ConfigureWithArgs(args ...any) {
	instance.Configure()
}

func (instance *StubMenu) Pause() { Pause(instance) }
func (instance *StubMenu) Stop()  { Stop(instance) }
func (instance *StubMenu) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *StubMenu) WaitGroup() *sync.WaitGroup { return &instance.wg }
func (instance *StubMenu) Cleanup()                   {}
func (instance *StubMenu) Save()                      {}
func (instance *StubMenu) Load()                      {}
func (instance *StubMenu) SetStackIndex(i int)        { instance.stackIndex = i }
func (instance *StubMenu) GetStackIndex() int         { return instance.stackIndex }

func (instance *StubMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*StubMenu).Run() before (*StubMenu).Configure()!")
	}

	m := instance.parent
	display := m.Display
	defer display.Unlock()
	display.Lock()

	for {
		selection := m.ShowSelector(instance.default_args, instance.ctx)
		if selection == nil {
			m.ToStart()
			return
		}

		// It's a stub, so just do nothing on selection, and Pop back out
		// Or maybe stay in the menu? Let's just Pop back out.
		// For a more realistic stub, we might show a "Not implemented" alert, but simple pop is fine.
		m.ToStart()
	}
}

func (m *Menu) NewMessagesMenu() MenuInstance {
	return &StubMenu{
		parent: m,
		default_args: SelectorArgs{
			Title:          "Messages",
			SelectionClass: "messages.main",
			SelectorType:   SELECTOR_SIMPLE,
			ShowTitle:      true,
			Options: [][]string{
				{"Write message"},
				{"Read messages"},
				{"Sent items"},
				{"Templates"},
				{"Delivery reports"},
				{"Message settings", "Set 1", "Common"},
				{"Set 1", "Message centre number", "Messages sent as", "Message validity"},
				{"Common", "Delivery reports", "Reply via same centre", "Character support"},
				{"Voice mailbox number"},
				{"Listen to voice messages"},
				{"Broadcast messages"},
				{"Service command editor"},
			},
			ButtonLabel:            "Select",
			ShowPathInTitle:        true,
			AllowNumberKeyShortcut: true,
			PersistLastState:       true,
			ShowScrollbar:          true,
			ShowScrollbarPos:       true,
		},
	}
}

func (m *Menu) NewChatMenu() MenuInstance {
	return &StubMenu{
		parent: m,
		default_args: SelectorArgs{
			Title:          "Chat",
			SelectionClass: "chat.main",
			SelectorType:   SELECTOR_MULTI_3,
			ShowTitle:      true,
			Options: [][]string{
				{"Start chat"},
				{"Chat history"},
				{"Chat name"},
			},
			ButtonLabel:            "Select",
			ShowPathInTitle:        true,
			AllowNumberKeyShortcut: true,
			PersistLastState:       true,
		},
	}
}

func (m *Menu) NewCallRegisterMenu() MenuInstance {
	return &StubMenu{
		parent: m,
		default_args: SelectorArgs{
			Title:          "Call register",
			SelectionClass: "call_register.main",
			SelectorType:   SELECTOR_SIMPLE,
			ShowTitle:      true,
			Options: [][]string{
				{"Missed calls"},
				{"Received calls"},
				{"Dialled numbers"},
				{"Erase recent call lists"},
				{"Show call duration"},
				{"Show call costs", "Call cost settings"},
				{"Call cost settings"}, // Leaves an empty sub-level
			},
			ButtonLabel:            "Select",
			ShowPathInTitle:        true,
			AllowNumberKeyShortcut: true,
			PersistLastState:       true,
			ShowScrollbar:          true,
			ShowScrollbarPos:       true,
		},
	}
}

func (m *Menu) NewProfilesMenu() MenuInstance {
	return &StubMenu{
		parent: m,
		default_args: SelectorArgs{
			Title:          "Profiles",
			SelectionClass: "profiles.main",
			SelectorType:   SELECTOR_SIMPLE,
			ShowTitle:      true,
			Options: [][]string{
				{"Select"},
				{"Customize"},
				{"Rename"},
			},
			ButtonLabel:            "Select",
			ShowPathInTitle:        true,
			AllowNumberKeyShortcut: true,
			PersistLastState:       true,
			ShowScrollbar:          true,
			ShowScrollbarPos:       true,
		},
	}
}

func (m *Menu) NewCallDivertMenu() MenuInstance {
	return &StubMenu{
		parent: m,
		default_args: SelectorArgs{
			Title:          "Call divert",
			SelectionClass: "call_divert.main",
			SelectorType:   SELECTOR_SIMPLE,
			ShowTitle:      true,
			Options: [][]string{
				{"Divert all voice calls"},
				{"Divert if busy"},
				{"Divert if not answered"},
				{"Divert if out of reach"},
				{"Divert if not available"},
				{"Cancel all diverts"},
			},
			ButtonLabel:            "Select",
			ShowPathInTitle:        true,
			AllowNumberKeyShortcut: true,
			PersistLastState:       true,
			ShowScrollbar:          true,
			ShowScrollbarPos:       true,
		},
	}
}

func (m *Menu) NewAppsMenu() MenuInstance {
	return &StubMenu{
		parent: m,
		default_args: SelectorArgs{
			Title:          "Apps",
			SelectionClass: "apps.main",
			SelectorType:   SELECTOR_SIMPLE,
			ShowTitle:      true,
			Options: [][]string{
				{"Snake"},
				{"Space Impact"},
				{"Bumper"},
			},
			ButtonLabel:            "Select",
			ShowPathInTitle:        true,
			AllowNumberKeyShortcut: true,
			PersistLastState:       true,
			ShowScrollbar:          true,
			ShowScrollbarPos:       true,
		},
	}
}

func (m *Menu) NewClockMenu() MenuInstance {
	return &StubMenu{
		parent: m,
		default_args: SelectorArgs{
			Title:          "Clock",
			SelectionClass: "clock.main",
			SelectorType:   SELECTOR_SIMPLE,
			ShowTitle:      true,
			Options: [][]string{
				{"Alarm clock"},
				{"Clock settings"},
				{"Date setting"},
				{"Stopwatch"},
				{"Countdown timer"},
				{"Automatic update of date and time"},
			},
			ButtonLabel:            "Select",
			ShowPathInTitle:        true,
			AllowNumberKeyShortcut: true,
			PersistLastState:       true,
			ShowScrollbar:          true,
			ShowScrollbarPos:       true,
		},
	}
}

func (m *Menu) NewRemindersMenu() MenuInstance {
	return &StubMenu{
		parent: m,
		default_args: SelectorArgs{
			Title:          "Reminders",
			SelectionClass: "reminders.main",
			SelectorType:   SELECTOR_SIMPLE,
			ShowTitle:      true,
			Options: [][]string{
				{"Add new", "Alarm on", "Alarm off"},
				{"Alarm on"},
				{"Alarm off"},
				{"View all"},
				{"Erase", "One by one", "All at once"},
				{"One by one"},
				{"All at once"},
			},
			ButtonLabel:            "Select",
			ShowPathInTitle:        true,
			AllowNumberKeyShortcut: true,
			PersistLastState:       true,
			ShowScrollbar:          true,
			ShowScrollbarPos:       true,
		},
	}
}

func (m *Menu) NewTonesMenu() MenuInstance {
	return &StubMenu{
		parent: m,
		default_args: SelectorArgs{
			Title:          "Tones",
			SelectionClass: "tones.main",
			SelectorType:   SELECTOR_SIMPLE,
			ShowTitle:      true,
			Options: [][]string{
				{"Incoming call alert"},
				{"Ringing tone"},
				{"Ringing volume"},
				{"Vibrating alert"},
				{"Message alert tone"},
				{"Keypad tones"},
				{"Warning tones"},
				{"Composer"},
			},
			ButtonLabel:            "Select",
			ShowPathInTitle:        true,
			AllowNumberKeyShortcut: true,
			PersistLastState:       true,
			ShowScrollbar:          true,
			ShowScrollbarPos:       true,
		},
	}
}
