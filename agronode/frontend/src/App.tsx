import { useEffect, useState } from 'react'

import { loadSession, onSessionChange } from './api/session'
import { DashboardPage } from './pages/DashboardPage'
import { LoginPage } from './pages/LoginPage'
import './App.css'

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(() => loadSession() !== null)

  useEffect(() => {
    return onSessionChange(() => {
      setIsAuthenticated(loadSession() !== null)
    })
  }, [])

  if (!isAuthenticated) {
    return <LoginPage onSuccess={() => setIsAuthenticated(true)} />
  }

  return <DashboardPage />
}

export default App
