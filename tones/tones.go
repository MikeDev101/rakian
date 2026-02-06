package tones

import (
	"context"
	"math"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/host/v3"
)

type Tones struct {
	pout     gpio.PinOut
	vibrator gpio.PinOut
}

type Note struct {
	Key      int
	Duration time.Duration
	Divider  uint8
}

type Vibrate struct {
	State    bool
	Duration time.Duration
}

func New() *Tones {
	if _, err := host.Init(); err != nil {
		panic(err)
	}

	p := gpioreg.ByName("GPIO13")
	if p == nil {
		panic(" Failed to find tone pin!")
	}

	pout, ok := p.(gpio.PinOut)
	if !ok {
		panic(" Tone pin does not support PWM!")
	}

	vibrator_pin := gpioreg.ByName("GPIO12")
	if vibrator_pin == nil {
		panic(" Failed to find vibrator pin!")
	}

	if err := vibrator_pin.Out(gpio.Low); err != nil {
		panic(err)
	}

	return &Tones{
		pout:     pout,
		vibrator: vibrator_pin,
	}
}

func (t *Tones) starttone(freq physic.Frequency, divider uint8) {
	if err := t.pout.PWM(gpio.DutyMax/gpio.Duty(divider), freq); err != nil {
		panic(err)
	}
}

func (t *Tones) stoptone() {
	t.pout.PWM(0, 0)
}

func (t *Tones) startvibrate() {
	if err := t.vibrator.Out(gpio.High); err != nil {
		panic(err)
	}
}

func (t *Tones) stopvibrate() {
	if err := t.vibrator.Out(gpio.Low); err != nil {
		panic(err)
	}
}

func (t *Tones) Stop() {
	t.stoptone()
}

func (t *Tones) Tone(note int, divider uint8) {
	t.starttone(note_to_freq(note), divider)
}

func (t *Tones) StartVibrate() {
	t.startvibrate()
}

func (t *Tones) StopVibrate() {
	t.stopvibrate()
}

func note_to_freq(Note int) physic.Frequency {
	// MIDI Note 69 = A4 = 440Hz
	return physic.Frequency(440*math.Pow(2, float64(Note-69)/12)) * physic.Hertz
}

func (t *Tones) Play(ctx context.Context, notes []Note) {
	for _, n := range notes {
		select {
		case <-ctx.Done():
			t.stoptone()
			return
		default:
			t.starttone(note_to_freq(n.Key), n.Divider)
			timer := time.NewTimer(n.Duration)
			select {
			case <-ctx.Done():
				timer.Stop()
				t.stoptone()
				return
			case <-timer.C:
			}
		}
	}
	t.stoptone()
}

func (t *Tones) Vibrate(ctx context.Context, states []Vibrate) {
	for _, n := range states {
		select {
		case <-ctx.Done():
			t.stopvibrate()
			return
		default:
			if n.State {
				t.startvibrate()
			} else {
				t.stopvibrate()
			}

			timer := time.NewTimer(n.Duration)
			select {
			case <-ctx.Done():
				timer.Stop()
				t.stopvibrate()
				return
			case <-timer.C:
			}
		}
	}
	t.stopvibrate()
}
