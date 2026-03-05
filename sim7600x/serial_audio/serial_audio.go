package serial_audio

import (
	"context"
	"log"
	"sync"
	"time"

	"modem"

	"github.com/maltegrosse/go-modemmanager"
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

	// Define heartbeat interval
	const heartbeat_delay = 5 * time.Second

	// Mic -> Serial Supervisor
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			sa.captureLoop(ctx, capture, serialPort, current_call)
			if ctx.Err() != nil {
				return
			}
			log.Println("⚠️ Capture loop exited prematurely. Restarting...")
			select {
			case <-ctx.Done():
				return
			case <-time.After(60 * time.Millisecond):
			}
		}
	}()

	// Serial -> Speaker Supervisor
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			sa.playbackLoop(ctx, playback, serialPort, current_call)
			if ctx.Err() != nil {
				return
			}
			log.Println("⚠️ Playback loop exited prematurely. Restarting...")
			select {
			case <-ctx.Done():
				return
			case <-time.After(60 * time.Millisecond):
			}
		}
	}()

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

func (sa *SerialAudio) captureLoop(ctx context.Context, capture *pulse.Stream, serialPort serial.Port, current_call *modem.Call) {
	const heartbeat_delay = 5 * time.Second
	ready := make(chan bool, 1)

	go func() {
		// Wait until current call is in an active state
		for {
			switch current_call.State {
			case modemmanager.MmCallStateActive:
				ready <- true
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Millisecond):
			}
		}
	}()

	// Wait for ready signal
	log.Println("ℹ️  Mic -> Serial loop waiting for call to start...")
	select {
	case <-ctx.Done():
		log.Println("ℹ️  Mic -> Serial loop aborted...")
		return
	case <-ready:
	}

	buf := make([]byte, frameSize)
	last_read_counter := 0
	last_read_tstamp := time.Now()
	last_write_counter := 0
	last_write_tstamp := time.Now()

	defer func() {
		log.Println("ℹ️  Mic -> Serial loop stopped...")
	}()
	log.Println("ℹ️  Mic -> Serial loop started")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			type readRes struct {
				n   int
				err error
			}
			var capture_n int
			var capture_err error
			captured_ch := make(chan readRes, 1)

			go func() {
				n, err := capture.Read(buf)
				captured_ch <- readRes{n, err}
			}()

			select {
			case res := <-captured_ch:
				capture_n = res.n
				capture_err = res.err
			case <-ctx.Done():
				return
			}

			if capture_err != nil {
				if ctx.Err() == nil {
					log.Printf("⚠️ Mic Read Error: %v", capture_err)
				}
				return
			}

			last_read_counter += capture_n
			if time.Since(last_read_tstamp) > heartbeat_delay {
				last_read_tstamp = time.Now()
				log.Printf("❤️  Mic -> Serial Heartbeat: %d bytes read from capture", last_read_counter)
				last_read_counter = 0
			}

			var write_n int
			var write_err error
			writer_ch := make(chan readRes, 1)

			go func() {
				n, err := serialPort.Write(buf[:capture_n])
				writer_ch <- readRes{n, err}
			}()

			select {
			case res := <-writer_ch:
				write_n = res.n
				write_err = res.err
			case <-ctx.Done():
				return
			}

			if write_err != nil {
				if ctx.Err() == nil {
					log.Printf("⚠️ Serial Write Error: %v", write_err)
				}
				return
			}

			last_write_counter += write_n
			if time.Since(last_write_tstamp) > heartbeat_delay {
				last_write_tstamp = time.Now()
				log.Printf("❤️  Mic -> Serial Heartbeat: %d bytes written to serial", last_write_counter)
				last_write_counter = 0
			}
		}
	}
}

func (sa *SerialAudio) playbackLoop(ctx context.Context, playback *pulse.Stream, serialPort serial.Port, current_call *modem.Call) {
	const heartbeat_delay = 5 * time.Second
	tempBuf := make([]byte, frameSize)
	var remainder []byte
	last_read_counter := 0
	last_read_tstamp := time.Now()
	last_write_counter := 0
	last_write_tstamp := time.Now()

	log.Println("ℹ️  Serial -> Speaker loop started")
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

			last_read_counter += read_n
			if time.Since(last_read_tstamp) > heartbeat_delay {
				last_read_tstamp = time.Now()
				log.Printf("❤️  Serial -> Speaker Heartbeat: %d bytes read for playback", last_read_counter)
				last_read_counter = 0
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

					var write_n int
					var write_err error
					select {
					case res := <-writeCh:
						write_n = res.n
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

					last_write_counter += write_n
					if time.Since(last_write_tstamp) > heartbeat_delay {
						last_write_tstamp = time.Now()
						log.Printf("❤️  Serial -> Speaker Heartbeat: %d bytes written to playback", last_write_counter)
						last_write_counter = 0
					}
				}
			}
		}
	}
}
