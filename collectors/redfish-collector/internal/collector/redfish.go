package collector

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// Device holds all data needed to poll a single BMC.
type Device struct {
	ID        string // UUID from PostgreSQL
	Hostname  string
	IPAddress string // host:port or just host
	Username  string
	Password  string // decrypted at load time
	Protocol  string
	Status    string
}

type Metric struct {
	Name      string
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
}

type RedfishCollector struct {
	db              *sql.DB
	metricsWriter   MetricsWriter
	retryManager    *RetryManager
	encryptionKey   []byte // 32-byte AES-256 key for credential decryption
	devices         []Device
	devicesMutex    sync.RWMutex
	pollingInterval time.Duration
	reloadInterval  time.Duration
	maxConcurrent   int
	timeout         time.Duration
	httpClient      *http.Client
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

type MetricsWriter interface {
	WriteMetric(name string, value float64, labels map[string]string, timestamp time.Time) error
}

func NewRedfishCollector(db *sql.DB, metricsWriter MetricsWriter, encryptionKey string, pollingInterval, reloadInterval time.Duration, maxConcurrent int, timeout time.Duration) *RedfishCollector {
	ctx, cancel := context.WithCancel(context.Background())

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // BMCs use self-signed certs
			},
		},
	}

	return &RedfishCollector{
		db:              db,
		metricsWriter:   metricsWriter,
		encryptionKey:   []byte(encryptionKey),
		retryManager:    NewRetryManager(),
		devices:         make([]Device, 0),
		pollingInterval: pollingInterval,
		reloadInterval:  reloadInterval,
		maxConcurrent:   maxConcurrent,
		timeout:         timeout,
		httpClient:      httpClient,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// decryptCredential decrypts AES-256-GCM encrypted credential bytes.
func (c *RedfishCollector) decryptCredential(ciphertext []byte) (string, error) {
	if len(ciphertext) == 0 {
		return "", nil
	}
	block, err := aes.NewCipher(c.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("cipher init: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm init: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plain), nil
}

func (c *RedfishCollector) Start() error {
	if err := c.loadDevices(); err != nil {
		return fmt.Errorf("failed to load devices: %w", err)
	}
	slog.Info("loaded redfish devices", "event", "devices_loaded", "device_count", len(c.devices))

	c.wg.Add(1)
	go c.deviceReloadLoop()

	c.wg.Add(1)
	go c.pollingLoop()

	return nil
}

func (c *RedfishCollector) Stop() {
	slog.Info("stopping redfish collector", "event", "collector_stopping")
	c.cancel()
	c.wg.Wait()
	slog.Info("redfish collector stopped", "event", "collector_stopped")
}

// loadDevices queries PostgreSQL for all devices that have redfish credentials.
// Credentials are stored AES-256-GCM encrypted and decrypted here.
func (c *RedfishCollector) loadDevices() error {
	query := `
		SELECT
			d.id::text,
			d.hostname,
			COALESCE(host(d.bmc_ip_address), host(d.ip_address)) AS ip_address,
			COALESCE(dc.username, '') AS username,
			COALESCE(dc.password_encrypted, ''::bytea) AS password_encrypted,
			dc.protocol,
			d.status
		FROM devices d
		INNER JOIN device_credentials dc ON d.id = dc.device_id AND dc.protocol = 'redfish'
		WHERE d.status != 'deleted'
	`

	rows, err := c.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query devices: %w", err)
	}
	defer rows.Close()

	devices := make([]Device, 0)
	for rows.Next() {
		var d Device
		var passwordEncrypted []byte
		if err := rows.Scan(&d.ID, &d.Hostname, &d.IPAddress, &d.Username, &passwordEncrypted, &d.Protocol, &d.Status); err != nil {
			slog.Error("error scanning device row", "event", "device_scan_error", "error", err.Error())
			continue
		}

		// Decrypt password
		if len(passwordEncrypted) > 0 {
			plain, err := c.decryptCredential(passwordEncrypted)
			if err != nil {
				slog.Error("failed to decrypt credential",
					"event", "credential_decrypt_error",
					"device_id", d.ID,
					"hostname", d.Hostname,
					"error", err.Error())
				continue // skip device — can't authenticate without password
			}
			d.Password = plain
		}

		slog.Info("loaded device",
			"event", "device_loaded",
			"device_id", d.ID,
			"hostname", d.Hostname,
			"ip_address", d.IPAddress)

		devices = append(devices, d)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("row iteration error: %w", err)
	}

	c.devicesMutex.Lock()
	c.devices = devices
	c.devicesMutex.Unlock()

	slog.Info("device load complete", "event", "devices_loaded", "count", len(devices))
	return nil
}

func (c *RedfishCollector) deviceReloadLoop() {
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
				n := len(c.devices)
				c.devicesMutex.RUnlock()
				slog.Info("reloaded redfish devices", "event", "devices_reloaded", "device_count", n)
			}
		}
	}
}

