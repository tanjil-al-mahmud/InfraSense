/**
 * React Query Hooks for Alert Management
 * 
 * Custom hooks for alert operations using React Query.
 * - Automatic caching with 15-second stale time for active alerts
 * - Automatic refetch every 15 seconds for active alerts
 * - Automatic cache invalidation on mutations
 * 
 * Requirements: 21.1, 21.2, 21.5, 21.7
 */

import {
  useQuery,
  useMutation,
  useQueryClient,
  UseQueryResult,
  UseMutationResult,
} from '@tanstack/react-query';
import {
  fetchAlerts,
  fetchAlertHistory,
  acknowledgeAlert,
} from '../services/alertApi';
import {
  Alert,
  AlertListParams,
  AcknowledgeResponse,
} from '../types/alert';

// Query keys for React Query cache management
export const alertKeys = {
  all: ['alerts'] as const,
  lists: () => [...alertKeys.all, 'list'] as const,
  list: (params?: AlertListParams) => [...alertKeys.lists(), params] as const,
  history: () => [...alertKeys.all, 'history'] as const,
  historyList: (params?: AlertListParams) => [...alertKeys.history(), params] as const,
};

/**
 * Hook: useAlerts
 * 
 * Fetches list of active alerts with optional filtering.
 * GET /api/v1/alerts
 * 
 * Automatically refetches every 15 seconds to keep alerts up-to-date.
 * 
 * @param params - Filtering parameters (severity, device)
 * @returns Query result with active alerts list
 * 
 * Requirements: 21.1, 21.3, 21.4, 21.7
 */
export const useAlerts = (
  params?: AlertListParams
): UseQueryResult<Alert[], Error> => {
  return useQuery({
    queryKey: alertKeys.list(params),
    queryFn: () => fetchAlerts(params),
    staleTime: 15000, // 15 seconds
    refetchInterval: 15000, // Auto-refetch every 15 seconds
    refetchOnWindowFocus: true,
    refetchOnMount: true,
  });
};

/**
 * Hook: useAlertHistory
 * 
 * Fetches alert history including resolved alerts with optional filtering.
 * GET /api/v1/alerts/history
 * 
 * @param params - Filtering parameters (severity, device)
 * @returns Query result with alert history
 * 
 * Requirements: 21.2, 21.3, 21.4
 */
export const useAlertHistory = (
  params?: AlertListParams
): UseQueryResult<Alert[], Error> => {
  return useQuery({
    queryKey: alertKeys.historyList(params),
    queryFn: () => fetchAlertHistory(params),
    staleTime: 30000, // 30 seconds (history doesn't change as frequently)
    refetchOnWindowFocus: true,
    refetchOnMount: true,
  });
};

/**
 * Hook: useAcknowledgeAlert
 * 
 * Acknowledges an alert.
 * POST /api/v1/alerts/{id}/acknowledge
 * 
 * Automatically invalidates alert caches on success to refetch updated data.
 * 
 * @returns Mutation result with acknowledge function
 * 
 * Requirements: 21.5
 */
export const useAcknowledgeAlert = (): UseMutationResult<
  AcknowledgeResponse,
  Error,
  string,
  unknown
> => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: acknowledgeAlert,
    onSuccess: () => {
      // Invalidate all alert queries to refetch with updated acknowledgment status
      queryClient.invalidateQueries({ queryKey: alertKeys.lists() });
      queryClient.invalidateQueries({ queryKey: alertKeys.history() });
    },
    onError: (error: Error) => {
      console.error('Failed to acknowledge alert:', error.message);
    },
  });
};

