package menu

import (
	"time"
	"context"
	"sync"
	"log"
	
	// "sh1107"
	// "timers"
	"misc"
)

type LowBatteryAlert struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
}

func (m *Menu) NewLowBatteryAlert() *LowBatteryAlert {
	return &LowBatteryAlert{
		parent: m,
	}
}

func (instance *LowBatteryAlert) render() {
	instance.parent.RenderAlert("low_battery", []string{"Low", "battery!"})
}

func (instance *LowBatteryAlert) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *LowBatteryAlert) Run() {
	if !instance.configured {
		panic("Attempted to call (*LowBatteryAlert).Run() before (*LowBatteryAlert).Configure()!")
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

func (instance *LowBatteryAlert) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Low battery alert pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *LowBatteryAlert) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Low battery alert stop timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}