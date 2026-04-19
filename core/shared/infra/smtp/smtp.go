package smtp

import (
	"bytes"
	"context"
	"crypto/tls"
	"embed"
	"fmt"
	"mime"
	"net/smtp"
	"path/filepath"
	"strings"
	"text/template"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/logging"

	"go.uber.org/zap"
)

//go:embed template/*
var templateFS embed.FS

type SMTP struct {
	host      string
	port      int
	secure    bool
	user      string
	pass      string
	from      string
	templates *template.Template
}

func NewSMTP(cfg *config.Config) SMTP {
	tpl := template.Must(template.ParseFS(templateFS, "template/*"))

	return SMTP{
		host:      cfg.SMTPConfig.Host,
		port:      cfg.SMTPConfig.Port,
		secure:    cfg.SMTPConfig.Secure,
		user:      cfg.SMTPConfig.User,
		pass:      cfg.SMTPConfig.Pass,
		from:      cfg.SMTPConfig.From,
		templates: tpl,
	}
}

func (s SMTP) Send(ctx context.Context, to, subject, body string) error {
	log := logging.FromContext(ctx).Named("Send")

	if s.host == "" || s.port == 0 || s.user == "" || s.pass == "" || s.from == "" {
		return fmt.Errorf("smtp config is incomplete")
	}
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("recipient is required")
	}

	msg := buildMessage(s.from, to, subject, body)

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.pass, s.host)

	if s.secure {
		if err := s.sendWithTLS(addr, auth, to, msg); err != nil {
			log.Errorw("failed to send email via tls",
				zap.String("to", to),
				zap.String("subject", subject),
				zap.Error(err),
			)
			return err
		}
	} else {
		if err := s.sendWithSTARTTLS(addr, auth, to, msg); err != nil {
			log.Errorw("failed to send email via starttls",
				zap.String("to", to),
				zap.String("subject", subject),
				zap.Error(err),
			)
			return err
		}
	}

	log.Infow("email sent successfully",
		zap.String("to", to),
		zap.String("subject", subject),
	)

	return nil
}

func (s SMTP) SendTemplate(ctx context.Context, to, subject, templateName string, data any) error {
	body, err := s.RenderTemplate(templateName, data)
	if err != nil {
		return err
	}
	return s.Send(ctx, to, subject, body)
}

func (s SMTP) RenderTemplate(templateName string, data any) (string, error) {
	if s.templates == nil {
		return "", fmt.Errorf("templates not initialized")
	}

	name := filepath.Base(templateName)

	var buf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("render template %q: %w", name, err)
	}

	return buf.String(), nil
}

func (s SMTP) sendWithTLS(addr string, auth smtp.Auth, to string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: s.host,
		MinVersion: tls.VersionTLS12,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("new smtp client: %w", err)
	}
	defer client.Quit()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close writer: %w", err)
	}

	return nil
}

func (s SMTP) sendWithSTARTTLS(addr string, auth smtp.Auth, to string, msg []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer client.Quit()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: s.host,
			MinVersion: tls.VersionTLS12,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close writer: %w", err)
	}

	return nil
}

func buildMessage(from, to, subject, body string) []byte {
	encodedSubject := mime.QEncoding.Encode("utf-8", subject)

	headers := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", encodedSubject),
		"MIME-Version: 1.0",
		`Content-Type: text/html; charset="UTF-8"`,
		"",
		body,
	}

	return []byte(strings.Join(headers, "\r\n"))
}
