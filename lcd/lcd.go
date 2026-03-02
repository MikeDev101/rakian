package lcd

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/fogleman/gg"
	"golang.org/x/image/bmp"

	"github.com/stianeikeland/go-rpio/v4"
)

//go:embed sprites/*
var sprite_fs embed.FS

var Black color.Color = color.Gray{Y: 0}
var White color.Color = color.Gray{Y: 255}

const (
	PLAY_FRAME int = 1 // Plays a frame with respect to either the FrameDelay or the DelayOverride.
	STOP_FRAME int = 2 // Halts an animation and allows looping to occur if Loop is set to true.
)

type Frame struct {
	Type          int
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

type LCD struct {
	*gg.Context
	dc             rpio.Pin
	rst            rpio.Pin
	Width          int
	Height         int
	IsOn           bool
	SpriteCache    map[string]image.Image
	FontCache      map[string]map[rune]image.Image
	AnimationCache map[string]Animation
	render_lock    *sync.Mutex
	fb_lock        *sync.Mutex
	cmd_lock       *sync.Mutex
	data_lock      *sync.Mutex
	spi_lock       *sync.Mutex
	draw_lock      *sync.Mutex
}

// Alignment Constants
const (
	AlignNone   int = 0
	AlignLeft   int = 1
	AlignRight  int = 2
	AlignAbove  int = 3
	AlignBelow  int = 4
	AlignCenter int = 5
)

const (
	Normal            int = 0
	Flipped           int = 1
	UpsideDown        int = 2
	FlippedUpsideDown int = 3
)

const (
	WrapDown  int = 10
	WrapUp    int = 11
	WrapLeft  int = 12
	WrapRight int = 13
)

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

// New creates a new LCD display connection over SPI
func New(dc, rst rpio.Pin) *LCD {
	if err := rpio.SpiBegin(rpio.Spi0); err != nil {
		panic(err)
	}
	rpio.SpiSpeed(4000000)
	rpio.SpiChipSelect(0)

	display := &LCD{
		Context:        gg.NewContextForImage(image.NewGray(image.Rect(0, 0, 84, 48))),
		AnimationCache: make(map[string]Animation),
		SpriteCache:    make(map[string]image.Image),
		FontCache:      make(map[string]map[rune]image.Image),
		dc:             dc,
		rst:            rst,
		Width:          84,
		Height:         48,
		IsOn:           false,
		render_lock:    &sync.Mutex{},
		spi_lock:       &sync.Mutex{},
		fb_lock:        &sync.Mutex{},
		cmd_lock:       &sync.Mutex{},
		data_lock:      &sync.Mutex{},
		draw_lock:      &sync.Mutex{},
	}

	dc.Output()
	rst.Output()

	display.init()
	display.Clear(White) // White is "off" (background) for this LCD
	return display
}

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

	d.IsOn = true
}

func (d *LCD) On() {
	d.writeCommand(0x20)
	d.writeCommand(0x0C)
	d.IsOn = true
}

func (d *LCD) Off() {
	d.writeCommand(0x20)
	d.writeCommand(0x08) // Display blank
	d.IsOn = false
}

// Clears the gg.Context buffer and fills it with state color
func (d *LCD) Clear(state color.Color) {
	d.SetColor(state)
	d.DrawRectangle(0, 0, float64(d.Width), float64(d.Height))
	d.Fill()
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
	bounds := d.Image().Bounds()
	// 84 columns * 6 pages (48 pixels / 8) = 504 bytes
	raw := make([]byte, d.Width*(d.Height/8))

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Convert image pixel to Gray
			gray := color.GrayModel.Convert(d.Image().At(x, y)).(color.Gray)

			page := y / 8
			// Fixed your offset logic: it should be multiplied by X width, not Y height
			offset := page*d.Width + x
			bit := uint(y % 8)

			// Thresholding: dark pixels turn ON the LCD pixel
			if gray.Y < 127 {
				raw[offset] |= (1 << bit)
			} else {
				raw[offset] &^= (1 << bit)
			}
		}
	}
	return raw
}

