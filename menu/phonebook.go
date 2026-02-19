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
	ctx               context.Context
	configured        bool
	cancelFn          context.CancelFunc
	parent            *Menu
	wg                sync.WaitGroup
	process_selection bool
	selection_class   string
	selection_path    []string
	options           [][]string
}

func (*PhonebookMenu) Label() string {
	return "PhoneBook Menu"
}

func (m *Menu) NewPhonebookMenu() *PhonebookMenu {
	return &PhonebookMenu{
		parent:            m,
		process_selection: false,
		selection_path:    []string{},
		options: [][]string{
			{"Search"},
			{"Service Numbers"},
			{"Erase"},
			{"Edit"},
			{"Assign Tone"},
		},
	}
}

func (instance *PhonebookMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *PhonebookMenu) ConfigureWithArgs(args ...any) {

	// Check if we have args
	if len(args) > 0 {

		// Most likely our arg is a SelectorReturn from the selector.
		selection, ok := args[0].(*SelectorReturn)
		if !ok {
			panic("(*PhonebookMenu).ConfigureWithArgs() Type error: argument must be a *SelectorReturn type")
		}

		instance.process_selection = true
		instance.selection_path = selection.SelectionPath
		instance.selection_class = selection.SelectionClass
	}

	instance.Configure()
}

func (instance *PhonebookMenu) PhonebookMain(selection_path []string) int {
	switch selection_path[len(selection_path)-1] {
	case "Search":
		// TODO
	case "Service Numbers":
		// TODO
	case "Erase":
		// TODO
	case "Edit":
		// TODO
	case "Assign Tone":
		// TODO
	}

	return PhonebookActionShowSelector
}

func (instance *PhonebookMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*PhonebookMenu).Run() before (*PhonebookMenu).Configure()!")
	}

	log.Println("📱 Phonebook started")

	if !instance.process_selection {
		// Start the selector with the base phonebook menu
		log.Println("📱 Phonebook switching to selector")
		go instance.parent.PushWithArgs("selector", &SelectorArgs{
			Title:                      "Phonebook",
			SelectionClass:             "phonebook.main",
			Options:                    instance.options,
			ButtonLabel:                "Select",
			VisibleRows:                3,
			ShowPathInTitle:            true,
			ShowElemNumberInTitle:      true,
			ShowElemNumbersInSelection: true,
			AllowNumberKeyShortcut:     true,
			PersistLastState:           true,
		})
		return
	}

	log.Printf("📱 Phonebook %s: %s", instance.selection_class, instance.selection_path)

	// Process selected setting option
	switch instance.selection_class {
	case "phonebook.main":

		// Exit to main menu
		if len(instance.selection_path) == 0 {
			log.Println("📱 Phonebook path selected is empty, exiting...")
			instance.parent.Pop()
			return
		}

		// Launch phonebook main menu
		action := instance.PhonebookMain(instance.selection_path)

		switch action {
		case PhonebookActionExit:
			log.Println("📱 Phonebook exiting")
			instance.parent.Pop()
			return
		case PhonebookActionSubmenuPushed:
			// Do nothing, wait for submenu to return
			return
		}
	}

	instance.process_selection = false
	log.Println("📱 Phonebook switching back to selector")
	go instance.parent.PushWithArgs("selector", &SelectorArgs{
		Title:                      "Phonebook",
		SelectionClass:             "phonebook.main",
		Options:                    instance.options,
		ButtonLabel:                "Select",
		VisibleRows:                3,
		ShowPathInTitle:            true,
		ShowElemNumberInTitle:      true,
		ShowElemNumbersInSelection: true,
		AllowNumberKeyShortcut:     true,
		PersistLastState:           true,
	})
}

func (instance *PhonebookMenu) Pause() {
	instance.process_selection = true
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Phonebook handler pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *PhonebookMenu) Stop() {
	instance.process_selection = false
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Phonebook handler stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		go instance.cleanup()
	}
}

func (instance *PhonebookMenu) cleanup() {
	instance.process_selection = false
	instance.selection_path = []string{}
}
