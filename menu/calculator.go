package menu

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"misc"
	"sh1107"
)

type CalculatorMenu struct {
	ctx               context.Context
	configured        bool
	cancelFn          context.CancelFunc
	parent            *Menu
	wg                sync.WaitGroup
	calc_input        []rune
	calc_displayed    string
	lastAsteriskTime  time.Time
	process_selection bool
	selection_path    []string
}

func (m *Menu) NewCalculatorMenu() *CalculatorMenu {
	return &CalculatorMenu{
		parent:           m,
		lastAsteriskTime: time.Now(),
		/*options: [][]string{
			{"1. Equals"},
			{"2. Clear"},
			{"3. Exchange rate", "1. Foreign unit expressed as domestic units", "2. Domestic unit expressed as foreign units"},
			{"4. To domestic"},
			{"5. To foreign"},
		},*/
	}
}

func (instance *CalculatorMenu) render() {
	display := instance.parent.Display
	display.Clear(sh1107.Black)

	font := display.Use_Font8_Normal()
	display.DrawTextAligned(0, 20, font, "Calculator", false, sh1107.AlignRight, sh1107.AlignNone)

	font = display.Use_Font8_Bold()
	display.DrawTextAligned(64, 105, font, "Options", false, sh1107.AlignCenter, sh1107.AlignNone)
	display.DrawText(0, 40, display.Use_Font16(), instance.calc_displayed, false)

	display.SetColor(sh1107.White)
	display.SetLineWidth(1)
	display.DrawLine(0, 33, 127, 33)
	display.Stroke()

	display.Render()
}

func (instance *CalculatorMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *CalculatorMenu) ConfigureWithArgs(args ...any) {

	// Check if we have args
	if len(args) > 0 {

		// Most likely our arg is a SelectorReturn from the selector.
		selection, ok := args[0].(*SelectorReturn)
		if !ok {
			panic("(*CalculatorMenu).ConfigureWithArgs() Type error: argument must be a *SelectorReturn type")
		}

		instance.process_selection = true
		instance.selection_path = selection.SelectionPath
	}

	instance.Configure()
}

func (instance *CalculatorMenu) compute_displayed() {
	var number_groups = []string{}

	var numbs_valid = "0123456789."
	for _, c := range instance.calc_input {
		if strings.ContainsRune(numbs_valid, c) {

			if len(number_groups) == 0 {
				number_groups = append(number_groups, "")
			}

			number_groups[len(number_groups)-1] += string(c)

		} else {
			number_groups = append(number_groups, string(c))
			number_groups = append(number_groups, "")
		}
	}

	instance.calc_displayed = strings.Join(number_groups, " ")
}

func (instance *CalculatorMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*CalculatorMenu).Run() before (*CalculatorMenu).Configure()!")
	}

	if instance.process_selection {
		instance.process_selection = false

		// Do nothing if the selection path is empty (i.e. user cancelled the selector)
		if len(instance.selection_path) > 0 {

			switch instance.selection_path[0] {
			case "1. Equals":
				log.Println("Calculating current expression...")
				// TODO: calculate the result of the current expression

			case "2. Clear":
				instance.calc_input = []rune{}
				instance.calc_displayed = ""
				log.Println("Clearing current expression...")

			case "3. Exchange rate":
				// Check if the user selected a suboption
				if len(instance.selection_path) > 1 {
					switch instance.selection_path[1] {
					case "1. Foreign as domestic units":
						log.Println("Storing foreign unit exchange rate expressed as domestic unit...")
						// TODO

					case "2. Domestic as foreign units":
						log.Println("Storing domestic unit exchange rate expressed as foreign unit...")
						// TODO
					}
				}

			case "4. To domestic":
				log.Println("Converting current value (in foreign) to domestic based on stored exchange rate...")
				// TODO

			case "5. To foreign":
				log.Println("Converting current value (in domestic) to foreign based on stored exchange rate...")
				// TODO
			}

		}
	}

	instance.render()

	var operators = []rune{'+', '-', '*', '/'}
	var operatorIndex int
	instance.wg.Add(1)
	defer instance.wg.Done()
	for {
		select {
		case <-instance.ctx.Done():
			return

		case evt := <-instance.parent.KeypadEvents:

			log.Println(evt)

			if evt.State {

				instance.parent.Timers["keypad"].Reset()
				instance.parent.Timers["oled"].Reset()
				instance.parent.Display.On()
				misc.KeyLightsOn()
				go instance.parent.PlayKey()

				switch evt.Key {
				case '*':
					now := time.Now()

					if now.Sub(instance.lastAsteriskTime) <= 750*time.Millisecond {
						// Cycle to next operator
						operatorIndex = (operatorIndex + 1) % len(operators)

						if len(instance.calc_input) > 0 {
							instance.calc_input[len(instance.calc_input)-1] = operators[operatorIndex]
						} else {
							instance.calc_input = append(instance.calc_input, operators[operatorIndex])
						}
					} else {
						// Reset cycle and start with first operator
						operatorIndex = 0
						instance.calc_input = append(instance.calc_input, operators[operatorIndex])
					}

					instance.lastAsteriskTime = now
					instance.compute_displayed()
					instance.render()

				case 'C':
					if len(instance.calc_input) == 0 {
						go instance.parent.Pop()
						return
					}

					if len(instance.calc_input) > 0 {
						instance.calc_input = instance.calc_input[:len(instance.calc_input)-1]
					}

					instance.compute_displayed()
					instance.render()

				case 'P':
					go instance.parent.Push("power")
					return

				case '#':
					instance.calc_input = append(instance.calc_input, '.')
					instance.compute_displayed()
					instance.render()

				case 'S':
					go instance.parent.PushWithArgs("selector", &SelectorArgs{
						Title: "Calculator",
						Options: [][]string{
							{"1. Equals"},
							{"2. Clear"},
							{"3. Exchange rate", "1. Foreign as domestic units", "2. Domestic as foreign units"},
							{"4. To domestic"},
							{"5. To foreign"},
						},
						ButtonLabel: "Select",
						VisibleRows: 3,
					})
					return

				case 'U':
				case 'D':

				default:
					// Read input
					instance.calc_input = append(instance.calc_input, evt.Key)
					instance.compute_displayed()
					instance.render()

				}
			}
		}
	}
}

func (instance *CalculatorMenu) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Calculator menu pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *CalculatorMenu) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Calculator menu stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		instance.cleanup()
	}
}

func (instance *CalculatorMenu) cleanup() {
	instance.calc_input = []rune{}
	instance.calc_displayed = ""
	instance.selection_path = []string{}
	instance.process_selection = false
}
