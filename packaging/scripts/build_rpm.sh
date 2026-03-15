#!/usr/bin/env bash
# Build the InfraSense .rpm package.
#
# Usage:
#   ./build_rpm.sh [--version <version>]
#
# Prerequisites:
#   - rpmbuild (part of rpm-build package on RHEL/Rocky/AlmaLinux)
#   - Compiled Go binaries placed in a temporary build directory
#     (or stub scripts for structural testing)
#
# Output:
#   infrasense/dist/infrasense-<version>-1.x86_64.rpm

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
RPM_ROOT="$SCRIPT_DIR/../rpm"
VERSION="1.0.0"

# ── Argument parsing ──────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

DIST_DIR="$REPO_ROOT/dist"
BUILD_DIR="$HOME/rpmbuild"
OUTPUT_PKG="$DIST_DIR/infrasense-${VERSION}-1.x86_64.rpm"

echo "[INFO] Building infrasense ${VERSION} .rpm package"
echo "[INFO] Spec file    : $RPM_ROOT/infrasense.spec"
echo "[INFO] Build dir    : $BUILD_DIR"
echo "[INFO] Output       : $OUTPUT_PKG"

# ── Verify rpmbuild is available ──────────────────────────────────────────────
if ! command -v rpmbuild &>/dev/null; then
  echo "ERROR: rpmbuild not found. Install it with: dnf install rpm-build" >&2
  exit 1
fi

# ── Create dist directory ─────────────────────────────────────────────────────
mkdir -p "$DIST_DIR"

# ── Create rpmbuild directory structure ──────────────────────────────────────
echo "[INFO] Creating rpmbuild directory structure..."
mkdir -p "$BUILD_DIR"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

# ── Create build directory with all files ────────────────────────────────────
echo "[INFO] Preparing build directory..."
BUILD_ROOT="$BUILD_DIR/BUILD/infrasense-${VERSION}"
rm -rf "$BUILD_ROOT"
mkdir -p "$BUILD_ROOT"/{bin,config,systemd,migrations}

# ── Copy binaries ─────────────────────────────────────────────────────────────
echo "[INFO] Copying binaries..."
BINARIES=(
  "infrasense-api"
  "infrasense-ipmi-collector"
  "infrasense-redfish-collector"
  "infrasense-snmp-collector"
  "infrasense-proxmox-collector"
  "infrasense-notification-service"
)

# Check if we have compiled binaries or stubs
BIN_SOURCE_DIR=""
if [[ -d "$REPO_ROOT/backend/cmd/server" ]]; then
  # Try to find compiled binaries in common locations
  for dir in "$REPO_ROOT/bin" "$REPO_ROOT/dist/bin" "$REPO_ROOT/build"; do
    if [[ -d "$dir" ]] && [[ -f "$dir/infrasense-api" ]]; then
      BIN_SOURCE_DIR="$dir"
      break
    fi
  done
fi

# If no compiled binaries found, create stub scripts for testing
if [[ -z "$BIN_SOURCE_DIR" ]]; then
  echo "[WARN] No compiled binaries found. Creating stub scripts for testing..."
  for bin in "${BINARIES[@]}"; do
    cat > "$BUILD_ROOT/bin/${bin}" <<'EOF'
#!/bin/sh
echo "Stub binary: $0"
exit 0
EOF
    chmod 755 "$BUILD_ROOT/bin/${bin}"
  done
else
  echo "[INFO] Using compiled binaries from: $BIN_SOURCE_DIR"
  for bin in "${BINARIES[@]}"; do
    if [[ -f "$BIN_SOURCE_DIR/${bin}" ]]; then
      cp "$BIN_SOURCE_DIR/${bin}" "$BUILD_ROOT/bin/"
      chmod 755 "$BUILD_ROOT/bin/${bin}"
    else
      echo "[WARN] Binary not found: $BIN_SOURCE_DIR/${bin}, creating stub..."
      cat > "$BUILD_ROOT/bin/${bin}" <<'EOF'
#!/bin/sh
echo "Stub binary: $0"
exit 0
EOF
      chmod 755 "$BUILD_ROOT/bin/${bin}"
    fi
  done
fi

# ── Copy configuration files ──────────────────────────────────────────────────
echo "[INFO] Copying configuration files..."
if [[ -f "$REPO_ROOT/backend/config.example.yml" ]]; then
  cp "$REPO_ROOT/backend/config.example.yml" "$BUILD_ROOT/config/config.yml"
else
  # Create minimal config file
  cat > "$BUILD_ROOT/config/config.yml" <<'EOF'
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
fi
chmod 644 "$BUILD_ROOT/config/config.yml"

# ── Copy systemd service units ────────────────────────────────────────────────
echo "[INFO] Copying systemd service units..."
cp "$RPM_ROOT/systemd"/*.service "$BUILD_ROOT/systemd/"
chmod 644 "$BUILD_ROOT/systemd"/*.service

# ── Copy database migrations ──────────────────────────────────────────────────
echo "[INFO] Copying database migrations..."
if [[ -d "$REPO_ROOT/backend/internal/db/migrations" ]]; then
  cp -r "$REPO_ROOT/backend/internal/db/migrations"/* "$BUILD_ROOT/migrations/" 2>/dev/null || true
else
  echo "[WARN] No migrations found at $REPO_ROOT/backend/internal/db/migrations"
  # Create empty migrations directory
  touch "$BUILD_ROOT/migrations/.gitkeep"
fi

# ── Copy spec file to SPECS directory ─────────────────────────────────────────
echo "[INFO] Copying spec file..."
cp "$RPM_ROOT/infrasense.spec" "$BUILD_DIR/SPECS/"

# ── Update version in spec file ───────────────────────────────────────────────
sed -i "s/^Version:.*/Version:        $VERSION/" "$BUILD_DIR/SPECS/infrasense.spec"

# ── Build the package ─────────────────────────────────────────────────────────
echo "[INFO] Running rpmbuild..."
cd "$BUILD_DIR"
rpmbuild -ba \
  --define "_topdir $BUILD_DIR" \
  --define "_builddir $BUILD_DIR/BUILD" \
  SPECS/infrasense.spec

# ── Copy the built package to dist directory ──────────────────────────────────
BUILT_RPM=$(find "$BUILD_DIR/RPMS/x86_64" -name "infrasense-${VERSION}-*.rpm" | head -1)
if [[ -z "$BUILT_RPM" ]]; then
  echo "ERROR: Built RPM not found in $BUILD_DIR/RPMS/x86_64/" >&2
  exit 1
fi

cp "$BUILT_RPM" "$OUTPUT_PKG"

echo ""
echo "[INFO] Package built: $OUTPUT_PKG"

# ── Validate the package ──────────────────────────────────────────────────────
echo ""
echo "[INFO] Package info:"
rpm -qip "$OUTPUT_PKG"

echo ""
echo "[INFO] Package contents:"
rpm -qlp "$OUTPUT_PKG"

echo ""
echo "[SUCCESS] Build complete: $OUTPUT_PKG"
