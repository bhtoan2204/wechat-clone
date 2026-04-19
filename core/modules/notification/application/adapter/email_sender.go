package adapter

import "context"

//go:generate mockgen -package=adapter -destination=email_sender_mock.go -source=email_sender.go
type EmailSender interface {
	SendTemplate(ctx context.Context, to, subject, templateName string, data any) error
}
