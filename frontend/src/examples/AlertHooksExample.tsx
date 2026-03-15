/**
 * Example: Using Alert React Query Hooks
 * 
 * This file demonstrates how to use the alert management hooks
 * in React components.
 */

import React, { useState } from 'react';
import { useAlerts, useAlertHistory, useAcknowledgeAlert } from '../hooks/useAlerts';
import { AlertSeverity } from '../types/alert';

/**
 * Example 1: Display Active Alerts
 * 
 * Shows how to fetch and display active alerts with automatic
 * 15-second refresh.
 */
export const ActiveAlertsExample: React.FC = () => {
  const [severityFilter, setSeverityFilter] = useState<AlertSeverity | undefined>();
  const [deviceFilter, setDeviceFilter] = useState<string>('');

  // Fetch active alerts with optional filters
  const { data: alerts, isLoading, error } = useAlerts({
    severity: severityFilter,
    device: deviceFilter || undefined,
  });

  const acknowledgeMutation = useAcknowledgeAlert();

  const handleAcknowledge = (fingerprint: string) => {
    acknowledgeMutation.mutate(fingerprint, {
      onSuccess: () => {
        console.log('Alert acknowledged successfully');
      },
      onError: (error) => {
        console.error('Failed to acknowledge alert:', error);
      },
    });
  };

  if (isLoading) return <div>Loading alerts...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div>
      <h2>Active Alerts</h2>
      
      {/* Filters */}
      <div>
        <label>
          Severity:
          <select
            value={severityFilter || ''}
            onChange={(e) => setSeverityFilter(e.target.value as AlertSeverity || undefined)}
          >
            <option value="">All</option>
            <option value="critical">Critical</option>
            <option value="warning">Warning</option>
            <option value="info">Info</option>
          </select>
        </label>
        
        <label>
          Device:
          <input
            type="text"
            value={deviceFilter}
            onChange={(e) => setDeviceFilter(e.target.value)}
            placeholder="Filter by device name"
          />
        </label>
      </div>

      {/* Alert List */}
      <table>
        <thead>
          <tr>
            <th>Device</th>
            <th>Alert</th>
            <th>Severity</th>
            <th>Fired At</th>
            <th>Value</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {alerts?.map((alert) => (
            <tr key={alert.fingerprint}>
              <td>{alert.device_name}</td>
              <td>{alert.alert_name}</td>
              <td>{alert.severity}</td>
              <td>{new Date(alert.fired_at).toLocaleString()}</td>
              <td>{alert.current_value}</td>
              <td>{alert.acknowledged ? 'Acknowledged' : 'Active'}</td>
              <td>
                {!alert.acknowledged && (
                  <button
                    onClick={() => handleAcknowledge(alert.fingerprint)}
                    disabled={acknowledgeMutation.isPending}
                  >
                    Acknowledge
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {alerts?.length === 0 && <p>No active alerts</p>}
    </div>
  );
};

/**
 * Example 2: Display Alert History
 * 
 * Shows how to fetch and display alert history including
 * resolved alerts.
 */
export const AlertHistoryExample: React.FC = () => {
  const { data: history, isLoading, error } = useAlertHistory();

  if (isLoading) return <div>Loading alert history...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div>
      <h2>Alert History</h2>
      
      <table>
        <thead>
          <tr>
            <th>Device</th>
            <th>Alert</th>
            <th>Severity</th>
            <th>Fired At</th>
            <th>Resolved At</th>
            <th>Duration</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {history?.map((alert) => {
            const duration = alert.resolved_at
              ? Math.round(
                  (new Date(alert.resolved_at).getTime() -
                    new Date(alert.fired_at).getTime()) /
                    1000 /
                    60
                )
              : null;

            return (
              <tr key={alert.fingerprint}>
                <td>{alert.device_name}</td>
                <td>{alert.alert_name}</td>
                <td>{alert.severity}</td>
                <td>{new Date(alert.fired_at).toLocaleString()}</td>
                <td>
                  {alert.resolved_at
                    ? new Date(alert.resolved_at).toLocaleString()
                    : 'Active'}
                </td>
                <td>{duration ? `${duration} min` : '-'}</td>
                <td>
                  {alert.acknowledged ? 'Acknowledged' : 'Not Acknowledged'}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>

      {history?.length === 0 && <p>No alert history</p>}
    </div>
  );
};

/**
 * Example 3: Alert Count by Severity
 * 
 * Shows how to calculate alert statistics from the data.
 */
export const AlertStatsExample: React.FC = () => {
  const { data: alerts } = useAlerts();

  const stats = React.useMemo(() => {
    if (!alerts) return { critical: 0, warning: 0, info: 0 };

    return alerts.reduce(
      (acc, alert) => {
        acc[alert.severity] = (acc[alert.severity] || 0) + 1;
        return acc;
      },
      { critical: 0, warning: 0, info: 0 } as Record<AlertSeverity, number>
    );
  }, [alerts]);

  return (
    <div>
      <h2>Alert Statistics</h2>
      <div>
        <div>Critical: {stats.critical}</div>
        <div>Warning: {stats.warning}</div>
        <div>Info: {stats.info}</div>
        <div>Total: {(alerts?.length || 0)}</div>
      </div>
    </div>
  );
};

