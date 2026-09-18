import { useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { api, pollSocketUrl } from '../api.js'

export default function PollView() {
  const { id } = useParams()
  const [poll, setPoll] = useState(null)
  const [results, setResults] = useState(null)
  const [error, setError] = useState('')
  const [voting, setVoting] = useState(false)
  const [votedOption, setVotedOption] = useState(null)
  const [connected, setConnected] = useState(false)
  const wsRef = useRef(null)

  // Initial load: poll metadata + current snapshot.
  useEffect(() => {
    api.getPoll(id)
      .then((data) => {
        setPoll(data.poll)
        setResults(data.results)
      })
      .catch((err) => setError(err.message))
  }, [id])

  // Live updates: one WebSocket connection per poll page. The server pushes
  // a fresh results snapshot every time anyone votes — no polling/refresh needed.
  useEffect(() => {
    const ws = new WebSocket(pollSocketUrl(id))
    wsRef.current = ws

    ws.onopen = () => setConnected(true)
    ws.onclose = () => setConnected(false)
    ws.onerror = () => setConnected(false)
    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        setResults(data)
      } catch (e) {
        // ignore malformed frames
      }
    }

    return () => ws.close()
  }, [id])

  const castVote = async (optionId) => {
    setError('')
    setVoting(true)
    try {
      const data = await api.vote(id, optionId)
      setResults(data)
      setVotedOption(optionId)
    } catch (err) {
      setError(err.message)
    } finally {
      setVoting(false)
    }
  }

  const copyLink = () => {
    navigator.clipboard?.writeText(window.location.href)
  }

  if (error && !poll) {
    return <div className="container"><p className="error">{error}</p></div>
  }
  if (!poll) {
    return <div className="container"><p className="muted">Loading poll…</p></div>
  }

  const total = results?.total || 0

  return (
    <div className="container">
      <div className="card">
        <h2>{poll.question}</h2>

        <div className="share-box" style={{ marginBottom: 20 }}>
          <span>{window.location.href}</span>
          <button className="btn-secondary" onClick={copyLink}>Copy</button>
        </div>

        {!votedOption ? (
          <>
            {poll.options.map((opt) => (
              <button
                key={opt.id}
                className="btn-secondary"
                style={{ display: 'block', width: '100%', textAlign: 'left', marginTop: 8 }}
                disabled={voting || poll.is_closed}
                onClick={() => castVote(opt.id)}
              >
                {opt.text}
              </button>
            ))}
            {poll.is_closed && <p className="muted" style={{ marginTop: 10 }}>This poll is closed.</p>}
          </>
        ) : null}

        {error && <p className="error">{error}</p>}

        <h3 style={{ marginTop: 28 }}>
          <span className="live-dot" style={{ background: connected ? '#3ddc84' : '#666' }} />
          Live results {connected ? '' : '(reconnecting…)'}
        </h3>

        {poll.options.map((opt) => {
          const count = results?.counts?.[opt.id] || 0
          const pct = total > 0 ? Math.round((count / total) * 100) : 0
          return (
            <div className="result-row" key={opt.id}>
              <div className="result-label">
                <span>{opt.text}</span>
                <span className="muted">{count} vote{count === 1 ? '' : 's'} ({pct}%)</span>
              </div>
              <div className="result-bar-track">
                <div className="result-bar-fill" style={{ width: `${pct}%` }} />
              </div>
            </div>
          )
        })}

        <p className="muted" style={{ marginTop: 16 }}>Total votes: {total}</p>
      </div>
    </div>
  )
}
