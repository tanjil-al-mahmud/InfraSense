# SNMP Collector

The SNMP Collector is a microservice that polls UPS devices via SNMP protocol and pushes metrics to VictoriaMetrics.

## Features

- **Device List Loading**: Loads SNMP device targets from PostgreSQL on startup
- **Hot-Reload**: Automatically reloads device list every 5 minutes without restart
- **Concurrent Polling**: Polls devices concurrently using a goroutine pool (max 100 concurrent)
- **SNMP v2c and v3 Support**: Supports both SNMP v2c (community string) and v3 (authentication and privacy)
- **Timeout Handling**: 10-second timeout per device (configurable)
- **Health Endpoints**: Exposes `/health` and `/metrics` HTTP endpoints
- **Status Tracking**: Updates collector status in PostgreSQL after each poll

## Collected Metrics

The collector polls the following UPS metrics:

- `infrasense_snmp_ups_battery_charge` - Battery charge percentage
- `infrasense_snmp_ups_input_voltage` - Input voltage in volts
- `infrasense_snmp_ups_output_voltage` - Output voltage in volts
- `infrasense_snmp_ups_load_percent` - Load percentage
- `infrasense_snmp_ups_runtime_minutes` - Estimated runtime remaining in minutes
- `infrasense_snmp_ups_battery_status` - Battery status code

## Configuration

Configuration is loaded from `config.yml`:

```yaml
database:
  host: localhost
  port: 5432
  database: infrasense
  user: infrasense
  password: changeme
  ssl_mode: disable

metrics:
  victoriametrics_url: http://localhost:8428/api/v1/write
  batch_size: 1000
  batch_timeout: 10s

collector:
  polling_interval: 60s
  device_reload_interval: 5m
  max_concurrent: 100
  timeout: 10s

logging:
  level: info
  format: json

health_server:
  port: 8080
```

Environment variables can override configuration:
- `DB_HOST` - Database host
- `DB_PASSWORD` - Database password
- `VICTORIAMETRICS_URL` - VictoriaMetrics URL

## Building

```bash
go build -o snmp-collector ./cmd/main.go
```

## Running

```bash
./snmp-collector
```

## Docker

Build the Docker image:

```bash
docker build -t infrasense/snmp-collector:latest .
```

Run the container:

```bash
docker run -d \
  -e DB_HOST=postgres \
  -e DB_PASSWORD=secret \
  -e VICTORIAMETRICS_URL=http://victoriametrics:8428/api/v1/write \
  -p 8080:8080 \
  infrasense/snmp-collector:latest
```

## Health Check

Check collector health:

```bash
curl http://localhost:8080/health
```

Response:
```json
{"status":"healthy","device_count":5}
```

## Requirements

- Go 1.21+
- PostgreSQL 15+
- VictoriaMetrics
- SNMP-enabled UPS devices (APC, Eaton, etc.)
