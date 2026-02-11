package keypad

import (
	"context"
	"fmt"
	"log"
	"time"
	"timers"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
)

type KeypadEvent struct {
	State    bool
	Key      rune
	Duration float64
}

type PinIn struct {
	Label string
	gpio.PinIn
}

type PinOut struct {
	Label string
	gpio.PinOut
}

var KeyMap = map[[2]int]rune{
	{0, 1}: 'C',
	{0, 2}: '1',
	{0, 3}: '2',
	{0, 4}: '3',
	{1, 1}: 'S',
	{1, 2}: '4',
	{1, 3}: '5',
	{1, 4}: '6',
	{2, 0}: 'D',
	{2, 2}: '7',
	{2, 3}: '8',
	{2, 4}: '9',
	{3, 1}: 'U',
	{3, 2}: '*',
	{3, 3}: '0',
	{3, 4}: '#',
}

func debounceRead(ctx context.Context, pin *PinIn, state gpio.Level, duration time.Duration) bool {
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

func stop(colPins [5]*PinOut) {
	for _, colPin := range colPins {
		colPin.Out(gpio.Low)
	}
}

func Run(ctx context.Context) <-chan *KeypadEvent {
	eventsChan := make(chan *KeypadEvent, 10)

	// Must be first
	if _, err := host.Init(); err != nil {
		panic(err)
	}

	// Setup GPIOs AFTER host.Init()
	rowPins := [4]*PinIn{
		{"GPIO7", gpioreg.ByName("GPIO7")},
		{"GPIO8", gpioreg.ByName("GPIO8")},
		// {"GPIO25", gpioreg.ByName("GPIO25")},
		{"GPIP12", gpioreg.ByName("GPIO12")},
		{"GPIO24", gpioreg.ByName("GPIO24")},
	}
	colPins := [5]*PinOut{
		{"GPIO9", gpioreg.ByName("GPIO9")},
		{"GPIO10", gpioreg.ByName("GPIO10")},
		{"GPIO22", gpioreg.ByName("GPIO22")},
		{"GPIO27", gpioreg.ByName("GPIO27")},
		{"GPIO17", gpioreg.ByName("GPIO17")},
	}

	// Check for nil pins
	for i, pin := range rowPins {
		if pin == nil {
			panic("⚠️ Row pin " + fmt.Sprint(i) + " not found!")
		}
		if err := pin.In(gpio.PullDown, gpio.NoEdge); err != nil {
			panic("⚠️ Failed to init row " + fmt.Sprint(i) + ": " + err.Error())
		}
	}

	for i, pin := range colPins {
		if pin == nil {
			panic("⚠️ Col pin " + fmt.Sprint(i) + " not found!")
		}
		if err := pin.Out(gpio.Low); err != nil {
			panic("⚠️ Failed to init col " + fmt.Sprint(i) + ": " + err.Error())
		}
	}

	// Bind power button
	powerButton := &PinIn{"GPIO3", gpioreg.ByName("GPIO3")}
	if powerButton == nil {
		panic("⚠️ Failed to bind to GPIO3 (Power button)")
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
				stop(colPins)
				return
			default:

				// Scan power button
				if powerButton != nil {
					if debounceRead(ctx, powerButton, gpio.Low, 25*time.Millisecond) {
						eventsChan <- &KeypadEvent{
							State: true,
							Key:   'P',
						}
					release_power:
						for {
							select {
							case <-ctx.Done():
								stop(colPins)
								return
							default:
								if debounceRead(ctx, powerButton, gpio.High, 50*time.Millisecond) {
									eventsChan <- &KeypadEvent{
										State: false,
										Key:   'P',
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
					for colIdx, colPin := range colPins {
						colPin.Out(gpio.High)
						for rowIdx, rowPin := range rowPins {
							select {
							case <-ctx.Done():
								stop(colPins)
								return
							default:
								if rowPin.Read() == gpio.High {
									if debounceRead(ctx, rowPin, gpio.High, 25*time.Millisecond) {
										if faultyReads > 5 {
											log.Printf("⚠️ Too many faulty reads, suspected short on pins %s and %s (%d:%d)", rowPin.Label, colPin.Label, rowIdx, colIdx)
											colPin.Out(gpio.Low)
											time.Sleep(50 * time.Millisecond)
											continue
										}
										lastRow = rowPin
										lastCol = colPin
										lastRune = KeyMap[[2]int{rowIdx, colIdx}]
										if lastRune == 0 {
											log.Printf("⚠️ Invalid keypress detected (faulty keypad? short on pins %s and %s (%d:%d)), attempting to recover...", rowPin.Label, colPin.Label, rowIdx, colIdx)
											colPin.Out(gpio.Low)
											faultyReads++
											time.Sleep(50 * time.Millisecond)
											continue press
										}
										if faultyReads > 0 {
											log.Printf("⚠️ Recovered from keypad fault")
											faultyReads = 0
										}
										log.Printf("⌨️ Keypress detected on pins %s %s (%d:%d - %c)", rowPin.Label, colPin.Label, rowIdx, colIdx, lastRune)
										eventsChan <- &KeypadEvent{
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
						colPin.Out(gpio.Low)
					}
				}

				// Handle keypad release
				if lastRow != nil && lastCol != nil {
					var duration time.Duration
					lastCol.Out(gpio.High)
				release:
					for {
						select {
						case <-ctx.Done():
							stop(colPins)
							return
						default:
							if debounceRead(ctx, lastRow, gpio.Low, 50*time.Millisecond) {
								duration = time.Since(start)
								break release
							}
							timers.SleepWithContext(time.Millisecond, ctx)
						}
					}
					lastCol.Out(gpio.Low)
					lastRow = nil
					lastCol = nil
					eventsChan <- &KeypadEvent{
						State:    false,
						Key:      lastRune,
						Duration: duration.Seconds(),
					}
				}
				timers.SleepWithContext(time.Millisecond, ctx)
			}
		}
	}()

	return eventsChan
}
