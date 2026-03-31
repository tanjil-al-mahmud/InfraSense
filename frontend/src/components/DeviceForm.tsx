import React, { useState, useEffect } from 'react';
import { useForm, SubmitHandler, FieldValues } from 'react-hook-form';
import { useNavigate } from 'react-router-dom';
import { useCreateDevice, useUpdateDevice } from '../hooks/useDevices';
import { detectProtocol, saveDeviceCredentials } from '../services/deviceApi';
import { Device, CreateDeviceRequest, UpdateDeviceRequest, ProtocolDetectionResult } from '../types/device';

// ─── Device type definitions ───────────────────────────────────────────────

interface DeviceTypeOption {
  value: string;
  label: string;
  vendor: string;
  protocol: 'redfish' | 'ipmi' | 'snmp' | 'agent';
  defaultPort: number;
  defaultScheme?: 'https' | 'http' | '';
  description: string;
}

export const DEVICE_TYPE_OPTIONS: DeviceTypeOption[] = [
  // ─── Dell ───
  { value: 'dell_drac5',        label: 'Dell DRAC5',          vendor: 'Dell',      protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: '9th Gen (R610, R510) - IPMI only' },
  { value: 'dell_idrac6',       label: 'Dell iDRAC6',         vendor: 'Dell',      protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: '10th Gen (R710, R610) - IPMI only' },
  { value: 'dell_idrac7_ipmi',  label: 'Dell iDRAC7 (IPMI)',  vendor: 'Dell',      protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: '11th-12th Gen (R620, R720) - IPMI recommended' },
  { value: 'dell_idrac8_ipmi',  label: 'Dell iDRAC8 (IPMI)',  vendor: 'Dell',      protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: '13th Gen (R630, R730) - IPMI' },
  { value: 'dell_idrac8',       label: 'Dell iDRAC8 (Redfish)',vendor: 'Dell',     protocol: 'redfish', defaultPort: 443, defaultScheme: 'https', description: '13th Gen (R630, R730) - Redfish' },
  { value: 'dell_idrac9',       label: 'Dell iDRAC9 (Redfish)',vendor: 'Dell',     protocol: 'redfish', defaultPort: 443, defaultScheme: 'https', description: '14th-16th Gen (R740, R640, R750) - Redfish recommended' },
  { value: 'dell_idrac9_ipmi',  label: 'Dell iDRAC9 (IPMI)',  vendor: 'Dell',      protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: '14th-16th Gen - IPMI' },
  { value: 'dell_idrac10',      label: 'Dell iDRAC10 (Redfish)',vendor: 'Dell',    protocol: 'redfish', defaultPort: 443, defaultScheme: 'https', description: '17th Gen - Latest Redfish' },
  // ─── HPE ───
  { value: 'hpe_ilo3_ipmi',    label: 'HPE iLO3 (IPMI)',     vendor: 'HPE',       protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: 'G7 (DL380 G7, DL360 G7) - IPMI only' },
  { value: 'hpe_ilo4_ipmi',    label: 'HPE iLO4 (IPMI)',     vendor: 'HPE',       protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: 'Gen8/9 (DL380, DL360) - IPMI recommended' },
  { value: 'hpe_ilo4',         label: 'HPE iLO4 (Redfish)',  vendor: 'HPE',       protocol: 'redfish', defaultPort: 443, defaultScheme: 'https', description: 'Gen8/9 - Redfish limited' },
  { value: 'hpe_ilo5',         label: 'HPE iLO5 (Redfish)',  vendor: 'HPE',       protocol: 'redfish', defaultPort: 443, defaultScheme: 'https', description: 'Gen10 (DL380, DL360 Gen10) - Fully supported' },
  { value: 'hpe_ilo5_ipmi',    label: 'HPE iLO5 (IPMI)',     vendor: 'HPE',       protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: 'Gen10 - IPMI' },
  { value: 'hpe_ilo6',         label: 'HPE iLO6 (Redfish)',  vendor: 'HPE',       protocol: 'redfish', defaultPort: 443, defaultScheme: 'https', description: 'Gen11 (DL380, DL360 Gen11) - Redfish recommended' },
  // ─── Lenovo ───
  { value: 'lenovo_imm',       label: 'Lenovo IMM (IPMI)',     vendor: 'Lenovo',   protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: 'Old ThinkServer - IPMI only' },
  { value: 'lenovo_xcc_redfish',label: 'Lenovo XCC (Redfish)',   vendor: 'Lenovo',  protocol: 'redfish', defaultPort: 443, defaultScheme: 'https', description: 'ThinkSystem (SR650, SR630) - Redfish' },
  { value: 'lenovo_xcc_ipmi',  label: 'Lenovo XCC (IPMI)',     vendor: 'Lenovo',   protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: 'ThinkSystem - IPMI' },
  // ─── Supermicro ───
  { value: 'supermicro_old',   label: 'Supermicro BMC (IPMI old)', vendor: 'Supermicro', protocol: 'ipmi', defaultPort: 623, defaultScheme: '', description: 'Older X9/X10 boards - IPMI only' },
  { value: 'supermicro_ipmi',  label: 'Supermicro X11/X12 (IPMI)', vendor: 'Supermicro', protocol: 'ipmi', defaultPort: 623, defaultScheme: '', description: 'X11, X12 platform - IPMI' },
  { value: 'supermicro_redfish',label: 'Supermicro X11/X12 (Redfish)', vendor: 'Supermicro', protocol: 'redfish', defaultPort: 443, defaultScheme: 'https', description: 'X11, X12 platform - Redfish' },
  // ─── Cisco ───
  { value: 'cisco_cimc_ipmi',  label: 'Cisco CIMC (IPMI)',    vendor: 'Cisco',     protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: 'UCS C-Series (C220, C240) - IPMI' },
  { value: 'cisco_cimc_redfish',label: 'Cisco CIMC (Redfish)', vendor: 'Cisco',    protocol: 'redfish', defaultPort: 443, defaultScheme: 'https', description: 'UCS C-Series - Redfish partial' },
  { value: 'cisco_ucs',        label: 'Cisco UCS Manager',    vendor: 'Cisco',     protocol: 'redfish', defaultPort: 443, defaultScheme: 'https', description: 'UCS B-Series, X-Series - Redfish full' },
  // ─── Huawei ───
  { value: 'huawei_ibmc_redfish',label: 'Huawei iBMC (Redfish)', vendor: 'Huawei', protocol: 'redfish', defaultPort: 443, defaultScheme: 'https', description: 'FusionServer (2288H, 2488) - Redfish' },
  { value: 'huawei_ibmc_ipmi', label: 'Huawei iBMC (IPMI)',   vendor: 'Huawei',    protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: 'FusionServer - IPMI' },
  // ─── Fujitsu ───
  { value: 'fujitsu_irmc_redfish',label: 'Fujitsu iRMC (Redfish)', vendor: 'Fujitsu', protocol: 'redfish', defaultPort: 443, defaultScheme: 'https', description: 'PRIMERGY (RX2540, RX2530) - Redfish' },
  { value: 'fujitsu_irmc_ipmi',label: 'Fujitsu iRMC (IPMI)',  vendor: 'Fujitsu',   protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: 'PRIMERGY - IPMI' },
  // ─── ASUS ───
  { value: 'asus_asmb_ipmi',   label: 'ASUS ASMB (IPMI)',     vendor: 'ASUS',      protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: 'RS series servers - IPMI only' },
  // ─── Gigabyte ───
  { value: 'gigabyte_bmc_redfish',label: 'Gigabyte BMC (Redfish)', vendor: 'Gigabyte', protocol: 'redfish', defaultPort: 443, defaultScheme: 'https', description: 'R series servers - Redfish' },
  { value: 'gigabyte_bmc_ipmi',label: 'Gigabyte BMC (IPMI)',  vendor: 'Gigabyte',  protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: 'R series servers - IPMI' },
  // ─── Ericsson ───
  { value: 'ericsson_cru_redfish',label: 'Ericsson CRU (Redfish)', vendor: 'Ericsson', protocol: 'redfish', defaultPort: 443, defaultScheme: 'https', description: 'CRU series - Redfish' },
  { value: 'ericsson_bmc_ipmi',label: 'Ericsson BMC (IPMI)',  vendor: 'Ericsson',  protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: 'CRU series - IPMI only' },
  // ─── IEIT ───
  { value: 'ieit_bmc_ipmi',    label: 'IEIT BMC (IPMI)',      vendor: 'IEIT',      protocol: 'ipmi',    defaultPort: 623, defaultScheme: '', description: 'IEIT servers - IPMI only' },
  // ─── Generic ───
  { value: 'generic_ipmi',     label: 'Generic IPMI / Legacy BMC', vendor: 'Generic', protocol: 'ipmi', defaultPort: 623, defaultScheme: '', description: 'Any IPMI-compatible BMC' },
  // ─── UPS ───
  { value: 'apc_ups_snmp',     label: 'APC UPS (SNMP)',       vendor: 'APC',       protocol: 'snmp',    defaultPort: 161, defaultScheme: '', description: 'SNMP v2c/v3' },
  { value: 'eaton_ups_snmp',   label: 'Eaton UPS (SNMP)',     vendor: 'Eaton',     protocol: 'snmp',    defaultPort: 161, defaultScheme: '', description: 'SNMP v2c/v3' },
  { value: 'generic_ups_snmp', label: 'Generic UPS (SNMP)',   vendor: 'Generic',   protocol: 'snmp',    defaultPort: 161, defaultScheme: '', description: 'SNMP v2c/v3' },
  // ─── OS Agents ───
  { value: 'linux_agent',      label: 'Linux (node_exporter)', vendor: 'Linux',    protocol: 'agent',   defaultPort: 9100, defaultScheme: 'http', description: 'Linux server with node_exporter' },
  { value: 'windows_agent',    label: 'Windows (win_exporter)',vendor: 'Windows',  protocol: 'agent',   defaultPort: 9182, defaultScheme: 'http', description: 'Windows server with windows_exporter' },
];

