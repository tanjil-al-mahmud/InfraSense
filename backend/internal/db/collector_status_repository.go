package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/models"
)

type CollectorStatusRepository struct {
	db *DB
}

func NewCollectorStatusRepository(db *DB) *CollectorStatusRepository {
	return &CollectorStatusRepository{db: db}
}

// GetByID retrieves a collector status by ID
func (r *CollectorStatusRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.CollectorStatus, error) {
	cs := &models.CollectorStatus{}

	query := `
		SELECT id, collector_name, collector_type, status, last_poll_time, last_success_time, last_error, updated_at
		FROM collector_status
		WHERE id = $1
	`

	err := r.db.conn.QueryRowContext(ctx, query, id).Scan(
		&cs.ID, &cs.CollectorName, &cs.CollectorType, &cs.Status,
		&cs.LastPollTime, &cs.LastSuccessTime, &cs.LastError, &cs.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("collector not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get collector: %w", err)
	}

	return cs, nil
}

// List retrieves collectors with pagination and optional filtering
func (r *CollectorStatusRepository) List(ctx context.Context, filter models.CollectorListFilter) ([]models.CollectorStatus, int, error) {
	whereClauses := []string{}
	args := []interface{}{}
	argPos := 1

	if filter.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *filter.Status)
		argPos++
	}

	if filter.CollectorType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("collector_type = $%d", argPos))
		args = append(args, *filter.CollectorType)
		argPos++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM collector_status %s", whereClause)
	var total int
	if err := r.db.conn.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count collectors: %w", err)
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	offset := (filter.Page - 1) * filter.PageSize

	query := fmt.Sprintf(`
		SELECT id, collector_name, collector_type, status, last_poll_time, last_success_time, last_error, updated_at
		FROM collector_status
		%s
		ORDER BY collector_name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, filter.PageSize, offset)

	rows, err := r.db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list collectors: %w", err)
	}
	defer rows.Close()

	collectors := []models.CollectorStatus{}
	for rows.Next() {
		cs := models.CollectorStatus{}
		if err := rows.Scan(
			&cs.ID, &cs.CollectorName, &cs.CollectorType, &cs.Status,
			&cs.LastPollTime, &cs.LastSuccessTime, &cs.LastError, &cs.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan collector: %w", err)
		}
		collectors = append(collectors, cs)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating collectors: %w", err)
	}

	return collectors, total, nil
}

// MarkStaleCollectorsUnhealthy marks collectors as unhealthy if updated_at is older than 10 minutes.
// Returns the number of collectors updated.
func (r *CollectorStatusRepository) MarkStaleCollectorsUnhealthy(ctx context.Context) (int, error) {
	query := `
		UPDATE collector_status
		SET status = 'unhealthy', updated_at = NOW()
		WHERE status != 'unhealthy'
		  AND updated_at < NOW() - INTERVAL '10 minutes'
		RETURNING id
	`

	rows, err := r.db.conn.QueryContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to mark stale collectors unhealthy: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return count, fmt.Errorf("failed to scan collector id: %w", err)
		}
		count++
	}

	if err := rows.Err(); err != nil {
		return count, fmt.Errorf("error iterating stale collectors: %w", err)
	}

	return count, nil
}
