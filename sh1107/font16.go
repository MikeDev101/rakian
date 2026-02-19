package sh1107

import (
	"fmt"
	"image"
	"log"
)

var Font16_rune_map = map[rune]string{

	// Blank/space
	' ': "blank",

	// Uppercase characters
	'A': "upper/A",
	'B': "upper/B",
	'C': "upper/C",
	'D': "upper/D",
	'E': "upper/E",
	'F': "upper/F",
	'G': "upper/G",
	'H': "upper/H",
	'I': "upper/I",
	'J': "upper/J",
	'K': "upper/K",
	'L': "upper/L",
	'M': "upper/M",
	'N': "upper/N",
	'O': "upper/O",
	'P': "upper/P",
	'Q': "upper/Q",
	'R': "upper/R",
	'S': "upper/S",
	'T': "upper/T",
	'U': "upper/U",
	'V': "upper/V",
	'W': "upper/W",
	'X': "upper/X",
	'Y': "upper/Y",
	'Z': "upper/Z",

	// Lowercase characters
	'a': "lower/a",
	'b': "lower/b",
	'c': "lower/c",
	'd': "lower/d",
	'e': "lower/e",
	'f': "lower/f",
	'g': "lower/g",
	'h': "lower/h",
	'i': "lower/i",
	'j': "lower/j",
	'k': "lower/k",
	'l': "lower/l",
	'm': "lower/m",
	'n': "lower/n",
	'o': "lower/o",
	'p': "lower/p",
	'q': "lower/q",
	'r': "lower/r",
	's': "lower/s",
	't': "lower/t",
	'u': "lower/u",
	'v': "lower/v",
	'w': "lower/w",
	'x': "lower/x",
	'y': "lower/y",
	'z': "lower/z",

	// Symbols
	'&':  "symbols/and",
	'@':  "symbols/at",
	'\\': "symbols/back_slash",
	'`':  "symbols/backtick",
	'|':  "symbols/bar",
	'^':  "symbols/caret",
	']':  "symbols/close_bracket",
	')':  "symbols/close_parenthesis",
	':':  "symbols/colon",
	',':  "symbols/comma",
	'$':  "symbols/dollar",
	'=':  "symbols/equals",
	'€':  "symbols/euro",
	'!':  "symbols/exclamation",
	'/':  "symbols/forward_slash",
	'¤':  "symbols/generic_currency",
	'>':  "symbols/greater_than",
	'#':  "symbols/hash",
	'\'': "symbols/hyphen",
	'¡':  "symbols/inverted_exclamation",
	'¿':  "symbols/inverted_question",
	'<':  "symbols/less_than",
	'-':  "symbols/minus",
	'[':  "symbols/open_bracket",
	'(':  "symbols/open_parenthesis",
	'%':  "symbols/percent",
	'.':  "symbols/period",
	'+':  "symbols/plus",
	'£':  "symbols/pound",
	'?':  "symbols/question",
	'"':  "symbols/quote",
	'§':  "symbols/section",
	';':  "symbols/semicolon",
	'*':  "symbols/star",
	'~':  "symbols/tilde",
	'_':  "symbols/underscore",
	'¥':  "symbols/yen",

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

func (d *SH1107) load_font16_rune(char rune) (image.Image, error) {
	relPath, ok := Font16_rune_map[char]
	if !ok {
		return nil, fmt.Errorf("Rune %c not found in font 16", char)
	}
	filePath := "sprites/fonts/16/" + relPath + ".bmp"

	img, err := LoadSprite(filePath)

	return img, err
}

func (d *SH1107) Use_Font16() map[rune]image.Image {
	const fontPrefix = "16."
	cache := make(map[rune]image.Image, len(Font16_rune_map))
	for char, _ := range Font16_rune_map {
		key := fontPrefix + string(char)
		if cached, ok := d.FontCache[key]; ok {
			cache[char] = cached
			continue
		}
		panic(fmt.Sprintf("Rune %c was not cached in font 16", char))
	}
	return cache
}

func (d *SH1107) Load_Font16() {
	const fontPrefix = "16."

	// Load all font runes, or load them from d.FontCache
	cache := make(map[rune]image.Image, len(Font16_rune_map))
	var i = -1
	for char, _ := range Font16_rune_map {
		i++
		key := fontPrefix + string(char)

		// Don't re-load the file if already loaded
		if cached, ok := d.FontCache[key]; ok {
			cache[char] = cached
			continue
		}

		img, err := d.load_font16_rune(char)
		if err != nil {
			log.Printf("Font load failed for '%c': %v", char, err)
			continue
		}
		cache[char] = img

		// Keep loaded in memory
		d.FontCache[key] = img
	}
}
