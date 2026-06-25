import { reactive, computed, watch } from 'vue'
import { api } from '../api.js'
import { LUCIDE_MARKER_SHAPES } from '../lucideMarkers.js'

export const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

export const PRESET_COLORS = [
  '#0A84FF', '#30D158', '#FF9F0A', '#FF375F',
  '#BF5AF2', '#5AC8FA', '#FFCC00', '#FF6961',
  '#32ADE6', '#AC8E68', '#636366', '#1C1C1E',
]

// All marker shapes available to choose from (max 8 active at a time) — the full Lucide icon set.
export const MARKER_LIBRARY = LUCIDE_MARKER_SHAPES

const uid = () =>
  (typeof crypto !== 'undefined' && crypto.randomUUID)
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random().toString(16).slice(2)}`

// The viewing year is client-side UI state (not domain data); persist it as a
// lightweight preference so a reload keeps the same view.
const YEAR_KEY = 'atlas-view-year'
function initialYear() {
  const v = parseInt(localStorage.getItem(YEAR_KEY) || '', 10)
  return Number.isFinite(v) ? v : new Date().getFullYear()
}

// Reactive plan cache, populated from the backend (the single source of truth).
export const store = reactive({
  year: initialYear(),
  swimlanes: [],
  milestones: [],
  links: [],
  loaded: false,
})

// Session/auth + global settings state.
export const session = reactive({
  authenticated: false,
  publicReadEnabled: true,
  ready: false,
  error: null, // 'auth-required' | message string | null
})

// UI view preferences (today indicator etc.) — per-browser, persisted locally.
const SETTINGS_KEY = 'atlas-ui-settings'
const SETTINGS_DEFAULTS = {
  monthHighlight: { enabled: true, color: '#0A84FF', opacity: 0.06 },
  dayLine: { enabled: true, color: '#FF3B30', opacity: 0.7, width: 2 },
  weekNumbers: { enabled: true },
  monthLines: { enabled: false, color: '#E5E5EA', opacity: 1, width: 1 },
  weekLines: { enabled: true, color: '#8E8E93', opacity: 0.1, width: 1 },
  items: { fontSize: 14, fontWeight: 700, padding: 4, margin: 6, radius: 10, border: 2, labelOffset: 0, iconGap: 6, labelBuffer: 12, borderMode: 'hover', markerSize: 14, markerStroke: 2, eventOpacity: 0.13, maturitySize: 5, density: 'stack', densityRows: 3 },
  markers: [
    { shape: 'l:Diamond', label: 'Milestone', fill: true },
    { shape: 'l:Circle', label: 'Circle', fill: true },
    { shape: 'l:Triangle', label: 'Decision', fill: true },
    { shape: 'l:Star', label: 'Highlight', fill: true },
  ],
  eventLabel: 'Event (duration)',
  layout: { subAreaWidth: 240 },
  theme: 'light',
}

// Fixed 4-stage maturity scale (mirrors the ATLAS 4-square logo).
export const MATURITY_STAGES = ['Concept', 'Design', 'Production', 'Series']

// Parse a source-control URL into the pieces needed to render a compact badge.
// Supports GitHub, GitLab, Azure DevOps, Gitea/Forgejo/Codeberg and Bitbucket
// (plus a generic git fallback). Returns null for anything that isn't a URL.
// GitLab nests resources under a `/-/` segment (and supports sub-groups), so we
// split the path on that marker; GitHub/Gitea keep them directly under owner/repo;
// Azure DevOps is handled separately (its `_git`/`_workitems` layout differs).
export function parseScmUrl(url) {
  if (!url || typeof url !== 'string') return null
  let u
  try { u = new URL(url.trim()) } catch { return null }
  if (!/^https?:$/.test(u.protocol)) return null
  const host = u.hostname.replace(/^www\./, '')
  const parts = u.pathname.split('/').filter(Boolean)
  const dec = (s) => { try { return decodeURIComponent(s || '') } catch { return s || '' } }

  // Azure DevOps (Services dev.azure.com / *.visualstudio.com, or on-prem Server —
  // detectable by the `_git`/`_workitems` path segment).
  if (host === 'dev.azure.com' || host.endsWith('.visualstudio.com') ||
      parts.includes('_git') || parts.includes('_workitems')) {
    return parseAzureScm(u, host, parts, dec)
  }

  const provider =
    /github/.test(host) ? 'github' :
    /gitlab/.test(host) ? 'gitlab' :
    /bitbucket/.test(host) ? 'bitbucket' :
    (host === 'codeberg.org' || /gitea|forgejo/.test(host)) ? 'gitea' :
    'git'

  const dash = parts.indexOf('-')
  const repo = (dash > 0 ? parts.slice(0, dash) : parts.slice(0, 2)).join('/') || host
  const seg = dash > 0 ? parts.slice(dash + 1) : parts.slice(2)

  let type = 'repo', ref = ''
  switch (seg[0]) {
    case 'releases': type = 'release'; ref = dec(seg[1] === 'tag' ? seg[2] : seg[1]); break
    case 'tag': case 'tags': type = 'tag'; ref = dec(seg[1]); break
    case 'pull': case 'pulls': case 'pull-requests': type = 'pr'; ref = seg[1] ? '#' + seg[1] : ''; break
    case 'merge_requests': type = 'pr'; ref = seg[1] ? '!' + seg[1] : ''; break
    case 'issues': type = 'issue'; ref = seg[1] ? '#' + seg[1] : ''; break
    case 'commit': case 'commits': type = 'commit'; ref = (seg[1] || '').slice(0, 7); break
    case 'tree': case 'branch': type = 'branch'; ref = dec(seg.slice(1).join('/')); break
    case 'src': // Gitea: src/branch|tag|commit/<ref>; Bitbucket: src/<branch>/<path>
      if (seg[1] === 'branch') { type = 'branch'; ref = dec(seg.slice(2).join('/')) }
      else if (seg[1] === 'tag') { type = 'tag'; ref = dec(seg.slice(2).join('/')) }
      else if (seg[1] === 'commit') { type = 'commit'; ref = (seg[2] || '').slice(0, 7) }
      else { type = 'branch'; ref = dec(seg[1]) }
      break
    case 'blob': type = 'file'; ref = dec(seg.slice(1).join('/')); break
    default: type = seg.length ? 'link' : 'repo'; ref = dec(seg.join('/'))
  }
  return { provider, repo, type, ref, url: u.href }
}

// Azure DevOps URLs: .../{project}/_git/{repo}/pullrequest/{id} | /commit/{sha} |
// ?version=GB{branch}/GT{tag}/GC{sha}; work items live under /_workitems/edit/{id}.
function parseAzureScm(u, host, parts, dec) {
  let repo = '', type = 'repo', ref = ''
  const gi = parts.indexOf('_git')
  const wi = parts.indexOf('_workitems')
  if (gi >= 0) {
    const project = gi >= 1 ? parts[gi - 1] : ''
    repo = [project, parts[gi + 1]].filter(Boolean).join('/') || host
    const res = parts[gi + 2], id = parts[gi + 3]
    if (res === 'pullrequest') { type = 'pr'; ref = id ? '#' + id : '' }
    else if (res === 'commit') { type = 'commit'; ref = (id || '').slice(0, 7) }
    else {
      const ver = u.searchParams.get('version') || '' // GB<branch> / GT<tag> / GC<sha>
      if (/^GB/.test(ver)) { type = 'branch'; ref = dec(ver.slice(2)) }
      else if (/^GT/.test(ver)) { type = 'tag'; ref = dec(ver.slice(2)) }
      else if (/^GC/.test(ver)) { type = 'commit'; ref = ver.slice(2, 9) }
    }
  } else if (wi >= 0) {
    repo = parts.slice(0, wi).join('/') || host
    type = 'issue'; ref = parts[wi + 2] ? '#' + parts[wi + 2] : '' // .../_workitems/edit/{id}
  } else {
    repo = parts.slice(0, 2).join('/') || host
    type = parts.length > 2 ? 'link' : 'repo'
  }
  return { provider: 'azure', repo, type, ref, url: u.href }
}

// Strip common markdown to plain prose (keeps line breaks). Used to render the
// bodies of synced/imported items (releases, issues, PRs) that arrive as markdown.
export function stripMarkdown(s) {
  return (s || '')
    .replace(/<!--[\s\S]*?-->/g, '')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, '')
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/^[ \t]*#{1,6}[ \t]*/gm, '')
    .replace(/^[ \t]*>[ \t]?/gm, '')
    .replace(/(\*\*|__|\*|`|~~)/g, '')
    .replace(/[ \t]+/g, ' ')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
}

