-- Add connection settings to device_credentials
ALTER TABLE device_credentials
    ADD COLUMN IF NOT EXISTS port              INTEGER,
    ADD COLUMN IF NOT EXISTS http_scheme       VARCHAR(8)  DEFAULT 'https',
    ADD COLUMN IF NOT EXISTS ssl_verify        BOOLEAN     DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS polling_interval  INTEGER     DEFAULT 60,
    ADD COLUMN IF NOT EXISTS timeout_seconds   INTEGER     DEFAULT 30,
    ADD COLUMN IF NOT EXISTS retry_attempts    INTEGER     DEFAULT 3;

-- Add connection/sync tracking to devices
ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS last_sync_at        TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS connection_status   VARCHAR(32) DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS connection_error    TEXT;
