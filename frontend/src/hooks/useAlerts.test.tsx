/**
 * Unit Tests for Alert React Query Hooks
 * 
 * Tests for alert management hooks using React Query.
 */

import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useAlerts, useAlertHistory, useAcknowledgeAlert } from './useAlerts';
import * as alertApi from '../services/alertApi';
import { Alert, AcknowledgeResponse } from '../types/alert';

// Mock the alert API module
vi.mock('../services/alertApi');

// Helper to create a wrapper with QueryClient
const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
      mutations: {
        retry: false,
      },
    },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('useAlerts', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should fetch active alerts successfully', async () => {
    const mockAlerts: Alert[] = [
      {
        fingerprint: 'abc123',
        device_name: 'server-01',
        alert_name: 'HighCPUTemperature',
        severity: 'critical',
        fired_at: '2024-01-01T00:00:00Z',
        current_value: '85°C',
        description: 'CPU temperature is above threshold',
        labels: { device: 'server-01', severity: 'critical' },
        acknowledged: false,
      },
    ];

    vi.mocked(alertApi.fetchAlerts).mockResolvedValue(mockAlerts);

    const { result } = renderHook(() => useAlerts(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(mockAlerts);
    expect(alertApi.fetchAlerts).toHaveBeenCalledTimes(1);
  });

  it('should fetch alerts with filtering parameters', async () => {
    const mockAlerts: Alert[] = [];
    vi.mocked(alertApi.fetchAlerts).mockResolvedValue(mockAlerts);

    const params = { severity: 'critical' as const, device: 'server-01' };
    const { result } = renderHook(() => useAlerts(params), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(alertApi.fetchAlerts).toHaveBeenCalledWith(params);
  });

  it('should handle fetch alerts error', async () => {
    const mockError = new Error('Network error');
    vi.mocked(alertApi.fetchAlerts).mockRejectedValue(mockError);

    const { result } = renderHook(() => useAlerts(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toEqual(mockError);
  });
});

describe('useAlertHistory', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should fetch alert history successfully', async () => {
    const mockHistory: Alert[] = [
      {
        fingerprint: 'abc123',
        device_name: 'server-01',
        alert_name: 'HighCPUTemperature',
        severity: 'critical',
        fired_at: '2024-01-01T00:00:00Z',
        resolved_at: '2024-01-01T01:00:00Z',
        current_value: '85°C',
        description: 'CPU temperature is above threshold',
        labels: { device: 'server-01', severity: 'critical' },
        acknowledged: true,
        acknowledged_at: '2024-01-01T00:30:00Z',
      },
    ];

    vi.mocked(alertApi.fetchAlertHistory).mockResolvedValue(mockHistory);

    const { result } = renderHook(() => useAlertHistory(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(mockHistory);
    expect(alertApi.fetchAlertHistory).toHaveBeenCalledTimes(1);
  });

  it('should fetch history with filtering parameters', async () => {
    const mockHistory: Alert[] = [];
    vi.mocked(alertApi.fetchAlertHistory).mockResolvedValue(mockHistory);

    const params = { severity: 'warning' as const };
    const { result } = renderHook(() => useAlertHistory(params), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(alertApi.fetchAlertHistory).toHaveBeenCalledWith(params);
  });

  it('should handle fetch history error', async () => {
    const mockError = new Error('Failed to fetch history');
    vi.mocked(alertApi.fetchAlertHistory).mockRejectedValue(mockError);

    const { result } = renderHook(() => useAlertHistory(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toEqual(mockError);
  });
});

describe('useAcknowledgeAlert', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should acknowledge alert successfully', async () => {
    const mockResponse: AcknowledgeResponse = {
      message: 'Alert acknowledged successfully',
      data: {
        id: 'ack-1',
        user_id: 'user-1',
        alert_fingerprint: 'abc123',
        acknowledged_at: '2024-01-01T00:30:00Z',
      },
    };

    vi.mocked(alertApi.acknowledgeAlert).mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useAcknowledgeAlert(), {
      wrapper: createWrapper(),
    });

    result.current.mutate('abc123');

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(mockResponse);
    expect(alertApi.acknowledgeAlert).toHaveBeenCalledWith('abc123');
  });

  it('should handle acknowledge error', async () => {
    const mockError = new Error('Failed to acknowledge alert');
    vi.mocked(alertApi.acknowledgeAlert).mockRejectedValue(mockError);

    const { result } = renderHook(() => useAcknowledgeAlert(), {
      wrapper: createWrapper(),
    });

    result.current.mutate('abc123');

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toEqual(mockError);
  });
});

