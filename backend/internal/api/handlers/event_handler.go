package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/api/response"
	"github.com/infrasense/backend/internal/db"
)

// EventHandler serves hardware event log endpoints backed by the hardware_events table.
type EventHandler struct {
	eventRepo *db.HardwareEventRepository
}

func NewEventHandler(eventRepo *db.HardwareEventRepository) *EventHandler {
	return &EventHandler{eventRepo: eventRepo}
}

// eventRow is the JSON shape returned to the frontend.
type eventRow struct {
	ID             string `json:"id"`
	DeviceID       string `json:"device_id"`
	Hostname       string `json:"hostname"`
	EventTime      string `json:"event_time"`
	SourceProtocol string `json:"source_protocol"`
	Component      string `json:"component"`
	EventType      string `json:"event_type"`
	Severity       string `json:"severity"`
	Message        string `json:"message"`
}

// parseEventFilter builds an HardwareEventListFilter from query params.
func parseEventFilter(c *gin.Context) db.HardwareEventListFilter {
	f := db.HardwareEventListFilter{Page: 1, PageSize: 50}

	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &f.Page) //nolint
	}
	if ps := c.Query("limit"); ps != "" {
		fmt.Sscanf(ps, "%d", &f.PageSize) //nolint
	}
	if sev := c.Query("severity"); sev != "" && sev != "all" {
		f.Severity = &sev
	}
	if q := c.Query("search"); q != "" {
		f.Search = &q
	}
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			f.DateFrom = &t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			f.DateTo = &t
		}
	}
	return f
}

// derefStr safely dereferences a *string, returning "" if nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ListDeviceEvents handles GET /api/v1/devices/:id/events
func (h *EventHandler) ListDeviceEvents(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	f := parseEventFilter(c)
	f.DeviceID = &id

	rawEvents, total, err := h.eventRepo.ListEvents(c.Request.Context(), f)
	if err != nil {
		response.InternalError(c, "Failed to list device events")
		return
	}

	rows := make([]eventRow, 0, len(rawEvents))
	for _, ev := range rawEvents {
		t := ""
		if ev.OccurredAt != nil {
			t = ev.OccurredAt.UTC().Format(time.RFC3339)
		}
		rows = append(rows, eventRow{
			ID:             ev.ID.String(),
			DeviceID:       ev.DeviceID.String(),
			Hostname:       derefStr(ev.Vendor), // Vendor field carries hostname (see repo)
			EventTime:      t,
			SourceProtocol: ev.SourceProtocol,
			Component:      ev.Component,
			EventType:      ev.EventType,
			Severity:       ev.Severity,
			Message:        ev.Message,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"events": rows,
		"total":  total,
		"page":   f.Page,
		"limit":  f.PageSize,
	})
}

// ListAllEvents handles GET /api/v1/events
func (h *EventHandler) ListAllEvents(c *gin.Context) {
	f := parseEventFilter(c)

	// Optional device_id filter
	if did := c.Query("device_id"); did != "" {
		id, err := uuid.Parse(did)
		if err != nil {
			response.BadRequest(c, "Invalid device_id", "INVALID_ID")
			return
		}
		f.DeviceID = &id
	}

	rawEvents, total, err := h.eventRepo.ListEvents(c.Request.Context(), f)
	if err != nil {
		response.InternalError(c, "Failed to list events")
		return
	}

	rows := make([]eventRow, 0, len(rawEvents))
	for _, ev := range rawEvents {
		t := ""
		if ev.OccurredAt != nil {
			t = ev.OccurredAt.UTC().Format(time.RFC3339)
		}
		rows = append(rows, eventRow{
			ID:             ev.ID.String(),
			DeviceID:       ev.DeviceID.String(),
			Hostname:       derefStr(ev.Vendor),
			EventTime:      t,
			SourceProtocol: ev.SourceProtocol,
			Component:      ev.Component,
			EventType:      ev.EventType,
			Severity:       strings.ToLower(ev.Severity),
			Message:        ev.Message,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"events": rows,
		"total":  total,
		"page":   f.Page,
		"limit":  f.PageSize,
	})
}

// GetEventSummary handles GET /api/v1/events/summary
func (h *EventHandler) GetEventSummary(c *gin.Context) {
	summary, err := h.eventRepo.GetEventSummary(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to get event summary")
		return
	}
	response.Success(c, summary)
}

// ClearDeviceEvents handles POST /api/v1/devices/:id/events/clear  (admin only)
func (h *EventHandler) ClearDeviceEvents(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	n, err := h.eventRepo.ClearDeviceEvents(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, "Failed to clear device events")
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": n, "message": "Events cleared successfully"})
}
