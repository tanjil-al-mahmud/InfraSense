import React, { useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { deviceKeys, useDeleteDevice } from '../hooks/useDevices';
import { fetchDevices } from '../services/deviceApi';
import { DeviceType, DeviceStatus, DeviceListParams, Device } from '../types/device';
import { useToast } from '../contexts/ToastContext';
import AddDeviceModal from '../components/devices/AddDeviceModal';

const PAGE_SIZE = 20;
const REFRESH_INTERVAL_MS = 30_000;

const STATUS_COLORS: Record<DeviceStatus, { bg: string; text: string; label: string; dot: string }> = {
  healthy:     { bg: '#052e16', text: '#4ade80', label: 'Healthy',  dot: '#22c55e' },
  warning:     { bg: '#422006', text: '#fbbf24', label: 'Warning',  dot: '#f59e0b' },
  critical:    { bg: '#450a0a', text: '#f87171', label: 'Critical', dot: '#ef4444' },
  unavailable: { bg: '#111827', text: '#9ca3af', label: 'Offline',  dot: '#6b7280' },
  unknown:     { bg: '#111827', text: '#9ca3af', label: 'Unknown',  dot: '#6b7280' },
};

const DEVICE_TYPE_LABELS: Record<DeviceType, string> = {
  ipmi: 'IPMI', redfish: 'Redfish', snmp: 'SNMP',
  linux_agent: 'Linux Agent', windows_agent: 'Windows Agent',
};

const ALL_DEVICE_TYPES: DeviceType[] = ['ipmi', 'redfish', 'snmp', 'linux_agent', 'windows_agent'];
const ALL_STATUSES: DeviceStatus[] = ['healthy', 'warning', 'critical', 'unavailable', 'unknown'];

// Derive vendor label from device_type string
function getVendorLabel(deviceType: string): string {
  if (deviceType.startsWith('dell_')) return 'Dell';
  if (deviceType.startsWith('hpe_')) return 'HPE';
  if (deviceType.startsWith('supermicro_')) return 'Supermicro';
  if (deviceType.startsWith('lenovo_')) return 'Lenovo';
  if (deviceType.startsWith('cisco_')) return 'Cisco';
  if (deviceType.startsWith('huawei_')) return 'Huawei';
  if (deviceType.startsWith('fujitsu_')) return 'Fujitsu';
  if (deviceType.startsWith('asus_')) return 'ASUS';
  if (deviceType.startsWith('gigabyte_')) return 'Gigabyte';
  if (deviceType.startsWith('ericsson_')) return 'Ericsson';
  if (deviceType.startsWith('ieit_')) return 'IEIT';
  if (deviceType.startsWith('apc_')) return 'APC';
  if (deviceType.startsWith('eaton_')) return 'Eaton';
  if (deviceType.startsWith('generic_')) return 'Generic';
  if (deviceType === 'linux_agent') return 'Linux';
  if (deviceType === 'windows_agent') return 'Windows';
  return deviceType;
}

function getTypeLabel(deviceType: string): string {
  if (deviceType.includes('redfish')) return 'Redfish';
  if (deviceType.includes('ipmi')) return 'IPMI';
  if (deviceType.includes('snmp')) return 'SNMP';
  if (deviceType === 'linux_agent') return 'Linux Agent';
  if (deviceType === 'windows_agent') return 'Windows Agent';
  return DEVICE_TYPE_LABELS[deviceType as DeviceType] ?? deviceType;
}

const Devices: React.FC = () => {
  const navigate = useNavigate();
  const { showToast } = useToast();
  const deleteDevice = useDeleteDevice();

  const [page, setPage] = useState(1);
  const [deviceTypeFilter, setDeviceTypeFilter] = useState<DeviceType | ''>('');
  const [statusFilter, setStatusFilter] = useState<DeviceStatus | ''>('');
  const [searchQuery, setSearchQuery] = useState('');
  const [showAddModal, setShowAddModal] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Device | null>(null);

  const queryParams: DeviceListParams = {
    page,
    page_size: PAGE_SIZE,
    ...(deviceTypeFilter ? { device_type: deviceTypeFilter } : {}),
    ...(statusFilter ? { status: statusFilter } : {}),
    ...(searchQuery ? { search: searchQuery } : {}),
  };

  const { data, isLoading, isError, error } = useQuery({
    queryKey: deviceKeys.list(queryParams),
    queryFn: () => fetchDevices(queryParams),
    staleTime: 30_000,
    refetchInterval: REFRESH_INTERVAL_MS,
    refetchOnWindowFocus: true,
  });

  const devices = data?.data ?? [];
  const totalPages = data ? Math.ceil(data.meta.total / PAGE_SIZE) : 1;

  const resetPage = useCallback(() => setPage(1), []);

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await deleteDevice.mutateAsync(deleteTarget.id);
      showToast(`Device "${deleteTarget.hostname}" deleted successfully`, 'success');
      setDeleteTarget(null);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to delete device';
      showToast(`Failed to delete device: ${msg}`, 'error');
    }
  };

  // Loading skeleton rows
  const SkeletonRows = () => (
    <>
      {Array.from({ length: 5 }).map((_, i) => (
        <tr key={i}>
          {Array.from({ length: 8 }).map((_, j) => (
            <td key={j} style={styles.td}>
              <div style={{ height: '14px', backgroundColor: '#1e293b', borderRadius: '4px', width: j === 0 ? '60px' : '80%' }} />
            </td>
          ))}
        </tr>
      ))}
    </>
  );

  return (
    <div style={styles.page}>
      {/* Header */}
      <div style={styles.pageHeader}>
        <h1 style={styles.pageTitle}>Devices</h1>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
          {data && (
            <span style={styles.totalCount}>
              {data.meta.total} device{data.meta.total !== 1 ? 's' : ''}
            </span>
          )}
          <button style={styles.addBtn} onClick={() => setShowAddModal(true)}>
            + Add Device
          </button>
        </div>
      </div>

      {/* Filters */}
      <div style={styles.filterBar}>
        <input
          type="search"
          placeholder="Search by name or IP..."
          value={searchQuery}
          onChange={(e) => { setSearchQuery(e.target.value); resetPage(); }}
          style={styles.searchInput}
          aria-label="Search devices"
        />
        <select
          value={deviceTypeFilter}
          onChange={(e) => { setDeviceTypeFilter(e.target.value as DeviceType | ''); resetPage(); }}
          style={styles.select}
          aria-label="Filter by device type"
        >
          <option value="">All Types</option>
          {ALL_DEVICE_TYPES.map((t) => (
            <option key={t} value={t}>{DEVICE_TYPE_LABELS[t]}</option>
          ))}
        </select>
        <select
          value={statusFilter}
          onChange={(e) => { setStatusFilter(e.target.value as DeviceStatus | ''); resetPage(); }}
          style={styles.select}
          aria-label="Filter by status"
        >
          <option value="">All Statuses</option>
          {ALL_STATUSES.map((s) => (
            <option key={s} value={s}>{STATUS_COLORS[s].label}</option>
          ))}
        </select>
      </div>

      {/* Error */}
      {isError && error && (
        <div style={styles.errorBanner} role="alert">Failed to load devices: {error.message}</div>
      )}

      {/* Table */}
      <div style={styles.tableWrapper}>
        <table style={styles.table}>
          <thead>
            <tr>
              <th style={styles.th}>Name</th>
              <th style={styles.th}>IP / Hostname</th>
              <th style={styles.th}>Vendor</th>
              <th style={styles.th}>Type</th>
              <th style={styles.th}>Location</th>
              <th style={styles.th}>Status</th>
              <th style={styles.th}>Last Seen</th>
              <th style={styles.th}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <SkeletonRows />
            ) : devices.length === 0 ? (
              <tr>
                <td colSpan={8} style={{ ...styles.td, textAlign: 'center', padding: '3rem', color: '#64748b' }}>
                  No devices found. Click <strong style={{ color: '#94a3b8' }}>+ Add Device</strong> to get started.
                </td>
              </tr>
            ) : (
              devices.map((device) => {
                const sc = STATUS_COLORS[device.status] ?? STATUS_COLORS.unknown;
                return (
                  <tr
                    key={device.id}
                    style={styles.tr}
                    onClick={() => navigate(`/devices/${device.id}`)}
                    role="button"
                    tabIndex={0}
                    onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') navigate(`/devices/${device.id}`); }}
                  >
                    <td style={{ ...styles.td, fontWeight: 500 }}>{device.hostname}</td>
                    <td style={styles.td}>{device.ip_address}</td>
                    <td style={styles.td}>{getVendorLabel(device.device_type)}</td>
                    <td style={styles.td}>{getTypeLabel(device.device_type)}</td>
                    <td style={styles.td}>{device.location ?? '—'}</td>
                    <td style={styles.td}>
                      <span style={{ ...styles.badge, backgroundColor: sc.bg, color: sc.text }}>
                        <span style={{ width: 6, height: 6, borderRadius: '50%', background: sc.dot, display: 'inline-block' }} />
                        {sc.label}
                      </span>
                    </td>
                    <td style={styles.td}>{device.last_seen ? new Date(device.last_seen).toLocaleString() : '—'}</td>
                    <td style={styles.td} onClick={(e) => e.stopPropagation()}>
                      <div style={{ display: 'flex', gap: '0.4rem' }}>
                        <button style={styles.actionBtn} onClick={() => navigate(`/devices/${device.id}`)}>View</button>
                        <button style={styles.actionBtn} onClick={() => navigate(`/devices/${device.id}`)}>Edit</button>
                        <button
                          style={{ ...styles.actionBtn, ...styles.deleteActionBtn }}
                          onClick={() => setDeleteTarget(device)}
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div style={styles.pagination}>
          <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page === 1} style={{ ...styles.pageBtn, ...(page === 1 ? styles.pageBtnDisabled : {}) }}>Previous</button>
          <span style={styles.pageInfo}>Page {page} of {totalPages}</span>
          <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages} style={{ ...styles.pageBtn, ...(page >= totalPages ? styles.pageBtnDisabled : {}) }}>Next</button>
        </div>
      )}

      {/* Add Device Modal */}
      {showAddModal && (
        <AddDeviceModal
          onClose={() => setShowAddModal(false)}
          onSuccess={(device) => {
            setShowAddModal(false);
            showToast(`Device "${device.hostname}" added successfully`, 'success');
          }}
        />
      )}

      {/* Delete Confirm Dialog */}
      {deleteTarget && (
        <div style={styles.overlay} role="dialog" aria-modal="true" aria-label="Confirm delete">
          <div style={styles.confirmCard}>
            <h2 style={{ fontSize: '1.125rem', fontWeight: 600, color: '#f1f5f9', margin: '0 0 0.75rem' }}>Delete Device</h2>
            <p style={{ color: '#94a3b8', fontSize: '0.875rem', margin: '0 0 1.5rem' }}>
              Are you sure you want to delete <strong style={{ color: '#e2e8f0' }}>{deleteTarget.hostname}</strong>? This action cannot be undone.
            </p>
            {deleteDevice.isError && (
              <div style={styles.errorBanner} role="alert">{deleteDevice.error?.message}</div>
            )}
            <div style={{ display: 'flex', gap: '0.75rem', justifyContent: 'flex-end' }}>
              <button onClick={() => setDeleteTarget(null)} style={styles.cancelBtn} disabled={deleteDevice.isPending}>Cancel</button>
              <button
                onClick={handleDelete}
                style={{ ...styles.deleteBtn, ...(deleteDevice.isPending ? { opacity: 0.6, cursor: 'not-allowed' } : {}) }}
                disabled={deleteDevice.isPending}
              >
                {deleteDevice.isPending ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

const styles: Record<string, React.CSSProperties> = {
  page: { padding: '1.5rem', maxWidth: '1400px', margin: '0 auto' },
  pageHeader: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1rem', flexWrap: 'wrap', gap: '0.75rem' },
  pageTitle: { fontSize: '1.5rem', fontWeight: 700, color: '#f1f5f9', margin: 0 },
  totalCount: { fontSize: '0.875rem', color: '#64748b' },
  addBtn: { backgroundColor: '#2563eb', border: 'none', borderRadius: '6px', color: '#fff', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 600, padding: '0.5rem 1.25rem' },
  filterBar: { display: 'flex', flexWrap: 'wrap', gap: '0.75rem', marginBottom: '1.25rem' },
  searchInput: { border: '1px solid #334155', borderRadius: '6px', fontSize: '0.875rem', padding: '0.5rem 0.75rem', minWidth: '220px', flex: '1 1 220px', background: '#1e293b', color: '#e2e8f0' },
  select: { border: '1px solid #334155', borderRadius: '6px', fontSize: '0.875rem', padding: '0.5rem 0.75rem', backgroundColor: '#1e293b', color: '#e2e8f0', minWidth: '150px' },
  errorBanner: { backgroundColor: '#450a0a', border: '1px solid #dc2626', borderRadius: '6px', color: '#f87171', fontSize: '0.875rem', padding: '0.75rem 1rem', marginBottom: '1rem' },
  tableWrapper: { overflowX: 'auto', borderRadius: '10px', border: '1px solid #1e293b' },
  table: { width: '100%', borderCollapse: 'collapse', fontSize: '0.875rem' },
  th: { backgroundColor: '#1e293b', borderBottom: '1px solid #334155', color: '#64748b', fontWeight: 700, fontSize: '0.7rem', textTransform: 'uppercase', letterSpacing: '0.08em', padding: '0.75rem 1rem', textAlign: 'left', whiteSpace: 'nowrap' },
  tr: { cursor: 'pointer', borderBottom: '1px solid #1e293b' },
  td: { padding: '0.75rem 1rem', color: '#94a3b8', verticalAlign: 'middle' },
  badge: { borderRadius: '9999px', display: 'inline-flex', alignItems: 'center', gap: 5, fontSize: '0.75rem', fontWeight: 600, padding: '0.2rem 0.6rem', whiteSpace: 'nowrap' },
  actionBtn: { backgroundColor: '#1e293b', border: '1px solid #334155', borderRadius: '4px', color: '#94a3b8', cursor: 'pointer', fontSize: '0.75rem', fontWeight: 500, padding: '0.25rem 0.6rem' },
  deleteActionBtn: { backgroundColor: '#450a0a', borderColor: '#dc2626', color: '#f87171' },
  pagination: { display: 'flex', alignItems: 'center', gap: '1rem', justifyContent: 'center', marginTop: '1.25rem' },
  pageBtn: { backgroundColor: '#2563eb', border: 'none', borderRadius: '4px', color: '#fff', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 500, padding: '0.5rem 1rem' },
  pageBtnDisabled: { backgroundColor: '#1e3a5f', cursor: 'not-allowed', color: '#475569' },
  pageInfo: { color: '#64748b', fontSize: '0.875rem' },
  overlay: { position: 'fixed', inset: 0, backgroundColor: 'rgba(0,0,0,0.7)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000, padding: '1rem' },
  confirmCard: { backgroundColor: '#0f172a', border: '1px solid #1e293b', borderRadius: '12px', boxShadow: '0 25px 60px rgba(0,0,0,0.5)', maxWidth: '420px', width: '100%', padding: '1.5rem' },
  cancelBtn: { backgroundColor: '#1e293b', border: '1px solid #334155', borderRadius: '6px', color: '#94a3b8', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 500, padding: '0.5rem 1rem' },
  deleteBtn: { backgroundColor: '#dc2626', border: 'none', borderRadius: '6px', color: '#fff', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 600, padding: '0.5rem 1rem' },
};

export default Devices;
