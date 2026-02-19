package menu

import (
	"context"
	"log"
	"sync"
	"time"

	// "sh1107"
	// "timers"
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

func (*GenericAlert) Label() string {
	return "Generic Alert"
}

func (m *Menu) NewGenericAlert() *GenericAlert {
	return &GenericAlert{
		parent: m,
	}
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

func (instance *GenericAlert) Run() {
	if !instance.configured {
		panic("Attempted to call (*GenericAlert).Run() before (*GenericAlert).Configure()!")
	}

	// Wait for display to be ready
	instance.parent.Display.Ready()

	// Reset context
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)

	instance.configLock.Lock()
	currentEvents := instance.events
	instance.events = nil
	instance.configLock.Unlock()

	for _, event := range currentEvents {
		instance.parent.RenderAlert(event.Icon, event.Label)

		if instance.parent.Get("CanVibrate").(bool) {
			go misc.VibrateAlert(instance.parent.Player, instance.ctx)
		}

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

	instance.parent.Timers["oled"].Restart()
	instance.parent.Timers["keypad"].Restart()
}

func (instance *GenericAlert) Pause() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Power alert pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *GenericAlert) Stop() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Power alert stop timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}
