package menu

import (
	"context"
	"sync"
	"time"

	"misc"
)

type BatteryChargedAlert struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
	stackIndex int
}

func (m *Menu) NewBatteryChargedAlert() MenuInstance {
	return &BatteryChargedAlert{
		parent: m,
	}
}

func (instance *BatteryChargedAlert) Label() string {
	return "Battery Fully Charged Alert"
}

func (instance *BatteryChargedAlert) render() {
	m := instance.parent
	display := m.Display
	defer display.Unlock()
	display.Lock()
	m.RenderAlert("battery_full", []string{"Battery", "charged"})
}

func (instance *BatteryChargedAlert) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *BatteryChargedAlert) ConfigureWithArgs(args ...any) {
	// Unused
	instance.Configure()
}

func (instance *BatteryChargedAlert) Pause() {
	Pause(instance)
}

func (instance *BatteryChargedAlert) Stop() {
	Stop(instance)
}

func (instance *BatteryChargedAlert) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *BatteryChargedAlert) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *BatteryChargedAlert) Cleanup() {}

func (instance *BatteryChargedAlert) Save() {}

func (instance *BatteryChargedAlert) Load() {}

func (instance *BatteryChargedAlert) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *BatteryChargedAlert) GetStackIndex() int {
	return instance.stackIndex
}

func (instance *BatteryChargedAlert) Run() {
	if !instance.configured {
		panic("Attempted to call (*BatteryChargedAlert).Run() before (*BatteryChargedAlert).Configure()!")
	}

	if instance.parent.Get("CanRing").(bool) || instance.parent.Get("BeepOnly").(bool) {
		instance.wg.Add(1)
		go func() {
			defer instance.wg.Done()
			misc.PlayBeep(instance.parent.Player, instance.ctx)
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
