package db

import (
	"time"

	"github.com/maltegrosse/go-modemmanager"
)

// KVStore represents the database schema
type KVStore struct {
	Key   string `gorm:"primaryKey;uniqueIndex"`
	Value any    `gorm:"serializer:json"`
}

type Messages struct {
	MessageID int    `gorm:"primaryKey;autoIncrement"`
	Message   string `gorm:"serializer:json"`
	Number    string
	Read      bool
	Received  time.Time
}

type Contacts struct {
	Number string `gorm:"primaryKey;uniqueIndex"`
	First  string
	Last   string
	Tone   int
}

type CallSessionEvents struct {
	EventID  int `gorm:"primaryKey;autoIncrement"`
	Number   string
	Received time.Time
	Duration time.Duration
}

type CallStateEvents struct {
	EventID   int `gorm:"primaryKey;autoIncrement"`
	Number    string
	Reason    modemmanager.MMCallStateReason
	Timestamp time.Time
}
