#!/bin/sh
# Generate self-signed TLS certificate for development use.
# Usage: ./generate-certs.sh [output-dir]
#
# Outputs cert.pem and key.pem to the specified directory (default: ./ssl).

set -e

OUTPUT_DIR="${1:-./ssl}"

mkdir -p "$OUTPUT_DIR"

echo "Generating self-signed certificate in $OUTPUT_DIR ..."

openssl req -x509 \
    -newkey rsa:4096 \
    -keyout "$OUTPUT_DIR/key.pem" \
    -out "$OUTPUT_DIR/cert.pem" \
    -days 365 \
    -nodes \
    -subj "/C=US/ST=Dev/L=Dev/O=InfraSense/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

echo "Certificate generated:"
echo "  Certificate: $OUTPUT_DIR/cert.pem"
echo "  Private key: $OUTPUT_DIR/key.pem"
echo ""
echo "NOTE: This is a self-signed certificate for development only."
echo "      Use a CA-signed certificate in production."
