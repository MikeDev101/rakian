package menu

import (
	"context"
	"fmt"
	"log"
	"misc"
	"os/exec"
	"sh1107"
	"sort"
	"strings"
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
	ap_cache          map[string]gonetworkmanager.AccessPoint
	conn_cache        map[string]gonetworkmanager.Connection
	bt_cache          map[string]string
	current_target    string
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
		display.DrawTextAligned(0, 55, font, net_conn, false, sh1107.AlignRight, sh1107.AlignNone)
	}

	if net_ipv4_cfg, err := network_info.GetPropertyIP4Config(); err == nil {
		net_ipv4_addrs, err := net_ipv4_cfg.GetPropertyAddresses()
		if err == nil {
			net_ipv4_addr := net_ipv4_addrs[0].Address
			net_ipv4_addr += "/" + fmt.Sprint(net_ipv4_addrs[0].Prefix)
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

	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		log.Println("Warning: failed to enable bluetooth adapter:", err)
	}

	return &SettingsMenu{
		parent:            m,
		adapter:           adapter,
		ap_cache:          make(map[string]gonetworkmanager.AccessPoint),
		conn_cache:        make(map[string]gonetworkmanager.Connection),
		bt_cache:          make(map[string]string),
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
		log.Println("⚙️ Scanning for networks...")
		instance.parent.RenderAlert("loading", []string{"Scanning", "networks..."})

		// Request a scan
		if instance.parent.WifiDevice != nil {
			go instance.parent.WifiDevice.RequestScan()
			// Wait a moment for scan results
			time.Sleep(3 * time.Second)

			aps, err := instance.parent.WifiDevice.GetPropertyAccessPoints()
			if err != nil {
				log.Println("Error getting APs:", err)
				instance.parent.RenderAlert("alert", []string{"Scan", "failed"})
				time.Sleep(2 * time.Second)
				return SettingsActionShowSelector
			}

			// Clear cache
			instance.ap_cache = make(map[string]gonetworkmanager.AccessPoint)
			var ap_names []string

			for _, ap := range aps {
				ssid, _ := ap.GetPropertySSID()
				if ssid == "" {
					continue
				}
				// Deduplicate by keeping the strongest signal?
				// For simplicity, we just overwrite, or check if exists.
				// NetworkManager usually handles the best AP for an SSID.
				instance.ap_cache[ssid] = ap

				// Check if already in list
				found := false
				for _, name := range ap_names {
					if name == ssid {
						found = true
						break
					}
				}
				if !found {
					ap_names = append(ap_names, ssid)
				}
			}
			sort.Strings(ap_names)

			var options [][]string
			for _, name := range ap_names {
				options = append(options, []string{name})
			}

			if len(options) == 0 {
				instance.parent.RenderAlert("info", []string{"No", "networks", "found"})
				time.Sleep(2 * time.Second)
				return SettingsActionShowSelector
			}

			go instance.parent.PushWithArgs("selector", &SelectorArgs{
				SelectionClass: "settings.wifi_join",
				Title:          "Join network",
				Options:        options,
				ButtonLabel:    "Join",
				VisibleRows:    3,
			})
			return SettingsActionSubmenuPushed
		} else {
			instance.parent.RenderAlert("alert", []string{"WiFi", "device", "error"})
			time.Sleep(2 * time.Second)
		}

	case "Saved networks":
		return instance.handleSavedNetworks()

	case "Toggle Bluetooth":
		if misc.IsBluetoothEnabled() {
			exec.Command("bluetoothctl", "power", "off").Run()
			instance.parent.RenderAlert("ok", []string{"Turning", "Bluetooth", "off"})
		} else {
			exec.Command("bluetoothctl", "power", "on").Run()
			instance.parent.RenderAlert("ok", []string{"Turning", "Bluetooth", "on"})
		}
		time.Sleep(2 * time.Second)

	case "Pair device":
		log.Println("⚙️ Scanning for devices...")

		// Ensure Bluetooth is on
		if !misc.IsBluetoothEnabled() {
			exec.Command("bluetoothctl", "power", "on").Run()
			time.Sleep(2 * time.Second)
		}

		// Temporarily stop timeouts for oled (prevent sleep mode from happening)
		instance.parent.Timers["oled"].Stop()
		instance.parent.Timers["keypad"].Stop()
		misc.KeyLightsOn()

		var found_devices [][]string

		instance.bt_cache = make(map[string]string)
		instance.parent.RenderAlert("loading", []string{"Scanning", "for", "devices..."})

		// Start scanning via bluetoothctl to force radio activity
		scanCmd := exec.Command("bluetoothctl", "scan", "on")
		scanCmd.Start()

		// Start scanning via tinygo/bluetooth to collect results
		go func() {
			time.Sleep(10 * time.Second)
			instance.adapter.StopScan()
			if scanCmd.Process != nil {
				scanCmd.Process.Kill()
			}
		}()

		// Collect devices
		discovered := make(map[string]string) // MAC -> Name
		err := instance.adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
			name := device.LocalName()
			mac := device.Address.String()
			if name == "" {
				// Don't include devices without names
				return
			}
			discovered[mac] = name
		})
		if err != nil {
			log.Println("Scan error:", err)
		}

		// Get paired devices to exclude
		pairedOut, _ := exec.Command("bluetoothctl", "devices", "Paired").Output()
		pairedMap := make(map[string]bool)
		for line := range strings.SplitSeq(string(pairedOut), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 2 && parts[0] == "Device" {
				pairedMap[parts[1]] = true
			}
		}

		// Retrieve list of devices
		for mac, name := range discovered {
			if pairedMap[mac] {
				continue
			}
			key := name
			if _, exists := instance.bt_cache[key]; exists {
				key = fmt.Sprintf("%s (%s)", name, mac)
			}
			instance.bt_cache[key] = mac
			found_devices = append(found_devices, []string{key})
		}
		sort.Slice(found_devices, func(i, j int) bool {
			return found_devices[i][0] < found_devices[j][0]
		})

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

	case "Saved devices":
		return instance.handleSavedBluetooth()

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

		mac, ok := instance.bt_cache[selection]
		if !ok {
			instance.parent.RenderAlert("alert", []string{"Device", "not found"})
			time.Sleep(2 * time.Second)
			return
		}

		instance.parent.RenderAlert("loading", []string{"Pairing", "..."})

		// Attempt to pair, trust, and connect
		exec.Command("bluetoothctl", "pair", mac).Run()
		exec.Command("bluetoothctl", "trust", mac).Run()
		err := exec.Command("bluetoothctl", "connect", mac).Run()
		if err != nil {
			log.Println("BT Connect error:", err)
			instance.parent.RenderAlert("alert", []string{"Connection", "failed"})
		} else {
			instance.parent.RenderAlert("ok", []string{"Connected"})
		}
		time.Sleep(2 * time.Second)
	}
}

