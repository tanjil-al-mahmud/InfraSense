package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/api/response"
	"github.com/infrasense/backend/internal/api/validation"
	"github.com/infrasense/backend/internal/db"
	"github.com/infrasense/backend/internal/models"
	"github.com/infrasense/backend/internal/services"
)

type MaintenanceWindowHandler struct {
	repo         *db.MaintenanceWindowRepository
	auditService *services.AuditService
}

func NewMaintenanceWindowHandler(repo *db.MaintenanceWindowRepository, auditService *services.AuditService) *MaintenanceWindowHandler {
	return &MaintenanceWindowHandler{
		repo:         repo,
		auditService: auditService,
	}
}

// AlertmanagerSilence represents the Alertmanager silence payload
type AlertmanagerSilence struct {
	Matchers  []AlertmanagerMatcher `json:"matchers"`
	StartsAt  time.Time             `json:"startsAt"`
	EndsAt    time.Time             `json:"endsAt"`
	CreatedBy string                `json:"createdBy"`
	Comment   string                `json:"comment"`
}

type AlertmanagerMatcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsRegex bool   `json:"isRegex"`
}

type AlertmanagerSilenceResponse struct {
	SilenceID string `json:"silenceID"`
}

// Create handles POST /api/v1/maintenance-windows
func (h *MaintenanceWindowHandler) Create(c *gin.Context) {
	var req models.MaintenanceWindowCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := validation.FormatBindingErrors(err)
		log.Printf("validation failed: create maintenance window - %v", err)
		response.BadRequest(c, msg, "INVALID_REQUEST")
		return
	}

	if !req.StartTime.Before(req.EndTime) {
		log.Printf("validation failed: end_time must be after start_time (start=%s, end=%s)", req.StartTime, req.EndTime)
		response.BadRequest(c, fmt.Sprintf("end_time (%s) must be after start_time (%s)", req.EndTime.Format(time.RFC3339), req.StartTime.Format(time.RFC3339)), "INVALID_TIME_RANGE")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	window, err := h.repo.Create(c.Request.Context(), req, userID.(uuid.UUID))
	if err != nil {
		log.Printf("Failed to create maintenance window: %v", err)
		response.InternalError(c, "Failed to create maintenance window")
		return
	}

	silenceID, err := h.createAlertmanagerSilence(window)
	if err != nil {
		log.Printf("Failed to create Alertmanager silence: %v", err)
	} else {
		if err := h.repo.UpdateSilenceID(c.Request.Context(), window.ID, silenceID); err != nil {
			log.Printf("Failed to update silence_id: %v", err)
		}
		window.SilenceID = &silenceID
	}

	h.auditService.LogMaintenanceWindowCreate(c.Request.Context(), userID.(uuid.UUID), window.ID, c.ClientIP(), map[string]interface{}{
		"device_id":  window.DeviceID.String(),
		"start_time": window.StartTime,
		"end_time":   window.EndTime,
		"reason":     window.Reason,
	})

	response.Created(c, window)
}

// List handles GET /api/v1/maintenance-windows
func (h *MaintenanceWindowHandler) List(c *gin.Context) {
	activeOnly := c.Query("active") == "true"

	windows, err := h.repo.List(c.Request.Context(), activeOnly)
	if err != nil {
		log.Printf("Failed to list maintenance windows: %v", err)
		response.InternalError(c, "Failed to list maintenance windows")
		return
	}

	response.Success(c, windows)
}

// Delete handles DELETE /api/v1/maintenance-windows/:id
func (h *MaintenanceWindowHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid maintenance window ID", "INVALID_ID")
		return
	}

	window, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "maintenance window not found" {
			response.NotFound(c, "Maintenance window not found")
			return
		}
		log.Printf("Failed to get maintenance window: %v", err)
		response.InternalError(c, "Failed to get maintenance window")
		return
	}

	if window.SilenceID != nil && *window.SilenceID != "" {
		if err := h.deleteAlertmanagerSilence(*window.SilenceID); err != nil {
			log.Printf("Failed to delete Alertmanager silence: %v", err)
		}
	}

	err = h.repo.Delete(c.Request.Context(), id)
	if err != nil {
		log.Printf("Failed to delete maintenance window: %v", err)
		response.InternalError(c, "Failed to delete maintenance window")
		return
	}

	userID, _ := c.Get("user_id")
	h.auditService.LogMaintenanceWindowDelete(c.Request.Context(), userID.(uuid.UUID), id, c.ClientIP(), map[string]interface{}{
		"device_id": window.DeviceID.String(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "Maintenance window deleted successfully"})
}

// createAlertmanagerSilence creates a silence in Alertmanager
func (h *MaintenanceWindowHandler) createAlertmanagerSilence(window *models.MaintenanceWindow) (string, error) {
	silence := AlertmanagerSilence{
		Matchers: []AlertmanagerMatcher{
			{
				Name:    "device_id",
				Value:   window.DeviceID.String(),
				IsRegex: false,
			},
		},
		StartsAt:  window.StartTime,
		EndsAt:    window.EndTime,
		CreatedBy: window.CreatedBy.String(),
		Comment:   window.Reason,
	}

	payload, err := json.Marshal(silence)
	if err != nil {
		return "", fmt.Errorf("failed to marshal silence payload: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("POST", "http://alertmanager:9093/api/v2/silences", bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("alertmanager returned status %d: %s", resp.StatusCode, string(body))
	}

	var silenceResp AlertmanagerSilenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&silenceResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	log.Printf("Created Alertmanager silence: %s", silenceResp.SilenceID)
	return silenceResp.SilenceID, nil
}

// deleteAlertmanagerSilence deletes a silence from Alertmanager
func (h *MaintenanceWindowHandler) deleteAlertmanagerSilence(silenceID string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	url := fmt.Sprintf("http://alertmanager:9093/api/v2/silence/%s", silenceID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("alertmanager returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("Deleted Alertmanager silence: %s", silenceID)
	return nil
}
