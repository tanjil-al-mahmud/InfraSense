package handlers

import (
	"log"
	"math"
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

type AlertRuleHandler struct {
	repo         *db.AlertRuleRepository
	auditService *services.AuditService
}

func NewAlertRuleHandler(repo *db.AlertRuleRepository, auditService *services.AuditService) *AlertRuleHandler {
	return &AlertRuleHandler{
		repo:         repo,
		auditService: auditService,
	}
}

// Create handles POST /api/v1/alert-rules
func (h *AlertRuleHandler) Create(c *gin.Context) {
	var req models.AlertRuleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := validation.FormatBindingErrors(err)
		log.Printf("validation failed: create alert rule - %v", err)
		response.BadRequest(c, msg, "INVALID_REQUEST")
		return
	}

	validOperators := map[string]bool{"gt": true, "lt": true, "eq": true, "ne": true, "gte": true, "lte": true}
	if !validOperators[req.Operator] {
		log.Printf("validation failed: operator - invalid value '%s'", req.Operator)
		response.BadRequest(c, "Invalid operator. Must be one of: gt, lt, eq, ne, gte, lte", "INVALID_OPERATOR")
		return
	}

	validSeverities := map[string]bool{"critical": true, "warning": true, "info": true}
	if !validSeverities[req.Severity] {
		log.Printf("validation failed: severity - invalid value '%s'", req.Severity)
		response.BadRequest(c, "Invalid severity. Must be one of: critical, warning, info", "INVALID_SEVERITY")
		return
	}

	if math.IsNaN(req.Threshold) || math.IsInf(req.Threshold, 0) {
		log.Printf("validation failed: threshold - not a finite number")
		response.BadRequest(c, "Field 'threshold' must be a valid finite number", "INVALID_THRESHOLD")
		return
	}

	rule, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		log.Printf("Failed to create alert rule: %v", err)
		response.InternalError(c, "Failed to create alert rule")
		return
	}

	userID, _ := c.Get("user_id")
	h.auditService.LogAlertRuleCreate(c.Request.Context(), userID.(uuid.UUID), rule.ID, c.ClientIP(), map[string]interface{}{
		"name":        rule.Name,
		"metric_name": rule.MetricName,
		"operator":    rule.Operator,
		"threshold":   rule.Threshold,
		"severity":    rule.Severity,
	})

	go h.reloadPrometheus()

	response.Created(c, rule)
}

// List handles GET /api/v1/alert-rules
func (h *AlertRuleHandler) List(c *gin.Context) {
	rules, err := h.repo.List(c.Request.Context())
	if err != nil {
		log.Printf("Failed to list alert rules: %v", err)
		response.InternalError(c, "Failed to list alert rules")
		return
	}

	response.Success(c, rules)
}

// GetByID handles GET /api/v1/alert-rules/:id
func (h *AlertRuleHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid alert rule ID", "INVALID_ID")
		return
	}

	rule, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "alert rule not found" {
			response.NotFound(c, "Alert rule not found")
			return
		}
		log.Printf("Failed to get alert rule: %v", err)
		response.InternalError(c, "Failed to get alert rule")
		return
	}

	response.Success(c, rule)
}

// Update handles PUT /api/v1/alert-rules/:id
func (h *AlertRuleHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid alert rule ID", "INVALID_ID")
		return
	}

	var req models.AlertRuleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := validation.FormatBindingErrors(err)
		log.Printf("validation failed: update alert rule - %v", err)
		response.BadRequest(c, msg, "INVALID_REQUEST")
		return
	}

	if req.Operator != nil {
		validOperators := map[string]bool{"gt": true, "lt": true, "eq": true, "ne": true, "gte": true, "lte": true}
		if !validOperators[*req.Operator] {
			log.Printf("validation failed: operator - invalid value '%s'", *req.Operator)
			response.BadRequest(c, "Invalid operator. Must be one of: gt, lt, eq, ne, gte, lte", "INVALID_OPERATOR")
			return
		}
	}

	if req.Severity != nil {
		validSeverities := map[string]bool{"critical": true, "warning": true, "info": true}
		if !validSeverities[*req.Severity] {
			log.Printf("validation failed: severity - invalid value '%s'", *req.Severity)
			response.BadRequest(c, "Invalid severity. Must be one of: critical, warning, info", "INVALID_SEVERITY")
			return
		}
	}

	if req.Threshold != nil && (math.IsNaN(*req.Threshold) || math.IsInf(*req.Threshold, 0)) {
		log.Printf("validation failed: threshold - not a finite number")
		response.BadRequest(c, "Field 'threshold' must be a valid finite number", "INVALID_THRESHOLD")
		return
	}

	rule, err := h.repo.Update(c.Request.Context(), id, req)
	if err != nil {
		if err.Error() == "alert rule not found" {
			response.NotFound(c, "Alert rule not found")
			return
		}
		log.Printf("Failed to update alert rule: %v", err)
		response.InternalError(c, "Failed to update alert rule")
		return
	}

	userID, _ := c.Get("user_id")
	details := map[string]interface{}{"rule_id": rule.ID.String()}
	if req.Name != nil {
		details["name"] = *req.Name
	}
	if req.MetricName != nil {
		details["metric_name"] = *req.MetricName
	}
	if req.Operator != nil {
		details["operator"] = *req.Operator
	}
	if req.Threshold != nil {
		details["threshold"] = *req.Threshold
	}
	if req.Severity != nil {
		details["severity"] = *req.Severity
	}
	h.auditService.LogAlertRuleUpdate(c.Request.Context(), userID.(uuid.UUID), rule.ID, c.ClientIP(), details)

	go h.reloadPrometheus()

	response.Success(c, rule)
}

// Delete handles DELETE /api/v1/alert-rules/:id
func (h *AlertRuleHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid alert rule ID", "INVALID_ID")
		return
	}

	err = h.repo.Delete(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "alert rule not found" {
			response.NotFound(c, "Alert rule not found")
			return
		}
		log.Printf("Failed to delete alert rule: %v", err)
		response.InternalError(c, "Failed to delete alert rule")
		return
	}

	userID, _ := c.Get("user_id")
	h.auditService.LogAlertRuleDelete(c.Request.Context(), userID.(uuid.UUID), id, c.ClientIP(), nil)

	go h.reloadPrometheus()

	c.JSON(http.StatusOK, gin.H{"message": "Alert rule deleted successfully"})
}

// reloadPrometheus triggers Prometheus to reload its configuration
func (h *AlertRuleHandler) reloadPrometheus() {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", "http://prometheus:9090/-/reload", nil)
	if err != nil {
		log.Printf("Failed to create Prometheus reload request: %v", err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to reload Prometheus: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Prometheus reload returned non-200 status: %d", resp.StatusCode)
		return
	}

	log.Println("Prometheus configuration reloaded successfully")
}
