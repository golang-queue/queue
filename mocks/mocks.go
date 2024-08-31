package mocks

//go:generate mockgen -package=mocks -destination=mock_worker.go github.com/golang-queue/queue/core Worker
//go:generate mockgen -package=mocks -destination=mock_message.go github.com/golang-queue/queue/core QueuedMessage
