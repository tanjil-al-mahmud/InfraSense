package models

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	UserID         *uuid.UUID             `json:"user_id,omitempty" db:"user_id"`
	ActionType     string                 `json:"action_type" db:"action_type"`
	TargetResource string                 `json:"target_resource" db:"target_resource"`
	TargetID       *uuid.UUID             `json:"target_id,omitempty" db:"target_id"`
	SourceIP       string                 `json:"source_ip" db:"source_ip"`
	Details        map[string]interface{} `json:"details,omitempty" db:"details"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

const (
	ActionDeviceCreate           = "device_create"
	ActionDeviceUpdate           = "device_update"
	ActionDeviceDelete           = "device_delete"
	ActionAlertRuleCreate        = "alert_rule_create"
	ActionAlertRuleUpdate        = "alert_rule_update"
	ActionAlertRuleDelete        = "alert_rule_delete"
	ActionMaintenanceWindowCreate = "maintenance_window_create"
	ActionMaintenanceWindowDelete = "maintenance_window_delete"
	ActionUserLogin              = "user_login"
	ActionUserLoginFailed        = "user_login_failed"
	ActionUserCreate             = "user_create"
	ActionUserUpdate             = "user_update"
	ActionUserDelete             = "user_delete"
	ActionAlertAcknowledge       = "alert_acknowledge"
)
