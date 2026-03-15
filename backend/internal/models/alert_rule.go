package models

import (
	"time"

	"github.com/google/uuid"
)

type AlertRule struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	MetricName    string     `json:"metric_name"`
	Operator      string     `json:"operator"` // gt, lt, eq, ne
	Threshold     float64    `json:"threshold"`
	Severity      string     `json:"severity"` // critical, warning, info
	DeviceID      *uuid.UUID `json:"device_id,omitempty"`
	DeviceGroupID *uuid.UUID `json:"device_group_id,omitempty"`
	Enabled       bool       `json:"enabled"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type AlertRuleCreateRequest struct {
	Name          string     `json:"name" binding:"required"`
	MetricName    string     `json:"metric_name" binding:"required"`
	Operator      string     `json:"operator" binding:"required"`
	Threshold     float64    `json:"threshold" binding:"required"`
	Severity      string     `json:"severity" binding:"required"`
	DeviceID      *uuid.UUID `json:"device_id"`
	DeviceGroupID *uuid.UUID `json:"device_group_id"`
	Enabled       *bool      `json:"enabled"`
}

type AlertRuleUpdateRequest struct {
	Name          *string    `json:"name"`
	MetricName    *string    `json:"metric_name"`
	Operator      *string    `json:"operator"`
	Threshold     *float64   `json:"threshold"`
	Severity      *string    `json:"severity"`
	DeviceID      *uuid.UUID `json:"device_id"`
	DeviceGroupID *uuid.UUID `json:"device_group_id"`
	Enabled       *bool      `json:"enabled"`
}
