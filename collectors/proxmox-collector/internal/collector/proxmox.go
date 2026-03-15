package collector

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// AuthenticationError represents an authentication failure with the Proxmox API
type AuthenticationError struct {
	Err error
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("authentication failed: %v", e.Err)
}

func (e *AuthenticationError) Unwrap() error {
	return e.Err
}

type Device struct {
	ID        int64
	Hostname  string
	IPAddress string
	APIToken  string // API token from device_credentials
	Status    string
}

type ProxmoxCollector struct {
	db                 *sql.DB
	metricsWriter      MetricsWriter
	devices            []Device
	devicesMutex       sync.RWMutex
	pollingInterval    time.Duration
	reloadInterval     time.Duration
	maxConcurrent      int
	timeout            time.Duration
	ctx                context.Context
	cancel             context.CancelFunc
	wg                 sync.WaitGroup
	retryManager       *RetryManager
	pollProxmoxMetrics func(device Device, timestamp time.Time) error
}

type MetricsWriter interface {
	WriteMetric(name string, value float64, labels map[string]string, timestamp time.Time) error
}

func NewProxmoxCollector(db *sql.DB, metricsWriter MetricsWriter, pollingInterval, reloadInterval time.Duration, maxConcurrent int, timeout time.Duration) *ProxmoxCollector {
	ctx, cancel := context.WithCancel(context.Background())
	c := &ProxmoxCollector{
		db:              db,
		metricsWriter:   metricsWriter,
		devices:         make([]Device, 0),
		pollingInterval: pollingInterval,
		reloadInterval:  reloadInterval,
		maxConcurrent:   maxConcurrent,
		timeout:         timeout,
		ctx:             ctx,
		cancel:          cancel,
		retryManager:    NewRetryManager(),
	}
	c.pollProxmoxMetrics = c.defaultPollProxmoxMetrics
	return c
}

func (c *ProxmoxCollector) Start() error {
	// Initial device load
	if err := c.loadDevices(); err != nil {
		return fmt.Errorf("failed to load devices: %w", err)
	}

	log.Printf("Loaded %d Proxmox devices", len(c.devices))

	// Start device reload goroutine
	c.wg.Add(1)
	go c.deviceReloadLoop()

	// Start polling goroutine
	c.wg.Add(1)
	go c.pollingLoop()

	return nil
}

func (c *ProxmoxCollector) Stop() {
	log.Println("Stopping Proxmox collector...")
	c.cancel()
	c.wg.Wait()
	log.Println("Proxmox collector stopped")
}

func (c *ProxmoxCollector) loadDevices() error {
	query := `
		SELECT 
			d.id, 
			d.hostname, 
			d.ip_address,
			COALESCE(dc.password_encrypted, '') as api_token,
			d.status
		FROM devices d
		LEFT JOIN device_credentials dc ON d.id = dc.device_id AND dc.protocol = 'proxmox'
		WHERE d.device_type = 'proxmox' AND d.status != 'deleted'
	`

	rows, err := c.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query devices: %w", err)
	}
	defer rows.Close()

	devices := make([]Device, 0)
	for rows.Next() {
		var d Device
		var apiTokenBytes []byte
		if err := rows.Scan(&d.ID, &d.Hostname, &d.IPAddress, &apiTokenBytes, &d.Status); err != nil {
			log.Printf("Error scanning device row: %v", err)
			continue
		}
		// Decrypt API token
		d.APIToken = string(apiTokenBytes)
		devices = append(devices, d)
	}

	c.devicesMutex.Lock()
	c.devices = devices
	c.devicesMutex.Unlock()

	return nil
}

func (c *ProxmoxCollector) deviceReloadLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.reloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.loadDevices(); err != nil {
				log.Printf("Error reloading devices: %v", err)
			} else {
				c.devicesMutex.RLock()
				deviceCount := len(c.devices)
				c.devicesMutex.RUnlock()
				log.Printf("Reloaded %d Proxmox devices", deviceCount)
			}
		}
	}
}

