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
	"misc"
	"os/exec"
	"strings"
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

func (instance *Menu) InputPrompt(args InputPromptArgs, ctx context.Context) string {

	// Text entry handler
	var input []rune
	cursorPos := 0
	scrollOffset := 0
	display := instance.Display

	// Key mapping
	keyMap := map[rune]string{
		'1': ".,?!-&`:1", '2': "abc2", '3': "def3",
		'4': "ghi4", '5': "jkl5", '6': "mno6",
		'7': "pqrs7", '8': "tuv8", '9': "wxyz9",
		'0': " 0",
	}

	// T9 mode mapping
	t9Map := map[int]string{
		T9Lowercase: "lowercase",
		T9Uppercase: "uppercase",
		T9Numbers:   "numbers",
	}

	t9Mode := T9Lowercase
	var lastKey rune
	var lastPressTime time.Time
	var lastAsteriskTime time.Time
	var cycleIndex int

	if args.PhoneNumberOnly {
		t9Mode = T9Numbers
	}

	// Temporarily stop timeouts
	instance.Timers["screensaver"].Stop()
	instance.Timers["keypad"].Stop()
	misc.KeyLightsOn()
	defer instance.Timers["screensaver"].Restart()
	defer instance.Timers["keypad"].Restart()

	render := func() {
		const visibleWidth = 76
		const textStartX = 4

		display.Clear(lcd.White)

		// Draw mode helper
		display.DrawImage(display.Use_Sprites()["pencil"], 0, 0)
		if !args.PhoneNumberOnly {
			display.DrawImageAligned(display.Use_Sprites()[t9Map[t9Mode]], 13, 0, lcd.AlignNone, lcd.AlignNone)
		}

		// Draw label
		font := display.Use_Font_Small_Bold()
		display.DrawTextWrapped(0, 8, 84, 16, font, args.Title, false, lcd.WrapDown, lcd.WrapLeft)

		// Draw box
		display.SetColor(lcd.Black)
		display.SetLineWidth(1)
		display.DrawRectangle(0.5, 18.5, 83, 17)
		display.Stroke()

		// Draw text
		font = display.Use_Font_Large_Bold()

		// Calculate cursor position and scroll offset
		prefix := string(input[:cursorPos])
		cursorPx, _ := display.GetTextBounds(font, prefix)

		if cursorPx < scrollOffset {
			scrollOffset = cursorPx
		}
		if cursorPx > scrollOffset+visibleWidth {
			scrollOffset = cursorPx - visibleWidth
		}

		display.DrawText(textStartX-scrollOffset, 20, font, string(input), false)

		// Draw cursor
		curX := float64(textStartX+cursorPx-scrollOffset) + 0.5
		display.DrawLine(curX, 20, curX, 33)
		display.Stroke()

		// Draw bottom
		if len(input) == 0 {
			instance.RenderFooter("Cancel", true)
		} else {
			instance.RenderFooter("OK", true)
		}
		display.Render()
	}

	render()

	for {
		select {
		case <-ctx.Done():
			return ""
		case evt := <-instance.KeypadEvents:
			if !evt.State {
				continue
			}
			misc.KeyLightsOn()
			go instance.PlayKey()

			now := time.Now()

			switch evt.Key {
			case 'S':
				return string(input)
			case 'C':
				lastKey = 0
				if len(input) > 0 {
					if cursorPos > 0 {
						input = append(input[:cursorPos-1], input[cursorPos:]...)
						cursorPos--
					}
					render()
				} else {
					return ""
				}
			case 'P':
				go instance.Push("power")
				return ""
			case 'U':
				lastKey = 0
				if cursorPos > 0 {
					cursorPos--
					render()
				}
			case 'D':
				lastKey = 0
				if cursorPos < len(input) {
					cursorPos++
					render()
				}
			case '#':
				if args.PhoneNumberOnly {
					if args.CharacterLimit > 0 && len(input) >= args.CharacterLimit {
						continue
					}
					input = append(input[:cursorPos], append([]rune{'#'}, input[cursorPos:]...)...)
					cursorPos++
					render()
					continue
				}

				lastKey = 0
				t9Mode = (t9Mode + 1) % 3
				render()
			case '*':
				if args.PhoneNumberOnly {
					if now.Sub(lastAsteriskTime) <= 750*time.Millisecond {
						if cursorPos > 0 && input[cursorPos-1] == '*' {
							input[cursorPos-1] = '+'
							lastAsteriskTime = now
							render()
							continue
						}
					}
					if args.CharacterLimit > 0 && len(input) >= args.CharacterLimit {
						continue
					}
					input = append(input[:cursorPos], append([]rune{'*'}, input[cursorPos:]...)...)
					lastAsteriskTime = now
					cursorPos++
					render()
					continue
				}

				lastKey = 0
				// Symbol selector
				symbols := []rune(".,?!:;-+#*()'\"_@&$£%/<>¿¡§=¤¥є")
				selIdx := 0

				// Mini loop for symbol selection
			symbolLoop:
				for {
					display.Clear(lcd.White)

					// Draw mode helper
					display.DrawImage(display.Use_Sprites()["pencil"], 0, 0)
					display.DrawImageAligned(display.Use_Sprites()["symbols"], 13, 0, lcd.AlignNone, lcd.AlignNone)

					// Draw symbols grid
					for i, r := range symbols {
						col := i % 10
						row := i / 10
						x := 2 + col*8
						y := 8 + row*10
						if i == selIdx {
							display.SetColor(lcd.Black)
							display.DrawRectangle(float64(x), float64(y)-1, 8, 10)
							display.Fill()
							display.DrawText(x+2, y, display.Use_Font_Small_Bold(), string(r), true)
						} else {
							display.DrawText(x+2, y, display.Use_Font_Small_Plain(), string(r), false)
						}
					}

					instance.RenderFooter("Use", true)
					display.Render()

					select {
					case <-ctx.Done():
						return ""
					case sEvt := <-instance.KeypadEvents:
						if !sEvt.State {
							continue
						}
						go instance.PlayKey()
						switch sEvt.Key {
						case 'U': // Left
							if selIdx > 0 {
								selIdx--
							} else if selIdx == 0 {
								selIdx = len(symbols) - 1
							}
						case 'D': // Right
							if selIdx < len(symbols)-1 {
								selIdx++
							}
						case '2': // Up
							if selIdx >= 10 {
								selIdx -= 10
							}
						case '8': // Down
							if selIdx+10 < len(symbols) {
								selIdx += 10
							}
						case 'S': // Select
							// Insert symbol
							if args.CharacterLimit > 0 && len(input) >= args.CharacterLimit {
								break symbolLoop
							}
							input = append(input[:cursorPos], append([]rune{symbols[selIdx]}, input[cursorPos:]...)...)
							cursorPos++
							break symbolLoop
						case 'C': // Cancel
							break symbolLoop
						}
					}
				}
				render()

			default:
				if args.PhoneNumberOnly {
					if evt.Key >= '0' && evt.Key <= '9' {
						if args.CharacterLimit > 0 && len(input) >= args.CharacterLimit {
							continue
						}
						input = append(input[:cursorPos], append([]rune{evt.Key}, input[cursorPos:]...)...)
						cursorPos++
						render()
					}
					continue
				}

				chars, ok := keyMap[evt.Key]
				if ok {
					// Adjust chars based on mode
					switch t9Mode {
					case T9Numbers:
						chars = string(evt.Key)
					case T9Uppercase:
						chars = strings.ToUpper(chars)
					}

					if evt.Key == lastKey && now.Sub(lastPressTime) < 1*time.Second {
						// Cycle
						cycleIndex = (cycleIndex + 1) % len(chars)
						if len(input) > 0 {
							// Replace char at cursor-1
							if cursorPos > 0 {
								input[cursorPos-1] = rune(chars[cycleIndex])
							}
						}
					} else {
						// New char
						cycleIndex = 0
						if args.CharacterLimit > 0 && len(input) >= args.CharacterLimit {
							continue
						}
						// Insert at cursor
						newChar := rune(chars[0])
						input = append(input[:cursorPos], append([]rune{newChar}, input[cursorPos:]...)...)
						cursorPos++
					}
					lastKey = evt.Key
					lastPressTime = now
					render()
				}
			}
		}
	}
}
