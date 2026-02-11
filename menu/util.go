package menu

import (
	"fmt"
	"sh1107"
	"time"

	"github.com/Wifx/gonetworkmanager/v3"
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
		m.Display.DrawImage(m.Sprites[icon], 100, 40)
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

		// === 1.1: SIM CARD STATUS ===

		// Get the width of the SIM card state
		sim_width := 0

		// If the SIM card is not inserted, show an icon (ignore in airplane mode)
		if !m.Modem.FlightMode && !m.Modem.SimCardInserted {

			// Get the icon width
			sim_width, _ = m.Display.GetImageBounds(m.Sprites["cell/no_sim"])

			// Draw the icon
			m.Display.DrawImage(m.Sprites["cell/no_sim"], 0, 20)
		}

		// Update the counter
		multi_render_width += sim_width + multi_render_padding

		// == 1.2: NETWORK GENERATION ===

		netgen_font := m.Display.Use_Font8_Bold()
		netgen_width, _ := m.Display.GetTextBounds(netgen_font, m.Modem.NetworkGeneration)
		m.Display.DrawTextAligned(multi_render_width, 21, netgen_font, m.Modem.NetworkGeneration, false, sh1107.AlignRight, sh1107.AlignNone)

		// Update the counter
		multi_render_width += netgen_width + multi_render_padding

		// === 1.3: SIGNAL STATUS ===

		// Get modem state
		cell_image := "cell/off"
		if m.Modem.FlightMode {
			// Set the icon to airplane mode
			cell_image = "cell/airplane"
		} else {
			// Set the icon to the signal strength
			cell_image = fmt.Sprintf("cell/%d", m.Modem.SignalStrength)
		}

		// Get the width of the modem icon state
		modem_image_width, _ = m.Display.GetImageBounds(m.Sprites[cell_image])

		// Draw the icon
		m.Display.DrawImage(m.Sprites[cell_image], multi_render_width, 20)

		// 1.4: DATA STATUS
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
		panic(err)
	}

	wifi_enabled, err := m.NetworkManager.GetPropertyWirelessEnabled()
	if err != nil {
		panic(err)
	}

	network_status, err := m.NetworkManager.GetPropertyState()
	if err != nil {
		panic(err)
	}

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

	// Update to add further stages as necessary

	// At the end, draw the borderline below the status bar
	m.Display.SetColor(sh1107.White)
	m.Display.SetLineWidth(1)
	m.Display.DrawLine(0, 33, 127, 33)
	m.Display.Stroke()
}