function loadSettings() {
  const def = JSON.parse(JSON.stringify(SETTINGS_DEFAULTS))
  try {
    const raw = JSON.parse(localStorage.getItem(SETTINGS_KEY))
    if (raw) {
      // markers: use stored list, else migrate the old fixed legend labels.
      let markers = def.markers
      // Existing markers with no explicit fill default to filled.
      if (Array.isArray(raw.markers) && raw.markers.length) markers = raw.markers.map(m => ({ ...m, fill: m.fill === undefined ? true : m.fill }))
      else if (raw.legend) markers = [
        { shape: 'diamond', label: raw.legend.milestone || 'Milestone' },
        { shape: 'circle', label: raw.legend.circle || 'Circle' },
        { shape: 'cone', label: raw.legend.cone || 'Cone' },
        { shape: 'flag', label: raw.legend.flag || 'Flag' },
      ]
      let eventLabel = def.eventLabel
      if (typeof raw.eventLabel === 'string') eventLabel = raw.eventLabel
      else if (raw.legend && raw.legend.event) eventLabel = raw.legend.event
      return {
        monthHighlight: { ...def.monthHighlight, ...(raw.monthHighlight || {}) },
        dayLine: { ...def.dayLine, ...(raw.dayLine || {}) },
        weekNumbers: { ...def.weekNumbers, ...(raw.weekNumbers || {}) },
        monthLines: { ...def.monthLines, ...(raw.monthLines || {}) },
        weekLines: { ...def.weekLines, ...(raw.weekLines || {}) },
        items: { ...def.items, ...(raw.items || {}) },
        layout: { ...def.layout, ...(raw.layout || {}) },
        markers,
        eventLabel,
        theme: (raw.theme === 'dark' || raw.theme === 'light') ? raw.theme : def.theme,
      }
    }
  } catch { /* ignore */ }
  return def
}
export const settings = reactive(loadSettings())
watch(settings, (v) => {
  try { localStorage.setItem(SETTINGS_KEY, JSON.stringify(v)) } catch { /* ignore */ }
}, { deep: true })

