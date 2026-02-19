package menu

import (
	"context"
	"fmt"
	"log"
	"misc"
	"sh1107"
	"strings"
	"time"

	"github.com/Wifx/gonetworkmanager/v3"
)

const (
	T9Lowercase = 0
	T9Uppercase = 1
	T9Numbers   = 2
)

func (m *Menu) PlayAlert() {
	if m.Get("BeepOnly").(bool) {
		m.Player.Stop()
		m.Player.Tone(83, 5) // B5 (Alert/Stop sound)
		time.Sleep(50 * time.Millisecond)
		m.Player.Stop()
	} else if m.Get("CanRing").(bool) {
		m.Player.Stop()
		m.Player.Tone(83, 5) // B5 (Alert/Stop sound)
		time.Sleep(time.Second)
		m.Player.Stop()
	}
}

func (m *Menu) PlayKey() {
	if m.Get("BeepOnly").(bool) {
		m.Player.Tone(82, 2)
		time.Sleep(50 * time.Millisecond)
		m.Player.Stop()
	} else if m.Get("CanRing").(bool) {
		m.Player.Tone(82, 2)
		time.Sleep(150 * time.Millisecond)
		m.Player.Stop()
	}
}

func (m *Menu) RenderAlert(icon string, status []string) {
	font := m.Display.Use_Font16()
	m.Display.Clear(sh1107.Black)
	if icon != "" {
		m.Display.DrawImageAligned(m.Sprites[icon], 120, 40, sh1107.AlignLeft, sh1107.AlignBelow)
	}
	for i, str := range status {
		m.Display.DrawText(0, 36+(i*16), font, str, false)
	}
	m.Display.Render()
}

func (m *Menu) RenderBatteryIcon(flash *bool) {
	if !m.Get("BatteryOK").(bool) {
		if *flash {
			m.Display.DrawImage(m.Sprites["battery/0"], 105, 20)
		} else {
			m.Display.DrawImage(m.Sprites["battery/unknown"], 105, 20)
		}

	} else if m.Get("BatteryCharging").(bool) {
		if *flash {
			m.Display.DrawImage(m.Sprites["battery/0"], 105, 20)
		} else {
			m.Display.DrawImage(m.Sprites[fmt.Sprintf("battery/%d", m.Get("BatteryScaledPercent").(int))], 105, 20)
		}

	} else {
		if m.Get("BatteryPercent").(int) <= 5 {
			if *flash {
				m.Display.DrawImage(m.Sprites["battery/0"], 105, 20)
			} else {
				m.Display.DrawImage(m.Sprites["battery/0_warn"], 105, 20)
			}
		} else {
			m.Display.DrawImage(m.Sprites[fmt.Sprintf("battery/%d", m.Get("BatteryScaledPercent").(int))], 105, 20)
		}
	}
}

