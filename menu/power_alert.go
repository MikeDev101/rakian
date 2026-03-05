package menu

import (
	"context"
	"sync"
	"time"

	"misc"
)

type GenericAlert struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
	events     []*GenericAlertConfig
	configLock sync.Mutex
	stackIndex int
}

const (
	BeepTypeNone         = 0
	BeepTypeGeneric      = 1
	BeepTypeBatteryAlert = 2
	BeepTypeBatteryDead  = 3
)

type GenericAlertConfig struct {
	Icon     string
	Label    []string
	BeepType int
}

func (m *Menu) NewGenericAlert() *GenericAlert {
	return &GenericAlert{
		parent: m,
	}
}

func (*GenericAlert) Label() string {
	return "Generic Alert"
}

func (instance *GenericAlert) Configure() {
	// Do not reset context here to avoid clobbering active context before Stop() is called.
	instance.configured = true
}

func (instance *GenericAlert) ConfigureWithArgs(args ...any) {
	// Check if we have args
	if len(args) > 0 {

		// Expect our arg to be a GenericAlertConfig.
		selection, ok := args[0].(*GenericAlertConfig)
		if !ok {
			panic("(*GenericAlert).ConfigureWithArgs() Type error: argument must be a *GenericAlertConfig type")
		}

		instance.configLock.Lock()
		instance.events = append(instance.events, selection)
		instance.configLock.Unlock()
	}

	instance.Configure()
}

func (instance *GenericAlert) Pause() {
	Pause(instance)
}

func (instance *GenericAlert) Stop() {
	Stop(instance)
}

func (instance *GenericAlert) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *GenericAlert) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *GenericAlert) Cleanup() {}

func (instance *GenericAlert) Save() {}

func (instance *GenericAlert) Load() {}

func (instance *GenericAlert) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *GenericAlert) GetStackIndex() int {
	return instance.stackIndex
}

func (instance *GenericAlert) Run() {
	if !instance.configured {
		panic("Attempted to call (*GenericAlert).Run() before (*GenericAlert).Configure()!")
	}

	m := instance.parent
	display := m.Display
	defer display.Unlock()
	display.Lock()

	// Reset context
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)

	instance.configLock.Lock()
	currentEvents := instance.events
	instance.events = nil
	instance.configLock.Unlock()

	for _, event := range currentEvents {
		instance.parent.RenderAlert(event.Icon, event.Label)

		if instance.parent.Get("CanRing").(bool) || instance.parent.Get("BeepOnly").(bool) {
			switch event.BeepType {
			case BeepTypeGeneric:
				go misc.PlayBeep(instance.parent.Player, instance.ctx)

			case BeepTypeBatteryAlert:
				go misc.PlayLowBattery(instance.parent.Player, instance.ctx)

			case BeepTypeBatteryDead:
				go misc.PlayDeadBattery(instance.parent.Player, instance.ctx)
			}
		}

		timer := time.NewTimer(3 * time.Second)

		select {
		case <-instance.ctx.Done():
			timer.Stop()

		case <-timer.C:

		case evt := <-instance.parent.KeypadEvents:
			if evt.State {
				timer.Stop()
				go instance.parent.PlayKey()
			}
		}
	}

	instance.parent.Timers["screensaver"].Restart()
	instance.parent.Timers["keypad"].Restart()
}
