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

func (instance *BatteryChargingAlert) render() {
	instance.parent.RenderAlert("battery_charging", []string{"Battery", "charging"})
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

	if instance.parent.Get("CanVibrate").(bool) {
		instance.wg.Add(1)
		go func() {
			defer instance.wg.Done()
			misc.VibrateAlert(instance.parent.Player, instance.ctx)
		}()
	}

	if instance.parent.Get("CanRing").(bool) || instance.parent.Get("BeepOnly").(bool) {
		instance.wg.Add(1)
		go func() {
			defer instance.wg.Done()
			misc.PlayBeep(instance.parent.Player, instance.ctx)
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
			instance.parent.Timers["oled"].Restart()
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
