import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api.js'

export default function CreatePoll() {
  const [question, setQuestion] = useState('')
  const [options, setOptions] = useState(['', ''])
  const [allowMulti, setAllowMulti] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  const updateOption = (i, value) => {
    const next = [...options]
    next[i] = value
    setOptions(next)
  }

  const addOption = () => {
    if (options.length < 10) setOptions([...options, ''])
  }

  const removeOption = (i) => {
    if (options.length > 2) setOptions(options.filter((_, idx) => idx !== i))
  }

  const submit = async (e) => {
    e.preventDefault()
    setError('')
    const cleaned = options.map((o) => o.trim()).filter(Boolean)
    if (cleaned.length < 2) {
      setError('Add at least 2 non-empty options.')
      return
    }
    setLoading(true)
    try {
      const poll = await api.createPoll({ question, options: cleaned, allow_multi: allowMulti })
      navigate(`/polls/${poll.id}`)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="container">
      <div className="card">
        <h2>Create a poll</h2>
        <form onSubmit={submit}>
          <label>Question</label>
          <input value={question} onChange={(e) => setQuestion(e.target.value)} required />

          <label>Options</label>
          {options.map((opt, i) => (
            <div className="option-row" key={i}>
              <input
                value={opt}
                onChange={(e) => updateOption(i, e.target.value)}
                placeholder={`Option ${i + 1}`}
                required
              />
              {options.length > 2 && (
                <button type="button" className="btn-secondary" onClick={() => removeOption(i)}>×</button>
              )}
            </div>
          ))}
          {options.length < 10 && (
            <button type="button" className="btn-secondary" style={{ marginTop: 10 }} onClick={addOption}>
              + Add option
            </button>
          )}

          <label style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 18 }}>
            <input
              type="checkbox"
              style={{ width: 'auto' }}
              checked={allowMulti}
              onChange={(e) => setAllowMulti(e.target.checked)}
            />
            Allow voters to vote more than once
          </label>

          {error && <p className="error">{error}</p>}
          <button className="btn-primary" style={{ marginTop: 18, width: '100%' }} disabled={loading}>
            {loading ? 'Creating…' : 'Create poll'}
          </button>
        </form>
      </div>
    </div>
  )
}
