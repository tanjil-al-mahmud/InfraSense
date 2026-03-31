-- Migration 015: Hardware events + protocol capability tables

CREATE TABLE IF NOT EXISTS hardware_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  occurred_at TIMESTAMPTZ,
  observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  source_protocol VARCHAR(20) NOT NULL,
  vendor VARCHAR(100),
  model VARCHAR(100),
  firmware VARCHAR(100),
  component VARCHAR(50) NOT NULL,
  event_type VARCHAR(50) NOT NULL,
  severity VARCHAR(20) NOT NULL,
  message TEXT NOT NULL,
  classification JSONB,
  raw JSONB,
  dedupe_key TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Idempotency: prevent duplicate inserts per device
CREATE UNIQUE INDEX IF NOT EXISTS idx_hardware_events_device_dedupe
  ON hardware_events(device_id, dedupe_key);

CREATE INDEX IF NOT EXISTS idx_hardware_events_device_time
  ON hardware_events(device_id, occurred_at DESC, observed_at DESC);

CREATE INDEX IF NOT EXISTS idx_hardware_events_severity_time
  ON hardware_events(severity, observed_at DESC);


CREATE TABLE IF NOT EXISTS device_protocol_capabilities (
  device_id UUID PRIMARY KEY REFERENCES devices(id) ON DELETE CASCADE,
  supported_protocols TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
  preferred_protocol VARCHAR(20),
  last_success_protocol VARCHAR(20),
  last_failure_protocol VARCHAR(20),
  last_failure_reason TEXT,
  last_probe_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_device_protocol_capabilities_protocols
  ON device_protocol_capabilities USING GIN (supported_protocols);


CREATE TABLE IF NOT EXISTS log_cursors (
  device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  source_protocol VARCHAR(20) NOT NULL,
  cursor TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (device_id, source_protocol)
);

