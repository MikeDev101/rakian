package simulator

import (
	"context"
	"gfx"
	"image"
	"image/color"
	"keypad"
	"modem"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/fogleman/gg"
)

type Simulator struct {
	ctx            *gg.Context
	width          int
	height         int
	spriteCache    map[string]*image.Image
	fontCache      map[string]map[rune]*image.Image
	animationCache map[string]*gfx.Animation
	isOn           bool
	renderLock     sync.Mutex
	drawLock       sync.Mutex

	app       fyne.App
	window    fyne.Window
	canvasImg *canvas.Image
	mockModem modem.Modem
	keyEvents chan *keypad.KeypadEvent
}

// Ensure interfaces are implemented
// var _ modem.Modem = (*Simulator)(nil)
var _ gfx.Driver = (*Simulator)(nil)
var _ keypad.Keypad = (*Simulator)(nil)

func New() *Simulator {
	v := &Simulator{
		width:          84,
		height:         48,
		ctx:            gg.NewContext(84, 48),
		spriteCache:    make(map[string]*image.Image),
		fontCache:      make(map[string]map[rune]*image.Image),
		animationCache: make(map[string]*gfx.Animation),
		isOn:           true,
		keyEvents:      make(chan *keypad.KeypadEvent, 100),
	}

	// Init mock modem
	v.mockModem = &MockModem{
		ok:               true,
		signalStrength:   4,
		flightMode:       false,
		carrier:          "Debugger",
		roaming:          false,
		sos:              false,
		registered:       true,
		emergencyNumbers: []string{"911", "112"},
		unreadVoicemails: 0,
		ringingChan:      make(chan *modem.Call, 10),
		Calls:            make(map[string]*modem.Call),
	}

	// Initialize graphics
	v.ctx.SetColor(Primary)
	v.ctx.Clear()

	// Initialize Fyne
	v.app = app.New()
	v.window = v.app.NewWindow("Rakian")

	mainMenu := fyne.NewMainMenu(
		fyne.NewMenu("File",
			fyne.NewMenuItem("Quit", func() { v.app.Quit() }),
		),
		fyne.NewMenu(
			"Mock Modem",
			fyne.NewMenuItem("Toggle OK", func() {
				v.mockModem.(*MockModem).ok = !v.mockModem.(*MockModem).ok
			}),
			fyne.NewMenuItem("Toggle Registered", func() {
				v.mockModem.(*MockModem).registered = !v.mockModem.(*MockModem).registered
			}),
			fyne.NewMenuItem("Toggle Flight Mode", func() {
				v.mockModem.(*MockModem).flightMode = !v.mockModem.(*MockModem).flightMode
			}),
			fyne.NewMenuItem("Toggle SOS", func() {
				v.mockModem.(*MockModem).sos = !v.mockModem.(*MockModem).sos
			}),
			fyne.NewMenuItem("Set Signal Strength", func() {
				// TODO: Prompt for a number

			}),
			fyne.NewMenuItem("Set Carrier", func() {
				// TODO: Prompt for a string

			}),
			fyne.NewMenuItem("Toggle Roaming", func() {
				v.mockModem.(*MockModem).roaming = !v.mockModem.(*MockModem).roaming
			}),
		),
	)
	v.window.SetMainMenu(mainMenu)

	// Create a mutable image for Fyne
	v.canvasImg = canvas.NewImageFromImage(v.ctx.Image())
	v.canvasImg.ScaleMode = canvas.ImageScalePixels
	v.canvasImg.FillMode = canvas.ImageFillContain

	// Display Area with white border
	border := canvas.NewRectangle(color.White)
	// aspect := float32(v.width) / float32(v.height)
	displayContainer := container.New(layout.NewCustomPaddedLayout(
		25,
		25,
		15,
		15,
	),
		container.NewStack(
			border,
			container.NewPadded(v.canvasImg),
		),
	)

	// Keypad Area
	createKey := func(label string, r rune) *widget.Button {
		return widget.NewButton(label, func() {
			v.keyEvents <- &keypad.KeypadEvent{State: true, Key: r}
			go func() {
				time.Sleep(100 * time.Millisecond)
				v.keyEvents <- &keypad.KeypadEvent{State: false, Key: r}
			}()
		})
	}

	keypadContainer := container.NewGridWithColumns(3,
		createKey("C", 'C'), createKey("S", 'S'), createKey("U", 'U'),
		createKey("P", 'P'), layout.NewSpacer(), createKey("D", 'D'),
		createKey("1", '1'), createKey("2", '2'), createKey("3", '3'),
		createKey("4", '4'), createKey("5", '5'), createKey("6", '6'),
		createKey("7", '7'), createKey("8", '8'), createKey("9", '9'),
		createKey("*", '*'), createKey("0", '0'), createKey("#", '#'),
	)

	content := container.NewBorder(
		nil, keypadContainer, nil, nil,
		displayContainer,
	)

	// Setup Input
	v.setupInput()

	v.window.SetContent(content)
	v.window.Resize(fyne.NewSize(84*4, 48*5+300))

	return v
}

func (v *Simulator) Run(ctx context.Context, debug bool) <-chan *keypad.KeypadEvent {
	return v.keyEvents
}

func (v *Simulator) Start() {
	v.window.ShowAndRun()
}

func (v *Simulator) Stop() {
	v.app.Quit()
}
