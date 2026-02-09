package menu

import (
	"context"
	"fmt"
	"log"
	"misc"
	"sh1107"
	"sync"
	"time"
)

type SettingsMenu struct {
	ctx               context.Context
	configured        bool
	cancelFn          context.CancelFunc
	parent            *Menu
	wg                sync.WaitGroup
	process_selection bool
	selection_path    []string
	options           [][]string
}

func (instance *SettingsMenu) RenderAbout() {
	m := instance.parent
	display := m.Display

	display.Clear(sh1107.Black)

	font := display.Use_Font8_Bold()
	display.DrawTextAligned(0, 20, font, "About", false, sh1107.AlignRight, sh1107.AlignNone)
	display.DrawTextAligned(64, 110, font, "Check for updates", false, sh1107.AlignCenter, sh1107.AlignNone)

	display.DrawImageAligned(m.Sprites["logo"], 60, 50, sh1107.AlignCenter, sh1107.AlignCenter)

	font = display.Use_Font8_Normal()
	display.DrawTextAligned(60, 60, font, "Rakian OS", false, sh1107.AlignCenter, sh1107.AlignNone)
	display.DrawTextAligned(60, 70, font, fmt.Sprintf("v%s", m.Get("FirmwareVersion").(string)), false, sh1107.AlignCenter, sh1107.AlignNone)

	display.SetColor(sh1107.White)
	display.SetLineWidth(1)
	display.DrawLine(0, 33, 127, 33)
	display.Stroke()

	display.Render()
}

func (m *Menu) NewSettingsMenu() *SettingsMenu {
	return &SettingsMenu{
		parent:            m,
		process_selection: false,
		selection_path:    []string{},
		options: [][]string{
			{"WiFi Settings",
				"Toggle WiFi",
				"Current status",
				"Join network",
				"Saved networks",
			},
			{"Bluetooth Settings",
				"Toggle Bluetooth",
				"Connected devices",
				"Pair new device",
				"Paired devices",
			},
			{"Mobile Data Settings",
				"Toggle Mobile Data",
				"Connection Status",
				"Select Network Mode",
				"Choose APN",
			},
			{"Call Settings",
				"Automatic Redial",
				"Speed Redialing",
				"Call Waiting Options",
				"Own Number Sending",
				"Phone Line In Use",
				"Automatic Answer",
			},
			{"Phone Settings",
				"Language",
				"Cell Info Display",
				"Welcome Note",
				"Network Selection",
				"Lights",
			},
			{"Security Settings",
				"PIN code request",
				"Call barring service",
				"Fixed dialing",
				"Closed user group",
				"Phone security",
				"Change access codes",
			},
			{"SSH Service"},
			{"About"},
			{"Factory Reset"},
		},
	}
}

func (instance *SettingsMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *SettingsMenu) ConfigureWithArgs(args ...any) {

	// Check if we have args
	if len(args) > 0 {

		// Most likely our arg is a SelectorReturn from the selector.
		selection, ok := args[0].(*SelectorReturn)
		if !ok {
			panic("(*SettingsMenu).ConfigureWithArgs() Type error: argument must be a *SelectorReturn type")
		}

		instance.process_selection = true
		instance.selection_path = selection.SelectionPath
	}

	instance.Configure()
}

func (instance *SettingsMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*SettingsMenu).Run() before (*SettingsMenu).Configure()!")
	}

	log.Println("⚙️ Settings started")

	if instance.process_selection {
		instance.process_selection = false
		log.Println("⚙️ Settings path selected: ", instance.selection_path)

		if len(instance.selection_path) == 0 {
			log.Println("⚙️ Settings path selected is empty, exiting...")
			go instance.parent.Pop()
			return
		}

		// TODO: process selected setting option

		switch instance.selection_path[0] {

		case "About":
			instance.RenderAbout()
		about:
			for {
				select {
				case <-instance.ctx.Done():
					break about
				case evt := <-instance.parent.KeypadEvents:

					if evt.State {
						instance.parent.Timers["keypad"].Reset()
						instance.parent.Timers["oled"].Reset()
						instance.parent.Display.On()
						misc.KeyLightsOn()
						go instance.parent.PlayKey()
					}

					switch evt.Key {
					case 'S':
						// TODO: check for updates
					case 'C':
						break about
					}
				}
			}
		}

		log.Println("⚙️ Settings switching back to selector")
		go instance.parent.PushWithArgs("selector", &SelectorArgs{
			Title:                 "Settings",
			Options:               instance.options,
			ButtonLabel:           "Select",
			VisibleRows:           3,
			ShowPathInTitle:       true,
			ShowElemNumberInTitle: true,
		})
		return

	} else {
		// Start the selector with the base settings menu
		log.Println("⚙️ Settings switching to selector")
		go instance.parent.PushWithArgs("selector", &SelectorArgs{
			Title:                 "Settings",
			Options:               instance.options,
			ButtonLabel:           "Select",
			VisibleRows:           3,
			ShowPathInTitle:       true,
			ShowElemNumberInTitle: true,
		})
	}

	instance.wg.Go(func() {
		<-instance.ctx.Done()
		log.Println("⚙️ Settings exiting...")
	})
}

func (instance *SettingsMenu) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Settings handler pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *SettingsMenu) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Settings handler stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		go instance.cleanup()
	}
}

func (instance *SettingsMenu) cleanup() {
	instance.process_selection = false
	instance.selection_path = []string{}
}