func (c *RedfishCollector) pollingLoop() {
	defer c.wg.Done()
	ticker := time.NewTicker(c.pollingInterval)
	defer ticker.Stop()
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

func (c *RedfishCollector) pollAllDevices() {
	c.devicesMutex.RLock()
	devices := make([]Device, len(c.devices))
	copy(devices, c.devices)
	c.devicesMutex.RUnlock()

	if len(devices) == 0 {
		slog.Info("no redfish devices to poll", "event", "poll_cycle_skip")
		return
	}

	slog.Info("starting poll cycle", "event", "poll_cycle_start", "device_count", len(devices))

	sem := make(chan struct{}, c.maxConcurrent)
	var wg sync.WaitGroup
	for _, device := range devices {
		wg.Add(1)
		go func(d Device) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			c.PollDeviceWithRetry(d)
		}(device)
	}
	wg.Wait()
	slog.Info("poll cycle completed", "event", "poll_cycle_complete", "device_count", len(devices))
}

func (c *RedfishCollector) PollDeviceWithRetry(device Device) {
	if !c.retryManager.ShouldRetry(device.ID, device.Hostname) {
		return
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
	defer cancel()

	timestamp := time.Now()
	slog.Info("polling device",
		"event", "poll_attempt",
		"device_id", device.ID,
		"hostname", device.Hostname,
		"ip_address", device.IPAddress)

	metrics, err := c.collectRedfishData(ctx, device)

	if len(metrics) > 0 {
		slog.Info("collected metrics from device",
			"event", "metrics_collected",
			"device_id", device.ID,
			"hostname", device.Hostname,
			"metrics_count", len(metrics))
		for _, m := range metrics {
			if writeErr := c.metricsWriter.WriteMetric(m.Name, m.Value, m.Labels, m.Timestamp); writeErr != nil {
				slog.Error("error writing metric",
					"event", "metric_write_error",
					"device_id", device.ID,
					"metric", m.Name,
					"error", writeErr.Error())
			}
		}
	}

	if err != nil {
		slog.Error("error collecting redfish data",
			"event", "poll_error",
			"device_id", device.ID,
			"hostname", device.Hostname,
			"error", err.Error())

		if len(metrics) > 0 {
			c.updateCollectorStatus(device.ID, "degraded", fmt.Sprintf("Partial failure: %v", err), false)
		} else {
			backoff := c.retryManager.RecordFailure(device.ID, device.Hostname)
			c.updateCollectorStatus(device.ID, "unavailable", fmt.Sprintf("Connection failed: %v (retry in %v)", err, backoff), false)
		}
		return
	}

	slog.Info("device poll successful",
		"event", "poll_success",
		"device_id", device.ID,
		"hostname", device.Hostname,
		"metrics_count", len(metrics),
		"duration_ms", time.Since(timestamp).Milliseconds())

	c.retryManager.RecordSuccess(device.ID)
	c.updateCollectorStatus(device.ID, "healthy", "", true)

	// Collect inventory asynchronously
	go func() {
		invCtx, invCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer invCancel()
		inventory, err := c.collectInventory(invCtx, device)
		if err != nil {
			slog.Warn("failed to collect inventory",
				"event", "inventory_error",
				"device_id", device.ID,
				"error", err.Error())
			return
		}
		if err := c.storeInventory(device, inventory); err != nil {
			slog.Warn("failed to store inventory",
				"event", "inventory_store_error",
				"device_id", device.ID,
				"error", err.Error())
		}
	}()
}

// updateCollectorStatus updates both the devices table and collector_status table.
// The collector_status table tracks per-collector (not per-device) status.
func (c *RedfishCollector) updateCollectorStatus(deviceID, status, errMsg string, success bool) {
	// Update device status
	_, err := c.db.Exec(
		`UPDATE devices SET status = $1, last_seen = NOW(), updated_at = NOW() WHERE id = $2::uuid`,
		status, deviceID,
	)
	if err != nil {
		slog.Error("error updating device status", "device_id", deviceID, "error", err.Error())
	}

	// Upsert collector_status — one row per collector_name (not per device)
	// We store the last error across all devices in the collector name row.
	collectorName := fmt.Sprintf("redfish-collector-%s", deviceID)
	if success {
		_, err = c.db.Exec(`
			INSERT INTO collector_status (collector_name, collector_type, status, last_poll_time, last_success_time, last_error, updated_at)
			VALUES ($1, 'redfish', 'healthy', NOW(), NOW(), '', NOW())
			ON CONFLICT (collector_name) DO UPDATE SET
				status = 'healthy',
				last_poll_time = NOW(),
				last_success_time = NOW(),
				last_error = '',
				updated_at = NOW()
		`, collectorName)
	} else {
		_, err = c.db.Exec(`
			INSERT INTO collector_status (collector_name, collector_type, status, last_poll_time, last_error, updated_at)
			VALUES ($1, 'redfish', $2, NOW(), $3, NOW())
			ON CONFLICT (collector_name) DO UPDATE SET
				status = $2,
				last_poll_time = NOW(),
				last_error = $3,
				updated_at = NOW()
		`, collectorName, status, errMsg)
	}
	if err != nil {
		slog.Error("error updating collector_status", "device_id", deviceID, "error", err.Error())
	}
}

func (c *RedfishCollector) GetDeviceCount() int {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()
	return len(c.devices)
}

// redfishRequest makes an authenticated GET request to the Redfish API.
func (c *RedfishCollector) redfishRequest(ctx context.Context, device Device, path string) (map[string]any, error) {
	url := fmt.Sprintf("https://%s%s", device.IPAddress, path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.SetBasicAuth(device.Username, device.Password)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, url, truncate(string(body), 200))
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode JSON from %s: %w", url, err)
	}
	return result, nil
}

