/**
 * Alert API Service
 * 
 * API functions for alert management operations.
 * All functions use the configured apiClient with JWT authentication.
 */

import apiClient from './api';
import {
  Alert,
  AlertListParams,
  AlertsResponse,
  AcknowledgeResponse,
} from '../types/alert';

/**
 * Fetch list of active alerts with optional filtering
 * GET /api/v1/alerts
 */
export const fetchAlerts = async (
  params?: AlertListParams
): Promise<Alert[]> => {
  const response = await apiClient.get<AlertsResponse>('/alerts', {
    params,
  });
  return response.data.data || [];
};

/**
 * Fetch alert history (including resolved alerts)
 * GET /api/v1/alerts/history
 */
export const fetchAlertHistory = async (
  params?: AlertListParams
): Promise<Alert[]> => {
  const response = await apiClient.get<AlertsResponse>('/alerts/history', {
    params,
  });
  return response.data.data || [];
};

/**
 * Acknowledge an alert
 * POST /api/v1/alerts/{id}/acknowledge
 */
export const acknowledgeAlert = async (
  fingerprint: string
): Promise<AcknowledgeResponse> => {
  const response = await apiClient.post<AcknowledgeResponse>(
    `/alerts/${fingerprint}/acknowledge`
  );
  return response.data;
};

