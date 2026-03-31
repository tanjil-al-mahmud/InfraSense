package collector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/infrasense/ipmi-collector/internal/queue"
)

const (
	maxBackoffDuration = 10 * time.Minute
	initialBackoff     = 1 * time.Second
)

type RetryState struct {
	deviceID     string
	hostname     string
	failureCount int
	lastAttempt  time.Time
	nextAttempt  time.Time
}

type RetryManager struct {
	states map[string]*RetryState
}

func NewRetryManager() *RetryManager {
	return &RetryManager{
		states: make(map[string]*RetryState),
	}
}

func (rm *RetryManager) ShouldRetry(deviceID string, hostname string) bool {
	state, exists := rm.states[deviceID]
	if !exists {
		// First attempt
		return true
	}

	// Check if enough time has passed for next retry
	return time.Now().After(state.nextAttempt)
}

func (rm *RetryManager) RecordFailure(deviceID string, hostname string) time.Duration {
	state, exists := rm.states[deviceID]
	if !exists {
		state = &RetryState{
			deviceID:     deviceID,
			hostname:     hostname,
			failureCount: 0,
		}
		rm.states[deviceID] = state
	}

	state.failureCount++
	state.lastAttempt = time.Now()

	// Calculate exponential backoff: 1s, 2s, 4s, 8s, ..., max 10 minutes
	backoff := time.Duration(math.Pow(2, float64(state.failureCount-1))) * initialBackoff
	if backoff > maxBackoffDuration {
		backoff = maxBackoffDuration
	}

	state.nextAttempt = time.Now().Add(backoff)

	slog.Warn("device poll failed, scheduling retry",
		"event", "device_poll_failure",
		"device_id", deviceID,
		"hostname", hostname,
		"failure_count", state.failureCount,
		"retry_in_seconds", backoff.Seconds())

	return backoff
}

func (rm *RetryManager) RecordSuccess(deviceID string) {
	// Reset failure count on success
	delete(rm.states, deviceID)
}

func (rm *RetryManager) GetFailureCount(deviceID string) int {
	state, exists := rm.states[deviceID]
	if !exists {
		return 0
	}
	return state.failureCount
}

// PollDevice polls a single device by running all 5 ipmitool commands with a 30-second
// context timeout. On any command failure it logs the command name, exit code, and stderr,
// marks the device unavailable, and returns false. On full success it pushes all collected
// metrics to VictoriaMetrics, marks the device healthy, and returns true.
func (c *IPMICollector) PollDevice(device Device) bool {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	var allMetrics []Metric

	// 1. chassis status → infrasense_ipmi_chassis_power_state
	chassisMetrics, err := runChassisStatus(ctx, device)
	if err != nil {
		slog.Error("IPMI poll failed",
			"event", "poll_failed",
			"hostname", device.Hostname,
			"bmc_ip", device.BMCIPAddress,
			"step", "chassis status",
			"error", err.Error())
		c.updateDeviceStatus(device.ID, "offline", fmt.Sprintf("chassis status failed: %v", err))
		return false
	}
	allMetrics = append(allMetrics, chassisMetrics...)

	// 2. sensor list → infrasense_ipmi_sensor_value
	sensorMetrics, err := runSensorList(ctx, device)
	if err != nil {
		slog.Error("IPMI poll failed",
			"event", "poll_failed",
			"hostname", device.Hostname,
			"bmc_ip", device.BMCIPAddress,
			"step", "sensor list",
			"error", err.Error())
		c.updateDeviceStatus(device.ID, "offline", fmt.Sprintf("sensor list failed: %v", err))
		return false
	}
	allMetrics = append(allMetrics, sensorMetrics...)

	slog.Info("Collected sensors from device",
		"event", "sensors_collected",
		"hostname", device.Hostname,
		"bmc_ip", device.BMCIPAddress,
		"sensor_count", len(sensorMetrics))

	// 3. sdr → sensor metadata enrichment (no metrics emitted directly)
	_, err = runSDR(ctx, device)
	if err != nil {
		// Log but don't fail — SDR is supplementary
		slog.Warn("sdr metadata collection failed (non-fatal)",
			"event", "sdr_warn",
			"hostname", device.Hostname,
			"error", err.Error())
	}

	// 4. sel list → infrasense_ipmi_sel_entries_total{severity}
	selMetrics, err := runSELList(ctx, device)
	if err != nil {
		// Log but don't fail — SEL is supplementary
		slog.Warn("sel list collection failed (non-fatal)",
			"event", "sel_warn",
			"hostname", device.Hostname,
			"error", err.Error())
	} else {
		allMetrics = append(allMetrics, selMetrics...)
	}

	// 5. fru → upsert manufacturer/product/serial into device_inventory
	if err := runFRU(ctx, device, c.db); err != nil {
		// Log but don't fail — FRU is supplementary
		slog.Warn("fru inventory collection failed (non-fatal)",
			"event", "fru_warn",
			"hostname", device.Hostname,
			"error", err.Error())
	}

	// Push all collected metrics to VictoriaMetrics
	pushed := 0
	for _, metric := range allMetrics {
		if err := c.metricsWriter.WriteMetric(metric.Name, metric.Value, metric.Labels, metric.Timestamp); err != nil {
			slog.Error("metric write error",
				"event", "metric_write_error",
				"device_id", device.ID,
				"hostname", device.Hostname,
				"error", err.Error())
		} else {
			pushed++
		}
	}

	slog.Info("Pushed metrics to VictoriaMetrics",
		"event", "metrics_pushed",
		"hostname", device.Hostname,
		"metric_count", pushed)

	// Publish SEL entries as normalized events (best-effort)
	if c.publisher != nil {
		selOutput, err := ExecuteIPMIToolWithPort(ctx, device.BMCIPAddress, device.Port, device.Username, device.Password, "sel", "list")
		if err == nil {
			events := normalizeSEL(device.ID, selOutput)
			if len(events) > 0 {
				pubCtx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
				defer cancel()
				for _, ev := range events {
					_ = c.publisher.PublishEvent(pubCtx, ev)
				}
			}
		}
	}

	// Publish metrics to NATS JetStream for durable ingestion
	if c.publisher != nil && len(allMetrics) > 0 {
		samples := make([]queue.MetricSample, 0, len(allMetrics))
		for _, m := range allMetrics {
			samples = append(samples, queue.MetricSample{
				Name:      m.Name,
				Value:     m.Value,
				Labels:    m.Labels,
				Timestamp: m.Timestamp,
			})
		}
		pubCtx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
		defer cancel()
		_ = c.publisher.PublishMetrics(pubCtx, queue.MetricsBatch{
			SchemaVersion: "v1",
			DeviceID:      device.ID,
			Source:        "ipmi",
			CollectedAt:   time.Now(),
			Samples:       samples,
		})
	}

	return true
}

