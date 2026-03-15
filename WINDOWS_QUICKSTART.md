# InfraSense — Windows Quick Start Guide

Everything runs in Docker Desktop. No Go, Node, or other tooling required on your machine.

---

## Prerequisites

- [Docker Desktop for Windows](https://www.docker.com/products/docker-desktop/) installed and running
- WSL 2 backend enabled (recommended — set in Docker Desktop → Settings → General)
- PowerShell 5.1+ (built into Windows 10/11)

---

## Step 1 — Clone the project

```powershell
git clone <your-repo-url> infrasense-project
cd infrasense-project\infrasense
```

---

## Step 2 — Start the stack

The `.env` file is already pre-filled with test values. Run from the `infrasense\deploy` directory:

```powershell
cd deploy
docker compose --env-file ..\.env up -d --build
```

The first run builds all images — expect 5–10 minutes. Subsequent starts are fast.

---

## Step 3 — Check all services are healthy

```powershell
docker compose ps
```

All services should show `healthy` or `running`. To watch until everything is up:

```powershell
# Watch status every 5 seconds
while ($true) { docker compose ps; Start-Sleep 5; Clear-Host }
```

Press `Ctrl+C` to stop watching. You can also run the automated check script:

```powershell
cd ..
powershell -ExecutionPolicy Bypass -File scripts\windows-test.ps1
```

---

## Step 4 — Find the admin password

The admin password is `admin123`. The admin user needs to be seeded into the database once after first start:

```powershell
# From the infrasense\deploy directory, after services are healthy:
docker exec -i infrasense-postgres psql -U infrasense -d infrasense `
  -f /dev/stdin < ..\backend\scripts\create_admin.sql
```

Or using the full path approach:

```powershell
Get-Content ..\backend\scripts\create_admin.sql | `
  docker exec -i infrasense-postgres psql -U infrasense -d infrasense
```

Credentials: `admin` / `admin123`

The Grafana admin password is set in `.env`:

```powershell
Select-String "GRAFANA_ADMIN_PASSWORD" .\.env
```

Default: `admin123`

---

## Step 5 — Login to the dashboard

Open your browser and go to:

```
http://localhost/
```

- Frontend dashboard: `http://localhost/`
- Grafana: `http://localhost/grafana/` (user: `admin`, password: `admin123`)
- API health: `http://localhost/api/v1/health`

---

## Step 6 — Push a fake test metric to simulate a real device

VictoriaMetrics is exposed on port 8428. Push a metric using PowerShell:

```powershell
# Push a fake CPU temperature metric
$body = "cpu_temperature{device_id=`"test-device-001`",hostname=`"test-server`"} 72.5"
Invoke-RestMethod -Uri "http://localhost:8428/api/v1/import/prometheus" `
  -Method POST `
  -Body $body `
  -ContentType "text/plain"

# Verify it was stored
Invoke-RestMethod -Uri "http://localhost:8428/api/v1/query?query=cpu_temperature" | ConvertTo-Json
```

---

## Step 7 — Trigger a test alert

First, get a JWT token by logging in:

```powershell
$login = Invoke-RestMethod -Uri "http://localhost/api/v1/auth/login" `
  -Method POST `
  -ContentType "application/json" `
  -Body '{"username":"admin","password":"admin123"}'

$token = $login.token
```

Then create an alert rule:

```powershell
$alertBody = @{
  name        = "Test CPU Alert"
  metric      = "cpu_temperature"
  condition   = ">"
  threshold   = 70.0
  severity    = "warning"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost/api/v1/alert-rules" `
  -Method POST `
  -ContentType "application/json" `
  -Headers @{ Authorization = "Bearer $token" } `
  -Body $alertBody
```

---

## Step 8 — Verify everything is working

```powershell
# API health
Invoke-RestMethod http://localhost/api/v1/health

# VictoriaMetrics health
Invoke-RestMethod http://localhost:8428/health

# Alertmanager health
Invoke-RestMethod http://localhost:9093/-/healthy

# List devices (requires auth token from Step 7)
Invoke-RestMethod -Uri "http://localhost/api/v1/devices" `
  -Headers @{ Authorization = "Bearer $token" }
```

Or just run the automated test script which does all of this:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\windows-test.ps1
```

---

## Stopping the stack

```powershell
cd deploy
docker compose --env-file ..\.env down
```

To also remove all data volumes (full reset):

```powershell
docker compose --env-file ..\.env down -v
```

---

## Troubleshooting

**Port 80 already in use**
```powershell
netstat -ano | findstr ":80"
# Find the PID and stop the process, or change the port in deploy/docker-compose.yml
```

**Build fails for backend/frontend**
```powershell
docker compose build --no-cache api-server
docker compose build --no-cache frontend
```

**Services stuck in "starting" state**
```powershell
docker compose logs api-server --tail 50
docker compose logs postgres --tail 50
```

**Reset everything**
```powershell
docker compose down -v
docker compose up -d --build
```
