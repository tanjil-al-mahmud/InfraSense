ALTER TABLE device_credentials
    DROP COLUMN IF EXISTS port,
    DROP COLUMN IF EXISTS http_scheme,
    DROP COLUMN IF EXISTS ssl_verify,
    DROP COLUMN IF EXISTS polling_interval,
    DROP COLUMN IF EXISTS timeout_seconds,
    DROP COLUMN IF EXISTS retry_attempts;

ALTER TABLE devices
    DROP COLUMN IF EXISTS last_sync_at,
    DROP COLUMN IF EXISTS connection_status,
    DROP COLUMN IF EXISTS connection_error;
