package models

import (
	"time"

	"github.com/google/uuid"
)

type HardwareEvent struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	DeviceID       uuid.UUID              `json:"device_id" db:"device_id"`
	OccurredAt     *time.Time             `json:"occurred_at,omitempty" db:"occurred_at"`
	ObservedAt     time.Time              `json:"observed_at" db:"observed_at"`
	SourceProtocol string                 `json:"source_protocol" db:"source_protocol"`
	Vendor         *string                `json:"vendor,omitempty" db:"vendor"`
	Model          *string                `json:"model,omitempty" db:"model"`
	Firmware       *string                `json:"firmware,omitempty" db:"firmware"`
	Component      string                 `json:"component" db:"component"`
	EventType      string                 `json:"event_type" db:"event_type"`
	Severity       string                 `json:"severity" db:"severity"`
	Message        string                 `json:"message" db:"message"`
	Classification map[string]any         `json:"classification,omitempty" db:"classification"`
	Raw            map[string]any         `json:"raw,omitempty" db:"raw"`
	DedupeKey      string                 `json:"dedupe_key" db:"dedupe_key"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

