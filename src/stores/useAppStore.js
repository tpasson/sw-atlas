import { reactive, computed, watch } from 'vue'
import { api, setWorkspaceSlug } from '../api.js'
import { LUCIDE_MARKER_SHAPES } from '../lucideMarkers.js'

const IS_DEMO = !!import.meta.env.VITE_DEMO
// Top-level URL segments that are never workspace slugs.
const RESERVED_SLUGS = new Set(['api', 'assets', 'favicon.svg', 'w', 'health', 'index.html', 'explore', 'shared', 'projects'])

// workspaceSlugFromUrl reads the first path segment of the current URL as the
// workspace slug ('' = home/default). Never applies in the demo (single sandbox).
function workspaceSlugFromUrl() {
  if (IS_DEMO) return ''
  const seg = (window.location.pathname.split('/').filter(Boolean)[0] || '').toLowerCase()
  return RESERVED_SLUGS.has(seg) ? '' : seg
}

export const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

// Typed relationships for links (R1). For an edge a→b: `label` reads it from a's
// side, `inverse` from b's side. 'depends-on' keeps the legacy blocking wording.
// scheduling:true marks a temporal relationship — it only makes sense between
// two timeline items (dated). The rest are traceability relations, valid across
// families (incl. timeline ↔ backlog). See relationsFor() for the family rule.
export const RELATIONSHIP_TYPES = [
  { key: 'depends-on', label: 'Blocked by', inverse: 'Blocks', scheduling: true },
  { key: 'uses', label: 'Uses', inverse: 'Used by' },
  { key: 'relates-to', label: 'Relates to', inverse: 'Relates to' },
  { key: 'child-of', label: 'Child of', inverse: 'Parent of' },
  { key: 'implements', label: 'Implements', inverse: 'Implemented by' },
  { key: 'verifies', label: 'Verifies', inverse: 'Verified by' },
]

// Whether an item sits on the timeline (schedulable) vs. off-timeline backlog —
// decided by its type's behavior family. Unknown types default to schedulable
// (legacy items are milestones).
export function isSchedulableItem(item) {
  const fam = itemTypeByKey(item?.typeKey || item?.kind)?.family
  return fam !== 'work-item' && fam !== 'container'
}

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
const GRAN_KEY = 'atlas-view-gran'
const MONTH_KEY = 'atlas-view-month'
const VIEW_KEY = 'atlas-view-mode'
const SETTINGS_SECTION_KEY = 'atlas-settings-section'
const NAV_VIEWS = ['explorer', 'scm', 'cr', 'settings', 'project-settings'] // non-default views carried in the URL
function initialView() {
  const v = localStorage.getItem(VIEW_KEY)
  return NAV_VIEWS.includes(v) ? v : 'timeline'
}
function initialSettingsSection() {
  return localStorage.getItem(SETTINGS_SECTION_KEY) || 'areas'
}
function initialYear() {
  const v = parseInt(localStorage.getItem(YEAR_KEY) || '', 10)
  return Number.isFinite(v) ? v : new Date().getFullYear()
}
function initialGranularity() {
  return localStorage.getItem(GRAN_KEY) === 'month' ? 'month' : 'year'
}
function initialMonth() {
  const v = parseInt(localStorage.getItem(MONTH_KEY) || '', 10)
  return v >= 1 && v <= 12 ? v : new Date().getMonth() + 1
}

// Reactive plan cache, populated from the backend (the single source of truth).
export const store = reactive({
  year: initialYear(),
  granularity: initialGranularity(), // 'year' (12 month columns) | 'month' (day columns)
  viewMonth: initialMonth(),         // 1..12, the focused month when granularity === 'month'
  view: initialView(),               // 'timeline' | 'explorer' | 'scm'
  swimlanes: [],
  milestones: [],
  links: [],
  loaded: false,
})

// Session/auth + global settings state.
export const session = reactive({
  authenticated: false,
  username: null,
  role: null, // 'admin' | 'user' | null
  email: '',
  firstName: '',
  lastName: '',
  publicReadEnabled: true,
  publicCREnabled: false,
  ready: false,
  error: null, // 'auth-required' | 'private' | 'not-found' | message string | null
})

// Which workspace the SPA is viewing, and whether it's the logged-in user's own
// (only then is editing unlocked). mode 'landing' = the discovery directory at
// '/', 'plan' = a specific workspace at /{slug}. ownSlug = the logged-in user's
// own workspace (for the "My plan" link).
export const workspace = reactive({
  mode: 'plan',
  slug: '',
  isOwn: false,        // viewing my personal /{username} home workspace
  ownSlug: '',
  role: null,          // my role in the viewed workspace: owner | editor | viewer | null
  myWorkspaces: [],    // [{slug, name, role, visibility}] for the switcher
  members: [],         // roster of the viewed workspace: [{userId, username, role}]
})

