package simulator

import (
	"log"
	"sync"
	"time"

	"modem"

	"github.com/google/uuid"
	"github.com/maltegrosse/go-modemmanager"
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
	mute             bool
	emergencyNumbers []string
	unreadVoicemails int
	ringingChan      chan *modem.Call

	// Internal state for simulation
	Calls map[string]*modem.Call

	subscribers   map[string]func(event modem.IMSEvent) error
	subscribersMu sync.RWMutex
}

func (m *MockModem) Subscribe(callback func(event modem.IMSEvent) error) string {
	m.subscribersMu.Lock()
	defer m.subscribersMu.Unlock()
	if m.subscribers == nil {
		m.subscribers = make(map[string]func(modem.IMSEvent) error)
	}
	key := uuid.New().String()
	m.subscribers[key] = callback
	return key
}

func (m *MockModem) Unsubscribe(key string) {
	m.subscribersMu.Lock()
	defer m.subscribersMu.Unlock()
	if m.subscribers != nil {
		delete(m.subscribers, key)
	}
}

func (m *MockModem) publish(event modem.IMSEvent) {
	m.subscribersMu.RLock()
	defer m.subscribersMu.RUnlock()
	for _, callback := range m.subscribers {
		go func(cb func(modem.IMSEvent) error) {
			if err := cb(event); err != nil {
				log.Printf("⚠️ Subscriber callback failed: %v", err)
			}
		}(callback)
	}
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
		ID:        uuid.New().String(),
		StartTime: time.Time{},
		State:     modemmanager.MmCallStateDialing,
		Number:    number,
		Mute:      false,
		Volume:    0.5,
		Ended:     make(chan bool, 1),
		Announced: false,
	}
	m.Calls[call.ID] = call
	m.publish(modem.IMSEvent{Type: modem.IMS_Call_Outgoing, Data: call})
	go func() {
		time.Sleep(5 * time.Second)
		log.Printf("[MockModem] Simulated call %s ringing", call)
		call.State = modemmanager.MmCallStateRingingOut
		m.publish(modem.IMSEvent{Type: modem.IMS_Call_Ringing_Out, Data: call})
		time.Sleep(3 * time.Second)
		log.Printf("[MockModem] Simulated call %s active", call)
		call.State = modemmanager.MmCallStateActive
		call.StartTime = time.Now()
		m.publish(modem.IMSEvent{Type: modem.IMS_Call_Connected, Data: call})
	}()
	return call, nil
}

// AnswerCall answers the given call.
func (m *MockModem) AnswerCall(call *modem.Call) error {
	log.Printf("[MockModem] Answering call from %s", call)
	call.State = modemmanager.MmCallStateActive
	call.StartTime = time.Now()
	m.publish(modem.IMSEvent{Type: modem.IMS_Call_Connected, Data: call})
	return nil
}

// HangupCall terminates the given call.
func (m *MockModem) HangupCall(call *modem.Call) error {
	log.Printf("[MockModem] Hanging up call %s", call)
	call.State = modemmanager.MmCallStateTerminated
	m.publish(modem.IMSEvent{Type: modem.IMS_Call_Terminated, Data: call})
	select {
	case call.Ended <- true:
	default:
	}
	delete(m.Calls, call.ID)
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
	log.Printf("[MockModem] Sending DTMF %s to %s", tones, call)
	return nil
}

// HoldCall holds the call.
func (m *MockModem) HoldCall(call *modem.Call) error {
	log.Printf("[MockModem] Holding call %s", call)
	call.State = modemmanager.MmCallStateHeld
	m.publish(modem.IMSEvent{Type: modem.IMS_Call_Held, Data: call})
	return nil
}

// UnholdCall unholds the call.
func (m *MockModem) UnholdCall(call *modem.Call) error {
	log.Printf("[MockModem] Unholding call %s", call)
	call.State = modemmanager.MmCallStateActive
	m.publish(modem.IMSEvent{Type: modem.IMS_Call_Unhheld, Data: call})
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
		ID:        uuid.New().String(),
		StartTime: time.Time{},
		State:     modemmanager.MmCallStateRingingIn,
		Number:    number,
		Mute:      false,
		Volume:    0.5,
		Ended:     make(chan bool, 1),
		Announced: false,
	}
	m.Calls[call.ID] = call
	m.publish(modem.IMSEvent{Type: modem.IMS_Call_Incoming, Data: call})
	m.ringingChan <- call
}

func (m *MockModem) GetCalls() []*modem.Call {
	calls := make([]*modem.Call, 0, len(m.Calls))
	for _, call := range m.Calls {
		calls = append(calls, call)
	}
	return calls
}
