/**
 * React Query Hooks for Device Management
 * 
 * Custom hooks for device CRUD operations using React Query.
 * - Automatic caching with 30-second stale time
 * - Optimistic updates for mutations
 * - Automatic cache invalidation on mutations
 * 
 * Requirements: 13.1, 13.2, 13.3, 13.4
 */

import {
  useQuery,
  useMutation,
  useQueryClient,
  UseQueryResult,
  UseMutationResult,
} from '@tanstack/react-query';
import {
  fetchDevices,
  fetchDevice,
  createDevice,
  updateDevice,
  deleteDevice,
} from '../services/deviceApi';
import {
  Device,
  CreateDeviceRequest,
  UpdateDeviceRequest,
  DeviceListParams,
  PaginatedResponse,
} from '../types/device';

// Query keys for React Query cache management
export const deviceKeys = {
  all: ['devices'] as const,
  lists: () => [...deviceKeys.all, 'list'] as const,
  list: (params?: DeviceListParams) => [...deviceKeys.lists(), params] as const,
  details: () => [...deviceKeys.all, 'detail'] as const,
  detail: (id: string) => [...deviceKeys.details(), id] as const,
};

/**
 * Hook: useDevices
 * 
 * Fetches paginated list of devices with optional filtering.
 * GET /api/v1/devices
 * 
 * @param params - Pagination and filtering parameters
 * @returns Query result with devices list and pagination metadata
 * 
 * Requirements: 13.1, 13.2, 13.4
 */
export const useDevices = (
  params?: DeviceListParams
): UseQueryResult<PaginatedResponse<Device>, Error> => {
  return useQuery({
    queryKey: deviceKeys.list(params),
    queryFn: () => fetchDevices(params),
    staleTime: 30000, // 30 seconds
    refetchOnWindowFocus: true,
    refetchOnMount: true,
  });
};

/**
 * Hook: useDevice
 * 
 * Fetches a single device by ID.
 * GET /api/v1/devices/{id}
 * 
 * @param id - Device ID
 * @param enabled - Whether the query should run (default: true)
 * @returns Query result with device details
 * 
 * Requirements: 13.2
 */
export const useDevice = (
  id: string,
  enabled: boolean = true
): UseQueryResult<Device, Error> => {
  return useQuery({
    queryKey: deviceKeys.detail(id),
    queryFn: () => fetchDevice(id),
    staleTime: 30000, // 30 seconds
    enabled: enabled && !!id,
  });
};

/**
 * Hook: useCreateDevice
 * 
 * Creates a new device.
 * POST /api/v1/devices
 * 
 * Automatically invalidates device list cache on success.
 * 
 * @returns Mutation result with create function
 * 
 * Requirements: 13.1, 13.3
 */
export const useCreateDevice = (): UseMutationResult<
  Device,
  Error,
  CreateDeviceRequest,
  unknown
> => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createDevice,
    onSuccess: (newDevice: Device) => {
      // Invalidate all device list queries to refetch with new device
      queryClient.invalidateQueries({ queryKey: deviceKeys.lists() });
      
      // Optionally set the new device in cache
      queryClient.setQueryData(deviceKeys.detail(newDevice.id), newDevice);
    },
    onError: (error: Error) => {
      console.error('Failed to create device:', error.message);
    },
  });
};

/**
 * Hook: useUpdateDevice
 * 
 * Updates an existing device.
 * PUT /api/v1/devices/{id}
 * 
 * Automatically invalidates device cache on success.
 * 
 * @returns Mutation result with update function
 * 
 * Requirements: 13.4
 */
export const useUpdateDevice = (): UseMutationResult<
  Device,
  Error,
  { id: string; data: UpdateDeviceRequest },
  unknown
> => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateDeviceRequest }) => updateDevice(id, data),
    onSuccess: (updatedDevice: Device) => {
      // Update the device detail cache
      queryClient.setQueryData(
        deviceKeys.detail(updatedDevice.id),
        updatedDevice
      );
      
      // Invalidate device lists to refetch with updated data
      queryClient.invalidateQueries({ queryKey: deviceKeys.lists() });
    },
    onError: (error: Error) => {
      console.error('Failed to update device:', error.message);
    },
  });
};

/**
 * Hook: useDeleteDevice
 * 
 * Deletes a device.
 * DELETE /api/v1/devices/{id}
 * 
 * Automatically invalidates device cache on success.
 * 
 * @returns Mutation result with delete function
 * 
 * Requirements: 13.4
 */
export const useDeleteDevice = (): UseMutationResult<
  void,
  Error,
  string,
  unknown
> => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteDevice,
    onSuccess: (_: void, deletedId: string) => {
      // Remove the device from detail cache
      queryClient.removeQueries({ queryKey: deviceKeys.detail(deletedId) });
      
      // Invalidate device lists to refetch without deleted device
      queryClient.invalidateQueries({ queryKey: deviceKeys.lists() });
    },
    onError: (error: Error) => {
      console.error('Failed to delete device:', error.message);
    },
  });
};

import { testDeviceConnection, syncDevice } from '../services/deviceApi';
import { ConnectionTestResult, DeviceSyncResult } from '../types/device';

/**
 * Hook: useTestConnection
 * Tests BMC connectivity for a device.
 */
export const useTestConnection = (): UseMutationResult<ConnectionTestResult, Error, string, unknown> => {
  return useMutation({
    mutationFn: testDeviceConnection,
  });
};

/**
 * Hook: useSyncDevice
 * Triggers a full hardware sync from the BMC.
 * Invalidates device detail cache on success.
 */
export const useSyncDevice = (): UseMutationResult<DeviceSyncResult, Error, string, unknown> => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: syncDevice,
    onSuccess: (_result, deviceId) => {
      queryClient.invalidateQueries({ queryKey: deviceKeys.detail(deviceId) });
      queryClient.invalidateQueries({ queryKey: deviceKeys.lists() });
    },
  });
};

import { powerControlDevice } from '../services/deviceApi';
import { PowerControlRequest, PowerControlResult } from '../types/device';

/**
 * Hook: usePowerControl
 * Sends a power control action (On/Off/Restart/etc.) to the BMC.
 * Invalidates device detail cache on success so power state refreshes.
 */
export const usePowerControl = (): UseMutationResult<
  PowerControlResult,
  Error,
  { id: string; req: PowerControlRequest },
  unknown
> => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, req }: { id: string; req: PowerControlRequest }) =>
      powerControlDevice(id, req),
    onSuccess: (_result, { id }) => {
      // Refresh device detail so power state updates
      queryClient.invalidateQueries({ queryKey: deviceKeys.detail(id) });
    },
  });
};

import { bootControlDevice } from '../services/deviceApi';
import { BootControlRequest, BootControlResult } from '../types/device';

/**
 * Hook: useBootControl
 * Sends a boot override action (Pxe/Cd/Hdd/BiosSetup/None) to the BMC.
 */
export const useBootControl = (): UseMutationResult<
  BootControlResult,
  Error,
  { id: string; req: BootControlRequest },
  unknown
> => {
  return useMutation({
    mutationFn: ({ id, req }: { id: string; req: BootControlRequest }) =>
      bootControlDevice(id, req),
  });
};
