module hardware_keypad

go 1.26.0

replace (
	keypad => ../keypad
	timers => ../timers
)

require (
	github.com/stianeikeland/go-rpio/v4 v4.6.0
	keypad v0.0.0-00010101000000-000000000000
	timers v0.0.0-00010101000000-000000000000
)
