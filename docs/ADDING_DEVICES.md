# Adding Devices

This guide explains how to add and manage devices (servers, UPS, etc.) in InfraSense.

## Adding a Device via the Dashboard

1. Log in to the InfraSense dashboard (`http://localhost`).
2. Navigate to the **Devices** page from the sidebar.
3. Click the **Add Device** button in the top right corner.
4. Fill out the device details:
   - **Hostname**: The DNS name or friendly name of the device (e.g., `db-server-01`).
   - **IP Address**: The primary OS IP address (e.g., `10.0.0.50`).
   - **BMC IP Address**: (Optional but recommended) The IP of the out-of-band management interface (iDRAC, iLO, etc.).
   - **Location**: Physical location (e.g., `Rack 4, U12`).
5. Click **Save**.

## Configuring Credentials

To collect hardware telemetry, InfraSense needs credentials to authenticate with the device's BMC (Baseboard Management Controller).

1. On the Devices list, click on the device you just created to open its details page.
2. In the "Credentials" section, click **Set Credentials**.
3. Select the appropriate protocol (Redfish or IPMI).
4. Enter the username and password for the BMC.
5. Click **Save Credentials**. *(These are encrypted before being stored in the database).*

### Testing the Connection

After saving credentials, click **Test Connection** on the device details page. InfraSense will attempt to connect to the BMC and verify the credentials. You should see a success message if everything is configured correctly.

## Syncing Hardware Inventory

Once the connection is verified, you can manually trigger a full hardware inventory sync.

1. On the device details page, click the **Sync** button in the header.
2. InfraSense will pull data about CPUs, Memory, Storage, and PCIe devices.
3. Refresh the page after a few seconds to see the populated inventory tabs.

## Automatic Monitoring

If the connection test passes, the `redfish-collector` (or appropriate collector) will automatically start polling the device for metrics (temperature, fan speed, power consumption) based on its configured interval.

You can view these live metrics on the **Metrics** and **Sensors** tabs of the device details page.

## Adding Devices via API

You can also automate device addition using the REST API:

```bash
curl -X POST http://localhost/api/v1/devices \
  -H "Authorization: Bearer <your_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "hostname": "new-server-02",
    "ip_address": "10.0.0.52",
    "bmc_ip_address": "10.0.0.152",
    "vendor": "dell"
  }'
```

See the [API Reference](API.md) for more details.
