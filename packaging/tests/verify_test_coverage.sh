#!/usr/bin/env bash
# Verify that the integration test suite covers all requirements from task 6.12
#
# Task 6.12 Requirements:
# - Test .deb package installation on Ubuntu 22.04 container
# - Test .rpm package installation on Rocky Linux 8 container
# - Verify all services start successfully after installation
# - Verify database initialized correctly
# - Verify default admin user created
# - Verify API server accessible on port 80

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SUITE_FILE="$SCRIPT_DIR/suite.sh"
RUN_TESTS_FILE="$SCRIPT_DIR/run_tests.sh"

echo "Verifying integration test coverage for Task 6.12..."
echo ""

# Check if required files exist
if [[ ! -f "$SUITE_FILE" ]]; then
  echo "❌ FAIL: suite.sh not found"
  exit 1
fi

if [[ ! -f "$RUN_TESTS_FILE" ]]; then
  echo "❌ FAIL: run_tests.sh not found"
  exit 1
fi

echo "✅ Required test files exist"
echo ""

# Check for Ubuntu 22.04 support
if grep -q "ubuntu:22.04" "$RUN_TESTS_FILE"; then
  echo "✅ Ubuntu 22.04 container support: PRESENT"
else
  echo "❌ Ubuntu 22.04 container support: MISSING"
  exit 1
fi

# Check for Rocky Linux 8 support
if grep -q "rockylinux:8" "$RUN_TESTS_FILE"; then
  echo "✅ Rocky Linux 8 container support: PRESENT"
else
  echo "❌ Rocky Linux 8 container support: MISSING"
  exit 1
fi

echo ""

# Check for service start verification
if grep -q "assert_service_active" "$SUITE_FILE"; then
  echo "✅ Service start verification: PRESENT"
else
  echo "❌ Service start verification: MISSING"
  exit 1
fi

# Check for database initialization verification
if grep -q "assert_db_table_exists" "$SUITE_FILE"; then
  echo "✅ Database initialization verification: PRESENT"
else
  echo "❌ Database initialization verification: MISSING"
  exit 1
fi

# Check for admin user verification
if grep -q "assert_db_user_exists" "$SUITE_FILE"; then
  echo "✅ Admin user verification: PRESENT"
else
  echo "❌ Admin user verification: MISSING"
  exit 1
fi

# Check for API server port 80 verification
if grep -q "assert_port_open.*80" "$SUITE_FILE"; then
  echo "✅ API server port 80 verification: PRESENT"
else
  echo "❌ API server port 80 verification: MISSING"
  exit 1
fi

echo ""
echo "=========================================="
echo "✅ All Task 6.12 requirements are covered"
echo "=========================================="
echo ""
echo "Test suite is ready to run:"
echo "  cd $SCRIPT_DIR"
echo "  ./run_tests.sh"
