import React, { useState } from 'react';
import { useAlerts, useAlertHistory, useAcknowledgeAlert } from '../hooks/useAlerts';
import { AlertSeverity, Alert } from '../types/alert';

/**
 * Alerts Page Component
 *
 * Displays active alerts and alert history with filtering capabilities.
 * - Active alerts: device name, alert name, severity, fired time, current value
 * - Alert history: device name, alert name, severity, fired time, resolved time, duration
 * - Filter by severity level (info, warning, critical, emergency)
 * - Filter by device name
 * - Alert count by severity level
 * - Acknowledge alert button
 * - Automatic refresh every 15 seconds (via useAlerts hook)
 *
 * Requirements: 21.1, 21.2, 21.3, 21.4, 21.5, 21.6, 21.7
 */

const SEVERITY_LEVELS: AlertSeverity[] = ['critical', 'warning', 'info'];

const SEVERITY_STYLES: Record<AlertSeverity, { bg: string; text: string; border: string; label: string }> = {
  critical: { bg: '#450a0a', text: '#f87171', border: '#dc2626', label: 'Critical' },
  warning:  { bg: '#422006', text: '#fbbf24', border: '#d97706', label: 'Warning' },
  info:     { bg: '#0c1a2e', text: '#60a5fa', border: '#2563eb', label: 'Info' },
};

// Fallback for unknown severities
const defaultSeverityStyle = { bg: '#f3f4f6', text: '#374151', border: '#d1d5db', label: 'Unknown' };

function getSeverityStyle(severity: string) {
  return SEVERITY_STYLES[severity as AlertSeverity] ?? defaultSeverityStyle;
}

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString();
}

