package menu

import (
	"context"
	"fmt"
	"gfx"
	"strings"
	"time"
)

type InputPromptArgs struct {
	Title           string
	DefaultValue    string
	CharacterLimit  int
	PhoneNumberOnly bool
}

func (instance *Menu) InputPrompt(args InputPromptArgs, ctx context.Context) string {

	// Text entry handler
	var input []rune
	cursorPos := 0
	scrollOffset := 0
	display := instance.Display

	// Fill in default value
	input = append(input, []rune(args.DefaultValue)...)

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
	instance.Keypad.KeyLightsOn()
	defer instance.Timers["screensaver"].Restart()
	defer instance.Timers["keypad"].Restart()

	render := func() {
		const visibleWidth = 76
		const textStartX = 4

		display.Clear(display.Primary())

		// Draw mode helper
		display.DrawImage(display.Use_Sprites()["pencil"], 0, 0)
		if !args.PhoneNumberOnly {
			display.DrawImageAligned(display.Use_Sprites()[t9Map[t9Mode]], 13, 0, gfx.AlignNone, gfx.AlignNone)
		}

		// Draw labels
		font := display.Use_Font_Small_Bold()
		display.DrawTextWrapped(0, 8, 84, 16, font, args.Title, false, gfx.WrapDown, gfx.WrapLeft)
		if args.CharacterLimit > 0 {
			display.DrawTextAligned(84, 0, font, fmt.Sprintf("%d", args.CharacterLimit-len(input)), false, gfx.AlignLeft, gfx.AlignBelow)
		}

		// Draw box
		display.SetColor(display.Secondary())
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

	cancelChan := make(chan struct{})
	var cTimer *time.Timer
	defer func() {
		if cTimer != nil {
			cTimer.Stop()
		}
	}()

	for {
		select {
		case <-cancelChan:
			return ""
		case <-ctx.Done():
			return ""
		case evt := <-instance.KeypadEvents:
			if evt.Key == 'C' {
				if evt.State {
					if cTimer != nil {
						cTimer.Stop()
					}
					cTimer = time.AfterFunc(1*time.Second, func() {
						close(cancelChan)
					})

					instance.Keypad.KeyLightsOn()
					go instance.PlayKey()

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
				} else {
					if cTimer != nil {
						cTimer.Stop()
					}
				}
				continue
			}

			if !evt.State {
				continue
			}
			instance.Keypad.KeyLightsOn()
			go instance.PlayKey()

			now := time.Now()

			switch evt.Key {
			case 'S':
				return string(input)
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
					display.Clear(display.Primary())

					// Draw mode helper
					display.DrawImage(display.Use_Sprites()["pencil"], 0, 0)
					display.DrawImageAligned(display.Use_Sprites()["symbols"], 13, 0, gfx.AlignNone, gfx.AlignNone)

					// Draw symbols grid
					for i, r := range symbols {
						col := i % 10
						row := i / 10
						x := 2 + col*8
						y := 8 + row*10
						if i == selIdx {
							display.SetColor(display.Secondary())
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
							} else if selIdx == len(symbols)-1 {
								selIdx = 0
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
