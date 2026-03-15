/**
 * Unit Tests for Device React Query Hooks
 * 
 * Tests for device management hooks using React Query.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useDevices, useDevice, useCreateDevice, useUpdateDevice, useDeleteDevice } from './useDevices';
import * as deviceApi from '../services/deviceApi';
import { Device, CreateDeviceRequest, UpdateDeviceRequest } from '../types/device';

// Mock the device API module
vi.mock('../services/deviceApi');

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

describe('useDevices', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should fetch devices list successfully', async () => {
    const mockDevices = {
      data: [
        {
          id: '1',
          hostname: 'server-01',
          ip_address: '192.168.1.10',
          device_type: 'ipmi' as const,
          status: 'healthy' as const,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ],
      meta: {
        page: 1,
        page_size: 20,
        total: 1,
      },
    };

    vi.mocked(deviceApi.fetchDevices).mockResolvedValue(mockDevices);

    const { result } = renderHook(() => useDevices(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(mockDevices);
    expect(deviceApi.fetchDevices).toHaveBeenCalledTimes(1);
  });

  it('should handle fetch devices error', async () => {
    const mockError = new Error('Network error');
    vi.mocked(deviceApi.fetchDevices).mockRejectedValue(mockError);

    const { result } = renderHook(() => useDevices(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error).toEqual(mockError);
  });
});

describe('useDevice', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should fetch single device successfully', async () => {
    const mockDevice: Device = {
      id: '1',
      hostname: 'server-01',
      ip_address: '192.168.1.10',
      device_type: 'ipmi',
      status: 'healthy',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    };

    vi.mocked(deviceApi.fetchDevice).mockResolvedValue(mockDevice);

    const { result } = renderHook(() => useDevice('1'), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(mockDevice);
    expect(deviceApi.fetchDevice).toHaveBeenCalledWith('1');
  });

  it('should not fetch when enabled is false', () => {
    const { result } = renderHook(() => useDevice('1', false), {
      wrapper: createWrapper(),
    });

    expect(result.current.isFetching).toBe(false);
    expect(deviceApi.fetchDevice).not.toHaveBeenCalled();
  });
});

describe('useCreateDevice', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should create device successfully', async () => {
    const newDevice: CreateDeviceRequest = {
      hostname: 'server-02',
      ip_address: '192.168.1.11',
      device_type: 'redfish',
    };

    const createdDevice: Device = {
      id: '2',
      ...newDevice,
      status: 'unknown',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    };

    vi.mocked(deviceApi.createDevice).mockResolvedValue(createdDevice);

    const { result } = renderHook(() => useCreateDevice(), {
      wrapper: createWrapper(),
    });

    result.current.mutate(newDevice);

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(createdDevice);
    expect(deviceApi.createDevice).toHaveBeenCalledWith(newDevice);
  });
});

describe('useUpdateDevice', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should update device successfully', async () => {
    const updateData: UpdateDeviceRequest = {
      hostname: 'server-01-updated',
    };

    const updatedDevice: Device = {
      id: '1',
      hostname: 'server-01-updated',
      ip_address: '192.168.1.10',
      device_type: 'ipmi',
      status: 'healthy',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T01:00:00Z',
    };

    vi.mocked(deviceApi.updateDevice).mockResolvedValue(updatedDevice);

    const { result } = renderHook(() => useUpdateDevice(), {
      wrapper: createWrapper(),
    });

    result.current.mutate({ id: '1', data: updateData });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(updatedDevice);
    expect(deviceApi.updateDevice).toHaveBeenCalledWith('1', updateData);
  });
});

describe('useDeleteDevice', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should delete device successfully', async () => {
    vi.mocked(deviceApi.deleteDevice).mockResolvedValue(undefined);

    const { result } = renderHook(() => useDeleteDevice(), {
      wrapper: createWrapper(),
    });

    result.current.mutate('1');

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(deviceApi.deleteDevice).toHaveBeenCalledWith('1');
  });
});
