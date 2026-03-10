package eg25_g

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"db"
	"modem"

	"github.com/godbus/dbus/v5"
	"github.com/google/uuid"
	"github.com/maltegrosse/go-modemmanager"
	"gorm.io/gorm"
)

// EG25_G holds the channels we will use to broadcast events to the rest of our app.
type EG25_G struct {
	modemmanager.Modem
	db    *gorm.DB
	ok    bool
	debug bool

	// Mutable state
	signalStrength   int
	flightMode       bool
	carrier          string
	roaming          bool
	sos              bool
	registered       bool
	emergencyNumbers []string
	unreadVoicemails int
	microphoneMuted  bool
	ringingChan      chan *modem.Call

	// Private properties
	globalCtx                context.Context
	globalCancel             context.CancelFunc
	audio_lock               sync.Mutex
	audio_enabled            bool
	audio_hwid               string
	audioBridgeCancel        context.CancelFunc
	calls                    map[dbus.ObjectPath]*modem.Call
	calls_lock               sync.Mutex
	current_call             *modem.Call
	vmail_events             chan dbus.ObjectPath
	incoming_call_events     chan dbus.ObjectPath
	call_deleted_events      chan dbus.ObjectPath
	incoming_sms_events      chan dbus.ObjectPath
	modem_state_events       chan modemmanager.MMModemState
	modem_failure_events     chan modemmanager.MMModemStateFailedReason
	modem_power_state_events chan modemmanager.MMModemPowerState
	signal_strength_events   chan uint32
	call_state_events        chan callStateChange

	messaging    modemmanager.ModemMessaging
	messaging_ok bool
	sim          modemmanager.Sim
	sim_ok       bool
	modem3gpp    modemmanager.Modem3gpp
	modem3gpp_ok bool
}

type callStateChange struct {
	path     dbus.ObjectPath
	oldState modemmanager.MMCallState
	newState modemmanager.MMCallState
	reason   modemmanager.MMCallStateReason
}

func New(debug bool, db *gorm.DB) modem.Modem {
	ctx, cancel := context.WithCancel(context.Background())
	// Initialize
	instance := &EG25_G{
		debug:                  debug,
		db:                     db,
		ringingChan:            make(chan *modem.Call),
		calls:                  make(map[dbus.ObjectPath]*modem.Call, 0),
		incoming_call_events:   make(chan dbus.ObjectPath, 10),
		call_deleted_events:    make(chan dbus.ObjectPath, 10),
		incoming_sms_events:    make(chan dbus.ObjectPath, 10),
		modem_state_events:     make(chan modemmanager.MMModemState, 10),
		call_state_events:      make(chan callStateChange, 10),
		signal_strength_events: make(chan uint32, 10),
		signalStrength:         0,
		flightMode:             false,
		globalCtx:              ctx,
		globalCancel:           cancel,
	}

	// Connect to ModemManager
	mgr, err := modemmanager.NewModemManager()
	if err != nil {
		log.Printf("⚠️ Failed to connect to ModemManager: %v", err)
		instance.ok = false
	}

	// Get modem
	modems, err := mgr.GetModems()
	if err != nil || len(modems) == 0 {
		log.Println("⚠️ No modems found.")
		instance.ok = false

		// Return the instance gracefully
		return instance
	}

	instance.Modem = modems[0]
	instance.ok = true

	// Force enable the modem at startup
	if err := instance.Modem.Enable(); err != nil {
		log.Printf("⚠️ Failed to enable modem: %v", err)
		instance.ok = false
		return instance
	}

	// Check if the modem is in flight mode
	instance.CheckFlightMode()

	// Get the initial signal strength
	instance.CheckSignalStrength()

	// Get 3GPP interface
	instance.Get3GPP()

	// Get SIM interface
	instance.GetSIM()

	// Check carrier and roaming info
	if instance.modem3gpp_ok {
		go instance.GetCarrierAndRoaming()
	}

	// Get emergency numbers
	go instance.ReadEmergencyNumbers()

	// Get messaging interface and process any pending messages
	go instance.GetMessaging()

	// Start the background listeners
	go instance.ListenToModemEvents()
	go instance.ProcessEvents()
	go instance.ListenToVvmdEvents()

	instance.audio_hwid = instance.FindAlsaCard()

	// Return the instance with the initialized modem
	return instance
}

func (m *EG25_G) OK() bool                      { return m.ok }
func (m *EG25_G) GetSignalStrength() int        { return m.signalStrength }
func (m *EG25_G) IsFlightMode() bool            { return m.flightMode }
func (m *EG25_G) GetCarrier() string            { return m.carrier }
func (m *EG25_G) IsRoaming() bool               { return m.roaming }
func (m *EG25_G) IsSOS() bool                   { return m.sos }
func (m *EG25_G) IsRegistered() bool            { return m.registered }
func (m *EG25_G) GetEmergencyNumbers() []string { return m.emergencyNumbers }
func (m *EG25_G) GetUnreadVoicemails() int      { return m.unreadVoicemails }
func (m *EG25_G) Ringing() <-chan *modem.Call   { return m.ringingChan }
func (m *EG25_G) Stop()                         { m.globalCancel() }

func (m *EG25_G) GetSIM() {
	if !m.ok {
		return
	}

	var err error
	m.sim, err = m.Modem.GetSim()
	if err != nil {
		log.Printf("⚠️ Failed to get SIM: %v", err)
		m.sim_ok = false
	} else {
		m.sim_ok = true
	}
}

