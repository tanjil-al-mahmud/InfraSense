package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/models"
)

type DeviceCredentialRepository struct {
	db *DB
}

func NewDeviceCredentialRepository(db *DB) *DeviceCredentialRepository {
	return &DeviceCredentialRepository{db: db}
}

// Create creates a new device credential
func (r *DeviceCredentialRepository) Create(ctx context.Context, cred *models.DeviceCredential) error {
	query := `
		INSERT INTO device_credentials (
			id, device_id, protocol, username, password_encrypted, community_string_encrypted,
			auth_protocol, priv_protocol,
			port, http_scheme, ssl_verify, polling_interval, timeout_seconds, retry_attempts,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err := r.db.conn.ExecContext(
		ctx, query,
		cred.ID, cred.DeviceID, cred.Protocol, cred.Username,
		cred.PasswordEncrypted, cred.CommunityStringEncrypted,
		cred.AuthProtocol, cred.PrivProtocol,
		cred.Port, cred.HTTPScheme, cred.SSLVerify,
		cred.PollingInterval, cred.TimeoutSeconds, cred.RetryAttempts,
		cred.CreatedAt, cred.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create device credential: %w", err)
	}

	return nil
}

// GetByDeviceIDAndProtocol retrieves a credential by device ID and protocol
func (r *DeviceCredentialRepository) GetByDeviceIDAndProtocol(ctx context.Context, deviceID uuid.UUID, protocol string) (*models.DeviceCredential, error) {
	cred := &models.DeviceCredential{}

	query := `
		SELECT id, device_id, protocol, username, password_encrypted, community_string_encrypted,
		       auth_protocol, priv_protocol,
		       COALESCE(port, 0), COALESCE(http_scheme, 'https'), COALESCE(ssl_verify, false),
		       COALESCE(polling_interval, 60), COALESCE(timeout_seconds, 30), COALESCE(retry_attempts, 3),
		       created_at, updated_at
		FROM device_credentials
		WHERE device_id = $1 AND protocol = $2
	`

	var port int
	err := r.db.conn.QueryRowContext(ctx, query, deviceID, protocol).Scan(
		&cred.ID, &cred.DeviceID, &cred.Protocol, &cred.Username,
		&cred.PasswordEncrypted, &cred.CommunityStringEncrypted,
		&cred.AuthProtocol, &cred.PrivProtocol,
		&port, &cred.HTTPScheme, &cred.SSLVerify,
		&cred.PollingInterval, &cred.TimeoutSeconds, &cred.RetryAttempts,
		&cred.CreatedAt, &cred.UpdatedAt,
	)

	if port != 0 {
		cred.Port = &port
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("credential not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}

	return cred, nil
}

// Update updates a device credential
func (r *DeviceCredentialRepository) Update(ctx context.Context, cred *models.DeviceCredential) error {
	cred.UpdatedAt = time.Now()

	query := `
		UPDATE device_credentials
		SET username = $1, password_encrypted = $2, community_string_encrypted = $3,
		    auth_protocol = $4, priv_protocol = $5,
		    port = $6, http_scheme = $7, ssl_verify = $8,
		    polling_interval = $9, timeout_seconds = $10, retry_attempts = $11,
		    updated_at = $12
		WHERE device_id = $13 AND protocol = $14
	`

	result, err := r.db.conn.ExecContext(
		ctx, query,
		cred.Username, cred.PasswordEncrypted, cred.CommunityStringEncrypted,
		cred.AuthProtocol, cred.PrivProtocol,
		cred.Port, cred.HTTPScheme, cred.SSLVerify,
		cred.PollingInterval, cred.TimeoutSeconds, cred.RetryAttempts,
		cred.UpdatedAt, cred.DeviceID, cred.Protocol,
	)

	if err != nil {
		return fmt.Errorf("failed to update credential: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("credential not found")
	}

	return nil
}

// Delete deletes a device credential
func (r *DeviceCredentialRepository) Delete(ctx context.Context, deviceID uuid.UUID, protocol string) error {
	query := "DELETE FROM device_credentials WHERE device_id = $1 AND protocol = $2"

	result, err := r.db.conn.ExecContext(ctx, query, deviceID, protocol)
	if err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("credential not found")
	}

	return nil
}
