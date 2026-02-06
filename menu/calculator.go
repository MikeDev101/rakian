package menu

import (
	"time"
	"context"
	"sync"
	"log"
	"strings"
	
	"misc"
	"sh1107"
)

type CalculatorMenu struct {
	ctx              context.Context
	configured       bool
	cancelFn         context.CancelFunc
	parent           *Menu
	wg               sync.WaitGroup
	calc_input       []rune
	calc_displayed   string
	lastAsteriskTime time.Time
}

func (m *Menu) NewCalculatorMenu() *CalculatorMenu {
	return &CalculatorMenu{
		parent:           m,
		lastAsteriskTime: time.Now(),
	}
}

func (instance *CalculatorMenu) render() {
	display := instance.parent.Display
	display.Clear(sh1107.Black)
	
	font := display.Use_Font8_Normal()
	display.DrawTextAligned(0, 20, font, "Calculator", false, sh1107.AlignRight, sh1107.AlignNone)
	
	display.SetColor(sh1107.White)
	display.SetLineWidth(1)
	display.DrawLine(0, 33, 128, 33)
	display.Stroke()
	
	font = display.Use_Font8_Bold()
	display.DrawTextAligned(64, 105, font, "Options", false, sh1107.AlignCenter, sh1107.AlignNone)
	
	display.DrawText(0, 40, display.Use_Font16(), instance.calc_displayed, false)
	
	instance.parent.Display.Render()
}

func (instance *CalculatorMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
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
				
				switch evt.Key {
					case '*':
						go instance.parent.PlayKey()
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
					go instance.parent.PlayKey()
					
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
					go instance.parent.PlayKey()
					go instance.parent.Push("power")
					return
				
				case '#':
					instance.calc_input = append(instance.calc_input, '.')
					instance.compute_displayed()
					instance.render()
					go instance.parent.PlayKey()
				
				case 'U':
				case 'D':
				case 'S':
					/*
					Options
					
					1. Equals
					2. Add
					3. Subract
					4. Multiply
					5. Divide
					6. To domestic
					7. To foreign
					8. Exchange rate
						1. Foreign unit expressed as domestic units
						2. Domestic unit expressed as foreign units
					
					*/
				
				default:
					instance.calc_input = append(instance.calc_input, evt.Key)
					instance.compute_displayed()
					instance.render()
					go instance.parent.PlayKey()
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
}