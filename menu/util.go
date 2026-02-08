package menu

import (
	"fmt"
	"sh1107"
	"time"
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
	m.Display.DrawImage(m.Sprites[icon], 100, 40)
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
		*flash = !*flash

	} else {
		if m.Get("BatteryPercent").(int) <= 5 {
			if *flash {
				m.Display.DrawImage(m.Sprites["battery/0"], 105, 20)
			} else {
				m.Display.DrawImage(m.Sprites["battery/0_warn"], 105, 20)
			}
			*flash = !*flash
		} else {
			m.Display.DrawImage(m.Sprites[fmt.Sprintf("battery/%d", m.Get("BatteryScaledPercent").(int))], 105, 20)
		}
	}
}

func (m *Menu) RenderStatusBar(flash *bool) {
	m.RenderBatteryIcon(flash)

	if m.Modem == nil || m.Modem.FlightMode {
		m.Display.DrawImage(m.Sprites["cell/off"], 0, 20)
	} else {
		m.Display.DrawImage(m.Sprites[fmt.Sprintf("cell/%d", m.Modem.SignalStrength)], 0, 20)
	}

	if m.Get("WiFi_Connected").(bool) {
		m.Display.DrawImage(m.Sprites[fmt.Sprintf("wifi/%d", m.Get("WiFi_Strength").(int))], 20, 20)
	} else {
		m.Display.DrawImage(m.Sprites["wifi/no_networks"], 20, 20)
	}

	m.Display.SetColor(sh1107.White)
	m.Display.SetLineWidth(1)
	m.Display.DrawLine(0, 33, 127, 33)
	m.Display.Stroke()
}
