package collector

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/infrasense/ssh-collector/internal/metrics"
	_ "github.com/lib/pq"
	gossh "golang.org/x/crypto/ssh"
)

// Device is the legacy device type used by sshtool.go (int64 ID, plaintext credentials).
type Device struct {
	ID         int64
	Hostname   string
	IPAddress  string
	Username   string
	Password   string
	PrivateKey string
	SSHPort    int
}

// SSHDevice holds connection details for a single SSH-monitored device (UUID, encrypted creds).
type SSHDevice struct {
	ID          string // UUID
	Hostname    string
	IPAddress   string
	Username    string
	PasswordEnc []byte // AES-256-GCM encrypted password
	PrivKeyEnc  []byte // AES-256-GCM encrypted private key (optional)
	Port        int    // default 22
}

// sshRetryState tracks per-device exponential backoff state (string device IDs).
type sshRetryState struct {
	failureCount int
	nextAttempt  time.Time
}

// SSHRetryManager manages per-device retry/backoff state for string-keyed (UUID) devices.
type SSHRetryManager struct {
	mu     sync.Mutex
	states map[string]*sshRetryState
}

func newSSHRetryManager() *SSHRetryManager {
	return &SSHRetryManager{states: make(map[string]*sshRetryState)}
}

func (rm *SSHRetryManager) ShouldRetry(deviceID, _ string) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	state, exists := rm.states[deviceID]
	if !exists {
		return true
	}
	return time.Now().After(state.nextAttempt)
}

func (rm *SSHRetryManager) RecordFailure(deviceID, _ string) time.Duration {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	state, exists := rm.states[deviceID]
	if !exists {
		state = &sshRetryState{}
		rm.states[deviceID] = state
	}
	state.failureCount++
	backoff := time.Duration(math.Pow(2, float64(state.failureCount-1))) * initialBackoff
	if backoff > maxBackoffDuration {
		backoff = maxBackoffDuration
	}
	state.nextAttempt = time.Now().Add(backoff)
	return backoff
}

func (rm *SSHRetryManager) RecordSuccess(deviceID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.states, deviceID)
}

// SSHCollector polls SSH devices and writes metrics to VictoriaMetrics.
type SSHCollector struct {
	db              *sql.DB
	metricsWriter   *metrics.VictoriaMetricsWriter
	retryManager    *SSHRetryManager
	devices         []SSHDevice
	devicesMu       sync.RWMutex
	pollingInterval time.Duration
	reloadInterval  time.Duration
	maxConcurrent   int
	timeout         time.Duration
	encryptionKey   []byte
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// NewSSHCollector creates a new SSHCollector.
func NewSSHCollector(
	db *sql.DB,
	metricsWriter *metrics.VictoriaMetricsWriter,
	pollingInterval, reloadInterval time.Duration,
	maxConcurrent int,
	timeout time.Duration,
	encryptionKey string,
) *SSHCollector {
	ctx, cancel := context.WithCancel(context.Background())
	return &SSHCollector{
		db:              db,
		metricsWriter:   metricsWriter,
		retryManager:    newSSHRetryManager(),
		devices:         make([]SSHDevice, 0),
		pollingInterval: pollingInterval,
		reloadInterval:  reloadInterval,
		maxConcurrent:   maxConcurrent,
		timeout:         timeout,
		encryptionKey:   []byte(encryptionKey),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start loads devices and begins the polling and reload loops.
func (c *SSHCollector) Start() error {
	if err := c.loadDevices(); err != nil {
		return fmt.Errorf("failed to load devices: %w", err)
	}
	slog.Info("loaded ssh devices", "event", "devices_loaded", "device_count", len(c.devices))
	c.wg.Add(1)
	go c.deviceReloadLoop()
	c.wg.Add(1)
	go c.pollingLoop()
	return nil
}

// Stop gracefully shuts down the collector.
func (c *SSHCollector) Stop() {
	slog.Info("stopping ssh collector", "event", "collector_stopping")
	c.cancel()
	c.wg.Wait()
	slog.Info("ssh collector stopped", "event", "collector_stopped")
}

// GetDeviceCount returns the number of currently loaded devices.
func (c *SSHCollector) GetDeviceCount() int {
	c.devicesMu.RLock()
	defer c.devicesMu.RUnlock()
	return len(c.devices)
}

// loadDevices queries PostgreSQL for SSH devices and their credentials.
func (c *SSHCollector) loadDevices() error {
	query := `
		SELECT
			d.id::text,
			d.hostname,
			d.ip_address::text,
			COALESCE(dc.username, '') as username,
			COALESCE(dc.password_encrypted, NULL) as password_encrypted
		FROM devices d
		LEFT JOIN device_credentials dc ON d.id = dc.device_id AND dc.protocol = 'ssh'
		WHERE d.device_type = 'ssh' AND d.status != 'deleted'
	`
	rows, err := c.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query devices: %w", err)
	}
	defer rows.Close()

	devices := make([]SSHDevice, 0)
	for rows.Next() {
		var d SSHDevice
		var passwordEnc []byte
		if err := rows.Scan(&d.ID, &d.Hostname, &d.IPAddress, &d.Username, &passwordEnc); err != nil {
			slog.Error("error scanning device row", "event", "device_scan_error", "error", err.Error())
			continue
		}
		d.PasswordEnc = passwordEnc
		d.Port = 22
		devices = append(devices, d)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating device rows: %w", err)
	}
	c.devicesMu.Lock()
	c.devices = devices
	c.devicesMu.Unlock()
	return nil
}

func (c *SSHCollector) deviceReloadLoop() {
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
				c.devicesMu.RLock()
				count := len(c.devices)
				c.devicesMu.RUnlock()
				slog.Info("reloaded ssh devices", "event", "devices_reloaded", "device_count", count)
			}
		}
	}
}

