package tones

import (
	"context"
	"embed"
	"fmt"
	"io"
	"log"
	"math"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

//go:embed static/*
var static_fs embed.FS

type Tones struct {
	ctx         *oto.Context
	tonePlayer  *oto.Player
	dtmfPlayers map[rune]*oto.Player
	ringbackPlayer *oto.Player
	mu          sync.Mutex
}

type Note struct {
	Key      float64
	Duration time.Duration
	Divider  uint8
}

type Vibrate struct {
	State    bool
	Duration time.Duration
}

const (
	sampleRate   = 44100
	channelCount = 2
	format       = oto.FormatSignedInt16LE
)

func New() *Tones {
	op := &oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: channelCount,
		Format:       format,
	}
	ctx, ready, err := oto.NewContext(op)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize oto: %v", err))
	}
	<-ready

	return &Tones{
		ctx:         ctx,
		dtmfPlayers: make(map[rune]*oto.Player),
	}
}

// Close should be called when your app shuts down to unmap memory safely
func (t *Tones) Close() {
	t.Stop()
}

type DTMFWave struct {
	f1, f2 float64
	pos    int64
	volume float64
}

func (d *DTMFWave) Read(buf []byte) (int, error) {
	bytesPerSample := 4 // 16-bit stereo
	numSamples := len(buf) / bytesPerSample

	for i := 0; i < numSamples; i++ {
		tm := float64(d.pos) / float64(sampleRate)
		val := (0.5*math.Sin(2*math.Pi*d.f1*tm) + 0.5*math.Sin(2*math.Pi*d.f2*tm)) * d.volume
		if val > 1 {
			val = 1
		} else if val < -1 {
			val = -1
		}
		sample := int16(val * 32767)

		buf[i*4] = byte(sample)
		buf[i*4+1] = byte(sample >> 8)
		buf[i*4+2] = byte(sample)
		buf[i*4+3] = byte(sample >> 8)

		d.pos++
	}
	return numSamples * bytesPerSample, nil
}

type SineWave struct {
	freq   float64
	pos    int64
	volume float64
}

func (s *SineWave) Read(buf []byte) (int, error) {
	bytesPerSample := 4 // 16-bit stereo
	numSamples := len(buf) / bytesPerSample

	for i := 0; i < numSamples; i++ {
		val := math.Sin(2*math.Pi*s.freq*float64(s.pos)/float64(sampleRate)) * s.volume
		if val > 1 {
			val = 1
		} else if val < -1 {
			val = -1
		}
		sample := int16(val * 32767)

		buf[i*4] = byte(sample)
		buf[i*4+1] = byte(sample >> 8)
		buf[i*4+2] = byte(sample)
		buf[i*4+3] = byte(sample >> 8)

		s.pos++
	}
	return numSamples * bytesPerSample, nil
}

func (t *Tones) Tone(note float64, divider uint8) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.tonePlayer != nil {
		t.tonePlayer.SetVolume(0)
		t.tonePlayer = nil
	}

	if note <= 0 {
		return
	}

	freq := note_to_freq(note)
	vol := 2.0
	if divider > 2 {
		vol = 2.0 * (2.0 / float64(divider))
	}

	src := &SineWave{
		freq:   freq,
		pos:    0,
		volume: vol,
	}

	t.tonePlayer = t.ctx.NewPlayer(src)
	t.tonePlayer.SetVolume(1)
	t.tonePlayer.Play()
}

func (t *Tones) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.tonePlayer != nil {
		t.tonePlayer.SetVolume(0)
		t.tonePlayer = nil
	}
	// Also stop all DTMF tones
	for key, player := range t.dtmfPlayers {
		player.SetVolume(0)
		delete(t.dtmfPlayers, key)
	}
	if t.ringbackPlayer != nil {
		t.ringbackPlayer.SetVolume(0)
		t.ringbackPlayer = nil
	}
}

func note_to_freq(Note float64) float64 {
	// MIDI Note 69 = A4 = 440Hz
	return 440.0 * math.Pow(2, float64(Note-69)/12.0)
}

func (t *Tones) Play(ctx context.Context, notes []Note) {
	for _, n := range notes {
		select {
		case <-ctx.Done():
			t.Stop()
			return
		default:
			// If Key is greater than 0, play the tone. Otherwise, treat it as a Rest.
			if n.Key > 0 {
				t.Tone(n.Key, n.Divider)
			} else {
				t.Stop()
			}

			timer := time.NewTimer(n.Duration)
			select {
			case <-ctx.Done():
				timer.Stop()
				t.Stop()
				return
			case <-timer.C:
			}
		}
	}
	t.Stop()
}

