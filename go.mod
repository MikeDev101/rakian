module rakian

go 1.26.0

replace (
	db => ./db
	eg25_g => ./eg25_g
	gfx => ./gfx
	hardware_keypad => ./hardware_keypad
	keypad => ./keypad
	menu => ./menu
	misc => ./misc
	modem => ./modem
	pcd8544 => ./pcd8544
	serial_audio => ./sim7600x/serial_audio
	sim7600x => ./sim7600x
	simulator => ./simulator
	timers => ./timers
	tones => ./tones
)

require (
	db v0.0.0-00010101000000-000000000000
	eg25_g v0.0.0-00010101000000-000000000000
	gfx v0.0.0-00010101000000-000000000000
	github.com/Wifx/gonetworkmanager/v3 v3.2.0
	github.com/glebarez/sqlite v1.11.0
	github.com/godbus/dbus/v5 v5.2.2
	golang.org/x/sys v0.41.0
	gorm.io/gorm v1.31.1
	keypad v0.0.0-00010101000000-000000000000
	menu v0.0.0-00010101000000-000000000000
	misc v0.0.0-00010101000000-000000000000
	modem v0.0.0-00010101000000-000000000000
	simulator v0.0.0-00010101000000-000000000000
	timers v0.0.0-00010101000000-000000000000
	tones v0.0.0-00010101000000-000000000000
)

require (
	fyne.io/fyne/v2 v2.7.3 // indirect
	fyne.io/systray v1.12.0 // indirect
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/oto/v3 v3.4.0 // indirect
	github.com/ebitengine/purego v0.9.0 // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/fredbi/uri v1.1.1 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fyne-io/gl-js v0.2.0 // indirect
	github.com/fyne-io/glfw-js v0.3.0 // indirect
	github.com/fyne-io/image v0.1.1 // indirect
	github.com/fyne-io/oksvg v0.2.0 // indirect
	github.com/glebarez/go-sqlite v1.21.2 // indirect
	github.com/go-gl/gl v0.0.0-20231021071112-07e5d0ea2e71 // indirect
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20250301202403-da16c1255728 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-text/render v0.2.1 // indirect
	github.com/go-text/typesetting v0.3.4 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hack-pad/go-indexeddb v0.3.2 // indirect
	github.com/hack-pad/safejs v0.1.1 // indirect
	github.com/hajimehoshi/go-mp3 v0.3.4 // indirect
	github.com/jeandeaual/go-locale v0.0.0-20250612000132-0ef82f21eade // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jsummers/gobmp v0.0.0-20230614200233-a9de23ed2e25 // indirect
	github.com/maltegrosse/go-modemmanager v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646 // indirect
	github.com/nicksnyder/go-i18n/v2 v2.6.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rymdport/portal v0.4.2 // indirect
	github.com/saltosystems/winrt-go v0.0.0-20240509164145-4f7860a3bd2b // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/soypat/cyw43439 v0.0.0-20250505012923-830110c8f4af // indirect
	github.com/soypat/seqs v0.0.0-20250124201400-0d65bc7c1710 // indirect
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c // indirect
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/tinygo-org/cbgo v0.0.4 // indirect
	github.com/tinygo-org/pio v0.2.0 // indirect
	github.com/yuin/goldmark v1.7.16 // indirect
	golang.org/x/exp v0.0.0-20241204233417-43b7b7cde48d // indirect
	golang.org/x/image v0.36.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/sqlite v1.23.1 // indirect
	tinygo.org/x/bluetooth v0.14.0 // indirect
)
