package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/db"
	"github.com/infrasense/backend/internal/models"
	"github.com/infrasense/backend/internal/services"
)

// StreamHandler provides Server-Sent Events for real-time telemetry.
type StreamHandler struct {
	deviceRepo        *db.DeviceRepository
	credRepo          *db.DeviceCredentialRepository
	credentialService *services.CredentialService
	redfishService    *services.RedfishService
}

func NewStreamHandler(
	deviceRepo *db.DeviceRepository,
	credRepo *db.DeviceCredentialRepository,
	credSvc *services.CredentialService,
) *StreamHandler {
	return &StreamHandler{
		deviceRepo:        deviceRepo,
		credRepo:          credRepo,
		credentialService: credSvc,
		redfishService:    services.NewRedfishService(),
	}
}

// StreamTelemetry handles GET /api/v1/devices/:id/stream
// Streams real-time sensor telemetry via Server-Sent Events every 5 seconds.
func (h *StreamHandler) StreamTelemetry(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	device, err := h.deviceRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	if device.BMCIPAddress == nil || *device.BMCIPAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device has no BMC IP configured"})
		return
	}

	cred, err := h.resolveCredential(c.Request.Context(), id, device.DeviceType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No credentials configured"})
		return
	}

	password := ""
	if len(cred.PasswordEncrypted) > 0 {
		decrypted, err := h.credentialService.Decrypt(cred.PasswordEncrypted)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt credentials"})
			return
		}
		password = decrypted
	}

	// Determine polling interval (default 5s for telemetry)
	interval := 5 * time.Second
	if cred.PollingInterval > 0 && cred.PollingInterval < 60 {
		interval = time.Duration(cred.PollingInterval) * time.Second
	}

	// SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	clientGone := c.Request.Context().Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Heartbeat every 25s to keep the connection alive through proxies
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	// Send initial event immediately
	h.sendTelemetryEvent(c, device, cred, password)

	for {
		select {
		case <-clientGone:
			return
		case <-heartbeat.C:
			// Send SSE comment as keepalive (ignored by clients)
			fmt.Fprintf(c.Writer, ": keepalive\n\n")
			c.Writer.Flush()
		case <-ticker.C:
			h.sendTelemetryEvent(c, device, cred, password)
		}
	}
}

// TelemetryEvent is the payload sent over SSE.
type TelemetryEvent struct {
	DeviceID        string                      `json:"device_id"`
	Timestamp       int64                       `json:"timestamp"`
	PowerState      *string                     `json:"power_state,omitempty"`
	HealthStatus    *string                     `json:"health_status,omitempty"`
	Temperatures    []models.TemperatureReading `json:"temperatures,omitempty"`
	Fans            []models.FanReading         `json:"fans,omitempty"`
	PowerSupplies   []models.PowerSupplyInfo    `json:"power_supplies,omitempty"`
	TotalPowerWatts *float64                    `json:"total_power_watts,omitempty"`
	Voltages        []models.VoltageReading     `json:"voltages,omitempty"`
	Error           string                      `json:"error,omitempty"`
}

func (h *StreamHandler) sendTelemetryEvent(c *gin.Context, device *models.Device, cred *models.DeviceCredential, password string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	event := TelemetryEvent{
		DeviceID:  device.ID.String(),
		Timestamp: time.Now().Unix(),
	}

	syncResult, err := h.redfishService.SyncDevice(ctx, *device.BMCIPAddress, cred, password)
	if err != nil {
		event.Error = err.Error()
	} else {
		event.PowerState = syncResult.PowerState
		event.HealthStatus = syncResult.HealthStatus
		event.Temperatures = syncResult.Temperatures
		event.Fans = syncResult.Fans
		event.PowerSupplies = syncResult.PowerSupplies
		event.TotalPowerWatts = syncResult.TotalPowerWatts
		event.Voltages = syncResult.Voltages
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("stream: marshal error for device %s: %v", device.ID, err)
		return
	}

	// Write SSE frame
	fmt.Fprintf(c.Writer, "data: %s\n\n", data)
	c.Writer.Flush()
}

func (h *StreamHandler) resolveCredential(ctx context.Context, deviceID uuid.UUID, deviceType string) (*models.DeviceCredential, error) {
	cred, err := h.credRepo.GetByDeviceIDAndProtocol(ctx, deviceID, deviceType)
	if err == nil {
		return cred, nil
	}
	for _, proto := range []string{"redfish", "ipmi", "snmp_v3", "snmp_v2c"} {
		if strings.HasSuffix(deviceType, proto) || strings.Contains(deviceType, proto) {
			cred, err = h.credRepo.GetByDeviceIDAndProtocol(ctx, deviceID, proto)
			if err == nil {
				return cred, nil
			}
		}
	}
	return nil, fmt.Errorf("no credentials found for device %s", deviceID)
}
