const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

function authHeaders() {
  const token = localStorage.getItem('token');
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function request(path, options = {}) {
  const res = await fetch(`${API_BASE}/api${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...authHeaders(),
      ...(options.headers || {}),
    },
  });

  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || `Request failed (${res.status})`);
  }
  return data;
}

export const api = {
  signup: (payload) => request('/auth/signup', { method: 'POST', body: JSON.stringify(payload) }),
  login: (payload) => request('/auth/login', { method: 'POST', body: JSON.stringify(payload) }),
  createPoll: (payload) => request('/polls', { method: 'POST', body: JSON.stringify(payload) }),
  myPolls: () => request('/polls'),
  getPoll: (id) => request(`/polls/${id}`),
  vote: (id, optionId) =>
    request(`/polls/${id}/vote`, { method: 'POST', body: JSON.stringify({ option_id: optionId }) }),
  closePoll: (id) => request(`/polls/${id}/close`, { method: 'PATCH' }),
};

// Render's free tier spins backends down after inactivity; the first request
// can take 30-50s to wake it. This wraps any api call and flips a flag if it's
// taking unusually long, so the UI can show a "waking up" hint instead of
// looking stuck.
export function withColdStartHint(promiseFactory, onSlow, delayMs = 4000) {
  return async (...args) => {
    const timer = setTimeout(() => onSlow(true), delayMs)
    try {
      return await promiseFactory(...args)
    } finally {
      clearTimeout(timer)
      onSlow(false)
    }
  }
}

export function pollSocketUrl(id) {
  const wsBase = API_BASE.replace(/^http/, 'ws');
  return `${wsBase}/api/polls/${id}/ws`;
}
