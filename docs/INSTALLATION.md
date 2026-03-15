# Installation Guide

You can install InfraSense using Docker (Recommended) or natively on Ubuntu or RedHat/Rocky Linux using packaged installers.

## Option 1: Docker Compose (All Platforms)

This is the fastest and most reliable method. See the full [Docker Setup Guide](DOCKER-SETUP.md) for details.

```bash
git clone https://github.com/tanjil-al-mahmud/infrasense.git
cd infrasense
cp .env.example .env
cd deploy
docker compose --env-file ../.env up -d --build
```

## Option 2: Native Installation (Ubuntu 24.04)

We provide a bundled native installation script that automatically installs dependencies, builds the project, and configures systemd services.

*Ensure you run this script with `sudo` or as the `root` user.*

```bash
# 1. Clone the repository
git clone https://github.com/tanjil-al-mahmud/infrasense.git
cd infrasense

# 2. Make the script executable
chmod +x scripts/install-ubuntu-24.04.sh

# 3. Run the installer
sudo ./scripts/install-ubuntu-24.04.sh
```

**What the installer does:**

- Installs required packages (Go 1.22+, Node.js 20+, PostgreSQL, Nginx)
- Creates a dedicated `infrasense` system user and group
- Builds the Go API backend and collector binaries
- Builds the React frontend
- Configures the PostgreSQL database with the required schema
- Creates systemd service files (`infrasense-api`, `infrasense-redfish-collector`, `infrasense-notification-service`)
- Configures Nginx as a reverse proxy
- Starts and enables all services

After installation, the dashboard will be available at `http://localhost`. 

*(Wait a few minutes for the script to download dependencies and compile everything.)*

## Option 3: Package Installation (.deb / .rpm)

*Coming soon in a future release.* Pre-compiled `.deb` and `.rpm` packages will be available on the GitHub Releases page to simplify deployment without requiring Go or Node.js on the target machine.
