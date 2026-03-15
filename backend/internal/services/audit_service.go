package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/db"
	"github.com/infrasense/backend/internal/models"
)

type AuditService struct {
	repo *db.AuditRepository
}

func NewAuditService(repo *db.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

// LogDeviceCreate logs device creation
func (s *AuditService) LogDeviceCreate(ctx context.Context, userID uuid.UUID, deviceID uuid.UUID, sourceIP string, details map[string]interface{}) error {
	log := &models.AuditLog{
		UserID:         &userID,
		ActionType:     models.ActionDeviceCreate,
		TargetResource: "device",
		TargetID:       &deviceID,
		SourceIP:       sourceIP,
		Details:        details,
	}
	return s.repo.Create(ctx, log)
}

// LogDeviceUpdate logs device update
func (s *AuditService) LogDeviceUpdate(ctx context.Context, userID uuid.UUID, deviceID uuid.UUID, sourceIP string, details map[string]interface{}) error {
	log := &models.AuditLog{
		UserID:         &userID,
		ActionType:     models.ActionDeviceUpdate,
		TargetResource: "device",
		TargetID:       &deviceID,
		SourceIP:       sourceIP,
		Details:        details,
	}
	return s.repo.Create(ctx, log)
}

// LogDeviceDelete logs device deletion
func (s *AuditService) LogDeviceDelete(ctx context.Context, userID uuid.UUID, deviceID uuid.UUID, sourceIP string, details map[string]interface{}) error {
	log := &models.AuditLog{
		UserID:         &userID,
		ActionType:     models.ActionDeviceDelete,
		TargetResource: "device",
		TargetID:       &deviceID,
		SourceIP:       sourceIP,
		Details:        details,
	}
	return s.repo.Create(ctx, log)
}

// LogUserLogin logs successful user login
func (s *AuditService) LogUserLogin(ctx context.Context, userID uuid.UUID, username, sourceIP string) error {
	log := &models.AuditLog{
		UserID:         &userID,
		ActionType:     models.ActionUserLogin,
		TargetResource: "user",
		TargetID:       &userID,
		SourceIP:       sourceIP,
		Details: map[string]interface{}{
			"username": username,
		},
	}
	return s.repo.Create(ctx, log)
}

// LogUserLoginFailed logs failed user login
func (s *AuditService) LogUserLoginFailed(ctx context.Context, username, sourceIP string) error {
	log := &models.AuditLog{
		UserID:         nil,
		ActionType:     models.ActionUserLoginFailed,
		TargetResource: "user",
		TargetID:       nil,
		SourceIP:       sourceIP,
		Details: map[string]interface{}{
			"username": username,
		},
	}
	return s.repo.Create(ctx, log)
}

// LogAlertRuleCreate logs alert rule creation
func (s *AuditService) LogAlertRuleCreate(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID, sourceIP string, details map[string]interface{}) error {
	log := &models.AuditLog{
		UserID:         &userID,
		ActionType:     models.ActionAlertRuleCreate,
		TargetResource: "alert_rule",
		TargetID:       &ruleID,
		SourceIP:       sourceIP,
		Details:        details,
	}
	return s.repo.Create(ctx, log)
}

// LogAlertRuleUpdate logs alert rule update
func (s *AuditService) LogAlertRuleUpdate(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID, sourceIP string, details map[string]interface{}) error {
	log := &models.AuditLog{
		UserID:         &userID,
		ActionType:     models.ActionAlertRuleUpdate,
		TargetResource: "alert_rule",
		TargetID:       &ruleID,
		SourceIP:       sourceIP,
		Details:        details,
	}
	return s.repo.Create(ctx, log)
}

// LogAlertRuleDelete logs alert rule deletion
func (s *AuditService) LogAlertRuleDelete(ctx context.Context, userID uuid.UUID, ruleID uuid.UUID, sourceIP string, details map[string]interface{}) error {
	log := &models.AuditLog{
		UserID:         &userID,
		ActionType:     models.ActionAlertRuleDelete,
		TargetResource: "alert_rule",
		TargetID:       &ruleID,
		SourceIP:       sourceIP,
		Details:        details,
	}
	return s.repo.Create(ctx, log)
}

// LogMaintenanceWindowCreate logs maintenance window creation
func (s *AuditService) LogMaintenanceWindowCreate(ctx context.Context, userID uuid.UUID, windowID uuid.UUID, sourceIP string, details map[string]interface{}) error {
	log := &models.AuditLog{
		UserID:         &userID,
		ActionType:     models.ActionMaintenanceWindowCreate,
		TargetResource: "maintenance_window",
		TargetID:       &windowID,
		SourceIP:       sourceIP,
		Details:        details,
	}
	return s.repo.Create(ctx, log)
}

// LogMaintenanceWindowDelete logs maintenance window deletion
func (s *AuditService) LogMaintenanceWindowDelete(ctx context.Context, userID uuid.UUID, windowID uuid.UUID, sourceIP string, details map[string]interface{}) error {
	log := &models.AuditLog{
		UserID:         &userID,
		ActionType:     models.ActionMaintenanceWindowDelete,
		TargetResource: "maintenance_window",
		TargetID:       &windowID,
		SourceIP:       sourceIP,
		Details:        details,
	}
	return s.repo.Create(ctx, log)
}

// LogAction logs a generic action
func (s *AuditService) LogAction(ctx context.Context, userID uuid.UUID, actionType string, sourceIP string, details map[string]interface{}) error {
	log := &models.AuditLog{
		UserID:         &userID,
		ActionType:     actionType,
		TargetResource: "alert",
		SourceIP:       sourceIP,
		Details:        details,
	}
	return s.repo.Create(ctx, log)
}
