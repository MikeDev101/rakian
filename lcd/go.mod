module lcd

go 1.26.0

replace gfx => ../gfx

require (
	gfx v0.0.0-00010101000000-000000000000
	github.com/fogleman/gg v1.3.0
	github.com/stianeikeland/go-rpio/v4 v4.6.0
)

require (
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	golang.org/x/image v0.36.0 // indirect
)
