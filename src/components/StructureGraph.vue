<template>
  <div ref="wrapEl" class="sg" :class="{ empty: !layout.nodes.length }">
    <!-- Controls: on the graph they float top-right OVER the canvas (no banner,
         no divider); the table keeps them in-flow above its header. -->
    <div class="sg-bar" :class="{ overlay: view === 'graph' }">
      <div v-if="view === 'graph' && layout.nodes.length" class="sg-stats">
        <button type="button" class="sg-mini" @click="expandAll">Expand all</button>
        <button type="button" class="sg-mini" @click="collapseAll">Collapse</button>
        <span class="sg-dot">·</span>
        <span class="sg-stat"><strong>{{ layout.nodes.length }}</strong> shown</span>
        <span v-if="pinnedCount" class="sg-dot">·</span>
        <span v-if="pinnedCount" class="sg-stat sg-stat-pin"><strong>{{ pinnedCount }}</strong> pinned</span>
      </div>
      <div v-if="view === 'graph' && layout.nodes.length" class="sg-zoom">
        <button type="button" class="sg-zbtn" title="Zoom out" @click="zoomBy(1/1.25)">−</button>
        <button type="button" class="sg-zbtn sg-fit" title="Fit to view" @click="fit()">Fit</button>
        <button type="button" class="sg-zbtn" title="Zoom in" @click="zoomBy(1.25)">+</button>
      </div>

      <div v-if="view === 'table'" class="sg-stats">
        <button type="button" class="sg-mini" :class="{ on: instMode }" title="One row per usage position (RefDes) instead of aggregated quantities" @click="instMode = !instMode">Per instance</button>
        <span class="sg-dot">·</span>
        <button type="button" class="sg-mini" @click="expandAll">Expand all</button>
        <button type="button" class="sg-mini" @click="collapseAll">Collapse</button>
        <select class="sg-lvl-sel" :value="''" title="Expand to level" @change="applyLevel($event)">
          <option value="" disabled>Level…</option>
          <option v-for="l in [1, 2, 3, 4, 5, 6, 8, 10]" :key="l" :value="l">to {{ l }}</option>
          <option value="all">all</option>
        </select>
        <span class="sg-dot">·</span>
        <span class="sg-stat"><strong>{{ Math.max(0, bomRows.length - 1) }}</strong><template v-if="bomRows.length - 1 < bomTotal"> / {{ bomTotal }}</template> positions</span>
      </div>
      <div v-if="view === 'table'" class="sg-colpick">
        <button type="button" class="sg-mini" :class="{ on: exportOpen }" @click="exportOpen = !exportOpen">Export</button>
        <div v-if="exportOpen" class="sg-cols-bg" @click="exportOpen = false"></div>
        <div v-if="exportOpen" class="sg-cols-pop">
          <button type="button" class="sg-cols-row" @click="exportBom('csv')">CSV</button>
          <button type="button" class="sg-cols-row" @click="exportBom('md')">Markdown</button>
        </div>
      </div>
      <div v-if="view === 'table'" class="sg-colpick">
        <button type="button" class="sg-mini" :class="{ on: colsOpen }" @click="colsOpen = !colsOpen">Columns</button>
        <div v-if="colsOpen" class="sg-cols-bg" @click="colsOpen = false"></div>
        <div v-if="colsOpen" class="sg-cols-pop">
          <label v-for="c in allCols" :key="c.key" class="sg-cols-row">
            <input type="checkbox" :checked="visibleColKeys.has(c.key)" @change="toggleCol(c.key)" />
            <span>{{ c.label }}</span>
          </label>
        </div>
      </div>
    </div>

    <!-- Canvas -->
    <div class="sg-canvas">
      <!-- BOM table: the fully exploded tree as indented positions (1, 1.1, 1.2…),
           columns configurable via the picker (persisted per browser). -->
      <div v-if="view === 'table' && bomRows.length" class="sg-bom">
        <table class="sg-bt">
          <thead>
            <tr>
              <th class="bt-c-pos">#</th>
              <th class="bt-c-item">Item</th>
              <th v-for="c in activeCols" :key="c.key">{{ c.label }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in bomRows" :key="r.rowKey" :class="{ ref: r.n.ref, root: r.depth === 0 }" @click="emit('edit', r.n.raw)">
              <td class="bt-c-pos">{{ r.pos }}</td>
              <td class="bt-c-item">
                <!-- flex lives on an inner div — flex directly on a td/th breaks
                     table-cell layout (misaligned sticky header, broken row lines). -->
                <div class="bt-item">
                  <span class="bt-ind" :style="{ width: indentFor(r.depth) + 'px' }"></span>
                  <button v-if="r.hasKids" type="button" class="bt-tog" :class="{ open: !r.collapsed }" :title="r.collapsed ? 'Expand' : 'Collapse'" @click.stop="toggle(r.n)"><ChevronRight :size="14" :stroke-width="2.4" /></button>
                  <span v-else class="bt-tog-sp"></span>
                  <MarkerIcon :shape="r.n.icon" :color="r.n.color" :size="14" :fill="r.n.fill" />
                  <span v-if="r.inst" class="bt-inst">{{ r.inst }}</span>
                  <span class="bt-title">{{ r.n.title }}</span>
                  <span v-if="r.collapsed" class="bt-hid" title="Hidden positions inside">{{ r.hiddenCount }}</span>
                  <span v-if="r.n.ref" class="bt-refm" title="Circular reference — this item is already part of its own structure above">↻</span>
                </div>
              </td>
              <td v-for="c in activeCols" :key="c.key" class="bt-val">
                <template v-if="c.key === 'level'"><span class="bt-lvl">{{ r.depth }}</span></template>
                <template v-else-if="c.key === 'qty'"><span v-if="r.depth !== 0" class="bt-qty">{{ r.inst ? 1 : (r.n.qty || 1) }}×</span></template>
                <template v-else-if="c.key === 'refdes'">{{ r.inst || r.n.des }}</template>
                <template v-else-if="c.key === 'version'"><span class="bt-ver" :class="r.n.pin != null ? 'pinned' : 'latest'">v{{ r.n.pin ?? r.n.version }}</span><span class="bt-vtag" :class="r.n.pin != null ? 'pinned' : 'latest'">{{ r.n.pin != null ? 'pinned' : 'latest' }}</span></template>
                <template v-else-if="c.key === 'status'"><span v-if="statusOf(r.n)" class="bt-status"><span class="bt-dot" :style="{ background: r.n.color }"></span>{{ statusOf(r.n) }}</span></template>
                <template v-else>{{ colVal(r.n, c) }}</template>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <svg v-else-if="view === 'graph' && layout.nodes.length" ref="svgEl" class="sg-svg" @dblclick="fit()">
        <defs>
          <pattern id="sg-grid" width="26" height="26" patternUnits="userSpaceOnUse">
            <circle cx="1" cy="1" r="1" class="sg-grid-dot" />
          </pattern>
        </defs>

        <g :transform="`translate(${t.x},${t.y}) scale(${t.k})`">
          <rect :x="bounds.x - 3000" :y="bounds.y - 3000" :width="bounds.w + 6000" :height="bounds.h + 6000" fill="url(#sg-grid)" />

          <!-- links -->
          <path
            v-for="e in layout.edges" :key="e.id"
            class="sg-link" :class="{ hot: hotEdges.has(e.b), dim: hover && !hotEdges.has(e.b) }"
            :d="e.d"
          />

          <!-- nodes — rendered as native SVG (foreignObject is unreliable in SVG
               and silently dropped some labels, leaving "bare dots"). -->
          <g
            v-for="n in layout.nodes" :key="n.uid"
            class="sg-nodeg" :class="{ dim: hover && !hotNodes.has(n.uid) }"
            :transform="`translate(${n.sx},${n.sy})`"
            @mouseenter="hover = n.uid" @mouseleave="hover = null"
          >
            <!-- pill = hover background + click target -->
            <rect
              class="sg-pill" :class="{ root: n.depth === 0, focus: hover === n.uid }"
              :style="{ '--acc': n.color }"
              :x="n.pillX" y="-15" :width="n.pillW" height="30" rx="8"
              @click="emit('edit', n.raw)"
            />
            <!-- node point: toggle (expandable) or the type icon (leaf, edge terminus) -->
            <g v-if="n.hasKids" class="sg-tog" :class="{ open: !n.collapsed }" @click.stop="toggle(n)">
              <circle r="9.5" />
              <text class="sg-tog-t" x="0" y="0.5" text-anchor="middle" dominant-baseline="middle">{{ n.collapsed ? n.hiddenCount : '−' }}</text>
            </g>
            <g v-else class="sg-nico" transform="translate(-8.5,-8.5)"><MarkerIcon :shape="n.icon" :color="n.color" :size="17" :fill="n.fill" /></g>
            <!-- expandable nodes still show their category icon (after the toggle) -->
            <g v-if="n.hasKids" class="sg-nico" transform="translate(16,-8.5)"><MarkerIcon :shape="n.icon" :color="n.color" :size="17" :fill="n.fill" /></g>

            <!-- title + type + ref + version: one text; tspans flow after the real
                 title end. Version shows on EVERY node — the pinned one when the
                 edge pins a revision, else the current head marked "latest". -->
            <text class="sg-title" :class="{ root: n.depth === 0, ref: n.ref }" :ref="el => setTextRef(n.uid, el)" :x="n.titleX" y="4.5" @click="emit('edit', n.raw)"><tspan v-if="n.depth !== 0" class="sg-qty">{{ n.qty || 1 }}×</tspan><tspan :dx="n.depth !== 0 ? 7 : 0">{{ n.title }}</tspan><tspan class="sg-type" dx="11">{{ n.typeLabel }}</tspan><tspan v-if="n.ref" class="sg-refm" dx="9">↻</tspan><tspan v-if="n.pin != null" class="sg-ver pinned" dx="13">v{{ n.pin }}</tspan><tspan v-if="n.pin != null" class="sg-vtag pinned" dx="5">pinned</tspan><tspan v-if="n.pin == null" class="sg-ver latest" dx="13">v{{ n.version }}</tspan><tspan v-if="n.pin == null && n.depth !== 0" class="sg-vtag latest" dx="5">latest</tspan></text>
          </g>
        </g>
      </svg>

      <div v-else class="sg-empty">
        <div class="sg-empty-glyph">◇</div>
        <p>“{{ rootItem?.title || 'This item' }}” isn’t composed of anything.<br /><span>Add components under the <strong>Uses</strong> tab to build its structure.</span></p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { select, zoom, zoomIdentity, hierarchy, tree } from 'd3'
import { store, itemTypes, itemTypeByKey, itemStatus, statusColor, toneColor, memberName, parseRef, parseDesignators } from '../stores/useAppStore.js'
import { ChevronRight } from 'lucide-vue-next'
import MarkerIcon from './MarkerIcon.vue'

// The structure of ONE item, rendered inside the item's tabs — as a node graph
// (view="graph", the Graph tab) or as an indented BOM table (view="table", the
// BOM tab). The root is fixed to that item; "edit" bubbles a node's raw item.
const props = defineProps({
  rootId: { type: String, required: true },
  view: { type: String, default: 'graph' }, // 'graph' | 'table'
})
const emit = defineEmits(['edit'])

const ROW_H = 48    // vertical gap between sibling rows
const COL_W = 348   // horizontal gap between levels (longer connector lines)
const usesEdges = () => store.links.filter(l => (l.rel || '') === 'uses')

const rootItem = computed(() => store.milestones.find(m => m.id === props.rootId) || null)

// The BOM table walks the exploded tree with classic 1 / 1.1 / 1.2.3 numbering.
// It shares the graph's collapse state (same keys): collapsed rows hide their
// subtree but numbering stays stable — it's computed from the full tree shape.
//
// Per-instance mode ("Einzelansicht"): instead of one aggregated row (10×,
// C1-C17), every usage becomes its own row — named by its parsed designator, or
// #1…#n from qty when designators are missing/unparseable. Each instance repeats
// the node's subtree (its as-built reality); the toggle collapses ALL instances
// of a node together (shared key).
const instMode = ref(false)
function instancesOf(n) {
  const parsed = parseDesignators(n.des)
  if (parsed) return parsed
  const q = n.qty || 1
  return q > 1 ? Array.from({ length: q }, (_, i) => '#' + (i + 1)) : [null]
}
const bomRows = computed(() => {
  const t = fullTree.value
  if (!t) return []
  const rows = []
  const walk = (n, pos, inst) => {
    const isCollapsed = n.hasKids && collapsed.has(n.key)
    rows.push({
      n, pos: pos.join('.'), inst, depth: n.depth, hasKids: n.hasKids,
      collapsed: isCollapsed, hiddenCount: isCollapsed ? countHidden(n) : 0,
      rowKey: n.uid + (inst ? '@' + inst : ''),
    })
    if (isCollapsed) return
    n.children.forEach((c, i) => {
      if (instMode.value) {
        for (const label of instancesOf(c)) walk(c, [...pos, label ? `${i + 1}/${label}` : i + 1], label)
      } else {
        walk(c, [...pos, i + 1], null)
      }
    })
  }
  walk(t, [], null)
  return rows
})
// Total positions in the full explosion (root excluded) — shown as "12 / 240".
const bomTotal = computed(() => {
  const t = fullTree.value
  if (!t) return 0
  let c = -1
  const walk = n => { c++; n.children.forEach(walk) }
  walk(t)
  return c
})
// Expand to level N: everything at depth < N open, deeper closed. Deep BOMs
// (20 levels) are browsed this way instead of scrolling a full explosion.
function expandToLevel(n) {
  collapsed.clear()
  const t = fullTree.value
  if (!t) return
  const walk = (x) => { if (x.hasKids && x.depth >= n) collapsed.add(x.key); x.children.forEach(walk) }
  walk(t)
}
function applyLevel(ev) {
  const v = ev.target.value
  if (v === 'all') collapsed.clear()
  else if (v !== '') expandToLevel(Number(v))
  ev.target.value = '' // acts as a menu, not a state — reset to the placeholder
}
// Indent is capped so 20 levels don't push the title half a metre right — the
// depth stays readable via the position number and the Level column.
function indentFor(d) { return Math.min(d, 8) * 14 + Math.max(0, d - 8) * 4 }

// Column registry: the built-in attributes plus every field defined on any item
// type (custom types included). Off by default except the BOM essentials; the
// selection persists per browser.
const BASE_COLS = [
  { key: 'level', label: 'Level' },
  { key: 'qty', label: 'Qty' },
  { key: 'refdes', label: 'RefDes' },
  { key: 'type', label: 'Type' },
  { key: 'version', label: 'Version' },
  { key: 'status', label: 'Status' },
  { key: 'assignee', label: 'Assignee' },
  { key: 'maturity', label: 'Maturity' },
  { key: 'progress', label: 'Progress' },
  { key: 'when', label: 'Date / Start' },
  { key: 'end', label: 'End' },
]
const DEFAULT_COL_KEYS = ['level', 'qty', 'refdes', 'type', 'version', 'status']
const BOM_COLS_KEY = 'atlas-bom-cols'
const allCols = computed(() => {
  const seen = new Set(BASE_COLS.map(c => c.key))
  const out = [...BASE_COLS]
  for (const t of itemTypes.list) {
    for (const f of (t.fields || [])) {
      const k = 'f:' + f.key
      if (seen.has(k)) continue
      seen.add(k)
      out.push({ key: k, label: f.label || f.key, field: f })
    }
  }
  return out
})
function loadColKeys() {
  try {
    const v = JSON.parse(localStorage.getItem(BOM_COLS_KEY) || 'null')
    if (Array.isArray(v)) return v.filter(x => typeof x === 'string')
  } catch { /* ignore */ }
  return DEFAULT_COL_KEYS
}
const visibleColKeys = ref(new Set(loadColKeys()))
function toggleCol(k) {
  const s = new Set(visibleColKeys.value)
  if (s.has(k)) s.delete(k); else s.add(k)
  visibleColKeys.value = s
  try { localStorage.setItem(BOM_COLS_KEY, JSON.stringify([...s])) } catch { /* ignore */ }
}
const activeCols = computed(() => allCols.value.filter(c => visibleColKeys.value.has(c.key)))
const colsOpen = ref(false)

// ── BOM export (CSV / Markdown): the FULL explosion — collapse state is a view
// concern; an export always covers everything. Respects the active columns and
// the per-instance mode, so what you configured is what you hand to purchasing.
const exportOpen = ref(false)
function exportRows() {
  const t = fullTree.value
  if (!t) return []
  const rows = []
  const walk = (n, pos, inst) => {
    rows.push({ n, pos: pos.join('.'), inst, depth: n.depth })
    n.children.forEach((c, i) => {
      if (instMode.value) {
        for (const label of instancesOf(c)) walk(c, [...pos, label ? `${i + 1}/${label}` : i + 1], label)
      } else {
        walk(c, [...pos, i + 1], null)
      }
    })
  }
  walk(t, [], null)
  return rows
}
function exportCell(r, c) {
  if (c.key === 'level') return String(r.depth)
  if (c.key === 'qty') return r.depth === 0 ? '' : String(r.inst ? 1 : (r.n.qty || 1))
  if (c.key === 'refdes') return r.inst || r.n.des || ''
  if (c.key === 'version') return r.n.pin != null ? `v${r.n.pin} (pinned)` : `v${r.n.version} (latest)`
  if (c.key === 'status') return statusOf(r.n)
  return String(colVal(r.n, c) ?? '')
}
function exportBom(kind) {
  exportOpen.value = false
  const cols = activeCols.value
  const header = ['#', 'Item', ...cols.map(c => c.label)]
  const rows = exportRows().map(r => [r.pos, (r.n.ref ? '↻ ' : '') + (r.n.title || ''), ...cols.map(c => exportCell(r, c))])
  let text, mime, ext
  if (kind === 'csv') {
    const esc = v => /[",;\n]/.test(v) ? '"' + v.replace(/"/g, '""') + '"' : v
    text = [header, ...rows].map(cells => cells.map(esc).join(',')).join('\n')
    mime = 'text/csv'; ext = 'csv'
  } else {
    const esc = v => String(v).replace(/\|/g, '\\|')
    text = [
      `| ${header.map(esc).join(' | ')} |`,
      `| ${header.map(() => '---').join(' | ')} |`,
      ...rows.map(cells => `| ${cells.map(esc).join(' | ')} |`),
    ].join('\n')
    mime = 'text/markdown'; ext = 'md'
  }
  const name = `bom-${(rootItem.value?.title || 'item').replace(/[^\w.-]+/g, '_')}.${ext}`
  const url = URL.createObjectURL(new Blob([text], { type: mime + ';charset=utf-8' }))
  const a = document.createElement('a')
  a.href = url; a.download = name
  document.body.appendChild(a); a.click(); document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

function statusOf(n) { const st = itemStatus(n.raw); return st?.label || st?.key || '' }
function colVal(n, c) {
  const m = n.raw
  if (c.key === 'type') return n.typeLabel
  if (c.key === 'assignee') return m.assigneeId ? memberName(m.assigneeId) : ''
  if (c.key === 'maturity') return m.maturity || ''
  if (c.key === 'progress') return m.progress != null ? m.progress + '%' : ''
  if (c.key === 'when') return m.startDate || m.when || ''
  if (c.key === 'end') return m.endDate || ''
  if (c.field) {
    const v = m.data?.[c.field.key]
    if (v == null || v === '') return ''
    const entries = Array.isArray(v) ? v : [v]
    if (c.field.type === 'reference') {
      const byId = new Map(store.milestones.map(x => [x.id, x]))
      return entries.map(e => {
        const { id, version } = parseRef(e)
        const t = byId.get(id)?.title || id
        return version ? `${t} @v${version}` : t
      }).join(', ')
    }
    return entries.join(', ')
  }
  return ''
}

function nodeOf(m, depth) {
  const ty = itemTypeByKey(m.typeKey || m.kind || 'milestone')
  const st = itemStatus(m)
  return {
    id: m.id, raw: m, title: m.title || '(untitled)', version: m.version || 1, depth,
    typeLabel: ty?.label || (m.typeKey || m.kind || ''),
    icon: ty?.icon || 'l:Diamond', fill: ty?.fill !== false,
    color: st ? statusColor(st) : toneColor('neutral'),
  }
}

// Full exploded-BOM tree from the uses-links: each usage a branch with its FULL
// subtree — the same part under two assemblies is normal BOM reality, so both
// occurrences expand completely. Only a true CYCLE (an item already on its own
// ancestor path) stops as a "reference" leaf, else recursion would never end.
const fullTree = computed(() => {
  const root = props.rootId
  const byId = new Map(store.milestones.map(m => [m.id, m]))
  if (!root || !byId.has(root)) return null
  const links = usesEdges()
  let seq = 0
  const make = (id, pin, qty, des, path) => {
    const m = byId.get(id); if (!m) return null
    const base = nodeOf(m, path.length)
    const key = [...path, id].join('>')
    const node = { ...base, uid: 'n' + (seq++), key, pin, qty, des, ref: false, children: [] }
    if (path.includes(id)) { node.ref = true; node.hasKids = false; return node }
    const next = [...path, id]
    for (const l of links) {
      if (l.a !== id || !byId.has(l.b)) continue
      const child = make(l.b, l.version ?? null, l.qty ?? null, l.designators || '', next)
      if (child) node.children.push(child)
    }
    node.hasKids = node.children.length > 0
    return node
  }
  return make(root, null, null, '', [])
})

// Collapse state (by stable path key). Default: expand the root + its direct
// children, collapse everything deeper — a clean, drill-down starting point.
const collapsed = reactive(new Set())
watch(fullTree, (t) => {
  collapsed.clear()
  if (!t) return
  const walk = (n) => { if (n.hasKids && n.depth >= 1) collapsed.add(n.key); n.children.forEach(walk) }
  walk(t)
}, { immediate: true })
function toggle(n) { collapsed.has(n.key) ? collapsed.delete(n.key) : collapsed.add(n.key) }
function expandAll() { collapsed.clear() }
function collapseAll() {
  collapsed.clear()
  const t = fullTree.value; if (!t) return
  const walk = (n) => { if (n.hasKids && n.depth >= 1) collapsed.add(n.key); n.children.forEach(walk) }
  walk(t)
}

function countHidden(n) { let c = 0; const walk = x => { c += 1; x.children.forEach(walk) }; n.children.forEach(walk); return c }

// Prune collapsed subtrees → the visible tree that gets laid out.
const visibleTree = computed(() => {
  const full = fullTree.value
  if (!full) return null
  const prune = (n) => {
    const isCollapsed = n.hasKids && collapsed.has(n.key)
    return { ...n, collapsed: isCollapsed, hiddenCount: isCollapsed ? countHidden(n) : 0, children: isCollapsed ? [] : n.children.map(prune) }
  }
  return prune(full)
})

// Measured label widths (getBBox after render, in node-local coords) → exact pill
// width + edge start, so a line never cuts through a label or a "v117 LATEST" tag.
const textEls = new Map()
function setTextRef(uid, el) { if (el) textEls.set(uid, el); else textEls.delete(uid) }
const nodeRight = reactive({}) // uid → right edge of the label
function measure() {
  for (const [uid, el] of textEls) {
    try { const b = el.getBBox(); if (b && b.width) nodeRight[uid] = b.x + b.width } catch { /* not laid out yet */ }
  }
}

const MARKER_GAP = 22 // stop ~12px clear of the toggle/icon edge (its ~10px radius + gap)

// Horizontal (left-to-right) tidy tree — d3.tree gives crossing-free positions.
const layout = computed(() => {
  const vt = visibleTree.value
  if (!vt) return { nodes: [], edges: [] }
  const h = hierarchy(vt, d => d.children)
  tree().nodeSize([ROW_H, COL_W])(h)
  const nodes = h.descendants().map(d => {
    const dd = d.data
    const titleX = dd.hasKids ? 42 : 22   // clear the toggle+icon (expandable) or the leaf icon
    // exact label right edge once measured; a rough estimate for the first paint.
    const estRight = titleX + (dd.title || '').length * 8.6 + (dd.typeLabel || '').length * 6.6 + (dd.ref ? 20 : 0) + (dd.depth !== 0 ? 28 : 0) + 74
    const rightX = nodeRight[dd.uid] != null ? nodeRight[dd.uid] : estRight
    const pillX = -15
    return { ...dd, sx: d.y, sy: d.x, titleX, pillX, rightX, pillW: rightX + 16 - pillX, pathUids: d.ancestors().map(a => a.data.uid) }
  })
  // Outgoing edge starts at the parent's real right edge (past its label); the
  // incoming end stops just before the child's toggle/icon (never inside it).
  const byUid = new Map(nodes.map(n => [n.uid, n]))
  const edges = h.links().map((lk, i) => {
    const s = byUid.get(lk.source.data.uid), t = byUid.get(lk.target.data.uid)
    const sx = s.sx + s.rightX + 12, sy = s.sy, tx = t.sx - MARKER_GAP, ty = t.sy
    const mx = (sx + tx) / 2
    return { id: t.uid + '|' + i, b: t.uid, d: `M${sx},${sy} C${mx},${sy} ${mx},${ty} ${tx},${ty}` }
  })
  return { nodes, edges }
})
const pinnedCount = computed(() => layout.value.nodes.filter(n => n.pin != null).length)
const bounds = computed(() => {
  const ns = layout.value.nodes
  if (!ns.length) return { x: 0, y: 0, w: 100, h: 100 }
  let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity
  for (const n of ns) { x0 = Math.min(x0, n.sx + n.pillX - 8); x1 = Math.max(x1, n.sx + n.pillX + n.pillW + 8); y0 = Math.min(y0, n.sy - 20); y1 = Math.max(y1, n.sy + 20) }
  return { x: x0, y: y0, w: x1 - x0, h: y1 - y0 }
})

// Hover: light up the whole path from the hovered node back to the root.
const hover = ref(null)
const hotNodes = computed(() => {
  const set = new Set()
  if (hover.value) { const n = layout.value.nodes.find(x => x.uid === hover.value); if (n) n.pathUids.forEach(u => set.add(u)) }
  return set
})
const hotEdges = computed(() => hotNodes.value) // edge keyed by its target uid

// ── Pan / zoom (d3) ──────────────────────────────────────────────────────────
const wrapEl = ref(null); const svgEl = ref(null)
const t = reactive({ x: 0, y: 0, k: 1 })
let zoomB = null
function bindZoom() {
  if (!svgEl.value || zoomB) return
  zoomB = zoom().scaleExtent([0.15, 2.4]).on('zoom', ev => { t.x = ev.transform.x; t.y = ev.transform.y; t.k = ev.transform.k })
  select(svgEl.value).call(zoomB).on('dblclick.zoom', null)
}
function fit(animate = true) {
  if (!svgEl.value) { return } ; bindZoom(); if (!zoomB) return
  const r = svgEl.value.getBoundingClientRect()
  if (r.width < 10 || r.height < 10) return
  const b = bounds.value
  if (!(b.w > 0) || !(b.h > 0)) return
  const pad = 90
  const k = Math.max(0.15, Math.min(1.4, Math.min((r.width - pad * 2) / b.w, (r.height - pad * 2) / b.h)))
  const x = r.width / 2 - (b.x + b.w / 2) * k
  const y = r.height / 2 - (b.y + b.h / 2) * k
  const sel = select(svgEl.value)
  ;(animate ? sel.transition().duration(300) : sel).call(zoomB.transform, zoomIdentity.translate(x, y).scale(k))
}
function zoomBy(f) { bindZoom(); if (svgEl.value && zoomB) select(svgEl.value).transition().duration(160).call(zoomB.scaleBy, f) }

let resizeObs = null, fittedOnce = false
onMounted(() => {
  nextTick(() => {
    measure(); bindZoom(); fit(false)
    resizeObs = new ResizeObserver(() => {
      const r = svgEl.value?.getBoundingClientRect()
      if (r && r.width > 10 && r.height > 10 && !fittedOnce) { fittedOnce = true; measure(); fit(false) }
    })
    if (wrapEl.value) resizeObs.observe(wrapEl.value)
  })
})
watch(() => props.rootId, () => { fittedOnce = false; nextTick(() => { measure(); fit(true) }) })
// Re-measure whenever the visible node set changes (expand/collapse) so widths
// (and therefore edge starts + pills) stay exact.
watch(() => layout.value.nodes.map(n => n.uid).join(','), () => nextTick(measure))
onBeforeUnmount(() => { if (resizeObs) resizeObs.disconnect(); if (svgEl.value) select(svgEl.value).on('.zoom', null) })
</script>

<style scoped>
.sg { position: relative; display: flex; flex-direction: column; height: 100%; min-height: 0; background: var(--clr-bg); }

/* Controls row — no banner, no divider. In-flow above the BOM table; as a
   floating layer over the canvas in graph view (.overlay). */
.sg-bar { display: flex; align-items: center; gap: 16px; padding: 10px 14px; flex-shrink: 0; position: relative; z-index: 5; }
.sg-bar.overlay { position: absolute; top: 8px; right: 10px; padding: 0; z-index: 10; }
.sg-stats { display: flex; align-items: center; gap: 9px; font-size: 12.5px; color: var(--clr-text-3); margin-left: auto; }
.sg-stat strong { color: var(--clr-text-2); font-weight: 700; font-variant-numeric: tabular-nums; }
.sg-stat-pin strong { color: var(--clr-accent); }
.sg-dot { opacity: 0.4; }
.sg-mini { font-size: 12px; font-weight: 600; color: var(--clr-text-2); background: var(--clr-bg-2); border: 1px solid var(--clr-border);
  border-radius: 7px; padding: 4px 9px; cursor: pointer; transition: all 0.12s; }
.sg-mini:hover { color: var(--clr-text); border-color: var(--clr-accent); }
.sg-zoom { display: flex; align-items: center; gap: 4px; }

/* Expand-to-level menu (table view) */
.sg-lvl-sel { font-size: 12px; font-weight: 600; color: var(--clr-text-2); background: var(--clr-bg-2);
  border: 1px solid var(--clr-border); border-radius: 7px; padding: 4px 6px; cursor: pointer; outline: none; }
.sg-lvl-sel:hover { color: var(--clr-text); border-color: var(--clr-accent); }
.sg-lvl-sel option { color: var(--clr-text); background: var(--clr-surface); }

/* Column picker (table view) */
.sg-colpick { position: relative; flex-shrink: 0; }
.sg-mini.on { color: var(--clr-text); border-color: var(--clr-accent); }
.sg-cols-bg { position: fixed; inset: 0; z-index: 25; }
.sg-cols-pop { position: absolute; top: calc(100% + 6px); right: 0; z-index: 30; min-width: 190px; max-height: 320px; overflow-y: auto;
  background: var(--clr-surface); border: 1px solid var(--clr-border); border-radius: 10px;
  box-shadow: 0 12px 34px rgba(0, 0, 0, 0.35); padding: 7px; }
.sg-cols-row { display: flex; align-items: center; gap: 8px; padding: 5px 8px; border-radius: 6px; font-size: 12.5px;
  font-weight: 550; color: var(--clr-text-2); cursor: pointer; }
.sg-cols-row:hover { background: var(--clr-surface-2); color: var(--clr-text); }
.sg-cols-row input { accent-color: var(--clr-accent); }
.sg-zbtn { min-width: 30px; height: 30px; padding: 0 8px; border: 1px solid var(--clr-border); background: var(--clr-bg-2);
  color: var(--clr-text-2); border-radius: 8px; font-size: 16px; font-weight: 600; cursor: pointer; transition: all 0.13s;
  display: inline-flex; align-items: center; justify-content: center; }
.sg-zbtn:hover { background: var(--clr-surface-2); color: var(--clr-text); border-color: var(--clr-accent); }
.sg-fit { font-size: 12px; font-weight: 700; letter-spacing: 0.3px; }

/* Canvas */
.sg-canvas { position: relative; flex: 1; min-height: 0; overflow: hidden; }
.sg-svg { width: 100%; height: 100%; display: block; cursor: grab; }
.sg-svg:active { cursor: grabbing; }
.sg-grid-dot { fill: var(--clr-text); opacity: 0.05; }

/* Links */
.sg-link { fill: none; stroke: color-mix(in srgb, var(--clr-text-3) 45%, transparent); stroke-width: 1.6px;
  transition: stroke 0.15s, opacity 0.15s, stroke-width 0.15s; }
.sg-link.hot { stroke: var(--clr-accent); stroke-width: 2.2px; }
.sg-link.dim { opacity: 0.25; }

/* Node group */
.sg-nodeg { transition: opacity 0.15s; }
.sg-nodeg.dim { opacity: 0.4; }

/* Toggle */
.sg-tog { cursor: pointer; }
.sg-tog circle { fill: color-mix(in srgb, var(--clr-accent) 16%, var(--clr-surface)); stroke: color-mix(in srgb, var(--clr-accent) 50%, transparent); stroke-width: 1.5px; transition: stroke 0.13s, fill 0.13s; }
.sg-tog.open circle { fill: var(--clr-surface); stroke: color-mix(in srgb, var(--clr-text-3) 50%, transparent); }
.sg-tog:hover circle { stroke: var(--clr-accent); }
.sg-tog-t { font-size: 10px; font-weight: 800; font-variant-numeric: tabular-nums; pointer-events: none; fill: var(--clr-accent); }
.sg-tog.open .sg-tog-t { fill: var(--clr-text-2); font-weight: 600; }
.sg-nico { pointer-events: none; }

/* Version label at the row end (uniform on every node) */
.sg-ver { font-size: 11px; font-weight: 700; font-variant-numeric: tabular-nums; }
.sg-ver.pinned { fill: var(--clr-accent); }
.sg-ver.latest { fill: var(--clr-text-3); }
.sg-vtag { font-size: 9px; font-weight: 700; letter-spacing: 0.4px; text-transform: uppercase; }
.sg-vtag.pinned { fill: color-mix(in srgb, var(--clr-accent) 72%, transparent); }
.sg-vtag.latest { fill: var(--clr-text-3); opacity: 0.6; }

/* Node label — pure SVG (a pill hover/click target + a swatch + text). Native
   SVG text always renders; foreignObject silently dropped some labels. */
.sg-pill { fill: transparent; stroke: transparent; stroke-width: 1px; cursor: pointer; transition: fill 0.12s, stroke 0.12s; }
.sg-nodeg:hover .sg-pill, .sg-pill.focus { fill: var(--clr-surface-2); stroke: color-mix(in srgb, var(--acc) 55%, var(--clr-border)); }
.sg-pill.root { fill: color-mix(in srgb, var(--acc) 15%, var(--clr-surface)); stroke: color-mix(in srgb, var(--acc) 45%, transparent); }
.sg-title { fill: var(--clr-text); font-size: 13px; font-weight: 600; cursor: pointer; }
.sg-qty { fill: var(--clr-accent); font-weight: 800; font-variant-numeric: tabular-nums; }
.sg-title.root { font-weight: 750; }
.sg-title.ref { fill: var(--clr-text-2); font-style: italic; font-weight: 500; }
.sg-type { fill: var(--clr-text-3); font-size: 11px; font-weight: 500; }
.sg-refm { fill: var(--clr-text-3); font-size: 12px; }

/* BOM table */
.sg-bom { position: absolute; inset: 0; overflow: auto; }
/* border-collapse: separate — collapsed borders + sticky thead glitch on scroll
   (the header's line belongs to scrolling rows and tears off). With separate
   borders every cell keeps its own bottom line, so rows stay continuous. */
.sg-bt { width: 100%; border-collapse: separate; border-spacing: 0; font-size: 13px; }
.sg-bt th { position: sticky; top: 0; z-index: 2; text-align: left; font-size: 10.5px; font-weight: 700; letter-spacing: 0.5px;
  text-transform: uppercase; color: var(--clr-text-3); background: var(--clr-surface); padding: 9px 14px;
  border-bottom: 1px solid var(--clr-border); white-space: nowrap; }
.sg-bt td { padding: 7px 14px; border-bottom: 1px solid color-mix(in srgb, var(--clr-border) 55%, transparent);
  white-space: nowrap; vertical-align: middle; }
.sg-bt tbody tr { cursor: pointer; }
.sg-bt tbody tr:hover td { background: var(--clr-surface-2); }
.sg-bt tbody tr.root { font-weight: 700; }
.sg-bt tbody tr.root td { background: color-mix(in srgb, var(--clr-accent) 7%, transparent); }
.sg-bt tbody tr.ref .bt-title { color: var(--clr-text-2); font-style: italic; font-weight: 500; }
.bt-c-pos { color: var(--clr-text-3); font-variant-numeric: tabular-nums; font-size: 12px; width: 1%; }
.bt-item { display: flex; align-items: center; gap: 7px; min-width: 0; }
.bt-tog { flex-shrink: 0; width: 18px; height: 18px; display: inline-flex; align-items: center; justify-content: center;
  border: 0; background: none; padding: 0; color: var(--clr-text-3); cursor: pointer; border-radius: 5px;
  transition: color 0.12s, background 0.12s; }
.bt-tog svg { transition: transform 0.12s; }
.bt-tog.open svg { transform: rotate(90deg); }
.bt-tog:hover { color: var(--clr-accent); background: color-mix(in srgb, var(--clr-accent) 12%, transparent); }
.bt-tog-sp { flex-shrink: 0; width: 18px; }
.bt-hid { flex-shrink: 0; font-size: 10px; font-weight: 700; font-variant-numeric: tabular-nums; color: var(--clr-accent);
  background: color-mix(in srgb, var(--clr-accent) 14%, transparent); border-radius: 8px; padding: 1px 6px; }
.bt-lvl { color: var(--clr-text-3); font-variant-numeric: tabular-nums; font-size: 12px; }
/* Instance label (per-instance mode): the designator of this usage position. */
.bt-inst { flex-shrink: 0; font-size: 11px; font-weight: 750; font-variant-numeric: tabular-nums; color: var(--clr-accent);
  background: color-mix(in srgb, var(--clr-accent) 12%, transparent); border-radius: 5px; padding: 1px 6px; }
.bt-ind { flex-shrink: 0; }
.bt-title { font-weight: 600; overflow: hidden; text-overflow: ellipsis; max-width: 420px; }
.bt-refm { color: var(--clr-text-3); font-size: 12px; }
.bt-qty { font-weight: 750; color: var(--clr-accent); font-variant-numeric: tabular-nums; }
.bt-val { color: var(--clr-text-2); max-width: 300px; overflow: hidden; text-overflow: ellipsis; }
.bt-ver { font-weight: 700; font-variant-numeric: tabular-nums; font-size: 12px; }
.bt-ver.pinned { color: var(--clr-accent); }
.bt-ver.latest { color: var(--clr-text-3); }
.bt-vtag { font-size: 9px; font-weight: 700; letter-spacing: 0.4px; text-transform: uppercase; margin-left: 5px; }
.bt-vtag.pinned { color: color-mix(in srgb, var(--clr-accent) 72%, transparent); }
.bt-vtag.latest { color: var(--clr-text-3); opacity: 0.6; }
.bt-status { display: inline-flex; align-items: center; gap: 6px; }
.bt-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }

/* Empty */
.sg-empty { position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 14px; text-align: center; color: var(--clr-text-3); padding: 24px; }
.sg-empty-glyph { font-size: 52px; opacity: 0.35; line-height: 1; }
.sg-empty p { margin: 0; font-size: 14px; color: var(--clr-text-2); font-weight: 600; }
.sg-empty span { font-size: 13px; color: var(--clr-text-3); font-weight: 400; }
</style>
