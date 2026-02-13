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
	"tinygo.org/x/bluetooth"
)

const (
	SettingsActionExit = iota
	SettingsActionShowSelector
	SettingsActionSubmenuPushed
)

type SettingsMenu struct {
	ctx               context.Context
	configured        bool
	cancelFn          context.CancelFunc
	parent            *Menu
	wg                sync.WaitGroup
	process_selection bool
	selection_class   string
	selection_path    []string
	options           [][]string
	adapter           *bluetooth.Adapter
}

// RenderAbout renders the about screen, which displays the logo and version
// of the Rakian OS. It also shows a line for checking for updates.
func (instance *SettingsMenu) RenderAbout() {
	m := instance.parent
	display := m.Display

	display.Clear(sh1107.Black)

	display.DrawImageAligned(m.Sprites["logo"], 60, 50, sh1107.AlignCenter, sh1107.AlignCenter)

	font := display.Use_Font8_Normal()
	display.DrawTextAligned(0, 20, font, "About", false, sh1107.AlignRight, sh1107.AlignNone)
	display.DrawTextAligned(60, 60, font, "Rakian OS", false, sh1107.AlignCenter, sh1107.AlignNone)
	display.DrawTextAligned(60, 70, font, fmt.Sprintf("v%s", m.Get("FirmwareVersion").(string)), false, sh1107.AlignCenter, sh1107.AlignNone)
	display.DrawTextAligned(60, 80, font, misc.GetOSVersion(), false, sh1107.AlignCenter, sh1107.AlignNone)

	display.SetColor(sh1107.White)
	display.SetLineWidth(1)
	display.DrawLine(0, 33, 127, 33)
	display.Stroke()

	font = display.Use_Font8_Bold()
	display.DrawTextAligned(64, 105, font, "Check for updates", false, sh1107.AlignCenter, sh1107.AlignNone)

	display.Render()
}

// RenderInternetStatus renders the internet status screen, which displays the
// current network status message and the network information such as
// the WiFi SSID and the IP address of the network connection. It
// also shows the "Return" button.
func (instance *SettingsMenu) RenderInternetStatus(state_msg string, network_info gonetworkmanager.ActiveConnection) {
	m := instance.parent
	display := m.Display

	display.Clear(sh1107.Black)

	font := display.Use_Font8_Normal()
	display.DrawTextAligned(0, 20, font, "Internet status", false, sh1107.AlignRight, sh1107.AlignNone)
	display.DrawTextAligned(0, 40, font, state_msg, false, sh1107.AlignRight, sh1107.AlignNone)

	if net_conn, err := network_info.GetPropertyID(); err == nil {
		// instance.parent.Get("WiFi_SSID").(string)
		display.DrawTextAligned(0, 55, font, net_conn, false, sh1107.AlignRight, sh1107.AlignNone)
	}

	if net_ipv4_cfg, err := network_info.GetPropertyIP4Config(); err == nil {
		net_ipv4_addrs, err := net_ipv4_cfg.GetPropertyAddresses()
		if err == nil {
			net_ipv4_addr := net_ipv4_addrs[0].Address
			net_ipv4_addr += "/" + fmt.Sprint(net_ipv4_addrs[0].Prefix)
			// instance.parent.Get("WiFi_IP").(string)
			display.DrawTextAligned(0, 65, font, net_ipv4_addr, false, sh1107.AlignRight, sh1107.AlignNone)
			display.DrawTextAligned(0, 75, font, net_ipv4_addrs[0].Gateway, false, sh1107.AlignRight, sh1107.AlignNone)
		}
	}

	display.SetColor(sh1107.White)
	display.SetLineWidth(1)
	display.DrawLine(0, 33, 127, 33)
	display.Stroke()

	font = display.Use_Font8_Bold()
	display.DrawTextAligned(64, 105, font, "Return", false, sh1107.AlignCenter, sh1107.AlignNone)

	display.Render()
}

