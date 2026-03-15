package collector

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

type InventoryData struct {
	CPUModel         string
	CPUCores         int
	CPUThreads       int
	TotalRAMGB       int
	NICs             []NICInfo
	Disks            []DiskInfo
	FirmwareVersions map[string]string
}

type NICInfo struct {
	Model      string
	MACAddress string
}

type DiskInfo struct {
	Model        string
	CapacityGB   int
	SerialNumber string
}

// collectInventory collects hardware inventory from Redfish API.
func (c *RedfishCollector) collectInventory(ctx context.Context, device Device) (*InventoryData, error) {
	inv := &InventoryData{
		NICs:             make([]NICInfo, 0),
		Disks:            make([]DiskInfo, 0),
		FirmwareVersions: make(map[string]string),
	}

	if err := c.collectCPUInfo(ctx, device, inv); err != nil {
		slog.Warn("cpu info failed", "device_id", device.ID, "error", err.Error())
	}
	if err := c.collectMemoryInfo(ctx, device, inv); err != nil {
		slog.Warn("memory info failed", "device_id", device.ID, "error", err.Error())
	}
	if err := c.collectNICInfo(ctx, device, inv); err != nil {
		slog.Warn("nic info failed", "device_id", device.ID, "error", err.Error())
	}
	if err := c.collectDiskInfo(ctx, device, inv); err != nil {
		slog.Warn("disk info failed", "device_id", device.ID, "error", err.Error())
	}
	if err := c.collectFirmwareInfo(ctx, device, inv); err != nil {
		slog.Warn("firmware info failed", "device_id", device.ID, "error", err.Error())
	}

	return inv, nil
}

func (c *RedfishCollector) collectCPUInfo(ctx context.Context, device Device, inv *InventoryData) error {
	ids, err := c.listMembers(ctx, device, "/redfish/v1/Systems/System.Embedded.1/Processors")
	if err != nil {
		ids2, err2 := c.listMembers(ctx, device, "/redfish/v1/Systems/1/Processors")
		if err2 != nil {
			return fmt.Errorf("processors: %w", err)
		}
		ids = ids2
	}
	if len(ids) == 0 {
		return nil
	}
	cpuData, err := c.redfishRequest(ctx, device, ids[0])
	if err != nil {
		return err
	}
	inv.CPUModel, _ = cpuData["Model"].(string)
	if v, ok := cpuData["TotalCores"].(float64); ok {
		inv.CPUCores = int(v)
	}
	if v, ok := cpuData["TotalThreads"].(float64); ok {
		inv.CPUThreads = int(v)
	}
	return nil
}

func (c *RedfishCollector) collectMemoryInfo(ctx context.Context, device Device, inv *InventoryData) error {
	ids, err := c.listMembers(ctx, device, "/redfish/v1/Systems/System.Embedded.1/Memory")
	if err != nil {
		ids2, err2 := c.listMembers(ctx, device, "/redfish/v1/Systems/1/Memory")
		if err2 != nil {
			return fmt.Errorf("memory: %w", err)
		}
		ids = ids2
	}
	totalMiB := 0
	for _, id := range ids {
		memData, err := c.redfishRequest(ctx, device, id)
		if err != nil {
			continue
		}
		if v, ok := memData["CapacityMiB"].(float64); ok {
			totalMiB += int(v)
		}
	}
	inv.TotalRAMGB = totalMiB / 1024
	return nil
}

func (c *RedfishCollector) collectNICInfo(ctx context.Context, device Device, inv *InventoryData) error {
	ids, err := c.listMembers(ctx, device, "/redfish/v1/Systems/System.Embedded.1/EthernetInterfaces")
	if err != nil {
		ids2, err2 := c.listMembers(ctx, device, "/redfish/v1/Systems/1/EthernetInterfaces")
		if err2 != nil {
			return fmt.Errorf("nics: %w", err)
		}
		ids = ids2
	}
	for _, id := range ids {
		nicData, err := c.redfishRequest(ctx, device, id)
		if err != nil {
			continue
		}
		mac, _ := nicData["MACAddress"].(string)
		if mac == "" {
			continue
		}
		model, _ := nicData["Description"].(string)
		inv.NICs = append(inv.NICs, NICInfo{Model: model, MACAddress: mac})
	}
	return nil
}

func (c *RedfishCollector) collectDiskInfo(ctx context.Context, device Device, inv *InventoryData) error {
	storageIDs, err := c.listMembers(ctx, device, "/redfish/v1/Systems/System.Embedded.1/Storage")
	if err != nil {
		storageIDs2, err2 := c.listMembers(ctx, device, "/redfish/v1/Systems/1/Storage")
		if err2 != nil {
			return fmt.Errorf("storage: %w", err)
		}
		storageIDs = storageIDs2
	}
	for _, storageID := range storageIDs {
		ctrlData, err := c.redfishRequest(ctx, device, storageID)
		if err != nil {
			continue
		}
		drives, _ := ctrlData["Drives"].([]any)
		for _, d := range drives {
			dm, ok := d.(map[string]any)
			if !ok {
				continue
			}
			driveID, _ := dm["@odata.id"].(string)
			if driveID == "" {
				continue
			}
			driveData, err := c.redfishRequest(ctx, device, driveID)
			if err != nil {
				continue
			}
			disk := DiskInfo{}
			disk.Model, _ = driveData["Model"].(string)
			disk.SerialNumber, _ = driveData["SerialNumber"].(string)
			if v, ok := driveData["CapacityBytes"].(float64); ok {
				disk.CapacityGB = int(v / 1024 / 1024 / 1024)
			}
			inv.Disks = append(inv.Disks, disk)
		}
	}
	return nil
}

