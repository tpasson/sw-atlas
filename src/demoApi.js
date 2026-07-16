// Backend-less API used by the static GitHub Pages demo. It implements the same
// interface as the real `api` (see api.js) but operates on a localStorage-backed
// dataset, so the published demo is a fully interactive sandbox (no login, no
// server, changes persist in the browser only).
import { demoSeed } from './demoSeed.js'

const KEY = 'atlas-demo-v8'
const uid = () =>
  (typeof crypto !== 'undefined' && crypto.randomUUID)
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random().toString(16).slice(2)}`

function load() {
  try {
    const raw = localStorage.getItem(KEY)
    if (raw) return JSON.parse(raw)
  } catch { /* ignore */ }
  const seed = demoSeed()
  return {
    swimlanes: seed.swimlanes,
    milestones: seed.milestones,
    links: seed.links,
    baselines: [],
    palette: null,
    groups: [],
    settings: { publicReadEnabled: true },
    // Pre-connected source: the demo auto-syncs this live from GitHub on first load.
    githubSources: [{
      id: 'gh-atlas', owner: 'tpasson', repo: 'sw-atlas', provider: 'github',
      htmlUrl: 'https://github.com/tpasson/sw-atlas',
      includeReleases: true, includeTags: false, includeIssues: true, includePrs: false,
      stableOnly: false, stateFilter: 'all', sinceDate: '', maxPerType: 0,
      lastSyncedAt: null, lastStatus: '', createdAt: '2026-06-25T18:37:47Z',
    }],
  }
}

let db = load()
function save() {
  try { localStorage.setItem(KEY, JSON.stringify(db)) } catch { /* ignore quota */ }
}
const ok = (v = null) => Promise.resolve(v)

// ── GitHub source (no backend → fetch GitHub's REST API straight from the browser;
// its CORS headers allow this for public repos). Mirrors server/internal/store/github.go.
function parseRepo(raw) {
  let u
  try { u = new URL((raw || '').trim()) } catch { throw new Error('invalid URL') }
  const host = u.hostname.replace(/^www\./, '')
  if (host !== 'github.com') throw new Error('expected a https://github.com/owner/repo URL')
  const parts = u.pathname.split('/').filter(Boolean)
  if (parts.length < 2) throw new Error('URL must be https://github.com/owner/repo')
  return { owner: parts[0], repo: parts[1].replace(/\.git$/, '') }
}
function ghDate(ts) {
  if (!ts) return null
  const d = new Date(ts)
  if (isNaN(d.getTime())) return null
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  return { when: `${d.getFullYear()}-${mm}-${dd}`, year: d.getFullYear(), month: d.getMonth() + 1 }
}
function ghText(s, n) {
  // strip common markdown so the body reads as plain prose; keep line breaks
  s = (s || '')
    .replace(/<!--[\s\S]*?-->/g, '')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, '')
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/^[ \t]*#{1,6}[ \t]*/gm, '')
    .replace(/^[ \t]*>[ \t]?/gm, '')
    .replace(/(\*\*|__|\*|`|~~)/g, '')
    .replace(/[ \t]+/g, ' ')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
  return s.length > n ? s.slice(0, n).trim() + '…' : s
}
function ghState(cfg) {
  return cfg.stateFilter === 'open' || cfg.stateFilter === 'closed' ? cfg.stateFilter : 'all'
}
function ghLimit(items, since, max) {
  let out = items
  if (since) out = out.filter(it => (it.when || '') >= since)
  if (max > 0 && out.length > max) {
    out = out.slice().sort((a, b) => (b.when || '').localeCompare(a.when || '')).slice(0, max)
  }
  return out
}
async function ghFetch(cfg, path) {
  const headers = { Accept: 'application/vnd.github+json' }
  if (cfg.token) headers.Authorization = 'Bearer ' + cfg.token
  const res = await fetch(`https://api.github.com/repos/${cfg.owner}/${cfg.repo}${path}`, { headers })
  if (res.status === 404) throw new Error('repository not found (or private — add a token)')
  if (res.status === 403) throw new Error('GitHub rate limit reached — try later or add a token')
  if (!res.ok) throw new Error('GitHub returned ' + res.status)
  return res.json()
}
function ghItem(extId, sub, title, dt, marker, scmUrl, what, who, progress, maturity, color) {
  return {
    _extId: extId, _sub: sub, title, year: dt.year, month: dt.month, when: dt.when,
    kind: 'milestone', marker, scmUrl, data: what ? { what } : {}, // body → data.what; author (who) retired
    startDate: null, endDate: null, progress: progress ?? null, maturity: maturity ?? null, color: color ?? null,
  }
}
// Per-"workspace" synced-item colours (demo: stored locally). '' → inherit lane.
const DEFAULT_GIT_COLORS = { releaseStable: '', releasePre: '#FF9F0A', tag: '', issueOpen: '#3FB950', issueClosed: '#8957E5', prOpen: '#3FB950', prMerged: '#8957E5', prClosed: '#F85149' }
const gc = () => ({ ...DEFAULT_GIT_COLORS, ...(db.gitColors || {}) })
const gcv = (v) => v || null

