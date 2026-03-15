CREATE TABLE device_credentials (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  protocol VARCHAR(20) NOT NULL,
  username VARCHAR(255),
  password_encrypted BYTEA,
  community_string_encrypted BYTEA,
  auth_protocol VARCHAR(20),
  priv_protocol VARCHAR(20),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE(device_id, protocol)
);
