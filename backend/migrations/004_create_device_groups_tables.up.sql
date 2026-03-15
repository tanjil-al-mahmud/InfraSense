CREATE TABLE device_groups (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL UNIQUE,
  description TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE device_group_members (
  device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  group_id UUID NOT NULL REFERENCES device_groups(id) ON DELETE CASCADE,
  PRIMARY KEY (device_id, group_id)
);

CREATE INDEX idx_device_group_members_group_id ON device_group_members(group_id);
