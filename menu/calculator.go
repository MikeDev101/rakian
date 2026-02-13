package menu

import (
	"context"
	"fmt"
	"log"
	"strconv"
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

	/* TODO:
	 * - Determine what's a number group and track them for computation
	 * - Detect if groups of numbers have too many decimal points (there can only be one per group)
	 * - Detect if numbers are too big to fit in a float
	 */

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

func (instance *CalculatorMenu) evaluate() (float64, error) {
	expr := string(instance.calc_input)
	if len(expr) == 0 {
		return 0, nil
	}

	// 1. Tokenize
	var nums []float64
	var ops []rune

	var currentNum strings.Builder
	for i, r := range expr {
		if r == '+' || r == '-' || r == '*' || r == '/' {
			// Check for negative number at start or after operator
			if r == '-' && (i == 0 || strings.ContainsRune("+-*/", rune(expr[i-1]))) {
				currentNum.WriteRune(r)
				continue
			}

			if currentNum.Len() > 0 {
				val, err := strconv.ParseFloat(currentNum.String(), 64)
				if err != nil {
					return 0, err
				}
				nums = append(nums, val)
				currentNum.Reset()
			}
			ops = append(ops, r)
		} else {
			currentNum.WriteRune(r)
		}
	}
	if currentNum.Len() > 0 {
		val, err := strconv.ParseFloat(currentNum.String(), 64)
		if err != nil {
			return 0, err
		}
		nums = append(nums, val)
	}

	if len(nums) == 0 {
		return 0, nil
	}
	// If we have operators but not enough numbers (e.g. "5+")
	if len(ops) >= len(nums) {
		ops = ops[:len(nums)-1]
	}

	// 2. Process * and /
	var nums2 []float64
	var ops2 []rune

	nums2 = append(nums2, nums[0])
	for i := 0; i < len(ops); i++ {
		op := ops[i]
		nextNum := nums[i+1]

		if op == '*' || op == '/' {
			prevNum := nums2[len(nums2)-1]
			var res float64
			if op == '*' {
				res = prevNum * nextNum
			} else {
				if nextNum == 0 {
					return 0, fmt.Errorf("div by zero")
				}
				res = prevNum / nextNum
			}
			nums2[len(nums2)-1] = res
		} else {
			nums2 = append(nums2, nextNum)
			ops2 = append(ops2, op)
		}
	}

	// 3. Process + and -
	result := nums2[0]
	for i := 0; i < len(ops2); i++ {
		op := ops2[i]
		nextNum := nums2[i+1]
		if op == '+' {
			result += nextNum
		} else {
			result -= nextNum
		}
	}

	return result, nil
}

func (instance *CalculatorMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*CalculatorMenu).Run() before (*CalculatorMenu).Configure()!")
	}

	instance.parent.CreateOrLoadPersist("Calc_ExchangeRate", 1.0)

	if instance.process_selection {
		instance.process_selection = false

		// Do nothing if the selection path is empty (i.e. user cancelled the selector)
		if len(instance.selection_path) > 0 {

			switch instance.selection_path[0] {
			case "Equals":
				res, err := instance.evaluate()
				if err != nil {
					instance.calc_displayed = "Error"
					instance.calc_input = []rune{}
				} else {
					s := strconv.FormatFloat(res, 'f', -1, 64)
					instance.calc_input = []rune(s)
					instance.compute_displayed()
				}

			case "Clear":
				instance.calc_input = []rune{}
				instance.calc_displayed = ""

			case "Exchange rate":
				// Check if the user selected a suboption
				if len(instance.selection_path) > 1 {
					val, err := instance.evaluate()
					if err == nil && val != 0 {
						switch instance.selection_path[1] {
						case "Foreign as domestic":
							instance.parent.RenderAlert("ok", []string{"Rate", "saved"})
							instance.parent.Set("Calc_ExchangeRate", val)
							go instance.parent.SyncPersistent()
							go instance.parent.PlayKey()
							time.Sleep(time.Second)

						case "Domestic as foreign":
							instance.parent.RenderAlert("ok", []string{"Rate", "saved"})
							instance.parent.Set("Calc_ExchangeRate", 1.0/val)
							go instance.parent.SyncPersistent()
							go instance.parent.PlayKey()
							time.Sleep(time.Second)
						}
					}
				}

			case "To domestic":
				val, err := instance.evaluate()
				rate, ok := instance.parent.Get("Calc_ExchangeRate").(float64)
				if ok && err == nil {
					res := val * rate
					s := strconv.FormatFloat(res, 'f', -1, 64)
					instance.calc_input = []rune(s)
					instance.compute_displayed()
				}

			case "To foreign":
				val, err := instance.evaluate()
				rate, ok := instance.parent.Get("Calc_ExchangeRate").(float64)
				if ok && err == nil && rate != 0 {
					res := val / rate
					s := strconv.FormatFloat(res, 'f', -1, 64)
					instance.calc_input = []rune(s)
					instance.compute_displayed()
				}
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
					allowDecimal := true
					for i := len(instance.calc_input) - 1; i >= 0; i-- {
						c := instance.calc_input[i]
						if c == '.' {
							allowDecimal = false
							break
						}
						if c == '+' || c == '-' || c == '*' || c == '/' {
							break
						}
					}
					if allowDecimal {
						instance.calc_input = append(instance.calc_input, '.')
						instance.compute_displayed()
						instance.render()
					}

				case 'S':
					go instance.parent.PushWithArgs("selector", &SelectorArgs{
						Title:          "Calculator",
						SelectionClass: "calculator.main",
						Options: [][]string{
							{"Equals"},
							{"Clear"},
							{"Exchange rate",
								"Foreign as domestic",
								"Domestic as foreign",
							},
							{"To domestic"},
							{"To foreign"},
						},
						ButtonLabel:           "Select",
						VisibleRows:           3,
						ShowPathInTitle:       true,
						ShowElemNumberInTitle: true,
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
