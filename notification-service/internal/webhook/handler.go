package webhook

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// AlertmanagerWebhook represents the webhook payload from Alertmanager
type AlertmanagerWebhook struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []Alert           `json:"alerts"`
}

// Alert represents a single alert from Alertmanager
type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

// NotificationAlert represents a processed alert ready for notification
type NotificationAlert struct {
	Severity     string
	DeviceName   string
	DeviceID     string
	MetricName   string
	CurrentValue string
	Timestamp    time.Time
	Summary      string
	Description  string
	Status       string
}

// Notifier interface for notification channels
type Notifier interface {
	Send(alert NotificationAlert) error
	Name() string
}

// Handler handles Alertmanager webhook requests
type Handler struct {
	notifiers []Notifier
}

// NewHandler creates a new webhook handler
func NewHandler(notifiers []Notifier) *Handler {
	return &Handler{
		notifiers: notifiers,
	}
}

// HandleWebhook processes incoming Alertmanager webhooks
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var webhook AlertmanagerWebhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		slog.Error("error decoding webhook payload", "event", "webhook_decode_error", "error", err.Error())
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	slog.Info("received webhook", "event", "webhook_received", "alert_count", len(webhook.Alerts), "status", webhook.Status)

	// Process each alert
	for _, alert := range webhook.Alerts {
		notifAlert := h.parseAlert(alert)

		slog.Info("processing alert",
			"event", "alert_processing",
			"summary", notifAlert.Summary,
			"severity", notifAlert.Severity,
			"device_id", notifAlert.DeviceID,
			"status", notifAlert.Status)

		// Send to all configured notifiers in parallel
		h.sendToNotifiers(notifAlert)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// parseAlert converts Alertmanager alert to NotificationAlert
func (h *Handler) parseAlert(alert Alert) NotificationAlert {
	return NotificationAlert{
		Severity:     alert.Labels["severity"],
		DeviceName:   alert.Labels["hostname"],
		DeviceID:     alert.Labels["device_id"],
		MetricName:   alert.Annotations["metric_name"],
		CurrentValue: alert.Annotations["current_value"],
		Timestamp:    alert.StartsAt,
		Summary:      alert.Annotations["summary"],
		Description:  alert.Annotations["description"],
		Status:       alert.Status,
	}
}

// sendToNotifiers sends alert to all configured notifiers in parallel
func (h *Handler) sendToNotifiers(alert NotificationAlert) {
	for _, notifier := range h.notifiers {
		go func(n Notifier) {
			if err := n.Send(alert); err != nil {
				slog.Error("notification delivery failed",
					"event", "notification_delivery",
					"channel", n.Name(),
					"result", "error",
					"error", err.Error(),
					"device_id", alert.DeviceID,
					"severity", alert.Severity)
			} else {
				slog.Info("notification delivered successfully",
					"event", "notification_delivery",
					"channel", n.Name(),
					"result", "success",
					"device_id", alert.DeviceID,
					"severity", alert.Severity)
			}
		}(notifier)
	}
}

// FormatAlertMessage formats alert for human-readable notification
func FormatAlertMessage(alert NotificationAlert) string {
	status := "🔴 FIRING"
	if alert.Status == "resolved" {
		status = "✅ RESOLVED"
	}

	severityEmoji := ""
	switch alert.Severity {
	case "critical", "emergency":
		severityEmoji = "🚨"
	case "warning":
		severityEmoji = "⚠️"
	case "info":
		severityEmoji = "ℹ️"
	}

	msg := fmt.Sprintf("%s %s %s\n\n", status, severityEmoji, alert.Summary)

	if alert.Description != "" {
		msg += fmt.Sprintf("Description: %s\n", alert.Description)
	}

	if alert.DeviceID != "" {
		msg += fmt.Sprintf("Device ID: %s\n", alert.DeviceID)
	}

	if alert.DeviceName != "" {
		msg += fmt.Sprintf("Device: %s\n", alert.DeviceName)
	}

	if alert.MetricName != "" {
		msg += fmt.Sprintf("Metric: %s\n", alert.MetricName)
	}

	if alert.CurrentValue != "" {
		msg += fmt.Sprintf("Current Value: %s\n", alert.CurrentValue)
	}

	msg += fmt.Sprintf("Severity: %s\n", alert.Severity)
	msg += fmt.Sprintf("Time: %s\n", alert.Timestamp.Format("2006-01-02 15:04:05 MST"))

	return msg
}
