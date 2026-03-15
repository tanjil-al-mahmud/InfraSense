# InfraSense REST API Reference

The InfraSense REST API allows you to manage devices, alert rules, users, and settings. All endpoints use JSON.

**Base URL**: `http://localhost/api/v1`

## Authentication

All API endpoints (except `/auth/login`) require a valid JWT token.

```bash
# Get a token
curl -X POST http://localhost/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "yourpassword"}'
```

The response contains the token:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "abc...",
    "username": "admin",
    "role": "admin"
  }
}
```

Include the token in the `Authorization` header for all other requests:

```bash
curl -H "Authorization: Bearer <your_token>" http://localhost/api/v1/devices
```

## Devices

Manage monitored servers and equipment.

### List Devices
`GET /devices`

**Query Parameters:**
* `page` (optional): Page number (default: 1)
* `page_size` (optional): Items per page (default: 20)
* `status` (optional): Filter by status (`healthy`, `warning`, `critical`, `unavailable`)

```bash
curl -H "Authorization: Bearer <token>" 'http://localhost/api/v1/devices?page=1&page_size=10'
```

### Get Device
`GET /devices/:id`

Returns detailed information about a specific device.

### Create Device
`POST /devices`

```json
// Request Body
{
  "hostname": "server-01.local",
  "ip_address": "192.168.1.50",
  "bmc_ip_address": "192.168.1.51",
  "protocol": "redfish",
  "vendor": "dell"
}
```

### Update Device
`PUT /devices/:id`

Update an existing device's details (e.g., changing the hostname or location).

### Delete Device
`DELETE /devices/:id`

Remove a device and all associated data.

## Device Telemetry & Inventory

Access deep hardware data for a specific device.

### Get Hardware Inventory
`GET /devices/:id/inventory`

Returns CPUs, memory modules, storage controllers, physical drives, network interfaces, and PCIe devices.

### Get Metrics
`GET /devices/:id/metrics`

Returns recent time-series data for temperature, fan speed, and power consumption.

### Get Live Telemetry Stream (SSE)
`GET /devices/:id/stream?token=<token>`

Connects to a Server-Sent Events (SSE) stream for real-time sensor updates.

## Alerts

Manage triggered alerts.

### List Active Alerts
`GET /alerts`

Returns all currently active alerts.

### Get Alert History
`GET /alerts/history`

Returns a historical log of past alerts.

### Acknowledge Alert
`POST /alerts/:fingerprint/acknowledge`

Mark an alert as acknowledged, temporarily silencing notifications.

### Resolve Alert
`POST /alerts/:fingerprint/resolve`

Manually mark an alert as resolved.

## Alert Rules

Configure the thresholds that trigger alerts.

### List Alert Rules
`GET /alert-rules`

Returns all configured alert rules.

### Get Alert Rule
`GET /alert-rules/:id`

Returns details for a specific rule.

### Create Alert Rule
`POST /alert-rules`

```json
// Request Body
{
  "name": "High CPU Temperature",
  "metric_name": "cpu_temperature",
  "operator": "gt",
  "threshold": 80,
  "severity": "critical",
  "enabled": true
}
```

### Update Alert Rule
`PUT /alert-rules/:id`

Modify an existing alert rule's thresholds or severity.

### Delete Alert Rule
`DELETE /alert-rules/:id`

Remove an alert rule.

## Users

Manage dashboard users and access control (Admin only).

### List Users
`GET /users`

Returns all registered users.

### Create User
`POST /users`

```json
// Request Body
{
  "username": "operator1",
  "password": "securepassword",
  "role": "operator"
}
```

### Delete User
`DELETE /users/:id`

Remove a user account.
