package models

import (
	"time"

	"github.com/google/uuid"
)

type AlertAcknowledgment struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"user_id"`
	AlertFingerprint string    `json:"alert_fingerprint"`
	AcknowledgedAt   time.Time `json:"acknowledged_at"`
}
