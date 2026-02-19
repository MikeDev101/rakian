package sh1107

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"

	"math"
	"os"
	"sync"
	"time"

	"github.com/d2r2/go-i2c"
	"github.com/d2r2/go-logger"
	"github.com/fogleman/gg"
	"github.com/sergeymakinen/go-bmp"
)

//go:embed sprites/*
var sprite_fs embed.FS

var Black color.Color = color.Gray{Y: 0}
var White color.Color = color.Gray{Y: 255}

type SH1107 struct {
	*gg.Context   // Embed drawing context
	bus           *i2c.I2C
	rot           int
	Width, Height int
	IsOn          bool
	FontCache     map[string]image.Image
	render_lock   *sync.Mutex
	fb_lock       *sync.Mutex
	cmd_lock      *sync.Mutex
	data_lock     *sync.Mutex
}

const (
	Normal            int = 0
	Flipped           int = 1
	UpsideDown        int = 2
	FlippedUpsideDown int = 3
)

const (
	AlignNone   int = 0
	AlignLeft   int = 1
	AlignRight  int = 2
	AlignAbove  int = 3
	AlignBelow  int = 4
	AlignCenter int = 5
)

// Creates a new SH1107 display connection
func New(address byte, bus_device int, rotation int, width, height int) *SH1107 {
	logger.ChangePackageLogLevel("i2c", logger.PanicLevel)
	bus, err := i2c.NewI2C(address, bus_device)
	if err != nil {
		panic(err)
	}

	display := &SH1107{
		gg.NewContextForImage(image.NewGray(image.Rect(0, 0, width, height))),
		bus,
		rotation,
		width,
		height,
		false,
		make(map[string]image.Image),
		&sync.Mutex{},
		&sync.Mutex{},
		&sync.Mutex{},
		&sync.Mutex{},
	}

	display.init()
	display.SetRotation(rotation)
	display.Clear(Black)
	return display
}

// TODO: THIS CURRENTLY DOESN'T PROPERLY WORK FOR 90/270 DEGREES
func (d *SH1107) SetRotation(rot int) {
	switch rot % 4 {
	case 0: // 0°
		d.writeCommand(0xA0) // Segment remap normal
		d.writeCommand(0xC0) // COM scan flipped
	case 1: // 90°
		d.writeCommand(0xA1) // Segment remap
		d.writeCommand(0xC0) // COM scan flipped
	case 2: // 180°
		d.writeCommand(0xA1) // Segment remap
		d.writeCommand(0xC8) // COM scan normal
	case 3: // 270°
		d.writeCommand(0xA0) // Segment remap normal
		d.writeCommand(0xC8) // COM scan normal
	}
}

func (d *SH1107) multiCommand(cmd ...byte) {
	d.cmd_lock.Lock()
	defer d.cmd_lock.Unlock()
	buf := append([]byte{0x00}, cmd...)
	d.bus.WriteBytes(buf)
}

func (d *SH1107) writeCommand(cmd ...any) {
	d.cmd_lock.Lock()
	defer d.cmd_lock.Unlock()
	d.write(0x00, cmd...)
}

func (d *SH1107) writeData(data ...any) {
	d.data_lock.Lock()
	defer d.data_lock.Unlock()
	d.write(0x40, data...)
}

func (d *SH1107) write(cmd byte, data ...any) {
	for _, v := range data {
		switch val := v.(type) {
		case byte:
			d.bus.WriteBytes([]byte{cmd, val})
		case int:
			d.bus.WriteBytes([]byte{cmd, byte(val)})
		case []byte:
			d.bus.WriteBytes(append([]byte{cmd}, val...))
		default:
			panic(fmt.Sprintf("Unsupported type %T", v))
		}
	}
}

func (d *SH1107) init() {
	cmds := []byte{
		0xAE,       // display off
		0x00, 0x10, // set column addr low + high
		0xDC, 0x00, // display start line
		0x81, 0x7F, // contrast
		0x20, // page addressing
		0xA4, // disable entire display on
		0xA6, // normal display
		// 0xA7,       // Invert
		0xA8, 0x7F, // multiplex ratio = 127
		0xD3, 0x00, // display offset
		0xD5, 0x41, // osc
		0xD9, 0x22, // precharge
		0xDB, 0x35, // vcomh
		0xAD, 0x8A, // charge pump enable
	}
	for _, cmd := range cmds {
		d.writeCommand(cmd)
	}
}

// Closes the display connection
func (d *SH1107) Close() {
	d.bus.Close()
}

// Turns display on
func (d *SH1107) On() {
	d.writeCommand(0xAF)
	d.IsOn = true
}

// Turns display off
func (d *SH1107) Off() {
	d.writeCommand(0xAE)
	d.IsOn = false
}

// Set brightness from 0.0 to 1.0
func (d *SH1107) SetBrightness(level float64) {
	clamped := math.Max(0.0, math.Min(1.0, level))
	b := byte(clamped * 0xFF)
	d.writeCommand(0x81, b)
}

