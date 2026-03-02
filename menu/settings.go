package menu

import (
	"context"
	"fmt"
	"lcd"
	"log"
	"misc"
	"os/exec"
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
	ctx            context.Context
	configured     bool
	cancelFn       context.CancelFunc
	parent         *Menu
	wg             sync.WaitGroup
	options        [][]string
	adapter        *bluetooth.Adapter
	ap_cache       map[string]gonetworkmanager.AccessPoint
	conn_cache     map[string]gonetworkmanager.Connection
	bt_cache       map[string]string
	current_target string
	default_args   *SelectorArgs
}

// RenderAbout renders the about screen, which displays the logo and version
// of the Rakian OS. It also shows a line for checking for updates.
func (instance *SettingsMenu) RenderAbout() {
	m := instance.parent
	display := m.Display

	display.Clear(lcd.White)
	m.RenderHeader("About")

	font := display.Use_Font_Small_Plain()
	display.DrawTextAligned(0, 10, font, "Rakian OS", false, lcd.AlignNone, lcd.AlignNone)
	display.DrawTextAligned(0, 19, font, fmt.Sprintf("v%s", m.Get("FirmwareVersion").(string)), false, lcd.AlignNone, lcd.AlignNone)
	display.DrawTextAligned(0, 28, font, misc.GetOSVersion(), false, lcd.AlignNone, lcd.AlignNone)

	m.RenderFooter("Return", true)
	display.Render()
}

// RenderInternetStatus renders the internet status screen, which displays the
// current network status message and the network information such as
// the WiFi SSID and the IP address of the network connection. It
// also shows the "Return" button.
func (instance *SettingsMenu) RenderInternetStatus(state_msg string, network_info gonetworkmanager.ActiveConnection) {
	m := instance.parent
	display := m.Display

	display.Clear(lcd.White)
	m.RenderHeader("Internet Status")

	font := display.Use_Font_Small_Plain()
	display.DrawTextAligned(0, 8, font, state_msg, false, lcd.AlignNone, lcd.AlignNone)

	if net_conn, err := network_info.GetPropertyID(); err == nil {
		display.DrawTextAligned(0, 16, font, net_conn, false, lcd.AlignNone, lcd.AlignNone)
	}

	if net_ipv4_cfg, err := network_info.GetPropertyIP4Config(); err == nil {
		net_ipv4_addrs, err := net_ipv4_cfg.GetPropertyAddresses()
		if err == nil {
			net_ipv4_addr := net_ipv4_addrs[0].Address
			net_ipv4_addr += "/" + fmt.Sprint(net_ipv4_addrs[0].Prefix)
			display.DrawTextAligned(0, 24, font, net_ipv4_addr, false, lcd.AlignNone, lcd.AlignNone)
			display.DrawTextAligned(0, 32, font, net_ipv4_addrs[0].Gateway, false, lcd.AlignNone, lcd.AlignNone)
		}
	}

	m.RenderFooter("Return", true)
	display.Render()
}

