import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api.js'

export default function Dashboard() {
  const [polls, setPolls] = useState([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.myPolls()
      .then(setPolls)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  return (
    <div className="container">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2>Your polls</h2>
        <Link to="/create"><button className="btn-primary">+ New poll</button></Link>
      </div>

      <div className="card" style={{ marginTop: 20 }}>
        {loading && <p className="muted">Loading…</p>}
        {error && <p className="error">{error}</p>}
        {!loading && polls.length === 0 && (
          <p className="muted">Nothing here yet — create your first poll to get started.</p>
        )}
        {polls.map((p) => (
          <div className="poll-list-item" key={p.id}>
            <div>
              <strong>{p.question}</strong>
              <div className="muted">{p.options.length} options</div>
            </div>
            <Link to={`/polls/${p.id}`}><button className="btn-secondary">Open</button></Link>
          </div>
        ))}
      </div>
    </div>
  )
}
