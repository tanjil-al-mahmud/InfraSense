package models

import (
	"time"

	"github.com/google/uuid"
)

type MaintenanceWindow struct {
	ID        uuid.UUID  `json:"id"`
	DeviceID  uuid.UUID  `json:"device_id"`
	StartTime time.Time  `json:"start_time"`
	EndTime   time.Time  `json:"end_time"`
	Reason    string     `json:"reason"`
	CreatedBy uuid.UUID  `json:"created_by"`
	SilenceID *string    `json:"silence_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type MaintenanceWindowCreateRequest struct {
	DeviceID  uuid.UUID `json:"device_id" binding:"required"`
	StartTime time.Time `json:"start_time" binding:"required"`
	EndTime   time.Time `json:"end_time" binding:"required"`
	Reason    string    `json:"reason" binding:"required"`
}