// NewSettingsMenu returns a new SettingsMenu instance with the given parent and default settings.
func (m *Menu) NewSettingsMenu() *SettingsMenu {

	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		log.Println("Warning: failed to enable bluetooth adapter:", err)
	}

	return &SettingsMenu{
		parent:     m,
		adapter:    adapter,
		ap_cache:   make(map[string]gonetworkmanager.AccessPoint),
		conn_cache: make(map[string]gonetworkmanager.Connection),
		bt_cache:   make(map[string]string),
		default_args: &SelectorArgs{
			SelectionClass: "settings.main",
			Title:          "Settings",
			ShowTitle:      true,
			SelectorType:   SELECTOR_MULTI_3,
			Options: [][]string{
				{"Internet status"},
				{"WLAN",
					"Toggle",
					"Join network",
					"Saved networks",
				},
				{"Cellular",
					"Toggle",
					"Select network",
					"Configure APN",
				},
				{"Bluetooth",
					"Toggle",
					"Pair device",
					"Saved devices",
				},
				{"Call Settings",
					"Auto redial",
					"Auto answer",
					"Speed dial",
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
				{"About"},
				{"Factory Reset"},
			},
			ShowPathInTitle:  true,
			PersistLastState: true,
			ButtonLabel:      "Select",
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

// SettingsMain is the main entry point for the settings menu.
// It will switch on the last element of the selection path and
// call the corresponding method based on the selection class and element.
// For example, if the selection path is ["Phone Settings", "Language"],
// it will call the Language method.
func (instance *SettingsMenu) SettingsMain(selection_path []string) int {
	switch selection_path[len(selection_path)-1] {

	case "About":
		return instance.handleAbout()

	case "Internet status":
		return instance.handleInternetStatus()

	case "Join network":
		return instance.handleJoinNetwork()

	case "Saved networks":
		return instance.handleSavedNetworks()

	case "Toggle":
		return instance.handleToggle(selection_path)

	case "Pair device":
		return instance.handlePairDevice()

	case "Saved devices":
		return instance.handleSavedBluetooth()

	case "Factory Reset":
		// TODO

	case "Configure APN":
		return instance.handleConfigureAPN()
	}

	return SettingsActionShowSelector
}

func (instance *SettingsMenu) handleAbout() int {
	log.Println("⚙️ Showing About screen...")
	instance.RenderAbout()
	for {
		select {
		case <-instance.ctx.Done():
			return SettingsActionSubmenuPushed
		case evt := <-instance.parent.KeypadEvents:
			if !evt.State {
				continue
			}

			instance.parent.Timers["keypad"].Reset()
			instance.parent.Timers["screensaver"].Reset()
			instance.parent.Display.On()
			misc.KeyLightsOn()
			go instance.parent.PlayKey()

			switch evt.Key {
			case 'P':
				go instance.parent.Push("power")
				return SettingsActionSubmenuPushed
			case 'S', 'C':
				return SettingsActionShowSelector
			}
		}
	}
}

func (instance *SettingsMenu) handleInternetStatus() int {
	log.Println("⚙️ Showing internet status screen...")
	instance.RenderInternetStatus(instance.GetNetworkState(), instance.GetNetworkInfo())
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
			instance.parent.Timers["screensaver"].Reset()
			instance.parent.Display.On()
			misc.KeyLightsOn()
			go instance.parent.PlayKey()

			switch evt.Key {
			case 'P':
				go instance.parent.Push("power")
				return SettingsActionSubmenuPushed
			case 'S', 'C':
				return SettingsActionShowSelector
			}
		}
	}
}

func (instance *SettingsMenu) handleJoinNetwork() int {
	log.Println("⚙️ Scanning for networks...")
	state, err := instance.parent.NetworkManager.GetPropertyWirelessEnabled()
	if err != nil {
		panic(err.Error())
	}
	if !state {
		instance.parent.RenderAlert("prohibited", []string{"WLAN", "currently", "off"})
		go instance.parent.PlayAlert()
		time.Sleep(2 * time.Second)
		return SettingsActionShowSelector
	}

	instance.parent.RenderAlert("info", []string{"Scanning", "for", "networks..."})

	// Request a scan
	if instance.parent.WifiDevice != nil {
		go instance.parent.WifiDevice.RequestScan()
		// Wait a moment for scan results
		time.Sleep(3 * time.Second)

		aps, err := instance.parent.WifiDevice.GetPropertyAccessPoints()
		if err != nil {
			log.Println("Error getting APs:", err)
			instance.parent.RenderAlert("prohibited", []string{"Scan", "failed"})
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

		selection := instance.parent.ShowSelector(SelectorArgs{
			SelectionClass:   "settings.wifi_join",
			Title:            "Join network",
			ShowTitle:        true,
			SelectorType:     SELECTOR_MULTI_3,
			Options:          options,
			ButtonLabel:      "Join",
			PersistLastState: true,
		}, instance.ctx)
		if selection != nil {
			instance.handleWifiJoin(selection.SelectionPath)
		}
		return SettingsActionShowSelector
	} else {
		instance.parent.RenderAlert("prohibited", []string{"WiFi", "device", "error"})
		time.Sleep(2 * time.Second)
	}
	return SettingsActionShowSelector
}

func (instance *SettingsMenu) handleToggle(selection_path []string) int {
	animCtx, animCancel := context.WithCancel(instance.ctx)
	switch selection_path[0] {
	case "WLAN":
		log.Println("⚙️ Toggling WiFi...")
		state, err := instance.parent.NetworkManager.GetPropertyWirelessEnabled()
		if err != nil {
			panic(err.Error())
		}

		if !instance.parent.Get("DebugMode").(bool) {
			if state {
				instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"Turning", "WLAN", "off"})
			} else {
				instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"Turning", "WLAN", "on"})
			}
		} else {
			instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"Debug", "mode", "failsafe!"})
		}
		go instance.parent.PlayAccepted()

		// Don't accidentally disable WiFi if we're in debug mode
		if !instance.parent.Get("DebugMode").(bool) {
			go instance.parent.NetworkManager.SetPropertyWirelessEnabled(!state)
		}
		time.Sleep(2 * time.Second)

	case "Cellular":
		log.Println("⚙️ Toggling cellular data...")

		apn, _ := instance.parent.Get("APN").(string)
		enabled := misc.CheckCellularData(apn)
		if enabled {
			instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"Turning", "data", "off"})
		} else {
			instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"Turning", "data", "on"})
		}
		go misc.ToggleCellularData(apn)
		go instance.parent.PlayAccepted()
		time.Sleep(2 * time.Second)

	case "Bluetooth":
		log.Println("⚙️ Toggling Bluetooth...")
		enabled := misc.IsBluetoothEnabled()
		if enabled {
			instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"Turning", "Bluetooth", "off"})
		} else {
			instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"Turning", "Bluetooth", "on"})
		}
		go instance.parent.PlayAccepted()
		if enabled {
			exec.Command("bluetoothctl", "power", "off").Run()
		} else {
			exec.Command("bluetoothctl", "power", "on").Run()
		}
		time.Sleep(2 * time.Second)
	}
	animCancel()
	return SettingsActionShowSelector
}

