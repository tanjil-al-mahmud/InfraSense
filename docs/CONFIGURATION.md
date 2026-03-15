# Configuration Guide

InfraSense uses several configuration files to control its various components.

## Environment Variables (`.env`)

The primary way to configure InfraSense is through the `.env` file at the root of the project. This file controls database credentials, API keys, and global settings.

See [Environment Variables Reference](ENVIRONMENT_VARIABLES.md) for a complete list of options.

## Alert Rules

Alert rules define when notifications should be triggered based on collected metrics. These are managed primarily through the InfraSense UI or REST API, not directly in configuration files, as they are stored in the database.

However, the backend translates these into Prometheus rules. If you need to add raw Prometheus rules manually, you can edit `deploy/prometheus/alert_rules.yml`. Note that this is generally discouraged in favor of using the UI.

## Prometheus Configuration (`deploy/prometheus/prometheus.yml`)

This file configures how Prometheus scrapes metrics. While InfraSense automatically configures VictoriaMetrics as the primary data source, you can add additional external targets here if needed.

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'infrasense-collectors'
    static_configs:
      - targets: ['redfish-collector:8081']
        labels:
          component: 'collector'
```

## Grafana Provisioning

Pre-built dashboards and data sources are loaded automatically into Grafana upon startup.

- **Data Sources:** Located in `deploy/grafana/provisioning/datasources/`
- **Dashboards:** Located in `deploy/grafana/dashboards/`

To add a new dashboard permanently:
1. Export the dashboard JSON from Grafana.
2. Save it in the `deploy/grafana/dashboards/` directory.
3. Restart the Grafana container.

## Nginx Reverse Proxy (`deploy/nginx/nginx.conf`)

If you need to change how traffic is routed (e.g., adding custom headers or changing internal ports), edit this file.

It routes `/` to the frontend, `/api` to the backend, and `/grafana` to the Grafana instance.

## Log Aggregation (Loki/Promtail)

- **Loki Config:** `deploy/loki/loki-config.yml` controls log storage and retention.
- **Promtail Config:** `deploy/promtail/` directory contains configuration for how logs are scraped from the Docker daemon and system files before being sent to Loki.
