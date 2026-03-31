#!/usr/bin/env bash
# InfraSense package installation test suite.
# Executed INSIDE a Docker container by run_tests.sh.
#
# Environment (injected by run_tests.sh):
#   INSTALL_CMD  – shell command to install the package, e.g.
#                  "apt-get install -y /tmp/infrasense_pkg"
#
# The package is always mounted at /tmp/infrasense_pkg.

set -uo pipefail

# ── Colour helpers ────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}[PASS]${NC} $*"; }
fail() { echo -e "${RED}[FAIL]${NC} $*"; }
info() { echo -e "${YELLOW}[INFO]${NC} $*"; }

FAILURES=0

assert_pass() {
  local desc="$1"; shift
  if "$@" &>/dev/null; then
    pass "$desc"
  else
    fail "$desc"
    ((FAILURES++))
  fi
}

assert_fail() {
  local desc="$1"; shift
  if ! "$@" &>/dev/null; then
    pass "$desc (expected failure)"
  else
    fail "$desc (expected failure but succeeded)"
    ((FAILURES++))
  fi
}

assert_file_exists() {
  local path="$1"
  if [[ -e "$path" ]]; then
    pass "File exists: $path"
  else
    fail "File missing: $path"
    ((FAILURES++))
  fi
}

assert_service_active() {
  local svc="$1"
  if systemctl is-active --quiet "$svc" 2>/dev/null; then
    pass "Service active: $svc"
  else
    fail "Service not active: $svc"
    ((FAILURES++))
  fi
}

assert_port_open() {
  local host="$1" port="$2" desc="${3:-port $port}"
  local timeout=30
  local elapsed=0
  while ! bash -c "echo > /dev/tcp/$host/$port" 2>/dev/null; do
    sleep 2
    elapsed=$((elapsed + 2))
    if [[ $elapsed -ge $timeout ]]; then
      fail "Port not open after ${timeout}s: $desc ($host:$port)"
      ((FAILURES++))
      return
    fi
  done
  pass "Port open: $desc ($host:$port)"
}

assert_http_ok() {
  local url="$1" desc="${2:-$1}"
  local timeout=30
  local elapsed=0
  local code
  while true; do
    code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$url" 2>/dev/null || echo "000")
    if [[ "$code" == "200" ]]; then
      pass "HTTP 200: $desc"
      return
    fi
    sleep 2
    elapsed=$((elapsed + 2))
    if [[ $elapsed -ge $timeout ]]; then
      fail "HTTP not 200 after ${timeout}s (got $code): $desc"
      ((FAILURES++))
      return
    fi
  done
}

assert_db_table_exists() {
  local table="$1"
  if su -s /bin/bash infrasense -c \
      "psql -U infrasense -d infrasense -tAc \"SELECT 1 FROM information_schema.tables WHERE table_name='$table'\"" \
      2>/dev/null | grep -q 1; then
    pass "DB table exists: $table"
  else
    fail "DB table missing: $table"
    ((FAILURES++))
  fi
}

assert_db_user_exists() {
  local username="$1"
  if su -s /bin/bash infrasense -c \
      "psql -U infrasense -d infrasense -tAc \"SELECT 1 FROM users WHERE username='$username'\"" \
      2>/dev/null | grep -q 1; then
    pass "DB user exists: $username"
  else
    fail "DB user missing: $username"
    ((FAILURES++))
  fi
}

# ── Load install command ──────────────────────────────────────────────────────
# shellcheck source=/dev/null
[[ -f /tmp/install_cmd.env ]] && source /tmp/install_cmd.env

if [[ -z "${INSTALL_CMD:-}" ]]; then
  echo "ERROR: INSTALL_CMD not set" >&2
  exit 1
fi

# ── Detect package manager ────────────────────────────────────────────────────
if command -v apt-get &>/dev/null; then
  PKG_MANAGER="deb"
  info "Package manager: apt (Debian/Ubuntu)"
