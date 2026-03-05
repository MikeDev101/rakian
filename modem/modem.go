package modem

import (
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/maltegrosse/go-modemmanager"
)

// Call represents an ongoing or incoming call.
type Call struct {
	ID        string
	DBusPath  dbus.ObjectPath
	Call      modemmanager.Call
	StartTime time.Time
	State     modemmanager.MMCallState
	Reason    modemmanager.MMCallStateReason
	Mute      bool
	Volume    float64
	Ended     chan bool
	Number    string
	Announced bool
}

func (c *Call) String() string {
	state := fmt.Sprintf("%s - %s @ %v (%s)", c.ID, c.Number, time.Since(c.StartTime), c.State)
	if c.Mute {
		state += " (mute)"
	}
	if c.Volume == 0 {
		state += " (deafened)"
	}
	return state
}

// Modem is an interface that abstracts modem functionalities.
type Modem interface {
	// Status and Properties
	OK() bool
	GetSignalStrength() int
	IsFlightMode() bool
	GetCarrier() string
	IsRoaming() bool
	IsSOS() bool
	IsRegistered() bool
	GetEmergencyNumbers() []string
	GetUnreadVoicemails() int
	Ringing() <-chan *Call

	// Actions
	SetFlightMode(state bool)
	PlaceCall(number string) (*Call, error)
	AnswerCall(call *Call) error
	HangupCall(call *Call) error
	HangupAll() error
	SendDTMF(call *Call, tones string) error
	HoldCall(call *Call) error
	UnholdCall(call *Call) error
	SyncVoicemails()
	GetCalls() []*Call

	// Lifecycle
	Stop()
}
