module example

go 1.18

require (
	github.com/golang-queue/contrib v0.0.1
	github.com/golang-queue/queue v0.0.7
)

require github.com/rs/zerolog v1.26.1 // indirect

replace github.com/golang-queue/queue => ../../
