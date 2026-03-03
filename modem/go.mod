module modem

go 1.26.0

replace (
	db => ../db
	serial_audio => ./serial_audio
)

require (
	github.com/godbus/dbus/v5 v5.2.2
	github.com/maltegrosse/go-modemmanager v0.1.4
)

require golang.org/x/sys v0.27.0 // indirect
