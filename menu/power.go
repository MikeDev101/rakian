package menu

import (
	"context"
	"log"
	"sync"
	"time"
)

type PowerMenu struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
	exit       bool
	options    [][]string
}

func (m *Menu) NewPowerMenu() *PowerMenu {
	return &PowerMenu{
		parent: m,
		options: [][]string{
			{"Switch off!"},
			{"Loud"},
			{"Discreet"},
			{"Silent"},
			{"Reboot"},
			{"Restart"},
			{"Flight mode"},
		},
	}
}

func (instance *PowerMenu) Label() string {
	return "Power Menu"
}

func (instance *PowerMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *PowerMenu) ConfigureWithArgs(args ...any) {
	instance.Configure()
}

func (instance *PowerMenu) Run() {
	m := instance.parent
	if !instance.configured {
		panic("Attempted to call (*PowerMenu).Run() before (*PowerMenu).Configure()!")
	}

	log.Println("🔋 Power menu started")

	if instance.exit {
		log.Println("🔋 Power menu exited")
		go instance.parent.Pop()
		return
	}

	selection := instance.parent.ShowSelector(SelectorArgs{
		SelectionClass: "power",
		Options:        instance.options,
		SelectorType:   SELECTOR_MULTI_4,
		ButtonLabel:    "Select",
	}, instance.ctx)
	if selection == nil {
		go instance.parent.Pop()
		return
	}

	switch selection.SelectionPath[0] {
	case "Switch off!":
		go m.GlobalQuit(1) // Shutdown
		return

	case "Loud":
		m.Set("CanRing", true)
		m.Set("BeepOnly", false)
		go m.SyncPersistent()
		m.RenderAnimatedAlert("ok", instance.ctx, []string{"Loud", "mode on"})
		go m.PlayAccepted()
		time.Sleep(2 * time.Second)
		go m.Pop()
		return

	case "Discreet":
		m.Set("CanRing", true)
		m.Set("BeepOnly", true)
		go m.SyncPersistent()
		m.RenderAnimatedAlert("ok", instance.ctx, []string{"Discreet", "mode on"})
		go m.PlayAccepted()
		time.Sleep(2 * time.Second)
		go m.Pop()
		return

	case "Silent":
		m.Set("CanRing", false)
		m.Set("BeepOnly", false)
		go m.SyncPersistent()
		m.RenderAnimatedAlert("ok", instance.ctx, []string{"Silent", "mode on"})
		time.Sleep(2 * time.Second)
		go m.Pop()
		return

	case "Reboot":
		go m.GlobalQuit(2) // Hard reboot
		return

	case "Restart":
		go m.GlobalQuit(3) // Soft reboot
		return

	case "Flight mode":
		if m.Phone.OK {
			if m.Phone.FlightMode {
				go m.Phone.SetFlightMode(false)
				m.RenderAnimatedAlert("ok", instance.ctx, []string{"Flight", "mode off"})
			} else {
				go m.Phone.SetFlightMode(true)
				m.RenderAnimatedAlert("ok", instance.ctx, []string{"Flight", "mode on"})
			}
			go m.PlayAccepted()
			time.Sleep(2 * time.Second)
		}

		go m.Pop()
		return
	}
}

func (instance *PowerMenu) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Power menu pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *PowerMenu) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Power menu stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		go instance.cleanup()
	}
}

func (instance *PowerMenu) cleanup() {
	instance.exit = false
}
