#!/usr/bin/env bash
# Build the InfraSense .deb package.
#
# Usage:
#   ./build_deb.sh [--version <version>]
#
# Prerequisites:
#   - dpkg-deb (part of dpkg package on Debian/Ubuntu)
#   - Compiled Go binaries placed in packaging/deb/usr/local/bin/
#     (or stub scripts for structural testing)
#
# Output:
#   infrasense/dist/infrasense_<version>_amd64.deb

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DEB_ROOT="$SCRIPT_DIR/../deb"
VERSION="1.0.0"

# ── Argument parsing ──────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

DIST_DIR="$REPO_ROOT/dist"
OUTPUT_PKG="$DIST_DIR/infrasense_${VERSION}_amd64.deb"

echo "[INFO] Building infrasense ${VERSION} .deb package"
echo "[INFO] Package root : $DEB_ROOT"
echo "[INFO] Output       : $OUTPUT_PKG"

# ── Verify dpkg-deb is available ──────────────────────────────────────────────
if ! command -v dpkg-deb &>/dev/null; then
  echo "ERROR: dpkg-deb not found. Install it with: apt-get install dpkg" >&2
  exit 1
fi

# ── Create dist directory ─────────────────────────────────────────────────────
mkdir -p "$DIST_DIR"

# ── Set DEBIAN script permissions (must be 755) ───────────────────────────────
echo "[INFO] Setting DEBIAN script permissions..."
chmod 755 "$DEB_ROOT/DEBIAN/postinst"
chmod 755 "$DEB_ROOT/DEBIAN/prerm"

# ── Set binary permissions (755) ─────────────────────────────────────────────
echo "[INFO] Setting binary permissions..."
if [[ -d "$DEB_ROOT/usr/local/bin" ]]; then
  find "$DEB_ROOT/usr/local/bin" -type f -exec chmod 755 {} \;
fi

# ── Set config file permissions (644) ────────────────────────────────────────
echo "[INFO] Setting config file permissions..."
if [[ -d "$DEB_ROOT/etc/infrasense" ]]; then
  find "$DEB_ROOT/etc/infrasense" -type f -exec chmod 644 {} \;
  find "$DEB_ROOT/etc/infrasense" -type d -exec chmod 755 {} \;
fi

# ── Set systemd unit permissions (644) ───────────────────────────────────────
if [[ -d "$DEB_ROOT/etc/systemd" ]]; then
  find "$DEB_ROOT/etc/systemd" -type f -exec chmod 644 {} \;
  find "$DEB_ROOT/etc/systemd" -type d -exec chmod 755 {} \;
fi

# ── Set migrations permissions (644) ─────────────────────────────────────────
if [[ -d "$DEB_ROOT/usr/local/share/infrasense/migrations" ]]; then
  find "$DEB_ROOT/usr/local/share/infrasense/migrations" -type f -exec chmod 644 {} \;
  find "$DEB_ROOT/usr/local/share/infrasense/migrations" -type d -exec chmod 755 {} \;
fi

# ── Update Installed-Size in control file ────────────────────────────────────
echo "[INFO] Calculating installed size..."
INSTALLED_SIZE=$(du -sk "$DEB_ROOT" | awk '{print $1}')
sed -i "s/^Installed-Size:.*/Installed-Size: $INSTALLED_SIZE/" "$DEB_ROOT/DEBIAN/control"

# ── Build the package ─────────────────────────────────────────────────────────
echo "[INFO] Running dpkg-deb --build..."
dpkg-deb --root-owner-group --build "$DEB_ROOT" "$OUTPUT_PKG"

echo ""
echo "[INFO] Package built: $OUTPUT_PKG"

# ── Validate the package ──────────────────────────────────────────────────────
echo ""
echo "[INFO] Package info:"
dpkg-deb --info "$OUTPUT_PKG"

echo ""
echo "[INFO] Package contents:"
dpkg-deb --contents "$OUTPUT_PKG"

echo ""
echo "[SUCCESS] Build complete: $OUTPUT_PKG"
