import { describe, it, expect, beforeEach, vi } from 'vitest';
import { login, logout, getCurrentUser, getToken, isAuthenticated } from './auth';
import apiClient from './api';

// Mock the API client
vi.mock('./api');

const TOKEN_KEY = 'infrasense_token';

describe('Authentication Service', () => {
  beforeEach(() => {
    // Clear localStorage before each test
    localStorage.clear();
    vi.clearAllMocks();
  });

  describe('login', () => {
    it('should store token in localStorage on successful login', async () => {
      const mockResponse = {
        data: {
          token: 'test-jwt-token',
          user: {
            id: '123',
            username: 'testuser',
            email: 'test@example.com',
            role: 'admin' as const,
            enabled: true,
          },
        },
      };

      vi.mocked(apiClient.post).mockResolvedValue(mockResponse);

      const result = await login({ username: 'testuser', password: 'password' });

      expect(apiClient.post).toHaveBeenCalledWith('/auth/login', {
        username: 'testuser',
        password: 'password',
      });
      expect(localStorage.getItem(TOKEN_KEY)).toBe('test-jwt-token');
      expect(result).toEqual(mockResponse.data);
    });

    it('should throw error on failed login', async () => {
      vi.mocked(apiClient.post).mockRejectedValue(new Error('Invalid credentials'));

      await expect(login({ username: 'testuser', password: 'wrong' })).rejects.toThrow(
        'Invalid credentials'
      );
      expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    });
  });

  describe('logout', () => {
    it('should clear token from localStorage on logout', async () => {
      localStorage.setItem(TOKEN_KEY, 'test-token');
      vi.mocked(apiClient.post).mockResolvedValue({ data: {} });

      await logout();

      expect(apiClient.post).toHaveBeenCalledWith('/auth/logout');
      expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    });

    it('should clear token even if API call fails', async () => {
      localStorage.setItem(TOKEN_KEY, 'test-token');
      vi.mocked(apiClient.post).mockRejectedValue(new Error('Network error'));

      await logout();

      expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    });
  });

  describe('getCurrentUser', () => {
    it('should return current user from API', async () => {
      const mockUser = {
        id: '123',
        username: 'testuser',
        email: 'test@example.com',
        role: 'admin' as const,
        enabled: true,
      };

      vi.mocked(apiClient.get).mockResolvedValue({ data: mockUser });

      const result = await getCurrentUser();

      expect(apiClient.get).toHaveBeenCalledWith('/auth/me');
      expect(result).toEqual(mockUser);
    });

    it('should throw error if not authenticated', async () => {
      vi.mocked(apiClient.get).mockRejectedValue(new Error('Unauthorized'));

      await expect(getCurrentUser()).rejects.toThrow('Unauthorized');
    });
  });

  describe('getToken', () => {
    it('should return token from localStorage', () => {
      localStorage.setItem(TOKEN_KEY, 'test-token');
      expect(getToken()).toBe('test-token');
    });

    it('should return null if no token exists', () => {
      expect(getToken()).toBeNull();
    });
  });

  describe('isAuthenticated', () => {
    it('should return true if token exists', () => {
      localStorage.setItem(TOKEN_KEY, 'test-token');
      expect(isAuthenticated()).toBe(true);
    });

    it('should return false if no token exists', () => {
      expect(isAuthenticated()).toBe(false);
    });
  });
});
