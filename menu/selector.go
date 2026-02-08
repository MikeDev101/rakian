package menu

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"misc"
	"sh1107"
)

type Selector struct {
	ctx         context.Context
	configured  bool
	cancelFn    context.CancelFunc
	parent      *Menu
	wg          sync.WaitGroup
	selection   int
	viewOffset  int
	title       string
	buttonlabel string
	options     [][]string
	path        []string
	visibleRows int
}

type SelectorArgs struct {
	VisibleRows int
	Title       string
	Options     [][]string
	ButtonLabel string
}

type SelectorReturn struct {
	SelectionPath []string
}

func (m *Menu) NewSelector() *Selector {
	return &Selector{
		parent:    m,
		selection: 0,
		title:     "",
		options:   [][]string{},
		path:      []string{},
	}
}

func (instance *Selector) get_current_options() []string {
	if len(instance.path) == 0 {
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
		if len(row) > 0 && row[0] == instance.path[len(instance.path)-1] {
			return row[1:]
		}
	}
	return []string{}
}

func (instance *Selector) render() {
	display := instance.parent.Display

	display.Clear(sh1107.Black)

	font := display.Use_Font8_Normal()
	display.DrawText(0, 20, font, instance.title, false)

	current_options := instance.get_current_options()

	// Determine starting item index based on selection
	if instance.selection < instance.viewOffset {
		instance.viewOffset = instance.selection
	} else if instance.selection >= instance.viewOffset+instance.visibleRows {
		instance.viewOffset = instance.selection - instance.visibleRows + 1
	}

	start := int(instance.viewOffset)
	end := start + instance.visibleRows
	if int(end) > len(current_options) {
		end = len(current_options)
	}
	display.SetColor(sh1107.White)
	display.SetLineWidth(1)
	display.DrawLine(0, 33, 127, 33)
	display.Stroke()

	font = display.Use_Font8_Bold()
	for i, opt := range current_options[start:end] {
		y := 40 + i*20 // Adjust for font height and spacing
		if start+i == int(instance.selection) {
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

// ConfigureWithArgs configures the selector with the given arguments.
// The first argument must be a SelectorArgs type, which contains the title and options for the selector.
// If the first argument is not a SelectorArgs type, a panic will occur.
// After calling ConfigureWithArgs, the selector must be configured before calling Run().
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
	instance.path = []string{}

	// Reset context
	instance.Configure()
}

func (instance *Selector) Run() {
	if !instance.configured {
		panic("Attempted to call (*Selector).Run() before (*Selector).Configure()!")
	}

	instance.render()
	instance.wg.Add(1)
	go func() {
		defer instance.wg.Done()
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

					current_options := instance.get_current_options()
					log.Println("Current options: ", current_options)

					switch evt.Key {
					case 'U':
						go instance.parent.PlayKey()
						if instance.selection > 0 {
							instance.selection -= 1
							log.Println("Selection: ", current_options[instance.selection])
							instance.render()
						}
					case 'D':
						go instance.parent.PlayKey()
						if instance.selection < len(current_options)-1 {
							instance.selection += 1
							log.Println("Selection: ", current_options[instance.selection])
							instance.render()
						}
					case 'S':
						go instance.parent.PlayKey()

						// Check if we can go deeper
						selected_option := current_options[instance.selection]
						has_children := false

						log.Println("Selection chosen: ", selected_option)

						// Only root items have children in this structure
						if len(instance.path) == 0 {
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

						instance.path = append(instance.path, selected_option)

						if has_children {
							instance.selection = 0
							instance.viewOffset = 0
							instance.render()
						} else {
							// Return to the previous menu with our chosen selection
							go instance.parent.PopWithArgs(&SelectorReturn{
								SelectionPath: instance.path,
							})
							return
						}

					case 'C':
						go instance.parent.PlayKey()

						if len(instance.path) > 0 {
							instance.path = instance.path[:len(instance.path)-1]
							instance.selection = 0
							instance.viewOffset = 0
							instance.render()
						} else {
							// Return to the previous menu with an empty selection
							go instance.parent.PopWithArgs(&SelectorReturn{
								SelectionPath: []string{},
							})
							return
						}

					case 'P':
						go instance.parent.PlayKey()
						go instance.parent.Pop()
						return

					default:
						go instance.parent.PlayKey()

						// If key is a number in range of options, select it
						if evt.Key > '0' && evt.Key <= '9' {

							// Convert evt.Key to int
							idx := int(evt.Key-'0') - 1
							if idx >= 0 && idx < len(current_options) {
								instance.selection = idx

								selected_option := current_options[instance.selection]
								has_children := false

								if len(instance.path) == 0 {
									for _, row := range instance.options {
										if len(row) > 0 && row[0] == selected_option {
											if len(row) > 1 {
												has_children = true
											}
											break
										}
									}
								}

								instance.path = append(instance.path, selected_option)

								if has_children {
									instance.selection = 0
									instance.viewOffset = 0
									instance.render()
								} else {
									go instance.parent.PopWithArgs(&SelectorReturn{
										SelectionPath: instance.path,
									})
									return
								}
							}
						}
					}
				}
			}
		}
	}()
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
		go instance.cleanup()
	}
}

func (instance *Selector) cleanup() {
	instance.selection = 0
	instance.viewOffset = 0
	instance.title = ""
	instance.options = [][]string{}
	instance.path = []string{}
}
