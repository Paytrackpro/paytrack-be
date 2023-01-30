package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
)

type Config struct {
	Addr     string `yaml:"addr"`
	UserName string `yaml:"userName"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	From     string `yaml:"from"`
}

type MailClient struct {
	conf *Config
	tmpl *template.Template
}

func NewMailClient(conf Config) (*MailClient, error) {
	tmpl, err := template.New("").Parse(paymentNotify)
	if err != nil {
		return nil, err
	}
	return &MailClient{
		conf: &conf,
		tmpl: tmpl,
	}, nil
}

func (m *MailClient) Send(subject, tmplName string, data interface{}, toMails ...string) error {
	if len(toMails) == 0 {
		return fmt.Errorf("mail to must be required")
	}
	var w bytes.Buffer
	fmt.Fprintf(&w, "From: %s\n", m.conf.From)
	fmt.Fprintf(&w, "Subject: %s\n", subject)
	fmt.Fprintf(&w, "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n")

	err := m.tmpl.ExecuteTemplate(&w, tmplName, data)
	if err != nil {
		return err
	}
	return smtp.SendMail(m.conf.Addr,
		smtp.PlainAuth("", m.conf.UserName, m.conf.Password, m.conf.Host),
		m.conf.From, toMails, w.Bytes())
}
