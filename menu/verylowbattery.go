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

type VeryLowBatteryAlert struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
}

func (m *Menu) NewVeryLowBatteryAlert() *VeryLowBatteryAlert {
	return &VeryLowBatteryAlert{
		parent: m,
	}
}

func (instance *VeryLowBatteryAlert) render() {
	instance.parent.RenderAlert("very_low_battery", []string{"Battery", "almost empty!", "Please", "recharge the", "phone soon."})
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

func (instance *VeryLowBatteryAlert) Run() {
	if !instance.configured {
		panic("Attempted to call (*VeryLowBatteryAlert).Run() before (*VeryLowBatteryAlert).Configure()!")
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
			misc.PlayLowBattery(instance.parent.Player, instance.ctx)
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

func (instance *VeryLowBatteryAlert) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Very low battery alert pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *VeryLowBatteryAlert) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Very low battery alert stop timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}