// collectRedfishData collects all metrics with partial-failure support.
func (c *RedfishCollector) collectRedfishData(ctx context.Context, device Device) ([]Metric, error) {
	var allMetrics []Metric
	var errs []string

	// --- Chassis list → Thermal + Power ---
	chassisIDs, err := c.listMembers(ctx, device, "/redfish/v1/Chassis")
	if err != nil {
		// Fall back to well-known Dell iDRAC9 path
		chassisIDs = []string{"/redfish/v1/Chassis/System.Embedded.1"}
		slog.Warn("chassis list failed, using default path",
			"device_id", device.ID, "error", err.Error())
	}

	for _, chassisID := range chassisIDs {
		thermalMetrics, err := c.collectThermalData(ctx, device, chassisID+"/Thermal")
		if err != nil {
			errs = append(errs, fmt.Sprintf("thermal(%s): %v", chassisID, err))
		} else {
			allMetrics = append(allMetrics, thermalMetrics...)
		}

		powerMetrics, err := c.collectPowerData(ctx, device, chassisID+"/Power")
		if err != nil {
			errs = append(errs, fmt.Sprintf("power(%s): %v", chassisID, err))
		} else {
			allMetrics = append(allMetrics, powerMetrics...)
		}
	}

	// --- Systems list → health + storage ---
	systemIDs, err := c.listMembers(ctx, device, "/redfish/v1/Systems")
	if err != nil {
		systemIDs = []string{"/redfish/v1/Systems/System.Embedded.1"}
		slog.Warn("systems list failed, using default path",
			"device_id", device.ID, "error", err.Error())
	}

	for _, systemID := range systemIDs {
		sysMetrics, err := c.collectSystemHealth(ctx, device, systemID)
		if err != nil {
			errs = append(errs, fmt.Sprintf("system_health(%s): %v", systemID, err))
		} else {
			allMetrics = append(allMetrics, sysMetrics...)
		}

		storageMetrics, err := c.collectStorageData(ctx, device, systemID+"/Storage")
		if err != nil {
			errs = append(errs, fmt.Sprintf("storage(%s): %v", systemID, err))
		} else {
			allMetrics = append(allMetrics, storageMetrics...)
		}
	}

	if len(allMetrics) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("complete failure: %s", strings.Join(errs, "; "))
	}
	if len(errs) > 0 {
		return allMetrics, fmt.Errorf("partial failure: %s", strings.Join(errs, "; "))
	}
	return allMetrics, nil
}

