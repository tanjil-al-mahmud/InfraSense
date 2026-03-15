package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/models"
)

type MaintenanceWindowRepository struct {
	db *DB
}

func NewMaintenanceWindowRepository(db *DB) *MaintenanceWindowRepository {
	return &MaintenanceWindowRepository{db: db}
}

// Create creates a new maintenance window
func (r *MaintenanceWindowRepository) Create(ctx context.Context, req models.MaintenanceWindowCreateRequest, createdBy uuid.UUID) (*models.MaintenanceWindow, error) {
	window := &models.MaintenanceWindow{
		ID:        uuid.New(),
		DeviceID:  req.DeviceID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Reason:    req.Reason,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}

	query := `
		INSERT INTO maintenance_windows (id, device_id, start_time, end_time, reason, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, device_id, start_time, end_time, reason, created_by, silence_id, created_at
	`

	err := r.db.conn.QueryRowContext(
		ctx, query,
		window.ID, window.DeviceID, window.StartTime, window.EndTime,
		window.Reason, window.CreatedBy, window.CreatedAt,
	).Scan(
		&window.ID, &window.DeviceID, &window.StartTime, &window.EndTime,
		&window.Reason, &window.CreatedBy, &window.SilenceID, &window.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create maintenance window: %w", err)
	}

	return window, nil
}

// GetByID retrieves a maintenance window by ID
func (r *MaintenanceWindowRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.MaintenanceWindow, error) {
	window := &models.MaintenanceWindow{}

	query := `
		SELECT id, device_id, start_time, end_time, reason, created_by, silence_id, created_at
		FROM maintenance_windows
		WHERE id = $1
	`

	err := r.db.conn.QueryRowContext(ctx, query, id).Scan(
		&window.ID, &window.DeviceID, &window.StartTime, &window.EndTime,
		&window.Reason, &window.CreatedBy, &window.SilenceID, &window.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("maintenance window not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get maintenance window: %w", err)
	}

	return window, nil
}

// List retrieves maintenance windows with optional active filter
func (r *MaintenanceWindowRepository) List(ctx context.Context, activeOnly bool) ([]models.MaintenanceWindow, error) {
	query := `
		SELECT id, device_id, start_time, end_time, reason, created_by, silence_id, created_at
		FROM maintenance_windows
	`

	if activeOnly {
		query += " WHERE start_time <= NOW() AND end_time >= NOW()"
	}

	query += " ORDER BY start_time DESC"

	rows, err := r.db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list maintenance windows: %w", err)
	}
	defer rows.Close()

	windows := []models.MaintenanceWindow{}
	for rows.Next() {
		window := models.MaintenanceWindow{}
		err := rows.Scan(
			&window.ID, &window.DeviceID, &window.StartTime, &window.EndTime,
			&window.Reason, &window.CreatedBy, &window.SilenceID, &window.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan maintenance window: %w", err)
		}
		windows = append(windows, window)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating maintenance windows: %w", err)
	}

	return windows, nil
}

// Delete deletes a maintenance window
func (r *MaintenanceWindowRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM maintenance_windows WHERE id = $1"

	result, err := r.db.conn.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete maintenance window: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("maintenance window not found")
	}

	return nil
}

// UpdateSilenceID updates the silence_id for a maintenance window
func (r *MaintenanceWindowRepository) UpdateSilenceID(ctx context.Context, id uuid.UUID, silenceID string) error {
	query := "UPDATE maintenance_windows SET silence_id = $1 WHERE id = $2"

	result, err := r.db.conn.ExecContext(ctx, query, silenceID, id)
	if err != nil {
		return fmt.Errorf("failed to update silence_id: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("maintenance window not found")
	}

	return nil
}
