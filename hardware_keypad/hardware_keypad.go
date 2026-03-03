package hardware_keypad

import (
	"context"
	"keypad"
	"log"
	"time"
	"timers"

	"github.com/stianeikeland/go-rpio/v4"
)

type PinIn struct {
	Label string
	rpio.Pin
}

type PinOut struct {
	Label string
	rpio.Pin
}

type Hardware_Keypad struct {
	Rows   []*PinIn
	Cols   []*PinOut
	Power  *PinIn
	KeyMap map[[2]int]rune
}

func New() keypad.Keypad {

	// Must be first
	if err := rpio.Open(); err != nil {
		panic(err)
	}

	// Setup GPIOs
	rowPins := []*PinIn{
		{"GPIO24", rpio.Pin(24)},
		{"GPIO25", rpio.Pin(25)},
		{"GPIO5", rpio.Pin(5)},
		{"GPIO6", rpio.Pin(6)},
	}

	colPins := []*PinOut{
		{"GPIO4", rpio.Pin(4)},
		{"GPIO17", rpio.Pin(17)},
		{"GPIO27", rpio.Pin(27)},
		{"GPIO22", rpio.Pin(22)},
		{"GPIO23", rpio.Pin(23)},
	}

	kp := &Hardware_Keypad{
		Rows:   rowPins,
		Cols:   colPins,
		Power:  &PinIn{"GPIO3", rpio.Pin(3)},
		KeyMap: keypad.KeyMap,
	}

	return kp
}

func stop(colPins []*PinOut) {
	for _, colPin := range colPins {
		colPin.Low()
	}
}

func debounceRead(ctx context.Context, pin *PinIn, state rpio.State, duration time.Duration) bool {
	steps := 10
	interval := duration / time.Duration(steps)

	for range steps {
		if pin.Read() != state {
			return false
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(interval):
			// wait the interval before next check
		}
	}
	return true
}

func (k *Hardware_Keypad) Run(ctx context.Context, debug bool) <-chan *keypad.KeypadEvent {
	eventsChan := make(chan *keypad.KeypadEvent, 10)

	for _, pin := range k.Rows {
		pin.Input()
		pin.PullDown()
	}

	for _, pin := range k.Cols {
		pin.Output()
		pin.Low()
	}

	// Bind power button
	if k.Power != nil {
		k.Power.Input()
		// GPIO3 has fixed pull-up on RPi, so we don't need to set it explicitly
	}

	// Scanner loop
	go func() {
		var lastRune rune
		var faultyReads int = 0
		var lastRow *PinIn
		var lastCol *PinOut
		var start time.Time
		for {
			select {
			case <-ctx.Done():
				stop(k.Cols)
				return
			case <-time.After(20 * time.Millisecond):

				// Scan power button
				if k.Power != nil {
					if debounceRead(ctx, k.Power, rpio.Low, 25*time.Millisecond) {
						eventsChan <- &keypad.KeypadEvent{
							State:    true,
							Key:      'P',
							Duration: 0,
						}
						start = time.Now()
					release_power:
						for {
							select {
							case <-ctx.Done():
								stop(k.Cols)
								return
							default:
								if debounceRead(ctx, k.Power, rpio.High, 50*time.Millisecond) {
									eventsChan <- &keypad.KeypadEvent{
										State:    false,
										Key:      'P',
										Duration: time.Since(start).Seconds(),
									}
									break release_power
								}
								timers.SleepWithContext(time.Millisecond, ctx)
							}
						}
					}
				}

				// Scan keypad
				if lastRow == nil && lastCol == nil {
				press:
					for colIdx, colPin := range k.Cols {
						colPin.High()
						for rowIdx, rowPin := range k.Rows {
							select {
							case <-ctx.Done():
								stop(k.Cols)
								return
							default:
								if rowPin.Read() == rpio.High {
									if debounceRead(ctx, rowPin, rpio.High, 25*time.Millisecond) {
										if faultyReads > 5 {
											log.Printf("⚠️ Too many faulty reads, suspected short on pins %s and %s (%d:%d)", rowPin.Label, colPin.Label, rowIdx, colIdx)
											colPin.Low()
											time.Sleep(50 * time.Millisecond)
											continue
										}
										lastRow = rowPin
										lastCol = colPin
										lastRune = k.KeyMap[[2]int{rowIdx, colIdx}]
										if lastRune == 0 {
											log.Printf("⚠️ Invalid keypress detected (faulty keypad? short on pins %s and %s (%d:%d)), attempting to recover...", rowPin.Label, colPin.Label, rowIdx, colIdx)
											colPin.Low()
											faultyReads++
											time.Sleep(50 * time.Millisecond)
											continue press
										}
										if faultyReads > 0 {
											log.Printf("⚠️ Recovered from keypad fault")
											faultyReads = 0
										}

										if debug {
											log.Printf("⌨️  Keypress detected on pins %s %s (%d:%d - %c)", rowPin.Label, colPin.Label, rowIdx, colIdx, lastRune)
										}

										eventsChan <- &keypad.KeypadEvent{
											State:    true,
											Key:      lastRune,
											Duration: 0,
										}
										start = time.Now()
										break press
									}
								}
							}
						}
						colPin.Low()
					}
				}

				// Handle keypad release
				if lastRow != nil && lastCol != nil {
					var duration time.Duration
					lastCol.High()
				release:
					for {
						select {
						case <-ctx.Done():
							stop(k.Cols)
							return
						default:
							if debounceRead(ctx, lastRow, rpio.Low, 50*time.Millisecond) {
								duration = time.Since(start)
								break release
							}
							timers.SleepWithContext(time.Millisecond, ctx)
						}
					}
					lastCol.Low()
					if debug {
						log.Printf("⌨️  Keypress released on pins %s %s (%c)", lastRow.Label, lastCol.Label, lastRune)
					}
					lastRow = nil
					lastCol = nil
					eventsChan <- &keypad.KeypadEvent{
						State:    false,
						Key:      lastRune,
						Duration: duration.Seconds(),
					}
				}
			}
		}
	}()

	return eventsChan
}
