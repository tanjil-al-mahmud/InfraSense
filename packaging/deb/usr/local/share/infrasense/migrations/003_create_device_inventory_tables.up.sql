CREATE TABLE device_inventory (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  cpu_model VARCHAR(255),
  cpu_cores INT,
  cpu_threads INT,
  ram_total_gb INT,
  firmware_bmc VARCHAR(100),
  firmware_bios VARCHAR(100),
  firmware_raid VARCHAR(100),
  collected_at TIMESTAMP NOT NULL,
  UNIQUE(device_id)
);

CREATE TABLE device_nics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  nic_model VARCHAR(255),
  mac_address MACADDR NOT NULL,
  collected_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_device_nics_device_id ON device_nics(device_id);

CREATE TABLE device_disks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  disk_model VARCHAR(255),
  capacity_gb INT,
  serial_number VARCHAR(255),
  collected_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_device_disks_device_id ON device_disks(device_id);