// Apply the light/dark theme to <html> (CSS variables switch via [data-theme]).
function applyTheme(t) {
  if (typeof document !== 'undefined') {
    document.documentElement.setAttribute('data-theme', t === 'dark' ? 'dark' : 'light')
  }
}
watch(() => settings.theme, applyTheme, { immediate: true })
export function toggleTheme() {
  settings.theme = settings.theme === 'dark' ? 'light' : 'dark'
}

// Restore all view settings (today indicator, gridlines, CW, item style, legend
// labels) to their built-in defaults. Persists via the watch above.
export function resetSettings() {
  const d = JSON.parse(JSON.stringify(SETTINGS_DEFAULTS))
  for (const k of Object.keys(d)) {
    if (k === 'theme') continue   // keep the user's current light/dark choice
    settings[k] = d[k]
  }
}

// Shared, editor-managed colour palette (persisted in the backend DB). The
// PRESET_COLORS are just the default seed — every colour here can be removed.
export const palette = reactive({ colors: [] })
// The colours offered in every area colour picker (deduped).
export const swatchColors = computed(() => {
  const seen = new Set()
  const out = []
  for (const c of palette.colors) {
    const k = String(c).toLowerCase()
    if (!seen.has(k)) { seen.add(k); out.push(c) }
  }
  return out
})

// Item groups (shared, persisted) + transient UI state for group highlighting.
export const groups = reactive({ list: [] })
export const ui = reactive({ hoverGroupId: null, focusItemId: null })

// Baselines (P2): named snapshots + a diff against the live plan.
export const baselines = reactive({
  list: [],
  activeId: null,  // null = Live
  activeItems: [], // snapshot items of the active baseline
})

