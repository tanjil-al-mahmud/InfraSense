import axios, { AxiosInstance, AxiosError, InternalAxiosRequestConfig } from 'axios';

/**
 * API Client Service
 * 
 * Configured Axios instance for communicating with the InfraSense backend API.
 * - Base URL: /api/v1/
 * - JWT token authentication via Authorization header
 * - Automatic 401 error handling (redirect to login)
 * - Network error handling
 */

const API_BASE_URL = '/api/v1/';
const TOKEN_KEY = 'infrasense_token';

// Create Axios instance with base configuration
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 30000, // 30 second timeout
});

/**
 * Request Interceptor
 * Adds JWT token to Authorization header if available
 */
apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem(TOKEN_KEY);
    
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    
    return config;
  },
  (error: AxiosError) => {
    return Promise.reject(error);
  }
);

/**
 * Response Interceptor
 * Handles 401 errors by redirecting to login page
 * Handles network failures with appropriate error messages
 */
apiClient.interceptors.response.use(
  (response) => {
    return response;
  },
  (error: AxiosError) => {
    // Handle 401 Unauthorized - redirect to login
    if (error.response?.status === 401) {
      // Clear token from localStorage
      localStorage.removeItem(TOKEN_KEY);
      
      // Redirect to login page
      window.location.href = '/login';
      
      return Promise.reject(new Error('Session expired. Please login again.'));
    }
    
    // Handle network failures
    if (!error.response) {
      // Network error (no response received)
      const networkError = new Error(
        'Network error: Unable to connect to the server. Please check your connection.'
      );
      return Promise.reject(networkError);
    }
    
    // Handle other HTTP errors
    const responseData = error.response?.data as { error?: string } | undefined;
    const errorMessage = responseData?.error || error.message || 'An unexpected error occurred';
    return Promise.reject(new Error(errorMessage));
  }
);

export default apiClient;
