package gfx

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"golang.org/x/image/bmp"
)

type RenderType int
type FrameType int

// Alignment Constants
const (
	AlignNone   RenderType = 0
	AlignLeft   RenderType = 1
	AlignRight  RenderType = 2
	AlignAbove  RenderType = 3
	AlignBelow  RenderType = 4
	AlignCenter RenderType = 5
)

const (
	Normal            RenderType = 0
	Flipped           RenderType = 1
	UpsideDown        RenderType = 2
	FlippedUpsideDown RenderType = 3
)

const (
	WrapDown  RenderType = 10
	WrapUp    RenderType = 11
	WrapLeft  RenderType = 12
	WrapRight RenderType = 13
)

const (
	PLAY_FRAME FrameType = 1 // Plays a frame with respect to either the FrameDelay or the DelayOverride.
	STOP_FRAME FrameType = 2 // Halts an animation and allows looping to occur if Loop is set to true.
)

type Frame struct {
	Type          FrameType
	Image         string
	DelayOverride time.Duration
}

type Animation struct {
	InitialFrame      *Frame
	InitialFrameDelay time.Duration
	FrameDelay        time.Duration
	LoopDelay         time.Duration
	Frames            []Frame
	Loop              bool
}

//go:embed sprites/*
var sprite_fs embed.FS

type Driver interface {
	// Primitives
	On()
	Off()
	Render()
	Clear(color.Color)
	Stop()
	Start(chan os.Signal, context.Context)

	// Acccessors
	Width() int
	Height() int
	Context() *gg.Context
	IsOn() bool
	GetAnimationCache(string) *Animation
	SetAnimationCache(string, *Animation)
	GetSpriteCache(string) *image.Image
	SetSpriteCache(string, *image.Image)
	GetSpriteCacheAll() map[string]*image.Image
	GetFontCache(string) map[rune]*image.Image
	SetFontCache(string, map[rune]*image.Image)
	Primary() color.Color
	Secondary() color.Color

	// Mutex
	TryLock() bool
	Lock()
	Unlock()

	// Misc
	FlipImage(*image.Image, RenderType) *image.Image
	InvertImage(*image.Image) *image.Image

	// Compatibility helpers
	Load_Sprites()
	Load_Animations()
	DrawImage(*image.Image, int, int)
	PlayAnimation(context.Context, string, int, int, RenderType, RenderType)
	DrawText(int, int, map[rune]*image.Image, string, bool)
	DrawTextAligned(int, int, map[rune]*image.Image, string, bool, RenderType, RenderType)
	DrawTextWrapped(int, int, int, int, map[rune]*image.Image, string, bool, RenderType, RenderType)
	DrawImageAligned(*image.Image, int, int, RenderType, RenderType)
	GetTextBounds(map[rune]*image.Image, string) (int, int)
	GetImageBounds(*image.Image) (int, int)
	Use_Font_Small_Bold() map[rune]*image.Image
	Use_Font_Small_Plain() map[rune]*image.Image
	Use_Font_Large_Bold() map[rune]*image.Image
	Use_Font_Tiny() map[rune]*image.Image
	Use_Sprites() map[string]*image.Image

	// Passthrough font loading
	Load_Font_Tiny()
	Load_Font_Small_Bold()
	Load_Font_Small_Plain()
	Load_Font_Large_Bold()

	// Passthrough primitives to gg
	SetColor(color.Color)
	SetLineWidth(float64)
	DrawLine(float64, float64, float64, float64)
	DrawRectangle(float64, float64, float64, float64)
	Stroke()
	Fill()
}

func Load_Font_Map(name string, mapping map[rune]string, mapfunc func(rune) (*image.Image, error), d Driver) {

	elem := d.GetFontCache(name)
	if elem == nil {
		elem = make(map[rune]*image.Image)
	}

	// Load all font runes, or load them from d.FontCache
	for char := range mapping {

		// Create prefix element if it doesn't exist
		if _, ok := elem[char]; ok {
			// Don't re-load the file if already loaded
			continue

		} else {
			// Load the rune image
			img, err := mapfunc(char)
			if err != nil {
				log.Printf("Font load failed for '%c': %v", char, err)
				continue
			}

			// Keep loaded in memory
			elem[char] = img
			d.SetFontCache(name, elem)
		}
	}
}

func Load_Sprite_Map(mapping map[string]string, mapfunc func(string) (*image.Image, error), d Driver) {

	// Load all sprites
	for elem := range mapping {

		// Create sprite element if it doesn't exist
		if d.GetSpriteCache(elem) == nil {

			// Load the image
			img, err := mapfunc(elem)
			if err != nil {
				log.Printf("Sprite load failed for '%s': %v", elem, err)
				continue
			}

			// Keep loaded in memory
			d.SetSpriteCache(elem, img)
		}
	}
}

