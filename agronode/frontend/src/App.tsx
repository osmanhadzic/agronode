import { useEffect, useState } from 'react'
import { Navigate, Outlet, Route, Routes } from 'react-router-dom'

import { clearSession, loadSession, onSessionChange } from './api/session'
import { AdminDashboardPage } from './pages/AdminDashboardPage'
import { LoginPage } from './pages/LoginPage'
import { OrganizationDashboardPage } from './pages/OrganizationDashboardPage'
import './App.css'

function isAllowedRole(role: string): role is 'admin' | 'organization' {
  return role === 'admin' || role === 'organization'
}

function getRoleHomePath(role: 'admin' | 'organization') {
  return role === 'admin' ? '/admin' : '/org'
}

type ProtectedRouteProps = {
  role: string | null
  allowedRoles: Array<'admin' | 'organization'>
}

function ProtectedRoute({ role, allowedRoles }: ProtectedRouteProps) {
  if (!role) {
    return <Navigate to="/login" replace />
  }

  if (!allowedRoles.includes(role as 'admin' | 'organization')) {
    return <Navigate to={getRoleHomePath(role as 'admin' | 'organization')} replace />
  }

  return <Outlet />
}

function App() {
  const [session, setSession] = useState(() => loadSession())

  useEffect(() => {
    return onSessionChange(() => {
      setSession(loadSession())
    })
  }, [])

  const role = session?.role ?? null

  if (role && !isAllowedRole(role)) {
    clearSession()
    return <Navigate to="/login" replace />
  }

  const defaultAuthenticatedPath = role ? getRoleHomePath(role) : '/login'

  return (
    <Routes>
      <Route path="/login" element={role ? <Navigate to={defaultAuthenticatedPath} replace /> : <LoginPage />} />

      <Route element={<ProtectedRoute role={role} allowedRoles={['admin']} />}>
        <Route path="/admin" element={<AdminDashboardPage />} />
      </Route>

      <Route element={<ProtectedRoute role={role} allowedRoles={['organization']} />}>
        <Route path="/org" element={<OrganizationDashboardPage />} />
      </Route>

      <Route path="/" element={<Navigate to={defaultAuthenticatedPath} replace />} />
      <Route path="*" element={<Navigate to={defaultAuthenticatedPath} replace />} />
    </Routes>
  )
}

export default App
