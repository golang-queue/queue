module example

go 1.18

require (
	github.com/golang-queue/contrib v1.0.0
	github.com/golang-queue/queue v0.0.7
)

require (
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/rs/zerolog v1.32.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
)

replace github.com/golang-queue/queue => ../../
