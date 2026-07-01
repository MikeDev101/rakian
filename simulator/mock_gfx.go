package simulator

import (
	"context"
	"gfx"
	"image"
	"image/color"

	"fyne.io/fyne/v2"
	"github.com/fogleman/gg"
)

var Primary color.Color = color.White
var Secondary color.Color = color.Black

func (v *Simulator) Display() gfx.Driver {
	var display gfx.Driver = v
	return display
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
func (v *Simulator) GetImageBounds(i *image.Image) (int, int) {
	return gfx.GetImageBounds(*i)
}
