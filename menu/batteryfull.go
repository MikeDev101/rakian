package menu

import (
	"context"
	"log"
	"sync"
	"time"

	// "lcd"
	// "timers"
	"misc"
)

type BatteryChargedAlert struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
}

func (m *Menu) NewBatteryChargedAlert() *BatteryChargedAlert {
	return &BatteryChargedAlert{
		parent: m,
	}
}

func (instance *BatteryChargedAlert) render() {
	instance.parent.RenderAlert("battery_full", []string{"Battery", "charged"})
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

func (instance *BatteryChargedAlert) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Battery charged alert pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *BatteryChargedAlert) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Battery charged alert stop timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}
