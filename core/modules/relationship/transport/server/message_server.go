package server

import (
	"fmt"

	relationshipmessaging "wechat-clone/core/modules/relationship/application/messaging"
	"wechat-clone/core/shared/pkg/stackErr"
)

type Server interface {
	Start() error
	Stop() error
}

type relationshipServer struct {
	messageHandler relationshipmessaging.MessageHandler
}

func NewServer(messageHandler relationshipmessaging.MessageHandler) (Server, error) {
	if messageHandler == nil {
		return nil, stackErr.Error(fmt.Errorf("message handler can not be nil"))
	}
	return &relationshipServer{messageHandler: messageHandler}, nil
}

func (s *relationshipServer) Start() error {
	return s.messageHandler.Start()
}

func (s *relationshipServer) Stop() error {
	return s.messageHandler.Stop()
}
