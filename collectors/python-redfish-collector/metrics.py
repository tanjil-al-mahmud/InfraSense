import requests
import os
import structlog

log = structlog.get_logger()
VICTORIAMETRICS_URL = os.getenv('VICTORIAMETRICS_URL', 'http://victoriametrics:8428')


def format_prometheus_line(metric):
    """Format metric as Prometheus text format"""
    labels = metric.get('labels', {})
    label_str = ','.join([f'{k}="{v}"' for k, v in labels.items()])
    if label_str:
        label_str = '{' + label_str + '}'
    timestamp = metric.get('timestamp', '')
    ts_str = f' {timestamp}' if timestamp else ''
    return f"{metric['name']}{label_str} {metric['value']}{ts_str}"


def push_metrics_to_victoriametrics(metrics):
    """Push metrics batch to VictoriaMetrics"""
    if not metrics:
        return
    lines = [format_prometheus_line(m) for m in metrics]
    payload = '\n'.join(lines)
    try:
        response = requests.post(
            f"{VICTORIAMETRICS_URL}/api/v1/import/prometheus",
            data=payload,
            headers={'Content-Type': 'text/plain'},
            timeout=10
        )
        response.raise_for_status()
    except Exception as e:
        log.error("victoriametrics_push_failed",
                  metric_count=len(metrics),
                  error=str(e))