// Clears the display with either all black or all white. White/Black are constants color.Gray{Y:255} or color.Gray{Y:0}
func (d *SH1107) Clear(state color.Color) {
	d.fb_lock.Lock()
	defer d.fb_lock.Unlock()
	d.SetColor(state)
	d.DrawRectangle(0, 0, float64(d.Width), float64(d.Height))
	d.Fill()
}

// Shows a checkerboard pattern on the display
func (d *SH1107) TestPattern() {
	d.fb_lock.Lock()
	defer d.fb_lock.Unlock()
	d.Clear(Black)
	for y := 0; y < d.Height; y++ {
		for x := 0; x < d.Width; x++ {
			if (x+y)%2 == 0 {
				d.SetColor(Black)
				d.SetPixel(x, y)
			} else {
				d.SetColor(White)
				d.SetPixel(x, y)
			}
		}
	}
	d.Render()
}

// Waits until the d.render_lock is freed
func (d *SH1107) Ready() {
	// Wait until the render lock is free
	d.render_lock.Lock()
	defer d.render_lock.Unlock()
	time.Sleep(100 * time.Millisecond)
}

// Displays whatever is stored in the framebuffer
func (d *SH1107) Render() {
	d.render_lock.Lock()
	raw := d.to_bytes()
	width := d.Width
	pages := d.Height / 8 // 128px height / 8 pixels per page
	for page := range pages {

		// Combine multiple commands into a single transaction
		d.multiCommand(
			0xB0|byte(page), // page address
			0x00,            // low nibble
			0x10,            // high nibble
		)

		// Transmit data as a single transaction
		offset := page * width
		end := offset + width
		d.writeData(raw[offset:end])
	}
	d.render_lock.Unlock()
}

// Display single image to screen
func (d *SH1107) to_bytes() []byte {
	bounds := d.Image().Bounds()
	raw := make([]byte, bounds.Max.X*(bounds.Max.Y/8))

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray := color.GrayModel.Convert(d.Image().At(x, y)).(color.Gray)

			page := y / 8
			offset := page*bounds.Max.Y + x
			bit := uint(y % 8)

			if gray.Y > 127 {
				raw[offset] |= (1 << bit)
			} else {
				raw[offset] &^= (1 << bit)
			}
		}
	}
	return raw
}

// Plays a sequence of images
func (d *SH1107) PlayAnimation(dir string, frameCount int, fps int) {
	const bufferSize = 30
	frameDuration := time.Second / time.Duration(fps)
	buffer := make([]image.Image, 0, bufferSize)
	startTime := time.Now()
	for i := 1; i <= frameCount; i++ {
		filename := fmt.Sprintf("%s/%d.bmp", dir, i)
		file, err := os.Open(filename)
		if err != nil {
			fmt.Printf("Failed to open %s: %v\n", filename, err)
			continue
		}
		img, err := bmp.Decode(file)
		file.Close()
		if err != nil {
			fmt.Printf("Failed to decode %s: %v\n", filename, err)
			continue
		}

		buffer = append(buffer, img)

		if len(buffer) == bufferSize || i == frameCount {
			for j, frame := range buffer {
				targetTime := startTime.Add(time.Duration(i-bufferSize+j) * frameDuration)

				d.DrawImage(frame, 0, 0)
				d.Render()

				// This might not be perfectly consistent
				sleepUntil := time.Until(targetTime)
				if sleepUntil > 0 {
					time.Sleep(sleepUntil)
				}
			}
			buffer = buffer[:0]
		}
	}
}

func (d *SH1107) DrawText(x, y int, f map[rune]image.Image, s string, invert bool) {
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

func (d *SH1107) GetImageBounds(i image.Image) (int, int) {
	bounds := i.Bounds()
	return bounds.Dx(), bounds.Dy()
}

func (d *SH1107) GetTextBounds(f map[rune]image.Image, s string) (int, int) {
	var sum_width, sum_height int
	for _, ch := range s {
		sprite, ok := f[ch]
		if !ok {
			continue
		}
		bounds := sprite.Bounds()
		sum_width += bounds.Dx()
		sum_height += bounds.Dy()
	}
	return sum_width, sum_height
}

func (d *SH1107) DrawTextAligned(x, y int, f map[rune]image.Image, s string, invert bool, h_align, v_align int) {
	sum_width, sum_height := d.GetTextBounds(f, s)
	avg_height := int(math.Round(float64(sum_height) / float64(len(s))))

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
		cy = y + avg_height
	case AlignCenter:
		cy = y + int(math.Round(float64(avg_height)/2))
	}

	d.DrawText(cx, cy, f, s, invert)
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

func (d *SH1107) DrawProgressBar(x, y, w, h float64, status float64) {
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

func (d *SH1107) DrawImageAligned(src image.Image, x, y, h_align, v_align int) {
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
		cy = y + height
	case AlignCenter:
		cy = y - int(math.Round(float64(height)/2))
	}

	d.DrawImage(src, cx, cy)
}
