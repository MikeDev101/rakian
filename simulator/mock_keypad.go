package simulator

import (
	"keypad"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

func (v *Simulator) Keypad() keypad.Keypad {
	var kp keypad.Keypad = v
	return kp
}

func (v *Simulator) KeyLightsOff() {}
func (v *Simulator) KeyLightsOn()  {}

func mapKey(name fyne.KeyName) rune {
	switch name {
	case fyne.Key1:
		return '1'
	case fyne.Key2:
		return '2'
	case fyne.Key3:
		return '3'
	case fyne.Key4:
		return '4'
	case fyne.Key5:
		return '5'
	case fyne.Key6:
		return '6'
	case fyne.Key7:
		return '7'
	case fyne.Key8:
		return '8'
	case fyne.Key9:
		return '9'
	case fyne.Key0:
		return '0'
	case fyne.KeyAsterisk:
		return '*'
	case fyne.KeyReturn, fyne.KeyEnter:
		return 'S'
	case fyne.KeyBackspace, fyne.KeyDelete, fyne.KeyC:
		return 'C'
	case fyne.KeyUp:
		return 'U'
	case fyne.KeyDown:
		return 'D'
	case fyne.KeyS:
		return 'S'
	case fyne.KeyD:
		return 'D'
	case fyne.KeyU:
		return 'U'
	case fyne.KeyP:
		return 'P'
	}
	return 0
}

func (v *Simulator) setupInput() {
	if desk, ok := v.window.Canvas().(desktop.Canvas); ok {
		desk.SetOnKeyDown(func(ev *fyne.KeyEvent) {
			k := mapKey(ev.Name)
			if k != 0 {
				v.keyEvents <- &keypad.KeypadEvent{
					State: true,
					Key:   k,
				}
			}
		})
		desk.SetOnKeyUp(func(ev *fyne.KeyEvent) {
			k := mapKey(ev.Name)
			if k != 0 {
				v.keyEvents <- &keypad.KeypadEvent{
					State: false,
					Key:   k,
				}
			}
		})
	}
}
