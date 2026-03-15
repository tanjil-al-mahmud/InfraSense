-- Add vendor, management controller type, and protocol fields to devices
ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS vendor                  VARCHAR(100),
    ADD COLUMN IF NOT EXISTS management_controller   VARCHAR(100),
    ADD COLUMN IF NOT EXISTS protocol                VARCHAR(20),
    ADD COLUMN IF NOT EXISTS polling_interval        INTEGER DEFAULT 60,
    ADD COLUMN IF NOT EXISTS ssl_verify              BOOLEAN DEFAULT FALSE;

-- Extend device_inventory with full telemetry fields
ALTER TABLE device_inventory
    ADD COLUMN IF NOT EXISTS service_tag             VARCHAR(100),
    ADD COLUMN IF NOT EXISTS asset_tag               VARCHAR(100),
    ADD COLUMN IF NOT EXISTS system_model            VARCHAR(255),
    ADD COLUMN IF NOT EXISTS manufacturer            VARCHAR(255),
    ADD COLUMN IF NOT EXISTS power_state             VARCHAR(50),
    ADD COLUMN IF NOT EXISTS health_status           VARCHAR(50),
    ADD COLUMN IF NOT EXISTS bmc_firmware            VARCHAR(100),
    ADD COLUMN IF NOT EXISTS bmc_mac_address         VARCHAR(50),
    ADD COLUMN IF NOT EXISTS bmc_dns_name            VARCHAR(255),
    ADD COLUMN IF NOT EXISTS bmc_license             VARCHAR(255),
    ADD COLUMN IF NOT EXISTS bmc_hardware_version    VARCHAR(100),
    ADD COLUMN IF NOT EXISTS lifecycle_controller_ver VARCHAR(100),
    ADD COLUMN IF NOT EXISTS total_power_watts       NUMERIC(10,2),
    ADD COLUMN IF NOT EXISTS memory_total_gb         NUMERIC(10,2);

-- Store full sync telemetry as JSONB for processors, memory, storage, thermal, power, NICs, SEL
ALTER TABLE device_inventory
    ADD COLUMN IF NOT EXISTS processors_json         JSONB,
    ADD COLUMN IF NOT EXISTS memory_modules_json     JSONB,
    ADD COLUMN IF NOT EXISTS storage_controllers_json JSONB,
    ADD COLUMN IF NOT EXISTS drives_json             JSONB,
    ADD COLUMN IF NOT EXISTS temperatures_json       JSONB,
    ADD COLUMN IF NOT EXISTS fans_json               JSONB,
    ADD COLUMN IF NOT EXISTS power_supplies_json     JSONB,
    ADD COLUMN IF NOT EXISTS nics_json               JSONB,
    ADD COLUMN IF NOT EXISTS sel_entries_json        JSONB,
    ADD COLUMN IF NOT EXISTS accelerators_json       JSONB,
    ADD COLUMN IF NOT EXISTS pcie_slots_json         JSONB,
    ADD COLUMN IF NOT EXISTS voltages_json           JSONB;
