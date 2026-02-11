package menu

import (
	"context"
	"fmt"
	"log"
	"misc"
	"sh1107"
	"sync"
	"time"

	"github.com/Wifx/gonetworkmanager/v3"
)

type SettingsMenu struct {
	ctx               context.Context
	configured        bool
	cancelFn          context.CancelFunc
	parent            *Menu
	wg                sync.WaitGroup
	process_selection bool
	selection_path    []string
	options           [][]string
}

func (instance *SettingsMenu) RenderAbout() {
	m := instance.parent
	display := m.Display

	display.Clear(sh1107.Black)

	display.DrawImageAligned(m.Sprites["logo"], 60, 50, sh1107.AlignCenter, sh1107.AlignCenter)

	font := display.Use_Font8_Normal()
	display.DrawTextAligned(0, 20, font, "About", false, sh1107.AlignRight, sh1107.AlignNone)
	display.DrawTextAligned(60, 60, font, "Rakian OS", false, sh1107.AlignCenter, sh1107.AlignNone)
	display.DrawTextAligned(60, 70, font, fmt.Sprintf("v%s", m.Get("FirmwareVersion").(string)), false, sh1107.AlignCenter, sh1107.AlignNone)

	display.SetColor(sh1107.White)
	display.SetLineWidth(1)
	display.DrawLine(0, 33, 127, 33)
	display.Stroke()

	font = display.Use_Font8_Bold()
	display.DrawTextAligned(64, 105, font, "Check for updates", false, sh1107.AlignCenter, sh1107.AlignNone)

	display.Render()
}

func (instance *SettingsMenu) RenderNetworkStatus(state_msg string, network_info gonetworkmanager.ActiveConnection) {
	m := instance.parent
	display := m.Display

	display.Clear(sh1107.Black)

	font := display.Use_Font8_Normal()
	display.DrawTextAligned(0, 20, font, "Network Status", false, sh1107.AlignRight, sh1107.AlignNone)
	display.DrawTextAligned(0, 40, font, state_msg, false, sh1107.AlignRight, sh1107.AlignNone)

	if net_conn, err := network_info.GetPropertyID(); err == nil {
		// instance.parent.Get("WiFi_SSID").(string)
		display.DrawTextAligned(0, 55, font, net_conn, false, sh1107.AlignRight, sh1107.AlignNone)
	}

	if net_ipv4_cfg, err := network_info.GetPropertyIP4Config(); err == nil {
		net_ipv4_addrs, _ := net_ipv4_cfg.GetPropertyAddresses()
		net_ipv4_addr := net_ipv4_addrs[0].Address
		net_ipv4_addr += "/" + fmt.Sprint(net_ipv4_addrs[0].Prefix)
		// instance.parent.Get("WiFi_IP").(string)
		display.DrawTextAligned(0, 65, font, net_ipv4_addr, false, sh1107.AlignRight, sh1107.AlignNone)
		display.DrawTextAligned(0, 75, font, net_ipv4_addrs[0].Gateway, false, sh1107.AlignRight, sh1107.AlignNone)
	}

	display.SetColor(sh1107.White)
	display.SetLineWidth(1)
	display.DrawLine(0, 33, 127, 33)
	display.Stroke()

	font = display.Use_Font8_Bold()
	display.DrawTextAligned(64, 105, font, "Return", false, sh1107.AlignCenter, sh1107.AlignNone)

	display.Render()
}

func (m *Menu) NewSettingsMenu() *SettingsMenu {

	return &SettingsMenu{
		parent:            m,
		process_selection: false,
		selection_path:    []string{},
		options: [][]string{
			{"Network Settings",
				"Current status",
				"Toggle WiFi",
				"Toggle Mobile Data",
				"Join WiFi network",
				"Saved WiFi networks",
			},
			{"Bluetooth Settings",
				"Toggle Bluetooth",
				"Connected devices",
				"Pair new device",
				"Paired devices",
			},
			{"Call Settings",
				"Automatic Redial",
				"Speed Redialing",
				"Call Waiting Options",
				"Own Number Sending",
				"Phone Line In Use",
				"Automatic Answer",
			},
			{"Phone Settings",
				"Language",
				"Cell Info Display",
				"Welcome Note",
				"Network Selection",
				"Lights",
			},
			{"Security Settings",
				"PIN code request",
				"Call barring service",
				"Fixed dialing",
				"Closed user group",
				"Phone security",
				"Change access codes",
			},
			{"SSH Service"},
			{"About"},
			{"Factory Reset"},
		},
	}
}

