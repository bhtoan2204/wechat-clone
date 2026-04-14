package server

import (
	"fmt"
	"go-socket/core/modules/room/application/messaging"
	"go-socket/core/shared/pkg/stackErr"
)

//go:generate mockgen -package=server -destination=message_server_mock.go -source=message_server.go
type Server interface {
	Start() error
	Stop() error
}

type notificationServer struct {
	messageHandler messaging.MessageHandler
}

func NewServer(messageHandler messaging.MessageHandler) (Server, error) {
	if messageHandler == nil {
		return nil, stackErr.Error(fmt.Errorf("message handler can not be nil"))
	}
	return &notificationServer{messageHandler: messageHandler}, nil
}

func (s *notificationServer) Start() error {
	return s.messageHandler.Start()
}

func (s *notificationServer) Stop() error {
	return s.messageHandler.Stop()
}
