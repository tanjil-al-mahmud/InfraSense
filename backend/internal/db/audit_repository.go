package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/models"
)

type AuditRepository struct {
	db *DB
}

func NewAuditRepository(db *DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create creates a new audit log entry
func (r *AuditRepository) Create(ctx context.Context, log *models.AuditLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()

	// Convert details map to JSON
	var detailsJSON []byte
	var err error
	if log.Details != nil {
		detailsJSON, err = json.Marshal(log.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal details: %w", err)
		}
	}

	query := `
		INSERT INTO audit_logs (id, user_id, action_type, target_resource, target_id, source_ip, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.conn.ExecContext(
		ctx, query,
		log.ID, log.UserID, log.ActionType, log.TargetResource,
		log.TargetID, log.SourceIP, detailsJSON, log.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// List retrieves audit logs with pagination
func (r *AuditRepository) List(ctx context.Context, page, pageSize int) ([]models.AuditLog, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Get total count
	var total int
	err := r.db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_logs").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Get paginated results
	offset := (page - 1) * pageSize

	query := `
		SELECT id, user_id, action_type, target_resource, target_id, source_ip, details, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.conn.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	logs := []models.AuditLog{}
	for rows.Next() {
		log := models.AuditLog{}
		var detailsJSON []byte

		err := rows.Scan(
			&log.ID, &log.UserID, &log.ActionType, &log.TargetResource,
			&log.TargetID, &log.SourceIP, &detailsJSON, &log.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}

		// Parse details JSON
		if detailsJSON != nil {
			if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal details: %w", err)
			}
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return logs, total, nil
}