const VENDORS = [...new Set(DEVICE_TYPE_OPTIONS.map((d) => d.vendor))];

const IP_PATTERN = /^((\d{1,3}\.){3}\d{1,3}|([\da-fA-F]{0,4}:){2,7}[\da-fA-F]{0,4}|::[\da-fA-F]{0,4})$/;

interface FormValues extends FieldValues {
  hostname: string;
  ip_address: string;
  bmc_ip_address?: string;
  device_type: string;
  protocol?: string;
  port?: string;
  scheme?: string;
  location?: string;
  tags?: string;
  description?: string;
  // Credentials
  bmc_username?: string;
  bmc_password?: string;
  skip_tls_verify?: boolean;
  // SNMP
  snmp_version?: string;
  snmp_community?: string;
}

export interface DeviceFormProps {
  device?: Device;
  onSuccess?: (device: Device) => void;
  onCancel?: () => void;
}

const DeviceForm: React.FC<DeviceFormProps> = ({ device, onSuccess, onCancel }) => {
  const navigate = useNavigate();
  const isEditMode = !!device;

  const createDevice = useCreateDevice();
  const updateDevice = useUpdateDevice();

  const [detecting, setDetecting] = useState(false);
  const [detectionResult, setDetectionResult] = useState<ProtocolDetectionResult | null>(null);
  const [detectionError, setDetectionError] = useState<string | null>(null);
  const [showBmcPassword, setShowBmcPassword] = useState(false);

  // Derive current device type info
  const getDeviceTypeInfo = (typeValue: string) =>
    DEVICE_TYPE_OPTIONS.find((d) => d.value === typeValue);

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isSubmitting },
    setError,
  } = useForm<FormValues>({
    defaultValues: {
      hostname: device?.hostname ?? '',
      ip_address: device?.ip_address ?? '',
      bmc_ip_address: device?.bmc_ip_address ?? '',
      device_type: device?.device_type ?? 'dell_idrac9',
      protocol: (device as any)?.protocol ?? '',
      port: String((device as any)?.port ?? ''),
      scheme: (device as any)?.scheme ?? 'https',
      location: device?.location ?? '',
      tags: device?.tags?.join(', ') ?? '',
      description: (device as any)?.description ?? '',
      bmc_username: '',
      bmc_password: '',
      skip_tls_verify: true,
      snmp_version: 'v2c',
      snmp_community: 'public',
    },
  });

  const selectedDeviceType = watch('device_type');
  const bmcIP = watch('bmc_ip_address');
  const ipAddress = watch('ip_address');
  const snmpVersion = watch('snmp_version');

  const typeInfo = getDeviceTypeInfo(selectedDeviceType);
  const protocol = typeInfo?.protocol ?? 'redfish';

  // Auto-fill port/scheme when device type changes
  useEffect(() => {
    if (typeInfo && !isEditMode) {
      setValue('port', String(typeInfo.defaultPort));
      setValue('scheme', typeInfo.defaultScheme ?? 'https');
    }
  }, [selectedDeviceType, typeInfo, isEditMode, setValue]);

  const handleAutoDetect = async () => {
    const target = bmcIP || ipAddress;
    if (!target || !IP_PATTERN.test(target)) {
      setDetectionError('Enter a valid IP address or BMC IP first');
      return;
    }
    setDetecting(true);
    setDetectionResult(null);
    setDetectionError(null);
    try {
      const result = await detectProtocol(target);
      setDetectionResult(result);
      const recommended = result.recommended_protocol;
      if (recommended) {
        const match = DEVICE_TYPE_OPTIONS.find((d) => d.protocol === recommended);
        if (match) setValue('device_type', match.value);
      }
    } catch (err) {
      setDetectionError(err instanceof Error ? err.message : 'Protocol detection failed');
    } finally {
      setDetecting(false);
    }
  };

  const parseTags = (raw: string): string[] =>
    raw.split(',').map((t) => t.trim()).filter(Boolean);

  const onSubmit: SubmitHandler<FormValues> = async (data) => {
    const tags = parseTags(data.tags ?? '');
    const port = data.port ? parseInt(data.port, 10) : typeInfo?.defaultPort;
    const proto = typeInfo?.protocol ?? data.protocol;

    try {
      let savedDevice: Device;

      if (isEditMode && device) {
        const payload: UpdateDeviceRequest = {
          hostname: data.hostname,
          ip_address: data.ip_address,
          bmc_ip_address: data.bmc_ip_address || undefined,
          device_type: data.device_type,
          location: data.location || undefined,
          tags: tags.length > 0 ? tags : undefined,
          protocol: proto as any,
        };
        savedDevice = await updateDevice.mutateAsync({ id: device.id, data: payload });
      } else {
        const payload: CreateDeviceRequest = {
          hostname: data.hostname,
          ip_address: data.ip_address,
          bmc_ip_address: data.bmc_ip_address || undefined,
          device_type: data.device_type,
          location: data.location || undefined,
          tags: tags.length > 0 ? tags : undefined,
          protocol: proto as any,
          polling_interval: 60,
          ssl_verify: proto === 'redfish' ? !data.skip_tls_verify : undefined,
        };
        savedDevice = await createDevice.mutateAsync(payload);
      }

      // Save credentials when relevant
      if (proto === 'redfish' && (data.bmc_username || (!isEditMode && data.bmc_password))) {
        await saveDeviceCredentials(savedDevice.id, {
          protocol: 'redfish',
          username: data.bmc_username || undefined,
          password: data.bmc_password || undefined,
          port: port ?? 443,
          http_scheme: (data.scheme as 'https' | 'http') || 'https',
          ssl_verify: !data.skip_tls_verify,
          polling_interval: 60,
          timeout_seconds: 30,
          retry_attempts: 3,
        });
      } else if (proto === 'ipmi' && (data.bmc_username || (!isEditMode && data.bmc_password))) {
        await saveDeviceCredentials(savedDevice.id, {
          protocol: 'ipmi',
          username: data.bmc_username || undefined,
          password: data.bmc_password || undefined,
          port: port ?? 623,
          polling_interval: 60,
          timeout_seconds: 30,
          retry_attempts: 3,
        });
      } else if (proto === 'snmp') {
        if (data.snmp_version === 'v2c') {
          await saveDeviceCredentials(savedDevice.id, {
            protocol: 'snmp_v2c',
            community_string: data.snmp_community || 'public',
          });
        } else {
          await saveDeviceCredentials(savedDevice.id, { protocol: 'snmp_v3' });
        }
      }

      onSuccess?.(savedDevice);
      if (!onSuccess) navigate(`/devices/${savedDevice.id}`);
    } catch (err) {
      setError('root', { message: err instanceof Error ? err.message : 'An unexpected error occurred' });
    }
  };

  const handleCancel = () => {
    if (onCancel) onCancel();
    else navigate(-1);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate style={styles.form}>
      {errors.root && (
        <div style={styles.errorBanner} role="alert">{errors.root.message}</div>
      )}

      {/* Hostname */}
      <div style={styles.fieldGroup}>
        <label htmlFor="hostname" style={styles.label}>Hostname <span style={styles.required}>*</span></label>
        <input id="hostname" type="text" placeholder="e.g. server-01.example.com" disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.hostname ? styles.inputError : {}) }}
          {...register('hostname', { required: 'Hostname is required' })} />
        {errors.hostname && <p style={styles.fieldError} role="alert">{errors.hostname.message}</p>}
      </div>

      {/* IP Address */}
      <div style={styles.fieldGroup}>
        <label htmlFor="ip_address" style={styles.label}>IP Address <span style={styles.required}>*</span></label>
        <input id="ip_address" type="text" placeholder="e.g. 192.168.1.10" disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.ip_address ? styles.inputError : {}) }}
          {...register('ip_address', { required: 'IP address is required', pattern: { value: IP_PATTERN, message: 'Enter a valid IPv4 or IPv6 address' } })} />
        {errors.ip_address && <p style={styles.fieldError} role="alert">{errors.ip_address.message}</p>}
      </div>

      {/* BMC IP Address */}
      <div style={styles.fieldGroup}>
        <label htmlFor="bmc_ip_address" style={styles.label}>BMC IP Address</label>
        <input id="bmc_ip_address" type="text" placeholder="e.g. 192.168.1.11 (optional)" disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.bmc_ip_address ? styles.inputError : {}) }}
          {...register('bmc_ip_address', { validate: val => !val || IP_PATTERN.test(val) || 'Enter a valid IPv4 or IPv6 address' })} />
        {errors.bmc_ip_address && <p style={styles.fieldError} role="alert">{errors.bmc_ip_address.message}</p>}
      </div>

      {/* Device Type */}
      <div style={styles.fieldGroup}>
        <label htmlFor="device_type" style={styles.label}>Device Type <span style={styles.required}>*</span></label>
        <select id="device_type" disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.device_type ? styles.inputError : {}) }}
          {...register('device_type', { required: 'Device type is required' })}>
          <option value="">— Select device type —</option>
          {VENDORS.map((vendor) => (
            <optgroup key={vendor} label={`── ${vendor} ──`}>
              {DEVICE_TYPE_OPTIONS.filter((d) => d.vendor === vendor).map((d) => (
                <option key={d.value} value={d.value}>{d.label}</option>
              ))}
            </optgroup>
          ))}
        </select>
        {errors.device_type && <p style={styles.fieldError} role="alert">{errors.device_type.message}</p>}
        {typeInfo && (
          <p style={styles.hint}>
            <strong>{typeInfo.protocol.toUpperCase()}</strong> · Port: {typeInfo.defaultPort} · {typeInfo.description}
          </p>
        )}
      </div>

      {/* Port + Scheme row */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.75rem' }}>
        <div style={styles.fieldGroup}>
          <label htmlFor="port" style={styles.label}>Port</label>
          <input id="port" type="number" disabled={isSubmitting}
            style={styles.input} placeholder={String(typeInfo?.defaultPort ?? 443)}
            {...register('port')} />
        </div>
        {(protocol === 'redfish' || protocol === 'agent') && (
          <div style={styles.fieldGroup}>
            <label htmlFor="scheme" style={styles.label}>Scheme</label>
            <select id="scheme" disabled={isSubmitting} style={styles.input} {...register('scheme')}>
              <option value="https">HTTPS</option>
              <option value="http">HTTP</option>
            </select>
          </div>
        )}
      </div>

      {/* BMC / Redfish Credentials */}
      {(protocol === 'redfish' || protocol === 'ipmi') && (
        <fieldset style={styles.fieldset}>
          <legend style={styles.legend}>{protocol === 'redfish' ? 'Redfish / BMC' : 'IPMI'} Credentials</legend>

          {/* Protocol Auto-Detection */}
          <div style={{ marginBottom: '0.75rem' }}>
            <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
              <button type="button" onClick={handleAutoDetect} disabled={detecting || isSubmitting}
                style={{ ...styles.detectBtn, ...(detecting ? styles.btnDisabled : {}) }}>
                {detecting ? '🔍 Detecting…' : '🔍 Auto-Detect Protocol'}
              </button>
            </div>
            {detectionError && <div style={{ ...styles.errorBanner, marginTop: '0.5rem', fontSize: '0.8rem' }}>{detectionError}</div>}
            {detectionResult && (
              <div style={styles.detectionResult}>
                <div style={{ fontSize: '0.8rem', fontWeight: 700, color: '#4ade80', marginBottom: '0.25rem' }}>
                  ✅ Recommended: {detectionResult.recommended_protocol?.toUpperCase()}
                </div>
              </div>
            )}
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.75rem' }}>
            <div style={styles.fieldGroup}>
              <label style={styles.label}>BMC Username</label>
              <input type="text" autoComplete="off" disabled={isSubmitting}
                style={styles.input} placeholder="e.g. admin"
                {...register('bmc_username')} />
            </div>
            <div style={styles.fieldGroup}>
              <label style={styles.label}>BMC Password</label>
              <div style={{ display: 'flex', gap: '0.25rem' }}>
                <input type={showBmcPassword ? 'text' : 'password'} autoComplete="new-password" disabled={isSubmitting}
                  style={{ ...styles.input, flex: 1 }}
                  placeholder={isEditMode ? 'Leave blank to keep current password' : ''}
                  {...register('bmc_password')} />
                <button type="button" style={styles.eyeBtn} onClick={() => setShowBmcPassword(v => !v)}>
                  {showBmcPassword ? '🙈' : '👁'}
                </button>
              </div>
            </div>
          </div>

          {protocol === 'redfish' && (
            <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginTop: '0.5rem', fontSize: '0.875rem', color: '#374151', cursor: 'pointer' }}>
              <input type="checkbox" {...register('skip_tls_verify')} disabled={isSubmitting} />
              Skip TLS Verify (self-signed certificates)
            </label>
          )}
        </fieldset>
      )}

      {/* SNMP Credentials */}
      {protocol === 'snmp' && (
        <fieldset style={styles.fieldset}>
          <legend style={styles.legend}>SNMP Configuration</legend>
          <div style={styles.fieldGroup}>
            <label style={styles.label}>SNMP Version</label>
            <select style={styles.input} disabled={isSubmitting} {...register('snmp_version')}>
              <option value="v2c">v2c</option>
              <option value="v3">v3</option>
            </select>
          </div>
          {snmpVersion === 'v2c' && (
            <div style={{ ...styles.fieldGroup, marginTop: '0.5rem' }}>
              <label style={styles.label}>Community String</label>
              <input type="text" style={styles.input} disabled={isSubmitting}
                placeholder="public" {...register('snmp_community')} />
            </div>
          )}
        </fieldset>
      )}

      {/* Location */}
      <div style={styles.fieldGroup}>
        <label htmlFor="location" style={styles.label}>Location</label>
        <input id="location" type="text" placeholder="e.g. Rack A, Row 3 (optional)" disabled={isSubmitting}
          style={styles.input} {...register('location')} />
      </div>

      {/* Tags */}
      <div style={styles.fieldGroup}>
        <label htmlFor="tags" style={styles.label}>Tags</label>
        <input id="tags" type="text" placeholder="e.g. production, web, critical (comma-separated)" disabled={isSubmitting}
          style={styles.input} {...register('tags')} />
        <p style={styles.hint}>Separate multiple tags with commas.</p>
      </div>

      {/* Actions */}
      <div style={styles.actions}>
        <button type="button" onClick={handleCancel} disabled={isSubmitting} style={styles.cancelBtn}>Cancel</button>
        <button type="submit" disabled={isSubmitting}
          style={{ ...styles.submitBtn, ...(isSubmitting ? styles.btnDisabled : {}) }}>
          {isSubmitting ? (isEditMode ? 'Saving...' : 'Registering...') : (isEditMode ? 'Save Changes' : 'Register Device')}
        </button>
      </div>
    </form>
  );
};

