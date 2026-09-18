const API_BASE = import.meta.env.VITE_API_URL || '';

function getToken() {
  return localStorage.getItem('token');
}

async function request(path, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    ...options.headers,
  };

  const token = getToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  let response;
  try {
    response = await fetch(`${API_BASE}/api${path}`, {
      ...options,
      headers,
    });
  } catch (err) {
    throw new Error('Could not connect to backend server. Make sure the backend is running on port 8080.');
  }

  let data;
  const contentType = response.headers.get('content-type');
  if (contentType && contentType.includes('application/json')) {
    try {
      data = await response.json();
    } catch {
      data = null;
    }
  } else {
    try {
      const text = await response.text();
      data = text ? { message: text } : null;
    } catch {
      data = null;
    }
  }

  if (!response.ok) {
    const errorMsg = data?.error || data?.message || `Server returned error (${response.status})`;
    throw new Error(errorMsg);
  }

  return data;
}

export const auth = {
  signup: (data) =>
    request('/auth/signup', { method: 'POST', body: JSON.stringify(data) }),
  login: (data) =>
    request('/auth/login', { method: 'POST', body: JSON.stringify(data) }),
  me: () => request('/auth/me'),
};

export const polls = {
  create: (data) =>
    request('/polls', { method: 'POST', body: JSON.stringify(data) }),
  list: () => request('/polls'),
  get: (id) => request(`/polls/${id}`),
  close: (id) =>
    request(`/polls/${id}/close`, { method: 'PATCH' }),
  remove: (id) =>
    request(`/polls/${id}`, { method: 'DELETE' }),
  vote: (id, optionId) =>
    request(`/polls/${id}/vote`, {
      method: 'POST',
      body: JSON.stringify({ option_id: optionId }),
    }),
};

// Returns the SSE URL for live results
export function getLiveURL(pollId) {
  return `${API_BASE}/api/polls/${pollId}/live`;
}
