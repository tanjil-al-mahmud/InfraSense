package collector

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/infrasense/ipmi-collector/internal/metrics"
	"github.com/infrasense/ipmi-collector/internal/queue"
	_ "github.com/lib/pq"
)

type Device struct {
	ID           string
	Hostname     string
	IPAddress    string
	BMCIPAddress string // bmc_ip_address from devices table
	Port         int    // port from device_credentials table (default 623)
	Username     string
	Password     string
	Protocol     string
	Status       string
}

type IPMICollector struct {
	db              *sql.DB
	metricsWriter   *metrics.VictoriaMetricsWriter
	publisher       *queue.Publisher
	retryManager    *RetryManager
	devices         []Device
	devicesMutex    sync.RWMutex
	pollingInterval time.Duration
	reloadInterval  time.Duration
	maxConcurrent   int
	timeout         time.Duration
	encryptionKey   []byte
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

func NewIPMICollector(db *sql.DB, metricsWriter *metrics.VictoriaMetricsWriter, publisher *queue.Publisher, pollingInterval, reloadInterval time.Duration, maxConcurrent int, timeout time.Duration) *IPMICollector {
	ctx, cancel := context.WithCancel(context.Background())

	// Load encryption key from environment
	var encKey []byte
	if keyStr := os.Getenv("ENCRYPTION_KEY"); keyStr != "" {
		decoded, err := base64.StdEncoding.DecodeString(keyStr)
		if err != nil {
			// Try raw bytes
			decoded = []byte(keyStr)
		}
		if len(decoded) == 32 {
			encKey = decoded
		} else {
			slog.Warn("ENCRYPTION_KEY is not 32 bytes, credential decryption will be skipped",
				"event", "encryption_key_invalid",
				"key_length", len(decoded))
		}
	} else {
		slog.Warn("ENCRYPTION_KEY not set, credentials will not be decrypted",
			"event", "encryption_key_missing")
	}

	return &IPMICollector{
		db:              db,
		metricsWriter:   metricsWriter,
		publisher:       publisher,
		retryManager:    NewRetryManager(),
		devices:         make([]Device, 0),
		pollingInterval: pollingInterval,
		reloadInterval:  reloadInterval,
		maxConcurrent:   maxConcurrent,
		timeout:         timeout,
		encryptionKey:   encKey,
		ctx:             ctx,
		cancel:          cancel,
	}
}

func (c *IPMICollector) Start() error {
	// Initial device load
	if err := c.loadDevices(); err != nil {
		return fmt.Errorf("failed to load devices: %w", err)
	}

	c.devicesMutex.RLock()
	count := len(c.devices)
	c.devicesMutex.RUnlock()
	slog.Info("Loading IPMI devices from database",
		"event", "devices_loaded",
		"device_count", count)

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

// decrypt decrypts AES-256-GCM encrypted data encoded as base64
func (c *IPMICollector) decrypt(encryptedB64 string) (string, error) {
	if len(c.encryptionKey) == 0 {
		// No key — return as-is (plaintext fallback)
		return encryptedB64, nil
	}

	data, err := base64.StdEncoding.DecodeString(encryptedB64)
	if err != nil {
		// May already be plaintext
		return encryptedB64, nil
	}

	block, err := aes.NewCipher(c.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		// Not long enough to be encrypted — treat as plaintext
		return encryptedB64, nil
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Decryption failed — may be plaintext already
		return encryptedB64, nil
	}

	return string(plaintext), nil
}

func (c *IPMICollector) loadDevices() error {
	// Load devices that use IPMI protocol OR have IPMI in their device_type
	// This handles both protocol='ipmi' and device types that are IPMI-only
	query := `
		SELECT 
			d.id::text, 
			d.hostname, 
			COALESCE(d.ip_address::text, '') as ip_address,
			COALESCE(d.bmc_ip_address::text, d.ip_address::text, '') as bmc_ip,
			COALESCE(dc.port, 623) as ipmi_port,
			COALESCE(dc.username, '') as username,
			COALESCE(dc.password_encrypted, '') as password_encrypted,
			COALESCE(d.protocol, 'ipmi'),
			COALESCE(d.status, 'unknown')
		FROM devices d
		LEFT JOIN device_credentials dc ON d.id = dc.device_id 
			AND (dc.protocol = 'ipmi' OR dc.protocol IS NULL)
		WHERE (d.protocol = 'ipmi' OR d.device_type ILIKE '%ipmi%' 
		       OR d.device_type IN (
		           'dell_drac5', 'dell_idrac6', 'dell_idrac7', 'dell_idrac8_ipmi',
		           'hpe_ilo3_ipmi', 'hpe_ilo4_ipmi',
		           'lenovo_imm', 'lenovo_xcc_ipmi',
		           'supermicro_ipmi', 'supermicro_old',
		           'cisco_cimc_ipmi',
		           'huawei_ibmc_ipmi',
		           'fujitsu_irmc_ipmi',
		           'asus_asmb_ipmi',
		           'gigabyte_bmc_ipmi',
		           'ericsson_bmc_ipmi',
		           'ieit_bmc_ipmi',
		           'generic_ipmi'
		       ))
		  AND d.status != 'deleted'
	`

	rows, err := c.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query devices: %w", err)
	}
	defer rows.Close()

	devices := make([]Device, 0)
	for rows.Next() {
		var d Device
		var passwordEncrypted string
		if err := rows.Scan(
			&d.ID, &d.Hostname, &d.IPAddress, &d.BMCIPAddress,
			&d.Port, &d.Username, &passwordEncrypted, &d.Protocol, &d.Status,
		); err != nil {
			slog.Error("error scanning device row", "event", "device_scan_error", "error", err.Error())
			continue
		}

		// Decrypt password
		if passwordEncrypted != "" {
			decrypted, err := c.decrypt(passwordEncrypted)
			if err != nil {
				slog.Error("failed to decrypt password for device",
					"event", "decrypt_error",
					"device_id", d.ID,
					"hostname", d.Hostname,
					"error", err.Error())
				continue
			}
			d.Password = decrypted
		}

		// Ensure we have a BMC IP; if not, skip this device
		if d.BMCIPAddress == "" {
			slog.Warn("skipping device with no BMC IP address",
				"event", "no_bmc_ip",
				"device_id", d.ID,
				"hostname", d.Hostname)
			continue
		}

		if d.Port == 0 {
			d.Port = 623
		}

		devices = append(devices, d)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating device rows: %w", err)
	}

	c.devicesMutex.Lock()
	c.devices = devices
	c.devicesMutex.Unlock()

	slog.Info("Loading IPMI devices from database",
		"event", "devices_loaded",
		"device_count", len(devices))

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
		slog.Info("no IPMI devices to poll", "event", "poll_cycle_skip")
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

func (c *IPMICollector) updateDeviceStatus(deviceID string, status string, errorMsg string) {
	// Update devices table
	var query string
	if errorMsg != "" {
		query = `UPDATE devices SET status = $1, updated_at = NOW() WHERE id = $2::uuid`
	} else {
		query = `UPDATE devices SET status = $1, last_seen = NOW(), updated_at = NOW() WHERE id = $2::uuid`
	}
	if _, err := c.db.Exec(query, status, deviceID); err != nil {
		slog.Error("error updating device status", "event", "device_status_update_error", "device_id", deviceID, "error", err.Error())
	}

	// Update collector_status table
	statusQuery := `
		INSERT INTO collector_status (collector_name, collector_type, status, last_poll_time, last_error, updated_at)
		VALUES ($1, 'ipmi', $2, NOW(), $3, NOW())
		ON CONFLICT (collector_name)
		DO UPDATE SET
			status = $2,
			last_poll_time = NOW(),
			last_error = $3,
			updated_at = NOW()
	`
	collectorName := fmt.Sprintf("ipmi-collector-%s", deviceID)
	if _, err := c.db.Exec(statusQuery, collectorName, status, errorMsg); err != nil {
		slog.Error("error updating collector status", "event", "collector_status_update_error", "device_id", deviceID, "error", err.Error())
	}
}

func (c *IPMICollector) updateCollectorStatusSuccess(deviceID string) {
	// Update last_seen on the device
	if _, err := c.db.Exec(
		`UPDATE devices SET status = 'online', last_seen = NOW(), updated_at = NOW() WHERE id = $1::uuid`,
		deviceID,
	); err != nil {
		slog.Error("error updating device last_seen", "event", "device_last_seen_error", "device_id", deviceID, "error", err.Error())
	}

	statusQuery := `
		INSERT INTO collector_status (collector_name, collector_type, status, last_poll_time, last_success_time, last_error, updated_at)
		VALUES ($1, 'ipmi', 'healthy', NOW(), NOW(), '', NOW())
		ON CONFLICT (collector_name)
		DO UPDATE SET
			status = 'healthy',
			last_poll_time = NOW(),
			last_success_time = NOW(),
			last_error = '',
			updated_at = NOW()
	`
	collectorName := fmt.Sprintf("ipmi-collector-%s", deviceID)
	if _, err := c.db.Exec(statusQuery, collectorName); err != nil {
		slog.Error("error updating collector status", "event", "collector_status_update_error", "device_id", deviceID, "error", err.Error())
	}
}

func (c *IPMICollector) GetDeviceCount() int {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()
	return len(c.devices)
}

func (c *IPMICollector) setDeviceProtocol(deviceID string, protocol string) {
	if protocol == "" {
		return
	}
	if _, err := c.db.Exec(
		`UPDATE devices SET protocol = $1, updated_at = NOW() WHERE id = $2::uuid`,
		protocol, deviceID,
	); err != nil {
		slog.Error("failed to update device protocol",
			"event", "device_protocol_update_error",
			"device_id", deviceID,
			"protocol", protocol,
			"error", err.Error())
	}
}
