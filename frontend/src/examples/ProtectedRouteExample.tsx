import React from 'react';
import { Navigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

/**
 * Example Protected Route Component
 * 
 * Demonstrates how to protect routes that require authentication.
 * Redirects to login page if user is not authenticated.
 */
interface ProtectedRouteProps {
  children: React.ReactNode;
}

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ children }) => {
  const { isAuthenticated, loading } = useAuth();

  // Show loading state while checking authentication
  if (loading) {
    return <div>Loading...</div>;
  }

  // Redirect to login if not authenticated
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  // Render protected content
  return <>{children}</>;
};

/**
 * Example usage with React Router:
 * 
 * <Routes>
 *   <Route path="/login" element={<LoginPage />} />
 *   <Route
 *     path="/dashboard"
 *     element={
 *       <ProtectedRoute>
 *         <Dashboard />
 *       </ProtectedRoute>
 *     }
 *   />
 * </Routes>
 */
