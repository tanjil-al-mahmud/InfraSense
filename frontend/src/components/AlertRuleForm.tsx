import React from 'react';
import { useForm } from 'react-hook-form';
import {
  AlertRule,
  AlertRuleOperator,
  AlertSeverity,
  CreateAlertRuleRequest,
} from '../types/alert';
import { useDevices } from '../hooks/useDevices';
import { Device } from '../types/device';

/**
 * AlertRuleForm Component
 *
 * Reusable form for creating or editing an alert rule.
 * - Fields: name, metric name, comparison operator, threshold, severity, device
 * - Client-side validation via React Hook Form (required fields, numeric threshold)
 * - Displays API server validation errors via root error
 *
 * Requirements: 22.4, 22.5
 */

// Common hardware/OS metrics available for alerting
const AVAILABLE_METRICS = [
  'ipmi_temperature_celsius',
  'ipmi_fan_speed_rpm',
  'ipmi_voltage',
  'ipmi_psu_status',
  'redfish_temperature_celsius',
  'redfish_fan_speed_rpm',
  'redfish_psu_status',
  'redfish_raid_status',
  'redfish_disk_smart_status',
  'node_cpu_usage_percent',
  'node_memory_usage_percent',
  'node_disk_usage_percent',
  'node_network_bytes_sent',
  'node_network_bytes_received',
  'windows_cpu_usage_percent',
  'windows_memory_usage_percent',
  'windows_disk_usage_percent',
  'snmp_ups_battery_charge_percent',
  'snmp_ups_input_voltage',
  'snmp_ups_output_voltage',
  'snmp_ups_load_percent',
  'snmp_ups_runtime_minutes',
];

const OPERATOR_OPTIONS: { value: AlertRuleOperator; label: string }[] = [
  { value: 'gt', label: '> Greater than' },
  { value: 'lt', label: '< Less than' },
  { value: 'eq', label: '= Equal to' },
  { value: 'ne', label: '≠ Not equal to' },
];

const SEVERITY_OPTIONS: { value: AlertSeverity; label: string }[] = [
  { value: 'info', label: 'Info' },
  { value: 'warning', label: 'Warning' },
  { value: 'critical', label: 'Critical' },
];

interface FormValues {
  name: string;
  metric_name: string;
  operator: AlertRuleOperator;
  threshold: string; // string in form, parsed to number on submit
  severity: AlertSeverity;
  device_id: string;
}

export interface AlertRuleFormProps {
  initialValues?: AlertRule;
  onSubmit: (data: CreateAlertRuleRequest) => void;
  onCancel: () => void;
  isSubmitting: boolean;
}

