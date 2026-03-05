package serial_audio

import (
	"context"
	"log"
	"sync"
	"time"

	"modem"

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

func (sa *SerialAudio) Run(ctx context.Context, cancel context.CancelFunc, current_call *modem.Call) {
	log.Println("ℹ️  Starting Serial Audio...")
	var wg sync.WaitGroup

	// 1. Open Serial Port
	mode := &serial.Mode{BaudRate: baudRate}
	serialPort, err := serial.Open(serialDevice, mode)
	if err != nil {
		log.Printf("⚠️ Serial Audio port error: %v", err)
		cancel()
		return
	}
	serialPort.ResetInputBuffer()
	serialPort.ResetOutputBuffer()
	serialPort.SetReadTimeout(250 * time.Millisecond)
	log.Println("✅ Serial Audio port opened")

	// 2. PulseAudio Setup
	ss := pulse.SampleSpec{
		Format:   pulse.SAMPLE_S16LE,
		Rate:     sampleRate,
		Channels: channels}
	capture, err := pulse.Capture("SIMCom", "Mic", &ss)
	if err != nil {
		log.Printf("⚠️ Acquire capture device error: %v", err)
		serialPort.Close()
		cancel()
		return
	}
	playback, err := pulse.Playback("SIMCom", "Speaker", &ss)
	if err != nil {
		log.Printf("⚠️ Acquire playback device error: %v", err)
		serialPort.Close()
		capture.Free()
		cancel()
		return
	}

	// Mic -> Serial
	wg.Go(func() {
		buf := make([]byte, frameSize)

		defer func() {
			log.Println("ℹ️  Mic -> Serial loop stopped...")
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				var n int
				var err error
				var captured_n chan int
				var captured_err chan error

				go func() {
					n, err := capture.Read(buf)
					captured_n <- n
					captured_err <- err
				}()

				select {
				case n = <-captured_n:
				case err = <-captured_err:
				case <-ctx.Done():
					return
				}

				if err != nil {
					if ctx.Err() == nil {
						log.Printf("⚠️ Mic Read Error: %v", err)
					}
					return
				}

				if current_call != nil && current_call.Mute {
					// Overwrite the buffer with zeros
					for i := range n {
						buf[i] = 0
					}
				}
				if n > 0 {
					_, err := serialPort.Write(buf[:n])
					if err != nil {
						if ctx.Err() == nil {
							log.Printf("⚠️ Serial Write Error: %v", err)
						}
						return
					}
				}
			}
		}
	})

	// Serial -> Speaker
	wg.Go(func() {
		tempBuf := make([]byte, frameSize)
		var remainder []byte

		defer func() {
			log.Println("ℹ️  Serial -> Speaker loop stopped...")
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				type readRes struct {
					n   int
					err error
				}
				resCh := make(chan readRes, 1)

				go func() {
					n, err := serialPort.Read(tempBuf)
					resCh <- readRes{n, err}
				}()

				var read_n int
				var read_err error
				select {
				case res := <-resCh:
					read_n = res.n
					read_err = res.err
				case <-ctx.Done():
					return
				}

				if read_err != nil {
					if ctx.Err() == nil {
						log.Printf("⚠️ Serial Read Error: %v", read_err)
					}
					return
				}

				if read_n > 0 {
					data := append(remainder, tempBuf[:read_n]...)
					if len(data)%2 != 0 {
						remainder = []byte{data[len(data)-1]}
						data = data[:len(data)-1]
					} else {
						remainder = nil
					}
					if len(data) > 0 {

						// Calculate gain based on volume
						var gain float64 = 3.0
						if current_call != nil {
							vol := current_call.Volume
							if vol <= 0.5 {
								gain = 2.0 * vol
							} else {
								gain = 4.0*vol - 1.0
							}
						}

						for i := 0; i < len(data); i += 2 {
							sample := int16(uint16(data[i]) | uint16(data[i+1])<<8)
							val := int32(float64(sample) * gain)
							if val > 32767 {
								val = 32767
							} else if val < -32768 {
								val = -32768
							}
							sample = int16(val)
							data[i] = byte(sample)
							data[i+1] = byte(sample >> 8)
						}

						type writeRes struct {
							n   int
							err error
						}
						writeCh := make(chan writeRes, 1)

						go func(d []byte) {
							n, err := playback.Write(d)
							writeCh <- writeRes{n, err}
						}(data)

						var write_err error
						select {
						case res := <-writeCh:
							write_err = res.err
						case <-ctx.Done():
							return
						}

						if write_err != nil {
							if ctx.Err() == nil {
								log.Printf("⚠️ Speaker Write Error: %v", write_err)
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
		log.Println("ℹ️  Context cancelled, closing audio resources...")
		serialPort.Close()

		// Wait for goroutines to finish with a timeout
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			log.Println("ℹ️  Audio resources closed successfully.")
		case <-time.After(100 * time.Millisecond):
			log.Println("ℹ️  Wait timed out, forcing free of audio resources...")
		}

		// Free audio resources in a separate goroutine to prevent blocking
		go func() {
			capture.Free()
			playback.Free()
			log.Println("✅ Audio resources freed.")
		}()
	}()
}
