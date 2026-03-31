-- Rollback migration 017
DROP INDEX IF EXISTS idx_devices_protocol;
DROP INDEX IF EXISTS idx_devices_last_seen;

ALTER TABLE device_credentials
    DROP COLUMN IF EXISTS snmp_version,
    DROP COLUMN IF EXISTS port;

ALTER TABLE devices
    DROP COLUMN IF EXISTS last_error,
    DROP COLUMN IF EXISTS last_seen,
    DROP COLUMN IF EXISTS protocol;