// GetMessaging gets the messaging interface and processes any pending messages.
// If the modem is not available, this function will do nothing.
// If the messaging interface is not available, this function will log an error and return.
// If there are any pending messages, this function will log a message indicating the number of messages found and save them to the database.
// If there is an error while listing the messages, this function will log a fatal error and exit.
func (m *EG25_G) GetMessaging() {
	if !m.ok {
		return
	}
	var err error
	m.messaging, err = m.Modem.GetMessaging()
	if err != nil {
		log.Printf("⚠️ Failed to get messaging interface: %v", err)
		m.messaging_ok = false
	} else {
		m.messaging_ok = true
	}

	if !m.messaging_ok {
		return
	}

	smss, err := m.messaging.List()
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("✉️ Found %d SMS", len(smss))

	for _, sms := range smss {
		path := sms.GetObjectPath()
		m.SaveSMS(sms, path)
	}
}

func (m *EG25_G) ReadEmergencyNumbers() {
	if !m.ok {
		return
	}

	if !m.sim_ok {
		return
	}

	// Fetch the emergency numbers
	// The go-modemmanager wrapper should natively support this:
	numbers, err := m.sim.GetEmergencyNumbers()
	if err != nil {
		log.Printf("⚠️ Error reading emergency numbers: %v\n", err)

		// FALLBACK
		numbers = getEmergencyNumbersViaDBus(m.sim.GetObjectPath())
	}

	if len(numbers) == 0 {
		fmt.Println("⚠️ No emergency numbers found on this SIM.")
		return
	}

	fmt.Println("Emergency Numbers found on SIM:")
	for i, num := range numbers {
		fmt.Printf("  [%d]: %s\n", i+1, num)
	}

	m.emergencyNumbers = numbers
}

// getEmergencyNumbersViaDBus is a fallback helper to grab the property
// directly from the system bus if the wrapper library is missing the method.
func getEmergencyNumbersViaDBus(simPath dbus.ObjectPath) []string {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil
	}

	obj := conn.Object("org.freedesktop.ModemManager1", simPath)
	variant, err := obj.GetProperty("org.freedesktop.ModemManager1.Sim.EmergencyNumbers")
	if err != nil {
		log.Printf("D-Bus fallback failed: %v\n", err)
		return nil
	}

	// Safely cast the D-Bus variant to a slice of strings
	if nums, ok := variant.Value().([]string); ok {
		return nums
	}
	return nil
}

// SaveSMS reads an SMS from the messaging interface, logs its contents, and saves it to the database.
// If the modem is not available or the messaging interface is not available, this function will do nothing.
// If there is an error while reading the SMS or saving it to the database, this function will log a fatal error and exit.
func (m *EG25_G) SaveSMS(sms modemmanager.Sms, smsPath dbus.ObjectPath) {
	if !m.ok {
		return
	}
	if !m.messaging_ok {
		return
	}

	// Read the SMS
	text, _ := sms.GetText()
	received, _ := sms.GetTimestamp()
	number, _ := sms.GetNumber()

	likely_vvmd := false
	likely_vvmd_prefixes := []string{"91009000", "90008000"}
	for pfx := range likely_vvmd_prefixes {
		if strings.HasPrefix(number, likely_vvmd_prefixes[pfx]) {
			likely_vvmd = true
			break
		}
	}

	if likely_vvmd || strings.HasPrefix(text, "//VZ") || strings.HasPrefix(text, "//VVM") {
		is_likely := ""
		if likely_vvmd {
			is_likely = " (likely)"
		}
		log.Printf("🛑 Ignoring %s system VVM message at %s. Leaving it for vvmd.", is_likely, smsPath)
		return
	}

	// Log
	log.Printf("✉️ SMS @ DBus Path: %s\n", smsPath)
	log.Println(" > Message:", text)
	log.Println(" > From:", number)
	log.Println(" > Received:", received)

	// Save
	res := m.db.Create(&db.Messages{
		Message:  text,
		Number:   number,
		Received: received,
	})
	if res.Error != nil {
		log.Printf("⚠️ Failed to save SMS to database: %v", res.Error)
	} else {
		log.Println("✉️ Saved SMS to database")
	}

	// Clean up
	if err := m.messaging.Delete(sms); err != nil {
		log.Printf("⚠️ Failed to destroy SMS at path %s: %v", smsPath, err)
	} else {
		log.Println("✉️ Destroyed SMS at path:", smsPath)
	}
}

// SyncVoicemails attempts to sync vvmd, with a retry loop to wait for the D-Bus object to appear.
func (m *EG25_G) SyncVoicemails() {
	vvmd_session, err := dbus.SessionBus()
	if err != nil {
		log.Printf("⚠️ Failed to connect to session bus for VVM sync: %v", err)
		return
	}

	log.Println("Modem registered. Waiting for vvmd Mailbox to initialize...")

	// Try up to 5 times, waiting 2 seconds between tries
	for i := range 5 {
		vvmd_obj := vvmd_session.Object("org.kop316.vvm", "/org/kop316/vvm/Mailbox/0")
		vvmd_sync := vvmd_obj.Call("org.kop316.vvm.Mailbox.Sync", 0)

		if vvmd_sync.Err == nil {
			log.Println("📬 vvmd sync triggered successfully.")
			return
		}

		log.Printf("vvmd Mailbox not ready yet, retrying... (%d/5)", i+1)
		time.Sleep(2 * time.Second)
	}
	log.Println("⚠️ Failed to reach vvmd after 5 attempts.")
}