func (instance *SettingsMenu) handlePairDevice() int {
	log.Println("⚙️ Scanning for devices...")

	if !misc.IsBluetoothEnabled() {
		instance.parent.RenderAlert("prohibited", []string{"Bluetooth", "currently", "off"})
		go instance.parent.PlayAlert()
		time.Sleep(2 * time.Second)
		return SettingsActionShowSelector
	}

	// Temporarily stop timeouts for oled (prevent sleep mode from happening)
	instance.parent.Timers["screensaver"].Stop()
	instance.parent.Timers["keypad"].Stop()
	misc.KeyLightsOn()

	var found_devices [][]string

	instance.bt_cache = make(map[string]string)
	instance.parent.RenderAlert("info", []string{"Scanning", "for", "devices..."})

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
	instance.parent.Timers["screensaver"].Restart()
	instance.parent.Timers["keypad"].Restart()

	if len(found_devices) == 0 {
		instance.parent.RenderAlert("info", []string{"No", "devices", "found."})
		time.Sleep(2 * time.Second)
		return SettingsActionShowSelector
	}

	log.Println("⚙️ Settings switching to device pair selector")
	selection := instance.parent.ShowSelector(SelectorArgs{
		SelectionClass:   "settings.btpair",
		Title:            "Pair Device",
		ShowTitle:        true,
		SelectorType:     SELECTOR_MULTI_3,
		Options:          found_devices,
		ButtonLabel:      "Pair",
		PersistLastState: false,
	}, instance.ctx)
	if selection != nil {
		instance.BluetoothPair(selection.SelectionPath)
	}
	return SettingsActionShowSelector
}

func (instance *SettingsMenu) handleConfigureAPN() int {
	log.Println("⚙️ Configuring APN...")
	newAPN := instance.parent.InputPrompt(InputPromptArgs{Title: "Enter APN"}, instance.ctx)
	if newAPN != "" {

		// Check if currently connected
		apn, _ := instance.parent.Get("APN").(string)
		connected := misc.CheckCellularData(apn)
		if connected {
			// Disconnect from current APN
			instance.parent.RenderAlert("info", []string{"Please", "wait..."})
			misc.SetCellularDataState(apn, false)
		}

		// Save new APN
		instance.parent.Set("APN", newAPN)
		go instance.parent.SyncPersistent()

		// Connect to new APN
		if connected {
			misc.SetCellularDataState(newAPN, true)
		}

		// Notify user
		animCtx, animCancel := context.WithCancel(instance.ctx)
		instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"APN", "saved"})
		go instance.parent.PlayAccepted()
		time.Sleep(2 * time.Second)
		animCancel()
	}
	return SettingsActionShowSelector
}

func (instance *SettingsMenu) handleWifiJoin(selection_path []string) {
	if len(selection_path) > 0 {
		ssid := selection_path[0]
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
				password = instance.parent.InputPrompt(InputPromptArgs{Title: "Input password"}, instance.ctx)
				if password == "" {
					// User cancelled
					return
				}
			}
		}

		instance.parent.RenderAlert("info", []string{"Please", "wait..."})

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
		animCtx, animCancel := context.WithCancel(instance.ctx)
		if err != nil {
			log.Println("Connection error:", err)
			instance.parent.RenderAlert("prohibited", []string{"Failed to", "create", "connection"})
		} else {
			instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"Done"})
		}
		time.Sleep(2 * time.Second)
		animCancel()
	}
}

