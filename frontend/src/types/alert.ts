/**
 * Alert Type Definitions
 * 
 * TypeScript interfaces for alert-related data structures
 * matching the backend API models.
 */

export type AlertSeverity = 'critical' | 'warning' | 'info';

export interface Alert {
  fingerprint: string;
  device_name: string;
  alert_name: string;
  severity: AlertSeverity;
  fired_at: string;
  resolved_at?: string;
  current_value: string;
  description: string;
  labels: Record<string, string>;
  acknowledged: boolean;
  acknowledged_at?: string;
}

export interface AlertListParams {
  severity?: AlertSeverity;
  device?: string;
}

export interface AlertAcknowledgment {
  id: string;
  user_id: string;
  alert_fingerprint: string;
  acknowledged_at: string;
}

export interface AlertsResponse {
  data: Alert[];
}

export interface AcknowledgeResponse {
  message: string;
  data: AlertAcknowledgment;
}

/**
 * Alert Rule Type Definitions
 */

export type AlertRuleOperator = 'gt' | 'lt' | 'eq' | 'ne';

export interface AlertRule {
  id: string;
  name: string;
  metric_name: string;
  operator: AlertRuleOperator;
  threshold: number;
  severity: AlertSeverity;
  device_id?: string;
  device_group_id?: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateAlertRuleRequest {
  name: string;
  metric_name: string;
  operator: AlertRuleOperator;
  threshold: number;
  severity: AlertSeverity;
  device_id?: string;
  device_group_id?: string;
  enabled?: boolean;
}

export interface UpdateAlertRuleRequest {
  name?: string;
  metric_name?: string;
  operator?: AlertRuleOperator;
  threshold?: number;
  severity?: AlertSeverity;
  device_id?: string;
  device_group_id?: string;
  enabled?: boolean;
}

export interface AlertRulesResponse {
  data: AlertRule[];
}