func LoadSprite(filename string) (*image.Image, error) {
	data, err := sprite_fs.ReadFile(filename)
	if err != nil {
		fmt.Printf("Failed to read %s: %v\n", filename, err)
		return nil, err
	}

	reader := bytes.NewReader(data)
	sprite, err := bmp.Decode(reader)
	if err != nil {
		fmt.Printf("Failed to decode %s: %v\n", filename, err)
		return nil, err
	}

	return &sprite, nil
}

func RotateImage(src image.Image, angleDegrees float64) image.Image {
	w := src.Bounds().Dx()
	h := src.Bounds().Dy()

	// Create a new context with enough space to hold the rotated image
	dc := gg.NewContext(w, h)

	// Move origin to center
	dc.Translate(float64(w)/2, float64(h)/2)

	// Rotate canvas
	dc.Rotate(gg.Radians(angleDegrees))

	// Draw image centered at origin
	dc.DrawImageAnchored(src, 0, 0, 0.5, 0.5)

	return dc.Image()
}

func DrawImageAligned(d Driver, src image.Image, x, y int, h_align, v_align RenderType) {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var cx, cy int

	switch h_align {
	case AlignNone:
		cx = x
	case AlignLeft:
		cx = x - width
	case AlignRight:
		cx = x
	case AlignCenter:
		cx = x - int(math.Round(float64(width)/2))
	}

	switch v_align {
	case AlignNone:
		cy = y
	case AlignBelow:
		cy = y
	case AlignAbove:
		cy = y - height
	case AlignCenter:
		cy = y - int(math.Round(float64(height)/2))
	}

	ctx := d.Context()
	ctx.DrawImage(src, cx, cy)
}

func DrawProgressBar(d Driver, x, y, w, h float64, color1, color2 color.Color, status float64) {
	ctx := d.Context()

	// Erase area we will be drawing first
	ctx.SetColor(color1)
	ctx.DrawRectangle(x, y, w, h)
	ctx.Fill()

	// Draw outside border
	ctx.SetColor(color2)
	ctx.SetLineWidth(1)
	ctx.DrawRectangle(x, y, w, h)
	ctx.Stroke()

	// Draw the progress bar
	ctx.DrawRectangle(x+2, y+2, float64(w-4)*status, h-4)
	ctx.Fill()

	// Render it
	d.Render()
}

func PlayAnimation(d Driver, animCtx context.Context, name string, x, y int, align_x, align_y RenderType) {
	anim := d.GetAnimationCache(name)
	if anim == nil {
		fmt.Printf("⚠️ Missing Animation: %s\n", name)
		return
	}

	draw := func(frame Frame) {
		if img := d.GetSpriteCache(frame.Image); img != nil {
			DrawImageAligned(d, *img, x, y, align_x, align_y)
			d.Render()
		} else {
			fmt.Printf("⚠️ Missing sprite for Animation: %s\n", frame.Image)
		}
	}

	if anim.InitialFrame != nil {
		draw(*anim.InitialFrame)

		select {
		case <-animCtx.Done():
			return
		case <-time.After(anim.InitialFrameDelay):
		}
	}

	for {
		for _, frame := range anim.Frames {
			select {
			case <-animCtx.Done():
				return
			default:
			}

			delay := anim.FrameDelay
			if frame.DelayOverride > 0 {
				delay = frame.DelayOverride
			}

			switch frame.Type {
			case PLAY_FRAME:
				draw(frame)

			case STOP_FRAME:
				draw(frame)
				if anim.Loop && anim.LoopDelay > 0 {
					delay = anim.LoopDelay
				}
			}

			select {
			case <-animCtx.Done():
				return
			case <-time.After(delay):
			}
		}
		if !anim.Loop {
			return
		}
	}
}

func FlipImage(s *image.Image, mode RenderType) *image.Image {
	src := *s
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)

	w := bounds.Dx()
	h := bounds.Dy()

	for y := range h {
		for x := range w {
			var sx, sy int

			switch mode {
			case Normal:
				sx, sy = x, y
			case Flipped:
				sx, sy = w-1-x, y
			case UpsideDown:
				sx, sy = x, h-1-y
			case FlippedUpsideDown:
				sx, sy = w-1-x, h-1-y
			default:
				sx, sy = x, y
			}

			srcColor := src.At(bounds.Min.X+sx, bounds.Min.Y+sy)
			dst.Set(bounds.Min.X+x, bounds.Min.Y+y, srcColor)
		}
	}

	var img image.Image = dst
	return &img
}

