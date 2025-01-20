package mocks

//go:generate mockgen -package=mocks -destination=mock_worker.go github.com/golang-queue/queue/core Worker
//go:generate mockgen -package=mocks -destination=mock_queued_message.go github.com/golang-queue/queue/core QueuedMessage
//go:generate mockgen -package=mocks -destination=mock_task_message.go github.com/golang-queue/queue/core TaskMessage
