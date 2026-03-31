-- Migration 017: Add last_error, last_seen, protocol to devices
-- (community_string, snmp_version, auth/priv columns already added in 016)

ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS last_error  TEXT,
    ADD COLUMN IF NOT EXISTS last_seen   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS protocol    VARCHAR(50);

-- snmp_version and port on device_credentials (016 added community_string, auth/priv)
ALTER TABLE device_credentials
    ADD COLUMN IF NOT EXISTS snmp_version VARCHAR(10) DEFAULT 'v2c',
    ADD COLUMN IF NOT EXISTS port         INTEGER     DEFAULT 623;

CREATE INDEX IF NOT EXISTS idx_devices_protocol ON devices(protocol);
CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices(last_seen DESC);