func (instance *SettingsMenu) handleSavedNetworks() int {
	settings, err := gonetworkmanager.NewSettings()
	if err != nil {
		log.Println("Error getting settings:", err)
		return SettingsActionShowSelector
	}

	conns, err := settings.ListConnections()
	if err != nil {
		log.Println("Error listing connections:", err)
		return SettingsActionShowSelector
	}

	instance.conn_cache = make(map[string]gonetworkmanager.Connection)
	var conn_names []string

	for _, conn := range conns {
		connSettings, err := conn.GetSettings()
		if err != nil {
			continue
		}
		// Check for connection type "802-11-wireless"
		if connType, ok := connSettings["connection"]["type"].(string); ok && connType == "802-11-wireless" {
			id, ok := connSettings["connection"]["id"].(string)
			if !ok {
				continue
			}
			instance.conn_cache[id] = conn
			conn_names = append(conn_names, id)
		}
	}
	sort.Strings(conn_names)

	var options [][]string
	for _, name := range conn_names {
		options = append(options, []string{name})
	}

	if len(options) == 0 {
		instance.parent.RenderAlert("info", []string{"No", "saved", "networks"})
		time.Sleep(2 * time.Second)
		return SettingsActionShowSelector
	}

	go instance.parent.PushWithArgs("selector", &SelectorArgs{
		SelectionClass: "settings.wifi_saved",
		Title:          "Saved networks",
		Options:        options,
		ButtonLabel:    "Select",
		VisibleRows:    3,
	})
	return SettingsActionSubmenuPushed
}

