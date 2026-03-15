import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { AuthProvider, useAuth } from './AuthContext';
import * as authService from '../services/auth';

// Mock the auth service
vi.mock('../services/auth');

describe('AuthContext', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
  });

  it('should initialize with no user when no token exists', async () => {
    vi.mocked(authService.getCurrentUser).mockResolvedValue({
      id: '123',
      username: 'testuser',
      email: 'test@example.com',
      role: 'admin',
      enabled: true,
    });

    const { result } = renderHook(() => useAuth(), {
      wrapper: AuthProvider,
    });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
  });

  it('should fetch user on mount if token exists', async () => {
    localStorage.setItem('infrasense_token', 'test-token');
    
    const mockUser = {
      id: '123',
      username: 'testuser',
      email: 'test@example.com',
      role: 'admin' as const,
      enabled: true,
    };

    vi.mocked(authService.getCurrentUser).mockResolvedValue(mockUser);

    const { result } = renderHook(() => useAuth(), {
      wrapper: AuthProvider,
    });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(authService.getCurrentUser).toHaveBeenCalled();
    expect(result.current.user).toEqual(mockUser);
    expect(result.current.isAuthenticated).toBe(true);
  });

  it('should login user successfully', async () => {
    const mockLoginResponse = {
      token: 'new-token',
      user: {
        id: '123',
        username: 'testuser',
        email: 'test@example.com',
        role: 'admin' as const,
        enabled: true,
      },
    };

    vi.mocked(authService.login).mockResolvedValue(mockLoginResponse);

    const { result } = renderHook(() => useAuth(), {
      wrapper: AuthProvider,
    });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      await result.current.login({ username: 'testuser', password: 'password' });
    });

    expect(authService.login).toHaveBeenCalledWith({
      username: 'testuser',
      password: 'password',
    });
    expect(result.current.user).toEqual(mockLoginResponse.user);
    expect(result.current.isAuthenticated).toBe(true);
  });

  it('should logout user successfully', async () => {
    localStorage.setItem('infrasense_token', 'test-token');
    
    const mockUser = {
      id: '123',
      username: 'testuser',
      email: 'test@example.com',
      role: 'admin' as const,
      enabled: true,
    };

    vi.mocked(authService.getCurrentUser).mockResolvedValue(mockUser);
    vi.mocked(authService.logout).mockResolvedValue();

    const { result } = renderHook(() => useAuth(), {
      wrapper: AuthProvider,
    });

    await waitFor(() => {
      expect(result.current.user).toEqual(mockUser);
    });

    await act(async () => {
      await result.current.logout();
    });

    expect(authService.logout).toHaveBeenCalled();
    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
  });

  it('should throw error when useAuth is used outside AuthProvider', () => {
    expect(() => {
      renderHook(() => useAuth());
    }).toThrow('useAuth must be used within an AuthProvider');
  });
});
