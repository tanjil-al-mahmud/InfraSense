package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/models"
)

// HardwareEventRepository provides event query methods on top of the
// existing hardware_events table (created in migration 015).
type HardwareEventListFilter struct {
	DeviceID  *uuid.UUID
	Severity  *string
	Search    *string
	DateFrom  *time.Time
	DateTo    *time.Time
	Page      int
	PageSize  int
}

type HardwareEventSummary struct {
	TotalCritical int                       `json:"total_critical"`
	TotalWarning  int                       `json:"total_warning"`
	TotalInfo     int                       `json:"total_info"`
	Last24h       int                       `json:"last_24h"`
	Last7d        int                       `json:"last_7d"`
	ByDevice      []HardwareEventDeviceCount `json:"by_device"`
}

type HardwareEventDeviceCount struct {
	DeviceID  string `json:"device_id"`
	Hostname  string `json:"hostname"`
	Count     int    `json:"count"`
	MaxSeverity string `json:"max_severity"`
}

// ListEvents returns paginated hardware events using the existing hardware_events table.
func (r *HardwareEventRepository) ListEvents(ctx context.Context, f HardwareEventListFilter) ([]models.HardwareEvent, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 200 {
		f.PageSize = 50
	}
	offset := (f.Page - 1) * f.PageSize

	args := []interface{}{}
	idx := 1

	where := "WHERE 1=1"
	if f.DeviceID != nil {
		where += fmt.Sprintf(" AND he.device_id = $%d", idx)
		args = append(args, *f.DeviceID)
		idx++
	}
	if f.Severity != nil && *f.Severity != "" && *f.Severity != "all" {
		where += fmt.Sprintf(" AND he.severity = $%d", idx)
		args = append(args, *f.Severity)
		idx++
	}
	if f.Search != nil && *f.Search != "" {
		where += fmt.Sprintf(" AND he.message ILIKE $%d", idx)
		args = append(args, "%"+*f.Search+"%")
		idx++
	}
	if f.DateFrom != nil {
		where += fmt.Sprintf(" AND COALESCE(he.occurred_at, he.observed_at) >= $%d", idx)
		args = append(args, *f.DateFrom)
		idx++
	}
	if f.DateTo != nil {
		where += fmt.Sprintf(" AND COALESCE(he.occurred_at, he.observed_at) <= $%d", idx)
		args = append(args, *f.DateTo)
		idx++
	}

	// Count total
	countSQL := `SELECT COUNT(*) FROM hardware_events he ` + where
	var total int
	if err := r.db.conn.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count events: %w", err)
	}

	// Fetch page
	querySQL := fmt.Sprintf(`
		SELECT
			he.id, he.device_id,
			COALESCE(d.hostname, '') as hostname,
			COALESCE(he.occurred_at, he.observed_at) as event_time,
			he.source_protocol, he.component, he.event_type,
			he.severity, he.message, he.dedupe_key, he.created_at
		FROM hardware_events he
		LEFT JOIN devices d ON d.id = he.device_id
		%s
		ORDER BY COALESCE(he.occurred_at, he.observed_at) DESC
		LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)
	args = append(args, f.PageSize, offset)

	rows, err := r.db.conn.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var events []models.HardwareEvent
	for rows.Next() {
		var ev models.HardwareEvent
		var hostname string
		var eventTime time.Time
		if err := rows.Scan(
			&ev.ID, &ev.DeviceID, &hostname,
			&eventTime, &ev.SourceProtocol, &ev.Component, &ev.EventType,
			&ev.Severity, &ev.Message, &ev.DedupeKey, &ev.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan event: %w", err)
		}
		ev.OccurredAt = &eventTime
		ev.Vendor = &hostname // carries hostname for API response
		events = append(events, ev)
	}
	if events == nil {
		events = []models.HardwareEvent{}
	}
	return events, total, rows.Err()
}

// GetEventSummary returns counts by severity and counts for last 24h / 7d.
func (r *HardwareEventRepository) GetEventSummary(ctx context.Context) (*HardwareEventSummary, error) {
	summary := &HardwareEventSummary{
		ByDevice: []HardwareEventDeviceCount{},
	}

	// Severity counts
	rows, err := r.db.conn.QueryContext(ctx, `
		SELECT severity, COUNT(*) FROM hardware_events GROUP BY severity
	`)
	if err != nil {
		return nil, fmt.Errorf("severity counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sev string
		var cnt int
		if err := rows.Scan(&sev, &cnt); err != nil {
			continue
		}
		switch sev {
		case "critical":
			summary.TotalCritical = cnt
		case "warning":
			summary.TotalWarning = cnt
		default:
			summary.TotalInfo += cnt
		}
	}

	// Time-bucket counts
	_ = r.db.conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM hardware_events WHERE COALESCE(occurred_at, observed_at) >= NOW() - INTERVAL '24 hours'`,
	).Scan(&summary.Last24h)

	_ = r.db.conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM hardware_events WHERE COALESCE(occurred_at, observed_at) >= NOW() - INTERVAL '7 days'`,
	).Scan(&summary.Last7d)

	// Per-device breakdown (top 20)
	devRows, err := r.db.conn.QueryContext(ctx, `
		SELECT he.device_id::text, COALESCE(d.hostname,''), COUNT(*) as cnt,
			   MAX(CASE he.severity WHEN 'critical' THEN 3 WHEN 'warning' THEN 2 ELSE 1 END)
		FROM hardware_events he
		LEFT JOIN devices d ON d.id = he.device_id
		GROUP BY he.device_id, d.hostname
		ORDER BY cnt DESC
		LIMIT 20
	`)
	if err == nil {
		defer devRows.Close()
		for devRows.Next() {
			var dc HardwareEventDeviceCount
			var sevCode int
			if err := devRows.Scan(&dc.DeviceID, &dc.Hostname, &dc.Count, &sevCode); err != nil {
				continue
			}
			switch sevCode {
			case 3:
				dc.MaxSeverity = "critical"
			case 2:
				dc.MaxSeverity = "warning"
			default:
				dc.MaxSeverity = "info"
			}
			summary.ByDevice = append(summary.ByDevice, dc)
		}
	}

	return summary, nil
}

// ClearDeviceEvents deletes all hardware events for a given device.
func (r *HardwareEventRepository) ClearDeviceEvents(ctx context.Context, deviceID uuid.UUID) (int64, error) {
	res, err := r.db.conn.ExecContext(ctx,
		`DELETE FROM hardware_events WHERE device_id = $1`, deviceID)
	if err != nil {
		return 0, fmt.Errorf("clear device events: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
