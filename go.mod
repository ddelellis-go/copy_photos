module copy_photos

go 1.25.0

replace github.com/ddelellis-pkg/progbar => ../progbar

require (
	github.com/ddelellis-pkg/progbar v0.0.0-20260609010204-f7106e1e6647
	github.com/jessevdk/go-flags v1.6.1
	github.com/moby/sys/mount v0.3.5-0.20260529155943-fc52b7222d0b
	golang.org/x/net v0.55.0
)

require (
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	golang.org/x/sys v0.46.0 // indirect
)