func (c *ProxmoxCollector) pollingLoop() {
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

func (c *ProxmoxCollector) pollAllDevices() {
	c.devicesMutex.RLock()
	devices := make([]Device, len(c.devices))
	copy(devices, c.devices)
	c.devicesMutex.RUnlock()

	if len(devices) == 0 {
		return
	}

	log.Printf("Starting poll cycle for %d devices", len(devices))

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
	log.Printf("Poll cycle completed for %d devices", len(devices))
}

func (c *ProxmoxCollector) PollDeviceWithRetry(device Device) {
	// Check if we should retry this device
	if !c.retryManager.ShouldRetry(device.ID, device.Hostname) {
		return
	}

	log.Printf("Polling device %s (%s)", device.Hostname, device.IPAddress)

	timestamp := time.Now()

	// Poll Proxmox metrics
	err := c.pollProxmoxMetrics(device, timestamp)
	if err != nil {
		// Check if this is an authentication error
		var authErr *AuthenticationError
		isAuthError := false
		if e, ok := err.(*AuthenticationError); ok {
			authErr = e
			isAuthError = true
		}

		if isAuthError {
			// Log authentication error specifically
			log.Printf("Authentication error for device %s (%s): %v", device.Hostname, device.IPAddress, authErr.Err)

			// Record failure and get backoff duration
			backoff := c.retryManager.RecordFailure(device.ID, device.Hostname)

			// Update device status to auth_failed
			c.updateDeviceStatus(device.ID, "auth_failed", fmt.Sprintf("Authentication failed: %v (retry in %v)", authErr.Err, backoff))
		} else {
			// Log general polling error
			log.Printf("Failed to poll device %s (%s): %v", device.Hostname, device.IPAddress, err)

			// Record failure and get backoff duration
			backoff := c.retryManager.RecordFailure(device.ID, device.Hostname)

			// Update device status to unavailable
			c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("Polling failed: %v (retry in %v)", err, backoff))
		}

		return
	}

	// Record success (resets failure count)
	c.retryManager.RecordSuccess(device.ID)

	// Update device status to healthy
	c.updateDeviceStatus(device.ID, "healthy", "")

	// Update collector status with success
	c.updateCollectorStatusSuccess(device.ID)

	log.Printf("Successfully polled device %s (%s)", device.Hostname, device.IPAddress)
}

func (c *ProxmoxCollector) defaultPollProxmoxMetrics(device Device, timestamp time.Time) error {
	// Create HTTP client with TLS certificate validation and timeout
	client, err := c.createHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Authenticate and get ticket
	ticket, csrfToken, err := c.authenticate(client, device)
	if err != nil {
		return &AuthenticationError{Err: err}
	}

	// Retrieve node status
	nodeStats, err := c.getNodeStatus(client, device, ticket, csrfToken)
	if err != nil {
		log.Printf("Warning: Failed to retrieve node status for %s: %v", device.Hostname, err)
		// Continue with VM metrics even if node metrics fail
	} else {
		// Push node metrics to VictoriaMetrics
		if err := c.pushNodeMetrics(device, nodeStats, timestamp); err != nil {
			log.Printf("Warning: Failed to push node metrics for %s: %v", device.Hostname, err)
		}
	}

	// Retrieve VM list
	vms, err := c.getVMList(client, device, ticket, csrfToken)
	if err != nil {
		return fmt.Errorf("failed to retrieve VM list: %w", err)
	}

	// Retrieve metrics for each VM
	for _, vm := range vms {
		vmStats, err := c.getVMStatus(client, device, ticket, csrfToken, vm.Node, vm.VMID)
		if err != nil {
			log.Printf("Warning: Failed to retrieve VM %d status: %v", vm.VMID, err)
			continue
		}

		// Push VM metrics to VictoriaMetrics
		if err := c.pushVMMetrics(device, vm, vmStats, timestamp); err != nil {
			log.Printf("Warning: Failed to push VM %d metrics: %v", vm.VMID, err)
		}
	}

	return nil
}

