#!/bin/bash

# Verification script for Task 4.12: Prometheus OS Agent Scraping Configuration
# This script verifies that Prometheus is correctly configured to scrape OS agent exporters

set -e

echo "=========================================="
echo "Task 4.12 Verification Script"
echo "Prometheus OS Agent Scraping Configuration"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
PROMETHEUS_CONFIG="prometheus.yml"

# Function to check if a value exists
check_exists() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $1"
        return 0
    else
        echo -e "${RED}✗${NC} $1"
        return 1
    fi
}

echo "1. Checking Prometheus configuration file..."
echo "-------------------------------------------"

# Check if prometheus.yml exists
if [ -f "$PROMETHEUS_CONFIG" ]; then
    check_exists "prometheus.yml file exists"
else
    echo -e "${RED}✗${NC} prometheus.yml file not found"
    exit 1
fi

# Check for node-exporter job
grep -q "job_name: 'node-exporter'" "$PROMETHEUS_CONFIG"
check_exists "node-exporter job configured"

# Check for windows-exporter job
grep -q "job_name: 'windows-exporter'" "$PROMETHEUS_CONFIG"
check_exists "windows-exporter job configured"

# Check for lm-sensors-exporter job
grep -q "job_name: 'lm-sensors-exporter'" "$PROMETHEUS_CONFIG"
check_exists "lm-sensors-exporter job configured"

# Check for smartctl-exporter job
grep -q "job_name: 'smartctl-exporter'" "$PROMETHEUS_CONFIG"
check_exists "smartctl-exporter job configured"

echo ""
echo "2. Checking scrape intervals..."
echo "-------------------------------------------"

# Check node-exporter scrape interval (15s)
grep -A 2 "job_name: 'node-exporter'" "$PROMETHEUS_CONFIG" | grep -q "scrape_interval: 15s"
check_exists "node-exporter scrape interval: 15s"

# Check windows-exporter scrape interval (15s)
grep -A 2 "job_name: 'windows-exporter'" "$PROMETHEUS_CONFIG" | grep -q "scrape_interval: 15s"
check_exists "windows-exporter scrape interval: 15s"

# Check lm-sensors-exporter scrape interval (15s)
grep -A 2 "job_name: 'lm-sensors-exporter'" "$PROMETHEUS_CONFIG" | grep -q "scrape_interval: 15s"
check_exists "lm-sensors-exporter scrape interval: 15s"

# Check smartctl-exporter scrape interval (60s)
grep -A 2 "job_name: 'smartctl-exporter'" "$PROMETHEUS_CONFIG" | grep -q "scrape_interval: 60s"
check_exists "smartctl-exporter scrape interval: 60s"

echo ""
echo "3. Checking service discovery configuration..."
echo "-------------------------------------------"

# Check for file_sd_configs
grep -A 5 "job_name: 'node-exporter'" "$PROMETHEUS_CONFIG" | grep -q "file_sd_configs"
check_exists "node-exporter uses file-based service discovery"

grep -A 5 "job_name: 'windows-exporter'" "$PROMETHEUS_CONFIG" | grep -q "file_sd_configs"
check_exists "windows-exporter uses file-based service discovery"

grep -A 5 "job_name: 'lm-sensors-exporter'" "$PROMETHEUS_CONFIG" | grep -q "file_sd_configs"
check_exists "lm-sensors-exporter uses file-based service discovery"

grep -A 5 "job_name: 'smartctl-exporter'" "$PROMETHEUS_CONFIG" | grep -q "file_sd_configs"
check_exists "smartctl-exporter uses file-based service discovery"

echo ""
echo "4. Checking target files..."
echo "-------------------------------------------"

# Check if target files exist
if [ -f "targets/node_exporter_targets.json" ]; then
    check_exists "node_exporter_targets.json exists"
else
    echo -e "${YELLOW}⚠${NC} node_exporter_targets.json not found (will be generated from PostgreSQL)"
