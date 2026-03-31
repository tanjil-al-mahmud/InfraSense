import React, { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';

// ── Types ─────────────────────────────────────────────────────────────────────

interface HardwareEvent {
  id: string;
  device_id: string;
  hostname: string;
  event_time: string;
  source_protocol: string;
  component: string;
  event_type: string;
  severity: string;
  message: string;
}

interface EventsResponse {
  events: HardwareEvent[];
  total: number;
  page: number;
  limit: number;
}

interface EventSummary {
  total_critical: number;
  total_warning: number;
  total_info: number;
  last_24h: number;
  last_7d: number;
  by_device: Array<{ device_id: string; hostname: string; count: number; max_severity: string }>;
}

// ── Helpers ───────────────────────────────────────────────────────────────────

const API = '/api/v1';

async function apiFetch<T>(path: string, token: string): Promise<T> {
  const res = await fetch(API + path, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json() as Promise<T>;
}

const severityColor: Record<string, { bg: string; text: string; dot: string }> = {
  critical: { bg: 'rgba(239,68,68,0.12)', text: '#f87171', dot: '#ef4444' },
  warning:  { bg: 'rgba(234,179,8,0.12)',  text: '#facc15', dot: '#eab308' },
  info:     { bg: 'rgba(59,130,246,0.12)', text: '#60a5fa', dot: '#3b82f6' },
};

function sev(s: string) {
  return severityColor[s?.toLowerCase()] ?? severityColor.info;
}

function fmt(iso: string) {
  if (!iso) return '—';
  return new Date(iso).toLocaleString(undefined, {
    month: 'short', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit',
  });
}

function Badge({ severity }: { severity: string }) {
  const c = sev(severity);
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: 5,
      padding: '2px 8px', borderRadius: 99, fontSize: '0.72rem', fontWeight: 600,
      background: c.bg, color: c.text, letterSpacing: '0.03em', textTransform: 'uppercase',
    }}>
      <span style={{ width: 6, height: 6, borderRadius: '50%', background: c.dot, flexShrink: 0 }} />
      {severity}
    </span>
  );
}

function StatCard({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div style={{
      background: '#0f172a', border: '1px solid #1e293b', borderRadius: 12,
      padding: '1.25rem 1.5rem', flex: 1, minWidth: 140,
    }}>
      <div style={{ fontSize: '1.875rem', fontWeight: 700, color, lineHeight: 1 }}>{value.toLocaleString()}</div>
      <div style={{ fontSize: '0.8rem', color: '#64748b', marginTop: 6 }}>{label}</div>
    </div>
  );
}

// ── Component ─────────────────────────────────────────────────────────────────

