#!/usr/bin/env bash
# InfraSense Quick Smoke Test
# Usage: ./quick-test.sh [BASE_URL]
# Environment: ADMIN_PASSWORD (default: admin)

set -euo pipefail

BASE_URL="${1:-http://localhost}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin}"

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

PASS=0
FAIL=0
TOKEN=""

pass() { echo -e "${GREEN}PASS${NC} $1"; ((PASS++)); }
fail() { echo -e "${RED}FAIL${NC} $1"; ((FAIL++)); }

# Test 1: API health check
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/health" 2>/dev/null || echo "000")
if [ "$STATUS" = "200" ]; then
  pass "Test 1: API health check (GET /health → 200)"
else
  fail "Test 1: API health check (GET /health → expected 200, got ${STATUS})"
fi

# Test 2: Login with admin credentials
RESPONSE=$(curl -s -o /tmp/infrasense_login.json -w "%{http_code}" \
  -X POST "${BASE_URL}/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"admin\",\"password\":\"${ADMIN_PASSWORD}\"}" 2>/dev/null || echo "000")
if [ "$RESPONSE" = "200" ]; then
  TOKEN=$(jq -r '.token // .access_token // empty' /tmp/infrasense_login.json 2>/dev/null || echo "")
  if [ -n "$TOKEN" ]; then
    pass "Test 2: Login (POST /api/v1/auth/login → 200, token received)"
  else
    fail "Test 2: Login (POST /api/v1/auth/login → 200 but no token in response)"
  fi
else
  fail "Test 2: Login (POST /api/v1/auth/login → expected 200, got ${RESPONSE})"
fi

AUTH_HEADER=""
if [ -n "$TOKEN" ]; then
  AUTH_HEADER="Authorization: Bearer ${TOKEN}"
fi

# Test 3: List devices
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "${AUTH_HEADER}" \
  "${BASE_URL}/api/v1/devices" 2>/dev/null || echo "000")
if [ "$STATUS" = "200" ]; then
  pass "Test 3: List devices (GET /api/v1/devices → 200)"
else
  fail "Test 3: List devices (GET /api/v1/devices → expected 200, got ${STATUS})"
fi

# Test 4: List alerts
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "${AUTH_HEADER}" \
  "${BASE_URL}/api/v1/alerts" 2>/dev/null || echo "000")
if [ "$STATUS" = "200" ]; then
  pass "Test 4: List alerts (GET /api/v1/alerts → 200)"
else
  fail "Test 4: List alerts (GET /api/v1/alerts → expected 200, got ${STATUS})"
fi

# Test 5: List alert rules
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "${AUTH_HEADER}" \
  "${BASE_URL}/api/v1/alert-rules" 2>/dev/null || echo "000")
if [ "$STATUS" = "200" ]; then
  pass "Test 5: List alert rules (GET /api/v1/alert-rules → 200)"
else
  fail "Test 5: List alert rules (GET /api/v1/alert-rules → expected 200, got ${STATUS})"
fi

# Test 6: VictoriaMetrics health
VM_HOST="${BASE_URL%:*}"  # strip port if present, use same host
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  "${VM_HOST}:8428/health" 2>/dev/null || echo "000")
if [ "$STATUS" = "200" ]; then
  pass "Test 6: VictoriaMetrics health (GET :8428/health → 200)"
else
  fail "Test 6: VictoriaMetrics health (GET :8428/health → expected 200, got ${STATUS})"
fi

# Test 7: Prometheus health
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  "${VM_HOST}:9090/-/healthy" 2>/dev/null || echo "000")
if [ "$STATUS" = "200" ]; then
  pass "Test 7: Prometheus health (GET :9090/-/healthy → 200)"
else
  fail "Test 7: Prometheus health (GET :9090/-/healthy → expected 200, got ${STATUS})"
fi

# Summary
TOTAL=$((PASS + FAIL))
echo ""
echo "Results: ${PASS}/${TOTAL} tests passed"

rm -f /tmp/infrasense_login.json

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
