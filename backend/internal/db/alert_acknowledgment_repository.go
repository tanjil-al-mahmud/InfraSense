package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/models"
)

type AlertAcknowledgmentRepository struct {
	db *DB
}

func NewAlertAcknowledgmentRepository(db *DB) *AlertAcknowledgmentRepository {
	return &AlertAcknowledgmentRepository{db: db}
}

// Create creates a new alert acknowledgment
func (r *AlertAcknowledgmentRepository) Create(ctx context.Context, userID uuid.UUID, alertFingerprint string) (*models.AlertAcknowledgment, error) {
	query := `
		INSERT INTO alert_acknowledgments (user_id, alert_fingerprint)
		VALUES ($1, $2)
		ON CONFLICT (alert_fingerprint) DO UPDATE
		SET user_id = EXCLUDED.user_id, acknowledged_at = NOW()
		RETURNING id, user_id, alert_fingerprint, acknowledged_at
	`

	var ack models.AlertAcknowledgment
	err := r.db.conn.QueryRowContext(ctx, query, userID, alertFingerprint).Scan(
		&ack.ID,
		&ack.UserID,
		&ack.AlertFingerprint,
		&ack.AcknowledgedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create alert acknowledgment: %w", err)
	}

	return &ack, nil
}

// GetByFingerprint retrieves an alert acknowledgment by fingerprint
func (r *AlertAcknowledgmentRepository) GetByFingerprint(ctx context.Context, alertFingerprint string) (*models.AlertAcknowledgment, error) {
	query := `
		SELECT id, user_id, alert_fingerprint, acknowledged_at
		FROM alert_acknowledgments
		WHERE alert_fingerprint = $1
	`

	var ack models.AlertAcknowledgment
	err := r.db.conn.QueryRowContext(ctx, query, alertFingerprint).Scan(
		&ack.ID,
		&ack.UserID,
		&ack.AlertFingerprint,
		&ack.AcknowledgedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get alert acknowledgment: %w", err)
	}

	return &ack, nil
}

// List retrieves all alert acknowledgments
func (r *AlertAcknowledgmentRepository) List(ctx context.Context) ([]models.AlertAcknowledgment, error) {
	query := `
		SELECT id, user_id, alert_fingerprint, acknowledged_at
		FROM alert_acknowledgments
		ORDER BY acknowledged_at DESC
	`

	rows, err := r.db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert acknowledgments: %w", err)
	}
	defer rows.Close()

	var acks []models.AlertAcknowledgment
	for rows.Next() {
		var ack models.AlertAcknowledgment
		if err := rows.Scan(&ack.ID, &ack.UserID, &ack.AlertFingerprint, &ack.AcknowledgedAt); err != nil {
			return nil, fmt.Errorf("failed to scan alert acknowledgment: %w", err)
		}
		acks = append(acks, ack)
	}

	return acks, nil
}
