package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/models"
)

type HardwareEventRepository struct {
	db *DB
}

func NewHardwareEventRepository(db *DB) *HardwareEventRepository {
	return &HardwareEventRepository{db: db}
}

// InsertIfNotExists inserts a hardware event; duplicates (same device_id + dedupe_key) are ignored.
func (r *HardwareEventRepository) InsertIfNotExists(ctx context.Context, ev models.HardwareEvent) error {
	classJSON, _ := json.Marshal(ev.Classification)
	rawJSON, _ := json.Marshal(ev.Raw)

	_, err := r.db.conn.ExecContext(ctx, `
		INSERT INTO hardware_events (
			id, device_id, occurred_at, observed_at, source_protocol,
			vendor, model, firmware, component, event_type, severity, message,
			classification, raw, dedupe_key
		) VALUES (
			COALESCE($1, gen_random_uuid()), $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11, $12,
			$13::jsonb, $14::jsonb, $15
		)
		ON CONFLICT (device_id, dedupe_key) DO NOTHING
	`,
		uuidOrNil(ev.ID),
		ev.DeviceID,
		ev.OccurredAt,
		ev.ObservedAt,
		ev.SourceProtocol,
		ev.Vendor,
		ev.Model,
		ev.Firmware,
		ev.Component,
		ev.EventType,
		ev.Severity,
		ev.Message,
		classJSON,
		rawJSON,
		ev.DedupeKey,
	)
	if err != nil {
		return fmt.Errorf("insert hardware event: %w", err)
	}
	return nil
}

func uuidOrNil(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

