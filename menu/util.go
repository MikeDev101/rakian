package menu

import (
	"context"
	"fmt"
	"gfx"
	"image"
	"log"
	"time"
)

const (
	T9Lowercase = 0
	T9Uppercase = 1
	T9Numbers   = 2
)

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
	m.Display.Clear(m.Display.Primary())
	if icon != "" {
		if sprite, ok := m.Display.Use_Sprites()[icon]; ok {
			m.Display.DrawImageAligned(sprite, 84, 0, gfx.AlignLeft, gfx.AlignBelow)
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
	m.Display.Clear(m.Display.Primary())
	for i, str := range status {
		m.Display.DrawText(0, 0+(i*16), font, str, false)
	}
	m.Display.Render()
	go m.Display.PlayAnimation(ctx, animation, 84, 0, gfx.AlignLeft, gfx.AlignBelow)
}

func (m *Menu) RenderFooter(footer string, bold bool) {
	display := m.Display
	var font map[rune]*image.Image
	if bold {
		font = display.Use_Font_Small_Bold()
	} else {
		font = display.Use_Font_Small_Plain()
	}
	display.DrawTextAligned(42, 48, font, footer, false, gfx.AlignCenter, gfx.AlignAbove)
}

func (m *Menu) RenderHeader(header string) {
	display := m.Display
	font := display.Use_Font_Small_Plain()
	width, _ := display.GetTextBounds(font, header)
	display.DrawTextAligned(42, 0, font, header, false, gfx.AlignCenter, gfx.AlignBelow)

	// If the header text is too long, limit the width to 84
	if width > 84 {
		width = 84
	}

	// Get the positions of the left and right lines
	left_start := 42 - (width / 2) - 2
	right_start := 42 + (width / 2) + 1

	// Draw lines around the header text
	display.SetColor(display.Secondary())
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
	display.DrawTextAligned(77, 0, font, clock_str, false, gfx.AlignLeft, gfx.AlignNone)

	// Draw icons
	if cell_sprite, ok := display.Use_Sprites()["cell"]; ok {
		display.DrawImageAligned(cell_sprite, 0, 38, gfx.AlignRight, gfx.AlignAbove)
	}
	if battery_sprite, ok := display.Use_Sprites()["battery"]; ok {
		display.DrawImageAligned(battery_sprite, 84, 38, gfx.AlignLeft, gfx.AlignAbove)
	}

	battery_state := m.Get("BatteryScaledPercent").(int)
	if battery_state > 0 {
		display.SetColor(display.Secondary())
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
		display.SetColor(display.Secondary())
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

	instance.Player.PlayDTMF(key)
}

func (instance *Menu) stopDTMF(key rune) {
	// Don't generate tones if we're mute
	if !instance.Get("CanRing").(bool) && !instance.Get("BeepOnly").(bool) {
		return
	}
	instance.Player.StopDTMF(key)
}
