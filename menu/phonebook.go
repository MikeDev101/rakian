package menu

import (
	"context"
	"log"
	"sync"
	"time"
)

type PhonebookMenu struct {
	ctx               context.Context
	configured        bool
	cancelFn          context.CancelFunc
	parent            *Menu
	wg                sync.WaitGroup
	process_selection bool
	selection_path    []string
}

func (m *Menu) NewPhonebookMenu() *PhonebookMenu {
	return &PhonebookMenu{
		parent:            m,
		process_selection: false,
		selection_path:    []string{},
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
	}

	instance.Configure()
}

func (instance *PhonebookMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*PhonebookMenu).Run() before (*PhonebookMenu).Configure()!")
	}

	log.Println("📱 Phonebook started")

	if instance.process_selection {
		instance.process_selection = false
		log.Println("📱 Phonebook path selected: ", instance.selection_path)

		// TODO: process selected setting option

		go instance.parent.Pop()
		return
	} else {
		// Start the selector with the base phonebook menu
		log.Println("📱 Phonebook switching to selector")
		go instance.parent.PushWithArgs("selector", &SelectorArgs{
			Title: "Phonebook",
			Options: [][]string{
				{"Search"},
				{"Service Numbers"},
				{"Erase"},
				{"Edit"},
				{"Assign Tone"},
			},
			ButtonLabel:                "Select",
			VisibleRows:                3,
			ShowPathInTitle:            true,
			ShowElemNumberInTitle:      true,
			ShowElemNumbersInSelection: true,
		})
	}

	instance.wg.Go(func() {
		<-instance.ctx.Done()
		log.Println("📱 Phonebook exiting...")
	})
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
	instance.process_selection = false
	instance.selection_path = []string{}
}
