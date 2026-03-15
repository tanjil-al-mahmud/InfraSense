package collector

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"
)

const (
	maxBackoffDuration = 10 * time.Minute
	initialBackoff     = 1 * time.Second
)

type RetryState struct {
	deviceID     int64
	hostname     string
	failureCount int
	lastAttempt  time.Time
	nextAttempt  time.Time
}

type RetryManager struct {
	states map[int64]*RetryState
}

func NewRetryManager() *RetryManager {
	return &RetryManager{
		states: make(map[int64]*RetryState),
	}
}

func (rm *RetryManager) ShouldRetry(deviceID int64, hostname string) bool {
	state, exists := rm.states[deviceID]
	if !exists {
		// First attempt
		return true
	}

	// Check if enough time has passed for next retry
	return time.Now().After(state.nextAttempt)
}

func (rm *RetryManager) RecordFailure(deviceID int64, hostname string) time.Duration {
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

func (rm *RetryManager) RecordSuccess(deviceID int64) {
	// Reset failure count on success
	delete(rm.states, deviceID)
}

func (rm *RetryManager) GetFailureCount(deviceID int64) int {
	state, exists := rm.states[deviceID]
	if !exists {
		return 0
	}
	return state.failureCount
}

// PollDeviceWithRetry polls a device with exponential backoff retry logic
func (c *IPMICollector) PollDeviceWithRetry(device Device) {
	// Check if we should retry this device
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
		"timestamp", timestamp.Format(time.RFC3339))

	// Collect IPMI data
	data, err := CollectIPMIData(ctx, device)
	if err != nil {
		slog.Error("device poll failed",
			"event", "poll_attempt",
			"device_id", device.ID,
			"hostname", device.Hostname,
			"timestamp", timestamp.Format(time.RFC3339),
			"result", "error",
			"error", err.Error())

		slog.Error("connection error",
			"event", "connection_error",
			"device_id", device.ID,
			"hostname", device.Hostname,
			"timestamp", timestamp.Format(time.RFC3339),
			"error", err.Error())

		// Record failure and get backoff duration
		backoff := c.retryManager.RecordFailure(device.ID, device.Hostname)

		// Update device status to unavailable
		c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("Connection failed: %v (retry in %v)", err, backoff))

		return
	}

	slog.Info("device poll successful",
		"event", "poll_attempt",
		"device_id", device.ID,
		"hostname", device.Hostname,
		"timestamp", timestamp.Format(time.RFC3339),
		"result", "success",
		"metrics_count", len(data.Metrics))

	// Push metrics to VictoriaMetrics
	for _, metric := range data.Metrics {
		if err := c.metricsWriter.WriteMetric(metric.Name, metric.Value, metric.Labels, metric.Timestamp); err != nil {
			slog.Error("metric write error",
				"event", "metric_write_error",
				"device_id", device.ID,
				"hostname", device.Hostname,
				"timestamp", time.Now().Format(time.RFC3339),
				"error", err.Error())
		}
	}

	// Record success (resets failure count)
	c.retryManager.RecordSuccess(device.ID)

	// Update device status to healthy
	c.updateDeviceStatus(device.ID, "healthy", "")

	// Update collector status with success
	c.updateCollectorStatusSuccess(device.ID)
}
