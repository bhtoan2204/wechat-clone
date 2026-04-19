package smtp

import (
	"strings"
	"testing"
	"wechat-clone/core/shared/config"
)

func TestSMTP_RenderTemplate(t *testing.T) {
	cfg := &config.Config{
		SMTPConfig: config.SMTPConfig{
			Host:   "smtp.gmail.com",
			Port:   465,
			Secure: true,
			User:   "",
			Pass:   "",
			From:   "wechatclone@gmail.com",
		},
	}
	smtpClient := NewSMTP(cfg)

	body, err := smtpClient.RenderTemplate("otp.html", map[string]interface{}{
		"OTP":       "123456",
		"Name":      "John Doe",
		"ExpiredIn": "15 minutes",
	})
	if err != nil {
		t.Fatal("failed to render email template:", err)
	}
	if !strings.Contains(body, "123456") {
		t.Fatalf("expected rendered body to contain OTP, got %q", body)
	}
}