func (instance *SettingsMenu) handleWifiSavedSelection(selection_path []string) {
	if len(selection_path) > 0 {
		instance.current_target = selection_path[0]

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
		selection := instance.parent.ShowSelector(SelectorArgs{
			SelectionClass:   "settings.wifi_saved_action",
			Title:            instance.current_target,
			ShowTitle:        true,
			SelectorType:     SELECTOR_MULTI_3,
			Options:          [][]string{{action}, {"Forget"}},
			ButtonLabel:      "Select",
			PersistLastState: false,
		}, instance.ctx)
		if selection != nil {
			instance.handleWifiSavedAction(selection.SelectionPath)
		}
	}
}

func (instance *SettingsMenu) handleWifiSavedAction(selection_path []string) {
	if len(selection_path) > 0 && instance.current_target != "" {
		action := selection_path[0]
		conn, ok := instance.conn_cache[instance.current_target]

		if ok {
			animCtx, animCancel := context.WithCancel(instance.ctx)
			switch action {
			case "Forget":
				err := conn.Delete()
				if err == nil {
					instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"Network", "forgotten"})
				} else {
					instance.parent.RenderAlert("prohibited", []string{"Error", "forgetting"})
				}
			case "Connect":
				if state, err := instance.parent.NetworkManager.GetPropertyWirelessEnabled(); err != nil {
					panic(err.Error())
				} else if !state {
					instance.parent.RenderAlert("prohibited", []string{"WLAN", "currently", "off"})
					go instance.parent.PlayAlert()
				} else {
					instance.parent.RenderAlert("info", []string{"Please", "wait..."})
					_, err := instance.parent.NetworkManager.ActivateConnection(conn, instance.parent.WifiDevice, nil)
					if err != nil {
						instance.parent.RenderAlert("prohibited", []string{"Connect", "failed"})
					} else {
						instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"Done"})
					}
				}
			case "Disconnect":
				if state, err := instance.parent.NetworkManager.GetPropertyWirelessEnabled(); err != nil {
					panic(err.Error())
				} else if !state {
					instance.parent.RenderAlert("prohibited", []string{"WLAN", "currently", "off"})
					go instance.parent.PlayAlert()
				} else {
					instance.parent.RenderAlert("info", []string{"Please", "wait..."})
					activeConns, _ := instance.parent.NetworkManager.GetPropertyActiveConnections()
					for _, ac := range activeConns {
						id, _ := ac.GetPropertyID()
						if id == instance.current_target {
							err := instance.parent.NetworkManager.DeactivateConnection(ac)
							if err != nil {
								instance.parent.RenderAlert("prohibited", []string{"Disconnect", "failed"})
							} else {
								instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"Done"})
							}
							break
						}
					}
				}
			}
			time.Sleep(2 * time.Second)
			animCancel()
		}
		instance.current_target = ""
	}
}

func (instance *SettingsMenu) handleBtSavedSelection(selection_path []string) {
	if len(selection_path) > 0 {
		instance.current_target = selection_path[0]

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

		selection := instance.parent.ShowSelector(SelectorArgs{
			SelectionClass:   "settings.bt_saved_action",
			Title:            instance.current_target,
			ShowTitle:        true,
			SelectorType:     SELECTOR_MULTI_3,
			Options:          [][]string{{action}, {"Forget"}},
			ButtonLabel:      "Select",
			PersistLastState: false,
		}, instance.ctx)
		if selection != nil {
			instance.handleBtSavedAction(selection.SelectionPath)
		}
	}
}

func (instance *SettingsMenu) handleBtSavedAction(selection_path []string) {
	if len(selection_path) > 0 && instance.current_target != "" {
		action := selection_path[0]
		mac, ok := instance.bt_cache[instance.current_target]

		if ok {
			animCtx, animCancel := context.WithCancel(instance.ctx)
			switch action {
			case "Forget":
				exec.Command("bluetoothctl", "remove", mac).Run()
				instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"Done"})
			case "Connect":
				instance.parent.RenderAlert("info", []string{"Please", "wait..."})
				if err := exec.Command("bluetoothctl", "connect", mac).Run(); err != nil {
					instance.parent.RenderAlert("prohibited", []string{"Connect", "failed"})
				} else {
					instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"Done"})
				}
			case "Disconnect":
				instance.parent.RenderAlert("info", []string{"Please", "wait..."})
				if err := exec.Command("bluetoothctl", "disconnect", mac).Run(); err != nil {
					instance.parent.RenderAlert("prohibited", []string{"connect", "failed"})
				} else {
					instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"Done"})
				}
			}
			time.Sleep(2 * time.Second)
			animCancel()
		}
		instance.current_target = ""
	}
}

