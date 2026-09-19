import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, pollSocketUrl, withColdStartHint } from '../api.js'

export default function Dashboard() {
  const [polls, setPolls] = useState([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [waking, setWaking] = useState(false)
  const [statusChangingId, setStatusChangingId] = useState(null)
  const [deletingId, setDeletingId] = useState(null)
  const socketsRef = useRef({}) // pollId -> WebSocket

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

  // One WebSocket per poll shown on the dashboard, so total_votes stays live
  // without refreshing. Sockets are opened/closed as the poll list changes,
  // and all closed on unmount.
  useEffect(() => {
    const currentIds = new Set(polls.map((p) => p.id))

    // Open a socket for any poll that doesn't have one yet.
    polls.forEach((p) => {
      if (socketsRef.current[p.id]) return
      const ws = new WebSocket(pollSocketUrl(p.id))
      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          setPolls((prev) =>
            prev.map((poll) =>
              poll.id === p.id ? { ...poll, total_votes: data.total } : poll
            )
          )
        } catch (e) {
          // ignore malformed frames
        }
      }
      socketsRef.current[p.id] = ws
    })

    // Close sockets for polls no longer in the list (e.g. after delete).
    Object.keys(socketsRef.current).forEach((id) => {
      if (!currentIds.has(id)) {
        socketsRef.current[id].close()
        delete socketsRef.current[id]
      }
    })

    return () => {
      // Full cleanup on unmount.
      Object.values(socketsRef.current).forEach((ws) => ws.close())
      socketsRef.current = {}
    }
  }, [polls.map((p) => p.id).join(',')])

  const handleToggleStatus = async (id, currentlyClosed) => {
    setStatusChangingId(id)
    try {
      await api.setPollStatus(id, !currentlyClosed)
      setPolls((prev) =>
        prev.map((p) => (p.id === id ? { ...p, is_closed: !currentlyClosed } : p))
      )
    } catch (err) {
      setError(err.message)
    } finally {
      setStatusChangingId(null)
    }
  }

  const handleDelete = async (id) => {
    if (!window.confirm('Delete this poll? This cannot be undone.')) return
    setDeletingId(id)
    try {
      await api.deletePoll(id)
      setPolls((prev) => prev.filter((p) => p.id !== id))
    } catch (err) {
      setError(err.message)
    } finally {
      setDeletingId(null)
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
              <button
                className="btn-secondary"
                onClick={() => handleToggleStatus(p.id, p.is_closed)}
                disabled={statusChangingId === p.id}
              >
                {statusChangingId === p.id
                  ? (p.is_closed ? 'Opening…' : 'Closing…')
                  : (p.is_closed ? 'Reopen' : 'Close')}
              </button>
              <Link to={`/polls/${p.id}`}><button className="btn-secondary">View</button></Link>
              <button
                className="btn-secondary"
                onClick={() => handleDelete(p.id)}
                disabled={deletingId === p.id}
              >
                {deletingId === p.id ? 'Deleting…' : 'Delete'}
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}