func (m *Menu) RenderStatusBar(batt_flash *bool, data_flash *bool) {
	m.RenderBatteryIcon(batt_flash)

	// Create first counter to determine element spacing
	multi_render_width := 0
	const multi_render_padding = 1

	modem_image_width := 0

	// === STAGE 1: MODEM ===
	if m.Modem != nil {

		// == 1.1: NETWORK GENERATION ===

		netgen_font := m.Display.Use_Font8_Bold()
		netgen_width, _ := m.Display.GetTextBounds(netgen_font, m.Modem.NetworkGeneration)
		m.Display.DrawTextAligned(multi_render_width, 21, netgen_font, m.Modem.NetworkGeneration, false, sh1107.AlignRight, sh1107.AlignNone)

		// Update the counter
		multi_render_width += netgen_width + multi_render_padding

		// === 1.2: SIGNAL STATUS ===

		// Get modem state
		cell_image := "cell/no_sim"
		if m.Modem.FlightMode {
			// Set the icon to airplane mode
			cell_image = "cell/airplane"
		} else if m.Modem.SimCardInserted {
			// Set the icon to the signal strength
			cell_image = fmt.Sprintf("cell/%d", m.Modem.SignalStrength)
		}

		// Get the width of the modem icon state
		modem_image_width, _ = m.Display.GetImageBounds(m.Sprites[cell_image])

		// Draw the icon
		m.Display.DrawImage(m.Sprites[cell_image], multi_render_width, 20)

		// 1.3: DATA STATUS
		data_image := ""
		data_show := false
		if !m.Modem.FlightMode {
			if m.Modem.Connected {
				if m.Modem.DataEnabled {
					if m.Modem.DataConnected {
						data_show = *data_flash
						data_image = "cell/data_active"
					} else {
						data_show = true
						data_image = "cell/data_inactive"
					}
				}
			}
		}

		// Get the width of the data icon and draw it
		data_image_width := 0

		if data_image != "" {
			data_image_width, _ = m.Display.GetImageBounds(m.Sprites[data_image])
		}

		if data_show {
			m.Display.DrawImage(m.Sprites[data_image], multi_render_width+modem_image_width+multi_render_padding, 20)
		}

		// Update the counter
		multi_render_width += data_image_width + multi_render_padding

	} else {
		modem_image_width, _ = m.Display.GetImageBounds(m.Sprites["cell/off"])
		m.Display.DrawImage(m.Sprites["cell/off"], multi_render_width, 20)
	}

	multi_render_width += modem_image_width + multi_render_padding

	// === STAGE 2: WIFI STATUS ===

	network_enabled, err := m.NetworkManager.GetPropertyNetworkingEnabled()
	if err != nil {
		log.Println("⚠️ Failed to get network status:", err)
	}

	wifi_enabled, err := m.NetworkManager.GetPropertyWirelessEnabled()
	if err != nil {
		log.Println("⚠️ Failed to get WiFi status:", err)
	}

	network_status, err := m.NetworkManager.GetPropertyState()
	if err != nil {
		log.Println("⚠️ Failed to get network status:", err)
	} else {
		// Check if the network is alive
		netstate_width := 0

		if network_enabled && wifi_enabled {
			if network_status == gonetworkmanager.NmStateConnectedLocal ||
				network_status == gonetworkmanager.NmStateConnectedSite {
				netstate_width, _ = m.Display.GetImageBounds(m.Sprites["wifi/no_internet"])
				m.Display.DrawImage(m.Sprites["wifi/no_internet"], multi_render_width, 20)
			}
		}

		// Update the counter
		multi_render_width += netstate_width + multi_render_padding
	}

	// Show the WiFi status icon
	var wifi_icon_target string

	if network_enabled && wifi_enabled {
		if m.Get("WiFi_Connected").(bool) {
			wifi_icon_target = fmt.Sprintf("wifi/%d", m.Get("WiFi_Strength").(int))
		} else {
			wifi_icon_target = "wifi/no_networks"
		}
	}

	// Get the width of the wifi icon
	wifi_icon_width := 0
	if network_enabled && wifi_icon_target != "" {
		wifi_icon_width, _ = m.Display.GetImageBounds(m.Sprites[wifi_icon_target])

		// Draw the icon
		m.Display.DrawImage(m.Sprites[wifi_icon_target], multi_render_width, 20)
	}

	// Update the counter
	multi_render_width += wifi_icon_width + multi_render_padding

	// === STAGE 3: BLUETOOTH STATUS ===

	if m.Get("BluetoothEnabled").(bool) {
		bluetooth_icon := "bluetooth/idle"

		// TODO: check if bluetooth has any active connections

		// Get the width of the bluetooth icon
		bluetooth_icon_width, _ := m.Display.GetImageBounds(m.Sprites[bluetooth_icon])

		// Draw the icon
		m.Display.DrawImage(m.Sprites[bluetooth_icon], multi_render_width, 20)

		// Update the counter
		multi_render_width += bluetooth_icon_width + multi_render_padding
	}

	// Update to add further stages as necessary

	// At the end, draw the borderline below the status bar
	m.Display.SetColor(sh1107.White)
	m.Display.SetLineWidth(1)
	m.Display.DrawLine(0, 33, 127, 33)
	m.Display.Stroke()
}

