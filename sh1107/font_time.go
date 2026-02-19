package sh1107

import (
	"fmt"
	"image"
	"log"
)

var Font_time_rune_map = map[rune]string{

	// Blank/space
	' ': "blank",

	// Symbols
	'-': "symbols/dash",
	'/': "symbols/slash",
	'.': "symbols/period",
	':': "symbols/colon",

	// Numbers
	'0': "numbers/0",
	'1': "numbers/1",
	'2': "numbers/2",
	'3': "numbers/3",
	'4': "numbers/4",
	'5': "numbers/5",
	'6': "numbers/6",
	'7': "numbers/7",
	'8': "numbers/8",
	'9': "numbers/9",
}

func (d *SH1107) load_font_time_rune(char rune) (image.Image, error) {
	relPath, ok := Font_time_rune_map[char]
	if !ok {
		return nil, fmt.Errorf("Rune %c not found in font time", char)
	}
	filePath := "sprites/fonts/time/" + relPath + ".bmp"
	img, err := LoadSprite(filePath)
	return img, err
}

func (d *SH1107) Use_Font_Time() map[rune]image.Image {
	const fontPrefix = "time."
	cache := make(map[rune]image.Image, len(Font_time_rune_map))
	for char, _ := range Font_time_rune_map {
		key := fontPrefix + string(char)
		if cached, ok := d.FontCache[key]; ok {
			cache[char] = cached
			continue
		}
		panic(fmt.Sprintf("Rune %c was not cached in font time", char))
	}
	return cache
}

func (d *SH1107) Load_Font_Time() {
	const fontPrefix = "time."

	// Load all font runes, or load them from d.FontCache
	cache := make(map[rune]image.Image, len(Font_time_rune_map))
	var i = -1
	for char, _ := range Font_time_rune_map {
		i++
		key := fontPrefix + string(char)

		// Don't re-load the file if already loaded
		if cached, ok := d.FontCache[key]; ok {
			cache[char] = cached
			continue
		}

		img, err := d.load_font_time_rune(char)
		if err != nil {
			log.Printf("Font load failed for '%c': %v", char, err)
			continue
		}
		cache[char] = img

		// Keep loaded in memory
		d.FontCache[key] = img
	}
}
