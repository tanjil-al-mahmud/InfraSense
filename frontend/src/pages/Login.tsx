import React, { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { useAuth } from '../contexts/AuthContext';

/**
 * Login Page Component
 *
 * Provides a login form with username and password fields.
 * - Uses React Hook Form for form state and validation
 * - Displays validation errors for required fields
 * - Displays API error messages on authentication failure
 * - Redirects to /dashboard on successful login
 *
 * Requirements: 16.1
 */

interface LoginFormValues {
  username: string;
  password: string;
}

const Login: React.FC = () => {
  const { login, isAuthenticated, loading } = useAuth();
  const navigate = useNavigate();

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setError,
  } = useForm<LoginFormValues>({
    defaultValues: { username: '', password: '' },
  });

  // Redirect to dashboard if already authenticated
  useEffect(() => {
    if (!loading && isAuthenticated) {
      navigate('/', { replace: true });
    }
  }, [isAuthenticated, loading, navigate]);

  const onSubmit = async (data: LoginFormValues) => {
    try {
      await login(data);
      navigate('/', { replace: true });
    } catch (err) {
      const message =
        err instanceof Error ? err.message : 'Invalid username or password';
      setError('root', { message });
    }
  };

  if (loading) {
    return (
      <div style={styles.loadingContainer}>
        <p>Loading...</p>
      </div>
    );
  }

  return (
    <div style={styles.page}>
      <div style={styles.card}>
        {/* Header */}
        <div style={styles.header}>
          <h1 style={styles.title}>InfraSense</h1>
          <p style={styles.subtitle}>Infrastructure Hardware Monitoring</p>
        </div>

        {/* Login Form */}
        <form onSubmit={handleSubmit(onSubmit)} noValidate style={styles.form}>
          {/* Root / API error */}
          {errors.root && (
            <div style={styles.errorBanner} role="alert">
              {errors.root.message}
            </div>
          )}

          {/* Username field */}
          <div style={styles.fieldGroup}>
            <label htmlFor="username" style={styles.label}>
              Username
            </label>
            <input
              id="username"
              type="text"
              autoComplete="username"
              disabled={isSubmitting}
              style={{
                ...styles.input,
                ...(errors.username ? styles.inputError : {}),
              }}
              {...register('username', { required: 'Username is required' })}
            />
            {errors.username && (
              <p style={styles.fieldError} role="alert">
                {errors.username.message}
              </p>
            )}
          </div>

          {/* Password field */}
          <div style={styles.fieldGroup}>
            <label htmlFor="password" style={styles.label}>
              Password
            </label>
            <input
              id="password"
              type="password"
              autoComplete="current-password"
              disabled={isSubmitting}
              style={{
                ...styles.input,
                ...(errors.password ? styles.inputError : {}),
              }}
              {...register('password', { required: 'Password is required' })}
            />
            {errors.password && (
              <p style={styles.fieldError} role="alert">
                {errors.password.message}
              </p>
            )}
          </div>

          {/* Submit button */}
          <button
            type="submit"
            disabled={isSubmitting}
            style={{
              ...styles.button,
              ...(isSubmitting ? styles.buttonDisabled : {}),
            }}
          >
            {isSubmitting ? 'Signing in...' : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  );
};

const styles: Record<string, React.CSSProperties> = {
  page: {
    minHeight: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#f3f4f6',
    padding: '1rem',
  },
  loadingContainer: {
    minHeight: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  card: {
    backgroundColor: '#ffffff',
    borderRadius: '8px',
    boxShadow: '0 4px 6px rgba(0,0,0,0.1)',
    padding: '2rem',
    width: '100%',
    maxWidth: '400px',
  },
  header: {
    textAlign: 'center',
    marginBottom: '1.5rem',
  },
  title: {
    fontSize: '1.75rem',
    fontWeight: 700,
    color: '#111827',
    margin: '0 0 0.25rem',
  },
  subtitle: {
    fontSize: '0.875rem',
    color: '#6b7280',
    margin: 0,
  },
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
  input: {
    border: '1px solid #d1d5db',
    borderRadius: '4px',
    fontSize: '0.875rem',
    padding: '0.5rem 0.75rem',
    outline: 'none',
    width: '100%',
    boxSizing: 'border-box',
  },
  inputError: {
    borderColor: '#ef4444',
  },
  fieldError: {
    color: '#ef4444',
    fontSize: '0.75rem',
    margin: 0,
  },
  button: {
    backgroundColor: '#2563eb',
    border: 'none',
    borderRadius: '4px',
    color: '#ffffff',
    cursor: 'pointer',
    fontSize: '0.875rem',
    fontWeight: 600,
    marginTop: '0.5rem',
    padding: '0.625rem 1rem',
    width: '100%',
  },
  buttonDisabled: {
    backgroundColor: '#93c5fd',
    cursor: 'not-allowed',
  },
};

export default Login;
