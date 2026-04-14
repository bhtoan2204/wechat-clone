package socket

import "context"

//go:generate mockgen -package=socket -destination=interfaces_mock.go -source=interfaces.go
type IClient interface {
	GetID() string
	GetUserID() string
	Send(ctx context.Context, message []byte)
	ReadPump(ctx context.Context, hub IHub)
	WritePump(ctx context.Context)
	Close(ctx context.Context)
}

//go:generate mockgen -package=socket -destination=interfaces_mock.go -source=interfaces.go
type IRoom interface {
	GetID() string
	AddClient(ctx context.Context, client IClient)
	RemoveClient(ctx context.Context, client IClient)
	Broadcast(ctx context.Context, message []byte)
	IsEmpty() bool
	ClientCount() int
}

//go:generate mockgen -package=socket -destination=interfaces_mock.go -source=interfaces.go
type IHub interface {
	Register(ctx context.Context, client IClient)
	Unregister(ctx context.Context, client IClient)

	JoinRoom(ctx context.Context, client IClient, roomID string) error
	LeaveRoom(ctx context.Context, client IClient, roomID string) error

	HandleMessage(ctx context.Context, client IClient, msg Message) error
	Close(ctx context.Context)
}
