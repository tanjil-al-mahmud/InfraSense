#!/usr/bin/env bash
# Test .rpm package installation on both Rocky Linux 8 and AlmaLinux 8
#
# Usage:
#   ./test_all_rpm.sh [<path-to-.rpm>]
#
# If no path is given, looks for infrasense-*.rpm in ../../dist/

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PKG="${1:-}"

if [[ -z "$PKG" ]]; then
  PKG=$(ls "$SCRIPT_DIR/../../dist"/infrasense-*.rpm 2>/dev/null | head -1 || true)
fi

if [[ -z "$PKG" || ! -f "$PKG" ]]; then
  echo "ERROR: .rpm package not found. Build it first or pass the path as an argument."
  echo "  Usage: $0 <path/to/infrasense-*.rpm>"
  exit 1
fi

echo "=========================================="
echo "Testing RPM package on multiple platforms"
echo "Package: $PKG"
echo "=========================================="
echo ""

FAILED=0

# Test on Rocky Linux 8
echo "[1/2] Testing on Rocky Linux 8..."
if "$SCRIPT_DIR/test_rpm_install.sh" "$PKG"; then
  echo "[✓] Rocky Linux 8 test PASSED"
else
  echo "[✗] Rocky Linux 8 test FAILED"
  FAILED=$((FAILED + 1))
fi
echo ""

# Test on AlmaLinux 8
echo "[2/2] Testing on AlmaLinux 8..."
if "$SCRIPT_DIR/test_rpm_almalinux.sh" "$PKG"; then
  echo "[✓] AlmaLinux 8 test PASSED"
else
  echo "[✗] AlmaLinux 8 test FAILED"
  FAILED=$((FAILED + 1))
fi
echo ""

echo "=========================================="
if [[ $FAILED -eq 0 ]]; then
  echo "All tests PASSED"
  exit 0
else
  echo "$FAILED test(s) FAILED"
  exit 1
fi
