package notifier

import (
	"crypto/tls"
	"fmt"
	"math"
	"net/smtp"
	"time"

	"github.com/infrasense/notification-service/internal/webhook"
)

type EmailNotifier struct {
	smtpHost string
	smtpPort int
	username string
	password string
	from     string
	to       string
	useTLS   bool
}

func NewEmailNotifier(smtpHost string, smtpPort int, username, password, from, to string, useTLS bool) *EmailNotifier {
	return &EmailNotifier{
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		username: username,
		password: password,
		from:     from,
		to:       to,
		useTLS:   useTLS,
	}
}

func (e *EmailNotifier) Name() string {
	return "Email"
}

func (e *EmailNotifier) Send(alert webhook.NotificationAlert) error {
	return e.sendWithRetry(alert, 3)
}

func (e *EmailNotifier) sendWithRetry(alert webhook.NotificationAlert, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			time.Sleep(backoff)
		}

		err := e.sendEmail(alert)
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func (e *EmailNotifier) sendEmail(alert webhook.NotificationAlert) error {
	subject := fmt.Sprintf("[%s] %s", alert.Severity, alert.Summary)
	body := webhook.FormatAlertMessage(alert)

	message := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", e.from, e.to, subject, body)

	addr := fmt.Sprintf("%s:%d", e.smtpHost, e.smtpPort)

	if e.useTLS {
		return e.sendWithTLS(addr, []byte(message))
	}

	auth := smtp.PlainAuth("", e.username, e.password, e.smtpHost)
	return smtp.SendMail(addr, auth, e.from, []string{e.to}, []byte(message))
}

func (e *EmailNotifier) sendWithTLS(addr string, message []byte) error {
	tlsConfig := &tls.Config{
		ServerName: e.smtpHost,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, e.smtpHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(smtp.PlainAuth("", e.username, e.password, e.smtpHost)); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if err := client.Mail(e.from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	if err := client.Rcpt(e.to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err := w.Write(message); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return client.Quit()
}
