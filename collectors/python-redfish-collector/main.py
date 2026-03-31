#!/usr/bin/env python3
import time
import signal
import sys
import os
import structlog
import schedule
from concurrent.futures import ThreadPoolExecutor, as_completed
from database import load_redfish_devices, update_device_status, save_hardware_events
from metrics import push_metrics_to_victoriametrics
from redfish_collector import RedfishCollector

log = structlog.get_logger()

POLL_INTERVAL = int(os.getenv('POLL_INTERVAL_SECONDS', 60))
running = True


def signal_handler(sig, frame):
    global running
    log.info("shutdown_signal_received")
    running = False
    sys.exit(0)


signal.signal(signal.SIGTERM, signal_handler)
signal.signal(signal.SIGINT, signal_handler)


def poll_device(device):
    """Poll a single Redfish device"""
    device_id = device['id']
    hostname = device['hostname']
    bmc_ip = device['bmc_ip']

    if not bmc_ip or not device['username']:
        log.warning("device_missing_credentials",
                    device_id=device_id, hostname=hostname)
        return

    log.info("polling_redfish_device",
             device_id=device_id,
             hostname=hostname,
             bmc_ip=bmc_ip)

    try:
        collector = RedfishCollector(device)
        metrics, events = collector.collect_all()

        if metrics:
            push_metrics_to_victoriametrics(metrics)
            log.info("redfish_metrics_pushed",
                     device_id=device_id,
                     hostname=hostname,
                     count=len(metrics))

        if events:
            save_hardware_events(device_id, hostname, events, source_protocol='redfish')

        update_device_status(device_id, 'online')

    except Exception as e:
        error_msg = str(e)
        log.error("redfish_poll_failed",
                  device_id=device_id,
                  hostname=hostname,
                  bmc_ip=bmc_ip,
                  error=error_msg)
        update_device_status(device_id, 'offline', error_msg)


def run_poll_cycle(devices):
    """Poll all devices concurrently - max 20 at a time"""
    if not devices:
        log.info("no_devices_to_poll")
        return

    log.info("redfish_poll_cycle_start", device_count=len(devices))

    with ThreadPoolExecutor(max_workers=20) as executor:
        futures = {executor.submit(poll_device, device): device for device in devices}
        for future in as_completed(futures):
            try:
                future.result()
            except Exception as e:
                device = futures[future]
                log.error("poll_thread_failed",
                          hostname=device['hostname'], error=str(e))

    log.info("redfish_poll_cycle_complete", device_count=len(devices))


def poll_job():
    try:
        devices = load_redfish_devices()
        run_poll_cycle(devices)
    except Exception as e:
        log.error("poll_job_failed", error=str(e))


def main():
    log.info("redfish_collector_starting",
             poll_interval=POLL_INTERVAL,
             version="2.0.0",
             protocol="redfish")

    poll_job()

    schedule.every(POLL_INTERVAL).seconds.do(poll_job)

    log.info("redfish_collector_running",
             poll_interval_seconds=POLL_INTERVAL)

    while running:
        schedule.run_pending()
        time.sleep(1)


if __name__ == '__main__':
    main()
