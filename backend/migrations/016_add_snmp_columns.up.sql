-- Create new migration file: backend/migrations/016_add_snmp_columns.up.sql
ALTER TABLE device_credentials
ADD COLUMN IF NOT EXISTS community_string VARCHAR(255),
ADD COLUMN IF NOT EXISTS auth_protocol VARCHAR(50),
ADD COLUMN IF NOT EXISTS auth_password_encrypted TEXT,
ADD COLUMN IF NOT EXISTS priv_protocol VARCHAR(50),
ADD COLUMN IF NOT EXISTS priv_password_encrypted TEXT;