// NewSettingsMenu returns a new SettingsMenu instance with the given parent and default settings.
func (m *Menu) NewSettingsMenu() *SettingsMenu {

	// Init bluetooth
	adapter := bluetooth.DefaultAdapter
	err := adapter.Enable()
	if err != nil {
		panic(err)
	}

	return &SettingsMenu{
		parent:            m,
		adapter:           adapter,
		process_selection: false,
		selection_path:    []string{},
		options: [][]string{
			{"Internet status"},
			{"WiFi Settings",
				"Toggle WiFi",
				"Join network",
				"Saved networks",
			},
			{"Cellular Settings",
				"Toggle data",
				"Network selection",
				"Configure APN",
			},
			{"Bluetooth Settings",
				"Toggle Bluetooth",
				"Pair device",
				"Saved devices",
			},
			{"Call Settings",
				"Automatic Redial",
				"Automatic Answer",
				"Speed Dialing",
			},
			{"Phone Settings",
				"Language",
				"Cell Info Display",
				"Welcome Note",
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

// Configure resets the context and prepares the menu to be run. It should
// be called before running the menu. It will panic if the menu is
// already configured.
func (instance *SettingsMenu) Configure() {
	// Reset context
	instance.configured = true
	instance.ctx, instance.cancelFn = context.WithCancel(instance.parent.GlobalContext)
}

// ConfigureWithArgs configures the SettingsMenu with the given arguments.
// The first argument must be a SelectorReturn type, which contains the selection path and class.
// If the first argument is not a SelectorReturn type, a panic will occur.
// After calling ConfigureWithArgs, the SettingsMenu must be configured before calling Run().
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
		instance.selection_class = selection.SelectionClass
	}

	instance.Configure()
}

// SettingsMain is the main entry point for the settings menu.
// It will switch on the last element of the selection path and
// call the corresponding method based on the selection class and element.
// For example, if the selection path is ["Phone Settings", "Language"],
// it will call the Language method.
func (instance *SettingsMenu) SettingsMain(selection_path []string) int {
	switch selection_path[len(selection_path)-1] {

	case "About":
		log.Println("⚙️ Showing About screen...")
		instance.RenderAbout()
	about:
		for {
			select {
			case <-instance.ctx.Done():
				return SettingsActionSubmenuPushed
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
					return SettingsActionSubmenuPushed
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
			go instance.parent.NetworkManager.SetPropertyWirelessEnabled(!state)
		}
		time.Sleep(2 * time.Second)

	case "Internet status":
		log.Println("⚙️ Showing internet status screen...")
		instance.RenderInternetStatus(instance.GetNetworkState(), instance.GetNetworkInfo())
	net_status:
		for {
			select {
			case <-instance.ctx.Done():
				return SettingsActionSubmenuPushed
			case <-time.After(500 * time.Millisecond):
				instance.RenderInternetStatus(instance.GetNetworkState(), instance.GetNetworkInfo())
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
					return SettingsActionSubmenuPushed
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

	case "Toggle Bluetooth":
		// TODO

	case "Pair device":
		log.Println("⚙️ Scanning for devices...")

		// Ensure any previous scan is stopped
		instance.adapter.StopScan()

		// Temporarily stop timeouts for oled (prevent sleep mode from happening)
		instance.parent.Timers["oled"].Stop()
		instance.parent.Timers["keypad"].Stop()
		misc.KeyLightsOn()

		seen := make(map[string]bool)
		var found_devices [][]string

		instance.parent.RenderAlert("info", []string{"Scanning", "for", "devices..."})
		go func() {
			time.Sleep(5 * time.Second)
			instance.adapter.StopScan()
		}()
		if err := instance.adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
			addr := device.Address.String()
			if !seen[addr] {
				seen[addr] = true
				name := device.LocalName()
				if name == "" {
					return
				}
				found_devices = append(found_devices, []string{name})
			}
		}); err != nil {
			log.Println("Scan error:", err)
		}

		// Resume timeouts
		instance.parent.Timers["oled"].Restart()
		instance.parent.Timers["keypad"].Restart()

		if len(found_devices) == 0 {
			instance.parent.RenderAlert("info", []string{"No", "devices", "found."})
			time.Sleep(2 * time.Second)
			return SettingsActionShowSelector
		}

		log.Println("⚙️ Settings switching to device pair selector")
		go instance.parent.PushWithArgs("selector", &SelectorArgs{
			SelectionClass: "settings.btpair",
			Title:          "Pair Device",
			Options:        found_devices,
			ButtonLabel:    "Pair",
			VisibleRows:    3,
		})
		return SettingsActionSubmenuPushed

	case "Factory Reset":
		// TODO
	}

	return SettingsActionShowSelector
}

// BluetoothPair is a helper function for SettingsMenu that handles
// the pairing process after a user has selected a device to pair
// with. It is called when the user selects a device from the
// list of available devices. It will initiate the pairing process
// and then re-render the main menu. For now, it just confirms
// the selection and re-renders the main menu.
func (instance *SettingsMenu) BluetoothPair(selection_path []string) {
	if len(instance.selection_path) > 0 {
		selection := instance.selection_path[0]
		log.Println("⚙️ Selected device:", selection)
		// Here we would initiate pairing with the device
		// For now, just confirm selection
		instance.parent.RenderAlert("ok", []string{"Pairing", "request", "sent"})
		go instance.parent.PlayAlert()
		time.Sleep(2 * time.Second)
	}
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
			SelectionClass:             "settings.main",
			Title:                      "Settings",
			Options:                    instance.options,
			ButtonLabel:                "Select",
			VisibleRows:                3,
			ShowPathInTitle:            true,
			ShowElemNumbersInSelection: true,
			ShowElemNumberInTitle:      true,
			AllowNumberKeyShortcut:     true,
			PersistLastState:           true,
		})
		return
	}

	log.Printf("⚙️ Settings %s: %s", instance.selection_class, instance.selection_path)

	// Process selected setting option
	switch instance.selection_class {
	case "settings.main":

		// Exit to main menu
		if len(instance.selection_path) == 0 {
			log.Println("⚙️ Settings path selected is empty, exiting...")
			go instance.parent.Pop()
			return
		}

		// Launch settings main menu
		action := instance.SettingsMain(instance.selection_path)

		switch action {
		case SettingsActionExit:
			log.Println("⚙️ Settings exiting")
			go instance.parent.Pop()
			return
		case SettingsActionSubmenuPushed:
			// Do nothing, wait for submenu to return
			return
		}

	case "settings.btpair":

		// Launch bluetooth pairing handler
		instance.BluetoothPair(instance.selection_path)
	}

	instance.process_selection = false
	log.Println("⚙️ Settings switching back to selector")
	go instance.parent.PushWithArgs("selector", &SelectorArgs{
		SelectionClass:             "settings.main",
		Title:                      "Settings",
		Options:                    instance.options,
		ButtonLabel:                "Select",
		VisibleRows:                3,
		ShowPathInTitle:            true,
		ShowElemNumbersInSelection: true,
		ShowElemNumberInTitle:      true,
		AllowNumberKeyShortcut:     true,
		PersistLastState:           true,
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
