import structlog
import base64
import os
import time
from pyghmi.ipmi import command as ipmi_command
from concurrent.futures import ThreadPoolExecutor, as_completed
from database import load_ipmi_devices, update_device_status, save_hardware_events
from metrics import push_metrics_to_victoriametrics
from sel import collect_sel_events

log = structlog.get_logger()


def decrypt_password(encrypted_password):
    """Decrypt password using AES-256-GCM - must match Go encryption"""
    from cryptography.hazmat.primitives.ciphers.aead import AESGCM

    encryption_key = os.getenv('ENCRYPTION_KEY', '').encode()
    if len(encryption_key) != 32:
        return encrypted_password  # return as-is if not encrypted

    try:
        data = base64.b64decode(encrypted_password)
        nonce = data[:12]
        ciphertext = data[12:]
        aesgcm = AESGCM(encryption_key)
        plaintext = aesgcm.decrypt(nonce, ciphertext, None)
        return plaintext.decode('utf-8')
    except Exception as e:
        log.error("password_decrypt_failed", error=str(e))
        return encrypted_password


def poll_device(device):
    """Poll a single IPMI device using pyghmi"""
    device_id = device['id']
    hostname = device['hostname']
    bmc_ip = device['bmc_ip']
    username = device['username']
    port = int(device['port']) if device['port'] else 623

    if not bmc_ip or not username:
        log.warning("device_missing_credentials",
                    device_id=device_id, hostname=hostname)
        return

    password = decrypt_password(device['password_encrypted'])

    log.info("polling_device",
             device_id=device_id,
             hostname=hostname,
             bmc_ip=bmc_ip,
             port=port)

    try:
        ipmisession = ipmi_command.Command(
            bmc=bmc_ip,
            userid=username,
            password=password,
            port=port
        )

        metrics = []
        timestamp = int(time.time() * 1000)

        # Collect sensor data
        try:
            sensors = ipmisession.get_sensor_data()
            for sensor in sensors:
                if sensor.value is None:
                    continue

                sensor_name = sensor.name.strip()
                value = float(sensor.value)

                # Temperature
                if 'temp' in sensor_name.lower() or sensor.type == 'Temperature':
                    metrics.append({
                        'name': 'infrasense_ipmi_temperature_celsius',
                        'value': value,
                        'labels': {
                            'device_id': device_id,
                            'hostname': hostname,
                            'sensor_name': sensor_name
                        },
                        'timestamp': timestamp
                    })

                # Fan speed
                elif 'fan' in sensor_name.lower() or sensor.type == 'Fan':
                    metrics.append({
                        'name': 'infrasense_ipmi_fan_speed_rpm',
                        'value': value,
                        'labels': {
                            'device_id': device_id,
                            'hostname': hostname,
                            'fan_name': sensor_name
                        },
                        'timestamp': timestamp
                    })

                # Voltage
                elif 'volt' in sensor_name.lower() or sensor.type == 'Voltage':
                    metrics.append({
                        'name': 'infrasense_ipmi_voltage_volts',
                        'value': value,
                        'labels': {
                            'device_id': device_id,
                            'hostname': hostname,
                            'sensor_name': sensor_name
                        },
                        'timestamp': timestamp
                    })

                # Power
                elif 'power' in sensor_name.lower() or 'watt' in sensor_name.lower():
                    metrics.append({
                        'name': 'infrasense_ipmi_power_watts',
                        'value': value,
                        'labels': {
                            'device_id': device_id,
                            'hostname': hostname,
                            'sensor_name': sensor_name
                        },
                        'timestamp': timestamp
                    })

        except Exception as e:
            log.error("sensor_collection_failed",
                      device_id=device_id, hostname=hostname, error=str(e))

        # Collect power status
        try:
            power = ipmisession.get_power()
            metrics.append({
                'name': 'infrasense_ipmi_power_state',
                'value': 1 if power.get('powerstate') == 'on' else 0,
                'labels': {
                    'device_id': device_id,
                    'hostname': hostname
                },
                'timestamp': timestamp
            })
        except Exception as e:
            log.warning("power_status_failed",
                        device_id=device_id, hostname=hostname, error=str(e))

        # Push metrics to VictoriaMetrics
        if metrics:
            push_metrics_to_victoriametrics(metrics)
            log.info("metrics_pushed",
                     device_id=device_id,
                     hostname=hostname,
                     count=len(metrics))

        # Collect SEL events
        try:
            events = collect_sel_events(ipmisession, device_id, hostname)
            if events:
                save_hardware_events(device_id, hostname, events, source_protocol='ipmi')
        except Exception as e:
            log.warning("sel_collection_failed",
                        device_id=device_id, hostname=hostname, error=str(e))

        # Update device status to online
        update_device_status(device_id, 'online')

    except Exception as e:
        error_msg = str(e)
        log.error("device_poll_failed",
                  device_id=device_id,
                  hostname=hostname,
                  bmc_ip=bmc_ip,
                  error=error_msg)
        update_device_status(device_id, 'offline', error_msg)


def run_poll_cycle(devices):
    """Poll all devices concurrently - max 50 at a time"""
    if not devices:
        log.info("no_devices_to_poll")
        return

    log.info("poll_cycle_start", device_count=len(devices))

    with ThreadPoolExecutor(max_workers=50) as executor:
        futures = {executor.submit(poll_device, device): device for device in devices}
        for future in as_completed(futures):
            try:
                future.result()
            except Exception as e:
                device = futures[future]
                log.error("poll_thread_failed",
                          hostname=device['hostname'], error=str(e))

    log.info("poll_cycle_complete", device_count=len(devices))
