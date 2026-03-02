package lcd

import (
	"fmt"
	"image"
	"log"
)

var Sprite_map = map[string]string{

	// Nokia hands logo
	"boot_0": "0331",
	"boot_1": "0332",
	"boot_2": "0333",
	"boot_3": "0334",
	"boot_4": "0335",
	"boot_5": "0336",
	"boot_6": "0337",

	"battery":            "battery",
	"cell":               "cell",
	"duck":               "duck",
	"logo":               "logo",
	"powered_by_armbian": "powered_by_armbian",

	"wifi_0": "wifi_0",
	"wifi_1": "wifi_1",
	"wifi_2": "wifi_2",
	"wifi_3": "wifi_3",
	"wifi_4": "wifi_4",
	"wifi_5": "wifi_5",
	"wifi_6": "wifi_6",
	"wifi_7": "wifi_7",

	"connecting":        "wifi_connecting",
	"networks_found":    "wifi_networks_found",
	"no_internet":       "wifi_no_internet",
	"no_networks_found": "wifi_no_networks",

	"fax":  "0001",
	"e":    "0002",
	"f":    "0003",
	"f...": "0004",

	"write":       "0007",
	"new_message": "0008",

	"lowercase": "0009",
	"uppercase": "0010",
	"numbers":   "0011",
	"symbols":   "0012",

	"pencil":      "0013",
	"notes":       "0014",
	"home":        "0015",
	"key":         "0016",
	"redirect":    "0017",
	"missed_call": "0018",
	"1":           "0019",
	"2":           "0020",
	"mute":        "0021",
	"1_redial":    "0022",
	"2_redial":    "0023",
	"12_redial":   "0024",

	"vmail_0": "0025",
	"vmail_1": "0026",
	"vmail_2": "0027",
	"vmail_3": "0028",

	"book":           "0029",
	"bell":           "0030",
	"clock":          "0031",
	"idk":            "0032",
	"unlocked":       "0033",
	"pinyin":         "0034",
	"something":      "0035",
	"also_something": "0036",
	"i_have_no_idea": "0037",
	"still_no_idea":  "0038",
	"really_no_idea": "0039",

	"arrow_left":  "0040",
	"arrow_right": "0041",

	"star":    "0042",
	"letters": "0043",

	// ????

	"filled_box":  "0048",
	"empty_box":   "0049",
	"partial_box": "0050",
	"pencil2":     "0051",
	"pencil3":     "0052",
	"bucket":      "0053",
	"character":   "0054",
	"emote":       "0055",
	"line":        "0056",
	"rectangle":   "0057",
	"circle":      "0058",

	"call_in_progress": "0059",
	"call_empty":       "0060",

	"dashes": "0061",

	"d":            "0063",
	"d...":         "0064",
	"connection":   "0065",
	"blink_blank":  "0066",
	"blink_filled": "0067",
	"file":         "0068",
	"arrow_select": "0069",
	"message":      "0070",
	"file2":        "0071",
	"i":            "0072",

	"pin": "0075",

	"telephone": "0077",
	"hands":     "0078",
	"gift":      "0079",

	// Static icons
	"trashcan": "0080",
	"sending":  "0080",

	// Misc
	"alarm_set":      "0082",
	"timer_sleeping": "0083",
	"sim_card":       "0084",
	"warning":        "0085",
	"info":           "0086",
	"prohibited":     "0087",

	// Battery states
	"battery_full":  "0230",
	"low_battery_0": "0231",
	"low_battery_1": "0232",
	"low_battery_2": "0233",
	"empty_battery": "0234",

	// Battery discharging
	"discharging_0": "0235",
	"discharging_1": "0236",
	"discharging_2": "0237",
	"discharging_3": "0238",

	// Battery charging
	"charging_0": "0239",
	"charging_1": "0240",
	"charging_2": "0241",
	"charging_3": "0242",

	// Message sending
	"sending_0":  "0243",
	"sending_1":  "0244",
	"sending_2":  "0245",
	"sending_3":  "0246",
	"sending_4":  "0247",
	"sending_5":  "0248",
	"sending_6":  "0249",
	"sending_7":  "0250",
	"sending_8":  "0251",
	"sending_9":  "0252",
	"sending_10": "0253",
	"sending_11": "0254",
	"sending_12": "0255",
	"sending_13": "0256",

	// Generic confirmation
	"ok_0": "0256",
	"ok_1": "0257",
	"ok_2": "0258",
	"ok_3": "0259",

	// Show how to unlock the keypad
	"keypad_unlock_0": "0266",
	"keypad_unlock_1": "0267",
	"keypad_unlock_2": "0268",
	"keypad_unlock_3": "0269",
	"keypad_unlock_4": "0270",
	"keypad_unlock_5": "0271",
	"keypad_unlock_6": "0272",

	// Play in reverse for unlock
	"keypad_locked_0": "0273",
	"keypad_locked_1": "0274",
	"keypad_locked_2": "0275",
	"keypad_locked_3": "0276",

	// Phone book
	"phonebook_0": "0088",
	"phonebook_1": "0089",
	"phonebook_2": "0090",
	"phonebook_3": "0091",

	// Messages
	"messages_0":  "0092",
	"messages_1":  "0093",
	"messages_2":  "0094",
	"messages_3":  "0095",
	"messages_4":  "0096",
	"messages_5":  "0097",
	"messages_6":  "0098",
	"messages_7":  "0099",
	"messages_8":  "0100",
	"messages_9":  "0101",
	"messages_10": "0102",
	"messages_11": "0103",
	"messages_12": "0104",
	"messages_13": "0105",

	// Chats
	"chats_0":  "0106",
	"chats_1":  "0107",
	"chats_2":  "0108",
	"chats_3":  "0109",
	"chats_4":  "0110",
	"chats_5":  "0111",
	"chats_6":  "0112",
	"chats_7":  "0113",
	"chats_8":  "0114",
	"chats_9":  "0115",
	"chats_10": "0116",
	"chats_11": "0117",

	// Call register
	"call_register_0":  "0118",
	"call_register_1":  "0119",
	"call_register_2":  "0120",
	"call_register_3":  "0121",
	"call_register_4":  "0122",
	"call_register_5":  "0123",
	"call_register_6":  "0124",
	"call_register_7":  "0125",
	"call_register_8":  "0126",
	"call_register_9":  "0127",
	"call_register_10": "0128",

	// Settings
	"settings_0": "0129",
	"settings_1": "0130",
	"settings_2": "0131",
	"settings_3": "0132",
	"settings_4": "0133",
	"settings_5": "0134",
	"settings_6": "0135",
	"settings_7": "0136",

	// Call divert
	"call_divert_0":  "0137",
	"call_divert_1":  "0138",
	"call_divert_2":  "0139",
	"call_divert_3":  "0140",
	"call_divert_4":  "0141",
	"call_divert_5":  "0142",
	"call_divert_6":  "0143",
	"call_divert_7":  "0144",
	"call_divert_8":  "0145",
	"call_divert_9":  "0146",
	"call_divert_10": "0147",

	// Apps (Games)
	"apps_0": "0148",
	"apps_1": "0149",
	"apps_2": "0150",
	"apps_3": "0151",

	// Calculator
	"calculator_0":  "0152",
	"calculator_1":  "0153",
	"calculator_2":  "0154",
	"calculator_3":  "0155",
	"calculator_4":  "0156",
	"calculator_5":  "0157",
	"calculator_6":  "0158",
	"calculator_7":  "0159",
	"calculator_8":  "0160",
	"calculator_9":  "0161",
	"calculator_10": "0162",
	"calculator_11": "0163",
	"calculator_12": "0164",

	// Notes
	"notes_0":  "0165",
	"notes_1":  "0166",
	"notes_2":  "0167",
	"notes_3":  "0168",
	"notes_4":  "0169",
	"notes_5":  "0170",
	"notes_6":  "0171",
	"notes_7":  "0172",
	"notes_8":  "0173",
	"notes_9":  "0174",
	"notes_10": "0175",
	"notes_11": "0176",
	"notes_12": "0177",
	"notes_13": "0178",

	// Clock
	"clock_0": "0179",
	"clock_1": "0180",
	"clock_2": "0181",
	"clock_3": "0182",
	"clock_4": "0183",
	"clock_5": "0184",
	"clock_6": "0185",
	"clock_7": "0186",

	// Profiles
	"profiles_0": "0187",
	"profiles_1": "0188",
	"profiles_2": "0189",
	"profiles_3": "0190",
	"profiles_4": "0191",
	"profiles_5": "0192",
	"profiles_6": "0193",
	"profiles_7": "0194",

	// Tones
	"tones_0":  "0195",
	"tones_1":  "0196",
	"tones_2":  "0197",
	"tones_3":  "0198",
	"tones_4":  "0199",
	"tones_5":  "0200",
	"tones_6":  "0201",
	"tones_7":  "0202",
	"tones_8":  "0203",
	"tones_9":  "0204",
	"tones_10": "0205",
	"tones_11": "0206",
	"tones_12": "0207",
	"tones_13": "0208",
	"tones_14": "0209",
	"tones_15": "0210",
	"tones_16": "0211",

	// SIM tools
	"sim_tools_0": "0212",
	"sim_tools_1": "0213",
	"sim_tools_2": "0214",
	"sim_tools_3": "0215",
	"sim_tools_4": "0216",
	"sim_tools_5": "0217",

	// Drawing
	"drawing_0": "0218",
	"drawing_1": "0219",
	"drawing_2": "0220",
	"drawing_3": "0221",
	"drawing_4": "0222",
	"drawing_5": "0223",
	"drawing_6": "0224",
	"drawing_7": "0225",
	"drawing_8": "0226",
}

func (d *LCD) load_sprite_elem(elem string) (image.Image, error) {
	relPath, ok := Sprite_map[elem]
	if !ok {
		return nil, fmt.Errorf("Elem %s not found in sprites", elem)
	}
	filePath := "sprites/" + relPath + ".bmp"

	img, err := LoadSprite(filePath)

	return img, err
}

func (d *LCD) Use_Sprites() map[string]image.Image {
	return d.SpriteCache
}

func (d *LCD) Load_Sprites() {

	// Load all sprite elements, or load them from d.SpriteCache
	cache := make(map[string]image.Image, len(Sprite_map))
	var i = -1
	for elem := range Sprite_map {
		i++

		// Don't re-load the file if already loaded
		if cached, ok := d.SpriteCache[elem]; ok {
			cache[elem] = cached
			continue
		}

		img, err := d.load_sprite_elem(elem)
		if err != nil {
			log.Printf("Sprite load failed for '%s': %v", elem, err)
			continue
		}
		cache[elem] = img

		// Keep loaded in memory
		d.SpriteCache[elem] = img
	}
}
