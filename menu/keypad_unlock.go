package menu

import (
	"context"
	"sync"
	"time"
)

type KeypadUnlockMenu struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
	stackIndex int
}

func (m *Menu) NewKeypadUnlockMenu() MenuInstance {
	return &KeypadUnlockMenu{
		parent: m,
	}
}

func (instance *KeypadUnlockMenu) Label() string {
	return "Keypad Unlock Menu"
}

func (instance *KeypadUnlockMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *KeypadUnlockMenu) ConfigureWithArgs(args ...any) {
	// Unused
	instance.Configure()
}

func (instance *KeypadUnlockMenu) Pause() {
	Pause(instance)
}

func (instance *KeypadUnlockMenu) Stop() {
	Stop(instance)
}

func (instance *KeypadUnlockMenu) Cancel() {
	if instance.cancelFn != nil {
		instance.cancelFn()
	}
}

func (instance *KeypadUnlockMenu) WaitGroup() *sync.WaitGroup {
	return &instance.wg
}

func (instance *KeypadUnlockMenu) Cleanup() {}

func (instance *KeypadUnlockMenu) Save() {}

func (instance *KeypadUnlockMenu) Load() {}

func (instance *KeypadUnlockMenu) SetStackIndex(i int) {
	instance.stackIndex = i
}

func (instance *KeypadUnlockMenu) GetStackIndex() int {
	return instance.stackIndex
}

func (instance *KeypadUnlockMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*KeypadUnlockMenu).Run() before (*KeypadUnlockMenu).Configure()!")
	}

	m := instance.parent
	display := m.Display

	defer display.Unlock()
	display.Lock()

	// Create animation context and play the unlock guide animation
	animCtx, animCancel := context.WithCancel(instance.ctx)
	go m.RenderAnimatedAlert("keypad_unlock_info", animCtx, []string{"Now press", "the * key"})

	instance.wg.Add(1)
	go func() {
		defer animCancel()
		defer instance.wg.Done()
		for {
			select {
			case <-instance.ctx.Done():
				return

			case <-time.After(5 * time.Second):
				go instance.parent.Pop()
				return

			case evt := <-instance.parent.KeypadEvents:

				if evt.State {
					instance.parent.Timers["keypad"].Reset()
					instance.parent.Timers["screensaver"].Reset()
					instance.parent.Display.On()
					instance.parent.Keypad.KeyLightsOn()

					if evt.Key == '*' {
						animCancel()

						// Play locked animation
						animCtx, animCancel = context.WithCancel(instance.ctx)
						go instance.parent.RenderAnimatedAlert("keypad_unlocked", instance.ctx, []string{"Keypad", "unlocked"})

						go instance.parent.PlayAccepted()

						time.Sleep(time.Second * 2)
						instance.parent.Set("KeypadLocked", false)
						go instance.parent.Pop()
						return
					}
				}
			}
		}
	}()
}