const _bnorm = v => v ?? ''

export const baselineDiff = computed(() => {
  const out = { active: !!baselines.activeId, status: {}, ghosts: [], counts: { added: 0, moved: 0, removed: 0 } }
  if (!out.active) return out
  const baseById = {}
  for (const b of baselines.activeItems) baseById[b.id] = b
  const liveIds = new Set(store.milestones.map(m => m.id))
  for (const m of store.milestones) {
    const b = baseById[m.id]
    if (!b) { out.status[m.id] = 'added'; out.counts.added++; continue }
    const moved = m.year !== b.year || m.month !== b.month ||
      _bnorm(m.when) !== _bnorm(b.when) ||
      _bnorm(m.startDate) !== _bnorm(b.startDate) ||
      _bnorm(m.endDate) !== _bnorm(b.endDate)
    out.status[m.id] = moved ? 'moved' : 'unchanged'
    if (moved) out.counts.moved++
  }
  for (const b of baselines.activeItems) {
    if (!liveIds.has(b.id)) { out.ghosts.push({ ...b, ghostType: 'removed' }); out.counts.removed++ }
    else if (out.status[b.id] === 'moved') out.ghosts.push({ ...b, ghostType: 'moved' })
  }
  return out
})

// ── Dependency risk ("order violated") ───────────────────────────────────────
// A link {a, b} means "a depends on b" (b is a prerequisite / sub-milestone of a).
// An item is at risk when one of its prerequisites is scheduled strictly AFTER it.
const itemDate = (m) => m.when || m.startDate || `${m.year}-${String(m.month).padStart(2, '0')}-01`
export const riskWarnings = computed(() => {
  const byId = {}
  for (const m of store.milestones) byId[m.id] = m
  const out = []
  for (const m of store.milestones) {
    const md = itemDate(m)
    const late = []
    for (const l of store.links) {
      if (l.a !== m.id) continue
      const p = byId[l.b]
      if (p && itemDate(p) > md) late.push(p)
    }
    if (late.length) out.push({ item: m, lateDeps: late })
  }
  return out
})
export const riskIds = computed(() => new Set(riskWarnings.value.map(w => w.item.id)))
export const riskByItem = computed(() => {
  const map = {}
  for (const w of riskWarnings.value) map[w.item.id] = w.lateDeps
  return map
})

export async function loadBaselines() {
  const list = await api.listBaselines()
  // Newest first → oldest at the bottom.
  list.sort((a, b) => String(b.createdAt || '').localeCompare(String(a.createdAt || '')))
  baselines.list = list
}

export async function loadPalette() {
  try {
    const p = await api.getPalette()
    // null = never configured → seed the built-in defaults; [] = deliberately empty.
    palette.colors = (p.colors == null) ? [...PRESET_COLORS] : p.colors
  } catch {
    palette.colors = [...PRESET_COLORS]
  }
}

export async function loadGroups() {
  try {
    const g = await api.getGroups()
    groups.list = g.groups || []
  } catch { groups.list = [] }
}

function onWriteError(err) {
  console.error('ATLAS write failed:', err)
  // Discard the optimistic change by re-syncing from the server.
  loadPlan().catch(() => {})
}

export async function loadPlan() {
  const plan = await api.getPlan()
  store.swimlanes = plan.swimlanes || []
  store.milestones = plan.milestones || []
  store.links = plan.links || []
  store.loaded = true
}

// ── Import / Export (portable JSON, the shared wire format) ──────────────────

