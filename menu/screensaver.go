package menu

import (
	"time"
	"context"
	"sync"
	"log"
	
	"sh1107"
	"timers"
	"misc"
)

type Screensaver struct {
	ctx        context.Context
	configured bool
	running    bool
	cancelFn   context.CancelFunc
	parent     *Menu
	wg         sync.WaitGroup
	dx, dy     int
	x, y       int
	flip       int
}

func (m *Menu) NewScreensaver() *Screensaver {
	return &Screensaver{
		parent: m,
		dx: 1,
		dy: 1,
		x: 0,
		y: 20,
		flip: sh1107.Normal,
	}
}

func (instance *Screensaver) render() {
	display := instance.parent.Display
	duck := instance.parent.Sprites["duck"]
	
	switch {
	case instance.dx < 0 && instance.dy < 0:
		instance.flip = sh1107.UpsideDown
	case instance.dx > 0 && instance.dy < 0:
		instance.flip = sh1107.FlippedUpsideDown
	case instance.dx < 0:
		instance.flip = sh1107.Normal
	default:
		instance.flip = sh1107.Flipped
	}
	
	display.Clear(sh1107.Black)
	display.DrawImage(
		sh1107.FlipImage(duck, instance.flip),
		instance.x, instance.y,
	)
	display.Render()

	// Bounce logic
	instance.x += instance.dx
	instance.y += instance.dy
	if instance.x <= 0 || instance.x + duck.Bounds().Max.X >= display.Width {
		instance.dx = -instance.dx
	}
	if instance.y <= 20 || instance.y + duck.Bounds().Max.Y >= display.Height - 5 {
		instance.dy = -instance.dy
	}
}

func (instance *Screensaver) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *Screensaver) Run() {
	if !instance.configured {
		panic("Attempted to call (*Screensaver).Run() before (*Screensaver).Configure()!")
	}
	
	if instance.running {
		panic("Attempted to run multiple entries of (*Screensaver).Run()")
	}
	instance.running = true
	
	instance.render()
	instance.parent.Display.SetBrightness(0.0)
	instance.parent.Timers["oled"].Stop()
	
	instance.wg.Add(1)
	go func() {
		defer instance.wg.Done()
		for {
			timers.SleepWithContext(time.Millisecond, instance.ctx)
			select {
			case <-instance.ctx.Done():
				return
			default:
				instance.render()
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
				
			case evt := <-instance.parent.KeypadEvents:				
				if evt.State {
					
					instance.parent.Timers["keypad"].Restart()
					instance.parent.Timers["oled"].Restart()
					instance.parent.Display.On()
					misc.KeyLightsOn()
					go instance.parent.PlayKey()
					go instance.parent.Pop()
					return
				}
			}
		}
	}()
}

func (instance *Screensaver) Pause() {
	instance.cancelFn()
	instance.parent.Display.SetBrightness(1.0)
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Screensaver pause timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		instance.running = false
	}
}

func (instance *Screensaver) Stop() {
	instance.cancelFn()
	instance.parent.Display.SetBrightness(1.0)
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Screensaver stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		instance.running = false
	}
}