/**
 * React Query Hooks for Alert Rule Management
 * 
 * Custom hooks for alert rule CRUD operations using React Query.
 * - Automatic caching with 30-second stale time
 * - Optimistic updates for mutations
 * - Automatic cache invalidation on mutations
 * 
 * Requirements: 12.1, 12.2, 12.3, 12.5
 */

import {
  useQuery,
  useMutation,
  useQueryClient,
  UseQueryResult,
  UseMutationResult,
} from '@tanstack/react-query';
import {
  fetchAlertRules,
  fetchAlertRule,
  createAlertRule,
  updateAlertRule,
  deleteAlertRule,
} from '../services/alertRuleApi';
import {
  AlertRule,
  CreateAlertRuleRequest,
  UpdateAlertRuleRequest,
} from '../types/alert';

// Query keys for React Query cache management
export const alertRuleKeys = {
  all: ['alertRules'] as const,
  lists: () => [...alertRuleKeys.all, 'list'] as const,
  list: () => [...alertRuleKeys.lists()] as const,
  details: () => [...alertRuleKeys.all, 'detail'] as const,
  detail: (id: string) => [...alertRuleKeys.details(), id] as const,
};

/**
 * Hook: useAlertRules
 * 
 * Fetches list of all alert rules.
 * GET /api/v1/alert-rules
 * 
 * @returns Query result with alert rules list
 * 
 * Requirements: 12.2
 */
export const useAlertRules = (): UseQueryResult<AlertRule[], Error> => {
  return useQuery({
    queryKey: alertRuleKeys.list(),
    queryFn: fetchAlertRules,
    staleTime: 30000, // 30 seconds
    refetchOnWindowFocus: true,
    refetchOnMount: true,
  });
};

/**
 * Hook: useAlertRule
 * 
 * Fetches a single alert rule by ID.
 * GET /api/v1/alert-rules/{id}
 * 
 * @param id - Alert rule ID
 * @param enabled - Whether the query should run (default: true)
 * @returns Query result with alert rule details
 * 
 * Requirements: 12.2
 */
export const useAlertRule = (
  id: string,
  enabled: boolean = true
): UseQueryResult<AlertRule, Error> => {
  return useQuery({
    queryKey: alertRuleKeys.detail(id),
    queryFn: () => fetchAlertRule(id),
    staleTime: 30000, // 30 seconds
    enabled: enabled && !!id,
  });
};

/**
 * Hook: useCreateAlertRule
 * 
 * Creates a new alert rule.
 * POST /api/v1/alert-rules
 * 
 * Automatically invalidates alert rule list cache on success.
 * 
 * @returns Mutation result with create function
 * 
 * Requirements: 12.1, 12.3
 */
export const useCreateAlertRule = (): UseMutationResult<
  AlertRule,
  Error,
  CreateAlertRuleRequest,
  unknown
> => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createAlertRule,
    onSuccess: (newAlertRule: AlertRule) => {
      // Invalidate all alert rule list queries to refetch with new rule
      queryClient.invalidateQueries({ queryKey: alertRuleKeys.lists() });
      
      // Optionally set the new alert rule in cache
      queryClient.setQueryData(alertRuleKeys.detail(newAlertRule.id), newAlertRule);
    },
    onError: (error: Error) => {
      console.error('Failed to create alert rule:', error.message);
    },
  });
};

/**
 * Hook: useUpdateAlertRule
 * 
 * Updates an existing alert rule.
 * PUT /api/v1/alert-rules/{id}
 * 
 * Automatically invalidates alert rule cache on success.
 * 
 * @returns Mutation result with update function
 * 
 * Requirements: 12.5
 */
export const useUpdateAlertRule = (): UseMutationResult<
  AlertRule,
  Error,
  { id: string; data: UpdateAlertRuleRequest },
  unknown
> => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateAlertRuleRequest }) =>
      updateAlertRule(id, data),
    onSuccess: (updatedAlertRule: AlertRule) => {
      // Update the alert rule detail cache
      queryClient.setQueryData(
        alertRuleKeys.detail(updatedAlertRule.id),
        updatedAlertRule
      );
      
      // Invalidate alert rule lists to refetch with updated data
      queryClient.invalidateQueries({ queryKey: alertRuleKeys.lists() });
    },
    onError: (error: Error) => {
      console.error('Failed to update alert rule:', error.message);
    },
  });
};

/**
 * Hook: useDeleteAlertRule
 * 
 * Deletes an alert rule.
 * DELETE /api/v1/alert-rules/{id}
 * 
 * Automatically invalidates alert rule cache on success.
 * 
 * @returns Mutation result with delete function
 * 
 * Requirements: 12.5
 */
export const useDeleteAlertRule = (): UseMutationResult<
  void,
  Error,
  string,
  unknown
> => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteAlertRule,
    onSuccess: (_: void, deletedId: string) => {
      // Remove the alert rule from detail cache
      queryClient.removeQueries({ queryKey: alertRuleKeys.detail(deletedId) });
      
      // Invalidate alert rule lists to refetch without deleted rule
      queryClient.invalidateQueries({ queryKey: alertRuleKeys.lists() });
    },
    onError: (error: Error) => {
      console.error('Failed to delete alert rule:', error.message);
    },
  });
};
