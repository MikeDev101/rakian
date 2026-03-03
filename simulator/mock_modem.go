package simulator

import (
	"log"
	"time"

	"modem"
)

// MockModem is a mock implementation of the Modem interface for testing.
type MockModem struct {
	ok               bool
	signalStrength   int
	flightMode       bool
	carrier          string
	roaming          bool
	sos              bool
	registered       bool
	emergencyNumbers []string
	unreadVoicemails int
	ringingChan      chan *modem.Call

	// Internal state for simulation
	Calls map[string]*modem.Call
}

// Software modem implementation
func (v *Simulator) Phone() modem.Modem { return v.mockModem }

// OK returns the modem status.
func (m *MockModem) OK() bool { return m.ok }

// GetSignalStrength returns the signal strength.
func (m *MockModem) GetSignalStrength() int { return m.signalStrength }

// IsFlightMode returns true if in flight mode.
func (m *MockModem) IsFlightMode() bool { return m.flightMode }

// GetCarrier returns the carrier name.
func (m *MockModem) GetCarrier() string { return m.carrier }

// IsRoaming returns true if roaming.
func (m *MockModem) IsRoaming() bool { return m.roaming }

// IsSOS returns true if in SOS mode.
func (m *MockModem) IsSOS() bool { return m.sos }

// IsRegistered returns true if registered to a network.
func (m *MockModem) IsRegistered() bool { return m.registered }

// GetEmergencyNumbers returns the list of emergency numbers.
func (m *MockModem) GetEmergencyNumbers() []string { return m.emergencyNumbers }

// GetUnreadVoicemails returns the count of unread voicemails.
func (m *MockModem) GetUnreadVoicemails() int { return m.unreadVoicemails }

// Ringing returns the channel for incoming calls.
func (m *MockModem) Ringing() <-chan *modem.Call { return m.ringingChan }

// SetFlightMode enables or disables flight mode.
func (m *MockModem) SetFlightMode(state bool) {
	m.flightMode = state
	log.Printf("[MockModem] SetFlightMode: %v", state)
}

// PlaceCall initiates a call to the given number.
func (m *MockModem) PlaceCall(number string) (*modem.Call, error) {
	log.Printf("[MockModem] Placing call to %s", number)
	call := &modem.Call{
		StartTime: time.Now(),
		State:     "outgoing",
		Number:    number,
		Active:    true,
		Ended:     make(chan bool, 1),
	}
	m.Calls[number] = call
	return call, nil
}

// AnswerCall answers the given call.
func (m *MockModem) AnswerCall(call *modem.Call) error {
	log.Printf("[MockModem] Answering call from %s", call.Number)
	call.State = "active"
	call.Active = true
	call.StartTime = time.Now()
	return nil
}

// HangupCall terminates the given call.
func (m *MockModem) HangupCall(call *modem.Call) error {
	log.Printf("[MockModem] Hanging up call %s", call.Number)
	call.State = "terminated"
	call.Active = false
	select {
	case call.Ended <- true:
	default:
	}
	delete(m.Calls, call.Number)
	return nil
}

// HangupAll terminates all calls.
func (m *MockModem) HangupAll() error {
	log.Println("[MockModem] Hanging up all calls")
	for _, call := range m.Calls {
		m.HangupCall(call)
	}
	return nil
}

// SendDTMF sends tones to the call.
func (m *MockModem) SendDTMF(call *modem.Call, tones string) error {
	log.Printf("[MockModem] Sending DTMF %s to %s", tones, call.Number)
	return nil
}

// HoldCall holds the call.
func (m *MockModem) HoldCall(call *modem.Call) error {
	log.Printf("[MockModem] Holding call %s", call.Number)
	call.State = "held"
	return nil
}

// UnholdCall unholds the call.
func (m *MockModem) UnholdCall(call *modem.Call) error {
	log.Printf("[MockModem] Unholding call %s", call.Number)
	call.State = "active"
	return nil
}

// SyncVoicemails simulates syncing voicemails.
func (m *MockModem) SyncVoicemails() {
	log.Println("[MockModem] Syncing voicemails (noop)")
}

// Stop stops the mock modem.
func (m *MockModem) Stop() {
	log.Println("[MockModem] Stopping")
}

// SimulateIncomingCall triggers an incoming call event.
func (m *MockModem) SimulateIncomingCall(number string) {
	call := &modem.Call{
		StartTime: time.Now(),
		State:     "incoming",
		Number:    number,
		Active:    false,
		Ended:     make(chan bool, 1),
	}
	m.Calls[number] = call
	m.ringingChan <- call
}