func (instance *SettingsMenu) Run() {
	if !instance.configured {
		panic("Attempted to call (*SettingsMenu).Run() before (*SettingsMenu).Configure()!")
	}

	log.Println("⚙️ Settings started")

	for {
		selection := instance.parent.ShowSelector(*instance.default_args, instance.ctx)
		if selection == nil {
			instance.parent.Pop()
			return
		}

		if selection.SelectionClass == "settings.main" {
			action := instance.SettingsMain(selection.SelectionPath)
			switch action {
			case SettingsActionExit:
				instance.parent.Pop()
				return
			case SettingsActionSubmenuPushed:
				// Do nothing
			}
		}
	}
}

func (instance *SettingsMenu) ConfigureWithArgs(args ...any) {
	instance.Configure()
}

// Helper to handle the saved network action since we need state persistence
// We will modify the `settings.wifi_saved` case to store the selection.
// And we need to modify the struct to hold `current_target`.

func (instance *SettingsMenu) Pause() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Settings handler pause timed out — goroutines may be stuck")
		// Optional: escalate here
	}
}

func (instance *SettingsMenu) Stop() {
	instance.cancelFn()
	if ok := waitWithTimeout(&instance.wg, 1*time.Second); !ok {
		log.Println("⚠️ Settings handler stop timed out — goroutines may be stuck")
		// Optional: escalate here
	} else {
		go instance.cleanup()
	}
}

func (instance *SettingsMenu) cleanup() {
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
		state_msg = "Connecting"

	case gonetworkmanager.NmStateUnknown:
		// Unknown
		state_msg = "Unknown"

	case gonetworkmanager.NmStateDisconnecting:
		// Disconnection in progress
		state_msg = "Disconnecting"

	case gonetworkmanager.NmStateConnectedLocal:
		// Connected but no working internet connection
		state_msg = "Connected (Local only)"

	case gonetworkmanager.NmStateConnectedSite:
		// Connected but no working internet connection (routes are present, however)
		state_msg = "Connected (Site only)"

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

// BluetoothPair is a helper function for SettingsMenu that handles
// the pairing process after a user has selected a device to pair
// with. It is called when the user selects a device from the
// list of available devices. It will initiate the pairing process
// and then re-render the main menu. For now, it just confirms
// the selection and re-renders the main menu.
func (instance *SettingsMenu) BluetoothPair(selection_path []string) {
	if len(selection_path) > 0 {
		selection := selection_path[0]
		log.Println("⚙️ Selected device:", selection)

		mac, ok := instance.bt_cache[selection]
		if !ok {
			instance.parent.RenderAlert("prohibited", []string{"Device", "not found"})
			go instance.parent.PlayAlert()
			time.Sleep(2 * time.Second)
			return
		}

		instance.parent.RenderAlert("info", []string{"Pairing", "..."})

		// Attempt to pair, trust, and connect
		exec.Command("bluetoothctl", "pair", mac).Run()
		exec.Command("bluetoothctl", "trust", mac).Run()
		err := exec.Command("bluetoothctl", "connect", mac).Run()
		animCtx, animCancel := context.WithCancel(instance.parent.GlobalContext)
		if err != nil {
			log.Println("BT Connect error:", err)
			instance.parent.RenderAlert("prohibited", []string{"Connection", "failed"})
			go instance.parent.PlayAlert()
		} else {
			instance.parent.RenderAnimatedAlert("ok", animCtx, []string{"Connected"})
		}
		time.Sleep(2 * time.Second)
		animCancel()
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

	selection := instance.parent.ShowSelector(SelectorArgs{
		SelectionClass:   "settings.wifi_saved",
		Title:            "Saved networks",
		ShowTitle:        true,
		SelectorType:     SELECTOR_MULTI_3,
		Options:          options,
		ButtonLabel:      "Select",
		PersistLastState: false,
	}, instance.ctx)
	if selection != nil {
		instance.handleWifiSavedSelection(selection.SelectionPath)
	}
	return SettingsActionShowSelector
}

func (instance *SettingsMenu) handleSavedBluetooth() int {
	out, err := exec.Command("bluetoothctl", "devices", "Paired").Output()
	if err != nil {
		instance.parent.RenderAlert("prohibited", []string{"Error", "listing"})
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

	selection := instance.parent.ShowSelector(SelectorArgs{
		SelectionClass:   "settings.bt_saved",
		Title:            "Saved devices",
		ShowTitle:        true,
		SelectorType:     SELECTOR_MULTI_3,
		Options:          options,
		ButtonLabel:      "Select",
		PersistLastState: false,
	}, instance.ctx)
	if selection != nil {
		instance.handleBtSavedSelection(selection.SelectionPath)
	}
	return SettingsActionShowSelector
}