func (instance *SettingsMenu) handleSavedBluetooth() int {
	out, err := exec.Command("bluetoothctl", "devices", "Paired").Output()
	if err != nil {
		instance.parent.RenderAlert("alert", []string{"Error", "listing"})
		return SettingsActionShowSelector
	}

	lines := strings.Split(string(out), "\n")
	var options [][]string
	instance.bt_cache = make(map[string]string)

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 3 && parts[0] == "Device" {
			mac := parts[1]
			name := strings.Join(parts[2:], " ")
			instance.bt_cache[name] = mac
			options = append(options, []string{name})
		}
	}

	if len(options) == 0 {
		instance.parent.RenderAlert("info", []string{"No", "saved", "devices"})
		time.Sleep(2 * time.Second)
		return SettingsActionShowSelector
	}

	go instance.parent.PushWithArgs("selector", &SelectorArgs{
		SelectionClass: "settings.bt_saved",
		Title:          "Saved devices",
		Options:        options,
		ButtonLabel:    "Select",
		VisibleRows:    3,
	})
	return SettingsActionSubmenuPushed
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

	case "settings.wifi_join":
		if len(instance.selection_path) > 0 {
			ssid := instance.selection_path[0]
			log.Println("⚙️ Joining network:", ssid)

			// Check if we have the AP info
			ap, ok := instance.ap_cache[ssid]
			password := ""

			if ok {
				// Check security flags
				flags, _ := ap.GetPropertyFlags()
				wpaFlags, _ := ap.GetPropertyWPAFlags()
				rsnFlags, _ := ap.GetPropertyRSNFlags()

				// Simple check: if any privacy flag is set, ask for password
				// NM_802_11_AP_FLAGS_PRIVACY = 0x1
				if (flags&1) != 0 || wpaFlags != 0 || rsnFlags != 0 {
					password = instance.parent.EnterText("Input password", instance.ctx)
					if password == "" {
						// User cancelled
						break
					}
				}
			}

			instance.parent.RenderAlert("loading", []string{"Connect", "in progress"})

			// Create connection settings
			connSettings := make(map[string]map[string]any)
			connSettings["connection"] = make(map[string]any)
			connSettings["connection"]["id"] = ssid
			connSettings["connection"]["type"] = "802-11-wireless"

			connSettings["802-11-wireless"] = make(map[string]any)
			connSettings["802-11-wireless"]["ssid"] = []byte(ssid)

			if password != "" {
				connSettings["802-11-wireless-security"] = make(map[string]any)
				connSettings["802-11-wireless-security"]["key-mgmt"] = "wpa-psk" // Assume WPA/WPA2 for now
				connSettings["802-11-wireless-security"]["psk"] = password
			}

			// Add and activate
			_, err := instance.parent.NetworkManager.AddAndActivateConnection(connSettings, instance.parent.WifiDevice)
			if err != nil {
				log.Println("Connection error:", err)
				instance.parent.RenderAlert("alert", []string{"Failed to", "create", "connection"})
			} else {
				instance.parent.RenderAlert("ok", []string{"Connection", "created and", "saved"})
			}
			time.Sleep(2 * time.Second)
		}

	case "settings.wifi_saved":
		if len(instance.selection_path) > 0 {
			instance.current_target = instance.selection_path[0]

			// Check if connected
			isConnected := false
			activeConns, _ := instance.parent.NetworkManager.GetPropertyActiveConnections()
			for _, ac := range activeConns {
				id, _ := ac.GetPropertyID()
				if id == instance.current_target {
					isConnected = true
					break
				}
			}

			action := "Connect"
			if isConnected {
				action = "Disconnect"
			}

			// Show options for the saved network
			go instance.parent.PushWithArgs("selector", &SelectorArgs{
				SelectionClass:   "settings.wifi_saved_action",
				Title:            instance.current_target,
				Options:          [][]string{{action}, {"Forget"}},
				ButtonLabel:      "Select",
				VisibleRows:      2,
				PersistLastState: false,
			})
			return
		}

	case "settings.wifi_saved_action":
		if len(instance.selection_path) > 0 && instance.current_target != "" {
			action := instance.selection_path[0]
			conn, ok := instance.conn_cache[instance.current_target]

			if ok {
				switch action {
				case "Forget":
					err := conn.Delete()
					if err == nil {
						instance.parent.RenderAlert("ok", []string{"Network", "forgotten"})
					} else {
						instance.parent.RenderAlert("alert", []string{"Error", "forgetting"})
					}
				case "Connect":
					instance.parent.RenderAlert("loading", []string{"Connect", "in progress"})
					_, err := instance.parent.NetworkManager.ActivateConnection(conn, instance.parent.WifiDevice, nil)
					if err != nil {
						instance.parent.RenderAlert("alert", []string{"Connection", "failed"})
					}
				case "Disconnect":
					instance.parent.RenderAlert("loading", []string{"Disconnect", "in progress"})
					activeConns, _ := instance.parent.NetworkManager.GetPropertyActiveConnections()
					for _, ac := range activeConns {
						id, _ := ac.GetPropertyID()
						if id == instance.current_target {
							err := instance.parent.NetworkManager.DeactivateConnection(ac)
							if err != nil {
								instance.parent.RenderAlert("alert", []string{"Disconnect", "failed"})
							} else {
								instance.parent.RenderAlert("ok", []string{"Disconnected"})
							}
							break
						}
					}
				}
				time.Sleep(2 * time.Second)
			}
			instance.current_target = ""
		}

	case "settings.bt_saved":
		if len(instance.selection_path) > 0 {
			instance.current_target = instance.selection_path[0]

			mac, ok := instance.bt_cache[instance.current_target]
			isConnected := false
			if ok {
				out, _ := exec.Command("bluetoothctl", "info", mac).Output()
				if strings.Contains(string(out), "Connected: yes") {
					isConnected = true
				}
			}

			action := "Connect"
			if isConnected {
				action = "Disconnect"
			}

			go instance.parent.PushWithArgs("selector", &SelectorArgs{
				SelectionClass:   "settings.bt_saved_action",
				Title:            instance.current_target,
				Options:          [][]string{{action}, {"Forget"}},
				ButtonLabel:      "Select",
				VisibleRows:      2,
				PersistLastState: false,
			})
			return
		}

	case "settings.bt_saved_action":
		if len(instance.selection_path) > 0 && instance.current_target != "" {
			action := instance.selection_path[0]
			mac, ok := instance.bt_cache[instance.current_target]

			if ok {
				switch action {
				case "Forget":
					exec.Command("bluetoothctl", "remove", mac).Run()
					instance.parent.RenderAlert("ok", []string{"Device", "forgotten"})
				case "Connect":
					instance.parent.RenderAlert("loading", []string{"Connect", "in progress"})
					if err := exec.Command("bluetoothctl", "connect", mac).Run(); err != nil {
						instance.parent.RenderAlert("alert", []string{"Connection", "failed"})
					} else {
						instance.parent.RenderAlert("ok", []string{"Connected"})
					}
				case "Disconnect":
					instance.parent.RenderAlert("loading", []string{"Disconnect", "in progress"})
					if err := exec.Command("bluetoothctl", "disconnect", mac).Run(); err != nil {
						instance.parent.RenderAlert("alert", []string{"Disconnect", "failed"})
					} else {
						instance.parent.RenderAlert("ok", []string{"Disconnected"})
					}
				}
				time.Sleep(2 * time.Second)
			}
			instance.current_target = ""
		}
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

// Helper to handle the saved network action since we need state persistence
// We will modify the `settings.wifi_saved` case to store the selection.
// And we need to modify the struct to hold `current_target`.

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
	instance.ap_cache = nil
	instance.conn_cache = nil
	instance.bt_cache = nil
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