// A person's display name: "First Last" if set, else the username.
export function personName(p) {
  if (!p) return ''
  const n = `${p.firstName || ''} ${p.lastName || ''}`.trim()
  return n || p.username || ''
}
// Initials from the real name if present, else the username.
export function personInitials(p) {
  if (!p) return ''
  const f = (p.firstName || '').trim(), l = (p.lastName || '').trim()
  if (f || l) return ((f[0] || '') + (l[0] || '')).toUpperCase()
  return ((p.username || '').trim()[0] || '?').toUpperCase()
}

// Resolve an assignee id to its member / initials / name (for X1 avatars).
export function memberById(id) { return id ? workspace.members.find(m => m.userId === id) || null : null }
export function memberInitials(id) { return personInitials(memberById(id)) }
export function memberName(id) { return personName(memberById(id)) }
export async function loadWorkspaceMembers() {
  try { workspace.members = await api.workspaceMembers() } catch { workspace.members = [] }
}

// Can the current user edit CONTENT in the viewed workspace (owner or editor)?
export function canEditWorkspace() {
  // Site admins are superusers: they can edit any workspace, not just their own.
  return session.authenticated && (session.role === 'admin' || workspace.role === 'owner' || workspace.role === 'editor')
}

// Can the current user administer the workspace CONFIGURATION (types, display,
// sources, sharing, change-request decisions, members)? Owner — or a site admin.
export function canAdminWorkspace() {
  return session.authenticated && (session.role === 'admin' || workspace.role === 'owner')
}

// Can the current user PROPOSE a change (Change Request)? Only people who can't
// edit directly — viewers (members) and, when the project opts in via public CRs,
// anyone (even without an account). Owners/editors/admins edit directly instead.
export function canProposeChanges() {
  return !baselines.activeId && !canEditWorkspace() && ((session.authenticated && !!workspace.role) || session.publicCREnabled)
}

// Appearance settings are stored per-workspace on the server (so a plan looks the
// same on any device and renders the owner's design for viewers). SETTINGS_KEY is
// now just a local cache for instant first paint; THEME_KEY keeps the light/dark
// choice per-browser (a viewing preference, not a plan property).
const SETTINGS_KEY = 'atlas-ui-settings'
const THEME_KEY = 'atlas-theme'
const SETTINGS_DEFAULTS = {
  monthHighlight: { enabled: true, color: '#0A84FF', opacity: 0.06 },
  dayLine: { enabled: true, color: '#FF3B30', opacity: 0.7, width: 2 },
  weekNumbers: { enabled: true },
  monthLines: { enabled: false, color: '#E5E5EA', opacity: 1, width: 1 },
  weekLines: { enabled: true, color: '#8E8E93', opacity: 0.1, width: 1 },
  items: { fontSize: 14, fontWeight: 700, padding: 4, margin: 6, radius: 10, border: 2, labelOffset: 0, iconGap: 6, labelBuffer: 20, borderMode: 'hover', markerSize: 14, markerStroke: 2, eventOpacity: 0.13, maturitySize: 5, density: 'stack', densityRows: 3 },
  markers: [
    { shape: 'l:Diamond', label: 'Milestone', fill: true },
    { shape: 'l:Circle', label: 'Circle', fill: true },
    { shape: 'l:Triangle', label: 'Decision', fill: true },
    { shape: 'l:Star', label: 'Highlight', fill: true },
  ],
  eventLabel: 'Event (duration)',
  layout: { subAreaWidth: 240, areaWidth: 168 },
  theme: 'light',
}

// Fixed 4-stage maturity scale (mirrors the ATLAS 4-square logo).
export const MATURITY_STAGES = ['Concept', 'Design', 'Production', 'Series']

// Status tones: you pick a MEANING, the colour follows — so "Approved" can't be
// red. Consistent across every type (green = good, red = bad, everywhere).
export const STATUS_TONES = [
  { key: 'neutral', label: 'Neutral', color: '#7C89A6' },
  { key: 'info', label: 'Info', color: '#0A84FF' },
  { key: 'progress', label: 'In progress', color: '#5E5CE6' },
  { key: 'positive', label: 'Positive', color: '#30D158' },
  { key: 'warning', label: 'Warning', color: '#FF9F0A' },
  { key: 'negative', label: 'Negative', color: '#FF453A' },
]
export const toneColor = (tone) => (STATUS_TONES.find(t => t.key === tone) || STATUS_TONES[0]).color
// A status's display colour: a custom per-status colour if set, else its tone's.
export const statusColor = (s) => (s && s.color) ? s.color : toneColor(s && s.tone)

