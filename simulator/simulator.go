package simulator

import (
	"context"
	"gfx"
	"image"
	"image/color"
	"keypad"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/fogleman/gg"
)

var Primary color.Color = color.White
var Secondary color.Color = color.Black

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
			"Options",
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

	v.window.SetContent(content)
	v.window.Resize(fyne.NewSize(84*4, 48*5+300))

	// Setup Input
	v.setupInput()

	return v
}

func (v *Simulator) setupInput() {
	if desk, ok := v.window.Canvas().(desktop.Canvas); ok {
		desk.SetOnKeyDown(func(ev *fyne.KeyEvent) {
			k := mapKey(ev.Name)
			if k != 0 {
				v.keyEvents <- &keypad.KeypadEvent{
					State: true,
					Key:   k,
				}
			}
		})
		desk.SetOnKeyUp(func(ev *fyne.KeyEvent) {
			k := mapKey(ev.Name)
			if k != 0 {
				v.keyEvents <- &keypad.KeypadEvent{
					State: false,
					Key:   k,
				}
			}
		})
	}
}

func mapKey(name fyne.KeyName) rune {
	switch name {
	case fyne.Key1:
		return '1'
	case fyne.Key2:
		return '2'
	case fyne.Key3:
		return '3'
	case fyne.Key4:
		return '4'
	case fyne.Key5:
		return '5'
	case fyne.Key6:
		return '6'
	case fyne.Key7:
		return '7'
	case fyne.Key8:
		return '8'
	case fyne.Key9:
		return '9'
	case fyne.Key0:
		return '0'
	case fyne.KeyAsterisk:
		return '*'
	case fyne.KeyReturn, fyne.KeyEnter:
		return 'S'
	case fyne.KeyBackspace, fyne.KeyDelete, fyne.KeyC:
		return 'C'
	case fyne.KeyUp:
		return 'U'
	case fyne.KeyDown:
		return 'D'
	case fyne.KeyS:
		return 'S'
	case fyne.KeyD:
		return 'D'
	case fyne.KeyU:
		return 'U'
	case fyne.KeyP:
		return 'P'
	}
	return 0
}

func (v *Simulator) Display() gfx.Driver {
	var display gfx.Driver = v
	return display
}

func (v *Simulator) Keypad() keypad.Keypad {
	var kp keypad.Keypad = v
	return kp
}

// gfx.Driver implementation
func (v *Simulator) On()                    { v.isOn = true }
func (v *Simulator) Off()                   { v.isOn = false }
func (v *Simulator) IsOn() bool             { return v.isOn }
func (v *Simulator) Width() int             { return v.width }
func (v *Simulator) Height() int            { return v.height }
func (v *Simulator) Context() *gg.Context   { return v.ctx }
func (v *Simulator) Primary() color.Color   { return Primary }
func (v *Simulator) Secondary() color.Color { return Secondary }
func (v *Simulator) Lock()                  { v.drawLock.Lock() }
func (v *Simulator) Unlock()                { v.drawLock.Unlock() }
func (v *Simulator) TryLock() bool          { return v.drawLock.TryLock() }
func (v *Simulator) SetColor(c color.Color) { v.ctx.SetColor(c) }
func (v *Simulator) SetLineWidth(w float64) { v.ctx.SetLineWidth(w) }
func (v *Simulator) Stroke()                { v.ctx.Stroke() }
func (v *Simulator) Fill()                  { v.ctx.Fill() }
func (v *Simulator) Load_Sprites()          { gfx.Load_Sprites(v) }
func (v *Simulator) Load_Animations()       { gfx.LoadAnimations(v) }
func (v *Simulator) Load_Font_Tiny()        { gfx.Load_Font_Tiny(v) }
func (v *Simulator) Load_Font_Small_Bold()  { gfx.Load_Font_Small_Bold(v) }
func (v *Simulator) Load_Font_Small_Plain() { gfx.Load_Font_Small_Plain(v) }
func (v *Simulator) Load_Font_Large_Bold()  { gfx.Load_Font_Large_Bold(v) }
func (v *Simulator) DrawImage(img *image.Image, x, y int) {
	v.ctx.DrawImage(*img, x, y)
}
func (v *Simulator) DrawLine(x1, y1, x2, y2 float64) { v.ctx.DrawLine(x1, y1, x2, y2) }
func (v *Simulator) DrawRectangle(x, y, w, h float64) {
	v.ctx.DrawRectangle(x, y, w, h)
}