// listMembers fetches a Redfish collection and returns the @odata.id of each member.
func (c *RedfishCollector) listMembers(ctx context.Context, device Device, path string) ([]string, error) {
	data, err := c.redfishRequest(ctx, device, path)
	if err != nil {
		return nil, err
	}
	members, ok := data["Members"].([]any)
	if !ok || len(members) == 0 {
		return nil, fmt.Errorf("no members in %s", path)
	}
	ids := make([]string, 0, len(members))
	for _, m := range members {
		if mm, ok := m.(map[string]any); ok {
			if id, ok := mm["@odata.id"].(string); ok {
				ids = append(ids, id)
			}
		}
	}
	return ids, nil
}

// collectThermalData collects temperature and fan metrics from a Thermal endpoint.
func (c *RedfishCollector) collectThermalData(ctx context.Context, device Device, path string) ([]Metric, error) {
	data, err := c.redfishRequest(ctx, device, path)
	if err != nil {
		return nil, err
	}

	var metrics []Metric
	now := time.Now()

	if temperatures, ok := data["Temperatures"].([]any); ok {
		for _, t := range temperatures {
			tm, ok := t.(map[string]any)
			if !ok {
				continue
			}
			name, _ := tm["Name"].(string)
			reading, ok := tm["ReadingCelsius"].(float64)
			if !ok {
				continue
			}
			metrics = append(metrics, Metric{
				Name:  "infrasense_redfish_temperature_celsius",
				Value: reading,
				Labels: map[string]string{
					"device_id":   device.ID,
					"sensor_name": name,
					"sensor_type": classifyTempSensor(name),
				},
				Timestamp: now,
			})
		}
	}

	if fans, ok := data["Fans"].([]any); ok {
		for _, f := range fans {
			fm, ok := f.(map[string]any)
			if !ok {
				continue
			}
			name, _ := fm["Name"].(string)
			reading, ok := fm["Reading"].(float64)
			if !ok {
				continue
			}
			metrics = append(metrics, Metric{
				Name:  "infrasense_redfish_fan_speed_rpm",
				Value: reading,
				Labels: map[string]string{
					"device_id": device.ID,
					"fan_name":  name,
				},
				Timestamp: now,
			})
		}
	}

	return metrics, nil
}

// collectPowerData collects PSU status and power consumption metrics.
func (c *RedfishCollector) collectPowerData(ctx context.Context, device Device, path string) ([]Metric, error) {
	data, err := c.redfishRequest(ctx, device, path)
	if err != nil {
		return nil, err
	}

	var metrics []Metric
	now := time.Now()

	if psus, ok := data["PowerSupplies"].([]any); ok {
		for _, p := range psus {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			name, _ := pm["Name"].(string)
			health := ""
			if status, ok := pm["Status"].(map[string]any); ok {
				health, _ = status["Health"].(string)
			}
			statusVal := 0.0
			if health == "OK" {
				statusVal = 1.0
			}
			metrics = append(metrics, Metric{
				Name:  "infrasense_redfish_psu_status",
				Value: statusVal,
				Labels: map[string]string{
					"device_id": device.ID,
					"psu_name":  name,
				},
				Timestamp: now,
			})
			if watts, ok := pm["PowerOutputWatts"].(float64); ok {
				metrics = append(metrics, Metric{
					Name:  "infrasense_redfish_psu_power_watts",
					Value: watts,
					Labels: map[string]string{
						"device_id": device.ID,
						"psu_name":  name,
					},
					Timestamp: now,
				})
			}
		}
	}

	return metrics, nil
}

// collectSystemHealth collects overall system, CPU, and memory health from a System endpoint.
func (c *RedfishCollector) collectSystemHealth(ctx context.Context, device Device, path string) ([]Metric, error) {
	data, err := c.redfishRequest(ctx, device, path)
	if err != nil {
		return nil, err
	}

	var metrics []Metric
	now := time.Now()

	healthToFloat := func(h string) float64 {
		if h == "OK" {
			return 1.0
		}
		return 0.0
	}

	// Overall system health
	if status, ok := data["Status"].(map[string]any); ok {
		health, _ := status["Health"].(string)
		metrics = append(metrics, Metric{
			Name:      "infrasense_redfish_system_health",
			Value:     healthToFloat(health),
			Labels:    map[string]string{"device_id": device.ID},
			Timestamp: now,
		})
	}

	// CPU health via ProcessorSummary
	if ps, ok := data["ProcessorSummary"].(map[string]any); ok {
		if status, ok := ps["Status"].(map[string]any); ok {
			health, _ := status["Health"].(string)
			metrics = append(metrics, Metric{
				Name:      "infrasense_redfish_cpu_health",
				Value:     healthToFloat(health),
				Labels:    map[string]string{"device_id": device.ID},
				Timestamp: now,
			})
		}
	}

	// Memory health via MemorySummary
	if ms, ok := data["MemorySummary"].(map[string]any); ok {
		if status, ok := ms["Status"].(map[string]any); ok {
			health, _ := status["Health"].(string)
			metrics = append(metrics, Metric{
				Name:      "infrasense_redfish_memory_health",
				Value:     healthToFloat(health),
				Labels:    map[string]string{"device_id": device.ID},
				Timestamp: now,
			})
		}
	}

	return metrics, nil
}

