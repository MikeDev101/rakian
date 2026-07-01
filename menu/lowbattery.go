package menu

import (
	"context"
	"sync"
	"time"

	"misc"
)

type LowBatteryAlert struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
	stackIndex int
}

func (m *Menu) NewLowBatteryAlert() MenuInstance {
	return &LowBatteryAlert{
		parent: m,
	}
}

func (instance *LowBatteryAlert) Label() string {
	return "Low Battery Alert"
}

func (instance *LowBatteryAlert) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *LowBatteryAlert) ConfigureWithArgs(args ...any) {
	// Unused
	instance.Configure()
}

func (instance *LowBatteryAlert) Pause() {
	Pause(instance)
}

func (instance *LowBatteryAlert) Stop() {
	Stop(instance)
}

func (instance *LowBatteryAlert) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *LowBatteryAlert) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *LowBatteryAlert) Cleanup() {}

func (instance *LowBatteryAlert) Save() {}

func (instance *LowBatteryAlert) Load() {}

func (instance *LowBatteryAlert) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *LowBatteryAlert) GetStackIndex() int {
	return instance.stackIndex
}

func (instance *LowBatteryAlert) Run() {
	if !instance.configured {
		panic("Attempted to call (*LowBatteryAlert).Run() before (*LowBatteryAlert).Configure()!")
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

func (instance *LowBatteryAlert) render() {
	m := instance.parent
	display := m.Display
	defer display.Unlock()
	display.Lock()
	m.RenderAnimatedAlert("low_battery", instance.ctx, []string{"Low", "battery"})
}
