module example

go 1.26.10

require (
	github.com/golang-queue/contrib v1.2.0
	github.com/golang-queue/queue v0.6.2
)

require (
	github.com/jpillora/backoff v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.16 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/rs/zerolog v1.35.1 // indirect
	golang.org/x/sys v0.50.0 // indirect
)

replace github.com/golang-queue/queue => ../../
