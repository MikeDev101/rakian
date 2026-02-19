package phone

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tarm/serial"
	"github.com/warthog618/sms/encoding/ucs2"
)

type CallState struct {
	Index            int
	ConnectionType   string
	Status           string
	IsConferenceCall bool
	IsCallInbound    bool
	PhoneNumber      string
	StartTime        time.Time
}

type Modem struct {
	CallState         *CallState
	Port              *serial.Port
	AudioPort         *serial.Port
	audioCmd          *exec.Cmd
	DebugMode         bool
	mu                sync.Mutex
	cmdMutex          sync.Mutex
	lastResponse      strings.Builder
	cmdWait           sync.WaitGroup
	RingingChan       chan bool
	MissedCallChan    chan bool
	CallStartChan     chan bool
	CallEndChan       chan bool
	CallErrorChan     chan bool
	CallHandledChan   chan bool
	SimCardInserted   bool
	NowRinging        bool
	urcChan           chan string
	inCommand         bool
	handlers          map[string]func(string)
	SignalStrength    int
	Carrier           string // i.e. T-Mobile, Verizon, AT&T, Fi
	DataEnabled       bool
	DataConnected     bool
	NetworkGeneration string // 2g/3g/4g/negotiating
	Connected         bool
	gatheredRingData  bool
	batteryWindow     []int
	FlightMode        bool
	SimulationMode    bool
}

func NewModem(port string, baud int, debug bool) (*Modem, error) {
	cfg := &serial.Config{Name: port, Baud: baud, ReadTimeout: time.Second}
	p, err := serial.OpenPort(cfg)
	if err != nil {
		return nil, err
	}

	m := &Modem{
		CallState:       &CallState{},
		Carrier:         "Searching...",
		Port:            p,
		DataEnabled:     false,
		DataConnected:   false, // stub - TODO: check if a wwan0 connection is alive
		RingingChan:     make(chan bool, 1),
		CallStartChan:   make(chan bool, 1),
		CallEndChan:     make(chan bool, 1),
		CallErrorChan:   make(chan bool, 1),
		CallHandledChan: make(chan bool, 1),
		urcChan:         make(chan string, 20),
		DebugMode:       debug,
	}

	m.handlers = map[string]func(string){
		"RING":  m.handleCall,
		"+CMT:": m.handleSMSDirectly,
		// "+CMTI:":       m.handleSMS,
		"+CSQ:":        m.handleSignalStrength,
		"+CNSMOD:":     m.handleConnectionType,
		"+SIMCARD:":    m.handleSIMCard,
		"+CPIN:":       m.handleCPIN,
		"+CCLK:":       m.handleClock,
		"+CME ERROR:":  m.handleCMEError,
		"+CMEE":        m.handleCMEE,
		"+CLCC:":       m.handleCallStatus,
		"MISSED_CALL:": m.handleMissedCall,
		"+COPS:":       m.handleCarrierInfo,
		"NO CARRIER":   m.handleNoCarrier,
		"+CREG:":       m.handleRegistrationUpdate,
		"+CGREG:":      m.handleRegistrationUpdate,
		"+CEREG:":      m.handleRegistrationUpdate,
	}

	go m.listenLoop()

	// Initial setup sequence
	initCmds := []string{
		"AT+CFUN=1",                    // Enable
		"ATE0",                         // Disable echo early to prevent polluted buffers
		"AT+COPS?",                     // Check network status
		"AT+CSQ",                       // Check signal strength
		"AT+CSCLK=0",                   // Disable sleep mode during initialization
		"AT+CATR=0",                    // Ensure URCs only go to the active port
		"AT+AUTOCSQ=0,0",               // Disable early signal reports until ready
		"AT+CLCC=1",                    // Call reporting
		"AT+COUTGAIN=8",                // Set speaker gain
		"AT+CMICGAIN=8",                // Set mic gain
		"AT+CSMS=1",                    // Enable SMS (GSM Phase 2+)
		"AT+CSCA=\"+19037029920\"",     // Set short code address for Verizon SMS
		"AT+CMGF=1",                    // Set SMS text mode
		"AT+CPMS=\"ME\",\"ME\",\"ME\"", // Set SMS storage to RAM
		"AT+CNMI=2,2,0,0,0",            // Configure notifications
		"AT+CNMP=2",                    // Automatic network mode
		"AT+CNSMOD=1",                  // Network mode updates
		"AT+CPCMFRM=1",                 // Configure 16 KHz audio mode
		"AT+CREG=2",                    // Configure network registration
		"AT+CEREG=2",                   // Configure network registration
		"AT+CPIN?",                     // Check SIM card status
		"AT+AUTOCSQ=1,1",               // Enable signal reports since we're ready
	}
	for _, cmd := range initCmds {
		resp, _ := m.send(cmd)
		m.HandleEvent(resp)
	}

	return m, nil
}

