package menu

import (
	"context"
	"log"
	"sync"
	"time"

	"misc"
)

type DeadBatteryAlert struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
	stackIndex int
}

func (m *Menu) NewDeadBatteryAlert() MenuInstance {
	return &DeadBatteryAlert{
		parent: m,
	}
}

func (instance *DeadBatteryAlert) Label() string {
	return "Dead Battery Alert"
}

func (instance *DeadBatteryAlert) render() {
	m := instance.parent
	display := m.Display
	defer display.Unlock()
	display.Lock()
	m.RenderAlert("empty_battery", []string{"Battery", "empty"})
}

func (instance *DeadBatteryAlert) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
	log.Println("▶️ Dead battery alert has been configured")
}

func (instance *DeadBatteryAlert) ConfigureWithArgs(args ...any) {
	// Unused
	instance.Configure()
}

func (instance *DeadBatteryAlert) Pause() {
	Pause(instance)
}

func (instance *DeadBatteryAlert) Stop() {
	Stop(instance)
}

func (instance *DeadBatteryAlert) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *DeadBatteryAlert) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *DeadBatteryAlert) Cleanup() {}

func (instance *DeadBatteryAlert) Save() {}

func (instance *DeadBatteryAlert) Load() {}

func (instance *DeadBatteryAlert) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *DeadBatteryAlert) GetStackIndex() int {
	return instance.stackIndex
}

func (instance *DeadBatteryAlert) Run() {
	if !instance.configured {
		panic("Attempted to call (*DeadBatteryAlert).Run() before (*DeadBatteryAlert).Configure()!")
	}

	log.Printf("▶️ Dead battery alert started")

	// Mask all further menus
	instance.parent.Mask()

	if instance.parent.Get("CanRing").(bool) || instance.parent.Get("BeepOnly").(bool) {
		instance.wg.Add(1)
		go func() {
			defer instance.wg.Done()
			misc.PlayDeadBattery(instance.parent.Player, instance.ctx)
		}()
	}

	instance.wg.Go(func() {
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
	})
}