// collectStorageData collects RAID controller and disk health metrics.
func (c *RedfishCollector) collectStorageData(ctx context.Context, device Device, path string) ([]Metric, error) {
	storageData, err := c.redfishRequest(ctx, device, path)
	if err != nil {
		return nil, err
	}

	var metrics []Metric
	now := time.Now()

	members, _ := storageData["Members"].([]any)
	for _, m := range members {
		mm, ok := m.(map[string]any)
		if !ok {
			continue
		}
		odataID, _ := mm["@odata.id"].(string)
		if odataID == "" {
			continue
		}

		ctrlData, err := c.redfishRequest(ctx, device, odataID)
		if err != nil {
			slog.Warn("failed to get storage controller", "device_id", device.ID, "path", odataID, "error", err.Error())
			continue
		}

		ctrlName, _ := ctrlData["Name"].(string)
		if status, ok := ctrlData["Status"].(map[string]any); ok {
			health, _ := status["Health"].(string)
			val := 0.0
			if health == "OK" {
				val = 1.0
			}
			metrics = append(metrics, Metric{
				Name:  "infrasense_redfish_raid_status",
				Value: val,
				Labels: map[string]string{
					"device_id":       device.ID,
					"controller_name": ctrlName,
				},
				Timestamp: now,
			})
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
				slog.Warn("failed to get drive", "device_id", device.ID, "path", driveID, "error", err.Error())
				continue
			}
			diskName, _ := driveData["Name"].(string)
			if status, ok := driveData["Status"].(map[string]any); ok {
				health, _ := status["Health"].(string)
				val := 0.0
				if health == "OK" {
					val = 1.0
				}
				metrics = append(metrics, Metric{
					Name:  "infrasense_redfish_disk_health",
					Value: val,
					Labels: map[string]string{
						"device_id": device.ID,
						"disk_name": diskName,
					},
					Timestamp: now,
				})
			}
		}
	}

	return metrics, nil
}

// classifyTempSensor returns a sensor_type string based on sensor name.
func classifyTempSensor(name string) string {
	upper := strings.ToUpper(name)
	switch {
	case strings.Contains(upper, "CPU"):
		return "cpu"
	case strings.Contains(upper, "INLET"):
		return "inlet"
	case strings.Contains(upper, "EXHAUST"):
		return "exhaust"
	case strings.Contains(upper, "SYSTEM"):
		return "system"
	default:
		return "other"
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// RetryManager handles exponential backoff for failed devices.
type RetryManager struct {
	states map[string]*RetryState
	mutex  sync.RWMutex
}

type RetryState struct {
	deviceID     string
	hostname     string
	failureCount int
	nextAttempt  time.Time
}

func NewRetryManager() *RetryManager {
	return &RetryManager{states: make(map[string]*RetryState)}
}

func (rm *RetryManager) ShouldRetry(deviceID, hostname string) bool {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	state, exists := rm.states[deviceID]
	if !exists {
		return true
	}
	return time.Now().After(state.nextAttempt)
}

func (rm *RetryManager) RecordFailure(deviceID, hostname string) time.Duration {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	state, exists := rm.states[deviceID]
	if !exists {
		state = &RetryState{deviceID: deviceID, hostname: hostname}
		rm.states[deviceID] = state
	}
	state.failureCount++
	backoff := time.Duration(math.Pow(2, float64(state.failureCount-1))) * time.Second
	if backoff > 10*time.Minute {
		backoff = 10 * time.Minute
	}
	state.nextAttempt = time.Now().Add(backoff)
	slog.Warn("device poll failed, scheduling retry",
		"event", "device_poll_failure",
		"device_id", deviceID,
		"hostname", hostname,
		"failure_count", state.failureCount,
		"retry_in", backoff.String())
	return backoff
}

func (rm *RetryManager) RecordSuccess(deviceID string) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	delete(rm.states, deviceID)
}
