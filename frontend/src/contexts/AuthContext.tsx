
import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';

import { login as loginService, logout as logoutService, getCurrentUser, User, LoginCredentials } from '../services/auth';



/**

 * Authentication Context

 * 

 * Provides authentication state and operations throughout the application:

 * - Current user information

 * - Loading state during authentication checks

 * - Login/logout functions

 * - Authentication status

 */



interface AuthContextType {

  user: User | null;

  loading: boolean;

  isAuthenticated: boolean;

  login: (credentials: LoginCredentials) => Promise<void>;

  logout: () => Promise<void>;

  refreshUser: () => Promise<void>;

}



const AuthContext = createContext<AuthContextType | undefined>(undefined);



interface AuthProviderProps {

  children: ReactNode;

}



/**

 * Authentication Provider Component

 * 

 * Wraps the application to provide authentication state and operations.

 * Automatically fetches current user on mount if token exists.

 */

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {

  const [user, setUser] = useState<User | null>(null);

  const [loading, setLoading] = useState<boolean>(true);



  /**

   * Fetch current user from API

   * Called on mount and after login

   */

  const fetchCurrentUser = async () => {

    try {

      const currentUser = await getCurrentUser();

      setUser(currentUser);

    } catch (error) {

      // If fetching user fails, clear user state

      setUser(null);

    } finally {

      setLoading(false);

    }

  };



  /**

   * Initialize authentication state on mount

   * Checks if token exists and fetches user info

   */

  useEffect(() => {

    const token = localStorage.getItem('infrasense_token');

    

    if (token) {

      // Token exists, fetch current user

      fetchCurrentUser();

    } else {

      // No token, set loading to false

      setLoading(false);

    }

  }, []);



  /**

   * Login user with credentials

   * Stores token and fetches user info on success

   */

  const login = async (credentials: LoginCredentials): Promise<void> => {

    setLoading(true);

    try {

      const response = await loginService(credentials);

      setUser(response.user);

    } finally {

      setLoading(false);

    }

  };



  /**

   * Logout current user

   * Clears token and user state

   */

  const logout = async (): Promise<void> => {

    setLoading(true);

    try {

      await logoutService();

    } finally {

      setUser(null);

      setLoading(false);

    }

  };



  /**

   * Refresh current user information

   * Useful after user profile updates

   */

  const refreshUser = async (): Promise<void> => {

    setLoading(true);

    await fetchCurrentUser();

  };



  const value: AuthContextType = {

    user,

    loading,

    isAuthenticated: user !== null,

    login,

    logout,

    refreshUser,

  };



  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;

};



/**

 * Hook to access authentication context

 * Must be used within AuthProvider

 * 

 * @returns Authentication context

 * @throws Error if used outside AuthProvider

 */

export const useAuth = (): AuthContextType => {

  const context = useContext(AuthContext);

  

  if (context === undefined) {

    throw new Error('useAuth must be used within an AuthProvider');

  }

  

  return context;

};