func (c *RedfishCollector) collectFirmwareInfo(ctx context.Context, device Device, inv *InventoryData) error {
	// Try iDRAC manager endpoint first
	for _, path := range []string{"/redfish/v1/Managers/iDRAC.Embedded.1", "/redfish/v1/Managers/1"} {
		data, err := c.redfishRequest(ctx, device, path)
		if err != nil {
			continue
		}
		if v, ok := data["FirmwareVersion"].(string); ok && v != "" {
			inv.FirmwareVersions["BMC"] = v
			break
		}
	}
	return nil
}

// storeInventory persists inventory to PostgreSQL using the correct schema column names.
func (c *RedfishCollector) storeInventory(device Device, inv *InventoryData) error {
	changed, err := c.hasInventoryChanged(device.ID, inv)
	if err != nil {
		return fmt.Errorf("check inventory change: %w", err)
	}
	if !changed {
		slog.Debug("inventory unchanged, skipping update", "device_id", device.ID)
		return nil
	}

	now := time.Now()
	bmcVersion := inv.FirmwareVersions["BMC"]

	// device_inventory uses ram_total_gb and collected_at (per migration 003)
	_, err = c.db.Exec(`
		INSERT INTO device_inventory (device_id, cpu_model, cpu_cores, cpu_threads, ram_total_gb, firmware_bmc, collected_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (device_id) DO UPDATE SET
			cpu_model     = EXCLUDED.cpu_model,
			cpu_cores     = EXCLUDED.cpu_cores,
			cpu_threads   = EXCLUDED.cpu_threads,
			ram_total_gb  = EXCLUDED.ram_total_gb,
			firmware_bmc  = EXCLUDED.firmware_bmc,
			collected_at  = EXCLUDED.collected_at
	`, device.ID, inv.CPUModel, inv.CPUCores, inv.CPUThreads, inv.TotalRAMGB, bmcVersion, now)
	if err != nil {
		return fmt.Errorf("upsert device_inventory: %w", err)
	}

	if err := c.storeNICs(device.ID, inv.NICs, now); err != nil {
		return fmt.Errorf("store NICs: %w", err)
	}
	if err := c.storeDisks(device.ID, inv.Disks, now); err != nil {
		return fmt.Errorf("store disks: %w", err)
	}

	slog.Info("inventory updated", "event", "inventory_stored", "device_id", device.ID)
	return nil
}

func (c *RedfishCollector) hasInventoryChanged(deviceID string, inv *InventoryData) (bool, error) {
	var existingModel string
	var existingCores, existingThreads, existingRAM int
	err := c.db.QueryRow(
		`SELECT cpu_model, cpu_cores, cpu_threads, ram_total_gb FROM device_inventory WHERE device_id = $1::uuid`,
		deviceID,
	).Scan(&existingModel, &existingCores, &existingThreads, &existingRAM)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return existingModel != inv.CPUModel ||
		existingCores != inv.CPUCores ||
		existingThreads != inv.CPUThreads ||
		existingRAM != inv.TotalRAMGB, nil
}

func (c *RedfishCollector) storeNICs(deviceID string, nics []NICInfo, now time.Time) error {
	if _, err := c.db.Exec(`DELETE FROM device_nics WHERE device_id = $1::uuid`, deviceID); err != nil {
		return err
	}
	for _, nic := range nics {
		if nic.MACAddress == "" {
			continue
		}
		if _, err := c.db.Exec(
			`INSERT INTO device_nics (device_id, nic_model, mac_address, collected_at) VALUES ($1::uuid, $2, $3::macaddr, $4)`,
			deviceID, nic.Model, nic.MACAddress, now,
		); err != nil {
			slog.Warn("failed to insert NIC", "device_id", deviceID, "mac", nic.MACAddress, "error", err.Error())
		}
	}
	return nil
}

func (c *RedfishCollector) storeDisks(deviceID string, disks []DiskInfo, now time.Time) error {
	if _, err := c.db.Exec(`DELETE FROM device_disks WHERE device_id = $1::uuid`, deviceID); err != nil {
		return err
	}
	for _, disk := range disks {
		if _, err := c.db.Exec(
			`INSERT INTO device_disks (device_id, disk_model, capacity_gb, serial_number, collected_at) VALUES ($1::uuid, $2, $3, $4, $5)`,
			deviceID, disk.Model, disk.CapacityGB, disk.SerialNumber, now,
		); err != nil {
			slog.Warn("failed to insert disk", "device_id", deviceID, "model", disk.Model, "error", err.Error())
		}
	}
	return nil
}
