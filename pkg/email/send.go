package email

import (
	"gopkg.in/gomail.v2"

	"show-live/config"
)

type EmailSender struct {
	Conf config.EmailConfig
}

func NewEmailSender(conf config.EmailConfig) *EmailSender {
	return &EmailSender{
		Conf: conf,
	}
}

func (e *EmailSender) Send(title, content string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", e.Conf.From)
	m.SetHeader("To", e.Conf.To)

	m.SetHeader("Subject", title)

	m.SetBody("text/html", content)

	d := gomail.NewPlainDialer(e.Conf.Server, e.Conf.Port, e.Conf.From, e.Conf.Password)
	err := d.DialAndSend(m)
	return err
}
