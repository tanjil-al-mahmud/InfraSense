#!/bin/bash
#
# Test script for install.sh validation
# Tests basic script structure and logic without actually installing
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_SCRIPT="$SCRIPT_DIR/../scripts/install.sh"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

run_test() {
    local test_name="$1"
    local test_command="$2"
    
    if eval "$test_command" >/dev/null 2>&1; then
        print_success "$test_name"
        ((TESTS_PASSED++))
    else
        print_error "$test_name"
        ((TESTS_FAILED++))
    fi
}

echo "========================================="
echo "Testing install.sh script"
echo "========================================="
echo ""

# Test 1: Script exists
run_test "Script file exists" "[ -f '$INSTALL_SCRIPT' ]"

# Test 2: Script has shebang
run_test "Script has bash shebang" "head -n 1 '$INSTALL_SCRIPT' | grep -q '^#!/bin/bash'"

# Test 3: Script has set -e
run_test "Script has 'set -e' for error handling" "grep -q '^set -e' '$INSTALL_SCRIPT'"

# Test 4: Script checks for root
run_test "Script checks for root privileges" "grep -q 'EUID.*-ne 0' '$INSTALL_SCRIPT'"

# Test 5: Script detects OS from /etc/os-release
run_test "Script detects OS from /etc/os-release" "grep -q '/etc/os-release' '$INSTALL_SCRIPT'"

# Test 6: Script checks port 80
run_test "Script checks port 80 availability" "grep -q 'check_port 80' '$INSTALL_SCRIPT'"

# Test 7: Script checks port 443
run_test "Script checks port 443 availability" "grep -q 'check_port 443' '$INSTALL_SCRIPT'"

# Test 8: Script checks port 5432
run_test "Script checks port 5432 availability" "grep -q 'check_port 5432' '$INSTALL_SCRIPT'"

# Test 9: Script checks port 8428
run_test "Script checks port 8428 availability" "grep -q 'check_port 8428' '$INSTALL_SCRIPT'"

# Test 10: Script has install_deb function
run_test "Script has install_deb function" "grep -q 'install_deb()' '$INSTALL_SCRIPT'"

# Test 11: Script has install_rpm function
run_test "Script has install_rpm function" "grep -q 'install_rpm()' '$INSTALL_SCRIPT'"

# Test 12: Script handles Ubuntu
run_test "Script handles Ubuntu distribution" "grep -q 'ubuntu)' '$INSTALL_SCRIPT'"

# Test 13: Script handles Debian
run_test "Script handles Debian distribution" "grep -q 'debian)' '$INSTALL_SCRIPT'"

# Test 14: Script handles RHEL
run_test "Script handles RHEL distribution" "grep -q 'rhel' '$INSTALL_SCRIPT'"

# Test 15: Script handles Rocky Linux
run_test "Script handles Rocky Linux distribution" "grep -q 'rocky' '$INSTALL_SCRIPT'"

# Test 16: Script handles AlmaLinux
run_test "Script handles AlmaLinux distribution" "grep -q 'almalinux' '$INSTALL_SCRIPT'"

# Test 17: Script shows Docker instructions for unsupported OS
run_test "Script has Docker installation instructions" "grep -q 'docker-compose' '$INSTALL_SCRIPT'"

# Test 18: Script installs dependencies
run_test "Script installs postgresql-client dependency" "grep -q 'postgresql' '$INSTALL_SCRIPT'"

# Test 19: Script installs systemd
run_test "Script installs systemd dependency" "grep -q 'systemd' '$INSTALL_SCRIPT'"

# Test 20: Script downloads .deb package
run_test "Script downloads .deb package" "grep -q 'infrasense_amd64.deb' '$INSTALL_SCRIPT'"

# Test 21: Script downloads .rpm package
run_test "Script downloads .rpm package" "grep -q 'infrasense_x86_64.rpm' '$INSTALL_SCRIPT'"

# Test 22: Script prints success message
run_test "Script prints installation success message" "grep -q 'Installation Complete' '$INSTALL_SCRIPT'"

# Test 23: Script prints access URL
run_test "Script prints access URL" "grep -q 'Access the dashboard' '$INSTALL_SCRIPT'"

# Test 24: Script prints admin credentials location
run_test "Script prints admin credentials info" "grep -q 'admin' '$INSTALL_SCRIPT'"

# Test 25: Script has error handling
run_test "Script has error handling with exit codes" "grep -q 'exit 1' '$INSTALL_SCRIPT'"

echo ""
echo "========================================="
echo "Test Results"
echo "========================================="
echo "Passed: $TESTS_PASSED"
echo "Failed: $TESTS_FAILED"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    print_success "All tests passed!"
    exit 0
else
    print_error "Some tests failed"
    exit 1
fi
