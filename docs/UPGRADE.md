# Upgrade Guide

This guide details how to safely upgrade InfraSense to a new version.

## Upgrade via Docker Compose (Recommended)

When a new version is released on GitHub (e.g., matching a new Docker image tag), you can upgrade your installation with a few commands with minimal downtime.

### Step-by-Step

1. **Navigate to the deployment directory:**
   ```bash
   cd ~/infrasense/deploy
   ```

2. **Pull the latest changes from the repository (optional):**
   *(If the `docker-compose.yml` file itself was updated)*
   ```bash
   git fetch origin
   git pull origin main
   ```

3. **Pull the newest Docker images:**
   ```bash
   docker compose pull
   ```
   *This downloads the updated containers for Grafana, Prometheus, etc.*

4. **Rebuild local images and restart services:**
   ```bash
   docker compose --env-file ../.env up -d --build
   ```
   *This step rebuilds the `api-server`, `frontend`, `redfish-collector`, and `notification-service` using the latest code from the repository, and cleanly restarts the containers.*

5. **Verify the upgrade:**
   Check that all services are marked as "Up" and "healthy":
   ```bash
   docker compose ps
   ```

### Database Migrations

The Go backend application (`api-server`) automatically runs required database schema migrations upon startup. There is no manual database patching required during a standard version upgrade. 

*If a migration fails, the `api-server` container will crash and report the error in the logs. In these rare cases, refer to the release notes for specific instructions.*

## Upgrading Native Installations (Ubuntu)

If you installed InfraSense natively using the provided `scripts/install-ubuntu-24.04.sh` script, you need to pull the new code and recompile the binaries.

1. **Stop the services:**
   ```bash
   sudo systemctl stop infrasense-api infrasense-redfish-collector infrasense-notification-service
   ```

2. **Pull the latest code:**
   ```bash
   cd ~/infrasense
   git pull origin main
   ```

3. **Re-run the installer script:**
   *The installer script is idempotent; it safely updates binaries and restarts services without erasing your database.*
   ```bash
   sudo ./scripts/install-ubuntu-24.04.sh
   ```

4. **Verify status:**
   ```bash
   sudo systemctl status infrasense-api
   ```
