// Backend-less API used by the static GitHub Pages demo. It implements the same
// interface as the real `api` (see api.js) but operates on a localStorage-backed
// dataset, so the published demo is a fully interactive sandbox (no login, no
// server, changes persist in the browser only).
import { demoSeed } from './demoSeed.js'

const KEY = 'atlas-demo-v3'
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
  }
}

let db = load()
function save() {
  try { localStorage.setItem(KEY, JSON.stringify(db)) } catch { /* ignore quota */ }
}
const ok = (v = null) => Promise.resolve(v)

export const demoApi = {
  // auth (the demo is an open, editable sandbox)
  me: () => ok({ authenticated: true }),
  login: () => ok({ authenticated: true }),
  logout: () => ok({ authenticated: false }),

  // Return clones so the reactive store never shares array refs with the db
  // (otherwise an optimistic push + the db push would duplicate the item).
  getPlan: () => ok({
    swimlanes: db.swimlanes.map(s => ({ ...s, subLanes: s.subLanes.map(sl => ({ ...sl })) })),
    milestones: db.milestones.map(m => ({ ...m })),
    links: db.links.map(l => ({ ...l })),
  }),
  getPublicRead: () => ok({ enabled: db.settings.publicReadEnabled }),
  setPublicRead: (enabled) => { db.settings.publicReadEnabled = enabled; save(); return ok({ enabled }) },
  getPalette: () => ok({ colors: db.palette == null ? null : [...db.palette] }),
  setPalette: (colors) => { db.palette = colors || []; save(); return ok({ colors: [...db.palette] }) },
  getGroups: () => ok({ groups: (db.groups || []).map(g => ({ ...g, itemIds: [...(g.itemIds || [])] })) }),
  setGroups: (groups) => { db.groups = groups || []; save(); return ok({ groups: db.groups }) },

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
  createSubLane: (swimlaneId, data) => {
    const sl = db.swimlanes.find(s => s.id === swimlaneId)
    const sub = { id: data.id || uid(), name: data.name }
    if (sl) sl.subLanes.push(sub); save(); return ok(sub)
  },
  updateSubLane: (id, name) => {
    for (const sl of db.swimlanes) { const sub = sl.subLanes.find(s => s.id === id); if (sub) { sub.name = name; break } }
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

  addLink: (a, b) => {
    if (a !== b && !db.links.some(l => (l.a === a && l.b === b) || (l.a === b && l.b === a))) db.links.push({ a, b })
    save(); return ok()
  },
  removeLink: (a, b) => {
    db.links = db.links.filter(l => !((l.a === a && l.b === b) || (l.a === b && l.b === a)))
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
}
