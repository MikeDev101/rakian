package menu

import (
	"context"
	"log"
	"sync"
)

const (
	PhonebookActionExit = iota
	PhonebookActionShowSelector
	PhonebookActionSubmenuPushed
)

type PhonebookMenu struct {
	ctx          context.Context
	configured   bool
	cancelFn     context.CancelFunc
	parent       *Menu
	wg           sync.WaitGroup
	default_args *SelectorArgs
	stackIndex   int
}

func (m *Menu) NewPhonebookMenu() MenuInstance {
	return &PhonebookMenu{
		parent: m,
		default_args: &SelectorArgs{
			Title:          "Phone book",
			SelectionClass: "phonebook.main",
			SelectorType:   SELECTOR_MULTI_3,
			ShowTitle:      true,
			Options: [][]string{
				{"Search"},
				{"Service Numbers"},
				{"Erase"},
				{"Erase all"},
				{"Edit"},
				{"Assign Tone"},
			},
			ButtonLabel:            "Select",
			ShowPathInTitle:        true,
			AllowNumberKeyShortcut: true,
			PersistLastState:       true,
		},
	}
}

func (*PhonebookMenu) Label() string {
	return "Phone book Menu"
}

func (instance *PhonebookMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *PhonebookMenu) ConfigureWithArgs(args ...any) {
	instance.Configure()
}
func (instance *PhonebookMenu) Pause() {
	Pause(instance)
}

func (instance *PhonebookMenu) Stop() {
	Stop(instance)
}

func (instance *PhonebookMenu) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *PhonebookMenu) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *PhonebookMenu) Cleanup() {}

func (instance *PhonebookMenu) Save() {}

func (instance *PhonebookMenu) Load() {}

func (instance *PhonebookMenu) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *PhonebookMenu) GetStackIndex() int {
	return instance.stackIndex
}

func (instance *PhonebookMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*PhonebookMenu).Run() before (*PhonebookMenu).Configure()!")
	}

	m := instance.parent
	display := m.Display
	defer display.Unlock()
	display.Lock()

	log.Println("📱 Phonebook started")

	for {
		selection := m.ShowSelector(*instance.default_args, instance.ctx)
		if selection == nil {
			m.Pop()
			return
		}

		if selection.SelectionClass == "phonebook.main" {
			action := instance.phonebookMain(selection.SelectionPath)
			switch action {
			case PhonebookActionExit:
				m.Pop()
				return
			case PhonebookActionSubmenuPushed:
				// Do nothing
			}
		}
	}
}

func (instance *PhonebookMenu) phonebookMain(selection_path []string) int {
	switch selection_path[len(selection_path)-1] {
	case "Search":
		// Show a Text entry prompt to search the phonebook
	case "Service Numbers":
		// Show a selector prompt with all the loaded EmergencyNumbers from the modem
	case "Erase":
		// Show a list of all contacts, allowing the user to erase them one at a time
	case "Edit":
		// Show a list of all contacts, allowing the user to edit one
	case "Assign Tone":
		// Show a list of all contacts, allowing the user to assign a tone to one
	}

	return PhonebookActionShowSelector
}
