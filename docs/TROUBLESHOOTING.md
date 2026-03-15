# Troubleshooting Guide

This guide covers common issues and their solutions when running InfraSense.

## General Diagnostics

Before diving into specific components, always check the status of all Docker containers:

```bash
docker compose ps
```

If a container is repeatedly restarting or marked as "unhealthy," view its logs:

```bash
docker compose logs -f <service-name>
# Example: docker compose logs -f api-server
```

## Common Issues

### 1. "Test Connection Failed: Unauthorized"

**Symptoms:**
When adding a new device, clicking "Test Connection" returns an authentication error, even though you know the password is correct.

**Possible Causes:**
- **Incorrect Protocol:** Ensure you selected the correct protocol (Redfish vs IPMI). Some older servers only support IPMI.
- **BMC Account Lockout:** Many BMCs (like Dell iDRAC) lock accounts for 5 minutes after multiple failed login attempts. Try logging in directly via the BMC web interface to check the status.
- **Privilege Level:** The account must have at least "Operator" privileges to read hardware inventory and sensor data.

### 2. "Device Sync Failed: Timeout"

**Symptoms:**
The manual hardware sync takes a long time and eventually fails with a timeout error.

**Possible Causes:**
- **Network Routing:** Ensure the server hosting InfraSense has a route to the BMC network management IP.
- **BMC Overload:** Management controllers have very slow processors. Wait 5 minutes and try the sync again.
- **Redfish Service Disabled:** Log into the BMC web UI and ensure the "Redfish REST API" service is enabled.

### 3. Missing Real-time Metrics or Empty Graphs

**Symptoms:**
The dashboard says the server is "Healthy," but the metrics graphs (Power, Temp, Fan) are completely empty or stop updating.

**Troubleshooting Steps:**
1. Check if the `redfish-collector` is running:
   ```bash
   docker compose ps redfish-collector
   ```
2. Check the collector logs for connection errors:
   ```bash
   docker compose logs --tail=100 redfish-collector
   ```
3. Ensure `VICTORIAMETRICS_URL` in `.env` matches the internal address: `http://victoriametrics:8428/api/v1/write`.
4. Check if VictoriaMetrics is accepting writes:
   ```bash
   docker compose logs --tail=50 victoriametrics
   ```

### 4. Alerts Are Not Firing

**Symptoms:**
A server goes offline or temperature spikes, but no notification is sent to Slack/Email.

**Troubleshooting Steps:**
1. Ensure the alert rule is enabled in the InfraSense UI.
2. Check if Prometheus sees the alert condition as firing: Navigate to `http://<your-server-ip>:9090/alerts` (you may need to temporarily expose port 9090 in `docker-compose.yml`).
3. Check AlertManager logs to see if it received the alert from Prometheus:
   ```bash
   docker compose logs alertmanager
   ```
4. Finally, verify the `notification-service` is correctly configured:
   ```bash
   docker compose logs notification-service
   ```
   *Look for errors regarding SMTP authentication or invalid Webhook URLs.*

### 5. Backend "500 Internal Server Error"

**Symptoms:**
The UI fails to load device lists or displays a generic 500 error.

**Possible Causes:**
- **Database Connection:** The backend lost connection to PostgreSQL. Check the Postgres logs:
  `docker compose logs postgres`
- **Missing Encryption Key:** If `ENCRYPTION_KEY` is missing from the `.env` file, the backend cannot decrypt stored credentials and will crash upon checking connections. Generate a key and restart the `api-server`.

## Still Stuck?

If you cannot resolve the issue, open a detailed bug report on GitHub including:
1. Docker Compose output (`docker compose ps`)
2. Relevant logs (`docker compose logs ...`)
3. The specific make/model of the server failing to report metrics.
