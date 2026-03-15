#!/usr/bin/env bash
# Standalone test: .deb package installation on Ubuntu 22.04
#
# Usage:
#   ./test_deb_install.sh [<path-to-.deb>]
#
# If no path is given, looks for infrasense_*.deb in ../../dist/

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PKG="${1:-}"

if [[ -z "$PKG" ]]; then
  PKG=$(ls "$SCRIPT_DIR/../../dist"/infrasense_*.deb 2>/dev/null | head -1 || true)
fi

if [[ -z "$PKG" || ! -f "$PKG" ]]; then
  echo "ERROR: .deb package not found. Build it first or pass the path as an argument."
  echo "  Usage: $0 <path/to/infrasense_*.deb>"
  exit 1
fi

echo "[INFO] Using package: $PKG"

CONTAINER=$(docker run -d \
  --privileged \
  -v "$PKG:/tmp/infrasense_pkg" \
  -v "$SCRIPT_DIR/suite.sh:/tmp/suite.sh:ro" \
  ubuntu:22.04 \
  sleep 3600)

echo "[INFO] Container: $CONTAINER"

docker exec "$CONTAINER" bash -c \
  "echo 'INSTALL_CMD=\"apt-get install -y /tmp/infrasense_pkg\"' > /tmp/install_cmd.env"

RC=0
docker exec "$CONTAINER" bash /tmp/suite.sh || RC=$?

docker rm -f "$CONTAINER" &>/dev/null || true

exit $RC
