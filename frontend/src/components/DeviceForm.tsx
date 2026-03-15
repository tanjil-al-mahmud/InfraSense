import React from 'react';
import { useForm, SubmitHandler, FieldValues } from 'react-hook-form';
import { useNavigate } from 'react-router-dom';
import { useCreateDevice, useUpdateDevice } from '../hooks/useDevices';
import { Device, CreateDeviceRequest, UpdateDeviceRequest, DeviceType } from '../types/device';

/**
 * DeviceForm Component
 *
 * Reusable form for registering a new device or editing an existing one.
 * - Fields: hostname, IP address, BMC IP address, device type, location, tags
 * - Client-side validation via React Hook Form (required fields, IP format)
 * - Displays API server validation errors
 * - Supports create mode (no device prop) and edit mode (device prop provided)
 *
 * Requirements: 22.1, 22.2, 22.7
 */

const DEVICE_TYPE_LABELS: Record<DeviceType, string> = {
  ipmi: 'IPMI',
  redfish: 'Redfish',
  snmp: 'SNMP',
  proxmox: 'Proxmox',
  linux_agent: 'Linux Agent',
  windows_agent: 'Windows Agent',
};

const ALL_DEVICE_TYPES: DeviceType[] = [
  'ipmi',
  'redfish',
  'snmp',
  'proxmox',
  'linux_agent',
  'windows_agent',
];

// Validates IPv4 or IPv6 address format
const IP_PATTERN =
  /^((\d{1,3}\.){3}\d{1,3}|([\da-fA-F]{0,4}:){2,7}[\da-fA-F]{0,4}|::[\da-fA-F]{0,4})$/;

interface FormValues extends FieldValues {
  hostname: string;
  ip_address: string;
  bmc_ip_address?: string;
  device_type: DeviceType;
  location?: string;
  tags?: string; // comma-separated string in the form
}

export interface DeviceFormProps {
  /** When provided, the form operates in edit mode pre-populated with this device's data */
  device?: Device;
  /** Called after a successful create or update */
  onSuccess?: (device: Device) => void;
  /** Called when the user cancels */
  onCancel?: () => void;
}