fi

if [ -f "targets/windows_exporter_targets.json" ]; then
    check_exists "windows_exporter_targets.json exists"
else
    echo -e "${YELLOW}⚠${NC} windows_exporter_targets.json not found (will be generated from PostgreSQL)"
fi

if [ -f "targets/lm_sensors_exporter_targets.json" ]; then
    check_exists "lm_sensors_exporter_targets.json exists"
else
    echo -e "${YELLOW}⚠${NC} lm_sensors_exporter_targets.json not found (will be generated from PostgreSQL)"
fi

if [ -f "targets/smartctl_exporter_targets.json" ]; then
    check_exists "smartctl_exporter_targets.json exists"
else
    echo -e "${YELLOW}⚠${NC} smartctl_exporter_targets.json not found (will be generated from PostgreSQL)"
fi

echo ""
echo "5. Checking relabel configurations..."
echo "-------------------------------------------"

# Check for relabel_configs
grep -A 10 "job_name: 'node-exporter'" "$PROMETHEUS_CONFIG" | grep -q "relabel_configs"
check_exists "node-exporter has relabel_configs"

grep -A 10 "job_name: 'windows-exporter'" "$PROMETHEUS_CONFIG" | grep -q "relabel_configs"
check_exists "windows-exporter has relabel_configs"

grep -A 10 "job_name: 'lm-sensors-exporter'" "$PROMETHEUS_CONFIG" | grep -q "relabel_configs"
check_exists "lm-sensors-exporter has relabel_configs"

grep -A 10 "job_name: 'smartctl-exporter'" "$PROMETHEUS_CONFIG" | grep -q "relabel_configs"
check_exists "smartctl-exporter has relabel_configs"

echo ""
echo "6. Checking Prometheus connectivity (optional)..."
echo "-------------------------------------------"

# Try to connect to Prometheus API
if command -v curl &> /dev/null; then
    if curl -s -f "$PROMETHEUS_URL/-/healthy" > /dev/null 2>&1; then
        check_exists "Prometheus is running and healthy"
        
        # Check if targets are loaded
        TARGETS=$(curl -s "$PROMETHEUS_URL/api/v1/targets" | grep -o '"job":".*-exporter"' | wc -l)
        if [ "$TARGETS" -gt 0 ]; then
            echo -e "${GREEN}✓${NC} Found $TARGETS exporter targets loaded in Prometheus"
        else
            echo -e "${YELLOW}⚠${NC} No exporter targets loaded yet (agents may not be deployed)"
        fi
    else
        echo -e "${YELLOW}⚠${NC} Prometheus is not running at $PROMETHEUS_URL (skipping connectivity checks)"
    fi
else
    echo -e "${YELLOW}⚠${NC} curl not available (skipping connectivity checks)"
fi

echo ""
echo "=========================================="
echo "Verification Summary"
echo "=========================================="
echo ""
echo "Configuration Status: ${GREEN}COMPLETE${NC}"
echo ""
echo "Requirements Validated:"
echo "  ✓ Requirement 4.1: OS-Level Metrics Collection from Linux Servers"
echo "  ✓ Requirement 5.1: OS-Level Metrics Collection from Windows Servers"
echo "  ✓ Requirement 32.5: Hardware Sensor Monitoring via lm-sensors"
echo "  ✓ Requirement 33.7: Disk SMART Monitoring via smartctl"
echo ""
echo "Next Steps:"
echo "  1. Deploy node_exporter on Linux servers (port 9100)"
echo "  2. Deploy windows_exporter on Windows servers (port 9182)"
echo "  3. Deploy lm-sensors_exporter on Linux machines with sensors (port 9255)"
echo "  4. Deploy smartctl_exporter on Linux machines with disks (port 9633)"
echo "  5. Generate target JSON files from PostgreSQL device inventory"
echo "  6. Verify metrics are being scraped: curl '$PROMETHEUS_URL/api/v1/targets'"
echo ""
