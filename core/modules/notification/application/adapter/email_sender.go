package adapter

import "context"

//go:generate mockgen -package=adapter -destination=email_sender_mock.go -source=email_sender.go
type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}