const DeviceForm: React.FC<DeviceFormProps> = ({ device, onSuccess, onCancel }) => {
  const navigate = useNavigate();
  const isEditMode = !!device;

  const createDevice = useCreateDevice();
  const updateDevice = useUpdateDevice();

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setError,
  } = useForm<FormValues>({
    defaultValues: {
      hostname: device?.hostname ?? '',
      ip_address: device?.ip_address ?? '',
      bmc_ip_address: device?.bmc_ip_address ?? '',
      device_type: (device?.device_type as DeviceType) ?? 'ipmi',
      location: device?.location ?? '',
      tags: device?.tags?.join(', ') ?? '',
    },
  });

  const parseTags = (raw: string): string[] =>
    raw
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean);

  const onSubmit: SubmitHandler<FormValues> = async (data) => {
    const tags = parseTags(data.tags ?? '');

    try {
      if (isEditMode && device) {
        const payload: UpdateDeviceRequest = {
          hostname: data.hostname,
          ip_address: data.ip_address,
          bmc_ip_address: data.bmc_ip_address || undefined,
          device_type: data.device_type,
          location: data.location || undefined,
          tags: tags.length > 0 ? tags : undefined,
        };
        const updated = await updateDevice.mutateAsync({ id: device.id, data: payload });
        onSuccess?.(updated);
      } else {
        const payload: CreateDeviceRequest = {
          hostname: data.hostname,
          ip_address: data.ip_address,
          bmc_ip_address: data.bmc_ip_address || undefined,
          device_type: data.device_type,
          location: data.location || undefined,
          tags: tags.length > 0 ? tags : undefined,
        };
        const created = await createDevice.mutateAsync(payload);
        onSuccess?.(created);
        if (!onSuccess) {
          navigate(`/devices/${created.id}`);
        }
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'An unexpected error occurred';
      setError('root', { message });
    }
  };

  const handleCancel = () => {
    if (onCancel) {
      onCancel();
    } else {
      navigate(-1);
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate style={styles.form}>
      {/* API / root error */}
      {errors.root && (
        <div style={styles.errorBanner} role="alert">
          {errors.root.message}
        </div>
      )}

      {/* Hostname */}
      <div style={styles.fieldGroup}>
        <label htmlFor="hostname" style={styles.label}>
          Hostname <span style={styles.required}>*</span>
        </label>
        <input
          id="hostname"
          type="text"
          placeholder="e.g. server-01.example.com"
          disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.hostname ? styles.inputError : {}) }}
          {...register('hostname', { required: 'Hostname is required' })}
        />
        {errors.hostname && (
          <p style={styles.fieldError} role="alert">
            {errors.hostname.message}
          </p>
        )}
      </div>

      {/* IP Address */}
      <div style={styles.fieldGroup}>
        <label htmlFor="ip_address" style={styles.label}>
          IP Address <span style={styles.required}>*</span>
        </label>
        <input
          id="ip_address"
          type="text"
          placeholder="e.g. 192.168.1.10"
          disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.ip_address ? styles.inputError : {}) }}
          {...register('ip_address', {
            required: 'IP address is required',
            pattern: {
              value: IP_PATTERN,
              message: 'Enter a valid IPv4 or IPv6 address',
            },
          })}
        />
        {errors.ip_address && (
          <p style={styles.fieldError} role="alert">
            {errors.ip_address.message}
          </p>
        )}
      </div>

      {/* BMC IP Address */}
      <div style={styles.fieldGroup}>
        <label htmlFor="bmc_ip_address" style={styles.label}>
          BMC IP Address
        </label>
        <input
          id="bmc_ip_address"
          type="text"
          placeholder="e.g. 192.168.1.11 (optional)"
          disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.bmc_ip_address ? styles.inputError : {}) }}
          {...register('bmc_ip_address', {
            validate: (val) =>
              !val || IP_PATTERN.test(val) || 'Enter a valid IPv4 or IPv6 address',
          })}
        />
        {errors.bmc_ip_address && (
          <p style={styles.fieldError} role="alert">
            {errors.bmc_ip_address.message}
          </p>
        )}
      </div>

      {/* Device Type */}
      <div style={styles.fieldGroup}>
        <label htmlFor="device_type" style={styles.label}>
          Device Type <span style={styles.required}>*</span>
        </label>
        <select
          id="device_type"
          disabled={isSubmitting}
          style={{ ...styles.input, ...(errors.device_type ? styles.inputError : {}) }}
          {...register('device_type', { required: 'Device type is required' })}
        >
          {ALL_DEVICE_TYPES.map((t) => (
            <option key={t} value={t}>
              {DEVICE_TYPE_LABELS[t]}
            </option>
          ))}
        </select>
        {errors.device_type && (
          <p style={styles.fieldError} role="alert">
            {errors.device_type.message}
          </p>
        )}
      </div>

      {/* Location */}
      <div style={styles.fieldGroup}>
        <label htmlFor="location" style={styles.label}>
          Location
        </label>
        <input
          id="location"
          type="text"
          placeholder="e.g. Rack A, Row 3 (optional)"
          disabled={isSubmitting}
          style={styles.input}
          {...register('location')}
        />
      </div>

      {/* Tags */}
      <div style={styles.fieldGroup}>
        <label htmlFor="tags" style={styles.label}>
          Tags
        </label>
        <input
          id="tags"
          type="text"
          placeholder="e.g. production, web, critical (comma-separated)"
          disabled={isSubmitting}
          style={styles.input}
          {...register('tags')}
        />
        <p style={styles.hint}>Separate multiple tags with commas.</p>
      </div>

      {/* Actions */}
      <div style={styles.actions}>
        <button
          type="button"
          onClick={handleCancel}
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
              : 'Registering...'
            : isEditMode
            ? 'Save Changes'
            : 'Register Device'}
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

export default DeviceForm;
