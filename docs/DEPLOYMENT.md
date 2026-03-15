# Deployment Guide

This guide covers advanced deployment topics, including securing your installation, configuring TLS/SSL, tuning performance, and setting up notifications.

## Securing Your Installation

Out of the box, InfraSense connects over HTTP on port 80. For any non-local deployment, **you must enable HTTPS/TLS**.

### Option A: Using Certbot (Let's Encrypt) with Nginx

If you run InfraSense natively (without Docker), you can secure it easily with Certbot. Ensure your domain is pointed at your server's IP.

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d monitor.yourdomain.com
```

Certbot will automatically configure the Nginx block created by the installer script and set up an auto-renewal cronjob.

### Option B: Reverse Proxy via Docker

If deploying via Docker, the simplest approach is to use a reverse proxy like [Caddy](https://caddyserver.com/) or [Traefik](https://traefik.io/) in front of the default Nginx container.

Example Caddy configuration (`Caddyfile`):

```caddy
monitor.yourdomain.com {
    reverse_proxy localhost:80
}
```

Caddy handles SSL certificate generation and renewal automatically.

## Performance Tuning

InfraSense uses VictoriaMetrics for time-series data storage, which is highly optimized. However, for large deployments (>1000 devices), consider the following:

### 1. Database Tuning
Modify the `.env` settings for PostgreSQL:
- Increase `POSTGRES_MAX_CONNECTIONS` if you experience connection limits during high-concurrency polling.
- Ensure PostgreSQL has sufficient allocated `shared_buffers` and `work_mem`.

### 2. Collection Intervals
The `redfish-collector` polling interval limits the load on target BMCs. The default is typically 30s-60s. Adjusting this in your custom configuration can reduce overhead at the cost of less granular data.

### 3. Metric Retention
VictoriaMetrics stores data efficiently, but a long retention period consumes disk space.
In `deploy/docker-compose.yml`, locate the `victoriametrics` command arguments and adjust `--retentionPeriod`:
```yaml
    command:
      - "--storageDataPath=/victoria-metrics-data"
      - "--retentionPeriod=90d" # Change this to 30d or 7d if needed
```

## Configuring Notifications

The `notification-service` requires external credentials to dispatch alerts. Define these in your `.env` file before starting the application:

### Slack
To send alerts to a Slack channel:
1. Create a [Slack App](https://api.slack.com/apps) in your workspace.
2. Enable Incoming Webhooks and create a new webhook URL for your target channel.
3. Update `.env`:
   ```env
   SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
   ```

### Telegram
To send alerts to a Telegram chat:
1. Message `@BotFather` on Telegram to create a new bot and obtain an access token.
2. Add the bot to your desired group chat (or message it directly) and obtain the `CHAT_ID`.
3. Update `.env`:
   ```env
   TELEGRAM_BOT_TOKEN="your_bot_token"
   TELEGRAM_CHAT_ID="your_chat_id"
   ```

### Email (SMTP)
To send alerts via email:
1. Obtain SMTP credentials from your provider (e.g., SendGrid, Mailgun, or your company server).
2. Update `.env`:
   ```env
   SMTP_HOST="smtp.example.com"
   SMTP_PORT="587"
   SMTP_USER="alerts@example.com"
   SMTP_PASS="your_smtp_password"
   SMTP_FROM="InfraSense Alerts <alerts@example.com>"
   ```

Restart the `notification-service` after modifying these variables for them to take effect.
