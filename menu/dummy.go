package menu

import (
	"context"
	"sync"
	"time"

	"timers"
)

type DummyMenu struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
	stackIndex int
}

func (m *Menu) NewDummyMenu() MenuInstance {
	return &DummyMenu{
		parent: m,
	}
}

func (instance *DummyMenu) Label() string {
	return "Dummy Menu"
}

func (instance *DummyMenu) render() {
	// Do nothing since this is a template
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
					instance.parent.Timers["screensaver"].Reset()
					instance.parent.Display.On()
					instance.parent.Keypad.KeyLightsOn()
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
	Pause(instance)
}

func (instance *DummyMenu) Stop() {
	Stop(instance)
}

func (instance *DummyMenu) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *DummyMenu) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *DummyMenu) Cleanup() {}

func (instance *DummyMenu) Save() {}

func (instance *DummyMenu) Load() {}

func (instance *DummyMenu) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *DummyMenu) GetStackIndex() int {
	return instance.stackIndex
}