func normalizeSEL(deviceID string, selOutput string) []queue.HardwareEvent {
	lines := strings.Split(selOutput, "\n")
	observedAt := time.Now()
	out := make([]queue.HardwareEvent, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "SEL has no entries") {
			continue
		}
		lower := strings.ToLower(line)
		sev := "info"
		switch {
		case strings.Contains(lower, "critical") || strings.Contains(lower, "non-recoverable") || strings.Contains(lower, "failure"):
			sev = "critical"
		case strings.Contains(lower, "warning") || strings.Contains(lower, "non-critical") || strings.Contains(lower, "degraded"):
			sev = "warning"
		}

		sum := sha256.Sum256([]byte(line))
		dedupe := hex.EncodeToString(sum[:])

		out = append(out, queue.HardwareEvent{
			SchemaVersion:  "v1",
			DeviceID:       deviceID,
			ObservedAt:     observedAt,
			SourceProtocol: "ipmi",
			Component:      "system",
			EventType:      "sel",
			Severity:       sev,
			Message:        line,
			Raw: map[string]any{
				"line": line,
			},
			DedupeKey: dedupe,
		})
	}
	return out
}

// PollDeviceWithRetry polls a device with exponential backoff retry logic
func (c *IPMICollector) PollDeviceWithRetry(device Device) {
	// Check if we should retry this device
	if !c.retryManager.ShouldRetry(device.ID, device.Hostname) {
		return
	}

	timestamp := time.Now()

	slog.Info("Starting IPMI poll for device",
		"event", "poll_attempt",
		"hostname", device.Hostname,
		"bmc_ip", device.BMCIPAddress,
		"timestamp", timestamp.Format(time.RFC3339))

	ok := c.PollDevice(device)
	if !ok {
		slog.Error("IPMI poll failed for device",
			"event", "poll_failed",
			"hostname", device.Hostname,
			"bmc_ip", device.BMCIPAddress,
			"timestamp", timestamp.Format(time.RFC3339))

		// Record failure and get backoff duration
		backoff := c.retryManager.RecordFailure(device.ID, device.Hostname)

		// Protocol fallback after repeated failures
		if c.retryManager.GetFailureCount(device.ID) >= 3 {
			slog.Warn("ipmi failed repeatedly; falling back to snmp",
				"event", "protocol_fallback",
				"device_id", device.ID,
				"from", "ipmi",
				"to", "snmp")
			c.setDeviceProtocol(device.ID, "snmp")
		}

		slog.Warn("device marked offline, scheduling retry",
			"event", "device_offline",
			"hostname", device.Hostname,
			"bmc_ip", device.BMCIPAddress,
			"retry_in_seconds", backoff.Seconds())

		return
	}

	slog.Info("IPMI poll successful",
		"event", "poll_success",
		"hostname", device.Hostname,
		"bmc_ip", device.BMCIPAddress,
		"timestamp", timestamp.Format(time.RFC3339))

	// Record success (resets failure count)
	c.retryManager.RecordSuccess(device.ID)

	// Update device status to online
	c.updateCollectorStatusSuccess(device.ID)
}
