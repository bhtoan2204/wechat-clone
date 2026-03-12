package server

import (
	"fmt"
	notificationmessaging "go-socket/core/modules/notification/application/messaging"
	stackerr "go-socket/core/shared/pkg/stackErr"
)

type Server interface {
	Start() error
	Stop() error
}

type notificationServer struct {
	messageHandler notificationmessaging.MessageHandler
}

func NewServer(messageHandler notificationmessaging.MessageHandler) (Server, error) {
	if messageHandler == nil {
		return nil, stackerr.Error(fmt.Errorf("message handler can not be nil"))
	}
	return &notificationServer{messageHandler: messageHandler}, nil
}

func (s *notificationServer) Start() error {
	return s.messageHandler.Start()
}

func (s *notificationServer) Stop() error {
	return s.messageHandler.Stop()
}
