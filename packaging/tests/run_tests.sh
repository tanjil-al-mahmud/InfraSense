#!/usr/bin/env bash
# InfraSense Package Installation Integration Tests
# Spins up Docker containers, installs packages, and verifies expected outcomes.
#
# Usage:
#   ./run_tests.sh [--deb-only | --rpm-only] [--package-dir <path>]
#
# Prerequisites:
#   - Docker installed and running
#   - .deb package built at <package-dir>/infrasense_*.deb  (default: ../../dist)
#   - .rpm package built at <package-dir>/infrasense-*.rpm  (default: ../../dist)
#
# Exit codes:
#   0  All tests passed
#   1  One or more tests failed

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="${PACKAGE_DIR:-$(cd "$SCRIPT_DIR/../../dist" 2>/dev/null && pwd || echo "$SCRIPT_DIR/../../dist")}"
RUN_DEB=true
RUN_RPM=true
PASS=0
FAIL=0

# ── Colour helpers ────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}[PASS]${NC} $*"; ((PASS++)); }
fail() { echo -e "${RED}[FAIL]${NC} $*"; ((FAIL++)); }
info() { echo -e "${YELLOW}[INFO]${NC} $*"; }

# ── Argument parsing ──────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --deb-only)   RUN_RPM=false; shift ;;
    --rpm-only)   RUN_DEB=false; shift ;;
    --package-dir) PACKAGE_DIR="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

# ── Docker availability check ─────────────────────────────────────────────────
if ! command -v docker &>/dev/null; then
  echo "ERROR: docker is not installed or not in PATH" >&2
  exit 1
fi

if ! docker info &>/dev/null; then
  echo "ERROR: Docker daemon is not running" >&2
  exit 1
fi

# ── Helper: run a test suite inside a container ───────────────────────────────
# run_suite <suite_name> <image> <package_file> <install_cmd>
run_suite() {
  local suite="$1"
  local image="$2"
  local pkg_file="$3"
  local install_cmd="$4"

  info "========================================================"
  info "Suite: $suite  (image: $image)"
  info "========================================================"

  if [[ ! -f "$pkg_file" ]]; then
    fail "$suite: package file not found: $pkg_file"
    info "Build the package first, then re-run this script."
    return
  fi

  local container
  container=$(docker run -d \
    --privileged \
    -v "$pkg_file:/tmp/infrasense_pkg" \
    -v "$SCRIPT_DIR/suite.sh:/tmp/suite.sh:ro" \
    "$image" \
    sleep 3600)

  info "Container started: $container"

  # Copy install command into container env
  docker exec "$container" bash -c "echo 'INSTALL_CMD=\"$install_cmd\"' > /tmp/install_cmd.env"

  # Run the test suite
  local output
  local rc=0
  output=$(docker exec "$container" bash /tmp/suite.sh 2>&1) || rc=$?

  echo "$output"

  # Parse PASS/FAIL counts from suite output
  local suite_pass suite_fail
  suite_pass=$(echo "$output" | grep -c '^\[PASS\]' || true)
  suite_fail=$(echo "$output"  | grep -c '^\[FAIL\]' || true)

  PASS=$((PASS + suite_pass))
  FAIL=$((FAIL + suite_fail))

  docker rm -f "$container" &>/dev/null || true

  if [[ $rc -ne 0 || $suite_fail -gt 0 ]]; then
    fail "$suite: suite exited with failures"
  else
    pass "$suite: all checks passed"
  fi
}

# ── Locate packages ───────────────────────────────────────────────────────────
DEB_PKG=$(ls "$PACKAGE_DIR"/infrasense_*.deb 2>/dev/null | head -1 || true)
RPM_PKG=$(ls "$PACKAGE_DIR"/infrasense-*.rpm 2>/dev/null | head -1 || true)

# ── Run suites ────────────────────────────────────────────────────────────────
if $RUN_DEB; then
  run_suite \
    "deb-ubuntu-22.04" \
    "ubuntu:22.04" \
    "${DEB_PKG:-$PACKAGE_DIR/infrasense_MISSING.deb}" \
    "apt-get install -y /tmp/infrasense_pkg"
fi

if $RUN_RPM; then
  run_suite \
    "rpm-rocky-8" \
    "rockylinux:8" \
    "${RPM_PKG:-$PACKAGE_DIR/infrasense_MISSING.rpm}" \
    "dnf install -y /tmp/infrasense_pkg"
fi

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "========================================================"
echo "  Results: ${PASS} passed, ${FAIL} failed"
echo "========================================================"

[[ $FAIL -eq 0 ]]
