package menu

import (
	"context"
	"strings"

	"lcd"
	"misc"
)

const (
	SELECTOR_SIMPLE           = 0 // Current implementation.
	SELECTOR_MULTI_3          = 1
	SELECTOR_MULTI_4          = 2
	SELECTOR_MULTI_5          = 3
	SELECTOR_SIMPLE_WITH_INFO = 4
)

type SelectorState struct {
	path         []string
	selectortype int
	selection    int
	persist      bool
	viewOffset   int
}

type SelectorArgs struct {
	PersistLastState       bool
	SelectionClass         string
	SelectorType           int
	Title                  string
	Options                [][]string
	ButtonLabel            string
	AllowNumberKeyShortcut bool
	ShowTitle              bool
	ShowPathInTitle        bool
	AppendTextToElemNumber string
}

type SelectorReturn struct {
	SelectionClass string
	SelectionPath  []string
}

func (m *Menu) ShowSelector(args SelectorArgs, ctx context.Context) *SelectorReturn {
	// Initialize state
	if m.SelectorStates == nil {
		m.SelectorStates = make(map[string]*SelectorState)
	}

	state, ok := m.SelectorStates[args.SelectionClass]
	if !ok {
		state = &SelectorState{
			path:       []string{},
			selection:  0,
			persist:    args.PersistLastState,
			viewOffset: 0,
		}
		m.SelectorStates[args.SelectionClass] = state
	} else if !args.PersistLastState {
		state.path = []string{}
		state.selection = 0
		state.viewOffset = 0
	}

	// Helper to get current options based on path
	get_current_options := func() []string {
		if len(state.path) == 0 {
			// Root level: return the first element of each option row
			opts := make([]string, len(args.Options))
			for i, row := range args.Options {
				if len(row) > 0 {
					opts[i] = row[0]
				}
			}
			return opts
		}
		for _, row := range args.Options {
			if len(row) > 0 && row[0] == state.path[len(state.path)-1] {
				return row[1:]
			}
		}
		return []string{}
	}

	// Temporarily stop timeouts
	m.Timers["screensaver"].Stop()
	m.Timers["keypad"].Stop()
	misc.KeyLightsOn()
	defer m.Timers["screensaver"].Restart()
	defer m.Timers["keypad"].Restart()

	render := func() {
		display := m.Display
		var startY int

		display.Clear(lcd.White)

		if args.ShowTitle {
			if args.ShowPathInTitle && len(state.path) > 0 {
				m.RenderHeader(strings.Join(state.path, "/ "))
			} else if len(args.Title) > 0 {
				m.RenderHeader(args.Title)
			}
			startY = 9
		} else {
			startY = 0
		}

		current_options := get_current_options()

		if args.SelectorType == SELECTOR_SIMPLE || args.SelectorType == SELECTOR_SIMPLE_WITH_INFO {
			display.SetColor(lcd.White)
			display.SetLineWidth(1)
			display.DrawLine(0, 33, 127, 33)
			display.Stroke()

			font := display.Use_Font_Small_Bold()

			if len(current_options) > 0 && state.selection < len(current_options) {
				display.DrawText(0, 12, font, current_options[state.selection], false)
			}

			if args.SelectorType == SELECTOR_SIMPLE_WITH_INFO {
				// Show info text on bottom right
				if len(state.path) == 0 && state.selection < len(args.Options) && len(args.Options[state.selection]) > 1 {
					info := args.Options[state.selection][1]
					display.DrawTextAligned(84, 48, display.Use_Font_Small_Plain(), info, false, lcd.AlignRight, lcd.AlignAbove)
				}
			}
		} else {
			// Multi-row selector
			visibleRows := 3
			lineHeight := 10

			switch args.SelectorType {
			case SELECTOR_MULTI_3:
				visibleRows = 3
			case SELECTOR_MULTI_4:
				visibleRows = 4
			case SELECTOR_MULTI_5:
				visibleRows = 5
			}

			font := display.Use_Font_Small_Bold()
			cnt := len(current_options)
			rowsToDraw := visibleRows
			if cnt < visibleRows {
				rowsToDraw = cnt
			}

			for i := 0; i < rowsToDraw; i++ {
				idx := (state.viewOffset + i) % cnt
				y := startY + i*lineHeight
				opt := current_options[idx]

				if idx == state.selection {
					display.SetColor(lcd.Black)
					display.DrawRectangle(0, float64(y), 84, float64(lineHeight))
					display.Fill()
					display.DrawText(1, y+1, font, opt, true)
				} else {
					display.DrawText(1, y+1, font, opt, false)
				}
			}
		}

		m.RenderFooter(args.ButtonLabel, true)
		display.Render()
	}

	render()

	for {
		select {
		case <-ctx.Done():
			return nil
		case evt := <-m.KeypadEvents:
			if evt.State {
				misc.KeyLightsOn()

				current_options := get_current_options()

				switch evt.Key {
				case 'U':
					if len(current_options) == 0 {
						continue
					}
					go m.PlayKey()
					if state.selection == 0 {
						state.selection = len(current_options) - 1
					} else if state.selection > 0 {
						state.selection -= 1
					}

					// Update viewOffset for infinite scrolling (Up)
					visibleRows := 3
					switch args.SelectorType {
					case SELECTOR_MULTI_3:
						visibleRows = 3
					case SELECTOR_MULTI_4:
						visibleRows = 4
					case SELECTOR_MULTI_5:
						visibleRows = 5
					}
					cnt := len(current_options)
					if cnt > visibleRows {
						dist := (state.selection - state.viewOffset + cnt) % cnt
						if dist >= visibleRows {
							state.viewOffset = state.selection
						}
					}

					render()
				case 'D':
					if len(current_options) == 0 {
						continue
					}
					go m.PlayKey()
					if state.selection < len(current_options)-1 {
						state.selection += 1
					} else if state.selection == len(current_options)-1 {
						state.selection = 0
					}

					// Update viewOffset for infinite scrolling (Down)
					visibleRows := 3
					switch args.SelectorType {
					case SELECTOR_MULTI_3:
						visibleRows = 3
					case SELECTOR_MULTI_4:
						visibleRows = 4
					case SELECTOR_MULTI_5:
						visibleRows = 5
					}
					cnt := len(current_options)
					if cnt > visibleRows {
						dist := (state.selection - state.viewOffset + cnt) % cnt
						if dist >= visibleRows {
							state.viewOffset = (state.selection - (visibleRows - 1) + cnt) % cnt
						}
					}

					render()

				case 'S':
					go m.PlayKey()

					if len(current_options) == 0 {
						continue
					}

					// Check if we can go deeper
					selected_option := current_options[state.selection]
					has_children := false

					// Only root items have children in this structure
					if len(state.path) == 0 && args.SelectorType != SELECTOR_SIMPLE_WITH_INFO {
						for _, row := range args.Options {
							if len(row) > 0 && row[0] == selected_option {
								if len(row) > 1 {
									has_children = true
								}
								break
							}
						}
					}

					if has_children {
						state.selection = 0
						state.viewOffset = 0
						state.path = append(state.path, selected_option)
						render()
					} else {
						// Return to the previous menu with our chosen selection
						return &SelectorReturn{
							SelectionClass: args.SelectionClass,
							SelectionPath:  append(state.path, selected_option),
						}
					}

				case 'C':
					go m.PlayKey()

					if len(state.path) > 0 {
						state.path = state.path[:len(state.path)-1]
						state.selection = 0
						state.viewOffset = 0
						render()
					} else {
						state.selection = 0
						state.path = []string{}
						return nil
					}

				case 'P':
					// Prevent recursive power menu calls
					if args.SelectionClass == "power" {
						continue
					}

					go m.PlayKey()
					go m.Push("power")
					return nil

				default:
					go m.PlayKey()

					// Allow number key shortcut if it is enabled
					if !args.AllowNumberKeyShortcut {
						continue
					}

					// If key is a number in range of options, select it
					if evt.Key > '0' && evt.Key <= '9' {

						// Convert evt.Key to int
						idx := int(evt.Key-'0') - 1
						if idx >= 0 && idx < len(current_options) {
							state.selection = idx

							selected_option := current_options[state.selection]
							has_children := false

							if len(state.path) == 0 && args.SelectorType != SELECTOR_SIMPLE_WITH_INFO {
								for _, row := range args.Options {
									if len(row) > 0 && row[0] == selected_option {
										if len(row) > 1 {
											has_children = true
										}
										break
									}
								}
							}

							if has_children {
								state.selection = 0
								state.path = append(state.path, selected_option)
								render()
							} else {
								return &SelectorReturn{
									SelectionClass: args.SelectionClass,
									SelectionPath:  append(state.path, selected_option),
								}
							}
						}
					}
				}
			}
		}
	}
}
