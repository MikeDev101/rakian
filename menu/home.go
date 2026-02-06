package menu

import (
	"time"
	"context"
	"sync"
	"log"
	"fmt"
	
	"timers"
	"misc"
	"sh1107"
)

type HomeMenu struct {
	ctx         context.Context
	configured  bool
	cancelFn    context.CancelFunc
	parent      *Menu
	wg          sync.WaitGroup
	batt_flash  bool
	render_loop *timers.ResettableTimer
}

func (m *Menu) NewHomeMenu() *HomeMenu {
	return &HomeMenu{
		parent: m,
	}
}

func (instance *HomeMenu) render() {
	display := instance.parent.Display
	display.Clear(sh1107.Black)
	instance.parent.RenderStatusBar(&instance.batt_flash)
	
	font := display.Use_Font8_Normal()
	
	now := time.Now().In(time.Local)
	hour := now.Hour() % 12
	if hour == 0 {
		hour = 12
	}
	
	display.DrawTextAligned(102, 21, font, fmt.Sprintf("%02d:%02d", hour, now.Minute()), false, sh1107.AlignLeft, sh1107.AlignNone)
	
	font = display.Use_Font8_Bold()
	display.DrawTextAligned(64, 105, font, "Menu", false, sh1107.AlignCenter, sh1107.AlignNone)
	
	var carrier_label string
	if instance.parent.Modem == nil {
		carrier_label = "Modem Error"
		
	} else if instance.parent.Modem.FlightMode {
		carrier_label = "Flight Mode"
		
	} else {
		carrier_label = instance.parent.Modem.Carrier + " (" + instance.parent.Modem.NetworkGeneration + ")"
	}
	
	display.DrawText(
		0,
		40,
		font,
		carrier_label,
		false,
	)
	
	if instance.parent.GlobalStorage["WiFi_Connected"].(bool) {
		font = display.Use_Font8_Normal()
		display.DrawText(0, 60, font, instance.parent.GlobalStorage["WiFi_IP"].(string), false)
		display.DrawText(0, 50, font, instance.parent.GlobalStorage["WiFi_SSID"].(string), false)
	}
	
	display.Render()
}

func (instance *HomeMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *HomeMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*HomeMenu).Run() before (*HomeMenu).Configure()!")
	}
		
	instance.wg.Add(1)
	go func() {
		defer instance.wg.Done()
		instance.render()
		for {
			select {
			case <-instance.ctx.Done():
				return
			case <-time.After(time.Second):
				if instance.parent.Display.IsOn {
					instance.render()
				}
			}			
		}
	}()
	
	instance.wg.Add(1)
	go func() {
		defer instance.wg.Done()
		for {
			select {
			case <-instance.ctx.Done():
				return
			
			case evt, ok := <-instance.parent.KeypadEvents:
				if !ok {
					return
				}
			
				if evt.State {
					
					instance.parent.Timers["keypad"].Reset()
					instance.parent.Timers["oled"].Reset()
					instance.parent.Display.On()
					misc.KeyLightsOn()
					go instance.parent.PlayKey()
					
					switch evt.Key {
						
					case 'P':
						go instance.parent.Push("power")
						return
						
					case 'S':
						go instance.parent.Push("home_selection")
						return
					case 'U':
					case 'D':
					case 'C':
					default:
						instance.parent.GlobalStorage["InitialKey"] = evt.Key
						go instance.parent.Push("dialer")
						return
					}
				}
			}
		}
	}()
}

func (instance *HomeMenu) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Home menu pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *HomeMenu) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Home menu stop timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}