func (v *Simulator) Render() {
	fyne.Do(func() {
		v.renderLock.Lock()
		defer v.renderLock.Unlock()
		v.canvasImg.Refresh()
	})
}

func (v *Simulator) Clear(c color.Color) {
	v.ctx.SetColor(c)
	v.ctx.Clear()
}

// Caches
func (v *Simulator) GetAnimationCache(n string) *gfx.Animation      { return v.animationCache[n] }
func (v *Simulator) SetAnimationCache(n string, a *gfx.Animation)   { v.animationCache[n] = a }
func (v *Simulator) GetSpriteCache(n string) *image.Image           { return v.spriteCache[n] }
func (v *Simulator) SetSpriteCache(n string, i *image.Image)        { v.spriteCache[n] = i }
func (v *Simulator) GetSpriteCacheAll() map[string]*image.Image     { return v.spriteCache }
func (v *Simulator) GetFontCache(n string) map[rune]*image.Image    { return v.fontCache[n] }
func (v *Simulator) SetFontCache(n string, f map[rune]*image.Image) { v.fontCache[n] = f }
func (v *Simulator) Use_Font_Small_Bold() map[rune]*image.Image     { return gfx.Use_Font_Small_Bold(v) }
func (v *Simulator) Use_Font_Small_Plain() map[rune]*image.Image    { return gfx.Use_Font_Small_Plain(v) }
func (v *Simulator) Use_Font_Large_Bold() map[rune]*image.Image     { return gfx.Use_Font_Large_Bold(v) }
func (v *Simulator) Use_Font_Tiny() map[rune]*image.Image           { return gfx.Use_Font_Tiny(v) }
func (v *Simulator) Use_Sprites() map[string]*image.Image           { return gfx.Use_Sprites(v) }

// Graphics primitives
func (v *Simulator) FlipImage(i *image.Image, m gfx.RenderType) *image.Image {
	return gfx.FlipImage(i, m)
}
func (v *Simulator) InvertImage(i *image.Image) *image.Image { return gfx.InvertImage(i) }
func (v *Simulator) PlayAnimation(c context.Context, n string, x, y int, ax, ay gfx.RenderType) {
	gfx.PlayAnimation(v, c, n, x, y, ax, ay)
}
func (v *Simulator) DrawText(x, y int, f map[rune]*image.Image, s string, i bool) {
	gfx.DrawText(v, x, y, f, s, i)
}
func (v *Simulator) DrawTextAligned(x, y int, f map[rune]*image.Image, s string, i bool, ha, va gfx.RenderType) {
	gfx.DrawTextAligned(v, x, y, f, s, i, ha, va)
}
func (v *Simulator) DrawTextWrapped(x1, y1, x2, y2 int, f map[rune]*image.Image, s string, i bool, ha, va gfx.RenderType) {
	gfx.DrawTextWrapped(v, x1, y1, x2, y2, f, s, i, ha, va)
}
func (v *Simulator) DrawImageAligned(i *image.Image, x, y int, ha, va gfx.RenderType) {
	gfx.DrawImageAligned(v, *i, x, y, ha, va)
}
func (v *Simulator) GetTextBounds(f map[rune]*image.Image, s string) (int, int) {
	return gfx.GetTextBounds(f, s)
}

// Keypad implementation
func (v *Simulator) Run(ctx context.Context, debug bool) <-chan *keypad.KeypadEvent {
	return v.keyEvents
}

func (v *Simulator) KeyLightsOff() {}
func (v *Simulator) KeyLightsOn()  {}

func (v *Simulator) Start() {
	v.window.ShowAndRun()
}

func (v *Simulator) Stop() {
	v.app.Quit()
}

// Software modem implementation
