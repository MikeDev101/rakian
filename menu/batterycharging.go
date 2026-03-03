package menu

import (
	"context"
	"log"
	"misc"
	"sync"
	"time"
	// "lcd"
	// "timers"
)

type BatteryChargingAlert struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
}

func (m *Menu) NewBatteryChargingAlert() *BatteryChargingAlert {
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

	instance.wg.Add(1)
	go func() {
		defer instance.wg.Done()
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
	}()
}

func (instance *BatteryChargingAlert) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Battery charging alert pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *BatteryChargingAlert) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Battery charging alert stop timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}