function formatDuration(firedAt: string, resolvedAt: string): string {
  const ms = new Date(resolvedAt).getTime() - new Date(firedAt).getTime();
  if (ms < 0) return '—';
  const totalSeconds = Math.floor(ms / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m ${seconds}s`;
  return `${seconds}s`;
}

// Count alerts by severity
function countBySeverity(alerts: Alert[]): Record<AlertSeverity, number> {
  const counts: Record<AlertSeverity, number> = { critical: 0, warning: 0, info: 0 };
  for (const alert of alerts) {
    if (alert.severity in counts) {
      counts[alert.severity as AlertSeverity]++;
    }
  }
  return counts;
}

const Alerts: React.FC = () => {
  const [severityFilter, setSeverityFilter] = useState<AlertSeverity | ''>('');
  const [deviceFilter, setDeviceFilter] = useState('');
  const [activeTab, setActiveTab] = useState<'active' | 'history'>('active');
  const [acknowledgingId, setAcknowledgingId] = useState<string | null>(null);

  const filterParams = {
    ...(severityFilter ? { severity: severityFilter } : {}),
    ...(deviceFilter ? { device: deviceFilter } : {}),
  };

  const {
    data: activeAlerts = [],
    isLoading: activeLoading,
    isError: activeError,
    error: activeErrorMsg,
  } = useAlerts(filterParams);

  const {
    data: historyAlerts = [],
    isLoading: historyLoading,
    isError: historyError,
    error: historyErrorMsg,
  } = useAlertHistory(filterParams);

  const { mutate: acknowledge, isPending: isAcknowledging } = useAcknowledgeAlert();

  const handleAcknowledge = (fingerprint: string) => {
    setAcknowledgingId(fingerprint);
    acknowledge(fingerprint, {
      onSettled: () => setAcknowledgingId(null),
    });
  };

  const severityCounts = countBySeverity(activeAlerts);

  const handleSeverityFilterChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setSeverityFilter(e.target.value as AlertSeverity | '');
  };

  const handleDeviceFilterChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setDeviceFilter(e.target.value);
  };

  return (
    <div style={styles.page}>
      {/* Page header */}
      <div style={styles.pageHeader}>
        <h1 style={styles.pageTitle}>Alerts</h1>
      </div>

      {/* Severity count summary (Requirement 21.6) */}
      <div style={styles.severitySummary} role="region" aria-label="Alert counts by severity">
        {SEVERITY_LEVELS.map((sev) => {
          const s = SEVERITY_STYLES[sev];
          return (
            <div
              key={sev}
              style={{
                ...styles.severityCard,
                backgroundColor: s.bg,
                borderColor: s.border,
              }}
            >
              <span style={{ ...styles.severityCardCount, color: s.text }}>
                {severityCounts[sev]}
              </span>
              <span style={{ ...styles.severityCardLabel, color: s.text }}>
                {s.label}
              </span>
            </div>
          );
        })}
      </div>

      {/* Filters (Requirements 21.3, 21.4) */}
      <div style={styles.filterBar}>
        <select
          value={severityFilter}
          onChange={handleSeverityFilterChange}
          style={styles.select}
          aria-label="Filter by severity"
        >
          <option value="">All Severities</option>
          {SEVERITY_LEVELS.map((sev) => (
            <option key={sev} value={sev}>
              {SEVERITY_STYLES[sev].label}
            </option>
          ))}
        </select>

        <input
          type="text"
          placeholder="Filter by device name..."
          value={deviceFilter}
          onChange={handleDeviceFilterChange}
          style={styles.filterInput}
          aria-label="Filter by device name"
        />
      </div>

      {/* Tabs */}
      <div style={styles.tabs} role="tablist">
        <button
          role="tab"
          aria-selected={activeTab === 'active'}
          onClick={() => setActiveTab('active')}
          style={{
            ...styles.tab,
            ...(activeTab === 'active' ? styles.tabActive : {}),
          }}
        >
          Active Alerts
          {activeAlerts.length > 0 && (
            <span style={styles.tabBadge}>{activeAlerts.length}</span>
          )}
        </button>
        <button
          role="tab"
          aria-selected={activeTab === 'history'}
          onClick={() => setActiveTab('history')}
          style={{
            ...styles.tab,
            ...(activeTab === 'history' ? styles.tabActive : {}),
          }}
        >
          Alert History
        </button>
      </div>

      {/* Active Alerts Tab (Requirements 21.1, 21.5, 21.7) */}
      {activeTab === 'active' && (
        <>
          {activeLoading && (
            <div style={styles.centered}>
              <p>Loading alerts...</p>
            </div>
          )}
          {activeError && activeErrorMsg && (
            <div style={styles.errorBanner} role="alert">
              Failed to load alerts: {activeErrorMsg.message}
            </div>
          )}
          {!activeLoading && !activeError && (
            activeAlerts.length === 0 ? (
              <div style={styles.centered}>
                <p style={styles.emptyText}>No active alerts.</p>
              </div>
            ) : (
              <div style={styles.tableWrapper}>
                <table style={styles.table}>
                  <thead>
                    <tr>
                      <th style={styles.th}>Severity</th>
                      <th style={styles.th}>Device</th>
                      <th style={styles.th}>Alert</th>
                      <th style={styles.th}>Fired At</th>
                      <th style={styles.th}>Current Value</th>
                      <th style={styles.th}>Status</th>
                      <th style={styles.th}>Action</th>
                    </tr>
                  </thead>
                  <tbody>
                    {activeAlerts.map((alert) => {
                      const s = getSeverityStyle(alert.severity);
                      const isThisAcknowledging =
                        isAcknowledging && acknowledgingId === alert.fingerprint;
                      return (
                        <tr key={alert.fingerprint} style={styles.tr}>
                          <td style={styles.td}>
                            <span
                              style={{
                                ...styles.severityBadge,
                                backgroundColor: s.bg,
                                color: s.text,
                                borderColor: s.border,
                              }}
                            >
                              {s.label}
                            </span>
                          </td>
                          <td style={{ ...styles.td, fontWeight: 500 }}>
                            {alert.device_name}
                          </td>
                          <td style={styles.td}>{alert.alert_name}</td>
                          <td style={styles.td}>{formatDateTime(alert.fired_at)}</td>
                          <td style={styles.td}>{alert.current_value}</td>
                          <td style={styles.td}>
                            {alert.acknowledged ? (
                              <span style={styles.acknowledgedBadge}>Acknowledged</span>
                            ) : (
                              <span style={styles.firingBadge}>Firing</span>
                            )}
                          </td>
                          <td style={styles.td}>
                            {!alert.acknowledged && (
                              <button
                                onClick={() => handleAcknowledge(alert.fingerprint)}
                                disabled={isThisAcknowledging}
                                style={{
                                  ...styles.ackBtn,
                                  ...(isThisAcknowledging ? styles.ackBtnDisabled : {}),
                                }}
                                aria-label={`Acknowledge alert ${alert.alert_name} for ${alert.device_name}`}
                              >
                                {isThisAcknowledging ? 'Acknowledging...' : 'Acknowledge'}
                              </button>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )
          )}
        </>
      )}

      {/* Alert History Tab (Requirement 21.2) */}
      {activeTab === 'history' && (
        <>
          {historyLoading && (
            <div style={styles.centered}>
              <p>Loading alert history...</p>
            </div>
          )}
          {historyError && historyErrorMsg && (
            <div style={styles.errorBanner} role="alert">
              Failed to load alert history: {historyErrorMsg.message}
            </div>
          )}
          {!historyLoading && !historyError && (
            historyAlerts.length === 0 ? (
              <div style={styles.centered}>
                <p style={styles.emptyText}>No alert history found.</p>
              </div>
            ) : (
              <div style={styles.tableWrapper}>
                <table style={styles.table}>
                  <thead>
                    <tr>
                      <th style={styles.th}>Severity</th>
                      <th style={styles.th}>Device</th>
                      <th style={styles.th}>Alert</th>
                      <th style={styles.th}>Fired At</th>
                      <th style={styles.th}>Resolved At</th>
                      <th style={styles.th}>Duration</th>
                    </tr>
                  </thead>
                  <tbody>
                    {historyAlerts.map((alert) => {
                      const s = getSeverityStyle(alert.severity);
                      return (
                        <tr key={alert.fingerprint} style={styles.tr}>
                          <td style={styles.td}>
                            <span
                              style={{
                                ...styles.severityBadge,
                                backgroundColor: s.bg,
                                color: s.text,
                                borderColor: s.border,
                              }}
                            >
                              {s.label}
                            </span>
                          </td>
                          <td style={{ ...styles.td, fontWeight: 500 }}>
                            {alert.device_name}
                          </td>
                          <td style={styles.td}>{alert.alert_name}</td>
                          <td style={styles.td}>{formatDateTime(alert.fired_at)}</td>
                          <td style={styles.td}>
                            {alert.resolved_at ? formatDateTime(alert.resolved_at) : '—'}
                          </td>
                          <td style={styles.td}>
                            {alert.resolved_at
                              ? formatDuration(alert.fired_at, alert.resolved_at)
                              : '—'}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )
          )}
        </>
      )}
    </div>
  );
};

const styles: Record<string, React.CSSProperties> = {
  page: {
    padding: '1.5rem',
    maxWidth: '1200px',
    margin: '0 auto',
  },
  pageHeader: {
    display: 'flex',
    alignItems: 'center',
    gap: '1rem',
    marginBottom: '1rem',
  },
  pageTitle: {
    fontSize: '1.5rem',
    fontWeight: 700,
    color: '#f1f5f9',
    margin: 0,
  },
  severitySummary: {
    display: 'flex',
    gap: '1rem',
    marginBottom: '1.25rem',
    flexWrap: 'wrap',
  },
  severityCard: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    border: '1px solid',
    borderRadius: '8px',
    padding: '0.75rem 1.5rem',
    minWidth: '100px',
  },
  severityCardCount: {
    fontSize: '1.75rem',
    fontWeight: 700,
    lineHeight: 1,
  },
  severityCardLabel: {
    fontSize: '0.75rem',
    fontWeight: 600,
    marginTop: '0.25rem',
    textTransform: 'uppercase',
    letterSpacing: '0.05em',
  },
  filterBar: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: '0.75rem',
    marginBottom: '1rem',
  },
  select: {
    border: '1px solid #334155',
    borderRadius: '4px',
    fontSize: '0.875rem',
    padding: '0.5rem 0.75rem',
    backgroundColor: '#1e293b',
    color: '#e2e8f0',
    minWidth: '160px',
  },
  filterInput: {
    border: '1px solid #334155',
    borderRadius: '4px',
    fontSize: '0.875rem',
    padding: '0.5rem 0.75rem',
    minWidth: '220px',
    flex: '1 1 220px',
    background: '#1e293b',
    color: '#e2e8f0',
  },
  tabs: {
    display: 'flex',
    gap: '0',
    borderBottom: '2px solid #1e293b',
    marginBottom: '1rem',
  },
  tab: {
    background: 'none',
    border: 'none',
    borderBottom: '2px solid transparent',
    cursor: 'pointer',
    fontSize: '0.875rem',
    fontWeight: 500,
    color: '#64748b',
    padding: '0.625rem 1rem',
    marginBottom: '-2px',
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
  },
  tabActive: {
    color: '#60a5fa',
    borderBottomColor: '#3b82f6',
  },
  tabBadge: {
    backgroundColor: '#450a0a',
    color: '#f87171',
    borderRadius: '9999px',
    fontSize: '0.7rem',
    fontWeight: 700,
    padding: '0.1rem 0.45rem',
  },
  centered: {
    textAlign: 'center',
    padding: '3rem 0',
  },
  emptyText: {
    color: '#64748b',
    fontSize: '0.875rem',
  },
  errorBanner: {
    backgroundColor: '#450a0a',
    border: '1px solid #dc2626',
    borderRadius: '4px',
    color: '#f87171',
    fontSize: '0.875rem',
    padding: '0.75rem 1rem',
    marginBottom: '1rem',
  },
  tableWrapper: {
    overflowX: 'auto',
    borderRadius: '8px',
    border: '1px solid #1e293b',
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse',
    fontSize: '0.875rem',
  },
  th: {
    backgroundColor: '#1e293b',
    borderBottom: '1px solid #334155',
    color: '#64748b',
    fontWeight: 700,
    fontSize: '0.7rem',
    textTransform: 'uppercase',
    letterSpacing: '0.08em',
    padding: '0.75rem 1rem',
    textAlign: 'left',
    whiteSpace: 'nowrap',
  },
  tr: {
    borderBottom: '1px solid #1e293b',
  },
  td: {
    padding: '0.75rem 1rem',
    color: '#94a3b8',
    verticalAlign: 'middle',
  },
  severityBadge: {
    border: '1px solid',
    borderRadius: '9999px',
    display: 'inline-block',
    fontSize: '0.75rem',
    fontWeight: 600,
    padding: '0.2rem 0.6rem',
    whiteSpace: 'nowrap',
  },
  firingBadge: {
    backgroundColor: '#450a0a',
    color: '#f87171',
    borderRadius: '9999px',
    display: 'inline-block',
    fontSize: '0.75rem',
    fontWeight: 600,
    padding: '0.2rem 0.6rem',
  },
  acknowledgedBadge: {
    backgroundColor: '#052e16',
    color: '#4ade80',
    borderRadius: '9999px',
    display: 'inline-block',
    fontSize: '0.75rem',
    fontWeight: 600,
    padding: '0.2rem 0.6rem',
  },
  ackBtn: {
    backgroundColor: '#1e3a5f',
    border: '1px solid #2563eb',
    borderRadius: '4px',
    color: '#60a5fa',
    cursor: 'pointer',
    fontSize: '0.8rem',
    fontWeight: 500,
    padding: '0.35rem 0.75rem',
    whiteSpace: 'nowrap',
  },
  ackBtnDisabled: {
    opacity: 0.5,
    cursor: 'not-allowed',
  },
};

export default Alerts;
