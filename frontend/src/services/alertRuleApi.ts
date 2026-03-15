/**
 * Alert Rule API Service
 * 
 * API functions for alert rule CRUD operations.
 * All functions use the configured apiClient with JWT authentication.
 * 
 * Requirements: 12.1, 12.2, 12.3, 12.5
 */

import apiClient from './api';
import {
  AlertRule,
  CreateAlertRuleRequest,
  UpdateAlertRuleRequest,
  AlertRulesResponse,
} from '../types/alert';

/**
 * Fetch all alert rules
 * GET /api/v1/alert-rules
 * 
 * @returns Promise with array of alert rules
 * @throws Error if request fails
 * 
 * Requirements: 12.2
 */
export const fetchAlertRules = async (): Promise<AlertRule[]> => {
  const response = await apiClient.get<AlertRulesResponse>('/alert-rules');
  return response.data.data;
};

/**
 * Fetch a single alert rule by ID
 * GET /api/v1/alert-rules/{id}
 * 
 * @param id - Alert rule ID
 * @returns Promise with alert rule details
 * @throws Error if request fails
 * 
 * Requirements: 12.2
 */
export const fetchAlertRule = async (id: string): Promise<AlertRule> => {
  const response = await apiClient.get<{ data: AlertRule }>(`/alert-rules/${id}`);
  return response.data.data;
};

/**
 * Create a new alert rule
 * POST /api/v1/alert-rules
 * 
 * @param data - Alert rule creation request
 * @returns Promise with created alert rule
 * @throws Error if request fails
 * 
 * Requirements: 12.1, 12.3
 */
export const createAlertRule = async (
  data: CreateAlertRuleRequest
): Promise<AlertRule> => {
  const response = await apiClient.post<{ data: AlertRule }>('/alert-rules', data);
  return response.data.data;
};

/**
 * Update an existing alert rule
 * PUT /api/v1/alert-rules/{id}
 * 
 * @param id - Alert rule ID
 * @param data - Alert rule update request
 * @returns Promise with updated alert rule
 * @throws Error if request fails
 * 
 * Requirements: 12.5
 */
export const updateAlertRule = async (
  id: string,
  data: UpdateAlertRuleRequest
): Promise<AlertRule> => {
  const response = await apiClient.put<{ data: AlertRule }>(`/alert-rules/${id}`, data);
  return response.data.data;
};

/**
 * Delete an alert rule
 * DELETE /api/v1/alert-rules/{id}
 * 
 * @param id - Alert rule ID
 * @returns Promise that resolves when deletion is complete
 * @throws Error if request fails
 * 
 * Requirements: 12.5
 */
export const deleteAlertRule = async (id: string): Promise<void> => {
  await apiClient.delete(`/alert-rules/${id}`);
};