func (d *LCD) PlayAnimation(ctx context.Context, name string, x int, y int, align_x int, align_y int) {
	anim, ok := d.AnimationCache[name]
	if !ok {
		fmt.Printf("⚠️ Missing animation: %s\n", name)
		return
	}

	draw := func(frame Frame) {
		if img, ok := d.SpriteCache[frame.Image]; ok {
			d.DrawImageAligned(img, x, y, align_x, align_y)
			d.Render()
		} else {
			fmt.Printf("⚠️ Missing sprite for animation: %s\n", frame.Image)
		}
	}

	if anim.InitialFrame != nil {
		draw(*anim.InitialFrame)

		select {
		case <-ctx.Done():
			return
		case <-time.After(anim.InitialFrameDelay):
		}
	}

	for {
		for _, frame := range anim.Frames {
			select {
			case <-ctx.Done():
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
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}
		if !anim.Loop {
			return
		}
	}
}

func (d *LCD) DrawText(x, y int, f map[rune]image.Image, s string, invert bool) {
	cursor := x
	for _, ch := range s {
		sprite, ok := f[ch]
		if !ok {
			continue
		}
		width := sprite.Bounds().Dx()

		if invert {
			d.DrawImage(InvertImage(sprite), cursor, y)
		} else {
			d.DrawImage(sprite, cursor, y)
		}

		cursor += width
	}
}

func (d *LCD) GetImageBounds(i image.Image) (int, int) {
	bounds := i.Bounds()
	return bounds.Dx(), bounds.Dy()
}

func (d *LCD) GetTextBounds(f map[rune]image.Image, s string) (int, int) {
	var sum_width, max_height int
	for _, ch := range s {
		sprite, ok := f[ch]
		if !ok {
			continue
		}
		bounds := sprite.Bounds()
		sum_width += bounds.Dx()
		if bounds.Dy() > max_height {
			max_height = bounds.Dy()
		}
	}

	return sum_width, max_height
}

func (d *LCD) DrawTextAligned(x, y int, f map[rune]image.Image, s string, invert bool, h_align, v_align int) {
	sum_width, height := d.GetTextBounds(f, s)

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

	d.DrawText(cx, cy, f, s, invert)
}

func (d *LCD) DrawTextWrapped(x1, y1, x2, y2 int, f map[rune]image.Image, s string, invert bool, h_align, v_align int) {
	words := strings.Fields(s)
	if len(words) == 0 {
		return
	}

	// Calculate max font height for consistent line spacing
	var fontHeight int
	for _, img := range f {
		h := img.Bounds().Dy()
		if h > fontHeight {
			fontHeight = h
		}
	}
	// Fallback if font is empty (unlikely)
	if fontHeight == 0 {
		fontHeight = 8
	}

	spaceWidth, _ := d.GetTextBounds(f, " ")
	maxWidth := x2 - x1

	var lines []string
	var currentLine string
	var currentWidth int

	for _, word := range words {
		w, _ := d.GetTextBounds(f, word)

		// Handle words that are too long for a single line
		if w > maxWidth {
			if len(currentLine) > 0 {
				lines = append(lines, currentLine)
				currentLine = ""
				currentWidth = 0
			}

			for _, ch := range word {
				chStr := string(ch)
				chW, _ := d.GetTextBounds(f, chStr)

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
			w, _ := d.GetTextBounds(f, line)
			d.DrawText(x2-w, yCursor, f, line, invert)
		} else {
			d.DrawText(x1, yCursor, f, line, invert)
		}
		yCursor += yInc
	}
}

func LoadSprite(filename string) (image.Image, error) {
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

	return sprite, nil
}

func (d *LCD) DrawProgressBar(x, y, w, h float64, status float64) {
	d.fb_lock.Lock()
	defer d.fb_lock.Unlock()

	// Erase area we will be drawing first
	d.SetColor(Black)
	d.DrawRectangle(x, y, w, h)
	d.Fill()

	// Draw outside border
	d.SetColor(White)
	d.SetLineWidth(1)
	d.DrawRectangle(x, y, w, h)
	d.Stroke()

	// Draw the progress bar
	d.DrawRectangle(x+2, y+2, float64(w-4)*status, h-4)
	d.Fill()

	// Render it
	d.Render()
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

// FlipImageGray returns a flipped copy of the input image as *image.Gray.
func FlipImage(src image.Image, mode int) *image.Gray {
	bounds := src.Bounds()
	dst := image.NewGray(bounds)

	w := bounds.Dx()
	h := bounds.Dy()

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
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

			srcColor := color.GrayModel.Convert(src.At(bounds.Min.X+sx, bounds.Min.Y+sy)).(color.Gray)
			dst.SetGray(x, y, srcColor)
		}
	}

	return dst
}

func InvertImage(src image.Image) *image.Gray {
	bounds := src.Bounds()
	dst := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray := color.GrayModel.Convert(src.At(x, y)).(color.Gray)
			inverted := color.Gray{Y: 255 - gray.Y}
			dst.SetGray(x, y, inverted)
		}
	}

	return dst
}

func (d *LCD) DrawImageAligned(src image.Image, x, y, h_align, v_align int) {
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

	d.DrawImage(src, cx, cy)
}
