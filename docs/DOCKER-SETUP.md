# Docker Setup Guide

This guide describes how to run InfraSense using Docker and Docker Compose. This is the recommended deployment method for most users as it guarantees a consistent environment across platforms.

## Prerequisites

- Docker Engine (v20.10.0+ recommended)
- Docker Compose (v2.0.0+ recommended)

## Quick Start

1. Clone the InfraSense repository.
2. Navigate to the project root and duplicate `.env.example`:
   ```bash
   cp .env.example .env
   ```
3. Update `.env` with your desired configuration (e.g., passwords, admin credentials).
4. Build and start the services:
   ```bash
   cd deploy
   docker compose --env-file ../.env up -d --build
   ```

## Managing Services

- **View Logs**:
  ```bash
  docker compose logs -f
  ```
  *(Add a service name like `api-server` or `frontend` to view specific logs)*

- **Stop Services**:
  ```bash
  docker compose down
  ```

- **Restart Services**:
  ```bash
  docker compose restart
  ```

- **Update Images (pull latest)**:
  ```bash
  docker compose pull
  docker compose up -d
  ```

## Service Overview

The `docker-compose.yml` file defines the following services:

| Service | Description | Port |
| :--- | :--- | :--- |
| `postgres` | Central database for configuration and device inventory | 5432 (Internal) |
| `victoriametrics` | Time-series database for storing hardware metrics | 8428 |
| `loki` | Log aggregation system | 3100 (Internal) |
| `promtail` | Log shipper sending logs to Loki | N/A |
| `prometheus` | Alerting engine evaluating rules against metrics | 9090 |
| `alertmanager` | Alert routing and deduplication | 9093 |
| `api-server` | Go backend managing devices, users, and API requests | 8080 (Internal) |
| `notification-service` | Handles sending alert notifications via Email/Slack/Telegram | 8080 (Internal) |
| `redfish-collector` | Polls Redfish APIs and writes to VictoriaMetrics | 8081 (Internal) |
| `frontend` | React dashboard UI | 80 (Internal) |
| `nginx` | Reverse proxy routing traffic to frontend, API, and Grafana | 80 (External) |
| `grafana` | Visualization and dashboard creation tool | 3000 (Internal) |

## Data Persistence

All data is stored in named Docker volumes to ensure persistence across restarts.

| Volume Name | Target Path | Description |
| :--- | :--- | :--- |
| `postgres_data` | `/var/lib/postgresql/data` | Relational database storage |
| `victoriametrics_data`| `/victoria-metrics-data` | Time-series metrics storage |
| `loki_data` | `/loki` | Log data storage |
| `alertmanager_data` | `/alertmanager` | Alert state and silences |
| `prometheus_data` | `/prometheus` | Prometheus state data |
| `grafana_data` | `/var/lib/grafana` | Grafana configuration and dashboards |

*To completely wipe all data and start fresh, run `docker compose down -v`.*