func (c *ProxmoxCollector) updateDeviceStatus(deviceID int64, status string, errorMsg string) {
	query := `UPDATE devices SET status = $1, updated_at = NOW() WHERE id = $2`
	if _, err := c.db.Exec(query, status, deviceID); err != nil {
		log.Printf("Error updating device status: %v", err)
		return
	}

	// Log the status update with device_id, timestamp, and error message
	log.Printf("Updated device %d status to %s at %s: %s", deviceID, status, time.Now().Format(time.RFC3339), errorMsg)

	// Update collector_status table
	statusQuery := `
		INSERT INTO collector_status (collector_name, device_id, last_poll_time, last_error)
		VALUES ('proxmox-collector', $1, NOW(), $2)
		ON CONFLICT (collector_name, device_id) 
		DO UPDATE SET 
			last_poll_time = NOW(),
			last_error = $2
	`
	if _, err := c.db.Exec(statusQuery, deviceID, errorMsg); err != nil {
		log.Printf("Error updating collector status: %v", err)
	}
}

func (c *ProxmoxCollector) updateCollectorStatusSuccess(deviceID int64) {
	statusQuery := `
		INSERT INTO collector_status (collector_name, device_id, last_poll_time, last_success_time, last_error)
		VALUES ('proxmox-collector', $1, NOW(), NOW(), '')
		ON CONFLICT (collector_name, device_id) 
		DO UPDATE SET 
			last_poll_time = NOW(),
			last_success_time = NOW(),
			last_error = ''
	`
	if _, err := c.db.Exec(statusQuery, deviceID); err != nil {
		log.Printf("Error updating collector status: %v", err)
	}
}

func (c *ProxmoxCollector) GetDeviceCount() int {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()
	return len(c.devices)
}

// Proxmox API response structures
type ProxmoxAuthResponse struct {
	Data struct {
		Ticket              string `json:"ticket"`
		CSRFPreventionToken string `json:"CSRFPreventionToken"`
		Username            string `json:"username"`
	} `json:"data"`
}

type ProxmoxNodeStatus struct {
	Data struct {
		CPU    float64 `json:"cpu"`
		Memory struct {
			Used  int64 `json:"used"`
			Total int64 `json:"total"`
		} `json:"memory"`
		RootFS struct {
			Used  int64 `json:"used"`
			Total int64 `json:"total"`
		} `json:"rootfs"`
		Uptime int64 `json:"uptime"`
	} `json:"data"`
}