func (c *SSHCollector) pollingLoop() {
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

func (c *SSHCollector) pollAllDevices() {
	c.devicesMu.RLock()
	devices := make([]SSHDevice, len(c.devices))
	copy(devices, c.devices)
	c.devicesMu.RUnlock()

	if len(devices) == 0 {
		return
	}
	slog.Info("starting poll cycle", "event", "poll_cycle_start", "device_count", len(devices))

	sem := make(chan struct{}, c.maxConcurrent)
	var wg sync.WaitGroup
	for _, device := range devices {
		wg.Add(1)
		go func(d SSHDevice) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			c.pollDeviceWithRetry(d)
		}(device)
	}
	wg.Wait()
	slog.Info("poll cycle completed", "event", "poll_cycle_complete", "device_count", len(devices))
}

func (c *SSHCollector) pollDeviceWithRetry(device SSHDevice) {
	if !c.retryManager.ShouldRetry(device.ID, device.Hostname) {
		return
	}
	timestamp := time.Now()
	slog.Info("polling ssh device",
		"event", "poll_attempt",
		"device_id", device.ID,
		"hostname", device.Hostname,
		"timestamp", timestamp.Format(time.RFC3339))

	ok := c.pollDevice(device)
	if !ok {
		slog.Error("ssh device poll failed",
			"event", "poll_failure",
			"device_id", device.ID,
			"hostname", device.Hostname,
			"timestamp", timestamp.Format(time.RFC3339))
		backoff := c.retryManager.RecordFailure(device.ID, device.Hostname)
		slog.Warn("device marked unavailable, scheduling retry",
			"event", "device_unavailable",
			"device_id", device.ID,
			"hostname", device.Hostname,
			"retry_in_seconds", backoff.Seconds())
		return
	}
	slog.Info("ssh device poll successful",
		"event", "poll_success",
		"device_id", device.ID,
		"hostname", device.Hostname,
		"timestamp", timestamp.Format(time.RFC3339))
	c.retryManager.RecordSuccess(device.ID)
	c.updateDeviceStatus(device.ID, "healthy", "")
	c.updateCollectorStatusSuccess(device.ID)
}

func (c *SSHCollector) pollDevice(device SSHDevice) bool {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
	defer cancel()

	client, err := c.connectSSH(ctx, device)
	if err != nil {
		slog.Error("ssh connection failed",
			"event", "ssh_connect_error",
			"device_id", device.ID,
			"hostname", device.Hostname,
			"timestamp", time.Now().Format(time.RFC3339),
			"error", err.Error())
		c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("connection failed: %v", err))
		return false
	}
	defer client.Close()

	now := time.Now()
	labels := map[string]string{"device_id": device.ID, "hostname": device.Hostname}

	if err := c.collectSSHCPU(ctx, client, labels, now); err != nil {
		slog.Error("failed to collect cpu metrics", "event", "metric_collect_error",
			"device_id", device.ID, "hostname", device.Hostname,
			"timestamp", now.Format(time.RFC3339), "error", err.Error())
		c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("cpu collection failed: %v", err))
		return false
	}
	if err := c.collectSSHMemory(ctx, client, labels, now); err != nil {
		slog.Error("failed to collect memory metrics", "event", "metric_collect_error",
			"device_id", device.ID, "hostname", device.Hostname,
			"timestamp", now.Format(time.RFC3339), "error", err.Error())
		c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("memory collection failed: %v", err))
		return false
	}
	if err := c.collectSSHUptime(ctx, client, labels, now); err != nil {
		slog.Error("failed to collect uptime metrics", "event", "metric_collect_error",
			"device_id", device.ID, "hostname", device.Hostname,
			"timestamp", now.Format(time.RFC3339), "error", err.Error())
		c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("uptime collection failed: %v", err))
		return false
	}
	if err := c.collectSSHNetwork(ctx, client, labels, now); err != nil {
		slog.Error("failed to collect network metrics", "event", "metric_collect_error",
			"device_id", device.ID, "hostname", device.Hostname,
			"timestamp", now.Format(time.RFC3339), "error", err.Error())
		c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("network collection failed: %v", err))
		return false
	}
	if err := c.collectSSHDisk(ctx, client, labels, now); err != nil {
		slog.Error("failed to collect disk metrics", "event", "metric_collect_error",
			"device_id", device.ID, "hostname", device.Hostname,
			"timestamp", now.Format(time.RFC3339), "error", err.Error())
		c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("disk collection failed: %v", err))
		return false
	}
	return true
}

