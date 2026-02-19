package menu

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"misc"
	"sh1107"
)

type Selector struct {
	ctx                        context.Context
	configured                 bool
	cancelFn                   context.CancelFunc
	parent                     *Menu
	wg                         sync.WaitGroup
	title                      string
	buttonlabel                string
	options                    [][]string
	selectors                  map[string]*SelectorState
	allowNumbKeys              bool
	showPathInTitle            bool
	showElemNumbersInSelection bool
	showElemNumberInTitle      bool
	visibleRows                int
	selectionclass             string
}

type SelectorState struct {
	path       []string
	selection  int
	viewOffset int
	persist    bool
}

type SelectorArgs struct {
	PersistLastState           bool
	VisibleRows                int
	SelectionClass             string
	Title                      string
	Options                    [][]string
	ButtonLabel                string
	AllowNumberKeyShortcut     bool
	ShowElemNumbersInSelection bool
	ShowElemNumberInTitle      bool
	ShowPathInTitle            bool
}

type SelectorReturn struct {
	SelectionClass string
	SelectionPath  []string
}

func (*Selector) Label() string {
	return "Selector"
}

func (m *Menu) NewSelector() *Selector {
	return &Selector{
		parent:    m,
		title:     "",
		options:   [][]string{},
		selectors: make(map[string]*SelectorState),
	}
}

func (instance *Selector) get_current_options() []string {
	state := instance.selectors[instance.selectionclass]
	if len(state.path) == 0 {
		// Root level: return the first element of each option row
		opts := make([]string, len(instance.options))
		for i, row := range instance.options {
			if len(row) > 0 {
				opts[i] = row[0]
			}
		}
		return opts
	}
	for _, row := range instance.options {
		if len(row) > 0 && row[0] == state.path[len(state.path)-1] {
			return row[1:]
		}
	}
	return []string{}
}

func (instance *Selector) render() {
	display := instance.parent.Display

	display.Clear(sh1107.Black)

	font := display.Use_Font8_Normal()

	state := instance.selectors[instance.selectionclass]

	if instance.showPathInTitle && len(state.path) > 0 {
		display.DrawText(0, 20, font, strings.Join(state.path, "/ "), false)

	} else {
		display.DrawText(0, 20, font, instance.title, false)
	}

	current_options := instance.get_current_options()

	// Determine starting item index based on selection
	if state.selection < state.viewOffset {
		state.viewOffset = state.selection
	} else if state.selection >= state.viewOffset+instance.visibleRows {
		state.viewOffset = state.selection - instance.visibleRows + 1
	}

	start := int(state.viewOffset)
	end := start + instance.visibleRows
	if int(end) > len(current_options) {
		end = len(current_options)
	}
	if start > end {
		start = end
	}
	display.SetColor(sh1107.White)
	display.SetLineWidth(1)
	display.DrawLine(0, 33, 127, 33)
	display.Stroke()

	font = display.Use_Font8_Bold()

	if instance.showElemNumberInTitle {
		display.DrawTextAligned(128, 20, font, fmt.Sprintf("%d", int(state.selection+1)), false, sh1107.AlignLeft, sh1107.AlignNone)
	}

	for i, opt := range current_options[start:end] {

		if instance.showElemNumbersInSelection {
			opt = fmt.Sprintf("%d. %s", start+i+1, opt)
		}

		y := 40 + i*20 // Adjust for font height and spacing
		if start+i == int(state.selection) {
			// Draw selection highlight box
			display.SetColor(sh1107.White)
			display.DrawRectangle(0, float64(y-1), 127, 16)
			display.Fill()
			display.DrawText(2, y+4, font, opt, true)
		} else {
			display.DrawText(2, y+4, font, opt, false)
		}
	}

	display.DrawTextAligned(64, 105, font, instance.buttonlabel, false, sh1107.AlignCenter, sh1107.AlignNone)

	display.Render()
}

func (instance *Selector) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

