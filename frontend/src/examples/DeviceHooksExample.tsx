/**
 * Example: Device Hooks Usage
 * 
 * Demonstrates how to use the device management React Query hooks
 * in React components.
 */

import React, { useState } from 'react';
import {
  useDevices,
  useDevice,
  useCreateDevice,
  useUpdateDevice,
  useDeleteDevice,
} from '../hooks/useDevices';
import { CreateDeviceRequest, UpdateDeviceRequest } from '../types/device';

/**
 * Example 1: Fetching and displaying a list of devices
 */
export const DeviceListExample: React.FC = () => {
  const [page, setPage] = useState(1);
  const { data, isLoading, isError, error } = useDevices({
    page,
    page_size: 20,
  });

  if (isLoading) return <div>Loading devices...</div>;
  if (isError) return <div>Error: {error.message}</div>;

  return (
    <div>
      <h2>Devices</h2>
      <ul>
        {data?.data.map((device) => (
          <li key={device.id}>
            {device.hostname} - {device.ip_address} ({device.status})
          </li>
        ))}
      </ul>
      <div>
        <button onClick={() => setPage(page - 1)} disabled={page === 1}>
          Previous
        </button>
        <span>
          Page {data?.meta.page} of {Math.ceil((data?.meta.total || 0) / (data?.meta.page_size || 1))}
        </span>
        <button
          onClick={() => setPage(page + 1)}
          disabled={page >= Math.ceil((data?.meta.total || 0) / (data?.meta.page_size || 1))}
        >
          Next
        </button>
      </div>
    </div>
  );
};

/**
 * Example 2: Fetching and displaying a single device
 */
export const DeviceDetailExample: React.FC<{ deviceId: string }> = ({
  deviceId,
}) => {
  const { data: device, isLoading, isError, error } = useDevice(deviceId);

  if (isLoading) return <div>Loading device...</div>;
  if (isError) return <div>Error: {error.message}</div>;
  if (!device) return <div>Device not found</div>;

  return (
    <div>
      <h2>Device Details</h2>
      <dl>
        <dt>Hostname:</dt>
        <dd>{device.hostname}</dd>
        <dt>IP Address:</dt>
        <dd>{device.ip_address}</dd>
        <dt>Type:</dt>
        <dd>{device.device_type}</dd>
        <dt>Status:</dt>
        <dd>{device.status}</dd>
        <dt>Location:</dt>
        <dd>{device.location || 'N/A'}</dd>
      </dl>
    </div>
  );
};

/**
 * Example 3: Creating a new device
 */
export const CreateDeviceExample: React.FC = () => {
  const createDevice = useCreateDevice();
  const [formData, setFormData] = useState<CreateDeviceRequest>({
    hostname: '',
    ip_address: '',
    device_type: 'ipmi',
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createDevice.mutate(formData, {
      onSuccess: () => {
        alert('Device created successfully!');
        // Reset form
        setFormData({
          hostname: '',
          ip_address: '',
          device_type: 'ipmi',
        });
      },
      onError: (error) => {
        alert(`Failed to create device: ${error.message}`);
      },
    });
  };

  return (
    <form onSubmit={handleSubmit}>
      <h2>Create Device</h2>
      <div>
        <label>
          Hostname:
          <input
            type="text"
            value={formData.hostname}
            onChange={(e) =>
              setFormData({ ...formData, hostname: e.target.value })
            }
            required
          />
        </label>
      </div>
      <div>
        <label>
          IP Address:
          <input
            type="text"
            value={formData.ip_address}
            onChange={(e) =>
              setFormData({ ...formData, ip_address: e.target.value })
            }
            required
          />
        </label>
      </div>
      <div>
        <label>
          Device Type:
          <select
            value={formData.device_type}
            onChange={(e) =>
              setFormData({
                ...formData,
                device_type: e.target.value as CreateDeviceRequest['device_type'],
              })
            }
          >
            <option value="ipmi">IPMI</option>
            <option value="redfish">Redfish</option>
            <option value="snmp">SNMP</option>
            <option value="linux_agent">Linux Agent</option>
            <option value="windows_agent">Windows Agent</option>
          </select>
        </label>
      </div>
      <button type="submit" disabled={createDevice.isPending}>
        {createDevice.isPending ? 'Creating...' : 'Create Device'}
      </button>
    </form>
  );
};

/**
 * Example 4: Updating a device
 */
export const UpdateDeviceExample: React.FC<{ deviceId: string }> = ({
  deviceId,
}) => {
  const { data: device } = useDevice(deviceId);
  const updateDevice = useUpdateDevice();
  const [hostname, setHostname] = useState('');

  React.useEffect(() => {
    if (device) {
      setHostname(device.hostname);
    }
  }, [device]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const updateData: UpdateDeviceRequest = { hostname };
    
    updateDevice.mutate(
      { id: deviceId, data: updateData },
      {
        onSuccess: () => {
          alert('Device updated successfully!');
        },
        onError: (error) => {
          alert(`Failed to update device: ${error.message}`);
        },
      }
    );
  };

  if (!device) return <div>Loading...</div>;

  return (
    <form onSubmit={handleSubmit}>
      <h2>Update Device</h2>
      <div>
        <label>
          Hostname:
          <input
            type="text"
            value={hostname}
            onChange={(e) => setHostname(e.target.value)}
            required
          />
        </label>
      </div>
      <button type="submit" disabled={updateDevice.isPending}>
        {updateDevice.isPending ? 'Updating...' : 'Update Device'}
      </button>
    </form>
  );
};

/**
 * Example 5: Deleting a device
 */
export const DeleteDeviceExample: React.FC<{ deviceId: string }> = ({
  deviceId,
}) => {
  const deleteDevice = useDeleteDevice();

  const handleDelete = () => {
    if (window.confirm('Are you sure you want to delete this device?')) {
      deleteDevice.mutate(deviceId, {
        onSuccess: () => {
          alert('Device deleted successfully!');
        },
        onError: (error) => {
          alert(`Failed to delete device: ${error.message}`);
        },
      });
    }
  };

  return (
    <button onClick={handleDelete} disabled={deleteDevice.isPending}>
      {deleteDevice.isPending ? 'Deleting...' : 'Delete Device'}
    </button>
  );
};

/**
 * Example 6: Filtering devices by type and status
 */
export const FilteredDeviceListExample: React.FC = () => {
  const [deviceType, setDeviceType] = useState<string>('');
  const [status, setStatus] = useState<string>('');

  const { data, isLoading } = useDevices({
    device_type: (deviceType || undefined) as import('../types/device').DeviceType | undefined,
    status: (status || undefined) as import('../types/device').DeviceStatus | undefined,
  });

  return (
    <div>
      <h2>Filtered Devices</h2>
      <div>
        <label>
          Device Type:
          <select value={deviceType} onChange={(e) => setDeviceType(e.target.value)}>
            <option value="">All</option>
            <option value="ipmi">IPMI</option>
            <option value="redfish">Redfish</option>
            <option value="snmp">SNMP</option>
          </select>
        </label>
        <label>
          Status:
          <select value={status} onChange={(e) => setStatus(e.target.value)}>
            <option value="">All</option>
            <option value="healthy">Healthy</option>
            <option value="warning">Warning</option>
            <option value="critical">Critical</option>
          </select>
        </label>
      </div>
      {isLoading ? (
        <div>Loading...</div>
      ) : (
        <ul>
          {data?.data.map((device) => (
            <li key={device.id}>
              {device.hostname} - {device.device_type} - {device.status}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};
