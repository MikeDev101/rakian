package menu

import (
	"context"
	"misc"
	"sync"
	"time"
)

type BatteryChargingAlert struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
	stackIndex int
}

func (m *Menu) NewBatteryChargingAlert() MenuInstance {
	return &BatteryChargingAlert{
		parent: m,
	}
}

func (instance *BatteryChargingAlert) Label() string {
	return "Battery Charging Alert"
}

func (instance *BatteryChargingAlert) render() {
	m := instance.parent
	display := m.Display
	defer display.Unlock()
	display.Lock()
	m.RenderAnimatedAlert("charging", instance.ctx, []string{"Battery", "charging"})
}

func (instance *BatteryChargingAlert) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *BatteryChargingAlert) ConfigureWithArgs(args ...any) {
	// Unused
	instance.Configure()
}

func (instance *BatteryChargingAlert) Pause() {
	Pause(instance)
}

func (instance *BatteryChargingAlert) Stop() {
	Stop(instance)
}

func (instance *BatteryChargingAlert) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *BatteryChargingAlert) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *BatteryChargingAlert) Cleanup() {}

func (instance *BatteryChargingAlert) Save() {}

func (instance *BatteryChargingAlert) Load() {}

func (instance *BatteryChargingAlert) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *BatteryChargingAlert) GetStackIndex() int {
	return instance.stackIndex
}

func (instance *BatteryChargingAlert) Run() {
	if !instance.configured {
		panic("Attempted to call (*BatteryChargingAlert).Run() before (*BatteryChargingAlert).Configure()!")
	}

	if instance.parent.Get("CanRing").(bool) || instance.parent.Get("BeepOnly").(bool) {
		instance.wg.Add(1)
		go func() {
			defer instance.wg.Done()
			misc.PlayBeep(instance.parent.Player, instance.ctx)
			// instance.parent.PlayAlert()
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
