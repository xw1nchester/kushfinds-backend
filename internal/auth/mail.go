package auth

import (
	"fmt"
	"net/smtp"

	"github.com/vetrovegor/kushfinds-backend/internal/config"
)

type mailManager struct {
	smtpConfig config.SMTP
}

func NewMailManager(smtpConfig config.SMTP) *mailManager {
	return &mailManager{
		smtpConfig: smtpConfig,
	}
}

func (m mailManager) SendMail(subject string, body string, to []string) error {
	auth := smtp.PlainAuth(
		"",
		m.smtpConfig.Username,
		m.smtpConfig.Password,
		m.smtpConfig.Host,
	)

	msg := "Subject: " + subject + "\n" + body

	return smtp.SendMail(
		fmt.Sprintf("%s:%s", m.smtpConfig.Host, m.smtpConfig.Port),
		auth,
		m.smtpConfig.Username,
		to,
		[]byte(msg),
	)
}
