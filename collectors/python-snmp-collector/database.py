#!/usr/bin/env python3
import psycopg2
import os
import structlog

log = structlog.get_logger()


def get_db_connection():
    return psycopg2.connect(
        host=os.getenv('DB_HOST', 'postgres'),
        port=int(os.getenv('DB_PORT', 5432)),
        dbname=os.getenv('DB_NAME', 'infrasense'),
        user=os.getenv('DB_USER', 'infrasense'),
        password=os.getenv('DB_PASSWORD'),
        connect_timeout=10
    )


def load_snmp_devices():
    """Load all SNMP-capable devices from database"""
    conn = get_db_connection()
    cur = conn.cursor()

    snmp_types = [
        'apc_ups', 'apc_pdu', 'eaton_ups',
        'cyberpower_ups', 'tripplite_ups',
        'cisco_switch', 'cisco_router',
        'juniper_switch', 'aruba_switch',
        'generic_ups', 'generic_snmp', 'snmp'
    ]

    placeholders = ','.join(['%s'] * len(snmp_types))

    cur.execute(f"""
        SELECT
            d.id::text,
            d.hostname,
            d.ip_address::text as ip_address,
            COALESCE(dc.community_string, 'public') as community_string,
            COALESCE(dc.snmp_version, 'v2c') as snmp_version,
            COALESCE(dc.port, 161) as port,
            COALESCE(dc.username, '') as username,
            COALESCE(dc.auth_password_encrypted, '') as auth_password_encrypted,
            COALESCE(dc.priv_password_encrypted, '') as priv_password_encrypted,
            COALESCE(dc.auth_protocol, 'SHA') as auth_protocol,
            COALESCE(dc.priv_protocol, 'AES') as priv_protocol,
            d.device_type,
            d.status
        FROM devices d
        LEFT JOIN device_credentials dc ON d.id = dc.device_id
        WHERE (
            d.protocol = 'snmp'
            OR d.device_type ILIKE '%%snmp%%'
            OR d.device_type IN ({placeholders})
        )
        AND d.status != 'deleted'
    """, snmp_types)

    columns = [
        'id', 'hostname', 'ip_address', 'community_string', 'snmp_version',
        'port', 'username', 'auth_password_encrypted', 'priv_password_encrypted',
        'auth_protocol', 'priv_protocol', 'device_type', 'status'
    ]
    devices = [dict(zip(columns, row)) for row in cur.fetchall()]

    cur.close()
    conn.close()

    log.info("loaded_snmp_devices", count=len(devices))
    return devices


def update_device_status(device_id, status, error=None):
    """Update device status after poll"""
    conn = get_db_connection()
    cur = conn.cursor()
    if error:
        cur.execute("""
            UPDATE devices SET status=%s, last_error=%s, updated_at=NOW()
            WHERE id=%s::uuid
        """, (status, error, device_id))
    else:
        cur.execute("""
            UPDATE devices SET status=%s, last_seen=NOW(), last_error=NULL, updated_at=NOW()
            WHERE id=%s::uuid
        """, (status, device_id))
    conn.commit()
    cur.close()
    conn.close()
