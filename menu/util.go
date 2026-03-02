package menu

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"lcd"
	"log"
	"math"
	"os/exec"
	"time"
)

const (
	T9Lowercase = 0
	T9Uppercase = 1
	T9Numbers   = 2
)

type InputPromptArgs struct {
	Title           string
	CharacterLimit  int
	PhoneNumberOnly bool
}

func (m *Menu) PlayAlert() {
	if m.Get("BeepOnly").(bool) {
		m.Player.Stop()
		m.Player.Tone(83, 5) // B5 (Alert/Stop sound)
		time.Sleep(50 * time.Millisecond)
		m.Player.Stop()
	} else if m.Get("CanRing").(bool) {
		m.Player.Stop()
		m.Player.Tone(83, 5) // B5 (Alert/Stop sound)
		time.Sleep(time.Second)
		m.Player.Stop()
	}
}

func (m *Menu) PlayAccepted() {
	if m.Get("BeepOnly").(bool) {
		m.Player.Tone(89, 5)
		time.Sleep(50 * time.Millisecond)
		m.Player.Stop()
	} else if m.Get("CanRing").(bool) {
		m.Player.Tone(89, 5)
		time.Sleep(500 * time.Millisecond)
		m.Player.Stop()
	}
}

func (m *Menu) PlayKey() {
	if m.Get("BeepOnly").(bool) {
		m.Player.Tone(81.5, 1)
		time.Sleep(50 * time.Millisecond)
		m.Player.Stop()
	} else if m.Get("CanRing").(bool) {
		m.Player.Tone(81.5, 1)
		time.Sleep(150 * time.Millisecond)
		m.Player.Stop()
	}
}

func (m *Menu) RenderAlert(icon string, status []string) {
	font := m.Display.Use_Font_Large_Bold()
	m.Display.Clear(lcd.White)
	if icon != "" {
		if sprite, ok := m.Display.Use_Sprites()[icon]; ok {
			m.Display.DrawImageAligned(sprite, 84, 0, lcd.AlignLeft, lcd.AlignBelow)
		} else {
			log.Printf("⚠️ Sprite '%s' not found", icon)
		}
	}
	for i, str := range status {
		m.Display.DrawText(0, 0+(i*16), font, str, false)
	}
	m.Display.Render()
}

func (m *Menu) RenderAnimatedAlert(animation string, ctx context.Context, status []string) {
	font := m.Display.Use_Font_Large_Bold()
	m.Display.Clear(lcd.White)
	for i, str := range status {
		m.Display.DrawText(0, 0+(i*16), font, str, false)
	}
	m.Display.Render()
	go m.Display.PlayAnimation(ctx, animation, 84, 0, lcd.AlignLeft, lcd.AlignBelow)
}

func (m *Menu) RenderFooter(footer string, bold bool) {
	display := m.Display
	var font map[rune]image.Image
	if bold {
		font = display.Use_Font_Small_Bold()
	} else {
		font = display.Use_Font_Small_Plain()
	}
	display.DrawTextAligned(42, 48, font, footer, false, lcd.AlignCenter, lcd.AlignAbove)
}

func (m *Menu) RenderHeader(header string) {
	display := m.Display
	font := display.Use_Font_Small_Plain()
	width, _ := display.GetTextBounds(font, header)
	display.DrawTextAligned(42, 0, font, header, false, lcd.AlignCenter, lcd.AlignBelow)

	// If the header text is too long, limit the width to 84
	if width > 84 {
		width = 84
	}

	// Get the positions of the left and right lines
	left_start := 42 - (width / 2) - 2
	right_start := 42 + (width / 2) + 1

	// Draw lines around the header text
	display.SetColor(lcd.Black)
	display.DrawRectangle(0, 3, float64(left_start), 1)
	display.Fill()
	display.DrawRectangle(float64(right_start), 3, float64(84-right_start), 1)
	display.Fill()
}

func (instance *Menu) RenderStateCommon() {
	m := instance
	display := m.Display

	// Read clock
	now := time.Now().In(time.Local)
	hour := now.Hour() % 12
	if hour == 0 {
		hour = 12
	}
	clock_str := fmt.Sprintf("%2d:%02d", hour, now.Minute())

	// Draw clock
	font := display.Use_Font_Tiny()
	display.DrawTextAligned(77, 0, font, clock_str, false, lcd.AlignLeft, lcd.AlignNone)

	// Draw icons
	if cell_sprite, ok := display.Use_Sprites()["cell"]; ok {
		display.DrawImageAligned(cell_sprite, 0, 38, lcd.AlignRight, lcd.AlignAbove)
	}
	if battery_sprite, ok := display.Use_Sprites()["battery"]; ok {
		display.DrawImageAligned(battery_sprite, 84, 38, lcd.AlignLeft, lcd.AlignAbove)
	}

	battery_state := m.Get("BatteryScaledPercent").(int)
	if battery_state > 0 {
		display.SetColor(lcd.Black)
		display.SetLineWidth(1)
		if battery_state == 4 {
			display.DrawRectangle(79, 0, 5, 7)
			display.Fill()
		}
		if battery_state >= 3 {
			display.DrawRectangle(81, 8, 3, 7)
			display.Fill()
		}
		if battery_state >= 2 {
			display.DrawRectangle(82, 16, 2, 7)
			display.Fill()
		}
		if battery_state >= 1 {
			display.DrawRectangle(82, 24, 2, 7)
			display.Fill()
		}
	}

	if m.Phone.OK {
		display.SetColor(lcd.Black)
		display.SetLineWidth(1)
		if m.Phone.SignalStrength >= 4 {
			display.DrawRectangle(0, 0, 5, 7)
			display.Fill()
		}
		if m.Phone.SignalStrength >= 3 {
			display.DrawRectangle(0, 8, 3, 7)
			display.Fill()
		}
		if m.Phone.SignalStrength >= 2 {
			display.DrawRectangle(0, 16, 2, 7)
			display.Fill()
		}
		if m.Phone.SignalStrength >= 1 {
			display.DrawRectangle(0, 24, 2, 7)
			display.Fill()
		}
	}
}

func (instance *Menu) playDTMF(key rune) {

	// Don't generate tones if we're mute
	if !instance.Get("CanRing").(bool) && !instance.Get("BeepOnly").(bool) {
		return
	}

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

	const sampleRate = 44100
	var duration = 200 * time.Millisecond

	// Use shorter tones if we're in discreet mode
	if instance.Get("BeepOnly").(bool) {
		duration = 50 * time.Millisecond
	}

	numSamples := int(sampleRate * duration.Seconds())

	buf := new(bytes.Buffer)
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		val := 0.5*math.Sin(2*math.Pi*f1*t) + 0.5*math.Sin(2*math.Pi*f2*t)
		binary.Write(buf, binary.LittleEndian, int16(val*32767))
	}

	go func() {
		cmd := exec.Command("aplay", "-f", "S16_LE", "-r", "44100", "-c", "1", "-q")
		cmd.Stdin = buf
		_ = cmd.Run()
	}()
}
