package modem

import (
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/maltegrosse/go-modemmanager"
)

type IMSEventType int

const (
	IMS_New_Message IMSEventType = iota
	IMS_New_Voicemail
	IMS_Call_Incoming
	IMS_Call_Outgoing
	IMS_Call_Connected
	IMS_Call_Terminated
	IMS_Call_Held
	IMS_Call_Unhheld
	IMS_Call_Muted
	IMS_Call_Deafened
	IMS_Call_Undeafened
	IMS_Call_Ringing_Out
	IMS_Call_Ringing_In
	IMS_Call_Error
	IMS_Registration_Active
	IMS_Registration_Failure
	IMS_Registration_Roaming
	IMS_Registration_Emergency_Only
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

type IMSEvent struct {
	Type IMSEventType
	Data any
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
	Subscribe(func(event IMSEvent) error) string
	Unsubscribe(key string)

	// Lifecycle
	Stop()
}