func (m *EG25_G) ListenToVvmdEvents() {
	// vvmd runs on the user/session bus

	// TODO:
	/*
	 * Figure out why this happens:
	 * ⚠️ Failed to connect to Session Bus for vvmd: exec: "dbus-launch": executable file not found in $PATH
	 */
	conn, err := dbus.SessionBus()
	if err != nil {
		log.Printf("⚠️ Failed to connect to Session Bus for vvmd: %v", err)
		return
	}

	// This tells D-Bus to route EVERY signal from vvmd to our application
	conn.AddMatchSignal(dbus.WithMatchSender("org.kop316.vvm"))

	// Create a large-ish buffer just in case vvmd gets chatty
	dbusChan := make(chan *dbus.Signal, 50)
	conn.Signal(dbusChan)

	log.Println("Listening for vvmd (Visual Voicemail) events on Session Bus...")

	for {
		select {
		case <-m.globalCtx.Done():
			return
		case signal := <-dbusChan:
			switch signal.Name {

			// --- MAILBOX MESSAGE DOWNLOADED ---
			case "org.kop316.vvm.Service.MessageAdded":
				if len(signal.Body) > 0 {
					// The signal body contains the D-Bus path to the new message object
					if msgPath, ok := signal.Body[0].(dbus.ObjectPath); ok {
						log.Printf("📥 [VVM] New Voicemail Downloaded! Path: %s", msgPath)
						m.ProcessNewVoicemail(conn, msgPath)
					} else {
						log.Printf("⚠️ Unexpected signal body format: %v", signal.Body)
					}
				}

			// --- MAILBOX PROPERTIES CHANGED (Unread Count) ---
			case "org.freedesktop.DBus.Properties.PropertiesChanged":
				if len(signal.Body) >= 2 {
					interfaceName, _ := signal.Body[0].(string)
					changedProps, ok := signal.Body[1].(map[string]dbus.Variant)

					if ok && interfaceName == "org.kop316.vvm.Mailbox" {
						if unreadVar, exists := changedProps["UnreadCount"]; exists {
							if count, isUint := unreadVar.Value().(uint32); isUint {
								log.Printf("🔔 [VVM] Unread Voicemails updated: %d", count)
								m.unreadVoicemails = int(count)
							}
						}
					}
				}

			case "org.kop316.vvm.ModemManager.ProvisionStatusChanged":
				log.Printf("🔔 [VVM] Provisioning Status Changed: %v", signal.Body)

			// --- OPTIONAL: LOG EVERYTHING ELSE FOR DEBUGGING ---
			default:
				log.Printf("🔍 [VVM-DEBUG] Unhandled Signal: %s on %s", signal.Name, signal.Path)
			}
		}
	}
}

func (m *EG25_G) ProcessNewVoicemail(conn *dbus.Conn, msgPath dbus.ObjectPath) {
	msgObj := conn.Object("org.kop316.vvm", msgPath)

	// Fetch all properties for this message in a single D-Bus call
	var props map[string]dbus.Variant
	err := msgObj.Call("org.freedesktop.DBus.Properties.GetAll", 0, "org.kop316.vvm.Message").Store(&props)
	if err != nil {
		log.Printf("⚠️ [VVM] Failed to fetch properties for %s: %v", msgPath, err)
		return
	}

	// Safely extract the values with fallbacks
	var sender, audioFile string
	var duration int

	if val, ok := props["Sender"]; ok {
		sender, _ = val.Value().(string)
	}

	if val, ok := props["AudioFile"]; ok {
		audioFile, _ = val.Value().(string)
	}

	if val, ok := props["Duration"]; ok {
		// Duration types can vary by architecture/daemon, safely cast it
		switch v := val.Value().(type) {
		case int32:
			duration = int(v)
		case uint32:
			duration = int(v)
		case int:
			duration = v
		}
	}

	log.Printf("   -> Caller: %s", sender)
	log.Printf("   -> Length: %d seconds", duration)
	log.Printf("   -> Audio File: %s", audioFile)

	// Optional: Fetch the Date it was left
	if val, ok := props["Date"]; ok {
		if dateStr, isStr := val.Value().(string); isStr {
			log.Printf("   -> Date: %s", dateStr)
		}
	}
}

func decodeVoicemail(amrFilePath string) (wavFilePath string, err error) {
	wavFilePath = amrFilePath + ".wav"

	// Convert AMR to 8kHz, 16-bit Mono WAV using standard ffmpeg
	cmd := exec.Command("ffmpeg", "-y", "-i", amrFilePath, "-ar", "8000", "-ac", "1", "-c:a", "pcm_s16le", wavFilePath)
	if err := cmd.Run(); err != nil {
		return "", err
	}

	return wavFilePath, nil
}

func (m *EG25_G) SaveCallStateEvent(call modemmanager.Call, callPath dbus.ObjectPath) {
	log.Printf("📞 Call @ DBus Path: %s\n", callPath)
	state, _ := call.GetState()
	number, _ := call.GetNumber()
	reason, _ := call.GetStateReason()
	timestamp := time.Now()
	log.Println("State:", state)
	log.Println("Number:", number)
	log.Println("Reason:", reason)

	res := m.db.Create(&db.CallStateEvents{
		Number:    number,
		Reason:    reason,
		Timestamp: timestamp,
	})

	if res.Error != nil {
		log.Printf("⚠️ Failed to save call state event to database: %v", res.Error)
	} else {
		log.Println("📞 Saved call state event to database")
	}
}