// Standard prose fields every type ships with (mirrors DescriptionFields()).
const DESC_FIELDS = [
  { key: 'what', label: 'What', type: 'textarea' },
  { key: 'why', label: 'Why', type: 'textarea' },
  { key: 'how', label: 'Where', type: 'textarea' },
]
// Item-type registry (demo): built-ins + any locally-saved custom types.
const BUILTIN_ITEM_TYPES = [
  { key: 'milestone', label: 'Milestone', family: 'timeline-point', icon: 'l:Diamond', color: '', fields: [...DESC_FIELDS], workflowKey: 'standard', builtin: true },
  { key: 'event', label: 'Event', family: 'timeline-range', icon: 'l:Flag', color: '', fields: [...DESC_FIELDS], workflowKey: 'standard', builtin: true },
]
const BUILTIN_TYPE_KEYS = new Set(['milestone', 'event'])

// Shipped default workflows (mirrors the server's DefaultWorkflows). "standard"
// carries a hand-arranged status-flow layout so the diagram looks right at once.
const DEFAULT_WORKFLOWS = [{
  key: 'standard', label: 'Standard', builtin: true,
  statuses: [
    { key: 'todo', label: 'To Do', tone: 'neutral', to: ['in-progress', 'cancelled'] },
    { key: 'in-progress', label: 'In Progress', tone: 'progress', to: ['blocked', 'done', 'cancelled'] },
    { key: 'blocked', label: 'Blocked', tone: 'warning', to: ['in-progress', 'cancelled'] },
    { key: 'done', label: 'Done', tone: 'positive', to: ['in-progress'] },
    { key: 'cancelled', label: 'Cancelled', tone: 'negative', to: ['todo'] },
  ],
  layout: { nodes: { todo: { x: 49, y: 28 }, 'in-progress': { x: 270, y: 27 }, blocked: { x: 270, y: 127 }, done: { x: 270, y: -76 }, cancelled: { x: 653, y: 31 } }, edges: { 'todo|cancelled': { x: 352, y: -147, a: 'T', b: 'T' }, 'in-progress|done': { a: 'T', b: 'B' }, 'in-progress|cancelled': { b: 'L' }, 'blocked|cancelled': { x: 526, y: 128, a: 'R', b: 'B' } } },
}]
// Built-in defaults, overridden by a stored workflow of the same key, then custom.
function allWorkflows() {
  const stored = db.workflows || []
  const byKey = Object.fromEntries(stored.map(w => [w.key, w]))
  const isDefault = new Set(DEFAULT_WORKFLOWS.map(d => d.key))
  const out = DEFAULT_WORKFLOWS.map(d => (byKey[d.key] ? { ...byKey[d.key], builtin: true } : d))
  for (const w of stored) if (!isDefault.has(w.key)) out.push({ ...w, builtin: false })
  return out
}
const itemTypeCatalog = () => {
  const stored = db.itemTypes || []
  const overrides = {}
  const custom = []
  for (const t of stored) {
    if (!t.key) continue
    if (BUILTIN_TYPE_KEYS.has(t.key)) overrides[t.key] = t
    else custom.push({ ...t, builtin: false, fields: t.fields || [], workflowKey: t.workflowKey || 'standard' })
  }
  const builtins = BUILTIN_ITEM_TYPES.map(d => {
    const ov = overrides[d.key]
    if (!ov) return d
    return { ...d, label: ov.label || d.label, icon: ov.icon || d.icon, color: ov.color || '', fill: ov.fill, fields: ov.fields || d.fields, workflowKey: ov.workflowKey || 'standard', statuses: ov.statuses, layout: ov.layout, builtin: true }
  })
  const out = [...builtins, ...custom]
  // Every type gains the standard description fields if missing (mirrors the
  // 00034 migration): prepended, deduped by key.
  for (const t of out) {
    const have = new Set((t.fields || []).map(f => f.key))
    t.fields = [...DESC_FIELDS.filter(d => !have.has(d.key)), ...(t.fields || [])]
  }
  // Resolve shared-workflow references (mirrors the server).
  const byKey = Object.fromEntries(allWorkflows().map(w => [w.key, w]))
  for (const t of out) {
    if (t.workflowKey && byKey[t.workflowKey]) { t.statuses = byKey[t.workflowKey].statuses; t.layout = byKey[t.workflowKey].layout }
  }
  return out
}

