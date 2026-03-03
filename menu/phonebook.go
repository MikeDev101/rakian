package menu

import (
	"context"
	"log"
	"sync"
	"time"
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
}

func (*PhonebookMenu) Label() string {
	return "PhoneBook Menu"
}

func (m *Menu) NewPhonebookMenu() *PhonebookMenu {
	return &PhonebookMenu{
		parent: m,
		default_args: &SelectorArgs{
			Title:          "Phonebook",
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

func (instance *PhonebookMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *PhonebookMenu) ConfigureWithArgs(args ...any) {
	instance.Configure()
}

func (instance *PhonebookMenu) PhonebookMain(selection_path []string) int {
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
			action := instance.PhonebookMain(selection.SelectionPath)
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

func (instance *PhonebookMenu) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Phonebook handler pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *PhonebookMenu) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Phonebook handler stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		go instance.cleanup()
	}
}

func (instance *PhonebookMenu) cleanup() {
}
