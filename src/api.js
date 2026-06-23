// Thin fetch wrapper around the ATLAS backend. All requests are same-origin
// (the Go server serves the SPA in production; Vite proxies /api in dev).
import { demoApi } from './demoApi.js'

const BASE = '/api'

async function req(method, path, body) {
  const opts = { method, headers: {}, credentials: 'same-origin' }
  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(body)
  }
  const res = await fetch(BASE + path, opts)
  if (!res.ok) {
    let msg = `HTTP ${res.status}`
    try {
      const j = await res.json()
      if (j && j.error) msg = j.error
    } catch { /* non-JSON error body */ }
    const err = new Error(msg)
    err.status = res.status
    throw err
  }
  if (res.status === 204) return null
  const ct = res.headers.get('content-type') || ''
  return ct.includes('application/json') ? res.json() : null
}

const realApi = {
  // auth
  me: () => req('GET', '/me'),
  login: (username, password) => req('POST', '/login', { username, password }),
  logout: () => req('POST', '/logout'),

  // plan + settings
  getPlan: () => req('GET', '/plan'),
  getPublicRead: () => req('GET', '/settings/public-read'),
  setPublicRead: (enabled) => req('PUT', '/settings/public-read', { enabled }),
  getPalette: () => req('GET', '/settings/palette'),
  setPalette: (colors) => req('PUT', '/settings/palette', { colors }),
  getGroups: () => req('GET', '/settings/groups'),
  setGroups: (groups) => req('PUT', '/settings/groups', { groups }),

  // swimlanes
  createSwimlane: (data) => req('POST', '/swimlanes', data),
  updateSwimlane: (id, patch) => req('PUT', `/swimlanes/${id}`, patch),
  deleteSwimlane: (id) => req('DELETE', `/swimlanes/${id}`),
  moveSwimlane: (id, dir) => req('POST', `/swimlanes/${id}/move`, { dir }),
  createSubLane: (swimlaneId, data) => req('POST', `/swimlanes/${swimlaneId}/sublanes`, data),
  updateSubLane: (id, name) => req('PUT', `/sublanes/${id}`, { name }),
  deleteSubLane: (id) => req('DELETE', `/sublanes/${id}`),

  // items
  createItem: (data) => req('POST', '/items', data),
  updateItem: (id, data) => req('PUT', `/items/${id}`, data),
  deleteItem: (id) => req('DELETE', `/items/${id}`),

  // links
  addLink: (a, b) => req('POST', '/links', { a, b }),
  removeLink: (a, b) => req('DELETE', '/links', { a, b }),

  // baselines (P2)
  listBaselines: () => req('GET', '/baselines'),
  getBaseline: (id) => req('GET', `/baselines/${id}`),
  createBaseline: (name, note = '') => req('POST', '/baselines', { name, note }),
  deleteBaseline: (id) => req('DELETE', `/baselines/${id}`),
}

// The static GitHub Pages build (VITE_DEMO=1) runs without a backend: swap in
// the localStorage-backed demo adapter.
export const api = import.meta.env.VITE_DEMO ? demoApi : realApi
