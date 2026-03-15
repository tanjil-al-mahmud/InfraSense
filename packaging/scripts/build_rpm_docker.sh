#!/usr/bin/env bash
# Build the InfraSense .rpm package using Docker (Rocky Linux 8 container).
#
# Usage:
#   ./build_rpm_docker.sh [--version <version>]
#
# Prerequisites:
#   - Docker
#
# Output:
#   infrasense/dist/infrasense-<version>-1.x86_64.rpm

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
VERSION="1.0.0"

# ── Argument parsing ──────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

DIST_DIR="$REPO_ROOT/dist"
OUTPUT_PKG="$DIST_DIR/infrasense-${VERSION}-1.x86_64.rpm"

echo "[INFO] Building infrasense ${VERSION} .rpm package using Docker"
echo "[INFO] Output: $OUTPUT_PKG"

# ── Verify Docker is available ────────────────────────────────────────────────
if ! command -v docker &>/dev/null; then
  echo "ERROR: docker not found. Please install Docker." >&2
  exit 1
fi

# ── Create dist directory ─────────────────────────────────────────────────────
mkdir -p "$DIST_DIR"

# ── Build in Rocky Linux 8 container ──────────────────────────────────────────
echo "[INFO] Starting Rocky Linux 8 container..."

docker run --rm \
  -v "$REPO_ROOT:/workspace:ro" \
  -v "$DIST_DIR:/output" \
  -w /workspace \
  rockylinux:8 \
  bash -c '
set -euo pipefail

echo "[INFO] Installing build dependencies..."
dnf install -y rpm-build rpmdevtools

echo "[INFO] Setting up rpmbuild directory..."
rpmdev-setuptree

VERSION="'"$VERSION"'"
BUILD_DIR="$HOME/rpmbuild"
BUILD_ROOT="$BUILD_DIR/BUILD/infrasense-${VERSION}"

echo "[INFO] Preparing build directory..."
mkdir -p "$BUILD_ROOT"/{bin,config,systemd,migrations}

# ── Copy binaries (create stubs for testing) ──────────────────────────────────
echo "[INFO] Creating stub binaries..."
BINARIES=(
  "infrasense-api"
  "infrasense-ipmi-collector"
  "infrasense-redfish-collector"
  "infrasense-snmp-collector"
  "infrasense-proxmox-collector"
  "infrasense-notification-service"
)

for bin in "${BINARIES[@]}"; do
  cat > "$BUILD_ROOT/bin/${bin}" <<'"'"'EOF'"'"'
#!/bin/sh
echo "Stub binary: $0"
exit 0
EOF
  chmod 755 "$BUILD_ROOT/bin/${bin}"
done

# ── Copy configuration files ──────────────────────────────────────────────────
echo "[INFO] Creating configuration file..."
cat > "$BUILD_ROOT/config/config.yml" <<'"'"'EOF'"'"'
# InfraSense Platform Configuration
database:
  host: localhost
  port: 5432
  name: infrasense
  user: infrasense
  password: ""
  sslmode: disable

server:
  port: 8080
  jwt_secret: ""

victoriametrics:
  url: http://localhost:8428
EOF
chmod 644 "$BUILD_ROOT/config/config.yml"

# ── Copy systemd service units ────────────────────────────────────────────────
echo "[INFO] Copying systemd service units..."
cp /workspace/packaging/rpm/systemd/*.service "$BUILD_ROOT/systemd/"
chmod 644 "$BUILD_ROOT/systemd"/*.service

# ── Copy database migrations ──────────────────────────────────────────────────
echo "[INFO] Copying database migrations..."
if [[ -d /workspace/backend/internal/db/migrations ]]; then
  cp -r /workspace/backend/internal/db/migrations/* "$BUILD_ROOT/migrations/" 2>/dev/null || true
else
  touch "$BUILD_ROOT/migrations/.gitkeep"
fi

# ── Copy spec file ────────────────────────────────────────────────────────────
echo "[INFO] Copying spec file..."
cp /workspace/packaging/rpm/infrasense.spec "$BUILD_DIR/SPECS/"

# ── Update version in spec file ───────────────────────────────────────────────
sed -i "s/^Version:.*/Version:        $VERSION/" "$BUILD_DIR/SPECS/infrasense.spec"

# ── Build the package ─────────────────────────────────────────────────────────
echo "[INFO] Running rpmbuild..."
cd "$BUILD_DIR"
rpmbuild -ba \
  --define "_topdir $BUILD_DIR" \
  --define "_builddir $BUILD_DIR/BUILD" \
  SPECS/infrasense.spec

# ── Copy the built package to output directory ────────────────────────────────
BUILT_RPM=$(find "$BUILD_DIR/RPMS/x86_64" -name "infrasense-${VERSION}-*.rpm" | head -1)
if [[ -z "$BUILT_RPM" ]]; then
  echo "ERROR: Built RPM not found" >&2
  exit 1
fi

cp "$BUILT_RPM" /output/
chmod 644 /output/$(basename "$BUILT_RPM")

echo ""
echo "[INFO] Package info:"
rpm -qip "$BUILT_RPM"

echo ""
echo "[SUCCESS] Build complete"
'

echo ""
echo "[SUCCESS] Package built: $OUTPUT_PKG"
