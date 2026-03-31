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


def load_ipmi_devices():
    """Load all IPMI devices from database"""
    conn = get_db_connection()
    cur = conn.cursor()

    ipmi_types = [
        'dell_drac5', 'dell_idrac6', 'dell_idrac7',
        'dell_idrac8_ipmi', 'hpe_ilo3_ipmi', 'hpe_ilo4_ipmi',
        'lenovo_imm', 'lenovo_xcc_ipmi',
        'supermicro_ipmi', 'supermicro_old',
        'cisco_cimc_ipmi', 'huawei_ibmc_ipmi',
        'fujitsu_irmc_ipmi', 'asus_asmb_ipmi',
        'gigabyte_bmc_ipmi', 'ericsson_bmc_ipmi',
        'ieit_bmc_ipmi', 'generic_ipmi', 'ipmi'
    ]

    placeholders = ','.join(['%s'] * len(ipmi_types))

    cur.execute(f"""
        SELECT
            d.id::text,
            d.hostname,
            COALESCE(d.bmc_ip_address::text, d.ip_address::text) as bmc_ip,
            COALESCE(dc.username, '') as username,
            COALESCE(dc.password_encrypted, '') as password_encrypted,
            COALESCE(dc.port, 623) as port,
            d.device_type,
            d.status
        FROM devices d
        LEFT JOIN device_credentials dc ON d.id = dc.device_id
        WHERE (
            d.protocol = 'ipmi'
            OR d.device_type ILIKE '%%ipmi%%'
            OR d.device_type IN ({placeholders})
        )
        AND d.status != 'deleted'
    """, ipmi_types)

    columns = ['id', 'hostname', 'bmc_ip', 'username', 'password_encrypted', 'port', 'device_type', 'status']
    devices = [dict(zip(columns, row)) for row in cur.fetchall()]

    cur.close()
    conn.close()

    log.info("loaded_ipmi_devices", count=len(devices))
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


def save_hardware_events(device_id, devices_hostname, events, source_protocol='ipmi'):
    """Save hardware events to hardware_events table (deduped by dedupe_key)"""
    if not events:
        return
    conn = get_db_connection()
    cur = conn.cursor()
    saved = 0
    for event in events:
        dedupe_key = f"{source_protocol}:{event.get('id', '')}:{event.get('message', '')[:64]}"
        try:
            cur.execute("""
                INSERT INTO hardware_events
                (id, device_id, occurred_at, observed_at, source_protocol,
                 component, event_type, severity, message, dedupe_key)
                VALUES (gen_random_uuid(), %s::uuid, %s, NOW(), %s,
                        %s, %s, %s, %s, %s)
                ON CONFLICT (device_id, dedupe_key) DO NOTHING
            """, (
                device_id,
                event.get('timestamp'),
                source_protocol,
                event.get('component', 'system'),
                event.get('type', 'system'),
                event.get('severity', 'info'),
                event.get('message', ''),
                dedupe_key,
            ))
            saved += cur.rowcount
        except Exception as e:
            log.warning("event_insert_failed", device_id=device_id, error=str(e))
            conn.rollback()
            continue
    conn.commit()
    cur.close()
    conn.close()
    if saved:
        log.info("hardware_events_saved",
                 device_id=device_id, hostname=devices_hostname, count=saved)