async function ghReleases(cfg) {
  const rels = await ghFetch(cfg, '/releases?per_page=100')
  const items = [], tagSet = new Set()
  for (const r of rels) {
    if (r.draft) continue
    tagSet.add(r.tag_name) // dedup tags against all release tags, even filtered-out ones
    if (cfg.stableOnly && r.prerelease) continue
    const dt = ghDate(r.published_at || r.created_at); if (!dt) continue
    items.push(ghItem('release:' + r.tag_name, 'releases', r.name || r.tag_name, dt, 'l:Tag', r.html_url,
      ghText(r.body, 4000), r.author && r.author.login, 100, 4, gcv(r.prerelease ? gc().releasePre : gc().releaseStable)))
  }
  return { items: ghLimit(items, cfg.since, cfg.maxPerType), tagSet }
}
async function ghTags(cfg, skip) {
  const tags = await ghFetch(cfg, '/tags?per_page=100')
  const items = []; let n = 0
  for (const t of tags) {
    if (skip.has(t.name) || n >= 30) { if (n >= 30) break; else continue }
    n++
    let dt = null
    try { const c = await ghFetch(cfg, '/commits/' + t.commit.sha); dt = ghDate(c.commit && c.commit.committer && c.commit.committer.date) } catch { dt = null }
    if (!dt) continue
    items.push(ghItem('tag:' + t.name, 'tags', t.name, dt, 'l:Tag',
      `https://github.com/${cfg.owner}/${cfg.repo}/releases/tag/${t.name}`, '', '', 100, 4, gcv(gc().tag)))
  }
  return ghLimit(items, cfg.since, cfg.maxPerType)
}
async function ghIssues(cfg) {
  const issues = await ghFetch(cfg, `/issues?state=${ghState(cfg)}&per_page=100`)
  const items = []
  for (const is of issues) {
    if (is.pull_request) continue
    const closed = is.state === 'closed'
    const dt = ghDate(closed && is.closed_at ? is.closed_at : is.created_at); if (!dt) continue
    items.push(ghItem('issue:' + is.number, 'issues', is.title, dt, 'l:CircleDot', is.html_url,
      ghText(is.body, 600), is.user && is.user.login, closed ? 100 : 0, null, gcv(closed ? gc().issueClosed : gc().issueOpen)))
  }
  return ghLimit(items, cfg.since, cfg.maxPerType)
}
async function ghPulls(cfg) {
  const prs = await ghFetch(cfg, `/pulls?state=${ghState(cfg)}&per_page=100`)
  const items = []
  for (const p of prs) {
    const merged = !!p.merged_at
    const ts = merged ? p.merged_at : (p.state === 'closed' && p.closed_at ? p.closed_at : p.created_at)
    const dt = ghDate(ts); if (!dt) continue
    let color = gc().prOpen, progress = 50
    if (merged) { color = gc().prMerged; progress = 100 }
    else if (p.state === 'closed') { color = gc().prClosed; progress = 0 }
    items.push(ghItem('pr:' + p.number, 'prs', p.title, dt, 'l:GitPullRequest', p.html_url,
      ghText(p.body, 600), p.user && p.user.login, progress, null, gcv(color)))
  }
  return ghLimit(items, cfg.since, cfg.maxPerType)
}
async function buildGitHubMirror(cfg) {
  let { releases, tags, issues, prs } = cfg
  if (!releases && !tags && !issues && !prs) releases = true
  const subLanes = [], items = []
  let relTags = new Set()
  if (releases) { const r = await ghReleases(cfg); relTags = r.tagSet; if (r.items.length) { subLanes.push({ id: 'releases', name: 'Releases' }); items.push(...r.items) } }
  if (tags)     { const t = await ghTags(cfg, relTags); if (t.length) { subLanes.push({ id: 'tags', name: 'Tags' }); items.push(...t) } }
  if (issues)   { const i = await ghIssues(cfg); if (i.length) { subLanes.push({ id: 'issues', name: 'Issues' }); items.push(...i) } }
  if (prs)      { const p = await ghPulls(cfg); if (p.length) { subLanes.push({ id: 'prs', name: 'Pull requests' }); items.push(...p) } }
  return { subLanes, items }
}
function removeSourceContent(srcId) {
  db.swimlanes = db.swimlanes.filter(s => s.sourceSystem !== srcId)
  db.milestones = db.milestones.filter(m => m.sourceSystem !== srcId)
}
async function ghSync(src) {
  removeSourceContent(src.id)
  try {
    const built = await buildGitHubMirror({
      owner: src.owner, repo: src.repo, token: src._token || '',
      releases: src.includeReleases, tags: src.includeTags, issues: src.includeIssues, prs: src.includePrs,
      stableOnly: src.stableOnly, stateFilter: src.stateFilter, since: src.sinceDate, maxPerType: src.maxPerType,
    })
    const laneId = uid(), subMap = {}
    const subLanes = built.subLanes.map(sl => { const i = uid(); subMap[sl.id] = i; return { id: i, name: sl.name } })
    db.swimlanes.push({ id: laneId, name: src.repo, color: '#6E5494', subLanes, sourceSystem: src.id, sourceKind: src.provider || 'github', hidden: false })
    const ext = `https://github.com/${src.owner}/${src.repo}`, now = new Date().toISOString()
    for (const it of built.items) {
      const { _extId, _sub, ...rest } = it
      db.milestones.push({ ...rest, id: uid(), swimlaneId: laneId, subLaneId: subMap[_sub] || null,
        sourceSystem: src.id, externalId: _extId, externalUrl: ext, lastSyncedAt: now })
    }
    src.lastStatus = `ok · ${built.items.length} items`
  } catch (e) {
    src.lastStatus = 'error: ' + (e.message || 'sync failed')
  }
  src.lastSyncedAt = new Date().toISOString()
}

