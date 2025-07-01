package auth

import (
	"fmt"
	"net/smtp"

	"github.com/vetrovegor/kushfinds-backend/internal/config"
)

//go:generate mockgen -source=mail.go -destination=mocks/mock.go -package=mockmail
type MailManager interface {
	SendMail(subject string, body string, to []string) error
}

type mailManager struct {
	smtpConfig config.SMTP
}

func NewMailManager(smtpConfig config.SMTP) MailManager {
	return &mailManager{
		smtpConfig: smtpConfig,
	}
}

// TODO: возможно стоит принудительно завершать выполнение после 10 сек
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