// SetFlightMode sets the flight mode state of the modem.
// If the modem is not available, this function will do nothing.
// If there is an error while setting the flight mode, this function will log a fatal error and exit.
// If the modem is being put out of flight mode, this function will also enable the modem.
func (m *EG25_G) SetFlightMode(state bool) {
	if !m.ok {
		return
	}
	var err error
	if state {
		err = m.Modem.SetPowerState(modemmanager.MmModemPowerStateLow)
	} else {
		err = m.Modem.SetPowerState(modemmanager.MmModemPowerStateOn)
	}
	if err != nil {
		log.Printf("⚠️ Failed to set flight mode to %v: %v", state, err)
	}

	if !state {
		if err := m.Modem.Enable(); err != nil {
			log.Printf("⚠️ Failed to enable modem out of flight mode: %v", err)
		}
	}
}

// CheckSignalStrength gets the initial signal strength of the modem and sets the SignalStrength field.
// If the modem is not available, this function will do nothing.
// If there is an error while getting the signal quality, this function will log a fatal error and exit.
// The signal strength is calculated as quality/25 and capped at 4 (100%).
// If the signal quality is greater than 0 but the calculated signal strength is 0, this function will set the signal strength to 1 (25%) to always show some signal strength when connected.
func (m *EG25_G) CheckSignalStrength() {
	if !m.ok {
		return
	}
	quality, _, err := m.Modem.GetSignalQuality()
	if err != nil {
		log.Printf("⚠️ Failed to get signal quality: %v", err)
	} else {
		log.Printf("Initial signal strength: %d%%\n", quality)
		m.signalStrength = min(int(quality/25), 4)
		if quality > 0 && m.signalStrength == 0 {
			// Always show some signal strength when connected
			m.signalStrength = 1
		}
	}
}

// CheckFlightMode checks the current power state of the modem and sets the FlightMode field accordingly.
// If the modem is not available, this function will do nothing.
// If there is an error while getting the power state, this function will log a fatal error and exit.
// The power state is used to determine if the modem is in flight mode (true) or not (false).
func (m *EG25_G) CheckFlightMode() {
	if !m.ok {
		return
	}
	state, err := m.Modem.GetPowerState()
	if err != nil {
		log.Printf("⚠️ Failed to get modem power state: %v", err)
	} else {
		switch state {
		case modemmanager.MmModemPowerStateOn:
			m.flightMode = false
		case modemmanager.MmModemPowerStateOff, modemmanager.MmModemPowerStateLow, modemmanager.MmModemPowerStateUnknown:
			m.flightMode = true
		}
		log.Println("Flight Mode: ", m.flightMode)
	}
}

// Get3GPP gets the 3GPP interface of the modem and sets the Modem3gpp field.
// If the modem is not available, this function will do nothing.
// If there is an error while getting the 3GPP interface, this function will log a fatal error and exit.
// The 3GPP interface is used to get the carrier name.
func (m *EG25_G) Get3GPP() {
	if !m.ok {
		return
	}
	var err error
	m.modem3gpp, err = m.Modem.Get3gpp()
	if err != nil {
		log.Printf("⚠️ Failed to get 3GPP interface: %v", err)
		m.modem3gpp_ok = false
	} else {
		m.modem3gpp_ok = true
	}
}

// GetCarrierAndRoaming gets the carrier name and the roaming status of the modem.
// If the modem is not available, this function will do nothing.
// If there is an error while getting the carrier name or roaming status, this function will log a fatal error and exit.
// The carrier name is stored in the Carrier field and the roaming status is stored in the Registered, Roaming, and SOS fields.
// The Registered field is set to true if the modem is registered, the Roaming field is set to true if the modem is currently roaming, and the SOS field is set to true if the modem is in an SOS state.
func (m *EG25_G) GetCarrierAndRoaming() {
	if !m.ok {
		return
	}

	if !m.modem3gpp_ok {
		return
	}

	// Get the carrier name
	var err error
	m.carrier, err = m.modem3gpp.GetOperatorName()
	if err != nil {
		log.Printf("⚠️ Failed to get operator name: %v", err)
	} else {
		log.Println("Carrier: ", m.carrier)
	}

	// Check if we are currently roaming
	regState, err := m.modem3gpp.GetRegistrationState()
	if err != nil {
		log.Printf("⚠️ Failed to get registration state: %v", err)
	} else {

		switch regState {
		case modemmanager.MmModem3gppRegistrationStateHome:
			m.registered = true
			m.roaming = false
			m.sos = false
		case modemmanager.MmModem3gppRegistrationStateRoaming, modemmanager.MmModem3gppRegistrationStateRoamingSmsOnly, modemmanager.MmModem3gppRegistrationStateRoamingCsfbNotPreferred:
			m.registered = true
			m.roaming = true
			m.sos = false
		case modemmanager.MmModem3gppRegistrationStateEmergencyOnly:
			m.registered = false
			m.roaming = false
			m.sos = true
		case modemmanager.MmModem3gppRegistrationStateDenied, modemmanager.MmModem3gppRegistrationStateUnknown, modemmanager.MmModem3gppRegistrationStateIdle, modemmanager.MmModem3gppRegistrationStateSearching:
			m.registered = false
			m.roaming = false
			m.sos = false
			if m.carrier == "" {
				m.carrier = "No service"
			}
			if regState == modemmanager.MmModem3gppRegistrationStateSearching {
				m.carrier = "Searching"
			}

		default:
			m.roaming = false
		}
		log.Println("Registration state:", regState)
		log.Println("Registered: ", m.registered)
		log.Println("SOS: ", m.sos)
		log.Println("Roaming: ", m.roaming)
	}
}

