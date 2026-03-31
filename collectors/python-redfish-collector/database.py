import psycopg2
import os
import base64
import structlog
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

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


def decrypt_password(encrypted_password):
    """Decrypt password using AES-256-GCM - must match Go encryption"""
    encryption_key = os.getenv('ENCRYPTION_KEY', '').encode()
    if len(encryption_key) != 32:
        return encrypted_password
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


def load_redfish_devices():
    """Load all Redfish-capable devices from database"""
    conn = get_db_connection()
    cur = conn.cursor()

    redfish_types = [
        'dell_idrac8', 'dell_idrac9', 'dell_idrac10',
        'hpe_ilo4', 'hpe_ilo5', 'hpe_ilo6',
        'lenovo_xcc', 'supermicro_redfish',
        'huawei_ibmc', 'cisco_cimc', 'fujitsu_irmc',
        'generic_redfish', 'redfish'
    ]

    placeholders = ','.join(['%s'] * len(redfish_types))

    cur.execute(f"""
        SELECT
            d.id::text,
            d.hostname,
            COALESCE(d.bmc_ip_address::text, d.ip_address::text) as bmc_ip,
            COALESCE(dc.username, '') as username,
            COALESCE(dc.password_encrypted, '') as password_encrypted,
            COALESCE(dc.port, 443) as port,
            d.device_type,
            d.status
        FROM devices d
        LEFT JOIN device_credentials dc ON d.id = dc.device_id
        WHERE (
            d.protocol = 'redfish'
            OR d.device_type ILIKE '%%redfish%%'
            OR d.device_type IN ({placeholders})
        )
        AND d.status != 'deleted'
    """, redfish_types)

    columns = ['id', 'hostname', 'bmc_ip', 'username', 'password_encrypted', 'port', 'device_type', 'status']
    devices = []
    for row in cur.fetchall():
        device = dict(zip(columns, row))
        device['password'] = decrypt_password(device['password_encrypted'])
        devices.append(device)

    cur.close()
    conn.close()

    log.info("loaded_redfish_devices", count=len(devices))
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


def save_hardware_events(device_id, hostname, events, source_protocol='redfish'):
    """Save hardware events to hardware_events table"""
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
                 device_id=device_id, hostname=hostname, count=saved)
