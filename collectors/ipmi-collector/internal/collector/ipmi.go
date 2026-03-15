package collector

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/infrasense/ipmi-collector/internal/metrics"
	_ "github.com/lib/pq"
)

type Device struct {
	ID        int64
	Hostname  string
	IPAddress string
	Username  string
	Password  string
	Protocol  string
	Status    string
}

type IPMICollector struct {
	db              *sql.DB
	metricsWriter   *metrics.VictoriaMetricsWriter
	retryManager    *RetryManager
	devices         []Device
	devicesMutex    sync.RWMutex
	pollingInterval time.Duration
	reloadInterval  time.Duration
	maxConcurrent   int
	timeout         time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

func NewIPMICollector(db *sql.DB, metricsWriter *metrics.VictoriaMetricsWriter, pollingInterval, reloadInterval time.Duration, maxConcurrent int, timeout time.Duration) *IPMICollector {
	ctx, cancel := context.WithCancel(context.Background())
	return &IPMICollector{
		db:              db,
		metricsWriter:   metricsWriter,
		retryManager:    NewRetryManager(),
		devices:         make([]Device, 0),
		pollingInterval: pollingInterval,
		reloadInterval:  reloadInterval,
		maxConcurrent:   maxConcurrent,
		timeout:         timeout,
		ctx:             ctx,
		cancel:          cancel,
	}
}

func (c *IPMICollector) Start() error {
	// Initial device load
	if err := c.loadDevices(); err != nil {
		return fmt.Errorf("failed to load devices: %w", err)
	}

	slog.Info("loaded ipmi devices", "event", "devices_loaded", "device_count", len(c.devices))

	// Start device reload goroutine
	c.wg.Add(1)
	go c.deviceReloadLoop()

	// Start polling goroutine
	c.wg.Add(1)
	go c.pollingLoop()

	return nil
}

func (c *IPMICollector) Stop() {
	slog.Info("stopping ipmi collector", "event", "collector_stopping")
	c.cancel()
	c.wg.Wait()
	slog.Info("ipmi collector stopped", "event", "collector_stopped")
}

func (c *IPMICollector) loadDevices() error {
	query := `
		SELECT 
			d.id, 
			d.hostname, 
			d.ip_address,
			COALESCE(dc.username, '') as username,
			COALESCE(dc.password, '') as password,
			d.protocol,
			d.status
		FROM devices d
		LEFT JOIN device_credentials dc ON d.id = dc.device_id
		WHERE d.protocol = 'ipmi' AND d.status != 'deleted'
	`

	rows, err := c.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query devices: %w", err)
	}
	defer rows.Close()

	devices := make([]Device, 0)
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Hostname, &d.IPAddress, &d.Username, &d.Password, &d.Protocol, &d.Status); err != nil {
			slog.Error("error scanning device row", "event", "device_scan_error", "error", err.Error())
			continue
		}
		devices = append(devices, d)
	}

	c.devicesMutex.Lock()
	c.devices = devices
	c.devicesMutex.Unlock()

	return nil
}

func (c *IPMICollector) deviceReloadLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.reloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.loadDevices(); err != nil {
				slog.Error("error reloading devices", "event", "device_reload_error", "error", err.Error())
			} else {
				c.devicesMutex.RLock()
				deviceCount := len(c.devices)
				c.devicesMutex.RUnlock()
				slog.Info("reloaded ipmi devices", "event", "devices_reloaded", "device_count", deviceCount)
			}
		}
	}
}

func (c *IPMICollector) pollingLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.pollingInterval)
	defer ticker.Stop()

	// Poll immediately on start
	c.pollAllDevices()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.pollAllDevices()
		}
	}
}

func (c *IPMICollector) pollAllDevices() {
	c.devicesMutex.RLock()
	devices := make([]Device, len(c.devices))
	copy(devices, c.devices)
	c.devicesMutex.RUnlock()

	if len(devices) == 0 {
		return
	}

	slog.Info("starting poll cycle", "event", "poll_cycle_start", "device_count", len(devices))

	// Create semaphore for concurrent polling
	sem := make(chan struct{}, c.maxConcurrent)
	var wg sync.WaitGroup

	for _, device := range devices {
		wg.Add(1)
		go func(d Device) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			c.PollDeviceWithRetry(d)
		}(device)
	}

	wg.Wait()
	slog.Info("poll cycle completed", "event", "poll_cycle_complete", "device_count", len(devices))
}

func (c *IPMICollector) updateDeviceStatus(deviceID int64, status string, errorMsg string) {
	query := `UPDATE devices SET status = $1, updated_at = NOW() WHERE id = $2`
	if _, err := c.db.Exec(query, status, deviceID); err != nil {
		slog.Error("error updating device status", "event", "device_status_update_error", "device_id", deviceID, "error", err.Error())
	}

	// Update collector_status table
	statusQuery := `
		INSERT INTO collector_status (collector_name, device_id, last_poll_time, last_error)
		VALUES ('ipmi-collector', $1, NOW(), $2)
		ON CONFLICT (collector_name, device_id) 
		DO UPDATE SET 
			last_poll_time = NOW(),
			last_error = $2
	`
	if _, err := c.db.Exec(statusQuery, deviceID, errorMsg); err != nil {
		slog.Error("error updating collector status", "event", "collector_status_update_error", "device_id", deviceID, "error", err.Error())
	}
}

func (c *IPMICollector) updateCollectorStatusSuccess(deviceID int64) {
	statusQuery := `
		INSERT INTO collector_status (collector_name, device_id, last_poll_time, last_success_time, last_error)
		VALUES ('ipmi-collector', $1, NOW(), NOW(), '')
		ON CONFLICT (collector_name, device_id) 
		DO UPDATE SET 
			last_poll_time = NOW(),
			last_success_time = NOW(),
			last_error = ''
	`
	if _, err := c.db.Exec(statusQuery, deviceID); err != nil {
		slog.Error("error updating collector status", "event", "collector_status_update_error", "device_id", deviceID, "error", err.Error())
	}
}

func (c *IPMICollector) GetDeviceCount() int {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()
	return len(c.devices)
}