const styles: Record<string, React.CSSProperties> = {
  form: { display: 'flex', flexDirection: 'column', gap: '1rem' },
  errorBanner: { backgroundColor: '#fee2e2', border: '1px solid #fca5a5', borderRadius: '4px', color: '#b91c1c', fontSize: '0.875rem', padding: '0.75rem 1rem' },
  fieldGroup: { display: 'flex', flexDirection: 'column', gap: '0.25rem' },
  label: { fontSize: '0.875rem', fontWeight: 500, color: '#374151' },
  required: { color: '#ef4444' },
  input: { border: '1px solid #d1d5db', borderRadius: '4px', fontSize: '0.875rem', padding: '0.5rem 0.75rem', width: '100%', boxSizing: 'border-box' as const, backgroundColor: '#fff', outline: 'none' },
  inputError: { borderColor: '#ef4444' },
  fieldError: { color: '#ef4444', fontSize: '0.75rem', margin: 0 },
  hint: { color: '#9ca3af', fontSize: '0.75rem', margin: 0 },
  actions: { display: 'flex', gap: '0.75rem', justifyContent: 'flex-end', marginTop: '0.5rem' },
  cancelBtn: { backgroundColor: '#f3f4f6', border: '1px solid #d1d5db', borderRadius: '4px', color: '#374151', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 500, padding: '0.5rem 1rem' },
  submitBtn: { backgroundColor: '#2563eb', border: 'none', borderRadius: '4px', color: '#fff', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 600, padding: '0.5rem 1.25rem' },
  btnDisabled: { opacity: 0.6, cursor: 'not-allowed' },
  detectBtn: { backgroundColor: '#1e293b', border: '1px solid #3b82f6', borderRadius: '4px', color: '#60a5fa', cursor: 'pointer', fontSize: '0.8rem', fontWeight: 600, padding: '0.45rem 0.9rem', whiteSpace: 'nowrap' as const },
  detectionResult: { marginTop: '0.5rem', background: '#0f172a', border: '1px solid #16a34a', borderRadius: '6px', padding: '0.5rem 0.75rem' },
  fieldset: { border: '1px solid #e5e7eb', borderRadius: '6px', padding: '0.75rem 1rem', margin: 0 },
  legend: { fontSize: '0.8rem', fontWeight: 600, color: '#374151', padding: '0 0.25rem' },
  eyeBtn: { background: '#f3f4f6', border: '1px solid #d1d5db', borderRadius: '4px', cursor: 'pointer', padding: '0 0.5rem', fontSize: '1rem', flexShrink: 0 },
};

export default DeviceForm;