func (t *Tones) PlayFile(path string) {

	// Load the static file
	path = "static/" + path + ".mp3"
	f, err := static_fs.Open(path)
	if err != nil {
		log.Printf("⚠️ Failed to open static file: %v", err)
		return
	}

	// Prepare the MP3 reader
	var stream io.Reader
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		f.Close()
		log.Printf("⚠️ Failed to seek static file: %v", err)
		return
	}

	// Prepare the MP3 decoder
	d, err := mp3.NewDecoder(rs)
	if err != nil {
		f.Close()
		log.Printf("⚠️ Failed to decode static file: %v", err)
		return
	}
	stream = d

	// Play the MP3
	wrappedStream := &fileStreamWrapper{Reader: stream, Closer: f}
	player := t.ctx.NewPlayer(wrappedStream)
	player.SetVolume(1)
	player.Play()
}

type fileStreamWrapper struct {
	io.Reader
	io.Closer
}

// Read reads from the underlying io.Reader and rescales the samples to 16-bit signed little-endian.
// It also closes the underlying io.Closer when the end of the stream is reached.
// The rescaling is done as follows: val = sample * 3. If val > 32767, it is set to 32767.
// If val < -32768, it is set to -32768. The resulting sample is then written back to p.
// The returned error is the same as the one returned by the underlying io.Reader.Read method.
// If the underlying io.Reader.Read method returns io.EOF, the underlying io.Closer is closed.
func (f *fileStreamWrapper) Read(p []byte) (n int, err error) {
	n, err = f.Reader.Read(p)
	if n > 0 {
		for i := 0; i < n-1; i += 2 {
			sample := int16(uint16(p[i]) | uint16(p[i+1])<<8)
			val := int32(sample) * 3
			if val > 32767 {
				val = 32767
			} else if val < -32768 {
				val = -32768
			}
			sample = int16(val)
			p[i] = byte(sample)
			p[i+1] = byte(sample >> 8)
		}
	}
	if err == io.EOF {
		f.Closer.Close()
	}
	return
}

func (t *Tones) PlayDTMF(key rune) {
	var f1, f2 float64
	switch key {
	case '1':
		f1, f2 = 697, 1209
	case '2':
		f1, f2 = 697, 1336
	case '3':
		f1, f2 = 697, 1477
	case '4':
		f1, f2 = 770, 1209
	case '5':
		f1, f2 = 770, 1336
	case '6':
		f1, f2 = 770, 1477
	case '7':
		f1, f2 = 852, 1209
	case '8':
		f1, f2 = 852, 1336
	case '9':
		f1, f2 = 852, 1477
	case '*':
		f1, f2 = 941, 1209
	case '0':
		f1, f2 = 941, 1336
	case '#':
		f1, f2 = 941, 1477
	default:
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.dtmfPlayers[key]; ok {
		return // Already playing
	}

	src := &DTMFWave{
		f1:     f1,
		f2:     f2,
		pos:    0,
		volume: 0.5, // 50% volume
	}

	player := t.ctx.NewPlayer(src)
	player.SetVolume(1)
	player.Play()

	t.dtmfPlayers[key] = player
}

func (t *Tones) StopDTMF(key rune) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if player, ok := t.dtmfPlayers[key]; ok {
		player.SetVolume(0)
		delete(t.dtmfPlayers, key)
	}
}

func (t *Tones) PlayRingback() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.ringbackPlayer != nil {
		return // Already playing
	}

	src := &DTMFWave{ // Reusing DTMFWave as it generates two sine waves
		f1:     440, // US Ringback tone freq 1
		f2:     480, // US Ringback tone freq 2
		pos:    0,
		volume: 0.5, // 50% volume to avoid clipping
	}

	player := t.ctx.NewPlayer(src)
	player.SetVolume(1)
	player.Play()

	t.ringbackPlayer = player
}

func (t *Tones) StopRingback() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.ringbackPlayer != nil {
		t.ringbackPlayer.SetVolume(0)
		t.ringbackPlayer = nil
	}
}

func (t *Tones) StopAllDTMF() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for key, player := range t.dtmfPlayers {
		player.SetVolume(0)
		delete(t.dtmfPlayers, key)
	}
}
