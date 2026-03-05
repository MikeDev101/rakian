package menu

import (
	"context"
	"log"
	"misc"
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
	stackIndex int
}

func (m *Menu) NewPowerMenu() MenuInstance {
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

func (instance *PowerMenu) Pause() {
	Pause(instance)
}

func (instance *PowerMenu) Stop() {
	Stop(instance)
}

func (instance *PowerMenu) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *PowerMenu) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *PowerMenu) Cleanup() {
	instance.exit = false
}

func (instance *PowerMenu) Save() {}

func (instance *PowerMenu) Load() {}

func (instance *PowerMenu) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *PowerMenu) GetStackIndex() int {
	return instance.stackIndex
}

func (instance *PowerMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*PowerMenu).Run() before (*PowerMenu).Configure()!")
	}

	m := instance.parent
	display := m.Display
	defer display.Unlock()
	display.Lock()

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
		if m.Phone.OK() {
			apn := m.Get("APN").(string)
			if m.Phone.IsFlightMode() {
				go m.RenderAnimatedAlert("ok", instance.ctx, []string{"Leaving", "flight", "mode"})
				go m.PlayAccepted()
				m.Phone.SetFlightMode(false)
				if m.Get("WasDataEnabled").(bool) {
					misc.SetCellularDataState(apn, true)
				}
			} else {
				go m.RenderAnimatedAlert("ok", instance.ctx, []string{"Entering", "flight", "mode"})
				go m.PlayAccepted()
				misc.SetCellularDataState(apn, false)
				m.Phone.SetFlightMode(true)
			}
			time.Sleep(2 * time.Second)
		}
		go m.Pop()
		return
	}
}
