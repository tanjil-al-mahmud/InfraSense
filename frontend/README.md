# InfraSense Frontend - Authentication Implementation

## Task 5.3: Authentication Service and Context

This implementation provides authentication functionality for the InfraSense Platform frontend.

## Files Created

### 1. `src/services/auth.ts`
Authentication service with the following functions:
- `login(credentials)` - POST /api/v1/auth/login
- `logout()` - POST /api/v1/auth/logout
- `getCurrentUser()` - GET /api/v1/auth/me
- `getToken()` - Retrieve JWT token from localStorage
- `isAuthenticated()` - Check if user has valid token

**JWT Token Storage**: Tokens are stored in localStorage with key `infrasense_token`

### 2. `src/contexts/AuthContext.tsx`
React context for managing authentication state:
- Provides `user`, `loading`, `isAuthenticated` state
- Provides `login()`, `logout()`, `refreshUser()` functions
- Automatically fetches user on mount if token exists
- Exports `AuthProvider` component and `useAuth()` hook

### 3. Test Files
- `src/services/auth.test.ts` - Unit tests for auth service
- `src/contexts/AuthContext.test.tsx` - Tests for AuthContext

## Usage

### 1. Wrap your app with AuthProvider

```tsx
import { AuthProvider } from './contexts/AuthContext';

function App() {
  return (
    <AuthProvider>
      {/* Your app components */}
    </AuthProvider>
  );
}
```

### 2. Use the useAuth hook in components

```tsx
import { useAuth } from './contexts/AuthContext';

function LoginPage() {
  const { login, loading } = useAuth();

  const handleLogin = async () => {
    try {
      await login({ username: 'admin', password: 'password' });
      // Redirect to dashboard
    } catch (error) {
      // Handle error
    }
  };

  return (
    <button onClick={handleLogin} disabled={loading}>
      Login
    </button>
  );
}
```

### 3. Access user information

```tsx
function Dashboard() {
  const { user, isAuthenticated, logout } = useAuth();

  if (!isAuthenticated) {
    return <Navigate to="/login" />;
  }

  return (
    <div>
      <h1>Welcome, {user?.username}</h1>
      <p>Role: {user?.role}</p>
      <button onClick={logout}>Logout</button>
    </div>
  );
}
```

## Installation

```bash
cd infrasense/frontend
npm install
```

## Running Tests

```bash
# Run tests once
npm test

# Run tests in watch mode
npm run test:watch
```

## API Integration

The authentication service integrates with the backend API:

- **Login**: `POST /api/v1/auth/login`
  - Request: `{ username: string, password: string }`
  - Response: `{ token: string, user: User }`

- **Logout**: `POST /api/v1/auth/logout`
  - No request body
  - Clears token from localStorage

- **Get Current User**: `GET /api/v1/auth/me`
  - Requires JWT token in Authorization header
  - Response: `User` object

## Token Management

- JWT tokens are stored in localStorage with key `infrasense_token`
- Tokens have 24-hour expiration (as per Requirement 16.3)
- The API client automatically includes the token in Authorization header
- On 401 responses, the token is cleared and user is redirected to login

## Requirements Satisfied

- ✅ Requirement 16.1: Username/password authentication
- ✅ Requirement 16.3: JWT token with 24-hour expiration
- ✅ Task 5.3: All acceptance criteria met
  - Created auth.ts service
  - Implemented login, logout, getCurrentUser functions
  - JWT token stored in localStorage
  - React context for authentication state
