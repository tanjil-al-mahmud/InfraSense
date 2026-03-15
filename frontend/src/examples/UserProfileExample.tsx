import React from 'react';
import { useAuth } from '../contexts/AuthContext';

/**
 * Example User Profile Component
 * 
 * Demonstrates how to access user information and logout functionality.
 */
export const UserProfileExample: React.FC = () => {
  const { user, logout, loading } = useAuth();

  const handleLogout = async () => {
    try {
      await logout();
      // Redirect to login page
      window.location.href = '/login';
    } catch (err) {
      console.error('Logout failed:', err);
    }
  };

  if (!user) {
    return <div>Not logged in</div>;
  }

  return (
    <div>
      <h2>User Profile</h2>
      <div>
        <p><strong>Username:</strong> {user.username}</p>
        <p><strong>Email:</strong> {user.email || 'N/A'}</p>
        <p><strong>Role:</strong> {user.role}</p>
        <p><strong>Status:</strong> {user.enabled ? 'Active' : 'Disabled'}</p>
      </div>
      <button onClick={handleLogout} disabled={loading}>
        {loading ? 'Logging out...' : 'Logout'}
      </button>
    </div>
  );
};