func InvertImage(s *image.Image) *image.Image {
	src := *s
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := color.RGBAModel.Convert(src.At(x, y)).(color.RGBA)
			invertedColor := color.RGBA{
				R: 255 - originalColor.R,
				G: 255 - originalColor.G,
				B: 255 - originalColor.B,
				A: originalColor.A,
			}
			dst.Set(x, y, invertedColor)
		}
	}

	var img image.Image = dst
	return &img
}

func DrawText(d Driver, x, y int, f map[rune]*image.Image, s string, invert bool) {
	ctx := d.Context()
	cursor := x
	for _, ch := range s {
		spr, ok := f[ch]
		if !ok {
			continue
		}
		sprite := *spr
		width := sprite.Bounds().Dx()

		if invert {
			ctx.DrawImage(*InvertImage(&sprite), cursor, y)
		} else {
			ctx.DrawImage(sprite, cursor, y)
		}

		cursor += width
	}
}

func GetImageBounds(i image.Image) (int, int) {
	bounds := i.Bounds()
	return bounds.Dx(), bounds.Dy()
}

func GetTextBounds(f map[rune]*image.Image, s string) (int, int) {
	var sum_width, max_height int
	for _, ch := range s {
		spr, ok := f[ch]
		if !ok {
			continue
		}
		sprite := *spr
		bounds := sprite.Bounds()
		sum_width += bounds.Dx()
		if bounds.Dy() > max_height {
			max_height = bounds.Dy()
		}
	}

	return sum_width, max_height
}

func DrawTextAligned(d Driver, x, y int, f map[rune]*image.Image, s string, invert bool, h_align, v_align RenderType) {
	sum_width, height := GetTextBounds(f, s)

	var cx, cy int

	switch h_align {
	case AlignNone:
		cx = x
	case AlignLeft:
		cx = x - sum_width
	case AlignRight:
		cx = x
	case AlignCenter:
		cx = x - int(math.Round(float64(sum_width)/2))
	}

	switch v_align {
	case AlignNone:
		cy = y
	case AlignBelow:
		cy = y
	case AlignAbove:
		cy = y - height
	case AlignCenter:
		cy = y - int(math.Round(float64(height)/2))
	}

	DrawText(d, cx, cy, f, s, invert)
}

func DrawTextWrapped(d Driver, x1, y1, x2, y2 int, f map[rune]*image.Image, s string, invert bool, h_align, v_align RenderType) {
	words := strings.Fields(s)
	if len(words) == 0 {
		return
	}

	// Calculate max font height for consistent line spacing
	var fontHeight int
	for _, i := range f {
		img := *i
		h := img.Bounds().Dy()
		if h > fontHeight {
			fontHeight = h
		}
	}
	// Fallback if font is empty (unlikely)
	if fontHeight == 0 {
		fontHeight = 8
	}

	spaceWidth, _ := GetTextBounds(f, " ")
	maxWidth := x2 - x1

	var lines []string
	var currentLine string
	var currentWidth int

	for _, word := range words {
		w, _ := GetTextBounds(f, word)

		// Handle words that are too long for a single line
		if w > maxWidth {
			if len(currentLine) > 0 {
				lines = append(lines, currentLine)
				currentLine = ""
				currentWidth = 0
			}

			for _, ch := range word {
				chStr := string(ch)
				chW, _ := GetTextBounds(f, chStr)

				if currentWidth+chW > maxWidth {
					lines = append(lines, currentLine)
					currentLine = chStr
					currentWidth = chW
				} else {
					currentLine += chStr
					currentWidth += chW
				}
			}
			continue
		}

		if len(currentLine) == 0 {
			currentLine = word
			currentWidth = w
		} else {
			newWidth := currentWidth + spaceWidth + w
			if newWidth <= maxWidth {
				currentLine += " " + word
				currentWidth = newWidth
			} else {
				lines = append(lines, currentLine)
				currentLine = word
				currentWidth = w
			}
		}
	}
	if len(currentLine) > 0 {
		lines = append(lines, currentLine)
	}

	var yCursor int
	var yInc int

	if v_align == WrapUp {
		yCursor = y2 - fontHeight
		yInc = -fontHeight
	} else {
		yCursor = y1
		yInc = fontHeight
	}

	// If wrapping up, we want the last line at the bottom, and the first line at the top.
	// Since we are rendering from y2 upwards, we should iterate backwards through the lines
	// so that the last line is drawn first (at the bottom), and previous lines are drawn above it.
	loopLines := lines
	if v_align == WrapUp {
		loopLines = make([]string, len(lines))
		for i, line := range lines {
			loopLines[len(lines)-1-i] = line
		}
	}

	for _, line := range loopLines {
		if h_align == WrapRight {
			w, _ := GetTextBounds(f, line)
			DrawText(d, x2-w, yCursor, f, line, invert)
		} else {
			DrawText(d, x1, yCursor, f, line, invert)
		}
		yCursor += yInc
	}
}
