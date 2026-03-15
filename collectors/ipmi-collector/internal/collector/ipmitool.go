package collector

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Metric struct {
	Name      string
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
}

type IPMIData struct {
	Metrics []Metric
	SELLogs []string
}

// ExecuteIPMITool executes ipmitool command with proper argument escaping
func ExecuteIPMITool(ctx context.Context, host, username, password string, args ...string) (string, error) {
	// Validate inputs to prevent command injection
	if err := validateIPMIInput(host); err != nil {
		return "", fmt.Errorf("invalid host: %w", err)
	}
	if err := validateIPMIInput(username); err != nil {
		return "", fmt.Errorf("invalid username: %w", err)
	}

	// Build ipmitool command
	cmdArgs := []string{
		"-I", "lanplus",
		"-H", host,
		"-U", username,
		"-P", password,
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, "ipmitool", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ipmitool command failed: %w (output: %s)", err, string(output))
	}

	return string(output), nil
}

// validateIPMIInput validates input to prevent command injection
func validateIPMIInput(input string) error {
	// Allow alphanumeric, dots, hyphens, underscores
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9.\-_]+$`)
	if !validPattern.MatchString(input) {
		return fmt.Errorf("input contains invalid characters")
	}
	return nil
}

// CollectIPMIData collects all IPMI sensor data and SEL logs
func CollectIPMIData(ctx context.Context, device Device) (*IPMIData, error) {
	data := &IPMIData{
		Metrics: make([]Metric, 0),
		SELLogs: make([]string, 0),
	}

	// Collect sensor data
	sensorOutput, err := ExecuteIPMITool(ctx, device.IPAddress, device.Username, device.Password, "sdr", "list", "full")
	if err != nil {
		return nil, fmt.Errorf("failed to collect sensor data: %w", err)
	}

	metrics, err := parseSensorData(sensorOutput, device.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sensor data: %w", err)
	}
	data.Metrics = append(data.Metrics, metrics...)

	// Collect PSU status
	psuMetrics, err := collectPSUStatus(ctx, device)
	if err != nil {
		// Log error but continue with other metrics
		fmt.Printf("Warning: failed to collect PSU status for device %s: %v\n", device.Hostname, err)
	} else {
		data.Metrics = append(data.Metrics, psuMetrics...)
	}

	// Collect SEL logs
	selOutput, err := ExecuteIPMITool(ctx, device.IPAddress, device.Username, device.Password, "sel", "list")
	if err != nil {
		// Log error but continue
		fmt.Printf("Warning: failed to collect SEL logs for device %s: %v\n", device.Hostname, err)
	} else {
		data.SELLogs = parseSELLogs(selOutput)
	}

	return data, nil
}

// parseSensorData parses ipmitool sdr output
func parseSensorData(output string, deviceID int64) ([]Metric, error) {
	metrics := make([]Metric, 0)
	scanner := bufio.NewScanner(strings.NewReader(output))

	// Regular expression to parse sensor lines
	// Example: "CPU Temp        | 45 degrees C      | ok"
	sensorRegex := regexp.MustCompile(`^([^|]+)\s*\|\s*([^|]+)\s*\|\s*(.+)$`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := sensorRegex.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue
		}

		sensorName := strings.TrimSpace(matches[1])
		valueStr := strings.TrimSpace(matches[2])
		status := strings.TrimSpace(matches[3])

		// Extract numeric value and unit
		value, unit, err := parseValue(valueStr)
		if err != nil {
			continue // Skip non-numeric sensors
		}

		// Determine sensor type
		sensorType := determineSensorType(sensorName, unit)

		metric := Metric{
			Name:  fmt.Sprintf("infrasense_ipmi_%s", sensorType),
			Value: value,
			Labels: map[string]string{
				"device_id":   fmt.Sprintf("%d", deviceID),
				"sensor_name": sensorName,
				"sensor_type": sensorType,
				"status":      status,
			},
			Timestamp: time.Now(),
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// parseValue extracts numeric value and unit from sensor reading
func parseValue(valueStr string) (float64, string, error) {
	// Handle different formats: "45 degrees C", "1200 RPM", "12.5 Volts", "250 Watts"
	parts := strings.Fields(valueStr)
	if len(parts) == 0 {
		return 0, "", fmt.Errorf("empty value")
	}

	// Try to parse first part as number
	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, "", fmt.Errorf("not a number: %s", parts[0])
	}

	unit := ""
	if len(parts) > 1 {
		unit = strings.ToLower(parts[1])
	}

	return value, unit, nil
}

// determineSensorType determines the metric type based on sensor name and unit
func determineSensorType(sensorName, unit string) string {
	sensorNameLower := strings.ToLower(sensorName)

	// Temperature sensors
	if strings.Contains(sensorNameLower, "temp") || strings.Contains(unit, "degrees") {
		if strings.Contains(sensorNameLower, "cpu") {
			return "temperature_cpu_celsius"
		} else if strings.Contains(sensorNameLower, "inlet") {
			return "temperature_inlet_celsius"
		} else if strings.Contains(sensorNameLower, "exhaust") {
			return "temperature_exhaust_celsius"
		} else if strings.Contains(sensorNameLower, "system") {
			return "temperature_system_celsius"
		}
		return "temperature_celsius"
	}

	// Fan sensors
	if strings.Contains(sensorNameLower, "fan") || strings.Contains(unit, "rpm") {
		return "fan_speed_rpm"
	}

	// Voltage sensors
	if strings.Contains(sensorNameLower, "volt") || strings.Contains(unit, "volts") {
		if strings.Contains(sensorNameLower, "12v") {
			return "voltage_12v"
		} else if strings.Contains(sensorNameLower, "5v") {
			return "voltage_5v"
		} else if strings.Contains(sensorNameLower, "3.3v") {
			return "voltage_3v3"
		} else if strings.Contains(sensorNameLower, "vcore") || strings.Contains(sensorNameLower, "cpu") {
			return "voltage_vcore"
		}
		return "voltage"
	}

	// Power sensors
	if strings.Contains(sensorNameLower, "power") || strings.Contains(sensorNameLower, "watt") || strings.Contains(unit, "watts") {
		return "power_watts"
	}

	return "other"
}

// collectPSUStatus collects PSU status information
func collectPSUStatus(ctx context.Context, device Device) ([]Metric, error) {
	output, err := ExecuteIPMITool(ctx, device.IPAddress, device.Username, device.Password, "sdr", "type", "Power Supply")
	if err != nil {
		return nil, err
	}

	metrics := make([]Metric, 0)
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Power Supply") {
			// Parse PSU status
			// Example: "PS1 Status       | 0x01              | ok"
			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
				psuName := strings.TrimSpace(parts[0])
				status := strings.TrimSpace(parts[2])

				// Convert status to numeric (1 = ok, 0 = failed)
				statusValue := 0.0
				if strings.Contains(strings.ToLower(status), "ok") {
					statusValue = 1.0
				}

				metric := Metric{
					Name:  "infrasense_ipmi_psu_status",
					Value: statusValue,
					Labels: map[string]string{
						"device_id":   fmt.Sprintf("%d", device.ID),
						"sensor_name": psuName,
						"sensor_type": "psu_status",
						"status":      status,
					},
					Timestamp: time.Now(),
				}
				metrics = append(metrics, metric)
			}
		}
	}

	return metrics, nil
}

// parseSELLogs parses System Event Log entries
func parseSELLogs(output string) []string {
	logs := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" && !strings.HasPrefix(line, "SEL has no entries") {
			logs = append(logs, line)
		}
	}

	return logs
}
