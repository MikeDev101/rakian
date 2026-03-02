module modem

go 1.26.0

replace (
	db => ../db
	serial_audio => ./serial_audio
)

require (
	db v0.0.0-00010101000000-000000000000
	github.com/godbus/dbus/v5 v5.2.2
	github.com/maltegrosse/go-modemmanager v0.1.4
	gorm.io/gorm v1.31.1
	serial_audio v0.0.0-00010101000000-000000000000
)

require (
	github.com/creack/goselect v0.1.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mesilliac/pulse-simple v0.0.0-20170506101341-75ac54e19fdf // indirect
	go.bug.st/serial v1.6.4 // indirect
	golang.org/x/sys v0.27.0 // indirect
	golang.org/x/text v0.20.0 // indirect
)
