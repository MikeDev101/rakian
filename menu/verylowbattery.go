package menu

import (
	"context"
	"sync"
	"time"

	"misc"

	"github.com/google/uuid"
)

type VeryLowBatteryAlert struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
	id         uuid.UUID
	stackIndex int
}

func (m *Menu) NewVeryLowBatteryAlert() MenuInstance {
	return &VeryLowBatteryAlert{
		parent: m,
		id:     uuid.New(),
	}
}

func (instance *VeryLowBatteryAlert) Label() string {
	return "Very Low Battery Alert"
}

func (instance *VeryLowBatteryAlert) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *VeryLowBatteryAlert) ConfigureWithArgs(args ...any) {
	// Unused
	instance.Configure()
}

func (instance *VeryLowBatteryAlert) Pause() {
	Pause(instance)
}

func (instance *VeryLowBatteryAlert) Stop() {
	Stop(instance)
}

func (instance *VeryLowBatteryAlert) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *VeryLowBatteryAlert) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *VeryLowBatteryAlert) Cleanup() {}

func (instance *VeryLowBatteryAlert) Save() {}

func (instance *VeryLowBatteryAlert) Load() {}

func (instance *VeryLowBatteryAlert) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *VeryLowBatteryAlert) GetStackIndex() int {
	return instance.stackIndex
}

func (instance *VeryLowBatteryAlert) Run() {
	if !instance.configured {
		panic("Attempted to call (*VeryLowBatteryAlert).Run() before (*VeryLowBatteryAlert).Configure()!")
	}

	if instance.parent.Get("CanRing").(bool) || instance.parent.Get("BeepOnly").(bool) {
		instance.wg.Add(1)
		go func() {
			defer instance.wg.Done()
			misc.PlayLowBattery(instance.parent.Player, instance.ctx)
		}()
	}

	instance.wg.Go(func() {
		instance.render()

		select {
		case <-instance.ctx.Done():
			return

		case <-time.After(3 * time.Second):
			instance.parent.Timers["screensaver"].Restart()
			instance.parent.Timers["keypad"].Restart()
			go instance.parent.Pop()
			return
		}
	})

	instance.wg.Go(func() {
		for {
			select {
			case <-instance.ctx.Done():
				return
			case evt, ok := <-instance.parent.KeypadEvents:
				if !ok {
					return
				}

				keypad_locked := instance.parent.Get("KeypadLocked").(bool)
				if evt.State && !keypad_locked {
					instance.parent.Timers["screensaver"].Restart()
					instance.parent.Timers["keypad"].Restart()
					go instance.parent.Pop()
					return
				}
			}
		}
	})
}

func (instance *VeryLowBatteryAlert) render() {
	m := instance.parent
	display := m.Display
	defer display.Unlock()
	display.Lock()
	m.RenderAnimatedAlert("very_low_battery", instance.ctx, []string{"Please", "recharge", "soon"})
}
