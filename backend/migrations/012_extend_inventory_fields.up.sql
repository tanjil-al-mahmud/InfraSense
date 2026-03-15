-- Migration 012: Add missing inventory fields referenced by device_repository.go
-- Adds: bmc_name, system_uuid, system_revision, system_uptime_seconds, boot_mode
-- Also adds missing spec fields: os_info, psu_redundancy, power_cap, disk_temperature,
-- dimm_serial, nic_firmware, storage_enclosures, lifecycle_logs_json

ALTER TABLE device_inventory
    ADD COLUMN IF NOT EXISTS bmc_name                 VARCHAR(255),
    ADD COLUMN IF NOT EXISTS system_uuid              VARCHAR(100),
    ADD COLUMN IF NOT EXISTS system_revision          VARCHAR(100),
    ADD COLUMN IF NOT EXISTS system_uptime_seconds    BIGINT,
    ADD COLUMN IF NOT EXISTS boot_mode                VARCHAR(50),
    ADD COLUMN IF NOT EXISTS os_name                  VARCHAR(255),
    ADD COLUMN IF NOT EXISTS os_version               VARCHAR(255),
    ADD COLUMN IF NOT EXISTS os_kernel                VARCHAR(255),
    ADD COLUMN IF NOT EXISTS os_hostname              VARCHAR(255),
    ADD COLUMN IF NOT EXISTS os_uptime_seconds        BIGINT,
    ADD COLUMN IF NOT EXISTS lifecycle_logs_json      JSONB,
    ADD COLUMN IF NOT EXISTS virtual_disks_json       JSONB,
    ADD COLUMN IF NOT EXISTS storage_enclosures_json  JSONB;
