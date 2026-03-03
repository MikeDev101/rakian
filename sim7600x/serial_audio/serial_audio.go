package serial_audio

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/mesilliac/pulse-simple"
	"go.bug.st/serial"
)

const (
	sampleRate   = 16000 // Requires AT+CPCMFRM=1
	channels     = 1
	serialDevice = "/dev/ttyUSB4"
	baudRate     = 921600 // Required to sustain 16kHz S16LE throughput
	frameSize    = 640    // 20ms @ 16kHz
)

type SerialAudio struct {
	capture_delay time.Duration
}

func Init() *SerialAudio {
	// Calculate the amount of time needed to synchronize the audio
	// 640 samples/frame @ 16KHz (+-5%) = ~16ms
	frameDuration := time.Duration(float64(frameSize) / (float64(sampleRate) * 2.5) * float64(time.Second))
	log.Printf("Capture delay is set to %v", frameDuration)

	return &SerialAudio{
		capture_delay: frameDuration,
	}
}

func (sa *SerialAudio) Run(ctx context.Context, cancel context.CancelFunc) {
	log.Println("Starting Serial Audio...")
	var wg sync.WaitGroup

	// 1. Open Serial Port
	mode := &serial.Mode{BaudRate: baudRate}
	serialPort, err := serial.Open(serialDevice, mode)
	if err != nil {
		log.Printf("Serial Error: %v", err)
		cancel()
		return
	}
	serialPort.ResetInputBuffer()
	serialPort.ResetOutputBuffer()
	serialPort.SetReadTimeout(250 * time.Millisecond)
	log.Println("Audio Serial Port Opened")

	// 2. PulseAudio Setup
	ss := pulse.SampleSpec{
		Format:   pulse.SAMPLE_S16LE,
		Rate:     sampleRate,
		Channels: channels}
	capture, err := pulse.Capture("SIMCom", "Mic", &ss)
	if err != nil {
		log.Printf("Capture Error: %v", err)
		serialPort.Close()
		cancel()
		return
	}
	playback, err := pulse.Playback("SIMCom", "Speaker", &ss)
	if err != nil {
		log.Printf("Playback Error: %v", err)
		serialPort.Close()
		capture.Free()
		cancel()
		return
	}

	// Mic -> Serial
	wg.Go(func() {
		defer cancel()
		buf := make([]byte, frameSize)
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(sa.capture_delay):
				n, err := capture.Read(buf)
				if err != nil {
					if ctx.Err() == nil {
						log.Printf("Mic Read Error: %v", err)
					}
					return
				}
				if n > 0 {
					_, err := serialPort.Write(buf[:n])
					if err != nil {
						if ctx.Err() == nil {
							log.Printf("Serial Write Error: %v", err)
						}
						return
					}
				}
			}
		}
	})

	// Serial -> Speaker
	wg.Go(func() {
		defer cancel()
		tempBuf := make([]byte, frameSize)
		var remainder []byte
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(sa.capture_delay):
				n, err := serialPort.Read(tempBuf)
				if err != nil {
					if ctx.Err() == nil {
						log.Printf("Serial Read Error: %v", err)
					}
					return
				}
				if n > 0 {
					data := append(remainder, tempBuf[:n]...)
					if len(data)%2 != 0 {
						remainder = []byte{data[len(data)-1]}
						data = data[:len(data)-1]
					} else {
						remainder = nil
					}
					if len(data) > 0 {

						// Software amplification (3x gain)
						for i := 0; i < len(data); i += 2 {
							sample := int16(uint16(data[i]) | uint16(data[i+1])<<8)
							val := int32(sample) * 3
							if val > 32767 {
								val = 32767
							} else if val < -32768 {
								val = -32768
							}
							sample = int16(val)
							data[i] = byte(sample)
							data[i+1] = byte(sample >> 8)
						}

						_, err := playback.Write(data)
						if err != nil {
							if ctx.Err() == nil {
								log.Printf("Speaker Write Error: %v", err)
							}
							return
						}
					}
				}
			}
		}
	})

	// Monitor context to close resources
	go func() {
		<-ctx.Done()
		log.Println("Context cancelled, closing audio resources...")
		serialPort.Close()
		wg.Wait()
		log.Println("Wait complete, freeing audio resources...")
		capture.Free()
		playback.Free()
		log.Println("Audio resources freed.")
	}()
}
