import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, withColdStartHint } from '../api.js'

export default function Dashboard() {
  const [polls, setPolls] = useState([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [waking, setWaking] = useState(false)
  const [closingId, setClosingId] = useState(null)

  const loadPolls = () => {
    setLoading(true)
    withColdStartHint(api.myPolls, setWaking)()
      .then(setPolls)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    loadPolls()
  }, [])

  const handleClose = async (id) => {
    setClosingId(id)
    try {
      await api.closePoll(id)
      setPolls((prev) => prev.map((p) => (p.id === id ? { ...p, is_closed: true } : p)))
    } catch (err) {
      setError(err.message)
    } finally {
      setClosingId(null)
    }
  }

  return (
    <div className="container">
      <div className="dashboard-banner">
        <div>
          <h2>Your polls</h2>
          <p className="muted" style={{ margin: '4px 0 0' }}>Every poll you've created, in one place.</p>
        </div>
        <Link to="/create"><button className="btn-primary">+ New poll</button></Link>
      </div>

      <div className="card">
        {loading && (
          <p className="muted">
            Loading…{waking && ' the server is waking up, this can take up to a minute on the free tier.'}
          </p>
        )}
        {error && <p className="error">{error}</p>}
        {!loading && polls.length === 0 && (
          <p className="muted">Nothing here yet — create your first poll to get started.</p>
        )}
        {polls.map((p) => (
          <div className="poll-list-item" key={p.id}>
            <div>
              <strong>{p.question}</strong>
              <div className="muted">
                {p.options.length} options · {p.total_votes} vote{p.total_votes === 1 ? '' : 's'}
                {p.is_closed && ' · closed'}
              </div>
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              {!p.is_closed && (
                <button
                  className="btn-secondary"
                  onClick={() => handleClose(p.id)}
                  disabled={closingId === p.id}
                >
                  {closingId === p.id ? 'Closing…' : 'Close'}
                </button>
              )}
              <Link to={`/polls/${p.id}`}><button className="btn-secondary">Open</button></Link>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