const EventLog: React.FC = () => {
  const token = (() => {
    try { return JSON.parse(localStorage.getItem('auth') || '{}').token ?? ''; } catch { return ''; }
  })();

  // filter state
  const [severity, setSeverity] = useState('all');
  const [search,   setSearch]   = useState('');
  const [page,     setPage]     = useState(1);
  const LIMIT = 50;

  // data state
  const [events,   setEvents]   = useState<HardwareEvent[]>([]);
  const [total,    setTotal]    = useState(0);
  const [summary,  setSummary]  = useState<EventSummary | null>(null);
  const [loading,  setLoading]  = useState(true);
  const [error,    setError]    = useState('');
  const [lastRefreshed, setLastRefreshed] = useState(new Date());

  // fetch events
  const fetchEvents = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const params = new URLSearchParams({
        page: String(page),
        limit: String(LIMIT),
        ...(severity !== 'all' ? { severity } : {}),
        ...(search ? { search } : {}),
      });
      const data = await apiFetch<EventsResponse>(`/events?${params}`, token);
      setEvents(data.events ?? []);
      setTotal(data.total ?? 0);
      setLastRefreshed(new Date());
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load events');
    } finally {
      setLoading(false);
    }
  }, [page, severity, search, token]);

  // fetch summary
  const fetchSummary = useCallback(async () => {
    try {
      const data = await apiFetch<EventSummary>('/events/summary', token);
      setSummary(data);
    } catch { /* non-fatal */ }
  }, [token]);

  useEffect(() => { fetchEvents(); fetchSummary(); }, [fetchEvents, fetchSummary]);

  // Auto-refresh every 30s
  useEffect(() => {
    const t = setInterval(() => { fetchEvents(); fetchSummary(); }, 30_000);
    return () => clearInterval(t);
  }, [fetchEvents, fetchSummary]);

  const totalPages = Math.max(1, Math.ceil(total / LIMIT));

  // CSV export
  const exportCSV = () => {
    const headers = ['Time', 'Device', 'Severity', 'Protocol', 'Component', 'Type', 'Message'];
    const rows = events.map(e => [
      fmt(e.event_time), e.hostname, e.severity,
      e.source_protocol, e.component, e.event_type,
      `"${e.message.replace(/"/g, '""')}"`,
    ]);
    const csv = [headers, ...rows].map(r => r.join(',')).join('\n');
    const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv' }));
    const a = document.createElement('a'); a.href = url; a.download = 'events.csv'; a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div style={{ padding: '2rem', maxWidth: 1400, margin: '0 auto' }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1.5rem', flexWrap: 'wrap', gap: '1rem' }}>
        <div>
          <h1 style={{ fontSize: '1.5rem', fontWeight: 700, color: '#f1f5f9', margin: 0 }}>Hardware Event Log</h1>
          <p style={{ color: '#64748b', fontSize: '0.8rem', margin: '4px 0 0' }}>
            Auto-refresh every 30s · Last: {lastRefreshed.toLocaleTimeString()}
          </p>
        </div>
        <div style={{ display: 'flex', gap: '0.5rem' }}>
          <button
            id="btn-refresh-events"
            onClick={() => { fetchEvents(); fetchSummary(); }}
            style={{ padding: '0.5rem 1rem', borderRadius: 8, border: '1px solid #334155', background: '#1e293b', color: '#94a3b8', cursor: 'pointer', fontSize: '0.85rem' }}
          >↺ Refresh</button>
          <button
            id="btn-export-events-csv"
            onClick={exportCSV}
            style={{ padding: '0.5rem 1rem', borderRadius: 8, border: '1px solid #334155', background: '#1e293b', color: '#94a3b8', cursor: 'pointer', fontSize: '0.85rem' }}
          >⬇ Export CSV</button>
        </div>
      </div>

      {/* Summary cards */}
      {summary && (
        <div style={{ display: 'flex', gap: '1rem', marginBottom: '1.5rem', flexWrap: 'wrap' }}>
          <StatCard label="Critical Events" value={summary.total_critical} color="#f87171" />
          <StatCard label="Warning Events"  value={summary.total_warning}  color="#facc15" />
          <StatCard label="Info Events"     value={summary.total_info}     color="#60a5fa" />
          <StatCard label="Last 24 Hours"   value={summary.last_24h}       color="#a78bfa" />
          <StatCard label="Last 7 Days"     value={summary.last_7d}        color="#34d399" />
        </div>
      )}

      {/* Filters */}
      <div style={{ display: 'flex', gap: '0.75rem', marginBottom: '1.25rem', flexWrap: 'wrap', alignItems: 'center' }}>
        {/* Severity pills */}
        {(['all', 'critical', 'warning', 'info'] as const).map(s => (
          <button
            key={s}
            id={`filter-severity-${s}`}
            onClick={() => { setSeverity(s); setPage(1); }}
            style={{
              padding: '0.35rem 0.875rem', borderRadius: 99, border: 'none', cursor: 'pointer',
              fontSize: '0.8rem', fontWeight: 600, textTransform: 'capitalize',
              background: severity === s
                ? (s === 'critical' ? '#7f1d1d' : s === 'warning' ? '#713f12' : s === 'info' ? '#1e3a5f' : '#334155')
                : '#1e293b',
              color: severity === s
                ? (s === 'critical' ? '#fca5a5' : s === 'warning' ? '#fde68a' : s === 'info' ? '#93c5fd' : '#e2e8f0')
                : '#64748b',
            }}
          >{s}</button>
        ))}

        {/* Search */}
        <input
          id="input-event-search"
          type="text"
          placeholder="Search messages…"
          value={search}
          onChange={e => { setSearch(e.target.value); setPage(1); }}
          style={{
            padding: '0.4rem 0.875rem', borderRadius: 8, border: '1px solid #334155',
            background: '#0f172a', color: '#e2e8f0', fontSize: '0.85rem', outline: 'none',
            minWidth: 220, flex: 1, maxWidth: 360,
          }}
        />

        <span style={{ color: '#64748b', fontSize: '0.8rem', marginLeft: 'auto' }}>
          {total.toLocaleString()} event{total !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Error */}
      {error && (
        <div style={{ background: 'rgba(239,68,68,0.1)', border: '1px solid #7f1d1d', borderRadius: 8, padding: '0.75rem 1rem', color: '#fca5a5', marginBottom: '1rem', fontSize: '0.875rem' }}>
          {error}
        </div>
      )}

      {/* Table */}
      <div style={{ background: '#0f172a', border: '1px solid #1e293b', borderRadius: 12, overflow: 'hidden' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', tableLayout: 'fixed' }}>
          <colgroup>
            <col style={{ width: '14%' }} />
            <col style={{ width: '13%' }} />
            <col style={{ width: '9%' }} />
            <col style={{ width: '9%' }} />
            <col style={{ width: '9%' }} />
            <col style={{ width: '9%' }} />
            <col />
          </colgroup>
          <thead>
            <tr style={{ borderBottom: '1px solid #1e293b' }}>
              {['Time', 'Device', 'Severity', 'Protocol', 'Component', 'Type', 'Message'].map(h => (
                <th key={h} style={{ padding: '0.75rem 1rem', textAlign: 'left', fontSize: '0.72rem', fontWeight: 600, color: '#475569', textTransform: 'uppercase', letterSpacing: '0.05em' }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr>
                <td colSpan={7} style={{ textAlign: 'center', padding: '3rem', color: '#475569' }}>
                  <div style={{ display: 'inline-block', width: 28, height: 28, border: '3px solid #1e293b', borderTopColor: '#3b82f6', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
                </td>
              </tr>
            )}
            {!loading && events.length === 0 && (
              <tr>
                <td colSpan={7} style={{ textAlign: 'center', padding: '3rem', color: '#475569', fontSize: '0.875rem' }}>
                  No events found
                </td>
              </tr>
            )}
            {!loading && events.map((ev, i) => {
              const c = sev(ev.severity);
              return (
                <tr
                  key={ev.id}
                  style={{
                    borderBottom: i < events.length - 1 ? '1px solid #0f172a' : 'none',
                    background: i % 2 === 0 ? 'transparent' : 'rgba(30,41,59,0.3)',
                    borderLeft: ev.severity === 'critical' ? `3px solid ${c.dot}` : '3px solid transparent',
                  }}
                >
                  <td style={{ padding: '0.6rem 1rem', fontSize: '0.78rem', color: '#94a3b8', whiteSpace: 'nowrap' }}>{fmt(ev.event_time)}</td>
                  <td style={{ padding: '0.6rem 1rem' }}>
                    <Link to={`/devices/${ev.device_id}`} style={{ color: '#38bdf8', textDecoration: 'none', fontSize: '0.82rem', fontWeight: 500 }}>
                      {ev.hostname || ev.device_id.slice(0, 8)}
                    </Link>
                  </td>
                  <td style={{ padding: '0.6rem 1rem' }}><Badge severity={ev.severity} /></td>
                  <td style={{ padding: '0.6rem 1rem', fontSize: '0.78rem', color: '#64748b', textTransform: 'uppercase' }}>{ev.source_protocol}</td>
                  <td style={{ padding: '0.6rem 1rem', fontSize: '0.78rem', color: '#64748b' }}>{ev.component}</td>
                  <td style={{ padding: '0.6rem 1rem', fontSize: '0.78rem', color: '#64748b' }}>{ev.event_type}</td>
                  <td style={{ padding: '0.6rem 1rem', fontSize: '0.82rem', color: '#cbd5e1', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={ev.message}>{ev.message}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div style={{ display: 'flex', justifyContent: 'center', gap: '0.5rem', marginTop: '1.25rem', alignItems: 'center' }}>
          <button
            id="btn-events-prev"
            onClick={() => setPage(p => Math.max(1, p - 1))}
            disabled={page === 1}
            style={{ padding: '0.4rem 0.875rem', borderRadius: 6, border: '1px solid #334155', background: '#1e293b', color: page === 1 ? '#475569' : '#e2e8f0', cursor: page === 1 ? 'default' : 'pointer', fontSize: '0.85rem' }}
          >← Prev</button>
          <span style={{ color: '#64748b', fontSize: '0.85rem' }}>Page {page} / {totalPages}</span>
          <button
            id="btn-events-next"
            onClick={() => setPage(p => Math.min(totalPages, p + 1))}
            disabled={page === totalPages}
            style={{ padding: '0.4rem 0.875rem', borderRadius: 6, border: '1px solid #334155', background: '#1e293b', color: page === totalPages ? '#475569' : '#e2e8f0', cursor: page === totalPages ? 'default' : 'pointer', fontSize: '0.85rem' }}
          >Next →</button>
        </div>
      )}

      {/* Spin animation */}
      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </div>
  );
};

export default EventLog;