export const demoApi = {
  // auth (the demo is an open, editable sandbox; no real accounts/roles)
  me: () => ok({ authenticated: true }),
  login: () => ok({ authenticated: true }),
  logout: () => ok({ authenticated: false }),
  changeOwnPassword: () => ok(),
  renameOwnUsername: () => Promise.reject(new Error('Renaming is unavailable in the demo')),
  updateOwnProfile: () => ok(),
  getInstanceUISettings: () => ok({ settings: null }),
  setInstanceUISettings: () => ok(),
  getServerInfo: () => ok({ settings: {}, stats: { version: 'demo', uptimeSeconds: 0, users: 0, workspaces: 0, items: 0 } }),
  setServerSettings: () => ok(),
  getLimits: () => ok({ writesPerMinute: 240, maxItemsPerPlan: 2000, maxProjectsPerUser: 50 }),
  setLimits: (l) => ok(l),

  // users — not available in the backend-less demo (admin UI stays hidden)
  listUsers: () => ok({ users: [] }),
  createUser: () => Promise.reject(new Error('User management is unavailable in the demo')),
  setUserRole: () => ok(),
  renameUser: () => ok(),
  setUserPassword: () => ok(),
  deleteUser: () => ok(),

  // explore — the demo is a single sandbox, no directory
  listPublicWorkspaces: () => ok({ workspaces: [] }),
  setWorkspaceFeatured: () => ok(),

  // Return clones so the reactive store never shares array refs with the db
  // (otherwise an optimistic push + the db push would duplicate the item).
  getPlan: () => ok({
    swimlanes: db.swimlanes.map(s => ({ ...s, subLanes: s.subLanes.map(sl => ({ ...sl })) })),
    milestones: db.milestones.map(m => ({ ...m })),
    links: db.links.map(l => ({ ...l })),
  }),
  exportPlan: () => ok({
    atlas: { schema: 1, kind: 'export', generatedAt: new Date().toISOString() },
    swimlanes: db.swimlanes.map(s => ({ ...s, subLanes: s.subLanes.map(sl => ({ ...sl })) })),
    milestones: db.milestones.map(m => ({ ...m })),
    links: db.links.map(l => ({ ...l })),
  }),
  importPlan: (env) => {
    const swMap = {}, subMap = {}, itMap = {}
    for (const sw of (env.swimlanes || [])) {
      const nid = uid(); swMap[sw.id] = nid
      const nsw = { id: nid, name: sw.name, color: sw.color || '#0A84FF', subLanes: [] }
      for (const sub of (sw.subLanes || [])) {
        const nsid = uid(); subMap[sub.id] = nsid
        nsw.subLanes.push({ id: nsid, name: sub.name })
      }
      db.swimlanes.push(nsw)
    }
    let items = 0, links = 0
    for (const m of (env.milestones || [])) {
      const nsw = swMap[m.swimlaneId]; if (!nsw) continue
      const nid = uid(); itMap[m.id] = nid
      const nsub = (m.subLaneId != null && subMap[m.subLaneId]) ? subMap[m.subLaneId] : null
      // Copy-mode: strip provenance so imported items are native/editable.
      db.milestones.push({ ...m, id: nid, swimlaneId: nsw, subLaneId: nsub,
        sourceSystem: null, externalId: null, externalUrl: null, lastSyncedAt: null })
      items++
    }
    for (const l of (env.links || [])) {
      const a = itMap[l.a], b = itMap[l.b]
      if (a && b && a !== b && !db.links.some(x => (x.a === a && x.b === b) || (x.a === b && x.b === a))) {
        db.links.push({ a, b }); links++
      }
    }
    save()
    return ok({ swimlanes: Object.keys(swMap).length, subLanes: Object.keys(subMap).length, items, links })
  },
  getPublicRead: () => ok({ enabled: db.settings.publicReadEnabled }),
  setPublicRead: (enabled) => { db.settings.publicReadEnabled = enabled; save(); return ok({ enabled }) },
  getPalette: () => ok({ colors: db.palette == null ? null : [...db.palette] }),
  setPalette: (colors) => { db.palette = colors || []; save(); return ok({ colors: [...db.palette] }) },
  getGroups: () => ok({ groups: (db.groups || []).map(g => ({ ...g, itemIds: [...(g.itemIds || [])] })) }),
  setGroups: (groups) => { db.groups = groups || []; save(); return ok({ groups: db.groups }) },
  // Display settings stay per-browser (localStorage) in the backend-less demo.
  getUISettings: () => ok({ settings: null }),
  setUISettings: () => ok(),
  getGitColors: () => ok(gc()),
  setGitColors: (c) => { db.gitColors = c; save(); return ok(c) },
  listProjects: () => ok([{ slug: '', name: 'Demo plan', role: 'owner', visibility: 'public' }]),
  createProject: () => ok({ slug: '', name: 'Demo plan', visibility: 'private' }),
  renameProject: () => ok(),
  deleteProject: () => Promise.reject(new Error('Deleting the project is disabled in the demo')),
  leaveProject: () => ok(),
  listMembers: () => ok([]),
  workspaceMembers: () => ok([]),
  inviteMember: () => Promise.reject(new Error('Inviting members is disabled in the demo')),
  setMemberRole: () => ok(),
  removeMember: () => ok(),
  itemTypes: () => ok(itemTypeCatalog()),
  setItemTypes: (types) => {
    // Store both built-in restyles and custom types; the catalog reconciles them.
    db.itemTypes = (types || []).filter(t => t.key).map(t => ({ ...t, fields: t.fields || [] }))
    save()
    return ok(itemTypeCatalog())
  },
  workflows: () => ok(allWorkflows()),
  setWorkflows: (wfs) => {
    db.workflows = (wfs || []).filter(w => w.key).map(w => ({ ...w, statuses: w.statuses || [] }))
    save()
    return ok(db.workflows)
  },

  createSwimlane: (data) => {
    const sw = { id: data.id || uid(), name: data.name, color: data.color || '#0A84FF', subLanes: [] }
    db.swimlanes.push(sw); save(); return ok(sw)
  },
  updateSwimlane: (id, patch) => {
    const s = db.swimlanes.find(s => s.id === id); if (s) Object.assign(s, patch); save(); return ok()
  },
  deleteSwimlane: (id) => {
    const ids = db.milestones.filter(m => m.swimlaneId === id).map(m => m.id)
    db.swimlanes = db.swimlanes.filter(s => s.id !== id)
    db.milestones = db.milestones.filter(m => m.swimlaneId !== id)
    db.links = db.links.filter(l => !ids.includes(l.a) && !ids.includes(l.b))
    save(); return ok()
  },
  moveSwimlane: (id, dir) => {
    const i = db.swimlanes.findIndex(s => s.id === id); const j = i + dir
    if (i >= 0 && j >= 0 && j < db.swimlanes.length) {
      const t = db.swimlanes[i]; db.swimlanes[i] = db.swimlanes[j]; db.swimlanes[j] = t
    }
    save(); return ok()
  },
  reorderSwimlanes: (ids) => {
    const byId = Object.fromEntries(db.swimlanes.map(s => [s.id, s]))
    db.swimlanes = ids.map(id => byId[id]).filter(Boolean)
    save(); return ok()
  },
  setSwimlaneHidden: (id, hidden) => {
    const s = db.swimlanes.find(s => s.id === id); if (s) s.hidden = hidden
    save(); return ok()
  },
  createSubLane: (swimlaneId, data) => {
    const sl = db.swimlanes.find(s => s.id === swimlaneId)
    const sub = { id: data.id || uid(), name: data.name }
    if (sl) sl.subLanes.push(sub); save(); return ok(sub)
  },
  updateSubLane: (id, name) => {
    for (const sl of db.swimlanes) { const sub = sl.subLanes.find(s => s.id === id); if (sub) { sub.name = name; break } }
    save(); return ok()
  },
  reorderSubLanes: (ids) => {
    const set = new Set(ids)
    for (const sl of db.swimlanes) {
      if (sl.subLanes.length && sl.subLanes.every(s => set.has(s.id))) {
        const byId = Object.fromEntries(sl.subLanes.map(s => [s.id, s]))
        sl.subLanes = ids.map(id => byId[id]).filter(Boolean); break
      }
    }
    save(); return ok()
  },
  deleteSubLane: (id) => {
    for (const sl of db.swimlanes) sl.subLanes = sl.subLanes.filter(s => s.id !== id)
    const ids = db.milestones.filter(m => m.subLaneId === id).map(m => m.id)
    db.milestones = db.milestones.filter(m => m.subLaneId !== id)
    db.links = db.links.filter(l => !ids.includes(l.a) && !ids.includes(l.b))
    save(); return ok()
  },

  createItem: (data) => { const it = { ...data, id: data.id || uid() }; db.milestones.push(it); save(); return ok(it) },
  updateItem: (id, data) => { const m = db.milestones.find(m => m.id === id); if (m) Object.assign(m, data); save(); return ok() },
  deleteItem: (id) => {
    db.milestones = db.milestones.filter(m => m.id !== id)
    db.links = db.links.filter(l => l.a !== id && l.b !== id)
    save(); return ok()
  },

  addLink: (a, b, rel = 'depends-on', version = null) => {
    if (a !== b) {
      const e = db.links.find(l => l.a === a && l.b === b && (l.rel || 'depends-on') === rel)
      if (e) e.version = version ?? null; else db.links.push({ a, b, rel, version: version ?? null })
    }
    save(); return ok()
  },
  removeLink: (a, b, rel = 'depends-on') => {
    db.links = db.links.filter(l => !(l.a === a && l.b === b && (l.rel || 'depends-on') === rel))
    save(); return ok()
  },

  // baselines
  listBaselines: () => ok(db.baselines.map(b => ({ id: b.id, name: b.name, note: b.note, createdAt: b.createdAt, itemCount: b.items.length }))),
  getBaseline: (id) => {
    const b = db.baselines.find(b => b.id === id)
    if (!b) return Promise.reject(Object.assign(new Error('not found'), { status: 404 }))
    return ok({ ...b, items: b.items.map(i => ({ ...i })) })
  },
  createBaseline: (name) => {
    const b = {
      id: uid(), name, note: '', createdAt: new Date().toISOString(),
      items: db.milestones.map(m => ({
        id: m.id, swimlaneId: m.swimlaneId, subLaneId: m.subLaneId, year: m.year, month: m.month,
        title: m.title, when: m.when, startDate: m.startDate, endDate: m.endDate, kind: m.kind, marker: m.marker,
      })),
    }
    db.baselines.push(b); save()
    return ok({ id: b.id, name: b.name, note: b.note, createdAt: b.createdAt, itemCount: b.items.length })
  },
  deleteBaseline: (id) => { db.baselines = db.baselines.filter(b => b.id !== id); save(); return ok() },

  // Demo has no server-side history; synthesize the current state as v1.
  listRevisions: (id) => {
    const m = db.milestones.find(x => x.id === id)
    if (!m) return ok([])
    return ok([{ version: m.version || 1, editedBy: null, editedAt: m.updatedAt || new Date().toISOString() }])
  },
  getRevision: (id, version) => {
    const m = db.milestones.find(x => x.id === id)
    if (!m) return Promise.reject(Object.assign(new Error('not found'), { status: 404 }))
    return ok({ version: version || m.version || 1, editedBy: null, editedAt: m.updatedAt || new Date().toISOString(), snapshot: { ...m } })
  },

  // Change requests (demo: you act as both proposer and owner).
  listChangeRequests: () => ok((db.changeRequests || []).map(c => ({ ...c }))),
  createChangeRequest: (data) => {
    const cr = {
      id: uid(), kind: data.kind === 'create' ? 'create' : 'edit', targetItemId: data.targetItemId || null,
      payload: data.payload || {}, note: data.note || '', status: 'pending',
      authorId: 'you', authorName: 'You', decidedBy: null, deciderName: '', decidedAt: null, decisionNote: '',
      createdAt: new Date().toISOString(),
      targetTitle: (db.milestones.find(m => m.id === data.targetItemId) || {}).title || '',
    }
    db.changeRequests = db.changeRequests || []
    db.changeRequests.push(cr); save()
    return ok(cr)
  },
  approveChangeRequest: (id, note = '') => {
    const cr = (db.changeRequests || []).find(c => c.id === id)
    if (!cr || cr.status !== 'pending') return Promise.reject(Object.assign(new Error('not found'), { status: 404 }))
    if (cr.kind === 'create') db.milestones.push({ ...cr.payload })
    else { const m = db.milestones.find(x => x.id === cr.targetItemId); if (m) Object.assign(m, cr.payload) }
    Object.assign(cr, { status: 'approved', decidedBy: 'you', deciderName: 'You', decidedAt: new Date().toISOString(), decisionNote: note })
    save(); return ok(cr)
  },
  rejectChangeRequest: (id, note = '') => {
    const cr = (db.changeRequests || []).find(c => c.id === id)
    if (!cr || cr.status !== 'pending') return Promise.reject(Object.assign(new Error('not found'), { status: 404 }))
    Object.assign(cr, { status: 'rejected', decidedBy: 'you', deciderName: 'You', decidedAt: new Date().toISOString(), decisionNote: note })
    save(); return ok(cr)
  },

  // GitHub sources (public repos; token used for the initial fetch but not persisted)
  listGitHubSources: () => ok({ sources: (db.githubSources || []).map(s => ({ ...s })) }),
  createGitHubSource: async (data) => {
    let parsed
    try { parsed = parseRepo(data.url) } catch (e) { return Promise.reject(e) }
    const src = {
      id: uid(), owner: parsed.owner, repo: parsed.repo, provider: 'github',
      htmlUrl: `https://github.com/${parsed.owner}/${parsed.repo}`,
      includeReleases: !!data.includeReleases, includeTags: !!data.includeTags,
      includeIssues: !!data.includeIssues, includePrs: !!data.includePrs,
      stableOnly: !!data.stableOnly,
      stateFilter: (data.stateFilter === 'open' || data.stateFilter === 'closed') ? data.stateFilter : 'all',
      sinceDate: (data.sinceDate || '').trim(),
      maxPerType: Math.max(0, Number(data.maxPerType) || 0),
      lastSyncedAt: null, lastStatus: '', createdAt: new Date().toISOString(),
      _token: data.token || '',
    }
    if (!src.includeReleases && !src.includeTags && !src.includeIssues && !src.includePrs) src.includeReleases = true
    await ghSync(src)
    const { _token, ...clean } = src
    db.githubSources = db.githubSources || []
    db.githubSources.push(clean) // persisted without the token
    save()
    return ok(clean)
  },
  syncGitHubSource: async (id) => {
    const src = (db.githubSources || []).find(s => s.id === id)
    if (!src) return Promise.reject(Object.assign(new Error('not found'), { status: 404 }))
    await ghSync(src)
    save()
    return ok({ ...src })
  },
  setGitHubSourceToken: async (id, token) => {
    const src = (db.githubSources || []).find(s => s.id === id)
    if (!src) return Promise.reject(Object.assign(new Error('not found'), { status: 404 }))
    src._token = (token || '').trim()
    await ghSync(src)
    delete src._token // never persist the token in the demo
    save()
    return ok({ ...src })
  },
  deleteGitHubSource: (id) => {
    removeSourceContent(id)
    db.githubSources = (db.githubSources || []).filter(s => s.id !== id)
    save()
    return ok()
  },
}
