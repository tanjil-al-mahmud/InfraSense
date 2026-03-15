import React, { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { deviceKeys, useDevice, useUpdateDevice, useDeleteDevice, useSyncDevice, usePowerControl, useBootControl } from '../hooks/useDevices';
import { fetchDeviceInventory, fetchDeviceMetrics, streamDeviceTelemetry } from '../services/deviceApi';
import {
  Device, UpdateDeviceRequest, DeviceStatus, DeviceMetricPoint,
  SELEntry, LifecycleLogEntry, FullNICInfo,
  DriveInfo, VirtualDiskInfo, TemperatureReading, FanReading,
  PowerSupplyInfo, VoltageReading, ProcessorInfo, MemoryInfo,
  StorageControllerInfo, AcceleratorInfo, PCIeSlotInfo,
  PowerControlRequest, BootControlRequest, TelemetryEvent, StorageEnclosureInfo,
} from '../types/device';
import { useForm } from 'react-hook-form';

// ── colour tokens ─────────────────────────────────────────────────────────────
const STATUS_CFG: Record<DeviceStatus, { bg: string; border: string; text: string; dot: string; label: string }> = {
  healthy:     { bg: '#052e16', border: '#16a34a', text: '#4ade80', dot: '#22c55e', label: 'Healthy' },
  warning:     { bg: '#422006', border: '#d97706', text: '#fbbf24', dot: '#f59e0b', label: 'Warning' },
  critical:    { bg: '#450a0a', border: '#dc2626', text: '#f87171', dot: '#ef4444', label: 'Critical' },
  unavailable: { bg: '#111827', border: '#4b5563', text: '#9ca3af', dot: '#6b7280', label: 'Offline' },
  unknown:     { bg: '#111827', border: '#4b5563', text: '#9ca3af', dot: '#6b7280', label: 'Unknown' },
};

function healthColor(h?: string) {
  if (!h) return '#64748b';
  const l = h.toLowerCase();
  if (l === 'ok' || l === 'good') return '#4ade80';
  if (l === 'warning') return '#fbbf24';
  if (l === 'critical' || l === 'error') return '#f87171';
  return '#64748b';
}

// ── shared styles ─────────────────────────────────────────────────────────────
const p: Record<string, React.CSSProperties> = {
  card: { background: '#0f172a', border: '1px solid #1e293b', borderRadius: 10, padding: '1.25rem', marginBottom: '1rem' },
  cardHeader: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1rem' },
  cardTitle: { fontSize: '0.8rem', fontWeight: 700, color: '#475569', textTransform: 'uppercase', letterSpacing: '0.08em' },
  grid2: { display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.75rem' },
  grid3: { display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '0.75rem' },
  grid4: { display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '0.75rem' },
  tableWrap: { overflowX: 'auto' as const },
  table: { width: '100%', borderCollapse: 'collapse' as const, fontSize: '0.8rem' },
  th: { padding: '0.5rem 0.75rem', textAlign: 'left' as const, color: '#475569', fontWeight: 600, borderBottom: '1px solid #1e293b', whiteSpace: 'nowrap' as const },
  td: { padding: '0.5rem 0.75rem', color: '#cbd5e1', borderBottom: '1px solid #0f172a', verticalAlign: 'top' as const },
};

// ── sub-components ────────────────────────────────────────────────────────────
const InfoField: React.FC<{ label: string; value: React.ReactNode }> = ({ label, value }) => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
    <span style={{ fontSize: '0.65rem', fontWeight: 700, color: '#475569', textTransform: 'uppercase', letterSpacing: '0.1em' }}>{label}</span>
    <span style={{ fontSize: '0.875rem', color: '#e2e8f0', fontWeight: 500 }}>{value || '—'}</span>
  </div>
);

const Card: React.FC<{ title: string; badge?: string; children: React.ReactNode; style?: React.CSSProperties }> = ({ title, badge, children, style }) => (
  <div style={{ ...p.card, ...style }}>
    <div style={p.cardHeader}>
      <span style={p.cardTitle}>{title}</span>
      {badge && <span style={{ fontSize: '0.7rem', color: '#475569' }}>{badge}</span>}
    </div>
    {children}
  </div>
);

const SevBadge: React.FC<{ sev: string }> = ({ sev }) => {
  const l = sev.toLowerCase();
  const color = l === 'critical' ? '#f87171' : l === 'warning' ? '#fbbf24' : '#94a3b8';
  return <span style={{ color, fontWeight: 700, fontSize: '0.75rem' }}>{sev}</span>;
};

const HealthDot: React.FC<{ health?: string }> = ({ health }) => (
  <span style={{ display: 'inline-block', width: 8, height: 8, borderRadius: '50%', background: healthColor(health), marginRight: 6 }} />
);

interface MetricChartProps { title: string; data: DeviceMetricPoint[]; unit: string; color: string; gradId: string }
const MetricChart: React.FC<MetricChartProps> = ({ title, data, unit, color, gradId }) => {
  const chartData = data.map(pt => ({
    time: new Date(pt.timestamp * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    value: pt.value,
  }));
  return (
    <div style={{ background: '#1e293b', border: '1px solid #334155', borderRadius: 10, padding: '1rem' }}>
      <div style={{ fontSize: '0.75rem', fontWeight: 700, color: '#94a3b8', marginBottom: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.06em' }}>{title}</div>
      <ResponsiveContainer width="100%" height={120}>
        <AreaChart data={chartData} margin={{ top: 4, right: 4, left: -20, bottom: 0 }}>
          <defs>
            <linearGradient id={gradId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={color} stopOpacity={0.3} />
              <stop offset="95%" stopColor={color} stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" />
          <XAxis dataKey="time" tick={{ fontSize: 9, fill: '#64748b' }} axisLine={false} tickLine={false} />
          <YAxis tick={{ fontSize: 9, fill: '#64748b' }} axisLine={false} tickLine={false} unit={unit} />
          <Tooltip contentStyle={{ background: '#0f172a', border: '1px solid #334155', borderRadius: 6, color: '#e2e8f0', fontSize: 11 }} formatter={(v: number) => [`${v}${unit}`, title]} />
          <Area type="monotone" dataKey="value" stroke={color} strokeWidth={2} fill={`url(#${gradId})`} dot={false} />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
};

// ── Edit Modal ────────────────────────────────────────────────────────────────
const ms: Record<string, React.CSSProperties> = {
  overlay: { position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000, padding: '1rem' },
  modal: { background: '#0f172a', border: '1px solid #1e293b', borderRadius: 12, boxShadow: '0 25px 60px rgba(0,0,0,0.5)', maxWidth: 520, width: '100%', padding: '1.5rem', maxHeight: '90vh', overflowY: 'auto' },
  input: { background: '#1e293b', border: '1px solid #334155', borderRadius: 6, color: '#e2e8f0', fontSize: '0.875rem', padding: '0.5rem 0.75rem', width: '100%', boxSizing: 'border-box' },
  errBanner: { background: '#450a0a', border: '1px solid #dc2626', borderRadius: 6, color: '#f87171', fontSize: '0.85rem', padding: '0.65rem 1rem' },
  cancelBtn: { background: '#1e293b', border: '1px solid #334155', borderRadius: 6, color: '#94a3b8', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 500, padding: '0.5rem 1rem' },
  saveBtn: { background: '#2563eb', border: 'none', borderRadius: 6, color: '#fff', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 600, padding: '0.5rem 1rem' },
};

const EditDeviceModal: React.FC<{ device: Device; onClose: () => void; onSaved: () => void }> = ({ device, onClose, onSaved }) => {
  const updateDevice = useUpdateDevice();
  const { register, handleSubmit, formState: { errors, isSubmitting }, setError } = useForm<UpdateDeviceRequest>({
    defaultValues: { hostname: device.hostname, ip_address: device.ip_address, bmc_ip_address: device.bmc_ip_address ?? '', location: device.location ?? '' },
  });
  const onSubmit = async (data: UpdateDeviceRequest) => {
    try {
      await updateDevice.mutateAsync({ id: device.id, data: { ...data, bmc_ip_address: data.bmc_ip_address || undefined, location: data.location || undefined } });
      onSaved(); onClose();
    } catch (err) {
      setError('root', { message: err instanceof Error ? err.message : 'Failed to update device' });
    }
  };
  return (
    <div style={ms.overlay} role="dialog" aria-modal="true">
      <div style={ms.modal}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1.25rem' }}>
          <h2 style={{ fontSize: '1.1rem', fontWeight: 700, color: '#f1f5f9', margin: 0 }}>Edit Device</h2>
          <button onClick={onClose} style={{ background: 'none', border: 'none', color: '#64748b', cursor: 'pointer', fontSize: '1.1rem' }}>✕</button>
        </div>
        <form onSubmit={handleSubmit(onSubmit)} style={{ display: 'flex', flexDirection: 'column', gap: '0.9rem' }}>
          {errors.root && <div style={ms.errBanner}>{errors.root.message}</div>}
          {([
            { label: 'Hostname *', key: 'hostname' as const, required: 'Required' },
            { label: 'IP Address *', key: 'ip_address' as const, required: 'Required' },
            { label: 'BMC IP Address', key: 'bmc_ip_address' as const },
            { label: 'Location', key: 'location' as const },
          ] as const).map(f => (
            <div key={f.key} style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
              <label style={{ fontSize: '0.8rem', fontWeight: 600, color: '#94a3b8' }}>{f.label}</label>
              <input style={{ ...ms.input, ...(errors[f.key] ? { borderColor: '#ef4444' } : {}) }}
                {...register(f.key, 'required' in f && f.required ? { required: f.required } : {})} />
              {errors[f.key] && <span style={{ color: '#f87171', fontSize: '0.75rem' }}>{errors[f.key]?.message}</span>}
            </div>
          ))}
          <div style={{ display: 'flex', gap: '0.75rem', justifyContent: 'flex-end', marginTop: 4 }}>
            <button type="button" onClick={onClose} style={ms.cancelBtn} disabled={isSubmitting}>Cancel</button>
            <button type="submit" style={{ ...ms.saveBtn, ...(isSubmitting ? { opacity: 0.6 } : {}) }} disabled={isSubmitting}>
              {isSubmitting ? 'Saving…' : 'Save Changes'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

// ── Power Control Modal ───────────────────────────────────────────────────────
const POWER_ACTIONS: { label: string; value: PowerControlRequest['reset_type']; color: string; confirm?: string }[] = [
  { label: 'Power On',          value: 'On',               color: '#22c55e' },
  { label: 'Graceful Shutdown', value: 'GracefulShutdown',  color: '#f59e0b', confirm: 'Gracefully shut down this server?' },
  { label: 'Force Off',         value: 'ForceOff',          color: '#ef4444', confirm: 'Force power off? This may cause data loss.' },
  { label: 'Graceful Restart',  value: 'GracefulRestart',   color: '#3b82f6', confirm: 'Gracefully restart this server?' },
  { label: 'Force Restart',     value: 'ForceRestart',      color: '#f97316', confirm: 'Force restart? This may cause data loss.' },
  { label: 'Power Cycle',       value: 'PowerCycle',        color: '#a855f7', confirm: 'Power cycle this server?' },
];

const PowerModal: React.FC<{ deviceId: string; onClose: () => void }> = ({ deviceId, onClose }) => {
  const powerControl = usePowerControl();
  const [msg, setMsg] = useState<{ text: string; ok: boolean } | null>(null);
  const [pending, setPending] = useState<string | null>(null);

  const execute = async (action: typeof POWER_ACTIONS[0]) => {
    if (action.confirm && !window.confirm(action.confirm)) return;
    setPending(action.value);
    setMsg(null);
    try {
      const res = await powerControl.mutateAsync({ id: deviceId, req: { reset_type: action.value } });
      setMsg({ text: res.message || 'Command sent', ok: res.success });
    } catch (e) {
      setMsg({ text: e instanceof Error ? e.message : 'Command failed', ok: false });
    } finally {
      setPending(null);
    }
  };

  return (
    <div style={ms.overlay} role="dialog" aria-modal="true">
      <div style={{ ...ms.modal, maxWidth: 420 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1.25rem' }}>
          <h2 style={{ fontSize: '1.1rem', fontWeight: 700, color: '#f1f5f9', margin: 0 }}>Power Control</h2>
          <button onClick={onClose} style={{ background: 'none', border: 'none', color: '#64748b', cursor: 'pointer', fontSize: '1.1rem' }}>✕</button>
        </div>
        {msg && (
          <div style={{ background: msg.ok ? '#052e16' : '#450a0a', border: `1px solid ${msg.ok ? '#16a34a' : '#dc2626'}`, borderRadius: 6, color: msg.ok ? '#4ade80' : '#f87171', fontSize: '0.85rem', padding: '0.65rem 1rem', marginBottom: '1rem' }}>
            {msg.text}
          </div>
        )}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.6rem' }}>
          {POWER_ACTIONS.map(a => (
            <button key={a.value} onClick={() => execute(a)} disabled={!!pending}
              style={{ background: '#1e293b', border: `1px solid ${a.color}33`, borderRadius: 8, color: a.color, cursor: pending ? 'not-allowed' : 'pointer', fontSize: '0.875rem', fontWeight: 600, padding: '0.65rem 1rem', textAlign: 'left', opacity: pending && pending !== a.value ? 0.5 : 1, display: 'flex', alignItems: 'center', gap: 8 }}>
              {pending === a.value ? '⏳' : '⚡'} {a.label}
            </button>
          ))}
        </div>
        <div style={{ marginTop: '1rem', textAlign: 'right' }}>
          <button onClick={onClose} style={ms.cancelBtn}>Close</button>
        </div>
      </div>
    </div>
  );
};

// ── Boot Control Modal ────────────────────────────────────────────────────────
const BOOT_TARGETS: { label: string; value: BootControlRequest['target']; icon: string }[] = [
  { label: 'PXE Network Boot', value: 'Pxe',      icon: '🌐' },
  { label: 'CD / DVD',         value: 'Cd',       icon: '💿' },
  { label: 'Hard Drive',       value: 'Hdd',      icon: '💾' },
  { label: 'BIOS Setup',       value: 'BiosSetup', icon: '⚙️' },
  { label: 'No Override',      value: 'None',     icon: '🔄' },
];

const BootModal: React.FC<{ deviceId: string; onClose: () => void }> = ({ deviceId, onClose }) => {
  const bootControl = useBootControl();
  const [once, setOnce] = useState(true);
  const [msg, setMsg] = useState<{ text: string; ok: boolean } | null>(null);
  const [pending, setPending] = useState<string | null>(null);

  const execute = async (target: BootControlRequest['target']) => {
    setPending(target); setMsg(null);
    try {
      const res = await bootControl.mutateAsync({ id: deviceId, req: { target, once } });
      setMsg({ text: res.message || 'Boot override set', ok: res.success });
    } catch (e) {
      setMsg({ text: e instanceof Error ? e.message : 'Command failed', ok: false });
    } finally { setPending(null); }
  };

  return (
    <div style={ms.overlay} role="dialog" aria-modal="true">
      <div style={{ ...ms.modal, maxWidth: 420 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1.25rem' }}>
          <h2 style={{ fontSize: '1.1rem', fontWeight: 700, color: '#f1f5f9', margin: 0 }}>Boot Override</h2>
          <button onClick={onClose} style={{ background: 'none', border: 'none', color: '#64748b', cursor: 'pointer', fontSize: '1.1rem' }}>✕</button>
        </div>
        <label style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: '1rem', cursor: 'pointer', fontSize: '0.875rem', color: '#94a3b8' }}>
          <input type="checkbox" checked={once} onChange={e => setOnce(e.target.checked)} />
          One-time override (reverts after next boot)
        </label>
        {msg && (
          <div style={{ background: msg.ok ? '#052e16' : '#450a0a', border: `1px solid ${msg.ok ? '#16a34a' : '#dc2626'}`, borderRadius: 6, color: msg.ok ? '#4ade80' : '#f87171', fontSize: '0.85rem', padding: '0.65rem 1rem', marginBottom: '1rem' }}>
            {msg.text}
          </div>
        )}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.6rem' }}>
          {BOOT_TARGETS.map(t => (
            <button key={t.value} onClick={() => execute(t.value)} disabled={!!pending}
              style={{ background: '#1e293b', border: '1px solid #334155', borderRadius: 8, color: '#e2e8f0', cursor: pending ? 'not-allowed' : 'pointer', fontSize: '0.875rem', fontWeight: 500, padding: '0.65rem 1rem', textAlign: 'left', opacity: pending && pending !== t.value ? 0.5 : 1, display: 'flex', alignItems: 'center', gap: 8 }}>
              {pending === t.value ? '⏳' : t.icon} {t.label}
            </button>
          ))}
        </div>
        <div style={{ marginTop: '1rem', textAlign: 'right' }}>
          <button onClick={onClose} style={ms.cancelBtn}>Close</button>
        </div>
      </div>
    </div>
  );
};

// ── SSE Live Telemetry Hook ───────────────────────────────────────────────────
function useLiveTelemetry(deviceId: string | undefined, enabled: boolean) {
  const [live, setLive] = useState<TelemetryEvent | null>(null);
  const esRef = useRef<EventSource | null>(null);
  const retryRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (!enabled || !deviceId) return;

    let cancelled = false;

    const connect = () => {
      if (cancelled) return;
      const token = localStorage.getItem('auth_token') ?? '';
      const es = streamDeviceTelemetry(deviceId, token);
      esRef.current = es;
      es.onmessage = (e) => {
        try { setLive(JSON.parse(e.data)); } catch { /* ignore */ }
      };
      es.onerror = () => {
        es.close();
        esRef.current = null;
        // Reconnect after 5 seconds
        if (!cancelled) {
          retryRef.current = setTimeout(connect, 5000);
        }
      };
    };

    connect();

    return () => {
      cancelled = true;
      if (retryRef.current) clearTimeout(retryRef.current);
      esRef.current?.close();
      esRef.current = null;
    };
  }, [deviceId, enabled]);

  return live;
}

// ── Storage Enclosures Section ────────────────────────────────────────────────
const EnclosuresSection: React.FC<{ enclosures: StorageEnclosureInfo[] }> = ({ enclosures }) => (
  <Card title="Storage Enclosures" badge={`${enclosures.length}`}>
    <div style={p.tableWrap}>
      <table style={p.table}>
        <thead><tr>{['Name','Backplane ID','Controller','Health'].map(h => <th key={h} style={p.th}>{h}</th>)}</tr></thead>
        <tbody>
          {enclosures.map((e, i) => (
            <tr key={i}>
              <td style={p.td}>{e.name}</td>
              <td style={p.td}>{e.backplane_id || '—'}</td>
              <td style={p.td}>{e.controller || '—'}</td>
              <td style={p.td}><HealthDot health={e.health} />{e.health || '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  </Card>
);
const ProcessorsSection: React.FC<{ procs: ProcessorInfo[] }> = ({ procs }) => (
  <Card title="Processors" badge={`${procs.length} CPU${procs.length !== 1 ? 's' : ''}`}>
    <div style={p.tableWrap}>
      <table style={p.table}>
        <thead>
          <tr>
            {['Name','Model','Socket','Cores/Threads','Speed','Cache','Temp','Health'].map(h => <th key={h} style={p.th}>{h}</th>)}
          </tr>
        </thead>
        <tbody>
          {procs.map((cpu, i) => (
            <tr key={i}>
              <td style={p.td}>{cpu.name}</td>
              <td style={p.td}>{cpu.model || '—'}</td>
              <td style={p.td}>{cpu.socket || '—'}</td>
              <td style={p.td}>{cpu.cores ? `${cpu.cores}C / ${cpu.threads || '?'}T` : '—'}</td>
              <td style={p.td}>{cpu.speed_mhz ? `${cpu.speed_mhz} MHz` : '—'}{cpu.max_speed_mhz ? ` (max ${cpu.max_speed_mhz})` : ''}</td>
              <td style={p.td}>{cpu.cache_size_mib ? `${cpu.cache_size_mib} MiB` : '—'}</td>
              <td style={p.td}>{cpu.temperature_celsius != null ? `${cpu.temperature_celsius}°C` : '—'}</td>
              <td style={p.td}><HealthDot health={cpu.health} />{cpu.health || '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  </Card>
);

const MemorySection: React.FC<{ modules: MemoryInfo[]; totalGB?: number }> = ({ modules, totalGB }) => (
  <Card title="Memory" badge={totalGB ? `${totalGB} GB Total` : `${modules.length} DIMMs`}>
    <div style={p.tableWrap}>
      <table style={p.table}>
        <thead>
          <tr>
            {['Slot','Capacity','Type','Speed','Manufacturer','Serial','ECC','Health'].map(h => <th key={h} style={p.th}>{h}</th>)}
          </tr>
        </thead>
        <tbody>
          {modules.map((m, i) => (
            <tr key={i}>
              <td style={p.td}>{m.location || m.name}</td>
              <td style={p.td}>{m.capacity_gb ? `${m.capacity_gb} GB` : '—'}</td>
              <td style={p.td}>{m.memory_type || '—'}</td>
              <td style={p.td}>{m.speed_mhz ? `${m.speed_mhz} MHz` : '—'}</td>
              <td style={p.td}>{m.manufacturer || '—'}</td>
              <td style={p.td}>{m.serial_number || '—'}</td>
              <td style={p.td}>{m.ecc_enabled == null ? '—' : m.ecc_enabled ? <span style={{ color: '#4ade80' }}>Yes</span> : 'No'}</td>
              <td style={p.td}><HealthDot health={m.health} />{m.health || '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  </Card>
);

const StorageSection: React.FC<{ controllers: StorageControllerInfo[]; drives: DriveInfo[]; vdisks: VirtualDiskInfo[] }> = ({ controllers, drives, vdisks }) => (
  <>
    {controllers.length > 0 && (
      <Card title="Storage Controllers" badge={`${controllers.length}`}>
        <div style={p.tableWrap}>
          <table style={p.table}>
            <thead><tr>{['Name','Model','Firmware','Battery','Health'].map(h => <th key={h} style={p.th}>{h}</th>)}</tr></thead>
            <tbody>
              {controllers.map((c, i) => (
                <tr key={i}>
                  <td style={p.td}>{c.name}</td>
                  <td style={p.td}>{c.model || '—'}</td>
                  <td style={p.td}>{c.firmware_version || '—'}</td>
                  <td style={p.td}>{c.battery_health || '—'}</td>
                  <td style={p.td}><HealthDot health={c.health} />{c.health || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    )}
    {drives.length > 0 && (
      <Card title="Physical Drives" badge={`${drives.length}`}>
        <div style={p.tableWrap}>
          <table style={p.table}>
            <thead><tr>{['Name','Model','Serial','Capacity','Interface','Media','Temp','Health'].map(h => <th key={h} style={p.th}>{h}</th>)}</tr></thead>
            <tbody>
              {drives.map((d, i) => (
                <tr key={i}>
                  <td style={p.td}>{d.name}</td>
                  <td style={p.td}>{d.model || '—'}</td>
                  <td style={p.td}>{d.serial_number || '—'}</td>
                  <td style={p.td}>{d.capacity_gb ? `${d.capacity_gb.toFixed(0)} GB` : '—'}</td>
                  <td style={p.td}>{d.bus_protocol || '—'}</td>
                  <td style={p.td}>{d.media_type || '—'}</td>
                  <td style={p.td}>{d.temperature_celsius != null ? `${d.temperature_celsius}°C` : '—'}</td>
                  <td style={p.td}><HealthDot health={d.health} />{d.health || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    )}
    {vdisks.length > 0 && (
      <Card title="Virtual Disks" badge={`${vdisks.length}`}>
        <div style={p.tableWrap}>
          <table style={p.table}>
            <thead><tr>{['Name','RAID Level','Capacity','Read Policy','Write Policy','Cache','Health'].map(h => <th key={h} style={p.th}>{h}</th>)}</tr></thead>
            <tbody>
              {vdisks.map((v, i) => (
                <tr key={i}>
                  <td style={p.td}>{v.name}</td>
                  <td style={p.td}>{v.raid_level || '—'}</td>
                  <td style={p.td}>{v.capacity_gb ? `${v.capacity_gb.toFixed(0)} GB` : '—'}</td>
                  <td style={p.td}>{v.read_policy || '—'}</td>
                  <td style={p.td}>{v.write_policy || '—'}</td>
                  <td style={p.td}>{v.cache_policy || '—'}</td>
                  <td style={p.td}><HealthDot health={v.health} />{v.health || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    )}
  </>
);

const NICsSection: React.FC<{ nics: FullNICInfo[] }> = ({ nics }) => (
  <Card title="Network Interfaces" badge={`${nics.length}`}>
    <div style={p.tableWrap}>
      <table style={p.table}>
        <thead><tr>{['Name','Model','MAC Address','IPv4','Link','Speed','Firmware','Health'].map(h => <th key={h} style={p.th}>{h}</th>)}</tr></thead>
        <tbody>
          {nics.map((n, i) => (
            <tr key={i}>
              <td style={p.td}>{n.name}</td>
              <td style={p.td}>{n.model || '—'}</td>
              <td style={{ ...p.td, fontFamily: 'monospace', fontSize: '0.75rem' }}>{n.mac_address}</td>
              <td style={p.td}>{n.ipv4_address || '—'}</td>
              <td style={p.td}>
                <span style={{ color: n.link_status?.toLowerCase() === 'up' ? '#4ade80' : '#f87171' }}>
                  {n.link_status || '—'}
                </span>
              </td>
              <td style={p.td}>{n.speed_mbps ? `${n.speed_mbps} Mbps` : '—'}</td>
              <td style={p.td}>{n.firmware_version || '—'}</td>
              <td style={p.td}><HealthDot health={n.health} />{n.health || '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  </Card>
);

const SensorsSection: React.FC<{ temps: TemperatureReading[]; fans: FanReading[]; psus: PowerSupplyInfo[]; volts: VoltageReading[]; totalPower?: number }> = ({ temps, fans, psus, volts, totalPower }) => (
  <>
    {temps.length > 0 && (
      <Card title="Temperature Sensors" badge={`${temps.length} sensors`}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '0.6rem' }}>
          {temps.map((t, i) => (
            <div key={i} style={{ background: '#1e293b', borderRadius: 8, padding: '0.75rem', border: `1px solid ${t.reading_celsius != null && t.upper_threshold_crit != null && t.reading_celsius >= t.upper_threshold_crit ? '#ef4444' : '#334155'}` }}>
              <div style={{ fontSize: '0.7rem', color: '#64748b', marginBottom: 4 }}>{t.name}</div>
              <div style={{ fontSize: '1.4rem', fontWeight: 700, color: t.reading_celsius != null && t.upper_threshold_crit != null && t.reading_celsius >= t.upper_threshold_crit ? '#f87171' : '#e2e8f0' }}>
                {t.reading_celsius != null ? `${t.reading_celsius}°C` : '—'}
              </div>
              {t.upper_threshold_warn && <div style={{ fontSize: '0.65rem', color: '#475569', marginTop: 2 }}>Warn: {t.upper_threshold_warn}°C / Crit: {t.upper_threshold_crit}°C</div>}
            </div>
          ))}
        </div>
      </Card>
    )}
    {fans.length > 0 && (
      <Card title="Fans" badge={`${fans.length}`}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: '0.6rem' }}>
          {fans.map((f, i) => (
            <div key={i} style={{ background: '#1e293b', borderRadius: 8, padding: '0.75rem', border: '1px solid #334155' }}>
              <div style={{ fontSize: '0.7rem', color: '#64748b', marginBottom: 4 }}>{f.name}</div>
              <div style={{ fontSize: '1.2rem', fontWeight: 700, color: '#e2e8f0' }}>{f.reading_rpm != null ? `${f.reading_rpm} RPM` : '—'}</div>
              <HealthDot health={f.health} /><span style={{ fontSize: '0.7rem', color: '#64748b' }}>{f.health || '—'}</span>
            </div>
          ))}
        </div>
      </Card>
    )}
    {psus.length > 0 && (
      <Card title="Power Supplies" badge={totalPower ? `${totalPower}W total` : `${psus.length}`}>
        <div style={p.tableWrap}>
          <table style={p.table}>
            <thead><tr>{['Name','Input (W)','Output (W)','Redundancy','Health','Status'].map(h => <th key={h} style={p.th}>{h}</th>)}</tr></thead>
            <tbody>
              {psus.map((ps, i) => (
                <tr key={i}>
                  <td style={p.td}>{ps.name}</td>
                  <td style={p.td}>{ps.power_input_watts != null ? `${ps.power_input_watts}W` : '—'}</td>
                  <td style={p.td}>{ps.last_power_output_watts != null ? `${ps.last_power_output_watts}W` : ps.power_output_watts != null ? `${ps.power_output_watts}W` : '—'}</td>
                  <td style={p.td}>{ps.redundancy || '—'}</td>
                  <td style={p.td}><HealthDot health={ps.health} />{ps.health || '—'}</td>
                  <td style={p.td}>{ps.status || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    )}
    {volts.length > 0 && (
      <Card title="Voltage Sensors" badge={`${volts.length}`}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: '0.6rem' }}>
          {volts.map((v, i) => (
            <div key={i} style={{ background: '#1e293b', borderRadius: 8, padding: '0.75rem', border: '1px solid #334155' }}>
              <div style={{ fontSize: '0.7rem', color: '#64748b', marginBottom: 4 }}>{v.name}</div>
              <div style={{ fontSize: '1.2rem', fontWeight: 700, color: '#e2e8f0' }}>{v.reading_volts != null ? `${v.reading_volts}V` : '—'}</div>
              <HealthDot health={v.health} /><span style={{ fontSize: '0.7rem', color: '#64748b' }}>{v.health || '—'}</span>
            </div>
          ))}
        </div>
      </Card>
    )}
  </>
);

const PCIeSection: React.FC<{ accelerators: AcceleratorInfo[]; slots: PCIeSlotInfo[] }> = ({ accelerators, slots }) => (
  <>
    {accelerators.length > 0 && (
      <Card title="PCIe Devices / Accelerators" badge={`${accelerators.length}`}>
        <div style={p.tableWrap}>
          <table style={p.table}>
            <thead><tr>{['Name','Model','Manufacturer','Class','Health'].map(h => <th key={h} style={p.th}>{h}</th>)}</tr></thead>
            <tbody>
              {accelerators.map((a, i) => (
                <tr key={i}>
                  <td style={p.td}>{a.name}</td>
                  <td style={p.td}>{a.model || '—'}</td>
                  <td style={p.td}>{a.manufacturer || '—'}</td>
                  <td style={p.td}>{a.device_class || '—'}</td>
                  <td style={p.td}><HealthDot health={a.health} />{a.health || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    )}
    {slots.length > 0 && (
      <Card title="PCIe Slots" badge={`${slots.length}`}>
        <div style={p.tableWrap}>
          <table style={p.table}>
            <thead><tr>{['Name','Slot Type','PCIe Type','Device','Status'].map(h => <th key={h} style={p.th}>{h}</th>)}</tr></thead>
            <tbody>
              {slots.map((s, i) => (
                <tr key={i}>
                  <td style={p.td}>{s.name}</td>
                  <td style={p.td}>{s.slot_type || '—'}</td>
                  <td style={p.td}>{s.pcie_type || '—'}</td>
                  <td style={p.td}>{s.device_name || '—'}</td>
                  <td style={p.td}>{s.status || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    )}
  </>
);

const EventLogsSection: React.FC<{ sel: SELEntry[]; lcLogs: LifecycleLogEntry[] }> = ({ sel, lcLogs }) => (
  <>
    {sel.length > 0 && (
      <Card title="System Event Log (SEL)" badge={`${sel.length} entries`}>
        <div style={p.tableWrap}>
          <table style={p.table}>
            <thead><tr>{['ID','Severity','Message','Time'].map(h => <th key={h} style={p.th}>{h}</th>)}</tr></thead>
            <tbody>
              {sel.slice(0, 50).map((e, i) => (
                <tr key={i}>
                  <td style={{ ...p.td, fontFamily: 'monospace', fontSize: '0.75rem' }}>{e.id}</td>
                  <td style={p.td}><SevBadge sev={e.severity} /></td>
                  <td style={p.td}>{e.message}</td>
                  <td style={{ ...p.td, whiteSpace: 'nowrap' }}>{e.created ? new Date(e.created).toLocaleString() : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    )}
    {lcLogs.length > 0 && (
      <Card title="Lifecycle Logs" badge={`${lcLogs.length} entries`}>
        <div style={p.tableWrap}>
          <table style={p.table}>
            <thead><tr>{['ID','Severity','Category','Message','Time'].map(h => <th key={h} style={p.th}>{h}</th>)}</tr></thead>
            <tbody>
              {lcLogs.slice(0, 50).map((e, i) => (
                <tr key={i}>
                  <td style={{ ...p.td, fontFamily: 'monospace', fontSize: '0.75rem' }}>{e.id}</td>
                  <td style={p.td}><SevBadge sev={e.severity} /></td>
                  <td style={p.td}>{e.category || '—'}</td>
                  <td style={p.td}>{e.message}</td>
                  <td style={{ ...p.td, whiteSpace: 'nowrap' }}>{e.created ? new Date(e.created).toLocaleString() : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    )}
  </>
);

// ── Tab definitions ───────────────────────────────────────────────────────────
type Tab = 'overview' | 'inventory' | 'sensors' | 'storage' | 'network' | 'pcie' | 'logs' | 'metrics';
const TABS: { id: Tab; label: string }[] = [
  { id: 'overview',  label: 'Overview' },
  { id: 'inventory', label: 'Hardware' },
  { id: 'sensors',   label: 'Sensors' },
  { id: 'storage',   label: 'Storage' },
  { id: 'network',   label: 'Network' },
  { id: 'pcie',      label: 'PCIe' },
  { id: 'logs',      label: 'Event Logs' },
  { id: 'metrics',   label: 'Metrics' },
];

// ── Main DeviceDetail Component ───────────────────────────────────────────────
const DeviceDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [tab, setTab] = useState<Tab>('overview');
  const [showEdit, setShowEdit] = useState(false);
  const [showPower, setShowPower] = useState(false);
  const [showBoot, setShowBoot] = useState(false);
  const [syncMsg, setSyncMsg] = useState<{ text: string; ok: boolean } | null>(null);

  const deleteDevice = useDeleteDevice();
  const syncDevice = useSyncDevice();

  const { data: device, isLoading: deviceLoading, error: deviceError } = useDevice(id ?? '');

  const liveTelemetry = useLiveTelemetry(id, tab === 'sensors');

  const { data: inventory, isLoading: invLoading } = useQuery({
    queryKey: ['inventory', id],
    queryFn: () => fetchDeviceInventory(id!),
    enabled: !!id,
    staleTime: 10 * 60 * 1000, // 10 min
    refetchInterval: 10 * 60 * 1000,
  });

  const { data: metrics } = useQuery({
    queryKey: ['metrics', id],
    queryFn: () => fetchDeviceMetrics(id!),
    enabled: !!id,
    staleTime: 5000,
    refetchInterval: 5000, // 5s real-time telemetry
  });

  const handleSync = async () => {
    if (!id) return;
    setSyncMsg(null);
    try {
      const res = await syncDevice.mutateAsync(id);
      setSyncMsg({ text: res.success ? 'Sync completed successfully' : res.message, ok: res.success });
      queryClient.invalidateQueries({ queryKey: ['inventory', id] });
    } catch (e) {
      setSyncMsg({ text: e instanceof Error ? e.message : 'Sync failed', ok: false });
    }
  };

  const handleDelete = async () => {
    if (!id || !window.confirm('Delete this device? This cannot be undone.')) return;
    await deleteDevice.mutateAsync(id);
    navigate('/devices');
  };

  if (deviceLoading) return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '60vh', color: '#64748b' }}>
      Loading device…
    </div>
  );

  if (deviceError || !device) return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '60vh', gap: '1rem' }}>
      <div style={{ color: '#f87171', fontSize: '1rem' }}>Device not found</div>
      <button onClick={() => navigate('/devices')} style={{ background: '#1e293b', border: '1px solid #334155', borderRadius: 6, color: '#94a3b8', cursor: 'pointer', padding: '0.5rem 1rem' }}>← Back</button>
    </div>
  );

  const statusCfg = STATUS_CFG[device.status] ?? STATUS_CFG.unknown;
  const inv = inventory;

  return (
    <div style={{ minHeight: '100vh', background: '#020617', color: '#e2e8f0', fontFamily: 'system-ui, sans-serif' }}>
      {/* Header */}
      <div style={{ background: '#0f172a', borderBottom: '1px solid #1e293b', padding: '1rem 1.5rem' }}>
        <div style={{ maxWidth: 1400, margin: '0 auto' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '0.5rem' }}>
            <button onClick={() => navigate('/devices')} style={{ background: 'none', border: 'none', color: '#64748b', cursor: 'pointer', fontSize: '0.875rem', padding: 0 }}>← Devices</button>
            <span style={{ color: '#334155' }}>/</span>
            <span style={{ color: '#94a3b8', fontSize: '0.875rem' }}>{device.hostname}</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', flexWrap: 'wrap', gap: '1rem' }}>
            <div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '0.25rem' }}>
                <h1 style={{ fontSize: '1.5rem', fontWeight: 700, color: '#f1f5f9', margin: 0 }}>{device.hostname}</h1>
                <span style={{ background: statusCfg.bg, border: `1px solid ${statusCfg.border}`, borderRadius: 20, color: statusCfg.text, fontSize: '0.7rem', fontWeight: 700, padding: '2px 10px', display: 'flex', alignItems: 'center', gap: 5 }}>
                  <span style={{ width: 6, height: 6, borderRadius: '50%', background: statusCfg.dot, display: 'inline-block' }} />
                  {statusCfg.label}
                </span>
              </div>
              <div style={{ display: 'flex', gap: '1.5rem', flexWrap: 'wrap' }}>
                <span style={{ fontSize: '0.8rem', color: '#64748b' }}>{device.ip_address}{device.bmc_ip_address ? ` · BMC: ${device.bmc_ip_address}` : ''}</span>
                {device.vendor && <span style={{ fontSize: '0.8rem', color: '#64748b' }}>{device.vendor}</span>}
                {device.management_controller && <span style={{ fontSize: '0.8rem', color: '#64748b' }}>{device.management_controller}</span>}
                {device.location && <span style={{ fontSize: '0.8rem', color: '#64748b' }}>📍 {device.location}</span>}
                {inv?.power_state && <span style={{ fontSize: '0.8rem', color: inv.power_state === 'On' ? '#4ade80' : '#f87171' }}>⚡ {inv.power_state}</span>}
              </div>
            </div>
            <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
              <button onClick={() => setShowEdit(true)} style={{ background: '#1e293b', border: '1px solid #334155', borderRadius: 6, color: '#94a3b8', cursor: 'pointer', fontSize: '0.8rem', fontWeight: 500, padding: '0.45rem 0.9rem' }}>Edit</button>
              <button onClick={handleSync} disabled={syncDevice.isPending} style={{ background: '#1e293b', border: '1px solid #3b82f6', borderRadius: 6, color: '#60a5fa', cursor: syncDevice.isPending ? 'not-allowed' : 'pointer', fontSize: '0.8rem', fontWeight: 500, padding: '0.45rem 0.9rem', opacity: syncDevice.isPending ? 0.6 : 1 }}>
                {syncDevice.isPending ? '⏳ Syncing…' : '🔄 Sync'}
              </button>
              <button onClick={() => setShowPower(true)} style={{ background: '#1e293b', border: '1px solid #f59e0b', borderRadius: 6, color: '#fbbf24', cursor: 'pointer', fontSize: '0.8rem', fontWeight: 500, padding: '0.45rem 0.9rem' }}>⚡ Power</button>
              <button onClick={() => setShowBoot(true)} style={{ background: '#1e293b', border: '1px solid #8b5cf6', borderRadius: 6, color: '#a78bfa', cursor: 'pointer', fontSize: '0.8rem', fontWeight: 500, padding: '0.45rem 0.9rem' }}>🥾 Boot</button>
              <button onClick={handleDelete} style={{ background: '#1e293b', border: '1px solid #dc2626', borderRadius: 6, color: '#f87171', cursor: 'pointer', fontSize: '0.8rem', fontWeight: 500, padding: '0.45rem 0.9rem' }}>Delete</button>
            </div>
          </div>
          {syncMsg && (
            <div style={{ marginTop: '0.75rem', background: syncMsg.ok ? '#052e16' : '#450a0a', border: `1px solid ${syncMsg.ok ? '#16a34a' : '#dc2626'}`, borderRadius: 6, color: syncMsg.ok ? '#4ade80' : '#f87171', fontSize: '0.8rem', padding: '0.5rem 0.9rem' }}>
              {syncMsg.text}
            </div>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div style={{ background: '#0f172a', borderBottom: '1px solid #1e293b', padding: '0 1.5rem' }}>
        <div style={{ maxWidth: 1400, margin: '0 auto', display: 'flex', gap: 0, overflowX: 'auto' }}>
          {TABS.map(t => (
            <button key={t.id} onClick={() => setTab(t.id)}
              style={{ background: 'none', border: 'none', borderBottom: tab === t.id ? '2px solid #3b82f6' : '2px solid transparent', color: tab === t.id ? '#60a5fa' : '#64748b', cursor: 'pointer', fontSize: '0.85rem', fontWeight: tab === t.id ? 600 : 400, padding: '0.75rem 1rem', whiteSpace: 'nowrap', transition: 'color 0.15s' }}>
              {t.label}
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      <div style={{ maxWidth: 1400, margin: '0 auto', padding: '1.5rem' }}>

        {/* OVERVIEW TAB */}
        {tab === 'overview' && (
          <>
            {/* System Identity */}
            <Card title="System Information">
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '1rem' }}>
                <InfoField label="Manufacturer" value={inv?.manufacturer} />
                <InfoField label="Model" value={inv?.system_model} />
                <InfoField label="Service Tag" value={inv?.service_tag} />
                <InfoField label="Asset Tag" value={inv?.asset_tag} />
                <InfoField label="System UUID" value={inv?.system_uuid} />
                <InfoField label="System Revision" value={inv?.system_revision} />
                <InfoField label="Power State" value={inv?.power_state} />
                <InfoField label="Health Status" value={inv?.health_status ? <span style={{ color: healthColor(inv.health_status) }}>{inv.health_status}</span> : undefined} />
                <InfoField label="BIOS Version" value={inv?.firmware_bios} />
                <InfoField label="Boot Mode" value={inv?.boot_mode} />
                <InfoField label="Uptime" value={inv?.system_uptime_seconds ? `${Math.floor(inv.system_uptime_seconds / 3600)}h ${Math.floor((inv.system_uptime_seconds % 3600) / 60)}m` : undefined} />
                <InfoField label="Lifecycle Controller" value={inv?.lifecycle_controller_version} />
              </div>
            </Card>

            {/* BMC Info */}
            <Card title="BMC / Management Controller">
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '1rem' }}>
                <InfoField label="BMC Name" value={inv?.bmc_name} />
                <InfoField label="BMC Firmware" value={inv?.firmware_bmc} />
                <InfoField label="Hardware Version" value={inv?.bmc_hardware_version} />
                <InfoField label="MAC Address" value={inv?.bmc_mac_address} />
                <InfoField label="DNS Name" value={inv?.bmc_dns_name} />
                <InfoField label="License" value={inv?.bmc_license} />
              </div>
            </Card>

            {/* Quick stats */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))', gap: '0.75rem', marginBottom: '1rem' }}>
              {[
                { label: 'CPUs', value: inv?.processors?.length ?? '—', sub: inv?.cpu_model },
                { label: 'Memory', value: inv?.ram_total_gb ? `${inv.ram_total_gb} GB` : '—', sub: `${inv?.memory_modules?.length ?? 0} DIMMs` },
                { label: 'Drives', value: inv?.drives?.length ?? '—', sub: `${inv?.storage_controllers?.length ?? 0} controllers` },
                { label: 'NICs', value: inv?.nics?.length ?? '—', sub: undefined },
                { label: 'Power', value: inv?.total_power_watts ? `${inv.total_power_watts}W` : '—', sub: `${inv?.power_supplies?.length ?? 0} PSUs` },
                { label: 'Temps', value: inv?.temperatures?.length ?? '—', sub: `${inv?.fans?.length ?? 0} fans` },
              ].map(s => (
                <div key={s.label} style={{ background: '#0f172a', border: '1px solid #1e293b', borderRadius: 10, padding: '1rem' }}>
                  <div style={{ fontSize: '0.65rem', color: '#475569', textTransform: 'uppercase', letterSpacing: '0.1em', marginBottom: 4 }}>{s.label}</div>
                  <div style={{ fontSize: '1.5rem', fontWeight: 700, color: '#f1f5f9' }}>{s.value}</div>
                  {s.sub && <div style={{ fontSize: '0.7rem', color: '#64748b', marginTop: 2 }}>{s.sub}</div>}
                </div>
              ))}
            </div>

            {/* Intrusion */}
            {inv?.intrusion_status && (
              <div style={{ background: inv.intrusion_status.toLowerCase() === 'normal' ? '#052e16' : '#450a0a', border: `1px solid ${inv.intrusion_status.toLowerCase() === 'normal' ? '#16a34a' : '#dc2626'}`, borderRadius: 8, padding: '0.75rem 1rem', marginBottom: '1rem', fontSize: '0.875rem', color: inv.intrusion_status.toLowerCase() === 'normal' ? '#4ade80' : '#f87171' }}>
                🔒 Chassis Intrusion: {inv.intrusion_status}
              </div>
            )}

            {/* Last sync */}
            {device.last_sync_at && (
              <div style={{ fontSize: '0.75rem', color: '#475569', textAlign: 'right' }}>
                Last synced: {new Date(device.last_sync_at).toLocaleString()}
              </div>
            )}
          </>
        )}

        {/* HARDWARE INVENTORY TAB */}
        {tab === 'inventory' && (
          <>
            {invLoading && <div style={{ color: '#64748b', padding: '2rem' }}>Loading inventory…</div>}
            {inv?.processors && inv.processors.length > 0 && <ProcessorsSection procs={inv.processors} />}
            {inv?.memory_modules && inv.memory_modules.length > 0 && <MemorySection modules={inv.memory_modules} totalGB={inv.ram_total_gb ?? undefined} />}
            {!inv && !invLoading && <div style={{ color: '#64748b', padding: '2rem' }}>No inventory data. Run a sync to collect hardware information.</div>}
          </>
        )}

        {/* SENSORS TAB */}
        {tab === 'sensors' && (
          <>
            {liveTelemetry && (
              <div style={{ background: '#0f172a', border: '1px solid #1e293b', borderRadius: 10, padding: '0.75rem 1rem', marginBottom: '1rem', display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
                <span style={{ width: 8, height: 8, borderRadius: '50%', background: '#22c55e', display: 'inline-block', boxShadow: '0 0 6px #22c55e' }} />
                <span style={{ fontSize: '0.75rem', color: '#4ade80', fontWeight: 600 }}>LIVE</span>
                <span style={{ fontSize: '0.75rem', color: '#64748b' }}>
                  Power: <span style={{ color: liveTelemetry.power_state === 'On' ? '#4ade80' : '#f87171' }}>{liveTelemetry.power_state ?? '—'}</span>
                  {liveTelemetry.total_power_watts != null && <> · {liveTelemetry.total_power_watts}W</>}
                  {liveTelemetry.health_status && <> · Health: <span style={{ color: healthColor(liveTelemetry.health_status) }}>{liveTelemetry.health_status}</span></>}
                </span>
                <span style={{ marginLeft: 'auto', fontSize: '0.7rem', color: '#334155' }}>
                  {new Date(liveTelemetry.timestamp * 1000).toLocaleTimeString()}
                </span>
              </div>
            )}
            {invLoading && <div style={{ color: '#64748b', padding: '2rem' }}>Loading sensors…</div>}
            <SensorsSection
              temps={liveTelemetry?.temperatures ?? inv?.temperatures ?? []}
              fans={liveTelemetry?.fans ?? inv?.fans ?? []}
              psus={liveTelemetry?.power_supplies ?? inv?.power_supplies ?? []}
              volts={liveTelemetry?.voltages ?? inv?.voltages ?? []}
              totalPower={liveTelemetry?.total_power_watts ?? inv?.total_power_watts ?? undefined}
            />
            {!inv && !invLoading && <div style={{ color: '#64748b', padding: '2rem' }}>No sensor data. Run a sync first.</div>}
          </>
        )}

        {/* STORAGE TAB */}
        {tab === 'storage' && (
          <>
            {invLoading && <div style={{ color: '#64748b', padding: '2rem' }}>Loading storage…</div>}
            <StorageSection
              controllers={inv?.storage_controllers ?? []}
              drives={inv?.drives ?? []}
              vdisks={inv?.virtual_disks ?? []}
            />
            {inv?.storage_enclosures && inv.storage_enclosures.length > 0 && (
              <EnclosuresSection enclosures={inv.storage_enclosures} />
            )}
            {!inv && !invLoading && <div style={{ color: '#64748b', padding: '2rem' }}>No storage data. Run a sync first.</div>}
          </>
        )}

        {/* NETWORK TAB */}
        {tab === 'network' && (
          <>
            {invLoading && <div style={{ color: '#64748b', padding: '2rem' }}>Loading network interfaces…</div>}
            {inv?.nics && inv.nics.length > 0 ? <NICsSection nics={inv.nics} /> : !invLoading && <div style={{ color: '#64748b', padding: '2rem' }}>No NIC data. Run a sync first.</div>}
          </>
        )}

        {/* PCIe TAB */}
        {tab === 'pcie' && (
          <>
            {invLoading && <div style={{ color: '#64748b', padding: '2rem' }}>Loading PCIe devices…</div>}
            <PCIeSection accelerators={inv?.accelerators ?? []} slots={inv?.pcie_slots ?? []} />
            {!inv && !invLoading && <div style={{ color: '#64748b', padding: '2rem' }}>No PCIe data. Run a sync first.</div>}
          </>
        )}

        {/* EVENT LOGS TAB */}
        {tab === 'logs' && (
          <>
            {invLoading && <div style={{ color: '#64748b', padding: '2rem' }}>Loading logs…</div>}
            <EventLogsSection sel={inv?.sel_entries ?? []} lcLogs={inv?.lifecycle_logs ?? []} />
            {!inv && !invLoading && <div style={{ color: '#64748b', padding: '2rem' }}>No log data. Run a sync first.</div>}
          </>
        )}

        {/* METRICS TAB */}
        {tab === 'metrics' && (
          <>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))', gap: '1rem' }}>
              {metrics?.temperature && metrics.temperature.length > 0 && (
                <MetricChart title="Temperature" data={metrics.temperature} unit="°C" color="#f87171" gradId="grad-temp" />
              )}
              {metrics?.fan_speed && metrics.fan_speed.length > 0 && (
                <MetricChart title="Fan Speed" data={metrics.fan_speed} unit=" RPM" color="#60a5fa" gradId="grad-fan" />
              )}
              {metrics?.power_consumption && metrics.power_consumption.length > 0 && (
                <MetricChart title="Power Consumption" data={metrics.power_consumption} unit="W" color="#a78bfa" gradId="grad-pwr" />
              )}
            </div>
            {(!metrics || (!metrics.temperature?.length && !metrics.fan_speed?.length && !metrics.power_consumption?.length)) && (
              <div style={{ color: '#64748b', padding: '2rem' }}>No metrics data available. Metrics are collected via VictoriaMetrics from the collector agent.</div>
            )}
            <div style={{ fontSize: '0.7rem', color: '#334155', marginTop: '0.5rem' }}>Auto-refreshes every 5 seconds</div>
          </>
        )}

      </div>

      {/* Modals */}
      {showEdit && <EditDeviceModal device={device} onClose={() => setShowEdit(false)} onSaved={() => queryClient.invalidateQueries({ queryKey: deviceKeys.detail(id!) })} />}
      {showPower && id && <PowerModal deviceId={id} onClose={() => setShowPower(false)} />}
      {showBoot && id && <BootModal deviceId={id} onClose={() => setShowBoot(false)} />}
    </div>
  );
};

export default DeviceDetail;
