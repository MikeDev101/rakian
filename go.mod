module rakian

go 1.25.2

replace (
	db => ./db
	keypad => ./keypad
	menu => ./menu
	misc => ./misc
	phone => ./phone
	sh1107 => ./sh1107
	timers => ./timers
	tones => ./tones
)

require (
	db v0.0.0-00010101000000-000000000000
	github.com/Wifx/gonetworkmanager/v3 v3.2.0
	github.com/glebarez/sqlite v1.11.0
	golang.org/x/sys v0.41.0
	gorm.io/gorm v1.31.1
	keypad v0.0.0-00010101000000-000000000000
	menu v0.0.0-00010101000000-000000000000
	misc v0.0.0-00010101000000-000000000000
	phone v0.0.0-00010101000000-000000000000
	sh1107 v0.0.0-00010101000000-000000000000
	timers v0.0.0-00010101000000-000000000000
	tones v0.0.0-00010101000000-000000000000
)

require (
	github.com/d2r2/go-i2c v0.0.0-20191123181816-73a8a799d6bc // indirect
	github.com/d2r2/go-logger v0.0.0-20210606094344-60e9d1233e22 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/glebarez/go-sqlite v1.21.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/saltosystems/winrt-go v0.0.0-20240509164145-4f7860a3bd2b // indirect
	github.com/sergeymakinen/go-bmp v1.0.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/soypat/cyw43439 v0.0.0-20250505012923-830110c8f4af // indirect
	github.com/soypat/seqs v0.0.0-20250124201400-0d65bc7c1710 // indirect
	github.com/tarm/serial v0.0.0-20180830185346-98f6abe2eb07 // indirect
	github.com/tinygo-org/cbgo v0.0.4 // indirect
	github.com/tinygo-org/pio v0.2.0 // indirect
	github.com/warthog618/sms v0.3.0 // indirect
	golang.org/x/exp v0.0.0-20241204233417-43b7b7cde48d // indirect
	golang.org/x/image v0.35.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/sqlite v1.23.1 // indirect
	periph.io/x/conn/v3 v3.7.2 // indirect
	periph.io/x/host/v3 v3.8.5 // indirect
	tinygo.org/x/bluetooth v0.14.0 // indirect
)