const AlertRuleForm: React.FC<AlertRuleFormProps> = ({
  initialValues,
  onSubmit,
  onCancel,
  isSubmitting,
}) => {
  const isEditMode = !!initialValues;

  const { data: devicesResponse } = useDevices({ page: 1, page_size: 200 });
  const devices = devicesResponse?.data ?? [];

  const {
    register,
    handleSubmit,
    formState: { errors },
    setError,
  } = useForm<FormValues>({
    defaultValues: {
      name: initialValues?.name ?? '',
      metric_name: initialValues?.metric_name ?? '',
      operator: initialValues?.operator ?? 'gt',
      threshold: initialValues?.threshold !== undefined ? String(initialValues.threshold) : '',
      severity: initialValues?.severity ?? 'warning',
      device_id: initialValues?.device_id ?? '',
    },
  });

  const handleFormSubmit = (data: FormValues) => {
    const threshold = parseFloat(data.threshold);
    if (isNaN(threshold)) {
      setError('threshold', { message: 'Threshold must be a valid number' });
      return;
    }

    const payload: CreateAlertRuleRequest = {
      name: data.name,
      metric_name: data.metric_name,
      operator: data.operator,
      threshold,
      severity: data.severity,
      device_id: data.device_id || undefined,
    };

    try {
      onSubmit(payload);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'An unexpected error occurred';
      setError('root', { message });
    }
  };

  return (
    <form onSubmit={handleSubmit(handleFormSubmit)} noValidate style={styles.form}>
      {/* API / root error */}
      {errors.root && (
        <div style={styles.errorBanner} role="alert">
          {errors.root.message}
        </div>
      )}

      {/* Name */}
      <div style={styles.fieldGroup}>
        <label htmlFor="name" style={styles.label}>
          Rule Name <span style={styles.required}>*</span>
        </label>
        <input
          id="name"
          type="text"
          placeholder="e.g. High CPU Temperature"
          disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.name ? styles.inputError : {}) }}
          {...register('name', { required: 'Rule name is required' })}
        />
        {errors.name && (
          <p style={styles.fieldError} role="alert">
            {errors.name.message}
          </p>
        )}
      </div>

      {/* Metric Name */}
      <div style={styles.fieldGroup}>
        <label htmlFor="metric_name" style={styles.label}>
          Metric Name <span style={styles.required}>*</span>
        </label>
        <select
          id="metric_name"
          disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.metric_name ? styles.inputError : {}) }}
          {...register('metric_name', { required: 'Metric name is required' })}
        >
          <option value="">Select a metric...</option>
          {AVAILABLE_METRICS.map((m) => (
            <option key={m} value={m}>
              {m}
            </option>
          ))}
        </select>
        {errors.metric_name && (
          <p style={styles.fieldError} role="alert">
            {errors.metric_name.message}
          </p>
        )}
      </div>

      {/* Operator */}
      <div style={styles.fieldGroup}>
        <label htmlFor="operator" style={styles.label}>
          Operator <span style={styles.required}>*</span>
        </label>
        <select
          id="operator"
          disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.operator ? styles.inputError : {}) }}
          {...register('operator', { required: 'Operator is required' })}
        >
          {OPERATOR_OPTIONS.map((op) => (
            <option key={op.value} value={op.value}>
              {op.label}
            </option>
          ))}
        </select>
        {errors.operator && (
          <p style={styles.fieldError} role="alert">
            {errors.operator.message}
          </p>
        )}
      </div>

      {/* Threshold */}
      <div style={styles.fieldGroup}>
        <label htmlFor="threshold" style={styles.label}>
          Threshold <span style={styles.required}>*</span>
        </label>
        <input
          id="threshold"
          type="number"
          step="any"
          placeholder="e.g. 85"
          disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.threshold ? styles.inputError : {}) }}
          {...register('threshold', {
            required: 'Threshold is required',
            validate: (val: string) =>
              (val !== '' && !isNaN(parseFloat(val))) || 'Threshold must be a valid number',
          })}
        />
        {errors.threshold && (
          <p style={styles.fieldError} role="alert">
            {errors.threshold.message}
          </p>
        )}
      </div>

      {/* Severity */}
      <div style={styles.fieldGroup}>
        <label htmlFor="severity" style={styles.label}>
          Severity <span style={styles.required}>*</span>
        </label>
        <select
          id="severity"
          disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.severity ? styles.inputError : {}) }}
          {...register('severity', { required: 'Severity is required' })}
        >
          {SEVERITY_OPTIONS.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </select>
        {errors.severity && (
          <p style={styles.fieldError} role="alert">
            {errors.severity.message}
          </p>
        )}
      </div>

      {/* Device (optional) */}
      <div style={styles.fieldGroup}>
        <label htmlFor="device_id" style={styles.label}>
          Device
        </label>
        <select
          id="device_id"
          disabled={isSubmitting}
          style={styles.input}
          {...register('device_id')}
        >
          <option value="">All devices (no filter)</option>
          {devices.map((d: Device) => (
            <option key={d.id} value={d.id}>
              {d.hostname} ({d.ip_address})
            </option>
          ))}
        </select>
        <p style={styles.hint}>Optionally restrict this rule to a specific device.</p>
      </div>

      {/* Actions */}
      <div style={styles.actions}>
        <button
          type="button"
          onClick={onCancel}
          disabled={isSubmitting}
          style={styles.cancelBtn}
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={isSubmitting}
          style={{ ...styles.submitBtn, ...(isSubmitting ? styles.btnDisabled : {}) }}
        >
          {isSubmitting
            ? isEditMode
              ? 'Saving...'
              : 'Creating...'
            : isEditMode
            ? 'Save Changes'
            : 'Create Rule'}
        </button>
      </div>
    </form>
  );
};

const styles: Record<string, React.CSSProperties> = {
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: '1rem',
  },
  errorBanner: {
    backgroundColor: '#fee2e2',
    border: '1px solid #fca5a5',
    borderRadius: '4px',
    color: '#b91c1c',
    fontSize: '0.875rem',
    padding: '0.75rem 1rem',
  },
  fieldGroup: {
    display: 'flex',
    flexDirection: 'column',
    gap: '0.25rem',
  },
  label: {
    fontSize: '0.875rem',
    fontWeight: 500,
    color: '#374151',
  },
  required: {
    color: '#ef4444',
  },
  input: {
    border: '1px solid #d1d5db',
    borderRadius: '4px',
    fontSize: '0.875rem',
    padding: '0.5rem 0.75rem',
    width: '100%',
    boxSizing: 'border-box' as const,
    backgroundColor: '#fff',
    outline: 'none',
  },
  inputError: {
    borderColor: '#ef4444',
  },
  fieldError: {
    color: '#ef4444',
    fontSize: '0.75rem',
    margin: 0,
  },
  hint: {
    color: '#9ca3af',
    fontSize: '0.75rem',
    margin: 0,
  },
  actions: {
    display: 'flex',
    gap: '0.75rem',
    justifyContent: 'flex-end',
    marginTop: '0.5rem',
  },
  cancelBtn: {
    backgroundColor: '#f3f4f6',
    border: '1px solid #d1d5db',
    borderRadius: '4px',
    color: '#374151',
    cursor: 'pointer',
    fontSize: '0.875rem',
    fontWeight: 500,
    padding: '0.5rem 1rem',
  },
  submitBtn: {
    backgroundColor: '#2563eb',
    border: 'none',
    borderRadius: '4px',
    color: '#fff',
    cursor: 'pointer',
    fontSize: '0.875rem',
    fontWeight: 600,
    padding: '0.5rem 1.25rem',
  },
  btnDisabled: {
    opacity: 0.6,
    cursor: 'not-allowed',
  },
};

export default AlertRuleForm;
