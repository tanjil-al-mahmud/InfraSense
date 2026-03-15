ALTER TABLE device_inventory
    DROP COLUMN IF EXISTS bmc_name,
    DROP COLUMN IF EXISTS system_uuid,
    DROP COLUMN IF EXISTS system_revision,
    DROP COLUMN IF EXISTS system_uptime_seconds,
    DROP COLUMN IF EXISTS boot_mode,
    DROP COLUMN IF EXISTS os_name,
    DROP COLUMN IF EXISTS os_version,
    DROP COLUMN IF EXISTS os_kernel,
    DROP COLUMN IF EXISTS os_hostname,
    DROP COLUMN IF EXISTS os_uptime_seconds,
    DROP COLUMN IF EXISTS lifecycle_logs_json,
    DROP COLUMN IF EXISTS virtual_disks_json,
    DROP COLUMN IF EXISTS storage_enclosures_json;
