module example

go 1.22

require (
	github.com/golang-queue/contrib v1.1.0
	github.com/golang-queue/queue v0.2.0
)

require (
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/rs/zerolog v1.33.0 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
)

replace github.com/golang-queue/queue => ../../
