package lcd

import (
	"fmt"
	"image"
	"log"
)

var Font_tiny_rune_map = map[rune]string{
	' ':  "0020",
	'!':  "0021",
	'"':  "0022",
	'&':  "0026",
	'\'': "0027",
	'(':  "0028",
	')':  "0029",
	'*':  "002a",
	'+':  "002b",
	',':  "002c",
	'-':  "002d",
	'.':  "002e",
	'/':  "002f",
	'0':  "0030",
	'1':  "0031",
	'2':  "0032",
	'3':  "0033",
	'4':  "0034",
	'5':  "0035",
	'6':  "0036",
	'7':  "0037",
	'8':  "0038",
	'9':  "0039",
	':':  "003a",
	';':  "003b",
	'=':  "003d",
	'?':  "003f",
	'A':  "0041",
	'B':  "0042",
	'C':  "0043",
	'D':  "0044",
	'E':  "0045",
	'F':  "0046",
	'G':  "0047",
	'H':  "0048",
	'I':  "0049",
	'J':  "004a",
	'K':  "004b",
	'L':  "004c",
	'M':  "004d",
	'N':  "004e",
	'O':  "004f",
	'P':  "0050",
	'Q':  "0051",
	'R':  "0052",
	'S':  "0053",
	'T':  "0054",
	'U':  "0055",
	'V':  "0056",
	'W':  "0057",
	'X':  "0058",
	'Y':  "0059",
	'Z':  "005a",
	'_':  "005f",
	'Ѐ':  "0400",
	'Ё':  "0401",
	'Ђ':  "0402",
	'Ѓ':  "0403",
	'Є':  "0404",
	'Ѕ':  "0405",
	'І':  "0406",
	'Ї':  "0407",
	'Ј':  "0408",
	'Љ':  "0409",
	'Њ':  "040a",
	'Ћ':  "040b",
	'Ќ':  "040c",
	'Ѝ':  "040d",
	'Ў':  "040e",
	'Џ':  "040f",
	'А':  "0410",
	'Б':  "0411",
	'В':  "0412",
	'Г':  "0413",
	'Д':  "0414",
	'Е':  "0415",
	'Ж':  "0416",
	'З':  "0417",
	'И':  "0418",
	'Й':  "0419",
	'К':  "041a",
	'Л':  "041b",
	'М':  "041c",
	'Н':  "041d",
	'О':  "041e",
	'П':  "041f",
	'Р':  "0420",
	'С':  "0421",
	'Т':  "0422",
	'У':  "0423",
	'Ф':  "0424",
	'Х':  "0425",
	'Ц':  "0426",
	'Ч':  "0427",
	'Ш':  "0428",
	'Щ':  "0429",
	'Ъ':  "042a",
	'Ы':  "042b",
	'Ь':  "042c",
	'Э':  "042d",
	'Ю':  "042e",
	'Я':  "042f",
	'а':  "0430",
	'б':  "0431",
	'в':  "0432",
	'г':  "0433",
	'д':  "0434",
	'е':  "0435",
	'ж':  "0436",
	'з':  "0437",
	'и':  "0438",
	'й':  "0439",
	'к':  "043a",
	'л':  "043b",
	'м':  "043c",
	'н':  "043d",
	'о':  "043e",
	'п':  "043f",
	'р':  "0440",
	'с':  "0441",
	'т':  "0442",
	'у':  "0443",
	'ф':  "0444",
	'х':  "0445",
	'ц':  "0446",
	'ч':  "0447",
	'ш':  "0448",
	'щ':  "0449",
	'ъ':  "044a",
	'ы':  "044b",
	'ь':  "044c",
	'э':  "044d",
	'ю':  "044e",
	'я':  "044f",
	'ѐ':  "0450",
	'ё':  "0451",
	'':  "e000",
}

func (d *LCD) load_font_tiny_plain_rune(char rune) (image.Image, error) {
	relPath, ok := Font_tiny_rune_map[char]
	if !ok {
		return nil, fmt.Errorf("Rune %c not found in font time", char)
	}
	filePath := "sprites/fonts/tinyplain/" + relPath + ".bmp"
	img, err := LoadSprite(filePath)
	return img, err
}

func (d *LCD) Use_Font_Tiny() map[rune]image.Image {
	const fontPrefix = "tiny"
	if cache, ok := d.FontCache[fontPrefix]; !ok {
		panic(fmt.Sprintf("Font %s not loaded", fontPrefix))
	} else {
		return cache
	}
}

func (d *LCD) Load_Font_Tiny() {
	const fontPrefix = "tiny"
	mapping := Font_tiny_rune_map
	mapfunc := d.load_font_tiny_plain_rune

	// Load all font runes, or load them from d.FontCache
	for char := range mapping {

		// Create prefix element if it doesn't exist
		if _, ok := d.FontCache[fontPrefix]; !ok {
			d.FontCache[fontPrefix] = make(map[rune]image.Image)
		}

		// Don't re-load the file if already loaded
		if _, ok := d.FontCache[fontPrefix][char]; ok {
			continue
		}

		// Load the rune image
		img, err := mapfunc(char)
		if err != nil {
			log.Printf("Font load failed for '%c': %v", char, err)
			continue
		}

		// Keep loaded in memory
		d.FontCache[fontPrefix][char] = img
	}
}
