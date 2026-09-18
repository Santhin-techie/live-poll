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
};

export function pollSocketUrl(id) {
  const wsBase = API_BASE.replace(/^http/, 'ws');
  return `${wsBase}/api/polls/${id}/ws`;
}
