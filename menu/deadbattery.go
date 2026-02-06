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

type DeadBatteryAlert struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
}

func (m *Menu) NewDeadBatteryAlert() *DeadBatteryAlert {
	return &DeadBatteryAlert{
		parent: m,
	}
}

func (instance *DeadBatteryAlert) render() {
	instance.parent.RenderAlert("dead_battery", []string{"Battery", "empty"})
}

func (instance *DeadBatteryAlert) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
	log.Println("▶️ Dead battery alert has been configured")
}

func (instance *DeadBatteryAlert) Run() {
	if !instance.configured {
		panic("Attempted to call (*DeadBatteryAlert).Run() before (*DeadBatteryAlert).Configure()!")
	}
	
	log.Printf("▶️ Dead battery alert started")
	
	// Mask all further menus
	instance.parent.Mask()
	
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
			misc.PlayDeadBattery(instance.parent.Player, instance.ctx)
		}()
	}
	
	instance.wg.Add(1)
	go func() {
		defer instance.wg.Done()
		instance.render()
		
		select {
		case <-instance.ctx.Done():
			log.Println("🛑 Dead battery alert canceled")
			return
		
		case <-time.After(3 * time.Second):
			log.Println("↩️ Dead battery alert calling global quit")
			go instance.parent.GlobalQuit(1)
			return
		}
	}()
}

func (instance *DeadBatteryAlert) Pause() {
	log.Println("⌛ Dead battery alert pausing...")
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Dead battery alert pause timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		log.Println("⏸️ Dead battery alert paused")
	}
}

func (instance *DeadBatteryAlert) Stop() {
	log.Println("⌛ Dead battery alert stopping...")
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Dead battery alert stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		log.Println("❌ Dead battery alert stopped")
		go instance.cleanup()
	}
}

func (instance *DeadBatteryAlert) cleanup() {
	
}