func (c *SSHCollector) connectSSH(ctx context.Context, device SSHDevice) (*gossh.Client, error) {
	var authMethods []gossh.AuthMethod

	if len(device.PrivKeyEnc) > 0 {
		if privKeyBytes, err := decryptCredential(c.encryptionKey, device.PrivKeyEnc); err == nil {
			if signer, err := gossh.ParsePrivateKey(privKeyBytes); err == nil {
				authMethods = append(authMethods, gossh.PublicKeys(signer))
			}
		}
	}
	if len(device.PasswordEnc) > 0 {
		if password, err := decryptCredential(c.encryptionKey, device.PasswordEnc); err == nil {
			authMethods = append(authMethods, gossh.Password(string(password)))
		}
	}
	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no valid authentication methods available")
	}

	port := device.Port
	if port == 0 {
		port = 22
	}
	sshConfig := &gossh.ClientConfig{
		User:            device.Username,
		Auth:            authMethods,
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), //nolint:gosec
		Timeout:         c.timeout,
	}
	addr := fmt.Sprintf("%s:%d", device.IPAddress, port)

	type dialResult struct {
		client *gossh.Client
		err    error
	}
	ch := make(chan dialResult, 1)
	go func() {
		client, err := gossh.Dial("tcp", addr, sshConfig)
		ch <- dialResult{client, err}
	}()
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("connection timed out: %w", ctx.Err())
	case result := <-ch:
		return result.client, result.err
	}
}

// runSSHCommand executes a command on the SSH client with context awareness.
func runSSHCommand(ctx context.Context, client *gossh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var buf bytes.Buffer
	session.Stdout = &buf

	done := make(chan error, 1)
	go func() { done <- session.Run(cmd) }()

	select {
	case <-ctx.Done():
		session.Signal(gossh.SIGKILL) //nolint:errcheck
		return "", fmt.Errorf("command timed out: %w", ctx.Err())
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("command %q failed: %w", cmd, err)
		}
	}
	return buf.String(), nil
}

func (c *SSHCollector) collectSSHCPU(ctx context.Context, client *gossh.Client, labels map[string]string, ts time.Time) error {
	out, err := runSSHCommand(ctx, client, "cat /proc/stat")
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return fmt.Errorf("unexpected /proc/stat format")
		}
		var vals [10]float64
		for i := 1; i < len(fields) && i <= 10; i++ {
			v, _ := strconv.ParseFloat(fields[i], 64)
			vals[i-1] = v
		}
		idle := vals[3] + vals[4]
		total := 0.0
		for _, v := range vals {
			total += v
		}
		var cpuUsage float64
		if total > 0 {
			cpuUsage = (1 - idle/total) * 100
		}
		return c.metricsWriter.WriteMetric("infrasense_ssh_cpu_usage_percent", cpuUsage, labels, ts)
	}
	return fmt.Errorf("cpu line not found in /proc/stat")
}

func (c *SSHCollector) collectSSHMemory(ctx context.Context, client *gossh.Client, labels map[string]string, ts time.Time) error {
	out, err := runSSHCommand(ctx, client, "cat /proc/meminfo")
	if err != nil {
		return err
	}
	var memTotal, memAvailable float64
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseFloat(fields[1], 64)
		switch fields[0] {
		case "MemTotal:":
			memTotal = val * 1024
		case "MemAvailable:":
			memAvailable = val * 1024
		}
	}
	if err := c.metricsWriter.WriteMetric("infrasense_ssh_memory_total_bytes", memTotal, labels, ts); err != nil {
		return err
	}
	return c.metricsWriter.WriteMetric("infrasense_ssh_memory_available_bytes", memAvailable, labels, ts)
}