func (m *EG25_G) CheckAudioState() {
	audioNeeded := false
	for _, call := range m.GetCalls() {
		if call.State == modemmanager.MmCallStateDialing || call.State == modemmanager.MmCallStateRingingOut || call.State == modemmanager.MmCallStateActive || call.State == modemmanager.MmCallStateHeld || call.State == modemmanager.MmCallStateWaiting {
			audioNeeded = true
			break
		}
	}

	if audioNeeded {
		log.Println("🔊 Audio needed")
		m.enableModemAudioRouting()
		m.startAudioBridge(m.globalCtx)
	} else {
		log.Println("🔇 Audio not needed")
		m.stopAudioBridge()
	}
}

// ListenToModemEvents taps into the D-Bus system bus and routes signals to our Go channels.
func (m *EG25_G) ListenToModemEvents() {
	// Connect directly to the System Bus (where ModemManager lives)
	conn, err := dbus.SystemBus()
	if err != nil {
		log.Fatalf("Failed to connect to system bus: %v", err)
	}

	// Tell D-Bus we want to hear about signals from our specific modem
	modemPath := m.Modem.GetObjectPath()

	// Add match rules for the interfaces we care about
	conn.AddMatchSignal(dbus.WithMatchObjectPath(modemPath))

	// Also match signals from Call objects (which have different paths)
	conn.AddMatchSignal(dbus.WithMatchInterface("org.freedesktop.ModemManager1.Call"))

	// Create a channel to receive the raw D-Bus signals
	dbusChan := make(chan *dbus.Signal, 100)
	conn.Signal(dbusChan)

	log.Println("Listening for ModemManager events...")

	for {
		select {
		case <-m.globalCtx.Done():
			return
		case signal := <-dbusChan:
			// Route the signal based on its D-Bus Member name
			switch signal.Name {

			// --- INCOMING CALLS ---
			case "org.freedesktop.ModemManager1.Modem.Voice.CallAdded":
				if len(signal.Body) > 0 {
					if callPath, ok := signal.Body[0].(dbus.ObjectPath); ok {
						m.incoming_call_events <- callPath
					}
				}

			// --- TERMINATED CALLS ---
			case "org.freedesktop.ModemManager1.Modem.Voice.CallDeleted":
				if len(signal.Body) > 0 {
					if callPath, ok := signal.Body[0].(dbus.ObjectPath); ok {
						m.call_deleted_events <- callPath
					}
				}

			// --- CALL STATE CHANGES ---
			case "org.freedesktop.ModemManager1.Call.StateChanged":
				if len(signal.Body) >= 3 {
					if oldState, ok1 := signal.Body[0].(int32); ok1 {
						if newState, ok2 := signal.Body[1].(int32); ok2 {
							if reason, ok3 := signal.Body[2].(uint32); ok3 {
								m.call_state_events <- callStateChange{path: signal.Path, oldState: modemmanager.MMCallState(oldState), newState: modemmanager.MMCallState(newState), reason: modemmanager.MMCallStateReason(reason)}
							}
						}
					}
				}

			// --- INCOMING SMS ---
			case "org.freedesktop.ModemManager1.Modem.Messaging.Added":
				// The Messaging.Added signal returns (ObjectPath, bool)
				// The boolean indicates if it was received (true) or drafted/sent (false)
				if len(signal.Body) == 2 {
					smsPath, okPath := signal.Body[0].(dbus.ObjectPath)
					isReceived, okRecv := signal.Body[1].(bool)
					if okPath && okRecv && isReceived {
						m.incoming_sms_events <- smsPath
					}
				}

			// --- MODEM STATE CHANGES ---
			case "org.freedesktop.ModemManager1.Modem.StateChanged":
				// StateChanged returns (oldState int32, newState int32, reason uint32)
				if len(signal.Body) >= 2 {
					if newState, ok := signal.Body[1].(int32); ok {
						m.modem_state_events <- modemmanager.MMModemState(newState)
					}
				}

			// --- MODEM POWER STATE CHANGES ---
			case "org.freedesktop.ModemManager1.Modem.PowerStateChanged":
				// PowerStateChanged returns (oldState int32, newState int32)
				if len(signal.Body) >= 2 {
					if newState, ok := signal.Body[1].(int32); ok {
						m.modem_power_state_events <- modemmanager.MMModemPowerState(newState)
					}
				}

			// --- MODEM FAIL EVENTS ---
			case "org.freedesktop.ModemManager1.Modem.Fail":
				// Fail returns (reason uint32)
				if len(signal.Body) >= 1 {
					if reason, ok := signal.Body[0].(uint32); ok {
						m.modem_failure_events <- modemmanager.MMModemStateFailedReason(reason)
					}
				}

			// --- PROPERTIES CHANGED (Signal Strength, Network, etc.) ---
			case "org.freedesktop.DBus.Properties.PropertiesChanged":
				// We only care about the Modem interface properties for signal strength
				if len(signal.Body) >= 2 {
					interfaceName, _ := signal.Body[0].(string)
					changedProps, _ := signal.Body[1].(map[string]dbus.Variant)

					if interfaceName == "org.freedesktop.ModemManager1.Modem" {
						if sigQ, exists := changedProps["SignalQuality"]; exists {
							// SignalQuality is a struct of (uint32, bool) -> (quality percentage, is_recent)
							if val, ok := sigQ.Value().([]any); ok && len(val) > 0 {
								if quality, ok := val[0].(uint32); ok {
									m.signal_strength_events <- quality
								}
							}
						}
					}
				}
			}
		}
	}
}