func (instance *Menu) EnterText(title string, ctx context.Context) string {

	// Text entry handler
	var input []rune
	cursorPos := 0
	display := instance.Display

	// Key mapping
	keyMap := map[rune]string{
		'1': ".,?!-&`:1", '2': "abc2", '3': "def3",
		'4': "ghi4", '5': "jkl5", '6': "mno6",
		'7': "pqrs7", '8': "tuv8", '9': "wxyz9",
		'0': " 0",
	}

	// T9 mode mapping
	t9Map := map[int]string{
		T9Lowercase: "lowercase",
		T9Uppercase: "uppercase",
		T9Numbers:   "numbers",
	}

	t9Mode := T9Lowercase
	var lastKey rune
	var lastPressTime time.Time
	var cycleIndex int

	// Temporarily stop timeouts
	instance.Timers["oled"].Stop()
	instance.Timers["keypad"].Stop()
	misc.KeyLightsOn()
	defer instance.Timers["oled"].Restart()
	defer instance.Timers["keypad"].Restart()

	render := func() {
		display.Clear(sh1107.Black)

		// Draw label
		font := display.Use_Font8_Normal()
		display.DrawTextAligned(0, 20, font, title, false, sh1107.AlignRight, sh1107.AlignNone)

		// Draw mode helper
		display.DrawImageAligned(instance.Sprites[t9Map[t9Mode]], 128, 20, sh1107.AlignLeft, sh1107.AlignNone)

		// Draw line
		display.SetColor(sh1107.White)
		display.DrawLine(0, 33, 127, 33)
		display.Stroke()

		font = display.Use_Font16()
		display.DrawText(0, 45, font, string(input), false)

		// Draw cursor
		prefix := string(input[:cursorPos])
		w, _ := display.GetTextBounds(font, prefix)
		display.DrawLine(float64(w), 45, float64(w), 60)
		display.Stroke()

		// Draw bottom
		font = display.Use_Font8_Bold()
		display.DrawTextAligned(64, 110, font, "Enter", false, sh1107.AlignCenter, sh1107.AlignNone)
		display.Render()
	}

	render()

	for {
		select {
		case <-ctx.Done():
			return ""
		case evt := <-instance.KeypadEvents:
			if !evt.State {
				continue
			}
			misc.KeyLightsOn()
			go instance.PlayKey()

			now := time.Now()

			switch evt.Key {
			case 'S':
				return string(input)
			case 'C':
				lastKey = 0
				if len(input) > 0 {
					if cursorPos > 0 {
						input = append(input[:cursorPos-1], input[cursorPos:]...)
						cursorPos--
					}
					render()
				} else {
					return ""
				}
			case 'P':
				go instance.Push("power")
				return ""
			case 'U':
				lastKey = 0
				if cursorPos > 0 {
					cursorPos--
					render()
				}
			case 'D':
				lastKey = 0
				if cursorPos < len(input) {
					cursorPos++
					render()
				}
			case '#':
				lastKey = 0
				t9Mode = (t9Mode + 1) % 3
				render()
			case '*':
				lastKey = 0
				// Symbol selector
				symbols := []rune(".,?!@_()[]{}#%^*+=/|\\<>~'\"")
				selIdx := 0

				// Mini loop for symbol selection
			symbolLoop:
				for {
					display.Clear(sh1107.Black)
					font := display.Use_Font8_Normal()
					display.DrawTextAligned(64, 20, font, "Select Symbol", false, sh1107.AlignCenter, sh1107.AlignNone)

					// Draw symbols grid (5 per row)
					font = display.Use_Font16()
					for i, r := range symbols {
						x := (i%5)*20 + 1
						y := 40 + (i/5)*15
						display.DrawText(x, y, font, string(r), i == selIdx)
					}
					display.Render()

					select {
					case <-ctx.Done():
						return ""
					case sEvt := <-instance.KeypadEvents:
						if !sEvt.State {
							continue
						}
						go instance.PlayKey()
						switch sEvt.Key {
						case 'U': // Left
							if selIdx > 0 {
								selIdx--
							}
						case 'D': // Right
							if selIdx < len(symbols)-1 {
								selIdx++
							}
						case '2': // Up
							if selIdx >= 5 {
								selIdx -= 5
							}
						case '8': // Down
							if selIdx+5 < len(symbols) {
								selIdx += 5
							}
						case 'S': // Select
							// Insert symbol
							input = append(input[:cursorPos], append([]rune{symbols[selIdx]}, input[cursorPos:]...)...)
							cursorPos++
							break symbolLoop
						case 'C': // Cancel
							break symbolLoop
						}
					}
				}
				render()

			default:
				chars, ok := keyMap[evt.Key]
				if ok {
					// Adjust chars based on mode
					switch t9Mode {
					case T9Numbers:
						chars = string(evt.Key)
					case T9Uppercase:
						chars = strings.ToUpper(chars)
					}

					if evt.Key == lastKey && now.Sub(lastPressTime) < 1*time.Second {
						// Cycle
						cycleIndex = (cycleIndex + 1) % len(chars)
						if len(input) > 0 {
							// Replace char at cursor-1
							if cursorPos > 0 {
								input[cursorPos-1] = rune(chars[cycleIndex])
							}
						}
					} else {
						// New char
						cycleIndex = 0
						// Insert at cursor
						newChar := rune(chars[0])
						input = append(input[:cursorPos], append([]rune{newChar}, input[cursorPos:]...)...)
						cursorPos++
					}
					lastKey = evt.Key
					lastPressTime = now
					render()
				}
			}
		}
	}
}