type ProxmoxVM struct {
	VMID   int    `json:"vmid"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Node   string `json:"node"`
}

type ProxmoxVMList struct {
	Data []ProxmoxVM `json:"data"`
}

type ProxmoxVMStatus struct {
	Data struct {
		Status    string  `json:"status"`
		CPU       float64 `json:"cpu"`
		Mem       int64   `json:"mem"`
		MaxMem    int64   `json:"maxmem"`
		DiskRead  int64   `json:"diskread"`
		DiskWrite int64   `json:"diskwrite"`
		NetIn     int64   `json:"netin"`
		NetOut    int64   `json:"netout"`
		Uptime    int64   `json:"uptime"`
	} `json:"data"`
}

// createHTTPClient creates an HTTP client with TLS certificate validation and timeout
func (c *ProxmoxCollector) createHTTPClient() (*http.Client, error) {
	// Create TLS config with certificate validation
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false, // Enable certificate validation
		MinVersion:         tls.VersionTLS12,
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: c.timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			MaxIdleConns:    10,
			IdleConnTimeout: 30 * time.Second,
		},
	}

	return client, nil
}

// authenticate authenticates with Proxmox API using API token
func (c *ProxmoxCollector) authenticate(client *http.Client, device Device) (string, string, error) {
	// Parse API token (format: user@realm!tokenid=secret)
	// For now, we'll use the stored token directly
	// In production, this should be decrypted from device.APIToken

	// Proxmox API token authentication doesn't require a ticket
	// We can use the token directly in the Authorization header
	// Return empty strings for ticket and CSRF token when using API tokens
	if device.APIToken != "" {
		return device.APIToken, "", nil
	}

	return "", "", fmt.Errorf("no API token configured for device %s", device.Hostname)
}

// getNodeStatus retrieves node CPU, RAM, and disk usage
func (c *ProxmoxCollector) getNodeStatus(client *http.Client, device Device, ticket, csrfToken string) (*ProxmoxNodeStatus, error) {
	// Construct API URL - assuming node name is same as hostname
	url := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/status", device.IPAddress, device.Hostname)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	if ticket != "" {
		req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s", ticket))
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check for authentication errors
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		return nil, &AuthenticationError{Err: fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var nodeStatus ProxmoxNodeStatus
	if err := json.NewDecoder(resp.Body).Decode(&nodeStatus); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &nodeStatus, nil
}

// getVMList retrieves list of all VMs on the node
func (c *ProxmoxCollector) getVMList(client *http.Client, device Device, ticket, csrfToken string) ([]ProxmoxVM, error) {
	// Get VMs from all nodes - use cluster resources endpoint
	url := fmt.Sprintf("https://%s:8006/api2/json/cluster/resources?type=vm", device.IPAddress)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	if ticket != "" {
		req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s", ticket))
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check for authentication errors
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		return nil, &AuthenticationError{Err: fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var vmList ProxmoxVMList
	if err := json.NewDecoder(resp.Body).Decode(&vmList); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return vmList.Data, nil
}

// getVMStatus retrieves VM status and resource usage
func (c *ProxmoxCollector) getVMStatus(client *http.Client, device Device, ticket, csrfToken, node string, vmid int) (*ProxmoxVMStatus, error) {
	url := fmt.Sprintf("https://%s:8006/api2/json/nodes/%s/qemu/%d/status/current", device.IPAddress, node, vmid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	if ticket != "" {
		req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s", ticket))
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check for authentication errors
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		return nil, &AuthenticationError{Err: fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var vmStatus ProxmoxVMStatus
	if err := json.NewDecoder(resp.Body).Decode(&vmStatus); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &vmStatus, nil
}

// pushNodeMetrics pushes node metrics to VictoriaMetrics
func (c *ProxmoxCollector) pushNodeMetrics(device Device, nodeStats *ProxmoxNodeStatus, timestamp time.Time) error {
	deviceID := fmt.Sprintf("%d", device.ID)

	// Push CPU usage percentage
	if err := c.metricsWriter.WriteMetric(
		"infrasense_proxmox_node_cpu_usage_percent",
		nodeStats.Data.CPU*100,
		map[string]string{"device_id": deviceID, "node_name": device.Hostname},
		timestamp,
	); err != nil {
		return fmt.Errorf("failed to write CPU metric: %w", err)
	}

	// Push RAM usage percentage
	ramUsagePercent := 0.0
	if nodeStats.Data.Memory.Total > 0 {
		ramUsagePercent = float64(nodeStats.Data.Memory.Used) / float64(nodeStats.Data.Memory.Total) * 100
	}
	if err := c.metricsWriter.WriteMetric(
		"infrasense_proxmox_node_ram_usage_percent",
		ramUsagePercent,
		map[string]string{"device_id": deviceID, "node_name": device.Hostname},
		timestamp,
	); err != nil {
		return fmt.Errorf("failed to write RAM metric: %w", err)
	}

	// Push disk usage percentage
	diskUsagePercent := 0.0
	if nodeStats.Data.RootFS.Total > 0 {
		diskUsagePercent = float64(nodeStats.Data.RootFS.Used) / float64(nodeStats.Data.RootFS.Total) * 100
	}
	if err := c.metricsWriter.WriteMetric(
		"infrasense_proxmox_node_disk_usage_percent",
		diskUsagePercent,
		map[string]string{"device_id": deviceID, "node_name": device.Hostname},
		timestamp,
	); err != nil {
		return fmt.Errorf("failed to write disk metric: %w", err)
	}

	return nil
}

// pushVMMetrics pushes VM metrics to VictoriaMetrics
func (c *ProxmoxCollector) pushVMMetrics(device Device, vm ProxmoxVM, vmStats *ProxmoxVMStatus, timestamp time.Time) error {
	deviceID := fmt.Sprintf("%d", device.ID)
	vmID := fmt.Sprintf("%d", vm.VMID)

	labels := map[string]string{
		"device_id": deviceID,
		"vm_id":     vmID,
		"vm_name":   vm.Name,
		"node_name": vm.Node,
	}

	// Push VM status (running=1, stopped=0, paused=2)
	statusValue := 0.0
	switch vmStats.Data.Status {
	case "running":
		statusValue = 1.0
	case "stopped":
		statusValue = 0.0
	case "paused":
		statusValue = 2.0
	}
	if err := c.metricsWriter.WriteMetric(
		"infrasense_proxmox_vm_status",
		statusValue,
		labels,
		timestamp,
	); err != nil {
		return fmt.Errorf("failed to write VM status metric: %w", err)
	}

	// Push VM CPU usage percentage
	if err := c.metricsWriter.WriteMetric(
		"infrasense_proxmox_vm_cpu_usage_percent",
		vmStats.Data.CPU*100,
		labels,
		timestamp,
	); err != nil {
		return fmt.Errorf("failed to write VM CPU metric: %w", err)
	}

	// Push VM RAM usage bytes
	if err := c.metricsWriter.WriteMetric(
		"infrasense_proxmox_vm_ram_usage_bytes",
		float64(vmStats.Data.Mem),
		labels,
		timestamp,
	); err != nil {
		return fmt.Errorf("failed to write VM RAM metric: %w", err)
	}

	// Push VM disk I/O bytes (read)
	diskReadLabels := make(map[string]string)
	for k, v := range labels {
		diskReadLabels[k] = v
	}
	diskReadLabels["direction"] = "read"
	if err := c.metricsWriter.WriteMetric(
		"infrasense_proxmox_vm_disk_io_bytes",
		float64(vmStats.Data.DiskRead),
		diskReadLabels,
		timestamp,
	); err != nil {
		return fmt.Errorf("failed to write VM disk read metric: %w", err)
	}

	// Push VM disk I/O bytes (write)
	diskWriteLabels := make(map[string]string)
	for k, v := range labels {
		diskWriteLabels[k] = v
	}
	diskWriteLabels["direction"] = "write"
	if err := c.metricsWriter.WriteMetric(
		"infrasense_proxmox_vm_disk_io_bytes",
		float64(vmStats.Data.DiskWrite),
		diskWriteLabels,
		timestamp,
	); err != nil {
		return fmt.Errorf("failed to write VM disk write metric: %w", err)
	}

	// Push VM network traffic bytes (in)
	netInLabels := make(map[string]string)
	for k, v := range labels {
		netInLabels[k] = v
	}
	netInLabels["direction"] = "in"
	if err := c.metricsWriter.WriteMetric(
		"infrasense_proxmox_vm_network_bytes",
		float64(vmStats.Data.NetIn),
		netInLabels,
		timestamp,
	); err != nil {
		return fmt.Errorf("failed to write VM network in metric: %w", err)
	}

	// Push VM network traffic bytes (out)
	netOutLabels := make(map[string]string)
	for k, v := range labels {
		netOutLabels[k] = v
	}
	netOutLabels["direction"] = "out"
	if err := c.metricsWriter.WriteMetric(
		"infrasense_proxmox_vm_network_bytes",
		float64(vmStats.Data.NetOut),
		netOutLabels,
		timestamp,
	); err != nil {
		return fmt.Errorf("failed to write VM network out metric: %w", err)
	}

	return nil
}