// A market-standard default workflow, seeded when a type opts into statuses.
export const DEFAULT_STATUSES = [
  { key: 'todo', label: 'To Do', tone: 'neutral', to: ['in-progress', 'cancelled'] },
  { key: 'in-progress', label: 'In Progress', tone: 'progress', to: ['blocked', 'done', 'cancelled'] },
  { key: 'blocked', label: 'Blocked', tone: 'warning', to: ['in-progress', 'cancelled'] },
  { key: 'done', label: 'Done', tone: 'positive', to: ['in-progress'] },
  { key: 'cancelled', label: 'Cancelled', tone: 'negative', to: ['todo'] },
]
// Resolve a status object for an item, from its type's status list. If the type
// has statuses but the item has none (or an unknown one), fall back to the start
// status (the first) — a status-typed item always shows a status.
export function itemStatus(item) {
  if (!item) return null
  const sts = itemTypeByKey(item.typeKey || item.kind)?.statuses || []
  if (!sts.length) return null
  return sts.find(s => s.key === item.status) || sts[0]
}

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
// Theme is a per-browser preference, separate from the server-stored appearance.
try { const t = localStorage.getItem(THEME_KEY); if (t === 'dark' || t === 'light') settings.theme = t } catch { /* ignore */ }

let uiSaveTimer = null
function saveUISettings() {
  const v = JSON.parse(JSON.stringify(settings))
  delete v.theme // theme stays local
  api.setInstanceUISettings(v).catch(() => { /* best-effort */ })
}
watch(settings, (v) => {
  try { localStorage.setItem(THEME_KEY, v.theme) } catch { /* ignore */ }
  // Cache the appearance locally for instant first paint (everyone).
  try { localStorage.setItem(SETTINGS_KEY, JSON.stringify(v)) } catch { /* ignore */ }
  if (IS_DEMO) return
  // Display is now GLOBAL (instance-wide): only a SITE ADMIN persists it, and only
  // from the Admin panel. Everyone else just renders with the admin's config.
  if (session.role === 'admin') {
    clearTimeout(uiSaveTimer)
    uiSaveTimer = setTimeout(saveUISettings, 600)
  }
}, { deep: true })

// mergeSettings applies a server-stored appearance object over the live settings
// (theme excluded — it's per-browser). Mirrors the merge in loadSettings().
function mergeSettings(raw) {
  if (!raw || typeof raw !== 'object') return
  const d = SETTINGS_DEFAULTS
  if (raw.monthHighlight) settings.monthHighlight = { ...d.monthHighlight, ...raw.monthHighlight }
  if (raw.dayLine) settings.dayLine = { ...d.dayLine, ...raw.dayLine }
  if (raw.weekNumbers) settings.weekNumbers = { ...d.weekNumbers, ...raw.weekNumbers }
  if (raw.monthLines) settings.monthLines = { ...d.monthLines, ...raw.monthLines }
  if (raw.weekLines) settings.weekLines = { ...d.weekLines, ...raw.weekLines }
  if (raw.items) settings.items = { ...d.items, ...raw.items }
  if (raw.layout) settings.layout = { ...d.layout, ...raw.layout }
  if (Array.isArray(raw.markers) && raw.markers.length) settings.markers = raw.markers.map(m => ({ ...m, fill: m.fill === undefined ? true : m.fill }))
  if (typeof raw.eventLabel === 'string') settings.eventLabel = raw.eventLabel
}
function resetAppearance() {
  const d = JSON.parse(JSON.stringify(SETTINGS_DEFAULTS))
  for (const k of Object.keys(d)) { if (k !== 'theme') settings[k] = d[k] }
}

// loadUISettings loads the GLOBAL (instance-wide) Display config from the server;
// every dashboard renders with it. If none is set yet, a site admin seeds it from
// the current defaults; everyone else falls back to defaults.
export async function loadUISettings() {
  if (IS_DEMO) return
  let r
  try { r = await api.getInstanceUISettings() } catch { return }
  if (r && r.settings) mergeSettings(r.settings)
  else if (session.role === 'admin') saveUISettings()
  else resetAppearance()
}

