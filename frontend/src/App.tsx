import React, { useState } from 'react';
import {
  BrowserRouter,
  Routes,
  Route,
  Navigate,
  NavLink,
  Outlet,
  useNavigate,
} from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { ToastProvider } from './contexts/ToastContext';

// Pages
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Devices from './pages/Devices';
import DeviceDetail from './pages/DeviceDetail';
import Alerts from './pages/Alerts';
import AlertRules from './pages/AlertRules';

/**
 * App Component
 *
 * Root application component providing:
 * - React Query client
 * - Authentication context
 * - React Router v6 routing
 * - Protected routes (require authentication)
 * - Public routes (login page)
 * - Responsive layout with Tailwind CSS (hamburger menu on mobile)
 *
 * Routes:
 *   /login          → Login (public)
 *   /               → Dashboard (protected)
 *   /devices        → Devices (protected)
 *   /devices/:id    → DeviceDetail (protected)
 *   /alerts         → Alerts (protected)
 *   /alert-rules    → AlertRules (protected)
 *
 * Requirements: 20.1, 20.7
 */

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 30_000,
    },
  },
});

// ── Protected Route ──────────────────────────────────────────────────────────

const ProtectedRoute: React.FC = () => {
  const { isAuthenticated, loading } = useAuth();

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen gap-4" style={{ background: '#020617' }}>
        <div
          className="w-8 h-8 border-4 rounded-full"
          style={{ borderColor: '#1e293b', borderTopColor: '#3b82f6', animation: 'spin 0.8s linear infinite' }}
          aria-hidden="true"
        />
        <p className="text-sm" style={{ color: '#64748b' }}>Loading...</p>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return (
    <div className="flex flex-col min-h-screen" style={{ background: '#020617' }}>
      <NavBar />
      <main className="flex-1 overflow-y-auto">
        <Outlet />
      </main>
    </div>
  );
};

// ── Navigation Bar ───────────────────────────────────────────────────────────

const NAV_LINKS = [
  { to: '/', label: 'Dashboard', end: true },
  { to: '/devices', label: 'Devices', end: false },
  { to: '/alerts', label: 'Alerts', end: false },
  { to: '/alert-rules', label: 'Alert Rules', end: false },
] as const;

/** External link rendered as a plain anchor (opens in new tab). */
const EXTERNAL_NAV_LINKS = [
  { href: '/grafana', label: 'Grafana' },
] as const;

/**
 * Responsive navigation bar.
 * - Desktop (md+): horizontal nav links + user info inline
 * - Mobile (<md): hamburger button toggles a vertical dropdown menu
 */
