package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/api/response"
	"github.com/infrasense/backend/internal/db"
	"github.com/infrasense/backend/internal/models"
)

type DeviceGroupHandler struct {
	repo *db.DeviceGroupRepository
}

func NewDeviceGroupHandler(repo *db.DeviceGroupRepository) *DeviceGroupHandler {
	return &DeviceGroupHandler{repo: repo}
}

// Create handles POST /api/v1/device-groups
func (h *DeviceGroupHandler) Create(c *gin.Context) {
	var req models.DeviceGroupCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), "INVALID_REQUEST")
		return
	}

	group, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		response.InternalError(c, "Failed to create device group")
		return
	}

	response.Created(c, group)
}

// List handles GET /api/v1/device-groups
func (h *DeviceGroupHandler) List(c *gin.Context) {
	groups, err := h.repo.List(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to list device groups")
		return
	}

	response.Success(c, groups)
}

// GetByID handles GET /api/v1/device-groups/:id
func (h *DeviceGroupHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid group ID", "INVALID_ID")
		return
	}

	group, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "device group not found" {
			response.NotFound(c, "Device group not found")
			return
		}
		response.InternalError(c, "Failed to get device group")
		return
	}

	response.Success(c, group)
}

// Update handles PUT /api/v1/device-groups/:id
func (h *DeviceGroupHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid group ID", "INVALID_ID")
		return
	}

	var req models.DeviceGroupUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), "INVALID_REQUEST")
		return
	}

	group, err := h.repo.Update(c.Request.Context(), id, req)
	if err != nil {
		if err.Error() == "device group not found" {
			response.NotFound(c, "Device group not found")
			return
		}
		response.InternalError(c, "Failed to update device group")
		return
	}

	response.Success(c, group)
}

// Delete handles DELETE /api/v1/device-groups/:id
func (h *DeviceGroupHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid group ID", "INVALID_ID")
		return
	}

	err = h.repo.Delete(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "device group not found" {
			response.NotFound(c, "Device group not found")
			return
		}
		response.InternalError(c, "Failed to delete device group")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device group deleted successfully"})
}

// AddDevice handles POST /api/v1/device-groups/:id/devices
func (h *DeviceGroupHandler) AddDevice(c *gin.Context) {
	groupIDStr := c.Param("id")
	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid group ID", "INVALID_ID")
		return
	}

	var req models.AddDeviceToGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), "INVALID_REQUEST")
		return
	}

	err = h.repo.AddDevice(c.Request.Context(), groupID, req.DeviceID)
	if err != nil {
		response.InternalError(c, "Failed to add device to group")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device added to group successfully"})
}

// RemoveDevice handles DELETE /api/v1/device-groups/:id/devices/:deviceId
func (h *DeviceGroupHandler) RemoveDevice(c *gin.Context) {
	groupIDStr := c.Param("id")
	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid group ID", "INVALID_ID")
		return
	}

	deviceIDStr := c.Param("deviceId")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	err = h.repo.RemoveDevice(c.Request.Context(), groupID, deviceID)
	if err != nil {
		if err.Error() == "device not in group" {
			response.NotFound(c, "Device not in group")
			return
		}
		response.InternalError(c, "Failed to remove device from group")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device removed from group successfully"})
}
