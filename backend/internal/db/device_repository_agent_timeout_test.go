package db

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/models"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	_ "github.com/lib/pq"
)

// **Validates: Requirements 4.7, 5.7**
// Property 43: Agent Timeout Detection
// For any agent (node_exporter or windows_exporter) that stops sending metrics for 5 minutes,
// the Agent_Receiver SHALL mark the device as unavailable.

func TestProperty_AgentTimeoutDetection(t *testing.T) {
	// Skip if no test database is available
	db := setupTestDatabase(t)
	if db == nil {
		t.Skip("Test database not available")
	}
	defer db.Close()

	repo := NewDeviceRepository(&DB{conn: db})

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("agents timing out after 5 minutes are marked unavailable", prop.ForAll(
		func(deviceCount int, timeoutMinutes int) bool {
			ctx := context.Background()

			// Clean up test data
			defer cleanupTestDevices(t, db)

			// Create test devices with various last_seen timestamps
			devices := createTestAgentDevices(t, db, deviceCount, timeoutMinutes)

			// Run the timeout detection
			timeout := 5 * time.Minute
			count, err := repo.MarkTimedOutAgentsUnavailable(ctx, timeout)
			if err != nil {
				t.Logf("Error marking timed-out agents: %v", err)
				return false
			}

			// Verify the correct devices were marked as unavailable
			expectedCount := countExpectedTimeouts(devices, timeout)
			if count != expectedCount {
				t.Logf("Expected %d devices to be marked unavailable, got %d", expectedCount, count)
				return false
			}

			// Verify device statuses in database
			for _, device := range devices {
				status, err := getDeviceStatus(db, device.ID)
				if err != nil {
					t.Logf("Error getting device status: %v", err)
					return false
				}

				shouldBeUnavailable := shouldDeviceBeMarkedUnavailable(device, timeout)
				if shouldBeUnavailable && status != "unavailable" {
					t.Logf("Device %s should be unavailable but has status %s", device.ID, status)
					return false
				}
				if !shouldBeUnavailable && device.Status != "unavailable" && status == "unavailable" {
					t.Logf("Device %s should not be unavailable but has status %s", device.ID, status)
					return false
				}
			}

			return true
		},
		gen.IntRange(1, 20), // deviceCount: 1-20 devices
		gen.IntRange(0, 15), // timeoutMinutes: 0-15 minutes ago
	))

	properties.TestingRun(t)
}

// createTestAgentDevices creates test devices with various configurations
func createTestAgentDevices(t *testing.T, db *sql.DB, deviceCount int, maxTimeoutMinutes int) []models.Device {
	devices := make([]models.Device, 0, deviceCount)

	deviceTypes := []string{"linux_agent", "windows_agent"}
	statuses := []string{"healthy", "warning", "unavailable"}

	for i := 0; i < deviceCount; i++ {
		device := models.Device{
			ID:         uuid.New(),
			Hostname:   fmt.Sprintf("test-agent-%d", i),
			IPAddress:  fmt.Sprintf("192.168.1.%d", i+1),
			DeviceType: deviceTypes[i%len(deviceTypes)],
			Status:     statuses[i%len(statuses)],
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		// Vary the last_seen timestamp
		// Some devices will be timed out, some won't
		minutesAgo := i % (maxTimeoutMinutes + 1)
		if minutesAgo > 0 {
			lastSeen := time.Now().Add(-time.Duration(minutesAgo) * time.Minute)
			device.LastSeen = &lastSeen
		} else {
			// Some devices have never sent metrics (NULL last_seen)
			device.LastSeen = nil
		}

		// Insert device into database
		query := `
			INSERT INTO devices (id, hostname, ip_address, device_type, status, last_seen, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err := db.Exec(query, device.ID, device.Hostname, device.IPAddress, device.DeviceType,
			device.Status, device.LastSeen, device.CreatedAt, device.UpdatedAt)
		if err != nil {
			t.Fatalf("Failed to insert test device: %v", err)
		}

		devices = append(devices, device)
	}

	return devices
}

// countExpectedTimeouts counts how many devices should be marked as unavailable
func countExpectedTimeouts(devices []models.Device, timeout time.Duration) int {
	count := 0
	for _, device := range devices {
		if shouldDeviceBeMarkedUnavailable(device, timeout) {
			count++
		}
	}
	return count
}

// shouldDeviceBeMarkedUnavailable determines if a device should be marked unavailable
func shouldDeviceBeMarkedUnavailable(device models.Device, timeout time.Duration) bool {
	// Only agent device types
	if device.DeviceType != "linux_agent" && device.DeviceType != "windows_agent" {
		return false
	}

	// Already unavailable devices are not counted
	if device.Status == "unavailable" {
		return false
	}

	// NULL last_seen or last_seen older than timeout
	if device.LastSeen == nil {
		return true
	}

	return time.Since(*device.LastSeen) > timeout
}

// getDeviceStatus retrieves the current status of a device from the database
func getDeviceStatus(db *sql.DB, deviceID uuid.UUID) (string, error) {
	var status string
	query := `SELECT status FROM devices WHERE id = $1`
	err := db.QueryRow(query, deviceID).Scan(&status)
	if err != nil {
		return "", err
	}
	return status, nil
}

// cleanupTestDevices removes all test devices from the database
func cleanupTestDevices(t *testing.T, db *sql.DB) {
	_, err := db.Exec(`DELETE FROM devices WHERE hostname LIKE 'test-agent-%'`)
	if err != nil {
		t.Logf("Warning: Failed to cleanup test devices: %v", err)
	}
}

// setupTestDatabase sets up a test database connection
func setupTestDatabase(t *testing.T) *sql.DB {
	// Try to connect to test database
	// This should be configured via environment variables in CI/CD
	connStr := "host=localhost port=5432 user=infrasense password=infrasense dbname=infrasense_test sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Logf("Cannot connect to test database: %v", err)
		return nil
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		t.Logf("Cannot ping test database: %v", err)
		db.Close()
		return nil
	}

	return db
}