func (c *SSHCollector) collectSSHUptime(ctx context.Context, client *gossh.Client, labels map[string]string, ts time.Time) error {
	out, err := runSSHCommand(ctx, client, "cat /proc/uptime")
	if err != nil {
		return err
	}
	fields := strings.Fields(strings.TrimSpace(out))
	if len(fields) < 1 {
		return fmt.Errorf("unexpected /proc/uptime format")
	}
	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return fmt.Errorf("failed to parse uptime: %w", err)
	}
	return c.metricsWriter.WriteMetric("infrasense_ssh_uptime_seconds", uptime, labels, ts)
}

func (c *SSHCollector) collectSSHNetwork(ctx context.Context, client *gossh.Client, labels map[string]string, ts time.Time) error {
	out, err := runSSHCommand(ctx, client, "cat /proc/net/dev")
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		iface := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}
		rxBytes, _ := strconv.ParseFloat(fields[0], 64)
		txBytes, _ := strconv.ParseFloat(fields[8], 64)
		ifaceLabels := map[string]string{
			"device_id": labels["device_id"],
			"hostname":  labels["hostname"],
			"interface": iface,
		}
		if err := c.metricsWriter.WriteMetric("infrasense_ssh_network_rx_bytes", rxBytes, ifaceLabels, ts); err != nil {
			return err
		}
		if err := c.metricsWriter.WriteMetric("infrasense_ssh_network_tx_bytes", txBytes, ifaceLabels, ts); err != nil {
			return err
		}
	}
	return nil
}

func (c *SSHCollector) collectSSHDisk(ctx context.Context, client *gossh.Client, labels map[string]string, ts time.Time) error {
	out, err := runSSHCommand(ctx, client, "df -k")
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(strings.NewReader(out))
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		totalKB, _ := strconv.ParseFloat(fields[1], 64)
		usedKB, _ := strconv.ParseFloat(fields[2], 64)
		mountpoint := fields[5]
		mpLabels := map[string]string{
			"device_id":  labels["device_id"],
			"hostname":   labels["hostname"],
			"mountpoint": mountpoint,
		}
		if err := c.metricsWriter.WriteMetric("infrasense_ssh_disk_total_bytes", totalKB*1024, mpLabels, ts); err != nil {
			return err
		}
		if err := c.metricsWriter.WriteMetric("infrasense_ssh_disk_used_bytes", usedKB*1024, mpLabels, ts); err != nil {
			return err
		}
	}
	return nil
}

// decryptCredential decrypts AES-256-GCM ciphertext. The first 12 bytes are the nonce.
func decryptCredential(key, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 12 {
		return nil, fmt.Errorf("ciphertext too short")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	plaintext, err := gcm.Open(nil, ciphertext[:12], ciphertext[12:], nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
}

// updateDeviceStatus updates device status in PostgreSQL and the collector-level status.
func (c *SSHCollector) updateDeviceStatus(deviceID, status, errorMsg string) {
	if _, err := c.db.Exec(
		`UPDATE devices SET status = $1, updated_at = NOW() WHERE id = $2::uuid`,
		status, deviceID,
	); err != nil {
		slog.Error("error updating device status",
			"event", "device_status_update_error",
			"device_id", deviceID, "error", err.Error())
	}
	if _, err := c.db.Exec(`
		INSERT INTO collector_status (collector_name, collector_type, status, last_poll_time, last_error, updated_at)
		VALUES ('ssh-collector', 'ssh', $1, NOW(), $2, NOW())
		ON CONFLICT (collector_name) DO UPDATE SET
			status = EXCLUDED.status,
			last_poll_time = NOW(),
			last_error = EXCLUDED.last_error,
			updated_at = NOW()
	`, status, errorMsg); err != nil {
		slog.Error("error updating collector status",
			"event", "collector_status_update_error",
			"device_id", deviceID, "error", err.Error())
	}
}

// updateCollectorStatusSuccess records a successful poll in the collector_status table.
func (c *SSHCollector) updateCollectorStatusSuccess(deviceID string) {
	if _, err := c.db.Exec(`
		INSERT INTO collector_status (collector_name, collector_type, status, last_poll_time, last_success_time, last_error, updated_at)
		VALUES ('ssh-collector', 'ssh', 'healthy', NOW(), NOW(), '', NOW())
		ON CONFLICT (collector_name) DO UPDATE SET
			status = 'healthy',
			last_poll_time = NOW(),
			last_success_time = NOW(),
			last_error = '',
			updated_at = NOW()
	`); err != nil {
		slog.Error("error updating collector status on success",
			"event", "collector_status_update_error",
			"device_id", deviceID, "error", err.Error())
	}
}