// ConfigureWithArgs configures the Selector with the given arguments.
// The first argument must be a *SelectorArgs type, which contains the
// title, options, button label, visible rows, and other options.
// If the first argument is not a *SelectorArgs type, a panic will occur.
// After calling ConfigureWithArgs, the Selector must be configured before
// calling Run().
// If persistLastState is false, the selection will be reset to 0 and the
// view offset will be reset to 0.
func (instance *Selector) ConfigureWithArgs(args ...any) {

	// See if we have args
	if len(args) < 1 {
		panic("(*Selector).ConfigureWithArgs() requires at least one argument")
	}

	// Cast and check args
	selector_args, ok := args[0].(*SelectorArgs)
	if !ok {
		panic(fmt.Sprintf("(*Selector).ConfigureWithArgs() Type error: argument must be a *SelectorArgs type, got %s", args[0]))
	}

	// Set title and options
	instance.title = selector_args.Title
	instance.options = selector_args.Options
	instance.buttonlabel = selector_args.ButtonLabel
	instance.visibleRows = selector_args.VisibleRows
	instance.allowNumbKeys = selector_args.AllowNumberKeyShortcut
	instance.showPathInTitle = selector_args.ShowPathInTitle
	instance.showElemNumbersInSelection = selector_args.ShowElemNumbersInSelection
	instance.showElemNumberInTitle = selector_args.ShowElemNumberInTitle
	instance.selectionclass = selector_args.SelectionClass

	if instance.selectionclass == "" {
		panic("(*Selector).ConfigureWithArgs() requires a selection class")
	}

	if instance.selectors == nil {
		instance.selectors = make(map[string]*SelectorState)
	}

	if e, ok := instance.selectors[instance.selectionclass]; !ok {
		instance.selectors[instance.selectionclass] = &SelectorState{
			path:       []string{},
			selection:  0,
			viewOffset: 0,
			persist:    selector_args.PersistLastState,
		}
	} else if !e.persist {
		e.path = []string{}
		e.selection = 0
		e.viewOffset = 0
	}

	// Reset context
	instance.Configure()
}

func (instance *Selector) Run() {
	if !instance.configured {
		panic("Attempted to call (*Selector).Run() before (*Selector).Configure()!")
	}

	// Wait for display to be ready
	instance.parent.Display.Ready()

	instance.render()
	instance.wg.Go(func() {
		for {
			select {
			case <-instance.ctx.Done():
				return
			case evt := <-instance.parent.KeypadEvents:
				if evt.State {

					instance.parent.Timers["keypad"].Reset()
					instance.parent.Timers["oled"].Reset()
					instance.parent.Display.On()
					misc.KeyLightsOn()

					state := instance.selectors[instance.selectionclass]
					current_options := instance.get_current_options()
					log.Println("Current options: ", current_options)

					switch evt.Key {
					case 'U':
						go instance.parent.PlayKey()
						if state.selection == 0 {
							state.selection = len(current_options) - 1
						} else if state.selection > 0 {
							state.selection -= 1
						}
						log.Println("Selection: ", current_options[state.selection])
						instance.render()
					case 'D':
						go instance.parent.PlayKey()
						if state.selection < len(current_options)-1 {
							state.selection += 1
						} else if state.selection == len(current_options)-1 {
							state.selection = 0
						}
						log.Println("Selection: ", current_options[state.selection])
						instance.render()

					case 'S':
						go instance.parent.PlayKey()

						if len(current_options) == 0 {
							continue
						}

						// Check if we can go deeper
						selected_option := current_options[state.selection]
						has_children := false

						log.Println("Selection chosen: ", selected_option)

						// Only root items have children in this structure
						if len(state.path) == 0 {
							for i, row := range instance.options {
								log.Println(i, row)
								if len(row) > 0 && row[0] == selected_option {
									if len(row) > 1 {
										log.Println("Selection has children: ", row[1:])
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
							instance.render()
						} else {
							// Return to the previous menu with our chosen selection
							go instance.parent.PopWithArgs(&SelectorReturn{
								SelectionClass: instance.selectionclass,
								SelectionPath:  append(state.path, selected_option),
							})
							return
						}

					case 'C':
						go instance.parent.PlayKey()

						if len(state.path) > 0 {
							state.path = state.path[:len(state.path)-1]
							state.selection = 0
							state.viewOffset = 0
							instance.render()
						} else {
							state.selection = 0
							state.viewOffset = 0
							state.path = []string{}

							// Return to the previous menu with an empty selection
							go instance.parent.PopWithArgs(&SelectorReturn{
								SelectionClass: instance.selectionclass,
								SelectionPath:  []string{},
							})
							return
						}

					case 'P':
						go instance.parent.PlayKey()
						go instance.parent.Push("power")
						return

					default:
						go instance.parent.PlayKey()

						// Allow number key shortcut if it is enabled
						if !instance.allowNumbKeys {
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

								if len(state.path) == 0 {
									for _, row := range instance.options {
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
									instance.render()
								} else {
									go instance.parent.PopWithArgs(&SelectorReturn{
										SelectionClass: instance.selectionclass,
										SelectionPath:  append(state.path, selected_option),
									})
									return
								}
							}
						}
					}
				}
			}
		}
	})
}

func (instance *Selector) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Selector pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *Selector) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Selector stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		instance.cleanup()
	}
}

func (instance *Selector) cleanup() {
	instance.title = ""
	instance.options = [][]string{}
	if state, ok := instance.selectors[instance.selectionclass]; ok {
		if !state.persist {
			state.path = []string{}
			state.selection = 0
			state.viewOffset = 0
		}
	}
}
