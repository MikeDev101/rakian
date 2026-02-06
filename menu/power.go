package menu

import (
	"time"
	"context"
	"sync"
	"log"
	"fmt"
	
	"sh1107"
	"misc"
)

type PowerMenu struct {
	ctx        context.Context
	configured bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
	selection  uint8
	viewOffset uint8
	options    []string
}

func (m *Menu) NewPowerMenu() *PowerMenu {
	return &PowerMenu{
		parent: m,
		selection: 0,
		options: []string{
			"Switch off!",
			"Loud",
			"Discreet",
			"Vibrate",
			"Silent",
			"Reboot device",
			"Restart Rakian",
			"Toggle WiFi",
			"Toggle Bluetooth",
			"Flight Mode",
		},
	}
}

func (instance *PowerMenu) render() {
	display := instance.parent.Display
	
	display.Clear(sh1107.Black)
	
	
	const visibleRows = 4
	
	// Determine starting item index based on selection
	if instance.selection < instance.viewOffset {
		instance.viewOffset = instance.selection
	} else if instance.selection >= instance.viewOffset+visibleRows {
		instance.viewOffset = instance.selection - visibleRows + 1
	}
	
	start := int(instance.viewOffset)
	end := start + visibleRows
	if int(end) > len(instance.options) {
		end = len(instance.options)
	}
	
	font := display.Use_Font8_Normal()
	display.DrawText(0, 20, font, "Power", false)
	
	font = display.Use_Font8_Bold()
	display.DrawTextAligned(128, 20, font, fmt.Sprintf("%d %%", instance.parent.Get("BatteryPercent").(int)), false, sh1107.AlignLeft, sh1107.AlignNone)
	
	display.SetColor(sh1107.White)
	display.SetLineWidth(1)
	display.DrawLine(0, 33, 128, 33)
	display.Stroke()
	
	font = display.Use_Font8_Bold()
	for i, opt := range instance.options[start:end] {
		y := 40 + i*20 // Adjust for font height and spacing
		if start+i == int(instance.selection) {
			// Draw selection highlight box
			display.SetColor(sh1107.White)
			display.DrawRectangle(0, float64(y-1), 128, 16)
			display.Fill()
			display.DrawText(2, y+4, font, opt, true)
		} else {
			display.DrawText(2, y+4, font, opt, false)
		}
	}
	
	display.Render()
}

func (instance *PowerMenu) handle_selection() {
	switch instance.selection {
		case 0: // Turn off now
			go instance.parent.GlobalQuit(1) // Shutdown
			return
		
		case 1: // Loud mode
			instance.parent.Set("CanVibrate", true)
			instance.parent.Set("CanRing", true)
			instance.parent.Set("BeepOnly", false)
			instance.parent.RenderAlert("ok", []string{"Loud", "mode on"})
			
			go instance.parent.PlayAlert()
			time.Sleep(2 * time.Second)
			go instance.parent.Pop()
			return
		
		case 2: // Discreet mode
			instance.parent.Set("CanVibrate", true)
			instance.parent.Set("CanRing", true)
			instance.parent.Set("BeepOnly", true)
			instance.parent.RenderAlert("ok", []string{"Discreet", "mode on"})
			
			go instance.parent.PlayAlert()
			time.Sleep(2 * time.Second)
			go instance.parent.Pop()
			return
			
		case 3: // Vibrate mode
			instance.parent.Set("CanVibrate", true)
			instance.parent.Set("CanRing", false)
			instance.parent.Set("BeepOnly", false)
			instance.parent.RenderAlert("ok", []string{"Vibrate only", "mode on"})
			
			go func() {
				for range(3) {
					instance.parent.Player.StartVibrate()
					time.Sleep(500 * time.Millisecond)
					instance.parent.Player.StopVibrate()
					time.Sleep(100 * time.Millisecond)
				}
			} ()
			
			time.Sleep(2 * time.Second)
			go instance.parent.Pop()
			return
			
		case 4: // Silent mode
			instance.parent.Set("CanVibrate", false)
			instance.parent.Set("CanRing", false)
			instance.parent.Set("BeepOnly", false)
			instance.parent.RenderAlert("ok", []string{"Silent", "mode on"})
			
			time.Sleep(2 * time.Second)
			go instance.parent.Pop()
			return
			
		case 5: // Hard reboot
			go instance.parent.GlobalQuit(2) // Hard reboot
			return
		
		case 6: // Soft reboot
			go instance.parent.GlobalQuit(3) // Soft reboot
			return
			
		case 9: // Flight mode
			var modem = instance.parent.Modem
			if modem == nil {
				instance.parent.RenderAlert("prohibited", []string{"Modem", "error!"})
				go instance.parent.PlayAlert()
				time.Sleep(2 * time.Second)
				go instance.parent.Pop()
				return
			}
		
			var msg = []string{"flight", "mode"}
			
			if instance.parent.Modem.FlightMode {
				msg = append([]string{"Leaving"}, msg...)
			} else {
				msg = append([]string{"Entering"}, msg...)
			}
			
			go modem.ToggleFlightMode()
			instance.parent.RenderAlert("ok", msg)
			
			go instance.parent.PlayAlert()
			time.Sleep(2 * time.Second)
			go instance.parent.Pop()
			return
	}
	
	// Generic handler
	go instance.parent.PlayKey()
	go instance.parent.Pop()
}

func (instance *PowerMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *PowerMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*PowerMenu).Run() before (*PowerMenu).Configure()!")
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
					
					switch evt.Key {
					case 'U':
						if instance.selection > 0 {
							instance.selection -= 1
							instance.render()
						}
					case 'D':
						if instance.selection < uint8(len(instance.options)-1) {
							instance.selection += 1
							instance.render()
						}
					case 'S':
						go instance.handle_selection()
						return
					case 'C':
						go instance.parent.PlayKey()
						go instance.parent.Pop()
						return
					case 'P':
						go instance.parent.PlayKey()
						go instance.parent.Pop()
						return
					}
					
					go instance.parent.PlayKey()
				}
			}
		}
	}()
}

func (instance *PowerMenu) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Power menu pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *PowerMenu) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Power menu stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		go instance.cleanup()
	}
}

func (instance *PowerMenu) cleanup() {
	instance.selection = 0
}