// Package mailer sends transactional e-mail (password reset, order confirmation).
// With no SMTP_HOST configured it only logs, so local/dev runs keep working.
package mailer

import (
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

type Mailer struct {
	host, port, user, pass, from string
	log                          *slog.Logger
}

func New(host, port, user, pass, from string, log *slog.Logger) *Mailer {
	if port == "" {
		port = "587"
	}
	return &Mailer{host: host, port: port, user: user, pass: pass, from: from, log: log}
}

func (m *Mailer) Enabled() bool { return m.host != "" && m.from != "" }

// header values must not contain CR/LF (header injection).
func clean(s string) string { return strings.NewReplacer("\r", " ", "\n", " ").Replace(s) }

// Send delivers a plain-text message. Errors are returned so callers can log
// them, but callers must never block a user-facing request on e-mail.
func (m *Mailer) Send(to, subject, body string) error {
	if !m.Enabled() {
		m.log.Info("email not sent (SMTP not configured)", "to", to, "subject", subject)
		return nil
	}
	msg := "From: " + clean(m.from) + "\r\nTo: " + clean(to) + "\r\nSubject: " + clean(subject) +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body
	var auth smtp.Auth
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}
	if err := smtp.SendMail(m.host+":"+m.port, auth, m.from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}
