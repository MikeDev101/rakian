package lcd

import (
	"context"
	"gfx"
	"image"
	"image/color"
	"sync"
	"time"

	"github.com/fogleman/gg"

	"github.com/stianeikeland/go-rpio/v4"
)

var Black color.Color = color.Gray{Y: 0}
var White color.Color = color.Gray{Y: 255}

// Registers
const (
	POWERDOWN           = 0x04
	ENTRYMODE           = 0x02
	EXTENDEDINSTRUCTION = 0x01
	FUNCTIONSET         = 0x20

	DISPLAYCONTROL  = 0x08
	DISPLAYBLANK    = 0x0
	DISPLAYNORMAL   = 0x04
	DISPLAYALLON    = 0x01
	DISPLAYINVERTED = 0x05

	SETYADDR = 0x40
	SETXADDR = 0x80
	SETTEMP  = 0x04
	SETBIAS  = 0x10
	SETVOP   = 0x80
)

type LCD struct {
	sprite_cache    map[string]*image.Image
	font_cache      map[string]map[rune]*image.Image
	animation_cache map[string]*gfx.Animation
	width           int
	height          int
	ctx             *gg.Context
	render_lock     *sync.Mutex
	draw_lock       *sync.Mutex

	// Hardware-specific values
	dc       rpio.Pin
	rst      rpio.Pin
	is_on    bool
	spi_lock *sync.Mutex
}

// New() creates a new LCD implementation of gfx.Driver over SPI.
func New(dc, rst rpio.Pin) gfx.Driver {
	if err := rpio.SpiBegin(rpio.Spi0); err != nil {
		panic(err)
	}
	rpio.SpiSpeed(4000000)
	rpio.SpiChipSelect(0)

	display := &LCD{
		ctx:             gg.NewContextForImage(image.NewGray(image.Rect(0, 0, 84, 48))),
		animation_cache: make(map[string]*gfx.Animation),
		sprite_cache:    make(map[string]*image.Image),
		font_cache:      make(map[string]map[rune]*image.Image),
		dc:              dc,
		rst:             rst,
		width:           84,
		height:          48,
		is_on:           false,
		render_lock:     &sync.Mutex{},
		spi_lock:        &sync.Mutex{},
		draw_lock:       &sync.Mutex{},
	}

	dc.Output()
	rst.Output()

	display.init()
	display.Clear(White) // White is "off" (background) for this LCD
	return display
}

// Locks
func (d *LCD) Lock()         { d.draw_lock.Lock() }
func (d *LCD) Unlock()       { d.draw_lock.Unlock() }
func (d *LCD) TryLock() bool { return d.draw_lock.TryLock() }

// Setters
func (d *LCD) SetAnimationCache(name string, anim *gfx.Animation) { d.animation_cache[name] = anim }
func (d *LCD) SetSpriteCache(name string, img *image.Image)       { d.sprite_cache[name] = img }
func (d *LCD) SetFontCache(name string, f map[rune]*image.Image)  { d.font_cache[name] = f }

// Getters
func (*LCD) Primary() color.Color                             { return White }
func (*LCD) Secondary() color.Color                           { return Black }
func (d *LCD) GetAnimationCache(name string) *gfx.Animation   { return d.animation_cache[name] }
func (d *LCD) GetSpriteCacheAll() map[string]*image.Image     { return d.sprite_cache }
func (d *LCD) GetSpriteCache(name string) *image.Image        { return d.sprite_cache[name] }
func (d *LCD) GetFontCache(name string) map[rune]*image.Image { return d.font_cache[name] }
func (d *LCD) Context() *gg.Context                           { return d.ctx }
func (d *LCD) IsOn() bool                                     { return d.is_on }

// Compatibility loaders
func (d *LCD) Load_Font_Tiny()        { gfx.Load_Font_Tiny(d) }
func (d *LCD) Load_Font_Small_Bold()  { gfx.Load_Font_Small_Bold(d) }
func (d *LCD) Load_Font_Small_Plain() { gfx.Load_Font_Small_Plain(d) }
func (d *LCD) Load_Font_Large_Bold()  { gfx.Load_Font_Large_Bold(d) }
func (d *LCD) Load_Sprites()          { gfx.Load_Sprites(d) }
func (d *LCD) Load_Animations()       { gfx.LoadAnimations(d) }

// Compatibility readers
func (d *LCD) Use_Font_Small_Bold() map[rune]*image.Image  { return gfx.Use_Font_Small_Bold(d) }
func (d *LCD) Use_Font_Small_Plain() map[rune]*image.Image { return gfx.Use_Font_Small_Plain(d) }
func (d *LCD) Use_Font_Large_Bold() map[rune]*image.Image  { return gfx.Use_Font_Large_Bold(d) }
func (d *LCD) Use_Font_Tiny() map[rune]*image.Image        { return gfx.Use_Font_Tiny(d) }
func (d *LCD) Use_Sprites() map[string]*image.Image        { return gfx.Use_Sprites(d) }

