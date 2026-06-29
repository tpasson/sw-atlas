// Thin fetch wrapper around the ATLAS backend. All requests are same-origin
// (the Go server serves the SPA in production; Vite proxies /api in dev).
import { demoApi } from './demoApi.js'

const BASE = '/api'

// The workspace the SPA is currently viewing (from the /{slug} URL). Sent on every
// request so the backend serves/guards the right tenant; empty = "own/default".
let workspaceSlug = ''
export function setWorkspaceSlug(slug) { workspaceSlug = slug || '' }

async function req(method, path, body) {
  const opts = { method, headers: {}, credentials: 'same-origin' }
  if (workspaceSlug) opts.headers['X-Atlas-Workspace'] = workspaceSlug
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
  changeOwnPassword: (password) => req('PUT', '/account/password', { password }),

  // users (admin only)
  listUsers: () => req('GET', '/users'),
  createUser: (data) => req('POST', '/users', data),
  setUserRole: (id, role) => req('PUT', `/users/${id}/role`, { role }),
  setUserPassword: (id, password) => req('PUT', `/users/${id}/password`, { password }),
  deleteUser: (id) => req('DELETE', `/users/${id}`),

  // explore / discovery
  listPublicWorkspaces: () => req('GET', '/explore'),
  setWorkspaceFeatured: (slug, featured) => req('PUT', `/workspaces/${slug}/featured`, { featured }),

  // projects (multi-member workspaces)
  listProjects: () => req('GET', '/projects'),
  createProject: (data) => req('POST', '/projects', data),
  renameProject: (slug, name) => req('PUT', `/projects/${slug}`, { name }),
  deleteProject: (slug) => req('DELETE', `/projects/${slug}`),
  leaveProject: (slug) => req('POST', `/projects/${slug}/leave`),
  listMembers: (slug) => req('GET', `/projects/${slug}/members`),
  inviteMember: (slug, username, role) => req('POST', `/projects/${slug}/members`, { username, role }),
  setMemberRole: (slug, userId, role) => req('PUT', `/projects/${slug}/members/${userId}/role`, { role }),
  removeMember: (slug, userId) => req('DELETE', `/projects/${slug}/members/${userId}`),
  // roster of the currently-viewed workspace (assignee picker + avatars)
  workspaceMembers: () => req('GET', '/members'),

  // plan + settings
  getPlan: () => req('GET', '/plan'),
  exportPlan: () => req('GET', '/export'),
  importPlan: (envelope) => req('POST', '/import', envelope),
  getPublicRead: () => req('GET', '/settings/public-read'),
  setPublicRead: (enabled) => req('PUT', '/settings/public-read', { enabled }),
  getPalette: () => req('GET', '/settings/palette'),
  setPalette: (colors) => req('PUT', '/settings/palette', { colors }),
  getGroups: () => req('GET', '/settings/groups'),
  setGroups: (groups) => req('PUT', '/settings/groups', { groups }),
  getUISettings: () => req('GET', '/settings/ui'),
  setUISettings: (settings) => req('PUT', '/settings/ui', { settings }),
  getGitColors: () => req('GET', '/settings/git-colors'),
  setGitColors: (colors) => req('PUT', '/settings/git-colors', colors),

  // Item-type registry: catalog (built-ins + custom) and saving custom types.
  itemTypes: () => req('GET', '/item-types'),
  setItemTypes: (types) => req('PUT', '/item-types', types),

  // swimlanes
  createSwimlane: (data) => req('POST', '/swimlanes', data),
  updateSwimlane: (id, patch) => req('PUT', `/swimlanes/${id}`, patch),
  deleteSwimlane: (id) => req('DELETE', `/swimlanes/${id}`),
  moveSwimlane: (id, dir) => req('POST', `/swimlanes/${id}/move`, { dir }),
  reorderSwimlanes: (ids) => req('POST', '/swimlanes/reorder', { ids }),
  createSubLane: (swimlaneId, data) => req('POST', `/swimlanes/${swimlaneId}/sublanes`, data),
  updateSubLane: (id, name) => req('PUT', `/sublanes/${id}`, { name }),
  reorderSubLanes: (ids) => req('POST', '/sublanes/reorder', { ids }),
  deleteSubLane: (id) => req('DELETE', `/sublanes/${id}`),

  // items
  createItem: (data) => req('POST', '/items', data),
  updateItem: (id, data) => req('PUT', `/items/${id}`, data),
  deleteItem: (id) => req('DELETE', `/items/${id}`),

  // links
  addLink: (a, b, rel = 'depends-on') => req('POST', '/links', { a, b, rel }),
  removeLink: (a, b, rel = 'depends-on') => req('DELETE', '/links', { a, b, rel }),

  // sharing — federation producer side
  listShareScopes: () => req('GET', '/share-scopes'),
  createShareScope: (data) => req('POST', '/share-scopes', data),
  deleteShareScope: (id) => req('DELETE', `/share-scopes/${id}`),
  listShareTokens: (scopeId) => req('GET', `/share-scopes/${scopeId}/tokens`),
  createShareToken: (scopeId, label) => req('POST', `/share-scopes/${scopeId}/tokens`, { label }),
  revokeShareToken: (id) => req('DELETE', `/share-tokens/${id}`),
  setShareScopePublished: (id, published) => req('POST', `/share-scopes/${id}/publish`, { published }),
  listAvailableShares: () => req('GET', '/shares/available'),

  // subscriptions — federation consumer side
  listSubscriptions: () => req('GET', '/subscriptions'),
  createSubscription: (data) => req('POST', '/subscriptions', data),
  deleteSubscription: (id) => req('DELETE', `/subscriptions/${id}`),
  syncSubscription: (id) => req('POST', `/subscriptions/${id}/sync`),
  setSwimlaneHidden: (id, hidden) => req('POST', `/swimlanes/${id}/hidden`, { hidden }),

  // GitHub sources — pull releases/tags/issues/PRs into a read-only swimlane
  listGitHubSources: () => req('GET', '/github-sources'),
  createGitHubSource: (data) => req('POST', '/github-sources', data),
  syncGitHubSource: (id) => req('POST', `/github-sources/${id}/sync`),
  setGitHubSourceToken: (id, token) => req('POST', `/github-sources/${id}/token`, { token }),
  deleteGitHubSource: (id) => req('DELETE', `/github-sources/${id}`),

  // baselines (P2)
  listBaselines: () => req('GET', '/baselines'),
  getBaseline: (id) => req('GET', `/baselines/${id}`),
  createBaseline: (name, note = '') => req('POST', '/baselines', { name, note }),
  deleteBaseline: (id) => req('DELETE', `/baselines/${id}`),

  // item version history (attribution + revisions)
  listRevisions: (id) => req('GET', `/items/${id}/revisions`),
  getRevision: (id, version) => req('GET', `/items/${id}/revisions/${version}`),

  // change requests (propose → owner approves/rejects → applied to the plan)
  listChangeRequests: () => req('GET', '/change-requests'),
  createChangeRequest: (data) => req('POST', '/change-requests', data),
  approveChangeRequest: (id, note = '') => req('POST', `/change-requests/${id}/approve`, { note }),
  rejectChangeRequest: (id, note = '') => req('POST', `/change-requests/${id}/reject`, { note }),
}

// The static GitHub Pages build (VITE_DEMO=1) runs without a backend: swap in
// the localStorage-backed demo adapter.
export const api = import.meta.env.VITE_DEMO ? demoApi : realApi
