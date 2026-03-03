package modem

import (
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/maltegrosse/go-modemmanager"
)

// Call represents an ongoing or incoming call.
type Call struct {
	DBusPath  dbus.ObjectPath
	Call      modemmanager.Call
	StartTime time.Time
	State     string
	Number    string
	Active    bool
	Ended     chan bool
}

func (c *Call) String() string {
	return fmt.Sprintf("%s @ %v (%s)", c.Number, time.Since(c.StartTime), c.State)
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

	// Lifecycle
	Stop()
}
