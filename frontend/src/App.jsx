import { Routes, Route, Link, Navigate, useNavigate } from 'react-router-dom'
import { useState } from 'react'
import Login from './pages/Login.jsx'
import Signup from './pages/Signup.jsx'
import Dashboard from './pages/Dashboard.jsx'
import CreatePoll from './pages/CreatePoll.jsx'
import PollView from './pages/PollView.jsx'

function useAuth() {
  const [token, setToken] = useState(localStorage.getItem('token'))
  const login = (t) => {
    localStorage.setItem('token', t)
    setToken(t)
  }
  const logout = () => {
    localStorage.removeItem('token')
    setToken(null)
  }
  return { token, login, logout }
}

function RequireAuth({ token, children }) {
  if (!token) return <Navigate to="/login" replace />
  return children
}

export default function App() {
  const auth = useAuth()
  const navigate = useNavigate()

  const handleLogout = () => {
    auth.logout()
    navigate('/login')
  }

  return (
    <>
      <nav className="navbar">
        <Link to="/" className="brand">Live<span>Polls</span></Link>
        <div style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
          {auth.token ? (
            <>
              <Link to="/dashboard">Dashboard</Link>
              <button className="btn-secondary" onClick={handleLogout}>Log out</button>
            </>
          ) : (
            <>
              <Link to="/login">Log in</Link>
              <Link to="/signup">Sign up</Link>
            </>
          )}
        </div>
      </nav>

      <Routes>
        <Route path="/" element={<Navigate to={auth.token ? '/dashboard' : '/login'} replace />} />
        <Route path="/login" element={<Login onLogin={auth.login} />} />
        <Route path="/signup" element={<Signup onLogin={auth.login} />} />
        <Route
          path="/dashboard"
          element={<RequireAuth token={auth.token}><Dashboard /></RequireAuth>}
        />
        <Route
          path="/create"
          element={<RequireAuth token={auth.token}><CreatePoll /></RequireAuth>}
        />
        <Route path="/polls/:id" element={<PollView />} />
      </Routes>
    </>
  )
}
