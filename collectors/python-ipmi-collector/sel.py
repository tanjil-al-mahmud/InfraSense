import structlog
from datetime import datetime

log = structlog.get_logger()

SEVERITY_MAP = {
    'ok': 'info',
    'warning': 'warning',
    'critical': 'critical',
    'failed': 'critical',
    'non-recoverable': 'critical',
    'non-critical': 'warning',
}


def collect_sel_events(ipmisession, device_id, hostname):
    """Collect System Event Log from device"""
    events = []
    try:
        sel = ipmisession.get_sel()
        for entry in sel:
            severity_raw = str(entry.get('severity', 'ok')).lower()
            severity = SEVERITY_MAP.get(severity_raw, 'info')

            events.append({
                'id': str(entry.get('id', '')),
                'severity': severity,
                'message': entry.get('message', str(entry)),
                'type': entry.get('type', 'system'),
                'component': 'system',
                'timestamp': datetime.utcnow()
            })

    except Exception as e:
        log.warning("sel_fetch_failed",
                    device_id=device_id,
                    hostname=hostname,
                    error=str(e))

    return events
