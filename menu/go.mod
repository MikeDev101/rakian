module menu

go 1.26.0

replace (
	db => ../db
	gfx => ../gfx
	keypad => ../keypad
	menu => ../menu
	misc => ../misc
	modem => ../modem
	pcd8544 => ../pcd8544
	serial_audio => ../modem/serial_audio
	timers => ../timers
	tones => ../tones
)

require (
	db v0.0.0-00010101000000-000000000000
	gfx v0.0.0-00010101000000-000000000000
	github.com/Wifx/gonetworkmanager/v3 v3.2.0
	github.com/google/uuid v1.6.0
	github.com/maltegrosse/go-modemmanager v0.1.4
	gorm.io/gorm v1.31.1
	keypad v0.0.0-00010101000000-000000000000
	misc v0.0.0-00010101000000-000000000000
	modem v0.0.0-00010101000000-000000000000
	timers v0.0.0-00010101000000-000000000000
	tinygo.org/x/bluetooth v0.14.0
	tones v0.0.0-00010101000000-000000000000
)

require (
	github.com/ebitengine/oto/v3 v3.4.0 // indirect
	github.com/ebitengine/purego v0.9.0 // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/hajimehoshi/go-mp3 v0.3.4 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/saltosystems/winrt-go v0.0.0-20240509164145-4f7860a3bd2b // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/soypat/cyw43439 v0.0.0-20250505012923-830110c8f4af // indirect
	github.com/soypat/seqs v0.0.0-20250124201400-0d65bc7c1710 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	github.com/tinygo-org/cbgo v0.0.4 // indirect
	github.com/tinygo-org/pio v0.2.0 // indirect
	golang.org/x/exp v0.0.0-20241204233417-43b7b7cde48d // indirect
	golang.org/x/image v0.36.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.34.0 // indirect
)
