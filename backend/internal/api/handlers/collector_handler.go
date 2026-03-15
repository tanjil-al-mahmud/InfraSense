package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/api/response"
	"github.com/infrasense/backend/internal/db"
	"github.com/infrasense/backend/internal/models"
)

type CollectorHandler struct {
	repo *db.CollectorStatusRepository
}

func NewCollectorHandler(repo *db.CollectorStatusRepository) *CollectorHandler {
	return &CollectorHandler{repo: repo}
}

// List handles GET /api/v1/collectors
func (h *CollectorHandler) List(c *gin.Context) {
	filter := models.CollectorListFilter{
		Page:     1,
		PageSize: 20,
	}

	if page := c.Query("page"); page != "" {
		var p int
		if _, err := fmt.Sscanf(page, "%d", &p); err == nil && p > 0 {
			filter.Page = p
		}
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		var ps int
		if _, err := fmt.Sscanf(pageSize, "%d", &ps); err == nil && ps > 0 {
			filter.PageSize = ps
		}
	}

	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}

	if collectorType := c.Query("collector_type"); collectorType != "" {
		filter.CollectorType = &collectorType
	}

	collectors, total, err := h.repo.List(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, "Failed to list collectors")
		return
	}

	response.Paginated(c, collectors, models.PaginationMeta{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Total:    total,
	})
}

// GetByID handles GET /api/v1/collectors/:id
func (h *CollectorHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid collector ID", "INVALID_ID")
		return
	}

	collector, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "collector not found" {
			response.NotFound(c, "Collector not found")
			return
		}
		response.InternalError(c, "Failed to get collector")
		return
	}

	response.Success(c, collector)
}
