module serial_audio

go 1.26.0

replace modem => ../../modem

require (
	github.com/mesilliac/pulse-simple v0.0.0-20170506101341-75ac54e19fdf
	go.bug.st/serial v1.6.4
	modem v0.0.0-00010101000000-000000000000
)

require (
	github.com/creack/goselect v0.1.2 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/maltegrosse/go-modemmanager v0.1.4 // indirect
	golang.org/x/sys v0.27.0 // indirect
)
