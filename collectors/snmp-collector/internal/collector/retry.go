package collector

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/gosnmp/gosnmp"
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

	log.Printf("Device %s failed %d times, next retry in %v", hostname, state.failureCount, backoff)

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
func (c *SNMPCollector) PollDeviceWithRetry(device Device) {
	// Check if we should retry this device
	if !c.retryManager.ShouldRetry(device.ID, device.Hostname) {
		return
	}

	log.Printf("Polling device %s (%s)", device.Hostname, device.IPAddress)

	// Create SNMP client
	snmpClient := &gosnmp.GoSNMP{
		Target:    device.IPAddress,
		Port:      161,
		Transport: "udp",
		Timeout:   c.timeout,
		Retries:   2,
	}

	// Configure SNMP version and authentication
	if device.SNMPVersion == "snmp_v3" {
		snmpClient.Version = gosnmp.Version3
		snmpClient.SecurityModel = gosnmp.UserSecurityModel
		snmpClient.MsgFlags = gosnmp.AuthPriv

		// Set authentication
		switch device.AuthProtocol {
		case "MD5":
			snmpClient.SecurityParameters = &gosnmp.UsmSecurityParameters{
				UserName:                 device.Username,
				AuthenticationProtocol:   gosnmp.MD5,
				AuthenticationPassphrase: device.AuthPassword,
				PrivacyProtocol:          gosnmp.DES,
				PrivacyPassphrase:        device.PrivPassword,
			}
		case "SHA":
			snmpClient.SecurityParameters = &gosnmp.UsmSecurityParameters{
				UserName:                 device.Username,
				AuthenticationProtocol:   gosnmp.SHA,
				AuthenticationPassphrase: device.AuthPassword,
				PrivacyProtocol:          gosnmp.AES,
				PrivacyPassphrase:        device.PrivPassword,
			}
		}
	} else {
		// Default to v2c
		snmpClient.Version = gosnmp.Version2c
		snmpClient.Community = device.Community
	}

	// Connect to device
	err := snmpClient.Connect()
	if err != nil {
		log.Printf("Failed to connect to device %s (%s): %v", device.Hostname, device.IPAddress, err)

		// Record failure and get backoff duration
		backoff := c.retryManager.RecordFailure(device.ID, device.Hostname)

		// Update device status to unavailable
		c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("Connection failed: %v (retry in %v)", err, backoff))

		return
	}
	defer snmpClient.Conn.Close()

	timestamp := time.Now()

	// Poll UPS metrics
	if err := c.pollUPSMetrics(snmpClient, device, timestamp); err != nil {
		log.Printf("Error polling device %s (%s): %v", device.Hostname, device.IPAddress, err)

		// Record failure and get backoff duration
		backoff := c.retryManager.RecordFailure(device.ID, device.Hostname)

		// Update device status to unavailable
		c.updateDeviceStatus(device.ID, "unavailable", fmt.Sprintf("Polling failed: %v (retry in %v)", err, backoff))

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
