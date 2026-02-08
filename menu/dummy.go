package menu

import (
	"context"
	"log"
	"sync"
	"time"

	"misc"
	"sh1107"
	"timers"
)

type DummyMenu struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
}

func (m *Menu) NewDummyMenu() *DummyMenu {
	return &DummyMenu{
		parent: m,
	}
}

func (instance *DummyMenu) render() {
	instance.parent.Display.Clear(sh1107.Black)
	font := instance.parent.Display.Use_Font8_Bold()
	instance.parent.Display.DrawText(50, 50, font, "DUMMY", false)
	instance.parent.Display.Render()
}

func (instance *DummyMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *DummyMenu) ConfigureWithArgs(args ...any) {
	// Unused
	instance.Configure()
}

func (instance *DummyMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*DummyMenu).Run() before (*DummyMenu).Configure()!")
	}

	instance.render()

	instance.wg.Add(1)
	go func() {
		defer instance.wg.Done()
		for {
			timers.SleepWithContext(250*time.Millisecond, instance.ctx)
			select {
			case <-instance.ctx.Done():
				return
			default:
				instance.render()
			}
		}
	}()

	instance.wg.Add(1)
	go func() {
		defer instance.wg.Done()
		for {
			select {
			case <-instance.ctx.Done():
				return
			case evt := <-instance.parent.KeypadEvents:

				if evt.State {

					instance.parent.Timers["keypad"].Reset()
					instance.parent.Timers["oled"].Reset()
					instance.parent.Display.On()
					misc.KeyLightsOn()
					go instance.parent.PlayKey()

					if evt.Key == 'P' {
						go instance.parent.Pop()
						return
					}
				}
			}
		}
	}()
}

func (instance *DummyMenu) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Dummy menu pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *DummyMenu) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Dummy menu stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		go instance.cleanup()
	}
}

func (instance *DummyMenu) cleanup() {

}