// GG context
func (d *LCD) Stroke()                          { d.ctx.Stroke() }
func (d *LCD) Fill()                            { d.ctx.Fill() }
func (d *LCD) SetColor(c color.Color)           { d.ctx.SetColor(c) }
func (d *LCD) SetLineWidth(w float64)           { d.ctx.SetLineWidth(w) }
func (d *LCD) DrawLine(x1, y1, x2, y2 float64)  { d.ctx.DrawLine(x1, y1, x2, y2) }
func (d *LCD) DrawRectangle(x, y, w, h float64) { d.ctx.DrawRectangle(x, y, w, h) }

// Graphics
func (d *LCD) InvertImage(img *image.Image) *image.Image { return gfx.InvertImage(img) }
func (d *LCD) FlipImage(img *image.Image, mode gfx.RenderType) *image.Image {
	return gfx.FlipImage(img, mode)
}
func (d *LCD) DrawImage(img *image.Image, x, y int) { d.ctx.DrawImage(*img, x, y) }
func (d *LCD) PlayAnimation(ctx context.Context, name string, x, y int, align_x, align_y gfx.RenderType) {
	gfx.PlayAnimation(d, ctx, name, x, y, align_x, align_y)
}

// Text
func (d *LCD) DrawImageAligned(img *image.Image, x, y int, h_align, v_align gfx.RenderType) {
	gfx.DrawImageAligned(d, *img, x, y, h_align, v_align)
}
func (d *LCD) DrawText(x, y int, f map[rune]*image.Image, s string, invert bool) {
	gfx.DrawText(d, x, y, f, s, invert)
}
func (d *LCD) DrawTextAligned(x, y int, f map[rune]*image.Image, s string, invert bool, h_align, v_align gfx.RenderType) {
	gfx.DrawTextAligned(d, x, y, f, s, invert, h_align, v_align)
}
func (d *LCD) DrawTextWrapped(x1, y1, x2, y2 int, f map[rune]*image.Image, s string, invert bool, h_align, v_align gfx.RenderType) {
	gfx.DrawTextWrapped(d, x1, y1, x2, y2, f, s, invert, h_align, v_align)
}
func (*LCD) GetTextBounds(f map[rune]*image.Image, s string) (int, int) {
	return gfx.GetTextBounds(f, s)
}

// SPI functions
func (d *LCD) writeCommand(cmd byte) {
	d.spi_lock.Lock()
	defer d.spi_lock.Unlock()
	d.dc.Low() // Low for commands
	rpio.SpiTransmit(cmd)
}

func (d *LCD) writeData(data []byte) {
	d.spi_lock.Lock()
	defer d.spi_lock.Unlock()
	d.dc.High() // High for data
	rpio.SpiTransmit(data...)
}

func (d *LCD) init() {
	// Hardware reset
	d.rst.Low()
	time.Sleep(10 * time.Millisecond) // Give it enough time
	d.rst.High()
	time.Sleep(10 * time.Millisecond)

	d.writeCommand(FUNCTIONSET | EXTENDEDINSTRUCTION) // H = 1
	d.writeCommand(SETVOP | 0x3f)                     // Vop = 0x3f
	d.writeCommand(SETTEMP | 0x03)                    // Temp = 0x03
	d.writeCommand(SETBIAS | 0x03)                    // Bias = 0x03
	d.writeCommand(FUNCTIONSET)                       // H = 0
	d.writeCommand(DISPLAYCONTROL | DISPLAYNORMAL)    // Normal display

	d.is_on = true
}

func (d *LCD) On() {
	d.writeCommand(0x20)
	d.writeCommand(0x0C)
	d.is_on = true
}

func (d *LCD) Off() {
	d.writeCommand(0x20)
	d.writeCommand(0x08) // Display blank
	d.is_on = false
}

// Clears the gg.Context buffer and fills it with state color
func (d *LCD) Clear(state color.Color) {
	d.ctx.SetColor(state)
	d.ctx.DrawRectangle(0, 0, float64(d.width), float64(d.height))
	d.ctx.Fill()
}

// Sends the gg.Context buffer to the screen
func (d *LCD) Render() {
	d.render_lock.Lock()
	defer d.render_lock.Unlock()

	raw := d.to_bytes()

	d.writeCommand(FUNCTIONSET) // H = 0
	d.writeCommand(SETXADDR)    // Reset X to 0
	d.writeCommand(SETYADDR)    // Reset Y to 0

	d.writeData(raw)
}

// Converts the 2D image buffer into the 1D page-addressed format for LCD
func (d *LCD) to_bytes() []byte {
	bounds := d.ctx.Image().Bounds()
	raw := make([]byte, d.width*(d.height/8))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray := color.GrayModel.Convert(d.ctx.Image().At(x, y)).(color.Gray)
			page := y / 8
			offset := page*d.width + x
			bit := uint(y % 8)
			if gray.Y < 127 {
				raw[offset] |= (1 << bit)
			} else {
				raw[offset] &^= (1 << bit)
			}
		}
	}
	return raw
}
