#!/usr/bin/env python3
"""
SNMP collector using pysnmp v6 hlapi.
Supports APC, Eaton, and generic UPS OIDs (RFC 1628).
"""
import structlog
from pysnmp.hlapi import (
    SnmpEngine, CommunityData, UsmUserData,
    UdpTransportTarget, ContextData,
    ObjectType, ObjectIdentity,
    getCmd,
    usmHMACSHAAuthProtocol, usmAesCfb128Protocol
)

log = structlog.get_logger()

# ── OID maps ───────────────────────────────────────────────────────────────────

# APC UPS OIDs
APC_OIDS = {
    'battery_charge':   '1.3.6.1.4.1.318.1.1.1.2.2.1.0',
    'battery_status':   '1.3.6.1.4.1.318.1.1.1.2.1.1.0',
    'battery_runtime':  '1.3.6.1.4.1.318.1.1.1.2.2.3.0',
    'input_voltage':    '1.3.6.1.4.1.318.1.1.1.3.2.1.0',
    'output_voltage':   '1.3.6.1.4.1.318.1.1.1.4.2.1.0',
    'output_load':      '1.3.6.1.4.1.318.1.1.1.4.2.3.0',
    'output_current':   '1.3.6.1.4.1.318.1.1.1.4.2.4.0',
}

# Eaton UPS OIDs
EATON_OIDS = {
    'battery_charge':   '1.3.6.1.4.1.534.1.2.4.0',
    'battery_runtime':  '1.3.6.1.4.1.534.1.2.1.0',
    'input_voltage':    '1.3.6.1.4.1.534.1.3.4.1.2.1',
    'output_voltage':   '1.3.6.1.4.1.534.1.4.4.1.2.1',
    'output_load':      '1.3.6.1.4.1.534.1.4.4.1.5.1',
}

# Generic UPS OIDs (RFC 1628)
GENERIC_UPS_OIDS = {
    'battery_charge':   '1.3.6.1.2.1.33.1.2.4.0',
    'battery_runtime':  '1.3.6.1.2.1.33.1.2.3.0',
    'input_voltage':    '1.3.6.1.2.1.33.1.3.3.1.3.1',
    'output_voltage':   '1.3.6.1.2.1.33.1.4.4.1.2.1',
    'output_load':      '1.3.6.1.2.1.33.1.4.4.1.5.1',
}


def get_oids_for_device(device_type: str) -> dict:
    """Return the correct OID map based on device_type string."""
    dt = device_type.lower()
    if 'apc' in dt:
        return APC_OIDS
    elif 'eaton' in dt:
        return EATON_OIDS
    else:
        return GENERIC_UPS_OIDS


def _build_auth(device: dict):
    """Build the pysnmp auth object from device config."""
    version = device.get('snmp_version', 'v2c').lower()
    if version == 'v3':
        return UsmUserData(
            device.get('username', ''),
            authKey=device.get('auth_password_encrypted', ''),
            privKey=device.get('priv_password_encrypted', ''),
            authProtocol=usmHMACSHAAuthProtocol,
            privProtocol=usmAesCfb128Protocol,
        )
    # v1 uses mpModel=0, v2c uses mpModel=1
    mp_model = 0 if version == 'v1' else 1
    return CommunityData(
        device.get('community_string', 'public'),
        mpModel=mp_model,
    )


def poll_snmp_device(device: dict) -> list:
    """
    Poll a single SNMP device.

    Returns a list of metric dicts suitable for push_metrics_to_victoriametrics().
    """
    device_id   = device['id']
    hostname    = device['hostname']
    ip          = device['ip_address']
    port        = int(device.get('port', 161))
    device_type = device.get('device_type', 'generic_ups')

    oids   = get_oids_for_device(device_type)
    auth   = _build_auth(device)
    transport = UdpTransportTarget((ip, port), timeout=10, retries=1)
    metrics = []

    log.info("polling_snmp_device",
             device_id=device_id,
             hostname=hostname,
             ip=ip,
             port=port,
             device_type=device_type)

    for metric_name, oid in oids.items():
        try:
            error_indication, error_status, error_index, var_binds = next(
                getCmd(
                    SnmpEngine(),
                    auth,
                    transport,
                    ContextData(),
                    ObjectType(ObjectIdentity(oid)),
                )
            )

            if error_indication:
                log.warning("snmp_error_indication",
                            hostname=hostname,
                            oid=oid,
                            error=str(error_indication))
                continue

            if error_status:
                log.warning("snmp_error_status",
                            hostname=hostname,
                            oid=oid,
                            error=str(error_status))
                continue

            for var_bind in var_binds:
                try:
                    value = float(var_bind[1])
                except (TypeError, ValueError):
                    continue

                metrics.append({
                    'name': f'infrasense_snmp_ups_{metric_name}',
                    'value': value,
                    'labels': {
                        'device_id': device_id,
                        'hostname': hostname,
                        'device_type': device_type,
                    },
                })

        except Exception as e:
            log.warning("snmp_oid_failed",
                        hostname=hostname,
                        oid=oid,
                        error=str(e))

    return metrics