// async reader splits command results from events
func (m *Modem) listenLoop() {
	reader := bufio.NewReader(m.Port)

	for {
		line, err := reader.ReadString('\r')
		if err != nil && err != io.EOF {
			log.Println("🔌 Read error:", err)
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		m.mu.Lock()
		if m.inCommand {
			log.Println(line)
			m.lastResponse.WriteString(line + "\n")
			if line == "OK" || strings.Contains(line, "ERROR") || strings.Contains(line, "NO CARRIER") {
				m.inCommand = false
				m.cmdWait.Done()
			}
			m.mu.Unlock()
		} else {
			m.mu.Unlock()
			log.Println(line)
			if m.isUnsolicited(line) {
				if strings.HasPrefix(line, "+CMT:") {
					body, err := reader.ReadString('\r')
					if err == nil {
						cleanBody := strings.TrimSpace(body)
						cleanBody = strings.ReplaceAll(cleanBody, "\n", "\r")
						line = fmt.Sprintf("%s\r%s", line, cleanBody)
					}
				}
				m.urcChan <- line
			}
		}
	}
}

func (m *Modem) handleRegistrationUpdate(line string) {
	var stat int
	var lac, ci, act string
	var prefix string

	if strings.HasPrefix(line, "+CEREG:") {
		prefix = "+CEREG:"
	} else if strings.HasPrefix(line, "+CGREG:") {
		prefix = "+CGREG:"
	} else if strings.HasPrefix(line, "+CREG:") {
		prefix = "+CREG:"
	} else {
		return
	}

	// Parse the line
	payload := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	parts := strings.Split(payload, ",")

	if len(parts) >= 1 {
		if val, err := strconv.Atoi(parts[0]); err == nil {
			stat = val
		}
	}

	if len(parts) >= 3 {
		lac = strings.Trim(parts[1], "\"")
		ci = strings.Trim(parts[2], "\"")
	}

	if len(parts) >= 4 {
		act = strings.Trim(parts[3], "\"")
	}

	statusMap := map[int]string{
		0: "Not registered, not searching",
		1: "Registered, home network",
		2: "Not registered, searching...",
		3: "Registration denied",
		4: "Not registered, unknown",
		5: "Registered, roaming",
		6: "Registered, SMS only (home)",
		7: "Registered, SMS only (roaming)",
		8: "Emergency services only",
	}

	statusStr, ok := statusMap[stat]
	if !ok {
		statusStr = fmt.Sprintf("Unknown status (%d)", stat)
	}

	if m.DebugMode {
		log.Printf("📡 Registration Update (%s): %s", strings.TrimSuffix(prefix, ":"), statusStr)
		if lac != "" && ci != "" {
			log.Printf("   Area Code: %s, Cell ID: %s", lac, ci)
		}
		if act != "" {
			log.Printf("   Access Tech: %s", act)
		}
	}

	// Update connected state based on stat
	switch stat {
	case 1, 5, 6, 7, 8:
		m.Connected = true
		actInt, _ := strconv.Atoi(act)
		m.NetworkGeneration = mapActToGen(actInt)
	case 0, 4:
		m.Connected = false
		m.SignalStrength = 0
		m.NetworkGeneration = ""
		m.Carrier = "No Service"
	case 2:
		m.Connected = false
		m.NetworkGeneration = ""
		m.SignalStrength = 0
		m.Carrier = "Searching..."
	}

	// Send AT+COPS? request
	resp, _ := m.send("AT+COPS?")
	m.HandleEvent(resp)
}

func (m *Modem) EnterNumber(key rune) {
	m.send("AT+VTS=\"" + string(key) + "\"")
}

func (m *Modem) isUnsolicited(line string) bool {
	prefixes := []string{
		"RING", "+CMT:", "+CMTI:", "+CSQ:", "+CLCC:", "+CCLK:", "+SIMCARD:",
		"+CPIN", "+CNSMOD:", "+CME ERROR:", "+CMEE", "MISSED_CALL:",
		"NO CARRIER", "+CBC:", "+CREG:", "+CEREG:",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(line, p) {
			return true
		}
	}
	return false
}

func (m *Modem) send(cmd string) (string, error) {
	m.cmdMutex.Lock()
	defer m.cmdMutex.Unlock()

	m.lastResponse.Reset()
	m.inCommand = true
	m.cmdWait = sync.WaitGroup{}
	m.cmdWait.Add(1)

	m.mu.Lock()
	log.Println(cmd)
	_, err := m.Port.Write([]byte(cmd + "\r"))
	m.mu.Unlock()

	if err != nil {
		return "", err
	}

	m.cmdWait.Wait()

	resp := m.lastResponse.String()

	// Trim newlines from response
	resp = strings.TrimSuffix(resp, "\n")
	resp = strings.TrimSuffix(resp, "\r")

	return resp, nil
}

func (m *Modem) send_no_wait(cmd string) error {
	m.cmdMutex.Lock()
	defer m.cmdMutex.Unlock()

	m.mu.Lock()
	m.lastResponse.Reset()
	m.inCommand = true
	log.Println(cmd)
	_, err := m.Port.Write([]byte(cmd + "\r"))
	m.mu.Unlock()

	m.inCommand = false
	return err
}

// convenience methods
func (m *Modem) Dial(number string) error {
	resp, err := m.send("ATD" + number + ";")
	m.HandleEvent(resp)
	if strings.Contains(resp, "ERROR") {

		// Attempt state recovery
		m.CallErrorChan <- true
		<-m.CallHandledChan

		// End the call
		m.CallEndChan <- true
	}

	return err
}
func (m *Modem) Answer() error {
	if !m.SimulationMode {
		resp, err := m.send("ATA")
		m.HandleEvent(resp)
		return err
	}

	// Simulate start call
	go m.handleCallStatus("+CLCC: 1,1,0,0,0,\"TEST NUMBER\",0")

	return nil
}
func (m *Modem) Hangup() error {
	if !m.SimulationMode {
		resp, err := m.send("AT+CHUP")
		m.HandleEvent(resp)
		return err
	}

	// Simulate end call
	go m.handleCallStatus("+CLCC: 1,1,6,0,0,\"TEST NUMBER\",0")

	return nil
}
func (m *Modem) MuteMic(b bool) error     { return m.toggle("AT+CMUT", b) }
func (m *Modem) MuteSpeaker(b bool) error { return m.toggle("AT+VMUTE", b) }
func (m *Modem) toggle(cmd string, b bool) error {
	val := 0
	if b {
		val = 1
	}
	_, err := m.send(fmt.Sprintf("%s=%d", cmd, val))
	return err
}
func (m *Modem) SendSMS(to, message string) error {
	if _, err := m.send("AT+CMGF=1"); err != nil {
		return err
	}
	if _, err := m.send(fmt.Sprintf("AT+CMGS=\"%s\"", to)); err != nil {
		return err
	}
	_, err := m.send(message + string(rune(26))) // Ctrl+Z
	return err
}

func (m *Modem) ToggleFlightMode() error {
	if m.FlightMode {
		resp, err := m.send("AT+CFUN=1")
		m.HandleEvent(resp)
		m.FlightMode = false
		return err
	} else {
		resp, err := m.send("AT+CFUN=0")
		m.HandleEvent(resp)
		m.FlightMode = true
		m.SimCardInserted = false
		return err
	}
}

func (m *Modem) handleCarrierInfo(line string) {
	// 1. Check for the "Searching" or "Deregistered" state first
	if line == "+COPS: 0" || strings.Contains(line, ",,,") {
		m.Carrier = "Searching..."
		m.SignalStrength = 0
		m.Connected = false
		return
	}

	// 2. Flexible Regex: Matches name or numeric ID
	// Supports: +COPS: <mode>[,<format>,<oper>[,<AcT>]]
	re := regexp.MustCompile(`\+COPS:\s*\d+,\d+,"([^"]+)"(?:,(\d+))?`)
	matches := re.FindStringSubmatch(line)

	if len(matches) > 1 {
		m.Carrier = matches[1]

		// Trim suffix spaces
		m.Carrier = strings.TrimSuffix(m.Carrier, " ")

		m.Connected = true

		// Map the Access Technology (matches[2])
		if len(matches) > 2 && matches[2] != "" {
			act, _ := strconv.Atoi(matches[2])
			m.NetworkGeneration = mapActToGen(act)
		}
	}

	if m.DebugMode {
		log.Printf("📶 Carrier Update: %s (%s)", m.Carrier, m.NetworkGeneration)
	}
}

// Helper for the Act (Access Technology) field
func mapActToGen(act int) string {
	switch act {
	case 0, 1:
		return "2G"
	case 2, 4, 5, 6:
		return "3G"
	case 7:
		return "LTE"
	default:
		return ""
	}
}

// smart decoding
func autoDecodeSMS(raw string) string {
	raw = strings.TrimSpace(raw)
	if isLikelyHexUCS2(raw) {
		decoded, err := decodeUCS2(raw)
		if err != nil {
			return fmt.Sprintf("[ucs2 decode error: %v]", err)
		}
		return decoded
	}
	return raw
}
func isLikelyHexUCS2(s string) bool {
	if len(s) < 4 || len(s)%4 != 0 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
func decodeUCS2(hexStr string) (string, error) {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}
	runes, err := ucs2.Decode(bytes)
	if err != nil {
		return "", err
	}
	return string(runes), nil
}

// event handlers
func (m *Modem) handleCall(_ string) {
	if m.DebugMode {
		log.Println("📞 Incoming call...")
	}
}

func (m *Modem) handleMissedCall(line string) {
	re := regexp.MustCompile(`MISSED_CALL:\s*(\d{1,2}:\d{2}[AP]M)\s+([+\d]+)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) > 2 {
		timeStr := matches[1]
		number := matches[2]
		log.Printf("📞 Missed call from %s at %s", number, timeStr)
	} else if m.DebugMode {
		log.Println("📞 Missed call (unknown details)...")
	}

	select {
	case m.MissedCallChan <- true:
	default:
	}
}

func (m *Modem) handleSignalStrength(line string) {
	var rssi, ber int
	if _, err := fmt.Sscanf(line, "+CSQ: %d,%d", &rssi, &ber); err == nil {

		if m.DebugMode {
			log.Printf("📶 Signal strength: RSSI=%d, BER=%d", rssi, ber)
		}

		if rssi >= 0 && rssi <= 31 {
			m.SignalStrength = min(rssi*8/31, 7)
			m.Connected = true
		} else {
			m.SignalStrength = 0
			m.Carrier = "No Service"
			m.Connected = false
		}

		// Get network connection type
		// network_type, _ := m.send("AT+CNSMOD?")
		// go m.handlers["+CNSMOD:"](network_type)

		// Get carrier status
		// carrier_status, _ := m.send("AT+COPS?")
		// go m.handlers["+COPS:"](carrier_status)
	}
}

func (m *Modem) handleCallStatus(line string) {
	re := regexp.MustCompile(`\+CLCC:\s*(\d+),(\d+),(\d+),(\d+),(\d+),"(.*?)",(\d+)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) == 0 {
		return
	}

	call_index_number, _ := strconv.Atoi(matches[1])
	is_call_inbound, _ := strconv.Atoi(matches[2])
	call_status, _ := strconv.Atoi(matches[3])
	call_type, _ := strconv.Atoi(matches[4])
	is_multiparty, _ := strconv.Atoi(matches[5])
	call_number := matches[6]

	call_statuses := map[int]string{
		0: "active",
		1: "held",
		2: "dialing",  // Outbound
		3: "ringing",  // Outbound
		4: "incoming", // Inbound
		5: "waiting",  // Inbound
		6: "disconnected",
	}

	call_types := map[int]string{
		0: "voice",
		1: "data",
		2: "fax",
		9: "unknown",
	}

	if m.DebugMode {
		log.Printf("☎️ Call status %s [Index: %d]", call_number, call_index_number)
		log.Println("   Is this a conference call? ", is_multiparty == 1)
		log.Println("   Was this call inbound? ", is_call_inbound == 1)
		log.Printf("   Status of call: %s", call_statuses[call_status])
		log.Printf("   Type of call: %s", call_types[call_type])
	}

	m.CallState.Index = call_index_number
	m.CallState.IsCallInbound = is_call_inbound == 1
	m.CallState.IsConferenceCall = is_multiparty == 1
	m.CallState.PhoneNumber = call_number
	m.CallState.Status = call_statuses[call_status]
	m.CallState.ConnectionType = call_types[call_type]

	switch call_status {
	case 0:
		if m.CallState.StartTime.IsZero() {
			m.CallState.StartTime = time.Now()
		}
	case 2, 3, 4, 5:
		m.CallState.StartTime = time.Time{}
	}

	if is_call_inbound == 1 { // Handle inbound call state

		switch call_status {
		case 0: // active
			m.CallStartChan <- true
			go m.InitPCMStream()

		case 4: // incoming
			m.RingingChan <- true

		case 6: // disconnected
			m.CallEndChan <- true
			go m.EndPCMStream()
			if m.SimulationMode {
				m.SimulationMode = false
			}
		}

	} else { // Handle outbound call state

		switch call_status {

		case 0: // active

		case 2: // dialing
			m.CallStartChan <- true
			go m.InitPCMStream()

		case 6: // disconnected
			m.CallEndChan <- true
			go m.EndPCMStream()
			if m.SimulationMode {
				m.SimulationMode = false
			}
		}
	}
}

func (m *Modem) InitPCMStream() {
	<-time.After(100 * time.Millisecond)
	resp, err := m.send("AT+CPCMREG=1")
	if err == nil {
		if m.DebugMode {
			log.Printf("🚿 PCM Stream started! (%s)", resp)
		}
	} else {
		log.Printf("🚿 PCM Steam start error: %v", err)
	}
	if m.audioCmd == nil {
		cmd := exec.Command("/usr/bin/serial_audio_16")
		stdout, _ := cmd.StdoutPipe()

		if err := cmd.Start(); err != nil {
			log.Printf("⚠️ Failed to start audio process: %v", err)
		} else {
			m.audioCmd = cmd

			// Print out messages from the audio process
			if stdout != nil {
				go func() {
					scanner := bufio.NewScanner(stdout)
					for scanner.Scan() {
						if m.DebugMode {
							log.Printf("🔊 %s", scanner.Text())
						}
					}
				}()
			}
		}
	}
}

func (m *Modem) EndPCMStream() {
	if m.audioCmd != nil && m.audioCmd.Process != nil {
		if err := m.audioCmd.Process.Signal(os.Interrupt); err != nil {
			log.Printf("⚠️ Failed to interrupt audio process: %v", err)
		}
		if err := m.audioCmd.Wait(); err != nil {
			log.Printf("⚠️ Audio process exited with error: %v", err)
		}
		m.audioCmd = nil
	}
	<-time.After(100 * time.Millisecond)
	resp, err := m.send("AT+CPCMREG=0")
	if err == nil {
		if m.DebugMode {
			log.Printf("🚿 PCM Stream stopped! (%s)", resp)
		}
	} else {
		log.Printf("🚿 PCM Steam stop error: %v", err)
	}
}

func (m *Modem) handleClock(line string) {
	re := regexp.MustCompile(`\+CCLK:\s*"(\d{2})/(\d{2})/(\d{2}),(\d{2}):(\d{2}):(\d{2})\+(\d+)"`)
	if matches := re.FindStringSubmatch(line); len(matches) == 8 {
		if m.DebugMode {
			log.Printf("🕒 Clock: %s/%s/%s %s:%s:%s (TZ +%s)", matches[1], matches[2], matches[3], matches[4], matches[5], matches[6], matches[7])
		}
	}
}

func (m *Modem) handleSMSDirectly(line string) {
	parts := strings.Split(line, "\r")
	if len(parts) < 2 {
		return
	}

	header := parts[0]
	body := strings.Join(parts[1:], "\n")

	re := regexp.MustCompile(`\+CMT:\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(header)
	sender := "Unknown"
	if len(matches) > 1 {
		sender = matches[1]
	}

	log.Printf("📩 New SMS from %s: %s", sender, autoDecodeSMS(body))
}

/* func (m *Modem) handleSMS(line string) {
	if m.DebugMode {
		log.Println("💡 New SMS:", line)
	}
	parts := strings.Split(line, ",")
	if len(parts) != 2 {
		return
	}
	index := strings.TrimSpace(parts[1])

	// Wait a moment for the message to be fully written to memory
	time.Sleep(500 * time.Millisecond)

	msg, _ := m.send("AT+CMGR=" + index)

	lines := strings.SplitSeq(msg, "\n")
	for l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "+CMGR:") {
			continue
		}
		if strings.HasPrefix(l, "OK") {
			continue
		}
		if m.DebugMode {
			log.Println("📩 SMS:", autoDecodeSMS(l))
		}
	}

	// m.send("AT+CMGD=" + index)
}*/

// unused but registered
func (m *Modem) handleConnectionType(line string) {
	if line == "+CNSMOD: 0" {
		return
	}

	re := regexp.MustCompile(`\+CNSMOD:\s*(\d+),(\d+)`)
	matches := re.FindStringSubmatch(line)
	knownGens := map[int]string{
		1:  "G",   // 2g (GSM)
		2:  "G",   // 2g (GPRS)
		3:  "E",   // 2g (EDGE)
		4:  "3G",  // 3g (WCDMA)
		5:  "H",   // 3g (HSDPA)
		6:  "H",   // 3g (HSUPA)
		7:  "H+",  // 3g (HSDPA+HSUPA)
		8:  "LTE", // 4g (LTE)
		9:  "3G",  // 3g (TDS-CDMA)
		10: "3G",  // 3g (TDS-HSDPA)
		11: "3G",  // 3g (TDS-HSUPA)
		12: "3G",  // 3g (TDS-HSPA)
		13: "1x",  // 2g (CDMA)
		14: "EV",  // 3g (EVDO)
		15: "EV",  // 3g (CDMA+EVDO)
		16: "LTE", // 4g (1XLTE)
		23: "EV",  // 3g (EHRPD)
		24: "3G",  // 3g (CDMA+EHRPD)
	}

	if len(matches) > 1 {
		gen, _ := strconv.Atoi(matches[len(matches)-1])
		if val, ok := knownGens[gen]; ok {
			m.NetworkGeneration = val
			if m.DebugMode {
				log.Printf("📶 Connected to a %s network", val)
			}
			m.Connected = true
		} else {
			m.NetworkGeneration = ""
			m.Carrier = "Searching..."
			m.Connected = false
		}
	}
}
func (m *Modem) handleSIMCard(string) {}
func (m *Modem) handleCPIN(line string) {
	re := regexp.MustCompile(`\+CPIN:\s*(.*)`)
	if matches := re.FindStringSubmatch(line); len(matches) > 1 {
		status := strings.TrimSpace(matches[1])
		log.Printf("🔒 SIM Status: %s", status)

		switch status {
		case "READY":
			m.SimCardInserted = true
		// TODO: add the rest
		case "SIM PIN":
		case "SIM PUK":
		case "PH-SIM PIN":
		case "SIM PIN2":
		case "SIM PUK2":
		case "PH-NET PIN":
		}
	}
}
func (m *Modem) handleCMEE(string) {}
func (m *Modem) handleCMEError(line string) {

	re := regexp.MustCompile(`\+CME ERROR:\s*(\d+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		code := matches[1]

		log.Printf("🚫 CME Error: %s", code)

		switch code {
		case "10": // No SIM card
			m.SimCardInserted = false
			m.NetworkGeneration = ""
			m.Carrier = "Insert SIM card"
			log.Printf("🚫 No SIM card inserted!")
		case "14": // SIM busy (ignore if we're starting up)
			log.Println("⚠️ SIM card is busy...")
		}
	}
}

func (m *Modem) handleNoCarrier(string) {
	m.CallEndChan <- true
}

// event listener
func (m *Modem) MonitorEvents() {
	for line := range m.urcChan {
		m.HandleEvent(line)
	}
}

func (m *Modem) HandleEvent(event string) {
	lines := strings.SplitSeq(strings.TrimSpace(event), "\n")
	for line := range lines {
		for prefix := range m.handlers {
			if strings.HasPrefix(line, prefix) {
				go m.handlers[prefix](line)
				break
			}
		}
	}
}

func Run(debug bool) *Modem {
	modem, err := NewModem("/dev/ttyUSB2", 115200, debug)
	if err != nil {
		log.Println(err)
		return nil
	}

	go modem.MonitorEvents()

	if debug {
		log.Println("📡 Modem initialized and monitoring events")
	}

	/* go func() {
		time.Sleep(8 * time.Second)

		// Simulate an incoming call
		modem.SimulationMode = true
		modem.handleCallStatus("+CLCC: 1,1,4,0,0,\"TEST NUMBER\",0")
	}() */

	return modem
}

func (m *Modem) SwitchToPowerSaveMode() {
	if m.DebugMode {
		log.Println("📡 Modem switching to low power mode")
	}

	for _, cmd := range []string{
		"AT+AUTOCSQ=0,0",
	} {
		resp, _ := m.send(cmd)
		m.HandleEvent(resp)
	}
}

func (m *Modem) SwitchToNormalMode() {
	if m.DebugMode {
		log.Println("📡 Modem switching to normal power mode")
	}
	for _, cmd := range []string{
		"AT+CSQ",
		"AT+COPS?",
		"AT+AUTOCSQ=1,1",
	} {
		resp, _ := m.send(cmd)
		m.HandleEvent(resp)
	}
}
