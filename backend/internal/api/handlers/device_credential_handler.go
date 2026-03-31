package handlers

import (
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

type DeviceCredentialHandler struct {
	repo              *db.DeviceCredentialRepository
	credentialService *services.CredentialService
}

func NewDeviceCredentialHandler(repo *db.DeviceCredentialRepository, credentialService *services.CredentialService) *DeviceCredentialHandler {
	return &DeviceCredentialHandler{
		repo:              repo,
		credentialService: credentialService,
	}
}

// Create handles POST /api/v1/devices/:id/credentials
func (h *DeviceCredentialHandler) Create(c *gin.Context) {
	deviceIDStr := c.Param("id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	var req models.DeviceCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := validation.FormatBindingErrors(err)
		log.Printf("validation failed: create credential - %v", err)
		response.BadRequest(c, msg, "INVALID_REQUEST")
		return
	}

	validProtocols := map[string]bool{
		models.ProtocolIPMI:    true,
		models.ProtocolRedfish: true,
		models.ProtocolSNMPv2c: true,
		models.ProtocolSNMPv3:  true,
	}
	if !validProtocols[req.Protocol] {
		log.Printf("validation failed: protocol - invalid value '%s'", req.Protocol)
		response.BadRequest(c, "Invalid protocol. Must be one of: ipmi, redfish, snmp_v2c, snmp_v3", "INVALID_PROTOCOL")
		return
	}

	cred := &models.DeviceCredential{
		ID:              uuid.New(),
		DeviceID:        deviceID,
		Protocol:        req.Protocol,
		Username:        req.Username,
		HTTPScheme:      "https",
		SSLVerify:       false,
		PollingInterval: 60,
		TimeoutSeconds:  30,
		RetryAttempts:   3,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Apply optional connection settings
	if req.Port != nil {
		cred.Port = req.Port
	}
	if req.HTTPScheme != nil && *req.HTTPScheme != "" {
		cred.HTTPScheme = *req.HTTPScheme
	}
	if req.SSLVerify != nil {
		cred.SSLVerify = *req.SSLVerify
	}
	if req.PollingInterval != nil && *req.PollingInterval > 0 {
		cred.PollingInterval = *req.PollingInterval
	}
	if req.TimeoutSeconds != nil && *req.TimeoutSeconds > 0 {
		cred.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.RetryAttempts != nil && *req.RetryAttempts >= 0 {
		cred.RetryAttempts = *req.RetryAttempts
	}

	if req.Password != nil && *req.Password != "" {
		encrypted, err := h.credentialService.Encrypt(*req.Password)
		if err != nil {
			response.InternalError(c, "Failed to encrypt password")
			return
		}
		cred.PasswordEncrypted = encrypted
	}

	if req.CommunityString != nil && *req.CommunityString != "" {
		encrypted, err := h.credentialService.Encrypt(*req.CommunityString)
		if err != nil {
			response.InternalError(c, "Failed to encrypt community string")
			return
		}
		cred.CommunityStringEncrypted = encrypted
	}

	if req.Protocol == models.ProtocolSNMPv3 {
		cred.AuthProtocol = req.AuthProtocol
		cred.PrivProtocol = req.PrivProtocol
	}

	if err := h.repo.Create(c.Request.Context(), cred); err != nil {
		response.InternalError(c, "Failed to create credential")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Credential created successfully"})
}

// Update handles PUT /api/v1/devices/:id/credentials
func (h *DeviceCredentialHandler) Update(c *gin.Context) {
	deviceIDStr := c.Param("id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	var req models.DeviceCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := validation.FormatBindingErrors(err)
		log.Printf("validation failed: update credential - %v", err)
		response.BadRequest(c, msg, "INVALID_REQUEST")
		return
	}

	cred, err := h.repo.GetByDeviceIDAndProtocol(c.Request.Context(), deviceID, req.Protocol)
	if err != nil {
		if err.Error() == "credential not found" {
			response.NotFound(c, "Credential not found")
			return
		}
		response.InternalError(c, "Failed to get credential")
		return
	}

	cred.Username = req.Username

	if req.Password != nil && *req.Password != "" {
		encrypted, err := h.credentialService.Encrypt(*req.Password)
		if err != nil {
			response.InternalError(c, "Failed to encrypt password")
			return
		}
		cred.PasswordEncrypted = encrypted
	}

	if req.CommunityString != nil && *req.CommunityString != "" {
		encrypted, err := h.credentialService.Encrypt(*req.CommunityString)
		if err != nil {
			response.InternalError(c, "Failed to encrypt community string")
			return
		}
		cred.CommunityStringEncrypted = encrypted
	}

	if req.Protocol == models.ProtocolSNMPv3 {
		cred.AuthProtocol = req.AuthProtocol
		cred.PrivProtocol = req.PrivProtocol
	}

	// Apply optional connection settings
	if req.Port != nil {
		cred.Port = req.Port
	}
	if req.HTTPScheme != nil && *req.HTTPScheme != "" {
		cred.HTTPScheme = *req.HTTPScheme
	}
	if req.SSLVerify != nil {
		cred.SSLVerify = *req.SSLVerify
	}
	if req.PollingInterval != nil && *req.PollingInterval > 0 {
		cred.PollingInterval = *req.PollingInterval
	}
	if req.TimeoutSeconds != nil && *req.TimeoutSeconds > 0 {
		cred.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.RetryAttempts != nil && *req.RetryAttempts >= 0 {
		cred.RetryAttempts = *req.RetryAttempts
	}

	if err := h.repo.Update(c.Request.Context(), cred); err != nil {
		response.InternalError(c, "Failed to update credential")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Credential updated successfully"})
}

// Delete handles DELETE /api/v1/devices/:id/credentials
func (h *DeviceCredentialHandler) Delete(c *gin.Context) {
	deviceIDStr := c.Param("id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	protocol := c.Query("protocol")
	if protocol == "" {
		response.BadRequest(c, "Protocol query parameter required", "MISSING_PROTOCOL")
		return
	}

	if err := h.repo.Delete(c.Request.Context(), deviceID, protocol); err != nil {
		if err.Error() == "credential not found" {
			response.NotFound(c, "Credential not found")
			return
		}
		response.InternalError(c, "Failed to delete credential")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Credential deleted successfully"})
}
