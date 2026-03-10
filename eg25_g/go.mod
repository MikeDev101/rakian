module eg25_g

go 1.26.0

replace (
	db => ../db
	modem => ../modem
)

require (
	db v0.0.0-00010101000000-000000000000
	github.com/godbus/dbus/v5 v5.2.2
	github.com/google/uuid v1.6.0
	github.com/maltegrosse/go-modemmanager v0.1.4
	gorm.io/gorm v1.31.1
	modem v0.0.0-00010101000000-000000000000
)

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/sys v0.27.0 // indirect
	golang.org/x/text v0.20.0 // indirect
)