// ProcessEvents is a separate consumer that reacts to the channel broadcasts.
func (m *EG25_G) ProcessEvents() {
	for {
		select {
		case <-m.globalCtx.Done():
			return
		case callPath := <-m.incoming_call_events:
			log.Printf("📞 [EVENT] Call Incoming Received! DBus Path: %s\n", callPath)
			call, _ := modemmanager.NewCall(callPath)
			m.SaveCallStateEvent(call, callPath)

			// Sync with the modem to create/update the call session
			m.SyncCalls()

			// After sync, find the call and check if it should be announced
			if session, ok := m.calls[callPath]; ok {
				if !session.Announced && session.State == modemmanager.MmCallStateRingingIn {
					switch session.Reason {
					case modemmanager.MmCallStateReasonIncomingNew, modemmanager.MmCallStateReasonTransferred:
						session.Announced = true
						go func() {
							m.ringingChan <- session
						}()
					}
				}
			}

		case callPath := <-m.call_deleted_events:
			log.Printf("📞 [EVENT] Call Deleted (Hung up)! DBus Path: %s\n", callPath)

			// Sync with modem state to ensure map is accurate
			m.SyncCalls()

			go m.CheckAudioState()

		case event := <-m.call_state_events:
			log.Printf("📞 [EVENT] Call State Changed! Path: %s, State: %d -> %d, Reason: %d\n", event.path, event.oldState, event.newState, event.reason)
			call, _ := modemmanager.NewCall(event.path)
			m.SaveCallStateEvent(call, event.path)
			if session := m.calls[event.path]; session != nil {
				session.State = event.newState

				switch event.newState {
				case modemmanager.MmCallStateActive:
					session.StartTime = time.Now()
					m.current_call = session
				case modemmanager.MmCallStateTerminated:
					select {
					case session.Ended <- true:
					default:
					}
					delete(m.calls, event.path)
					if m.current_call == session {
						m.current_call = nil
						for _, c := range m.calls {
							m.current_call = c
							break
						}
					}
				}
			}
			go m.CheckAudioState()

		case smsPath := <-m.incoming_sms_events:
			log.Printf("✉️ [EVENT] New SMS Received! DBus Path: %s\n", smsPath)
			// Instantiate the SMS to read it:
			sms, _ := modemmanager.NewSms(smsPath)
			m.SaveSMS(sms, smsPath)

		case state := <-m.modem_power_state_events:
			log.Printf("🔋 [EVENT] Modem Power State Changed to: %d\n", state)
			switch state {
			case modemmanager.MmModemPowerStateOn:
				log.Println("   -> Modem is on!")
				m.flightMode = false
			case modemmanager.MmModemPowerStateOff, modemmanager.MmModemPowerStateLow:
				log.Println("   -> Modem is off/in low power mode!")
				m.flightMode = true
			case modemmanager.MmModemPowerStateUnknown:
				log.Println("   -> Modem power state is unknown!")
				m.flightMode = true
			}

		case state := <-m.modem_failure_events:
			log.Printf("❌ [EVENT] Modem Failed! Reason: %d\n", state)
			switch state {
			case modemmanager.MmModemStateFailedReasonUnknown:
				log.Println("   -> Modem failed reason is unknown!")
			case modemmanager.MmModemStateFailedReasonSimError:
				log.Println("   -> SIM card error!")
			case modemmanager.MmModemStateFailedReasonSimMissing:
				log.Println("   -> SIM card missing!")
			}

		case state := <-m.modem_state_events:
			log.Printf("🔄 [EVENT] Modem State Changed to: %d\n", state)

			do_chores := func() {
				m.CheckFlightMode()
				m.GetCarrierAndRoaming()
			}

			switch state {
			case modemmanager.MmModemStateInitializing:
				log.Println("   -> Modem is initializing!")
			case modemmanager.MmModemStateConnecting:
				log.Println("   -> Modem is connecting...")
			case modemmanager.MmModemStateDisconnecting:
				log.Println("   -> Modem is disconnecting...")
			case modemmanager.MmModemStateConnected:
				log.Println("   -> Modem is connected!")
			case modemmanager.MmModemStateRegistered:
				log.Println("   -> Modem is registered to the network!")
				go m.SyncVoicemails()
			case modemmanager.MmModemStateEnabled:
				log.Println("   -> Modem is enabled!")
			case modemmanager.MmModemStateSearching:
				log.Println("   -> Modem is searching for a network...")
			case modemmanager.MmModemStateLocked:
				log.Println("   -> Modem is locked!")
			case modemmanager.MmModemStateDisabled:
				log.Println("   -> Modem is disabled! (powered down?)")
			}
			go do_chores()

		case quality := <-m.signal_strength_events:
			log.Printf("📶 [EVENT] Signal Strength updated: %d%%\n", quality)

			// Scale the signal strength between 0-4
			m.signalStrength = min(int(quality/25), 4)
			if quality > 0 && m.signalStrength == 0 {
				// Always show some signal strength when connected
				m.signalStrength = 1
			}
		}
	}
}

