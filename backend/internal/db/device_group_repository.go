package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/models"
)

type DeviceGroupRepository struct {
	db *DB
}

func NewDeviceGroupRepository(db *DB) *DeviceGroupRepository {
	return &DeviceGroupRepository{db: db}
}

// Create creates a new device group
func (r *DeviceGroupRepository) Create(ctx context.Context, req models.DeviceGroupCreateRequest) (*models.DeviceGroup, error) {
	group := &models.DeviceGroup{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO device_groups (id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, description, created_at, updated_at
	`

	err := r.db.conn.QueryRowContext(
		ctx, query,
		group.ID, group.Name, group.Description, group.CreatedAt, group.UpdatedAt,
	).Scan(
		&group.ID, &group.Name, &group.Description, &group.CreatedAt, &group.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create device group: %w", err)
	}

	return group, nil
}

// List retrieves all device groups
func (r *DeviceGroupRepository) List(ctx context.Context) ([]models.DeviceGroup, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM device_groups
		ORDER BY name ASC
	`

	rows, err := r.db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list device groups: %w", err)
	}
	defer rows.Close()

	groups := []models.DeviceGroup{}
	for rows.Next() {
		group := models.DeviceGroup{}
		err := rows.Scan(
			&group.ID, &group.Name, &group.Description, &group.CreatedAt, &group.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device group: %w", err)
		}
		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating device groups: %w", err)
	}

	return groups, nil
}

// GetByID retrieves a device group by ID
func (r *DeviceGroupRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DeviceGroup, error) {
	group := &models.DeviceGroup{}

	query := `
		SELECT id, name, description, created_at, updated_at
		FROM device_groups
		WHERE id = $1
	`

	err := r.db.conn.QueryRowContext(ctx, query, id).Scan(
		&group.ID, &group.Name, &group.Description, &group.CreatedAt, &group.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("device group not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device group: %w", err)
	}

	return group, nil
}

// Update updates a device group
func (r *DeviceGroupRepository) Update(ctx context.Context, id uuid.UUID, req models.DeviceGroupUpdateRequest) (*models.DeviceGroup, error) {
	// Get existing group
	group, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Name != nil {
		group.Name = *req.Name
	}
	if req.Description != nil {
		group.Description = req.Description
	}
	group.UpdatedAt = time.Now()

	query := `
		UPDATE device_groups
		SET name = $1, description = $2, updated_at = $3
		WHERE id = $4
		RETURNING id, name, description, created_at, updated_at
	`

	err = r.db.conn.QueryRowContext(
		ctx, query,
		group.Name, group.Description, group.UpdatedAt, id,
	).Scan(
		&group.ID, &group.Name, &group.Description, &group.CreatedAt, &group.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update device group: %w", err)
	}

	return group, nil
}

// Delete deletes a device group (preserves devices)
func (r *DeviceGroupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM device_groups WHERE id = $1"

	result, err := r.db.conn.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete device group: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("device group not found")
	}

	return nil
}

// AddDevice adds a device to a group
func (r *DeviceGroupRepository) AddDevice(ctx context.Context, groupID, deviceID uuid.UUID) error {
	query := `
		INSERT INTO device_group_members (device_id, group_id)
		VALUES ($1, $2)
		ON CONFLICT (device_id, group_id) DO NOTHING
	`

	_, err := r.db.conn.ExecContext(ctx, query, deviceID, groupID)
	if err != nil {
		return fmt.Errorf("failed to add device to group: %w", err)
	}

	return nil
}

// RemoveDevice removes a device from a group
func (r *DeviceGroupRepository) RemoveDevice(ctx context.Context, groupID, deviceID uuid.UUID) error {
	query := "DELETE FROM device_group_members WHERE device_id = $1 AND group_id = $2"

	result, err := r.db.conn.ExecContext(ctx, query, deviceID, groupID)
	if err != nil {
		return fmt.Errorf("failed to remove device from group: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("device not in group")
	}

	return nil
}
