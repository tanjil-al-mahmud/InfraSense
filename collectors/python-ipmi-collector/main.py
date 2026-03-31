#!/usr/bin/env python3
import time
import signal
import sys
import os
import structlog
import schedule
from database import load_ipmi_devices
from collector import run_poll_cycle

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


def poll_job():
    try:
        devices = load_ipmi_devices()
        run_poll_cycle(devices)
    except Exception as e:
        log.error("poll_job_failed", error=str(e))


def main():
    log.info("ipmi_collector_starting",
             poll_interval=POLL_INTERVAL,
             version="2.0.0",
             protocol="pyghmi")

    # Run immediately on startup
    poll_job()

    # Schedule recurring polls
    schedule.every(POLL_INTERVAL).seconds.do(poll_job)

    log.info("ipmi_collector_running",
             poll_interval_seconds=POLL_INTERVAL)

    while running:
        schedule.run_pending()
        time.sleep(1)


if __name__ == '__main__':
    main()