func (instance *SettingsMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

func (instance *SettingsMenu) ConfigureWithArgs(args ...any) {

	// Check if we have args
	if len(args) > 0 {

		// Most likely our arg is a SelectorReturn from the selector.
		selection, ok := args[0].(*SelectorReturn)
		if !ok {
			panic("(*SettingsMenu).ConfigureWithArgs() Type error: argument must be a *SelectorReturn type")
		}

		instance.process_selection = true
		instance.selection_path = selection.SelectionPath
	}

	instance.Configure()
}

func (instance *SettingsMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*SettingsMenu).Run() before (*SettingsMenu).Configure()!")
	}

	log.Println("⚙️ Settings started")

	if !instance.process_selection {
		// Start the selector with the base settings menu
		log.Println("⚙️ Settings switching to selector")
		go instance.parent.PushWithArgs("selector", &SelectorArgs{
			Title:                 "Settings",
			Options:               instance.options,
			ButtonLabel:           "Select",
			VisibleRows:           3,
			ShowPathInTitle:       true,
			ShowElemNumberInTitle: true,
			PersistLastState:      true,
		})
		return
	}

	log.Println("⚙️ Settings path selected: ", instance.selection_path)

	if len(instance.selection_path) == 0 {
		log.Println("⚙️ Settings path selected is empty, exiting...")
		go instance.parent.Pop()
		return
	}

	// TODO: process selected setting option

	switch instance.selection_path[len(instance.selection_path)-1] {

	case "About":
		log.Println("⚙️ Showing About screen...")
		instance.RenderAbout()
	about:
		for {
			select {
			case <-instance.ctx.Done():
				return
			case evt := <-instance.parent.KeypadEvents:
				if !evt.State {
					continue
				}

				instance.parent.Timers["keypad"].Reset()
				instance.parent.Timers["oled"].Reset()
				instance.parent.Display.On()
				misc.KeyLightsOn()
				go instance.parent.PlayKey()

				switch evt.Key {
				case 'P':
					go instance.parent.Push("power")
					return
				case 'S':
					// TODO: check for updates
				case 'C':
					break about
				}
			}
		}

	case "Toggle WiFi":
		log.Println("⚙️ Toggling WiFi...")
		state, err := instance.parent.NetworkManager.GetPropertyWirelessEnabled()
		if err != nil {
			panic(err.Error())
		}

		if !instance.parent.Get("DebugMode").(bool) {
			if state {
				instance.parent.RenderAlert("ok", []string{"Turning", "WiFi", "off"})
			} else {
				instance.parent.RenderAlert("ok", []string{"Turning", "WiFi", "on"})
			}
		} else {
			instance.parent.RenderAlert("ok", []string{"Debug", "mode", "failsafe!"})
		}
		go instance.parent.PlayAlert()

		// Don't accidentally disable WiFi if we're in debug mode
		if !instance.parent.Get("DebugMode").(bool) {
			instance.parent.NetworkManager.SetPropertyWirelessEnabled(!state)
		}
		time.Sleep(2 * time.Second)

	case "Current status":
		log.Println("⚙️ Showing Network Status screen...")
		instance.RenderNetworkStatus(instance.GetNetworkState(), instance.GetNetworkInfo())
	net_status:
		for {
			select {
			case <-instance.ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
				instance.RenderNetworkStatus(instance.GetNetworkState(), instance.GetNetworkInfo())
			case evt := <-instance.parent.KeypadEvents:
				if !evt.State {
					continue
				}

				instance.parent.Timers["keypad"].Reset()
				instance.parent.Timers["oled"].Reset()
				instance.parent.Display.On()
				misc.KeyLightsOn()
				go instance.parent.PlayKey()

				switch evt.Key {
				case 'P':
					go instance.parent.Push("power")
					return
				case 'S':
					break net_status
				case 'C':
					break net_status
				}
			}
		}

	case "Join network":
		// TODO

	case "Saved networks":
		// TODO

	case "Factory Reset":
		// TODO
	}

	instance.process_selection = false
	log.Println("⚙️ Settings switching back to selector")
	go instance.parent.PushWithArgs("selector", &SelectorArgs{
		Title:                 "Settings",
		Options:               instance.options,
		ButtonLabel:           "Select",
		VisibleRows:           3,
		ShowPathInTitle:       true,
		ShowElemNumberInTitle: true,
		PersistLastState:      true,
	})
}

func (instance *SettingsMenu) Pause() {
	instance.process_selection = true
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Settings handler pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *SettingsMenu) Stop() {
	instance.process_selection = false
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Settings handler stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		go instance.cleanup()
	}
}

func (instance *SettingsMenu) cleanup() {
	instance.process_selection = false
	instance.selection_path = []string{}
}

func (instance *SettingsMenu) GetNetworkState() string {
	state, _ := instance.parent.NetworkManager.GetPropertyState()
	var state_msg string

	switch state {
	case gonetworkmanager.NmStateAsleep:
		// Not connected
		state_msg = "No connection"

	case gonetworkmanager.NmStateConnecting:
		// Connection in progress
		state_msg = "Connection in progress"

	case gonetworkmanager.NmStateUnknown:
		// Unknown
		state_msg = "Unknown status"

	case gonetworkmanager.NmStateDisconnecting:
		// Disconnection in progress
		state_msg = "Disconnect in progress"

	case gonetworkmanager.NmStateConnectedLocal:
		// Connected but no working internet connection
		state_msg = "Connected, no internet"

	case gonetworkmanager.NmStateConnectedSite:
		// Connected but no working internet connection (routes are present, however)
		state_msg = "Connected, no internet"

	case gonetworkmanager.NmStateConnectedGlobal:
		// Connected
		state_msg = "Connected"

	case gonetworkmanager.NmStateDisconnected:
		// Not connected
		state_msg = "No connection"
	}
	return state_msg
}

func (instance *SettingsMenu) GetNetworkInfo() gonetworkmanager.ActiveConnection {
	wifi_network, err := instance.parent.NetworkManager.GetPropertyPrimaryConnection()
	if err != nil {
		panic(err)
	}
	return wifi_network
}