// exportPlanToFile downloads the whole plan as a JSON envelope (backup / move /
// hand to a colleague). Reuses the same format the live-share feed will serve.
export async function exportPlanToFile() {
  const env = await api.exportPlan()
  const blob = new Blob([JSON.stringify(env, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `atlas-export-${new Date().toISOString().slice(0, 10)}.json`
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

// importPlanFromFile parses a JSON export and adds it to the current plan
// (Copy-mode: new IDs, editable), then reloads. Returns the created counts.
export async function importPlanFromFile(file) {
  const text = await file.text()
  let env
  try { env = JSON.parse(text) } catch { throw new Error('Not a valid JSON file') }
  const summary = await api.importPlan(env)
  await loadPlan()
  try { await loadGroups() } catch { /* groups are non-fatal */ }
  return summary
}

// initApp resolves auth + settings, then loads the plan. Called once on mount.
export async function initApp() {
  session.error = null
  session.ready = false
  try {
    const me = await api.me()
    session.authenticated = !!me.authenticated
  } catch { /* treat as anonymous */ }

  try {
    const pr = await api.getPublicRead()
    session.publicReadEnabled = !!pr.enabled
  } catch (e) {
    // 401 here means public read is off and we're anonymous.
    if (e.status === 401) session.publicReadEnabled = false
  }

  try {
    await loadPlan()
  } catch (e) {
    session.error = e.status === 401 ? 'auth-required' : (e.message || 'Failed to load')
  }
  if (!session.error) {
    try { await loadBaselines() } catch { /* baselines non-fatal */ }
    await loadPalette()
    await loadGroups()
  }
  session.ready = true
}

export function useAppStore() {
  function persistYear() {
    try { localStorage.setItem(YEAR_KEY, String(store.year)) } catch { /* ignore */ }
  }
  function prevYear() { store.year--; persistYear() }
  function nextYear() { store.year++; persistYear() }

  // ── Swimlanes ───────────────────────────────────────────────────────────
  function addSwimlane(name, color) {
    const sw = { id: uid(), name, color, subLanes: [] }
    store.swimlanes.push(sw)
    api.createSwimlane({ id: sw.id, name, color }).catch(onWriteError)
    return sw
  }
  function updateSwimlane(id, patch) {
    const s = store.swimlanes.find(s => s.id === id)
    if (s) Object.assign(s, patch)
    api.updateSwimlane(id, patch).catch(onWriteError)
  }
  function deleteSwimlane(id) {
    const msIds = store.milestones.filter(m => m.swimlaneId === id).map(m => m.id)
    store.swimlanes = store.swimlanes.filter(s => s.id !== id)
    store.milestones = store.milestones.filter(m => m.swimlaneId !== id)
    store.links = store.links.filter(l => !msIds.includes(l.a) && !msIds.includes(l.b))
    api.deleteSwimlane(id).catch(onWriteError)
  }
  function moveSwimlane(id, dir) {
    const i = store.swimlanes.findIndex(s => s.id === id)
    if (i < 0) return
    const j = i + dir
    if (j < 0 || j >= store.swimlanes.length) return
    const tmp = store.swimlanes[i]
    store.swimlanes[i] = store.swimlanes[j]
    store.swimlanes[j] = tmp
    api.moveSwimlane(id, dir).catch(onWriteError)
  }
  function setLaneHidden(id, hidden) {
    const s = store.swimlanes.find(s => s.id === id)
    if (s) s.hidden = hidden
    api.setSwimlaneHidden(id, hidden).catch(onWriteError)
  }
  // Drag & drop reorder: moveSwimlaneTo splices locally (live preview),
  // commitSwimlaneOrder persists the final order in one call.
  function moveSwimlaneTo(from, to) {
    const arr = store.swimlanes
    if (from === to || from < 0 || to < 0 || from >= arr.length || to >= arr.length) return
    const [it] = arr.splice(from, 1)
    arr.splice(to, 0, it)
  }
  function commitSwimlaneOrder() {
    api.reorderSwimlanes(store.swimlanes.map(s => s.id)).catch(onWriteError)
  }
  function moveSubLaneTo(swimlaneId, from, to) {
    const sl = store.swimlanes.find(s => s.id === swimlaneId)
    if (!sl) return
    const arr = sl.subLanes
    if (from === to || from < 0 || to < 0 || from >= arr.length || to >= arr.length) return
    const [it] = arr.splice(from, 1)
    arr.splice(to, 0, it)
  }
  function commitSubLaneOrder(swimlaneId) {
    const sl = store.swimlanes.find(s => s.id === swimlaneId)
    if (sl) api.reorderSubLanes(sl.subLanes.map(s => s.id)).catch(onWriteError)
  }

  // ── Sub-lanes ───────────────────────────────────────────────────────────
  function addSubLane(swimlaneId, name) {
    const sl = store.swimlanes.find(s => s.id === swimlaneId)
    if (!sl) return
    const sub = { id: uid(), name }
    sl.subLanes.push(sub)
    api.createSubLane(swimlaneId, { id: sub.id, name }).catch(onWriteError)
    return sub
  }
  function updateSubLane(swimlaneId, subId, name) {
    const sl = store.swimlanes.find(s => s.id === swimlaneId)
    const sub = sl?.subLanes.find(s => s.id === subId)
    if (sub) sub.name = name
    api.updateSubLane(subId, name).catch(onWriteError)
  }
  function deleteSubLane(swimlaneId, subId) {
    const sl = store.swimlanes.find(s => s.id === swimlaneId)
    if (sl) sl.subLanes = sl.subLanes.filter(s => s.id !== subId)
    const msIds = store.milestones
      .filter(m => m.swimlaneId === swimlaneId && m.subLaneId === subId)
      .map(m => m.id)
    store.milestones = store.milestones.filter(
      m => !(m.swimlaneId === swimlaneId && m.subLaneId === subId)
    )
    store.links = store.links.filter(l => !msIds.includes(l.a) && !msIds.includes(l.b))
    api.deleteSubLane(subId).catch(onWriteError)
  }

  // ── Milestones / items ────────────────────────────────────────────────────
  function addMilestone(data) {
    const m = { id: uid(), kind: 'milestone', marker: 'l:Flag', ...data }
    store.milestones.push(m)
    api.createItem(m).catch(onWriteError)
    return m
  }
  function updateMilestone(id, data) {
    const m = store.milestones.find(m => m.id === id)
    if (m) Object.assign(m, data)
    api.updateItem(id, m || data).catch(onWriteError)
  }
  function deleteMilestone(id) {
    store.milestones = store.milestones.filter(m => m.id !== id)
    store.links = store.links.filter(l => l.a !== id && l.b !== id)
    // Drop the item from any groups it belonged to.
    let groupsChanged = false
    for (const g of groups.list) {
      if (g.itemIds?.includes(id)) { g.itemIds = g.itemIds.filter(i => i !== id); groupsChanged = true }
    }
    if (groupsChanged) api.setGroups(groups.list).catch(() => {})
    api.deleteItem(id).catch(onWriteError)
  }

  // ── Links ─────────────────────────────────────────────────────────────────
  function addLink(idA, idB) {
    if (idA === idB) return
    const exists = store.links.some(l =>
      (l.a === idA && l.b === idB) || (l.a === idB && l.b === idA)
    )
    if (exists) return
    store.links.push({ a: idA, b: idB })
    api.addLink(idA, idB).catch(onWriteError)
  }
  function removeLink(idA, idB) {
    store.links = store.links.filter(l =>
      !((l.a === idA && l.b === idB) || (l.a === idB && l.b === idA))
    )
    api.removeLink(idA, idB).catch(onWriteError)
  }
  function getLinkedIds(id) {
    return new Set(
      store.links
        .filter(l => l.a === id || l.b === id)
        .map(l => l.a === id ? l.b : l.a)
    )
  }
  // Directed: the prerequisites this item depends on (links where a === id).
  function dependsOnIds(id) {
    return new Set(store.links.filter(l => l.a === id).map(l => l.b))
  }
  // Directed: the items that depend on this one — its "parents" (links where b === id).
  function dependentIds(id) {
    return new Set(store.links.filter(l => l.b === id).map(l => l.a))
  }

  function cellMilestones(swimlaneId, subLaneId, month) {
    return store.milestones.filter(m =>
      m.swimlaneId === swimlaneId &&
      m.subLaneId  === subLaneId  &&
      m.month      === month      &&
      m.year       === store.year
    )
  }

  // ── Session / settings ────────────────────────────────────────────────────
  async function login(username, password) {
    await api.login(username, password)
    session.authenticated = true
    session.error = null
    try {
      const pr = await api.getPublicRead()
      session.publicReadEnabled = !!pr.enabled
    } catch { /* ignore */ }
    await loadPlan()
  }
  async function logout() {
    try { await api.logout() } finally { session.authenticated = false }
    try {
      await loadPlan()
    } catch (e) {
      if (e.status === 401) session.error = 'auth-required'
    }
  }
  async function setPublicRead(enabled) {
    await api.setPublicRead(enabled)
    session.publicReadEnabled = enabled
  }

  // ── Shared colour palette ───────────────────────────────────────────────────
  function addPaletteColor(hex) {
    if (!hex) return
    const k = hex.toLowerCase()
    if (palette.colors.some(c => c.toLowerCase() === k)) return
    palette.colors.push(hex)
    api.setPalette(palette.colors).catch(() => loadPalette())
  }
  function removePaletteColor(hex) {
    const k = hex.toLowerCase()
    palette.colors = palette.colors.filter(c => c.toLowerCase() !== k)
    api.setPalette(palette.colors).catch(() => loadPalette())
  }
  function resetPalette() {
    palette.colors = [...PRESET_COLORS]
    api.setPalette(palette.colors).catch(() => loadPalette())
  }

  // ── Item groups ─────────────────────────────────────────────────────────────
  function persistGroups() { api.setGroups(groups.list).catch(() => loadGroups()) }
  function addGroup(name, color) {
    const g = { id: uid(), name: name?.trim() || 'Group', color: color || PRESET_COLORS[0], itemIds: [] }
    groups.list.push(g)
    persistGroups()
    return g
  }
  function updateGroup(id, patch) {
    const g = groups.list.find(x => x.id === id)
    if (g) Object.assign(g, patch)
    persistGroups()
  }
  function deleteGroup(id) {
    groups.list = groups.list.filter(g => g.id !== id)
    persistGroups()
  }
  function toggleItemGroup(groupId, itemId) {
    const g = groups.list.find(x => x.id === groupId)
    if (!g) return
    if (!g.itemIds) g.itemIds = []
    g.itemIds = g.itemIds.includes(itemId)
      ? g.itemIds.filter(i => i !== itemId)
      : [...g.itemIds, itemId]
    persistGroups()
  }
  function itemGroupIds(itemId) {
    return groups.list.filter(g => (g.itemIds || []).includes(itemId)).map(g => g.id)
  }
  function setItemGroups(itemId, groupIdList) {
    const want = new Set(groupIdList)
    let changed = false
    for (const g of groups.list) {
      const has = (g.itemIds || []).includes(itemId)
      const should = want.has(g.id)
      if (should && !has) { g.itemIds = [...(g.itemIds || []), itemId]; changed = true }
      else if (!should && has) { g.itemIds = g.itemIds.filter(i => i !== itemId); changed = true }
    }
    if (changed) persistGroups()
  }

  // ── Baselines (P2) ────────────────────────────────────────────────────────
  async function selectBaseline(id) {
    if (!id) {
      baselines.activeId = null
      baselines.activeItems = []
      return
    }
    const b = await api.getBaseline(id)
    baselines.activeId = id
    baselines.activeItems = b.items || []
  }
  async function createBaseline(name) {
    const b = await api.createBaseline(name)
    await loadBaselines()
    return b
  }
  async function deleteBaseline(id) {
    await api.deleteBaseline(id)
    if (baselines.activeId === id) {
      baselines.activeId = null
      baselines.activeItems = []
    }
    await loadBaselines()
  }

  return {
    store, session, baselines,
    prevYear, nextYear,
    addSwimlane, updateSwimlane, deleteSwimlane, moveSwimlane, setLaneHidden, moveSwimlaneTo, commitSwimlaneOrder, moveSubLaneTo, commitSubLaneOrder,
    addSubLane, updateSubLane, deleteSubLane,
    addMilestone, updateMilestone, deleteMilestone,
    addLink, removeLink, getLinkedIds, dependsOnIds, dependentIds,
    cellMilestones,
    login, logout, setPublicRead, loadPlan,
    loadBaselines, selectBaseline, createBaseline, deleteBaseline,
    addPaletteColor, removePaletteColor, resetPalette,
    addGroup, updateGroup, deleteGroup, toggleItemGroup, itemGroupIds, setItemGroups,
  }
}
