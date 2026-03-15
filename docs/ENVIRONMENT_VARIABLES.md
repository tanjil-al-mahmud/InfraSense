# Environment Variables Reference

The `.env` file located in the root of the InfraSense project configures the entire stack.

Below is a complete description of all supported environment variables.

## Core System Settings

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| `DOMAIN` | `localhost` | The primary domain or IP address used to access the dashboard. Used by Nginx and Grafana for routing. |
| `ENCRYPTION_KEY` | *(Must be generated)*| **REQUIRED.** A 32-byte base64-encoded string used to encrypt sensitive data (like BMC passwords) in the database. DO NOT lose this key. |

*To generate a secure encryption key on Linux/macOS:*
```bash
openssl rand -base64 32
```

## Database (PostgreSQL)

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| `DB_PASSWORD` | `InfraSense@2024` | The password for the `infrasense` database user. **Change this for production.** |
| `DB_HOST` | `postgres` | The hostname of the database server. Leave as `postgres` if using the standard Docker Compose setup. |

## Application Security

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| `JWT_SECRET` | `InfraSenseJWTSecret2024XYZ12345` | The secret key used to sign JWT authentication tokens. **Change this for production.** |
| `GRAFANA_ADMIN_PASSWORD` | `admin` | The initial admin password for the Grafana dashboard. |

## Notification Service

These variables configure how alerts are sent via email, Slack, or Telegram.

### Email (SMTP)

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| `SMTP_HOST` | *(empty)* | The hostname of your SMTP server (e.g., `smtp.sendgrid.net`). |
| `SMTP_PORT` | `587` | The port for your SMTP server (typically 587 or 465). |
| `SMTP_USER` | *(empty)* | Your SMTP username. |
| `SMTP_PASS` | *(empty)* | Your SMTP password. |
| `SMTP_FROM` | `alerts@infrasense.local` | The "From" address used for alert emails. |

### Slack

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| `SLACK_WEBHOOK_URL`| *(empty)* | The Incoming Webhook URL provided by Slack for sending alerts to a specific channel. |

### Telegram

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| `TELEGRAM_BOT_TOKEN`| *(empty)* | The API token for your Telegram Bot (obtained from @BotFather). |
| `TELEGRAM_CHAT_ID` | *(empty)* | The ID of the Telegram user or group chat to send alerts to. |

## Advanced / Internal Settings

*(Usually do not need to be changed in a standard Docker Compose deployment)*

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| `VICTORIAMETRICS_URL`| `http://victoriametrics:8428/api/v1/write` | The internal endpoint collectors use to push metrics. |
| `LOG_LEVEL` | `info` | Control verbosity of backend logs (`debug`, `info`, `warn`, `error`). |
| `LOG_FORMAT` | `json` | Format of backend logs (`json` or `text`). |