// PlaceCall initiates a new outgoing voice call.
// Returns the Call object so you can track its state or path, and an error if it fails.
func (m *EG25_G) PlaceCall(number string) (*modem.Call, error) {
	if !m.ok {
		return nil, fmt.Errorf("modem is not ready")
	}

	voice, err := m.Modem.GetVoice()
	if err != nil {
		return nil, fmt.Errorf("failed to get voice interface: %v", err)
	}

	log.Printf("📞 Placing call to %s...", number)

	// Create the call
	call, err := voice.CreateCall(number)
	if err != nil {
		return nil, fmt.Errorf("failed to create call: %v", err)
	}

	dbuspath := call.GetObjectPath()

	output := &modem.Call{
		ID:        uuid.New().String(),
		DBusPath:  dbuspath,
		Call:      call,
		State:     modemmanager.MmCallStateDialing,
		Number:    number,
		Mute:      false,
		Volume:    1,
		Ended:     make(chan bool, 1),
		Announced: false,
	}

	m.calls[output.DBusPath] = output
	m.current_call = output

	// Dial the number
	if err := call.Start(); err != nil {
		delete(m.calls, output.DBusPath)
		if m.current_call == output {
			m.current_call = nil
			for _, c := range m.calls {
				m.current_call = c
				break
			}
		}
		return nil, fmt.Errorf("failed to start call: %v", err)
	}

	log.Printf("📞 Call started successfully! DBus Path: %s", dbuspath)
	return output, nil
}

// AnswerCall accepts an incoming ringing call given its DBus path.
func (m *EG25_G) AnswerCall(call *modem.Call) error {
	if !m.ok {
		return fmt.Errorf("modem is not ready")
	}

	log.Printf("📞 Answering call at %s...", call)

	if err := call.Call.Accept(); err != nil {
		return fmt.Errorf("failed to accept call: %v", err)
	}

	log.Println("📞 Call accepted.")
	return nil
}

func (m *EG25_G) GetCalls() []*modem.Call {
	calls := make([]*modem.Call, 0, len(m.calls))
	for _, call := range m.calls {
		calls = append(calls, call)
	}
	return calls
}

// HangupCall terminates an active or ringing call given its DBus path.
func (m *EG25_G) HangupCall(call *modem.Call) error {
	if !m.ok {
		return fmt.Errorf("modem is not ready")
	}

	log.Printf("📞 Hanging up call %s...", call)

	if err := call.Call.Hangup(); err != nil {
		return fmt.Errorf("failed to hang up call: %v", err)
	}

	log.Println("📞 Call hung up.")
	return nil
}

// HangupAll is a convenience wrapper to terminate all ongoing, ringing, or held calls at once.
func (m *EG25_G) HangupAll() error {
	if !m.ok {
		return fmt.Errorf("modem is not ready")
	}

	voice, err := m.Modem.GetVoice()
	if err != nil {
		return fmt.Errorf("failed to get voice interface: %v", err)
	}

	log.Println("📞 Hanging up and clearing all calls...")

	if err := voice.HangupAll(); err != nil {
		return fmt.Errorf("failed to clear calls: %v", err)
	}
	return nil
}

// SendDTMF sends dual-tone multi-frequency (DTMF) tones to the active call.
// This is used for navigating phone menus, or triggering carrier hold/transfer features.
// Tones can be numbers 0-9, *, #, or A-D.
func (m *EG25_G) SendDTMF(call *modem.Call, tones string) error {
	if !m.ok {
		return fmt.Errorf("modem is not ready")
	}

	log.Printf("📞 Sending DTMF tones '%s' to call %s...", tones, call)

	if err := call.Call.SendDtmf(tones); err != nil {
		return fmt.Errorf("failed to send DTMF tones: %v", err)
	}

	log.Println("📞 DTMF sent.")
	return nil
}

func (m *EG25_G) HoldCall(call *modem.Call) error {
	if !m.ok {
		return fmt.Errorf("modem is not ready")
	}

	log.Printf("📞 Holding call %s (AT+CHLD=2)...", call)
	if _, err := m.Modem.Command("AT+CHLD=2", 10); err != nil {
		log.Printf("⚠️ Failed to hold call: %v", err)
		return err
	}

	return nil
}

// SyncCalls synchronizes the local call list with the modem's actual state.
func (m *EG25_G) SyncCalls() {
	voice, err := m.Modem.GetVoice()
	if err != nil {
		log.Printf("⚠️ Failed to get voice interface for sync: %v", err)
		return
	}

	calls, err := voice.ListCalls()
	if err != nil {
		log.Printf("⚠️ Failed to list calls for sync: %v", err)
		return
	}

	// Track matched local calls to identify stale ones later
	matchedCalls := make(map[*modem.Call]bool)

	for _, mmCall := range calls {
		path := mmCall.GetObjectPath()
		number, _ := mmCall.GetNumber()
		state, _ := mmCall.GetState()

		// 1. Try to match by DBus Path (Primary Key)
		if session, exists := m.calls[path]; exists {
			session.State = state
			session.Call = mmCall
			matchedCalls[session] = true
			continue
		}

		// 2. Try to match by Phone Number (Stable Key)
		var foundSession *modem.Call
		var oldPath dbus.ObjectPath

		for p, session := range m.calls {
			if session.Number == number && !matchedCalls[session] {
				foundSession = session
				oldPath = p
				break
			}
		}

		if foundSession != nil {
			log.Printf("🧹 Sync: Re-associating call %s (old path %s -> new path %s)", number, oldPath, path)
			delete(m.calls, oldPath)
			foundSession.DBusPath = path
			foundSession.Call = mmCall
			foundSession.State = state
			m.calls[path] = foundSession
			matchedCalls[foundSession] = true
		} else {
			// 3. New Call
			if state == modemmanager.MmCallStateTerminated {
				continue
			}

			log.Printf("🧹 Sync: Found untracked call %s (%s)", path, number)
			reason, _ := mmCall.GetStateReason()
			newCall := &modem.Call{
				ID:        uuid.New().String(),
				DBusPath:  path,
				Call:      mmCall,
				State:     state,
				Number:    number,
				Reason:    reason,
				Mute:      false,
				Volume:    1,
				Ended:     make(chan bool, 1),
				Announced: false,
			}
			m.calls[path] = newCall
			matchedCalls[newCall] = true
		}
	}

	// 4. Prune stale calls
	for path, session := range m.calls {
		if !matchedCalls[session] {
			log.Printf("🧹 Sync: Pruning stale call %s (%s)", path, session.Number)
			session.State = modemmanager.MmCallStateTerminated
			select {
			case session.Ended <- true:
			default:
			}
			delete(m.calls, path)
		}
	}

	// 5. Validate current_call
	if m.current_call != nil {
		if session, exists := m.calls[m.current_call.DBusPath]; !exists || session != m.current_call {
			m.current_call = nil
		}
	}

	if m.current_call == nil {
		// Prefer active calls
		for _, c := range m.calls {
			if c.State == modemmanager.MmCallStateActive {
				m.current_call = c
				break
			}
		}
		// Fallback to any call
		if m.current_call == nil {
			for _, c := range m.calls {
				m.current_call = c
				break
			}
		}
	}

	go m.CheckAudioState()
}