const NavBar: React.FC = () => {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);

  const handleLogout = async () => {
    setMenuOpen(false);
    await logout();
    navigate('/login', { replace: true });
  };

  const toggleMenu = () => setMenuOpen((prev) => !prev);
  const closeMenu = () => setMenuOpen(false);

  return (
    <nav
      className="sticky top-0 z-50 flex-shrink-0"
      style={{ background: '#0f172a', borderBottom: '1px solid #1e293b' }}
      role="navigation"
      aria-label="Main navigation"
    >
      {/* ── Top bar ── */}
      <div className="flex items-center justify-between h-14 px-4 sm:px-6">
        {/* Brand */}
        <span className="text-white font-bold text-lg tracking-tight select-none">
          InfraSense
        </span>

        {/* Desktop nav links (hidden on mobile) */}
        <ul className="hidden md:flex items-center gap-1 flex-1 ml-6" role="list">
          {NAV_LINKS.map(({ to, label, end }) => (
            <li key={to}>
              <NavLink
                to={to}
                end={end}
                className={({ isActive }) =>
                  `px-3 py-1.5 rounded text-sm font-medium transition-colors duration-150 ${
                    isActive
                      ? 'bg-slate-700 text-white'
                      : 'text-slate-400 hover:text-white hover:bg-slate-700'
                  }`
                }
              >
                {label}
              </NavLink>
            </li>
          ))}
          {EXTERNAL_NAV_LINKS.map(({ href, label }) => (
            <li key={href}>
              <a
                href={href}
                target="_blank"
                rel="noopener noreferrer"
                className="px-3 py-1.5 rounded text-sm font-medium transition-colors duration-150 text-slate-400 hover:text-white hover:bg-slate-700 inline-flex items-center gap-1"
              >
                {label}
                <svg className="w-3 h-3 opacity-60" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                </svg>
              </a>
            </li>
          ))}
        </ul>

        {/* Desktop user info + logout (hidden on mobile) */}
        {user && (
          <div className="hidden md:flex items-center gap-3 flex-shrink-0">
            <div className="flex flex-col items-end gap-0.5">
              <span className="text-white text-sm font-medium leading-none">
                {user.username}
              </span>
              <span className="bg-slate-700 text-slate-400 text-xs font-semibold px-1.5 py-0.5 rounded-full capitalize">
                {user.role}
              </span>
            </div>
            <button
              onClick={handleLogout}
              className="border border-slate-500 text-slate-400 hover:text-white hover:border-slate-300 text-xs font-medium px-3 py-1.5 rounded transition-colors duration-150"
              aria-label="Sign out"
            >
              Sign Out
            </button>
          </div>
        )}

        {/* Hamburger button (visible on mobile only) */}
        <button
          className="md:hidden flex flex-col justify-center items-center w-9 h-9 gap-1.5 rounded focus:outline-none focus:ring-2 focus:ring-slate-400"
          onClick={toggleMenu}
          aria-label={menuOpen ? 'Close menu' : 'Open menu'}
          aria-expanded={menuOpen}
          aria-controls="mobile-menu"
        >
          {/* Three bars — animate to X when open */}
          <span
            className={`block w-5 h-0.5 bg-slate-300 transition-transform duration-200 origin-center ${
              menuOpen ? 'translate-y-2 rotate-45' : ''
            }`}
          />
          <span
            className={`block w-5 h-0.5 bg-slate-300 transition-opacity duration-200 ${
              menuOpen ? 'opacity-0' : ''
            }`}
          />
          <span
            className={`block w-5 h-0.5 bg-slate-300 transition-transform duration-200 origin-center ${
              menuOpen ? '-translate-y-2 -rotate-45' : ''
            }`}
          />
        </button>
      </div>

      {/* ── Mobile dropdown menu ── */}
      {menuOpen && (
        <div
          id="mobile-menu"
          className="md:hidden bg-slate-800 border-t border-slate-700 px-4 pb-4"
        >
          <ul className="flex flex-col gap-1 mt-2" role="list">
            {NAV_LINKS.map(({ to, label, end }) => (
              <li key={to}>
                <NavLink
                  to={to}
                  end={end}
                  onClick={closeMenu}
                  className={({ isActive }) =>
                    `block px-3 py-2 rounded text-sm font-medium transition-colors duration-150 ${
                      isActive
                        ? 'bg-slate-700 text-white'
                        : 'text-slate-400 hover:text-white hover:bg-slate-700'
                    }`
                  }
                >
                  {label}
                </NavLink>
              </li>
            ))}
            {EXTERNAL_NAV_LINKS.map(({ href, label }) => (
              <li key={href}>
                <a
                  href={href}
                  target="_blank"
                  rel="noopener noreferrer"
                  onClick={closeMenu}
                  className="flex px-3 py-2 rounded text-sm font-medium transition-colors duration-150 text-slate-400 hover:text-white hover:bg-slate-700 items-center gap-1"
                >
                  {label}
                  <svg className="w-3 h-3 opacity-60" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                  </svg>
                </a>
              </li>
            ))}
          </ul>

          {/* Mobile user info + logout */}
          {user && (
            <div className="mt-3 pt-3 border-t border-slate-700 flex items-center justify-between">
              <div className="flex flex-col gap-0.5">
                <span className="text-white text-sm font-medium">{user.username}</span>
                <span className="text-slate-400 text-xs capitalize">{user.role}</span>
              </div>
              <button
                onClick={handleLogout}
                className="border border-slate-500 text-slate-400 hover:text-white hover:border-slate-300 text-xs font-medium px-3 py-1.5 rounded transition-colors duration-150"
                aria-label="Sign out"
              >
                Sign Out
              </button>
            </div>
          )}
        </div>
      )}
    </nav>
  );
};

// ── Router ───────────────────────────────────────────────────────────────────

const AppRoutes: React.FC = () => (
  <Routes>
    {/* Public route */}
    <Route path="/login" element={<Login />} />

    {/* Protected routes — wrapped in ProtectedRoute layout */}
    <Route element={<ProtectedRoute />}>
      <Route path="/" element={<Dashboard />} />
      <Route path="/devices" element={<Devices />} />
      <Route path="/devices/:id" element={<DeviceDetail />} />
      <Route path="/alerts" element={<Alerts />} />
      <Route path="/alert-rules" element={<AlertRules />} />
    </Route>

    {/* Catch-all: redirect unknown paths to dashboard */}
    <Route path="*" element={<Navigate to="/" replace />} />
  </Routes>
);

// ── Root App ─────────────────────────────────────────────────────────────────

const App: React.FC = () => (
  <QueryClientProvider client={queryClient}>
    <BrowserRouter>
      <AuthProvider>
        <ToastProvider>
          <AppRoutes />
        </ToastProvider>
      </AuthProvider>
    </BrowserRouter>
  </QueryClientProvider>
);

export default App;
