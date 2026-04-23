package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"wechat-clone/core/shared/infra/smtp"
	"wechat-clone/core/shared/pkg/logging"

	"go.uber.org/zap"
)

//go:generate mockgen -package=service -destination=email_verification_service_mock.go -source=email_verification_service.go
type EmailVerificationService interface {
	SendTemplate(ctx context.Context, to, subject, templateName string, data any) error
	SendVerificationEmail(ctx context.Context, to, verificationURL string, expiresAt time.Time) error
}

type emailVerificationService struct {
	smtp smtp.SMTP
}

func newEmailVerificationService(smtp smtp.SMTP) EmailVerificationService {
	return &emailVerificationService{
		smtp: smtp,
	}
}

type verificationEmailTemplateData struct {
	VerificationURL string
	ExpiresAt       time.Time
}

func (s *emailVerificationService) SendVerificationEmail(
	ctx context.Context,
	to, verificationURL string,
	expiresAt time.Time,
) error {
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("recipient is required")
	}
	if strings.TrimSpace(verificationURL) == "" {
		return fmt.Errorf("verification url is required")
	}

	return s.smtp.SendTemplate(
		ctx,
		to,
		"Verify your email",
		"verify_email",
		verificationEmailTemplateData{
			VerificationURL: verificationURL,
			ExpiresAt:       expiresAt,
		},
	)
}

func (s *emailVerificationService) SendTemplate(ctx context.Context, to, subject, templateName string, data any) error {
	log := logging.FromContext(ctx).Named("SendTemplate")
	if err := s.smtp.SendTemplate(ctx, to, subject, templateName, data); err != nil {
		log.Warnw("SendTemplate failed", zap.Error(err))
	}
	return nil
}
