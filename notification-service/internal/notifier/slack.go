package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/infrasense/notification-service/internal/webhook"
)

type SlackNotifier struct {
	webhookURL  string
	client      *http.Client
	rateLimiter *RateLimiter
}

func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		// Rate limit: 1 message per second
		rateLimiter: NewRateLimiter(1, 1*time.Second),
	}
}

func (s *SlackNotifier) Name() string {
	return "Slack"
}

func (s *SlackNotifier) Send(alert webhook.NotificationAlert) error {
	// Wait for rate limiter
	for !s.rateLimiter.Allow() {
		time.Sleep(100 * time.Millisecond)
	}

	return s.sendWithRetry(alert, 3)
}

func (s *SlackNotifier) sendWithRetry(alert webhook.NotificationAlert, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			time.Sleep(backoff)
		}

		err := s.sendMessage(alert)
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func (s *SlackNotifier) sendMessage(alert webhook.NotificationAlert) error {
	color := s.getSeverityColor(alert.Severity)
	
	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":      color,
				"title":      alert.Summary,
				"text":       alert.Description,
				"footer":     "InfraSense Alert",
				"footer_icon": "https://platform.slack-edge.com/img/default_application_icon.png",
				"ts":         alert.Timestamp.Unix(),
				"fields": []map[string]interface{}{
					{
						"title": "Severity",
						"value": alert.Severity,
						"short": true,
					},
					{
						"title": "Status",
						"value": alert.Status,
						"short": true,
					},
					{
						"title": "Device ID",
						"value": alert.DeviceID,
						"short": true,
					},
					{
						"title": "Metric",
						"value": alert.MetricName,
						"short": true,
					},
					{
						"title": "Current Value",
						"value": alert.CurrentValue,
						"short": true,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := s.client.Post(s.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (s *SlackNotifier) getSeverityColor(severity string) string {
	switch severity {
	case "critical", "emergency":
		return "danger"  // Red
	case "warning":
		return "warning" // Yellow
	case "info":
		return "good"    // Green
	default:
		return "#808080" // Gray
	}
}
