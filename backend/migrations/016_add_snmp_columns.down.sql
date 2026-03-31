-- Down migration file: backend/migrations/016_add_snmp_columns.down.sql
ALTER TABLE device_credentials
DROP COLUMN IF EXISTS community_string,
DROP COLUMN IF EXISTS auth_protocol,
DROP COLUMN IF EXISTS auth_password_encrypted,
DROP COLUMN IF EXISTS priv_protocol,
DROP COLUMN IF EXISTS priv_password_encrypted;