// Apply the light/dark theme to <html> (CSS variables switch via [data-theme]).
function applyTheme(t) {
  if (typeof document !== 'undefined') {
    const el = document.documentElement
    el.setAttribute('data-theme', t === 'dark' ? 'dark' : 'light')
    // Keep the html background in sync with the theme (matches theme-init.js, and
    // avoids a stale colour on overscroll after toggling).
    try { el.style.backgroundColor = getComputedStyle(el).getPropertyValue('--clr-bg').trim() } catch { /* ignore */ }
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
// focusItemId/focusVersion: transient "jump to this item on the timeline" signal.
// explorerItemId/explorerItemVersion: the Explorer's currently open item — kept in
// the URL (?item=&v=) so browser back/forward restores it.
// Explorer view mode (Tree / Table / Board) — shared so the switcher can live in
// the top header while ExplorerView reads it. Persisted per browser.
export const EXPLORER_MODES = [{ key: 'folders', label: 'Tree' }, { key: 'table', label: 'Table' }, { key: 'board', label: 'Board' }]
const EXPLORER_MODE_KEY = 'atlas-explorer-mode'
function initialExplorerMode() {
  const v = localStorage.getItem(EXPLORER_MODE_KEY)
  return ['folders', 'table', 'board'].includes(v) ? v : 'folders'
}
export function setExplorerMode(k) {
  ui.explorerMode = k
  try { localStorage.setItem(EXPLORER_MODE_KEY, k) } catch { /* ignore */ }
}

export const ui = reactive({ hoverGroupId: null, focusItemId: null, focusVersion: null, explorerItemId: null, explorerItemVersion: null, highlightIds: null, settingsSection: initialSettingsSection(), focusCrId: null, explorerMode: initialExplorerMode() })

// A reference-field value can pin a specific version of its target, encoded as
// "<id>@v<n>". parseRef splits that back into { id, version } (version = null when
// the reference points at the live/head item). Plain ids pass through unchanged.
export function parseRef(v) {
  const m = /^(.*)@v(\d+)$/.exec(String(v ?? ''))
  return m ? { id: m[1], version: parseInt(m[2], 10) } : { id: String(v ?? ''), version: null }
}

// itemLink builds the stable deep-link for an item (optionally pinned to a
// version): {origin}/{slug}?item={id}[&v={n}]. Opening it lands on the item in
// the Explorer's web layout (see openDeepLinkTarget + ExplorerView).
export function itemLink(id, version, fmt) {
  const origin = (typeof window !== 'undefined') ? window.location.origin : ''
  const slug = workspace.slug || ''
  const base = origin + '/' + (slug ? encodeURIComponent(slug) : '')
  const q = new URLSearchParams({ item: id })
  if (version) q.set('v', String(version))
  if (fmt && fmt !== 'form') q.set('fmt', fmt)
  return base + '?' + q.toString()
}

// Global user-profile popover: any clickable user (owner, member, assignee) calls
// openProfile(person, event); a single <UserProfilePopover> renders it at the click.
export const profilePopover = reactive({ person: null, x: 0, y: 0 })
export function openProfile(person, ev) {
  if (!person) return
  profilePopover.person = person
  profilePopover.x = ev ? ev.clientX : 0
  profilePopover.y = ev ? ev.clientY : 0
}
export function closeProfile() { profilePopover.person = null }

// Item-type registry (built-ins + per-workspace custom types). Drives the type
// picker + the dynamic field set in the item modal.
export const itemTypes = reactive({ list: [] })
export function itemTypeByKey(key) {
  return itemTypes.list.find(t => t.key === key) || null
}

// Shared, reusable status workflows. A type may reference one by key; the server
// then resolves that workflow's statuses + layout onto the type (so item
// rendering reads type.statuses/type.layout unchanged). Managed in the type
// editor; edited once, reflected on every type that uses it.
export const workflows = reactive({ list: [] })
export function workflowByKey(key) {
  return workflows.list.find(w => w.key === key) || null
}

// Baselines (P2): named snapshots + a diff against the live plan.
export const baselines = reactive({
  list: [],
  activeId: null,  // null = Live
  activeItems: [], // snapshot items of the active baseline
})

// Change requests: proposed changes pending the owner's decision.
export const changeRequests = reactive({ list: [] })
export const pendingCRCount = computed(() => changeRequests.list.filter(c => c.status === 'pending').length)

export async function loadChangeRequests() {
  if (!session.authenticated) { changeRequests.list = []; return }
  try { changeRequests.list = (await api.listChangeRequests()) || [] }
  catch { changeRequests.list = [] } // not a member / no access
}
// Propose an edit to an existing item (payload = the item's proposed fields).
export async function proposeChange(targetItemId, payload, note = '') {
  await api.createChangeRequest({ kind: 'edit', targetItemId, payload, note })
  await loadChangeRequests()
}
// Propose creating a brand-new item (payload includes a generated id).
export async function proposeCreate(payload, note = '') {
  await api.createChangeRequest({ kind: 'create', payload, note })
  await loadChangeRequests()
}
export async function decideChangeRequest(id, approve, note = '') {
  if (approve) await api.approveChangeRequest(id, note)
  else await api.rejectChangeRequest(id, note)
  await loadChangeRequests()
  if (approve) await loadPlan() // an approved change is now live
}

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

// The items the timeline actually renders: the live plan normally, or — when a
// baseline is selected — that baseline's full snapshot, so you see the plan as it
// stood then (read-only) instead of the live plan with diff overlays.
export const viewItems = computed(() => baselines.activeId ? baselines.activeItems : store.milestones)

// ── Dependency risk ("order violated") ───────────────────────────────────────
// A link {a, b} means "a depends on b" (b is a prerequisite / sub-milestone of a).
// An item is at risk when a prerequisite FINISHES after the item is anchored.
// Date logic is purely field-based (independent of type/icon): a point uses its
// single date; a range uses its start as its anchor and its end as completion.
const fallbackDate = (m) => `${m.year}-${String(m.month).padStart(2, '0')}-01`
// When the item needs its prerequisites ready (its anchor / start).
const dueDate = (m) => m.when || m.startDate || fallbackDate(m)
// When the item itself is complete (a range only counts as done at its end).
const doneDate = (m) => m.when || m.endDate || m.startDate || fallbackDate(m)
export const riskWarnings = computed(() => {
  const byId = {}
  for (const m of store.milestones) byId[m.id] = m
  const out = []
  for (const m of store.milestones) {
    if (m.sourceSystem) continue // SCM items are references — never part of risk calc
    const md = dueDate(m)
    const late = []
    for (const l of store.links) {
      if (l.a !== m.id || (l.rel || 'depends-on') !== 'depends-on') continue
      const p = byId[l.b]
      if (p && doneDate(p) > md) late.push(p)
    }
    if (late.length) out.push({ item: m, lateDeps: late })
  }
  return out
})
export const riskIds = computed(() => new Set(riskWarnings.value.map(w => w.item.id)))

// Calendar-late: a TRACKED item (progress explicitly set) that's past its deadline
// and not finished. Deadline = end (range) or when (point); coarse year/month-only
// placements aren't judged. Mirrors the server's public-plan "late" count exactly.
export const lateItems = computed(() => {
  const today = new Date(); today.setHours(0, 0, 0, 0)
  return store.milestones.filter(m => {
    if (m.sourceSystem) return false // SCM items are read-only references, never "late"
    if (m.progress == null || m.progress >= 100) return false
    const due = m.endDate || m.when
    if (!due) return false
    const d = new Date(due)
    return !isNaN(d) && d < today
  })
})
export const lateIds = computed(() => new Set(lateItems.value.map(m => m.id)))
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

export async function loadItemTypes() {
  try {
    itemTypes.list = await api.itemTypes()
  } catch { itemTypes.list = [] }
}

// saveItemTypes persists the custom types and refreshes the catalog from the
// server response (which re-includes the built-ins).
export async function saveItemTypes(customTypes) {
  itemTypes.list = await api.setItemTypes(customTypes)
}

export async function loadWorkflows() {
  try {
    workflows.list = (api.workflows ? await api.workflows() : []) || []
  } catch { workflows.list = [] }
}

// saveWorkflows persists the shared workflows, then reloads the type catalog so
// each type's resolved statuses/layout reflect the edited workflow.
export async function saveWorkflows(list) {
  workflows.list = await api.setWorkflows(list)
  await loadItemTypes()
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

// Demo only: the seed ships a pre-connected GitHub source (tpasson/sw-atlas) with
// no mirrored content yet. Pull it live from GitHub once (cached in localStorage
// afterwards; a manual "Sync now" refreshes it). Runs in the background so the
// board renders immediately and the area pops in when the fetch resolves.
async function syncSeededDemoSources() {
  try {
    if (!api.listGitHubSources) return
    const { sources } = await api.listGitHubSources()
    const need = (sources || []).filter(s => !store.swimlanes.some(sw => sw.sourceSystem === s.id))
    if (!need.length) return
    await Promise.all(need.map(s => api.syncGitHubSource(s.id).catch(() => {})))
    await loadPlan()
  } catch { /* non-fatal — the demo area stays empty if GitHub is unreachable */ }
}

// goHomeView switches to the discovery landing WITHOUT a full page reload, so it
// doesn't flash white the way window.location.assign('/') does. Browser back/
// forward is reconciled by the popstate reload registered in initApp.
export function goHomeView() {
  if (IS_DEMO) return
  try { history.pushState({ atlasHome: true }, '', '/') } catch { /* ignore */ }
  session.error = null
  workspace.slug = ''
  workspace.mode = 'landing'
}

// applyNav reconciles the view + open item from the URL (?view=&item=&v=), without
// a reload. Used on first load and on browser back/forward, so those states are
// restorable. An ?item with no explicit ?view implies the Explorer (its web layout).
function applyNav() {
  if (typeof window === 'undefined') return
  let q
  try { q = new URL(window.location.href).searchParams } catch { return }
  const id = q.get('item')
  const view = q.get('view')
  const resolved = NAV_VIEWS.includes(view) ? view : (id ? 'explorer' : 'timeline')
  store.view = resolved
  try { localStorage.setItem(VIEW_KEY, resolved) } catch { /* ignore */ }
  if (resolved === 'settings' || resolved === 'project-settings') {
    const sec = q.get('section')
    if (sec) rememberSection(sec)
  }
  const v = parseInt(q.get('v') || '', 10)
  ui.explorerItemId = id || null
  ui.explorerItemVersion = id && Number.isFinite(v) && v > 0 ? v : null
}

// pushNav records a navigable state (view + optional open item) into the URL and
// browser history, so back/forward steps through views and items. Called from user
// gestures (view switch, item click). programmatic/popstate updates use applyNav
// and never push.
export function pushNav({ view, item, version, section, fmt } = {}, replace = false) {
  if (typeof window === 'undefined') return
  const v = view || store.view
  let url
  try {
    url = new URL(window.location.href)
    url.searchParams.delete('view'); url.searchParams.delete('item'); url.searchParams.delete('v'); url.searchParams.delete('section'); url.searchParams.delete('fmt')
    if (v && v !== 'timeline') url.searchParams.set('view', v)
    if (v === 'settings' || v === 'project-settings') { const s = section || ui.settingsSection; if (s) url.searchParams.set('section', s) }
    if (item) { url.searchParams.set('item', item); if (version) url.searchParams.set('v', String(version)); if (fmt && fmt !== 'form') url.searchParams.set('fmt', fmt) }
  } catch { return }
  const href = url.pathname + url.search
  const state = { atlasNav: true, view: v, item: item || null, version: version || null }
  try { replace ? history.replaceState(state, '', href) : history.pushState(state, '', href) } catch { /* ignore */ }
}

// Settings has two entries — Project (workspace config) and General (account /
// appearance / instance). Each remembers its own last section, so reopening an
// entry returns where you were; the current section also rides in the URL.
const SETTINGS_PROJ_KEY = 'atlas-settings-proj-section'
const SETTINGS_GEN_KEY = 'atlas-settings-gen-section'
export const PROJECT_SECTIONS = ['areas', 'groups', 'types', 'workflows', 'baselines', 'data', 'sharing', 'members']
function rememberSection(key) {
  ui.settingsSection = key
  try {
    localStorage.setItem(SETTINGS_SECTION_KEY, key)
    localStorage.setItem(PROJECT_SECTIONS.includes(key) ? SETTINGS_PROJ_KEY : SETTINGS_GEN_KEY, key)
  } catch { /* ignore */ }
}
function gotoSettings(view, section) {
  if (section) rememberSection(section)
  store.view = view
  try { localStorage.setItem(VIEW_KEY, view) } catch { /* ignore */ }
  pushNav({ view, section: ui.settingsSection })
}
// Project settings — its own view (Areas/Groups/Types/…); reopens its last section.
export function openProjectSettings(section) {
  gotoSettings('project-settings', section || localStorage.getItem(SETTINGS_PROJ_KEY) || 'areas')
}
// General settings — its own view (Account/Appearance/Instance); reopens its last section.
export function openGeneralSettings() {
  gotoSettings('settings', localStorage.getItem(SETTINGS_GEN_KEY) || 'account')
}
// Open straight onto an explicit project section (e.g. "Invite people…" → Members).
export function openSettings(section) { openProjectSettings(section) }

// Open the Change Requests view focused on a specific CR (from the Explorer).
export function openChangeRequest(id) {
  ui.focusCrId = id
  store.view = 'cr'
  try { localStorage.setItem(VIEW_KEY, 'cr') } catch { /* ignore */ }
  pushNav({ view: 'cr' })
}
// setSettingsSection changes the active section from the sidebar (within a view).
export function setSettingsSection(key) {
  rememberSection(key)
  if (store.view === 'settings' || store.view === 'project-settings') pushNav({ view: store.view, section: key })
}

// Browser back/forward: within the same plan, reconcile the view/item client-side
// (no flash); leaving the plan (landing, a different plan) falls back to a full
// load so routing + auth re-resolve, as before.
if (typeof window !== 'undefined') {
  window.addEventListener('popstate', () => {
    const target = workspaceSlugFromUrl()
    if (IS_DEMO || (workspace.mode === 'plan' && target === (workspace.slug || ''))) applyNav()
    else window.location.reload()
  })
}

// initApp resolves auth, picks the workspace to view (from the /{slug} URL),
// then loads its plan. Called once on mount.
export async function initApp() {
  session.error = null
  session.ready = false

  let me = { authenticated: false }
  try { me = await api.me() } catch { /* treat as anonymous */ }
  session.authenticated = !!me.authenticated
  session.username = me.username || null
  session.role = me.role || null
  session.email = me.email || ''
  session.firstName = me.firstName || ''
  session.lastName = me.lastName || ''
  workspace.ownSlug = me.workspace || ''

  // Global Display config is instance-wide — load it once, regardless of view.
  loadUISettings()

  // The bare root (outside the demo) is the discovery landing page, not a plan.
  if (!IS_DEMO && !workspaceSlugFromUrl()) {
    workspace.mode = 'landing'
    workspace.slug = ''
    workspace.isOwn = false
    setWorkspaceSlug('')
    // Load the user's plans so the header switcher works on the landing too.
    if (session.authenticated) await loadMyWorkspaces()
    session.ready = true
    return
  }

  workspace.mode = 'plan'
  resolveWorkspaceTarget(me)
  if (session.authenticated) await loadMyWorkspaces()

  try {
    const pr = await api.getPublicRead()
    session.publicReadEnabled = !!pr.enabled
  } catch (e) {
    if (e.status === 401 || e.status === 403) session.publicReadEnabled = false
  }
  try {
    const pc = await api.getPublicCR()
    session.publicCREnabled = !!pc.enabled
  } catch { session.publicCREnabled = false }

  try {
    await loadPlan()
  } catch (e) {
    session.error = workspaceLoadError(e)
  }
  if (!session.error) {
    try { await loadBaselines() } catch { /* baselines non-fatal */ }
    await loadPalette()
    await loadGroups()
    await loadWorkflows()
    await loadItemTypes()
    await loadWorkspaceMembers()
    await loadUISettings()
    await loadChangeRequests()
  }
  session.ready = true
  if (!session.error) applyNav()
  if (IS_DEMO && !session.error) syncSeededDemoSources()
}

// resolveWorkspaceTarget decides which workspace the request should target: the
// /{slug} in the URL, else the logged-in user's own (reflected into the URL bar),
// else the home/default plan. Sets the api header + the reactive workspace state.
function resolveWorkspaceTarget(me) {
  if (IS_DEMO) { workspace.slug = ''; workspace.isOwn = true; workspace.role = 'owner'; setWorkspaceSlug(''); return }
  const ownSlug = me.workspace || ''
  const urlSlug = workspaceSlugFromUrl()
  let target
  if (urlSlug) {
    target = urlSlug
  } else if (session.authenticated && ownSlug) {
    target = ownSlug
    try { window.history.replaceState(null, '', '/' + encodeURIComponent(ownSlug)) } catch { /* ignore */ }
  } else {
    target = ''
  }
  workspace.slug = target
  workspace.ownSlug = ownSlug
  workspace.isOwn = session.authenticated && !!ownSlug && target === ownSlug
  // Optimistic role for the common case (own home); loadMyWorkspaces() refines it
  // (e.g. when you're an editor of a shared project, or a non-member viewer).
  workspace.role = workspace.isOwn ? 'owner' : null
  setWorkspaceSlug(target)
}

// loadMyWorkspaces fetches the projects the user belongs to (for the switcher)
// and pins the role for the currently-viewed one.
export async function loadMyWorkspaces() {
  if (IS_DEMO || !session.authenticated) { workspace.myWorkspaces = []; return }
  try {
    const list = await api.listProjects()
    workspace.myWorkspaces = list || []
    const mine = (list || []).find(w => w.slug === workspace.slug)
    workspace.role = mine ? mine.role : null
  } catch { workspace.myWorkspaces = [] }
}

// createProject makes a new project and navigates to it.
export async function createProject(name) {
  const ws = await api.createProject({ name })
  window.location.assign('/' + encodeURIComponent(ws.slug))
  return ws
}

// workspaceLoadError maps a failed plan load to a user-facing state.
function workspaceLoadError(e) {
  if (e.status === 404) return 'not-found'
  if (e.status === 403) return 'private'
  if (e.status === 401) {
    return (workspace.slug === '' || workspace.slug === 'default') ? 'auth-required' : 'private'
  }
  return e.message || 'Failed to load'
}

export function useAppStore() {
  function persistYear() {
    try {
      localStorage.setItem(YEAR_KEY, String(store.year))
      localStorage.setItem(GRAN_KEY, store.granularity)
      localStorage.setItem(MONTH_KEY, String(store.viewMonth))
    } catch { /* ignore */ }
  }
  function prevYear() { store.year--; persistYear() }
  function nextYear() { store.year++; persistYear() }
  function setGranularity(g) { store.granularity = g === 'month' ? 'month' : 'year'; persistYear() }
  function setView(v) {
    const nv = NAV_VIEWS.includes(v) ? v : 'timeline'
    store.view = nv
    try { localStorage.setItem(VIEW_KEY, nv) } catch { /* ignore */ }
    // A view switch is a back-able step. Keep the open item only while in Explorer;
    // Settings carries its remembered section (pushNav reads ui.settingsSection).
    const item = nv === 'explorer' ? (ui.explorerItemId || null) : null
    pushNav({ view: nv, item, version: nv === 'explorer' ? ui.explorerItemVersion : null })
  }
  function prevMonth() {
    if (store.viewMonth <= 1) { store.viewMonth = 12; store.year-- } else { store.viewMonth-- }
    persistYear()
  }
  function nextMonth() {
    if (store.viewMonth >= 12) { store.viewMonth = 1; store.year++ } else { store.viewMonth++ }
    persistYear()
  }

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
    const m = { id: uid(), kind: 'milestone', marker: 'l:Flag', version: 1, ...data }
    // Optimistic attribution so "Added by" shows immediately (the server stamps the
    // creator too) — refined by the canonical row merged in below.
    const me = workspace.members.find(w => w.username === session.username)
    if (me) { m.createdBy = me.userId; m.updatedBy = me.userId }
    m.createdAt = new Date().toISOString()
    store.milestones.push(m)
    // Merge the server's canonical row (version, createdBy, timestamps).
    api.createItem(m).then(created => { if (created && created.id) Object.assign(m, created) }).catch(onWriteError)
    return m
  }
  function updateMilestone(id, data) {
    const m = store.milestones.find(m => m.id === id)
    if (m) {
      Object.assign(m, data)
      // Optimistic attribution: the server bumps the version and records us as
      // the editor — mirror that locally so it shows without a reload.
      m.version = (m.version || 1) + 1
      const me = workspace.members.find(w => w.username === session.username)
      if (me) m.updatedBy = me.userId
      m.updatedAt = new Date().toISOString()
    }
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
  function addLink(idA, idB, rel = 'depends-on', version = null) {
    if (idA === idB) return
    const v = version ?? null
    const existing = store.links.find(l => l.a === idA && l.b === idB && (l.rel || 'depends-on') === rel)
    if (existing) {
      if ((existing.version ?? null) === v) return // unchanged
      existing.version = v // version-only change (e.g. re-pin a "uses" link)
    } else {
      store.links.push({ a: idA, b: idB, rel, version: v })
    }
    api.addLink(idA, idB, rel, v).catch(onWriteError)
  }
  function removeLink(idA, idB, rel = 'depends-on') {
    store.links = store.links.filter(l => !(l.a === idA && l.b === idB && (l.rel || 'depends-on') === rel))
    api.removeLink(idA, idB, rel).catch(onWriteError)
  }
  function getLinkedIds(id) {
    return new Set(
      store.links
        .filter(l => l.a === id || l.b === id)
        .map(l => l.a === id ? l.b : l.a)
    )
  }
  // Directed: the prerequisites this item depends on (depends-on links, a === id).
  function dependsOnIds(id) {
    return new Set(store.links.filter(l => l.a === id && (l.rel || 'depends-on') === 'depends-on').map(l => l.b))
  }
  // Directed: the items that depend on this one — its "parents" (depends-on, b === id).
  function dependentIds(id) {
    return new Set(store.links.filter(l => l.b === id && (l.rel || 'depends-on') === 'depends-on').map(l => l.a))
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
    const res = await api.login(username, password)
    if (!IS_DEMO) {
      // Land on your own workspace; a full navigation re-runs initApp cleanly.
      const ownSlug = (res && res.workspace) || username
      window.location.assign('/' + encodeURIComponent(ownSlug))
      return
    }
    session.authenticated = true
    session.username = (res && res.username) || username
    session.role = (res && res.role) || null
    session.error = null
    workspace.isOwn = true
    workspace.role = 'owner'
    try {
      const pr = await api.getPublicRead()
      session.publicReadEnabled = !!pr.enabled
    } catch { /* ignore */ }
    try {
      const pc = await api.getPublicCR()
      session.publicCREnabled = !!pc.enabled
    } catch { /* ignore */ }
    await loadPlan()
  }
  async function logout() {
    if (!IS_DEMO) {
      try { await api.logout() } finally { window.location.assign('/') }
      return
    }
    try { await api.logout() } finally {
      session.authenticated = false
      session.username = null
      session.role = null
    }
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
  async function setPublicCR(enabled) {
    await api.setPublicCR(enabled)
    session.publicCREnabled = enabled
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
    prevYear, nextYear, setGranularity, prevMonth, nextMonth, setView,
    addSwimlane, updateSwimlane, deleteSwimlane, moveSwimlane, setLaneHidden, moveSwimlaneTo, commitSwimlaneOrder, moveSubLaneTo, commitSubLaneOrder,
    addSubLane, updateSubLane, deleteSubLane,
    addMilestone, updateMilestone, deleteMilestone,
    addLink, removeLink, getLinkedIds, dependsOnIds, dependentIds,
    cellMilestones,
    login, logout, setPublicRead, setPublicCR, loadPlan,
    loadBaselines, selectBaseline, createBaseline, deleteBaseline,
    addPaletteColor, removePaletteColor, resetPalette,
    addGroup, updateGroup, deleteGroup, toggleItemGroup, itemGroupIds, setItemGroups,
  }
}