elif command -v dnf &>/dev/null || command -v yum &>/dev/null; then
  PKG_MANAGER="rpm"
  info "Package manager: dnf/yum (RHEL/Rocky)"
else
  echo "ERROR: unsupported package manager" >&2
  exit 1
fi

# ── Test 1: Pre-installation sanity ──────────────────────────────────────────
info "--- Test 1: Pre-installation sanity ---"
assert_fail "infrasense binary absent before install" test -f /usr/local/bin/infrasense-api

# ── Test 2: Package installation ─────────────────────────────────────────────
info "--- Test 2: Package installation ---"

# Update package index for deb systems
if [[ "$PKG_MANAGER" == "deb" ]]; then
  info "Updating apt cache..."
  apt-get update -qq
fi

info "Running: $INSTALL_CMD"
if eval "$INSTALL_CMD"; then
  pass "Package installed without errors"
else
  fail "Package installation failed"
  ((FAILURES++))
  echo "FATAL: cannot continue without successful installation"
  exit 1
fi

# ── Test 3: Binary installation ───────────────────────────────────────────────
info "--- Test 3: Binaries installed to /usr/local/bin/ ---"
for bin in infrasense-api infrasense-ipmi-collector infrasense-redfish-collector \
           infrasense-snmp-collector infrasense-notification-service; do
  assert_file_exists "/usr/local/bin/$bin"
done

# ── Test 4: Configuration files ───────────────────────────────────────────────
info "--- Test 4: Configuration files in /etc/infrasense/ ---"
assert_file_exists "/etc/infrasense"
assert_file_exists "/etc/infrasense/config.yml"

# ── Test 5: Systemd service units ─────────────────────────────────────────────
info "--- Test 5: Systemd service units installed ---"
for svc in infrasense-api infrasense-ipmi-collector infrasense-redfish-collector \
           infrasense-snmp-collector infrasense-notification-service; do
  assert_file_exists "/etc/systemd/system/${svc}.service"
done

# ── Test 6: System user and group ─────────────────────────────────────────────
info "--- Test 6: infrasense system user and group created ---"
if id infrasense &>/dev/null; then
  pass "System user 'infrasense' exists"
else
  fail "System user 'infrasense' missing"
  ((FAILURES++))
fi

if getent group infrasense &>/dev/null; then
  pass "System group 'infrasense' exists"
else
  fail "System group 'infrasense' missing"
  ((FAILURES++))
fi

# ── Test 7: Services start successfully ───────────────────────────────────────
info "--- Test 7: Services start successfully ---"
# Give services a moment to start (post-install script should have started them)
sleep 5

for svc in infrasense-api; do
  assert_service_active "$svc"
done

# ── Test 8: Database initialized ──────────────────────────────────────────────
info "--- Test 8: Database schema initialized ---"
# Core tables that must exist after post-install migration
for table in devices users alert_rules maintenance_windows audit_logs; do
  assert_db_table_exists "$table"
done

# ── Test 9: Default admin user created ────────────────────────────────────────
info "--- Test 9: Default admin user created ---"
assert_db_user_exists "admin"

# ── Test 10: API server accessible on port 80 ─────────────────────────────────
info "--- Test 10: API server accessible on port 80 ---"
assert_port_open "127.0.0.1" 80 "API server"
assert_http_ok "http://127.0.0.1/health" "GET /health"

# ── Test 11: API login with default admin credentials ─────────────────────────
info "--- Test 11: API login with default admin credentials ---"
LOGIN_RESPONSE=$(curl -s -X POST http://127.0.0.1/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' 2>/dev/null || echo "")

if echo "$LOGIN_RESPONSE" | grep -q '"token"'; then
  pass "Admin login returns JWT token"
else
  fail "Admin login did not return a token (response: $LOGIN_RESPONSE)"
  ((FAILURES++))
fi

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "Suite complete. Failures: $FAILURES"
[[ $FAILURES -eq 0 ]]
