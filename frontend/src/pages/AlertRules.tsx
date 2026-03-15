import React, { useState } from 'react';
import {
  useAlertRules,
  useCreateAlertRule,
  useUpdateAlertRule,
  useDeleteAlertRule,
} from '../hooks/useAlertRules';
import { AlertRule, AlertRuleOperator, AlertSeverity } from '../types/alert';
// AlertRuleForm will be created in task 5.13
import AlertRuleForm from '../components/AlertRuleForm';

/**
 * Alert Rules Page Component
 *
 * Displays a list of alert rules and provides CRUD operations:
 * - Create alert rule (modal with form)
 * - Edit alert rule (modal with form)
 * - Delete alert rule (confirmation dialog)
 *
 * Requirements: 22.4, 22.5, 22.6
 */

const OPERATOR_LABELS: Record<AlertRuleOperator, string> = {
  gt: '>',
  lt: '<',
  eq: '=',
  ne: '≠',
};

const SEVERITY_STYLES: Record<AlertSeverity, { bg: string; text: string }> = {
  critical: { bg: '#450a0a', text: '#f87171' },
  warning: { bg: '#422006', text: '#fbbf24' },
  info: { bg: '#0c1a2e', text: '#60a5fa' },
};

const AlertRules: React.FC = () => {
  const { data: alertRules, isLoading, isError, error } = useAlertRules();
  const createMutation = useCreateAlertRule();
  const updateMutation = useUpdateAlertRule();
  const deleteMutation = useDeleteAlertRule();

  // Modal state
  const [showFormModal, setShowFormModal] = useState(false);
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null);

  // Delete confirmation state
  const [deletingRule, setDeletingRule] = useState<AlertRule | null>(null);

  const handleCreateClick = () => {
    setEditingRule(null);
    setShowFormModal(true);
  };

  const handleEditClick = (rule: AlertRule) => {
    setEditingRule(rule);
    setShowFormModal(true);
  };

  const handleDeleteClick = (rule: AlertRule) => {
    setDeletingRule(rule);
  };

  const handleFormSubmit = (data: Parameters<typeof createMutation.mutate>[0]) => {
    if (editingRule) {
      updateMutation.mutate(
        { id: editingRule.id, data },
        {
          onSuccess: () => {
            setShowFormModal(false);
            setEditingRule(null);
          },
        }
      );
    } else {
      createMutation.mutate(data, {
        onSuccess: () => {
          setShowFormModal(false);
        },
      });
    }
  };

  const handleFormCancel = () => {
    setShowFormModal(false);
    setEditingRule(null);
  };

  const handleDeleteConfirm = () => {
    if (!deletingRule) return;
    deleteMutation.mutate(deletingRule.id, {
      onSuccess: () => setDeletingRule(null),
    });
  };

  const handleDeleteCancel = () => {
    setDeletingRule(null);
  };

  const isMutating =
    createMutation.isPending || updateMutation.isPending || deleteMutation.isPending;

  return (
    <div style={styles.page}>
      {/* Page header */}
      <div style={styles.pageHeader}>
        <div>
          <h1 style={styles.pageTitle}>Alert Rules</h1>
          {alertRules && (
            <span style={styles.totalCount}>
              {alertRules.length} rule{alertRules.length !== 1 ? 's' : ''}
            </span>
          )}
        </div>
        <button
          onClick={handleCreateClick}
          style={styles.createBtn}
          aria-label="Create alert rule"
        >
          + Create Alert Rule
        </button>
      </div>

      {/* Mutation error banners */}
      {createMutation.isError && (
        <div style={styles.errorBanner} role="alert">
          Failed to create alert rule: {createMutation.error?.message}
        </div>
      )}
      {updateMutation.isError && (
        <div style={styles.errorBanner} role="alert">
          Failed to update alert rule: {updateMutation.error?.message}
        </div>
      )}
      {deleteMutation.isError && (
        <div style={styles.errorBanner} role="alert">
          Failed to delete alert rule: {deleteMutation.error?.message}
        </div>
      )}

      {/* Content */}
      {isLoading && (
        <div style={styles.centered}>
          <p>Loading alert rules...</p>
        </div>
      )}

      {isError && error && (
        <div style={styles.errorBanner} role="alert">
          Failed to load alert rules: {error.message}
        </div>
      )}

      {!isLoading && !isError && (
        <>
          {!alertRules || alertRules.length === 0 ? (
            <div style={styles.centered}>
              <p style={styles.emptyText}>No alert rules configured.</p>
              <p style={styles.emptySubText}>
                Create an alert rule to start monitoring metric thresholds.
              </p>
            </div>
          ) : (
            <div style={styles.tableWrapper}>
              <table style={styles.table}>
                <thead>
                  <tr>
                    <th style={styles.th}>Name</th>
                    <th style={styles.th}>Metric</th>
                    <th style={styles.th}>Condition</th>
                    <th style={styles.th}>Severity</th>
                    <th style={styles.th}>Status</th>
                    <th style={styles.th}>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {alertRules.map((rule) => {
                    const severityStyle =
                      SEVERITY_STYLES[rule.severity] ?? SEVERITY_STYLES.info;
                    return (
                      <tr key={rule.id} style={styles.tr}>
                        <td style={{ ...styles.td, fontWeight: 500 }}>{rule.name}</td>
                        <td style={styles.td}>
                          <code style={styles.metricCode}>{rule.metric_name}</code>
                        </td>
                        <td style={styles.td}>
                          <span style={styles.condition}>
                            {OPERATOR_LABELS[rule.operator] ?? rule.operator}{' '}
                            {rule.threshold}
                          </span>
                        </td>
                        <td style={styles.td}>
                          <span
                            style={{
                              ...styles.severityBadge,
                              backgroundColor: severityStyle.bg,
                              color: severityStyle.text,
                            }}
                          >
                            {rule.severity}
                          </span>
                        </td>
                        <td style={styles.td}>
                          <span
                            style={{
                              ...styles.statusBadge,
                              ...(rule.enabled
                                ? styles.statusEnabled
                                : styles.statusDisabled),
                            }}
                          >
                            {rule.enabled ? 'Enabled' : 'Disabled'}
                          </span>
                        </td>
                        <td style={styles.td}>
                          <div style={styles.actions}>
                            <button
                              onClick={() => handleEditClick(rule)}
                              style={styles.editBtn}
                              disabled={isMutating}
                              aria-label={`Edit alert rule ${rule.name}`}
                            >
                              Edit
                            </button>
                            <button
                              onClick={() => handleDeleteClick(rule)}
                              style={styles.deleteBtn}
                              disabled={isMutating}
                              aria-label={`Delete alert rule ${rule.name}`}
                            >
                              Delete
                            </button>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {/* Create / Edit Modal */}
      {showFormModal && (
        <div style={styles.modalOverlay} role="dialog" aria-modal="true">
          <div style={styles.modalContent}>
            <h2 style={styles.modalTitle}>
              {editingRule ? 'Edit Alert Rule' : 'Create Alert Rule'}
            </h2>
            <AlertRuleForm
              initialValues={editingRule ?? undefined}
              onSubmit={handleFormSubmit}
              onCancel={handleFormCancel}
              isSubmitting={createMutation.isPending || updateMutation.isPending}
            />
          </div>
        </div>
      )}

      {/* Delete Confirmation Dialog */}
      {deletingRule && (
        <div style={styles.modalOverlay} role="dialog" aria-modal="true">
          <div style={{ ...styles.modalContent, maxWidth: '420px' }}>
            <h2 style={styles.modalTitle}>Delete Alert Rule</h2>
            <p style={styles.confirmText}>
              Are you sure you want to delete the alert rule{' '}
              <strong>{deletingRule.name}</strong>? This action cannot be undone.
            </p>
            <div style={styles.confirmActions}>
              <button
                onClick={handleDeleteCancel}
                style={styles.cancelBtn}
                disabled={deleteMutation.isPending}
              >
                Cancel
              </button>
              <button
                onClick={handleDeleteConfirm}
                style={styles.confirmDeleteBtn}
                disabled={deleteMutation.isPending}
              >
                {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

const styles: Record<string, React.CSSProperties> = {
  page: { padding: '1.5rem', maxWidth: '1200px', margin: '0 auto' },
  pageHeader: { display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginBottom: '1.25rem', gap: '1rem' },
  pageTitle: { fontSize: '1.5rem', fontWeight: 700, color: '#f1f5f9', margin: 0 },
  totalCount: { fontSize: '0.875rem', color: '#64748b', display: 'block', marginTop: '0.25rem' },
  createBtn: { backgroundColor: '#2563eb', border: 'none', borderRadius: '6px', color: '#fff', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 600, padding: '0.5rem 1rem', whiteSpace: 'nowrap' },
  errorBanner: { backgroundColor: '#450a0a', border: '1px solid #dc2626', borderRadius: '6px', color: '#f87171', fontSize: '0.875rem', padding: '0.75rem 1rem', marginBottom: '1rem' },
  centered: { textAlign: 'center', padding: '3rem 0' },
  emptyText: { color: '#94a3b8', fontSize: '1rem', fontWeight: 500, margin: '0 0 0.5rem' },
  emptySubText: { color: '#64748b', fontSize: '0.875rem', margin: 0 },
  tableWrapper: { overflowX: 'auto', borderRadius: '10px', border: '1px solid #1e293b' },
  table: { width: '100%', borderCollapse: 'collapse', fontSize: '0.875rem' },
  th: { backgroundColor: '#1e293b', borderBottom: '1px solid #334155', color: '#64748b', fontWeight: 700, fontSize: '0.7rem', textTransform: 'uppercase', letterSpacing: '0.08em', padding: '0.75rem 1rem', textAlign: 'left', whiteSpace: 'nowrap' },
  tr: { borderBottom: '1px solid #1e293b' },
  td: { padding: '0.75rem 1rem', color: '#94a3b8', verticalAlign: 'middle' },
  metricCode: { backgroundColor: '#1e293b', borderRadius: '4px', fontFamily: 'monospace', fontSize: '0.8rem', padding: '0.15rem 0.4rem', color: '#60a5fa', border: '1px solid #334155' },
  condition: { fontFamily: 'monospace', fontSize: '0.875rem', color: '#e2e8f0' },
  severityBadge: { borderRadius: '9999px', display: 'inline-block', fontSize: '0.75rem', fontWeight: 600, padding: '0.2rem 0.6rem', textTransform: 'capitalize', whiteSpace: 'nowrap' },
  statusBadge: { borderRadius: '9999px', display: 'inline-block', fontSize: '0.75rem', fontWeight: 600, padding: '0.2rem 0.6rem', whiteSpace: 'nowrap' },
  statusEnabled: { backgroundColor: '#052e16', color: '#4ade80' },
  statusDisabled: { backgroundColor: '#111827', color: '#6b7280' },
  actions: { display: 'flex', gap: '0.5rem' },
  editBtn: { backgroundColor: '#1e293b', border: '1px solid #334155', borderRadius: '4px', color: '#94a3b8', cursor: 'pointer', fontSize: '0.8rem', fontWeight: 500, padding: '0.3rem 0.75rem' },
  deleteBtn: { backgroundColor: '#450a0a', border: '1px solid #dc2626', borderRadius: '4px', color: '#f87171', cursor: 'pointer', fontSize: '0.8rem', fontWeight: 500, padding: '0.3rem 0.75rem' },
  modalOverlay: { alignItems: 'center', backgroundColor: 'rgba(0,0,0,0.7)', bottom: 0, display: 'flex', justifyContent: 'center', left: 0, position: 'fixed', right: 0, top: 0, zIndex: 1000 },
  modalContent: { backgroundColor: '#0f172a', border: '1px solid #1e293b', borderRadius: '12px', boxShadow: '0 25px 60px rgba(0,0,0,0.5)', maxHeight: '90vh', maxWidth: '560px', overflowY: 'auto', padding: '1.5rem', width: '100%' },
  modalTitle: { fontSize: '1.125rem', fontWeight: 700, color: '#f1f5f9', margin: '0 0 1.25rem' },
  confirmText: { color: '#94a3b8', fontSize: '0.9rem', lineHeight: 1.6, margin: '0 0 1.5rem' },
  confirmActions: { display: 'flex', gap: '0.75rem', justifyContent: 'flex-end' },
  cancelBtn: { backgroundColor: '#1e293b', border: '1px solid #334155', borderRadius: '6px', color: '#94a3b8', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 500, padding: '0.5rem 1rem' },
  confirmDeleteBtn: { backgroundColor: '#dc2626', border: 'none', borderRadius: '6px', color: '#fff', cursor: 'pointer', fontSize: '0.875rem', fontWeight: 600, padding: '0.5rem 1rem' },
};

export default AlertRules;
