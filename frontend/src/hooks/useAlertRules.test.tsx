/**
 * Unit Tests for Alert Rule Hooks
 * 
 * Tests for React Query hooks managing alert rule operations.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';
import {
  useAlertRules,
  useCreateAlertRule,
  useUpdateAlertRule,
  useDeleteAlertRule,
} from './useAlertRules';
import * as alertRuleApi from '../services/alertRuleApi';
import { AlertRule, CreateAlertRuleRequest } from '../types/alert';

// Mock the alert rule API
vi.mock('../services/alertRuleApi');

describe('useAlertRules hooks', () => {
  let queryClient: QueryClient;

  // Create a wrapper with QueryClient for testing
  const createWrapper = () => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    return ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('useAlertRules', () => {
    it('should fetch alert rules successfully', async () => {
      const mockAlertRules: AlertRule[] = [
        {
          id: '123e4567-e89b-12d3-a456-426614174000',
          name: 'High CPU Temperature',
          metric_name: 'cpu_temperature',
          operator: 'gt',
          threshold: 80,
          severity: 'critical',
          enabled: true,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ];

      vi.mocked(alertRuleApi.fetchAlertRules).mockResolvedValue(mockAlertRules);

      const { result } = renderHook(() => useAlertRules(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(result.current.data).toEqual(mockAlertRules);
      expect(alertRuleApi.fetchAlertRules).toHaveBeenCalledTimes(1);
    });

    it('should handle fetch error', async () => {
      const mockError = new Error('Failed to fetch alert rules');
      vi.mocked(alertRuleApi.fetchAlertRules).mockRejectedValue(mockError);

      const { result } = renderHook(() => useAlertRules(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isError).toBe(true));

      expect(result.current.error).toEqual(mockError);
    });
  });

  describe('useCreateAlertRule', () => {
    it('should create alert rule successfully', async () => {
      const newAlertRule: CreateAlertRuleRequest = {
        name: 'High CPU Temperature',
        metric_name: 'cpu_temperature',
        operator: 'gt',
        threshold: 80,
        severity: 'critical',
        enabled: true,
      };

      const createdAlertRule: AlertRule = {
        id: '123e4567-e89b-12d3-a456-426614174000',
        ...newAlertRule,
        enabled: true,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      vi.mocked(alertRuleApi.createAlertRule).mockResolvedValue(createdAlertRule);

      const { result } = renderHook(() => useCreateAlertRule(), {
        wrapper: createWrapper(),
      });

      result.current.mutate(newAlertRule);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(result.current.data).toEqual(createdAlertRule);
      expect(alertRuleApi.createAlertRule).toHaveBeenCalledWith(newAlertRule);
    });
  });

  describe('useUpdateAlertRule', () => {
    it('should update alert rule successfully', async () => {
      const updateData = {
        id: '123e4567-e89b-12d3-a456-426614174000',
        data: { threshold: 85 },
      };

      const updatedAlertRule: AlertRule = {
        id: updateData.id,
        name: 'High CPU Temperature',
        metric_name: 'cpu_temperature',
        operator: 'gt',
        threshold: 85,
        severity: 'critical',
        enabled: true,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      vi.mocked(alertRuleApi.updateAlertRule).mockResolvedValue(updatedAlertRule);

      const { result } = renderHook(() => useUpdateAlertRule(), {
        wrapper: createWrapper(),
      });

      result.current.mutate(updateData);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(result.current.data).toEqual(updatedAlertRule);
      expect(alertRuleApi.updateAlertRule).toHaveBeenCalledWith(
        updateData.id,
        updateData.data
      );
    });
  });

  describe('useDeleteAlertRule', () => {
    it('should delete alert rule successfully', async () => {
      const alertRuleId = '123e4567-e89b-12d3-a456-426614174000';

      vi.mocked(alertRuleApi.deleteAlertRule).mockResolvedValue();

      const { result } = renderHook(() => useDeleteAlertRule(), {
        wrapper: createWrapper(),
      });

      result.current.mutate(alertRuleId);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(alertRuleApi.deleteAlertRule).toHaveBeenCalledWith(alertRuleId);
    });
  });
});
