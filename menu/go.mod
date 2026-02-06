module menu

go 1.25.2

replace (
	keypad => ../keypad
	misc => ../misc
	phone => ../phone
	sh1107 => ../sh1107
	timers => ../timers
	tones => ../tones
)

require (
	keypad v0.0.0-00010101000000-000000000000
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
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/sergeymakinen/go-bmp v1.0.0 // indirect
	github.com/tarm/serial v0.0.0-20180830185346-98f6abe2eb07 // indirect
	github.com/warthog618/sms v0.3.0 // indirect
	golang.org/x/image v0.35.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	periph.io/x/conn/v3 v3.7.2 // indirect
	periph.io/x/host/v3 v3.8.5 // indirect
)
