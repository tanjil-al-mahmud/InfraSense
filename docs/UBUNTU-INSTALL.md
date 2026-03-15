# Ubuntu Installation Guide

This guide provides instructions for installing InfraSense natively on Ubuntu 24.04 LTS.

## Prerequisites

- Ubuntu 24.04 LTS
- Root or sudo access
- At least 2GB of RAM
- 10GB of free disk space

## Automated Installation

The easiest way to install InfraSense is using the provided installation script:

```bash
cd scripts
sudo chmod +x install-ubuntu.sh
sudo ./install-ubuntu.sh
```

The script will:
1. Update system packages
2. Install required dependencies (PostgreSQL, Redis, Go, Node.js)
3. Set up the database
4. Build the backend and frontend
5. Configure systemd services

## Manual Installation

If you prefer to install manually, please refer to [INSTALLATION.md](INSTALLATION.md) for detailed steps.

## Post-Installation

Once installed, InfraSense will be available at `http://localhost:3000` (or the IP address of your server).

The following services will be running:
- `infrasense-backend`
- `infrasense-frontend`
- `infrasense-notification`
- `infrasense-ipmi-collector`
- `infrasense-snmp-collector`

## Troubleshooting

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common issues and solutions.