func (m *EG25_G) UnholdCall(call *modem.Call) error {
	if !m.ok {
		return fmt.Errorf("modem is not ready")
	}

	log.Printf("📞 Unholding call %s (AT+CHLD=2)...", call)
	if _, err := m.Modem.Command("AT+CHLD=2", 10); err != nil {
		log.Printf("⚠️ Failed to unhold call: %v", err)
		return err
	}

	call.State = modemmanager.MmCallStateActive
	return nil
}

// FindAlsaCard searches /proc/asound/cards for the EG25-G or Quectel entry.
func (m *EG25_G) FindAlsaCard() string {
	file, err := os.Open("/proc/asound/cards")
	if err != nil {
		return "hw:0,0" // Fallback
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "EG25G") || strings.Contains(line, "Quectel") {
			// Lines look like: " 3 [EG25G          ]: USB-Audio - EG25-G"
			fields := strings.Fields(line)
			if len(fields) > 0 {
				log.Printf("🔊 Found Modem Audio at ALSA card %s", fields[0])
				return fmt.Sprintf("plughw:%s,0", fields[0])
			}
		}
	}
	return "plughw:0,0"
}

// enableModemAudioRouting applies the volatile AT command to route audio to USB.
func (m *EG25_G) enableModemAudioRouting() {
	if !m.ok {
		return
	}
	m.audio_lock.Lock()
	defer m.audio_lock.Unlock()
	if m.audio_enabled {
		log.Println("🔊 Volatile routing already enabled")
		return
	}
	log.Println("🔊 Applying volatile routing: AT+QPCMV=1,2")
	_, err := m.Modem.Command("AT+QPCMV=1,2", 5)
	if err != nil {
		log.Printf("⚠️ Audio routing command failed: %v", err)
		return
	}
	m.audio_enabled = true
}

func (m *EG25_G) getAudioSampleRate() string {
	if !m.ok {
		return "8000"
	}

	resp, err := m.Modem.Command("AT+QDAI?", 2)
	if err != nil {
		log.Printf("⚠️ Failed to query audio sample rate: %v", err)
		return "8000"
	}

	log.Println("🎤 Got audio sample rate:", resp)

	// Response format: +QDAI: <io>,<mode>,<fs>,...
	// <fs>: 0=8000, 1=16000
	if after, ok := strings.CutPrefix(resp, "+QDAI: "); ok {
		parts := strings.Split(after, ",")
		if len(parts) >= 3 {
			if strings.TrimSpace(parts[2]) == "1" {
				return "16000"
			}
		}
	}
	return "8000"
}

func (m *EG25_G) startAudioBridge(ctx context.Context) {
	m.audio_lock.Lock()
	defer m.audio_lock.Unlock()

	if m.audioBridgeCancel != nil {
		return
	}

	rate := m.getAudioSampleRate()
	log.Printf("🎤 Starting Audio Bridge (alsaloop) at %sHz...", rate)
	bridgeCtx, cancel := context.WithCancel(ctx)
	m.audioBridgeCancel = cancel

	// Modem -> Speaker
	go func() {
		cmd := exec.CommandContext(bridgeCtx, "alsaloop", "-C", m.audio_hwid, "-P", "default", "-t", "100000", "-A", "2", "-b", "-S", "0", "-f", "S16_LE", "-c", "1", "-r", rate)
		if err := cmd.Run(); err != nil {
			log.Printf("⚠️ alsaloop (Modem->Speaker) exited: %v", err)
		}
	}()

	// Mic -> Modem
	go func() {
		cmd := exec.CommandContext(bridgeCtx, "alsaloop", "-C", "default", "-P", m.audio_hwid, "-t", "100000", "-A", "2", "-b", "-S", "0", "-f", "S16_LE", "-c", "1", "-r", rate)
		if err := cmd.Run(); err != nil {
			log.Printf("⚠️ alsaloop (Mic->Modem) exited: %v", err)
		}
	}()
}

func (m *EG25_G) stopAudioBridge() {
	m.audio_lock.Lock()
	defer m.audio_lock.Unlock()

	if m.audioBridgeCancel == nil {
		return
	}

	log.Println("🔇 Stopping Audio Bridge...")
	m.audioBridgeCancel()
	m.audioBridgeCancel = nil
	m.audio_enabled = false
}
