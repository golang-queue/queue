package mocks

import _ "github.com/golang/mock/mockgen/model"

//go:generate mockgen -package=mocks -destination=mock_worker.go github.com/golang-queue/queue/core Worker
//go:generate mockgen -package=mocks -destination=mock_message.go github.com/golang-queue/queue/core QueuedMessage
