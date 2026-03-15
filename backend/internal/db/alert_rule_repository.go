package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/models"
)

type AlertRuleRepository struct {
	db *DB
}

func NewAlertRuleRepository(db *DB) *AlertRuleRepository {
	return &AlertRuleRepository{db: db}
}

// Create creates a new alert rule
func (r *AlertRuleRepository) Create(ctx context.Context, req models.AlertRuleCreateRequest) (*models.AlertRule, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule := &models.AlertRule{
		ID:            uuid.New(),
		Name:          req.Name,
		MetricName:    req.MetricName,
		Operator:      req.Operator,
		Threshold:     req.Threshold,
		Severity:      req.Severity,
		DeviceID:      req.DeviceID,
		DeviceGroupID: req.DeviceGroupID,
		Enabled:       enabled,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	query := `
		INSERT INTO alert_rules (id, name, metric_name, comparison_operator, threshold_value, severity, device_id, device_group_id, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, name, metric_name, comparison_operator, threshold_value, severity, device_id, device_group_id, enabled, created_at, updated_at
	`

	err := r.db.conn.QueryRowContext(
		ctx, query,
		rule.ID, rule.Name, rule.MetricName, rule.Operator, rule.Threshold,
		rule.Severity, rule.DeviceID, rule.DeviceGroupID, rule.Enabled, rule.CreatedAt, rule.UpdatedAt,
	).Scan(
		&rule.ID, &rule.Name, &rule.MetricName, &rule.Operator, &rule.Threshold,
		&rule.Severity, &rule.DeviceID, &rule.DeviceGroupID, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create alert rule: %w", err)
	}

	return rule, nil
}

// GetByID retrieves an alert rule by ID
func (r *AlertRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.AlertRule, error) {
	rule := &models.AlertRule{}

	query := `
		SELECT id, name, metric_name, comparison_operator, threshold_value, severity, device_id, device_group_id, enabled, created_at, updated_at
		FROM alert_rules
		WHERE id = $1
	`

	err := r.db.conn.QueryRowContext(ctx, query, id).Scan(
		&rule.ID, &rule.Name, &rule.MetricName, &rule.Operator, &rule.Threshold,
		&rule.Severity, &rule.DeviceID, &rule.DeviceGroupID, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("alert rule not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get alert rule: %w", err)
	}

	return rule, nil
}

// List retrieves all alert rules
func (r *AlertRuleRepository) List(ctx context.Context) ([]models.AlertRule, error) {
	query := `
		SELECT id, name, metric_name, comparison_operator, threshold_value, severity, device_id, device_group_id, enabled, created_at, updated_at
		FROM alert_rules
		ORDER BY created_at DESC
	`

	rows, err := r.db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert rules: %w", err)
	}
	defer rows.Close()

	rules := []models.AlertRule{}
	for rows.Next() {
		rule := models.AlertRule{}
		err := rows.Scan(
			&rule.ID, &rule.Name, &rule.MetricName, &rule.Operator, &rule.Threshold,
			&rule.Severity, &rule.DeviceID, &rule.DeviceGroupID, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert rule: %w", err)
		}
		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating alert rules: %w", err)
	}

	return rules, nil
}

// Update updates an alert rule
func (r *AlertRuleRepository) Update(ctx context.Context, id uuid.UUID, req models.AlertRuleUpdateRequest) (*models.AlertRule, error) {
	// Build SET clause dynamically
	setClauses := []string{"updated_at = $1"}
	args := []interface{}{time.Now()}
	argPos := 2

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argPos))
		args = append(args, *req.Name)
		argPos++
	}

	if req.MetricName != nil {
		setClauses = append(setClauses, fmt.Sprintf("metric_name = $%d", argPos))
		args = append(args, *req.MetricName)
		argPos++
	}

	if req.Operator != nil {
		setClauses = append(setClauses, fmt.Sprintf("comparison_operator = $%d", argPos))
		args = append(args, *req.Operator)
		argPos++
	}

	if req.Threshold != nil {
		setClauses = append(setClauses, fmt.Sprintf("threshold_value = $%d", argPos))
		args = append(args, *req.Threshold)
		argPos++
	}

	if req.Severity != nil {
		setClauses = append(setClauses, fmt.Sprintf("severity = $%d", argPos))
		args = append(args, *req.Severity)
		argPos++
	}

	if req.DeviceID != nil {
		setClauses = append(setClauses, fmt.Sprintf("device_id = $%d", argPos))
		args = append(args, *req.DeviceID)
		argPos++
	}

	if req.DeviceGroupID != nil {
		setClauses = append(setClauses, fmt.Sprintf("device_group_id = $%d", argPos))
		args = append(args, *req.DeviceGroupID)
		argPos++
	}

	if req.Enabled != nil {
		setClauses = append(setClauses, fmt.Sprintf("enabled = $%d", argPos))
		args = append(args, *req.Enabled)
		argPos++
	}

	args = append(args, id)

	// Rebuild query properly
	setClause := setClauses[0]
	for i := 1; i < len(setClauses); i++ {
		setClause += ", " + setClauses[i]
	}

	query := fmt.Sprintf(`
		UPDATE alert_rules
		SET %s
		WHERE id = $%d
		RETURNING id, name, metric_name, comparison_operator, threshold_value, severity, device_id, device_group_id, enabled, created_at, updated_at
	`, setClause, argPos)

	rule := &models.AlertRule{}
	err := r.db.conn.QueryRowContext(ctx, query, args...).Scan(
		&rule.ID, &rule.Name, &rule.MetricName, &rule.Operator, &rule.Threshold,
		&rule.Severity, &rule.DeviceID, &rule.DeviceGroupID, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("alert rule not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update alert rule: %w", err)
	}

	return rule, nil
}

// Delete deletes an alert rule
func (r *AlertRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM alert_rules WHERE id = $1"

	result, err := r.db.conn.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete alert rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("alert rule not found")
	}

	return nil
}
