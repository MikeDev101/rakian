module misc

go 1.26.0

replace (
	lcd => ../lcd
	tones => ../tones
)

require (
	github.com/stianeikeland/go-rpio/v4 v4.6.0
	tones v0.0.0-00010101000000-000000000000
)

require (
	github.com/ebitengine/oto/v3 v3.4.0 // indirect
	github.com/ebitengine/purego v0.9.0 // indirect
	github.com/hajimehoshi/go-mp3 v0.3.4 // indirect
	golang.org/x/sys v0.36.0 // indirect
)
