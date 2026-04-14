package server

import (
	"fmt"

	ledgermessaging "go-socket/core/modules/ledger/application/messaging"
	"go-socket/core/shared/pkg/stackErr"
)

//go:generate mockgen -package=server -destination=message_server_mock.go -source=message_server.go
type Server interface {
	Start() error
	Stop() error
}

type ledgerServer struct {
	messageHandler ledgermessaging.MessageHandler
}

func NewServer(messageHandler ledgermessaging.MessageHandler) (Server, error) {
	if messageHandler == nil {
		return nil, stackErr.Error(fmt.Errorf("message handler can not be nil"))
	}
	return &ledgerServer{messageHandler: messageHandler}, nil
}

func (s *ledgerServer) Start() error {
	return s.messageHandler.Start()
}

func (s *ledgerServer) Stop() error {
	return s.messageHandler.Stop()
}
