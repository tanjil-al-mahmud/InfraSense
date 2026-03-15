package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/api/response"
	"github.com/infrasense/backend/internal/db"
	"github.com/infrasense/backend/internal/models"
	"github.com/infrasense/backend/internal/services"
)

type AlertHandler struct {
	ackRepo      *db.AlertAcknowledgmentRepository
	auditService *services.AuditService
}

func NewAlertHandler(ackRepo *db.AlertAcknowledgmentRepository, auditService *services.AuditService) *AlertHandler {
	return &AlertHandler{
		ackRepo:      ackRepo,
		auditService: auditService,
	}
}

// AlertmanagerAlert represents an alert from Alertmanager API
type AlertmanagerAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
	Status      struct {
		State string `json:"state"`
	} `json:"status"`
	Fingerprint string `json:"fingerprint"`
}

// Alert represents an alert for the frontend
type Alert struct {
	Fingerprint    string            `json:"fingerprint"`
	DeviceName     string            `json:"device_name"`
	AlertName      string            `json:"alert_name"`
	Severity       string            `json:"severity"`
	FiredAt        time.Time         `json:"fired_at"`
	ResolvedAt     *time.Time        `json:"resolved_at,omitempty"`
	CurrentValue   string            `json:"current_value"`
	Description    string            `json:"description"`
	Labels         map[string]string `json:"labels"`
	Acknowledged   bool              `json:"acknowledged"`
	AcknowledgedAt *time.Time        `json:"acknowledged_at,omitempty"`
}

// ListActive handles GET /api/v1/alerts
func (h *AlertHandler) ListActive(c *gin.Context) {
	severity := c.Query("severity")
	device := c.Query("device")

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("http://alertmanager:9093/api/v2/alerts?active=true")
	if err != nil {
		log.Printf("Failed to query Alertmanager: %v", err)
		response.Error(c, http.StatusInternalServerError, "Failed to query alerts", "ALERTMANAGER_ERROR")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Alertmanager returned non-200 status: %d", resp.StatusCode)
		response.Error(c, http.StatusInternalServerError, "Failed to query alerts", "ALERTMANAGER_ERROR")
		return
	}

	var amAlerts []AlertmanagerAlert
	if err := json.NewDecoder(resp.Body).Decode(&amAlerts); err != nil {
		log.Printf("Failed to decode Alertmanager response: %v", err)
		response.Error(c, http.StatusInternalServerError, "Failed to parse alerts", "PARSE_ERROR")
		return
	}

	acks, err := h.ackRepo.List(c.Request.Context())
	if err != nil {
		log.Printf("Failed to get acknowledgments: %v", err)
	}

	ackMap := make(map[string]*time.Time)
	for _, ack := range acks {
		ackTime := ack.AcknowledgedAt
		ackMap[ack.AlertFingerprint] = &ackTime
	}

	var alerts []Alert
	for _, amAlert := range amAlerts {
		if amAlert.Status.State != "active" {
			continue
		}

		alert := Alert{
			Fingerprint:  amAlert.Fingerprint,
			DeviceName:   amAlert.Labels["device"],
			AlertName:    amAlert.Labels["alertname"],
			Severity:     amAlert.Labels["severity"],
			FiredAt:      amAlert.StartsAt,
			CurrentValue: amAlert.Annotations["value"],
			Description:  amAlert.Annotations["description"],
			Labels:       amAlert.Labels,
			Acknowledged: false,
		}

		if ackTime, ok := ackMap[alert.Fingerprint]; ok {
			alert.Acknowledged = true
			alert.AcknowledgedAt = ackTime
		}

		if severity != "" && alert.Severity != severity {
			continue
		}
		if device != "" && alert.DeviceName != device {
			continue
		}

		alerts = append(alerts, alert)
	}

	response.Success(c, alerts)
}

// ListHistory handles GET /api/v1/alerts/history
func (h *AlertHandler) ListHistory(c *gin.Context) {
	severity := c.Query("severity")
	device := c.Query("device")

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("http://alertmanager:9093/api/v2/alerts")
	if err != nil {
		log.Printf("Failed to query Alertmanager: %v", err)
		response.Error(c, http.StatusInternalServerError, "Failed to query alerts", "ALERTMANAGER_ERROR")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Alertmanager returned non-200 status: %d", resp.StatusCode)
		response.Error(c, http.StatusInternalServerError, "Failed to query alerts", "ALERTMANAGER_ERROR")
		return
	}

	var amAlerts []AlertmanagerAlert
	if err := json.NewDecoder(resp.Body).Decode(&amAlerts); err != nil {
		log.Printf("Failed to decode Alertmanager response: %v", err)
		response.Error(c, http.StatusInternalServerError, "Failed to parse alerts", "PARSE_ERROR")
		return
	}

	acks, err := h.ackRepo.List(c.Request.Context())
	if err != nil {
		log.Printf("Failed to get acknowledgments: %v", err)
	}

	ackMap := make(map[string]*time.Time)
	for _, ack := range acks {
		ackTime := ack.AcknowledgedAt
		ackMap[ack.AlertFingerprint] = &ackTime
	}

	var alerts []Alert
	for _, amAlert := range amAlerts {
		alert := Alert{
			Fingerprint:  amAlert.Fingerprint,
			DeviceName:   amAlert.Labels["device"],
			AlertName:    amAlert.Labels["alertname"],
			Severity:     amAlert.Labels["severity"],
			FiredAt:      amAlert.StartsAt,
			CurrentValue: amAlert.Annotations["value"],
			Description:  amAlert.Annotations["description"],
			Labels:       amAlert.Labels,
			Acknowledged: false,
		}

		if amAlert.Status.State == "resolved" && !amAlert.EndsAt.IsZero() {
			alert.ResolvedAt = &amAlert.EndsAt
		}

		if ackTime, ok := ackMap[alert.Fingerprint]; ok {
			alert.Acknowledged = true
			alert.AcknowledgedAt = ackTime
		}

		if severity != "" && alert.Severity != severity {
			continue
		}
		if device != "" && alert.DeviceName != device {
			continue
		}

		alerts = append(alerts, alert)
	}

	response.Success(c, alerts)
}

// Acknowledge handles POST /api/v1/alerts/:id/acknowledge
func (h *AlertHandler) Acknowledge(c *gin.Context) {
	fingerprint := c.Param("id")
	if fingerprint == "" {
		response.BadRequest(c, "Alert fingerprint is required", "INVALID_REQUEST")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	ack, err := h.ackRepo.Create(c.Request.Context(), userID.(uuid.UUID), fingerprint)
	if err != nil {
		log.Printf("Failed to create acknowledgment: %v", err)
		response.InternalError(c, "Failed to acknowledge alert")
		return
	}

	h.auditService.LogAction(c.Request.Context(), userID.(uuid.UUID), models.ActionAlertAcknowledge, c.ClientIP(), map[string]interface{}{
		"alert_fingerprint": fingerprint,
		"acknowledged_at":   ack.AcknowledgedAt,
	})

	response.Success(c, ack)
}
