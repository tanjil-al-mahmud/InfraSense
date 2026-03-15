CREATE TABLE devices (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  hostname VARCHAR(255) NOT NULL,
  ip_address INET NOT NULL,
  bmc_ip_address INET,
  device_type VARCHAR(50) NOT NULL,
  location VARCHAR(255),
  tags TEXT[],
  status VARCHAR(20) NOT NULL DEFAULT 'unknown',
  last_seen TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_devices_device_type ON devices(device_type);
CREATE INDEX idx_devices_status ON devices(status);
