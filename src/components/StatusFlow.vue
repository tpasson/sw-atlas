<template>
  <!-- Status state machine. Bidirectional transitions are merged into one double-
       headed line (→ the graph is a DAG). Auto layout = dagre columns with bowed
       arcs; drag any node to make a custom layout, then Save it (shared per type).
       Current is highlighted; reachable ones advance on click. -->
  <div class="sf-overlay" @click.self="$emit('close')">
    <div class="sf-pop">
      <div class="sf-head">
        <span class="sf-title">Status flow</span>
        <button type="button" class="sf-x" title="Close" @click="$emit('close')">×</button>
      </div>

      <div class="sf-canvas">
        <svg ref="svgEl" class="sf-svg" :class="{ editing: canEdit }" :viewBox="vb" preserveAspectRatio="xMidYMid meet">
          <defs>
            <marker id="sf-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7.5" markerHeight="7.5" orient="auto-start-reverse">
              <path d="M0,0 L10,5 L0,10 z" :fill="mutedColor" />
            </marker>
            <marker id="sf-arrow-on" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="8" markerHeight="8" orient="auto-start-reverse">
              <path d="M0,0 L10,5 L0,10 z" :fill="accentColor" />
            </marker>
          </defs>
          <path v-for="(e, i) in edges" :key="'e' + i" class="sf-link" :class="{ on: e.on }" :d="e.d"
            :marker-start="e.arrowStart ? (e.on ? 'url(#sf-arrow-on)' : 'url(#sf-arrow)') : null"
            :marker-end="e.arrowEnd ? (e.on ? 'url(#sf-arrow-on)' : 'url(#sf-arrow)') : null" />
          <template v-if="canEdit">
            <!-- Invisible grab targets: the line body bends the curve; each end
                 snaps its dock to the nearest box side. No visible dots. -->
            <path v-for="e in edges" :key="'hl' + e.id" class="sf-hit-line" :d="e.d" @mousedown.stop="onLineDown(e, $event)" />
            <circle v-for="e in edges" :key="'ea' + e.id" class="sf-hit-end" :cx="e.endA.x" :cy="e.endA.y" r="11" @mousedown.stop="onEndDown(e, 'a', $event)" />
            <circle v-for="e in edges" :key="'eb' + e.id" class="sf-hit-end" :cx="e.endB.x" :cy="e.endB.y" r="11" @mousedown.stop="onEndDown(e, 'b', $event)" />
          </template>
          <g v-for="n in nodes" :key="n.key" class="sf-node"
            :class="{ current: n.key === current, reachable: reachable.has(n.key), muted: n.key !== current && !reachable.has(n.key), draggable: canEdit }"
            :transform="`translate(${pos[n.key].x},${pos[n.key].y})`"
            @mousedown="onDown(n, $event)" @click="onClick(n)">
            <rect :x="-n.w / 2" :y="-n.h / 2" :width="n.w" :height="n.h" :rx="n.h / 2" :style="nodeStyle(n)" />
            <text text-anchor="middle" dy="0.32em" :style="{ fill: nodeText(n), fontWeight: n.key === current ? 700 : 600 }">{{ n.label }}</text>
          </g>
        </svg>
      </div>

      <div class="sf-foot">
        <p class="sf-hint">
          <template v-if="!canEdit">↔ = transition both ways.</template>
          <template v-else-if="reachable.size">Click a <strong>highlighted</strong> status to move there · drag nodes, lines or arrow ends to arrange · ↔ = both ways.</template>
          <template v-else>End state — no transitions · drag nodes, lines or arrow ends to arrange.</template>
        </p>
        <div v-if="canEdit" class="sf-actions">
          <button v-if="mode === 'custom' || hasSaved" type="button" class="sf-btn" @click="onReset">Reset to auto</button>
          <button type="button" class="sf-btn primary" :disabled="!dirty" @click="onSave">Save arrangement</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onBeforeUnmount } from 'vue'
import dagre from '@dagrejs/dagre'
import { toneColor } from '../stores/useAppStore.js'

const props = defineProps({
  statuses: { type: Array, required: true },
  current: { type: String, default: '' },
  readOnly: { type: Boolean, default: false },
  layout: { type: Object, default: null }, // saved { key: {x,y} } or null
})
const emit = defineEmits(['advance', 'close', 'saveLayout', 'resetLayout'])
const canEdit = computed(() => !props.readOnly)

const rootStyle = getComputedStyle(document.documentElement)
const accentColor = rootStyle.getPropertyValue('--clr-accent').trim() || '#0071E3'
const mutedColor = rootStyle.getPropertyValue('--clr-text-3').trim() || '#8a8a8e'

const keys = new Set(props.statuses.map(s => s.key))
const cur = props.statuses.find(s => s.key === props.current)
const reachable = new Set((cur?.to || []).filter(k => keys.has(k)))

// Merge each connected pair into one edge (both directions → double arrow). In
// status-list order every edge points forward → the graph is a DAG.
const toSet = new Map(props.statuses.map(s => [s.key, new Set((s.to || []).filter(k => keys.has(k) && k !== s.key))]))
const pairs = []
for (let i = 0; i < props.statuses.length; i++) {
  for (let j = i + 1; j < props.statuses.length; j++) {
    const a = props.statuses[i].key, b = props.statuses[j].key
    const ab = toSet.get(a).has(b), ba = toSet.get(b).has(a)
    if (ab || ba) pairs.push({ a, b, arrowEnd: ab, arrowStart: ba })
  }
}

const NODE_H = 32
const nodes = props.statuses.map(s => {
  const label = s.label || s.key
  return { key: s.key, label, tone: s.tone || 'neutral', w: Math.max(74, label.length * 8 + 30), h: NODE_H }
})
const nodeByKey = new Map(nodes.map(n => [n.key, n]))

// ── Auto layout (dagre) — computed once, used as the default + for "reset". ─────
const g = new dagre.graphlib.Graph()
g.setGraph({ rankdir: 'LR', nodesep: 32, ranksep: 104, marginx: 12, marginy: 12 })
g.setDefaultEdgeLabel(() => ({}))
for (const n of nodes) g.setNode(n.key, { width: n.w, height: n.h })
for (const p of pairs) g.setEdge(p.a, p.b)
dagre.layout(g)
const autoPos = {}
for (const n of nodes) { const gn = g.node(n.key); autoPos[n.key] = { x: gn.x, y: gn.y } }
const xs = [...new Set(nodes.map(n => Math.round(autoPos[n.key].x)))].sort((a, b) => a - b)
const rankOf = new Map(nodes.map(n => [n.key, xs.indexOf(Math.round(autoPos[n.key].x))]))
// Fan slots for auto routing (source right / target left).
const autoFrac = new Map()
{
  const bySide = new Map()
  const put = (k, side, i, role, oy) => { const key = k + '|' + side; (bySide.get(key) || bySide.set(key, []).get(key)).push({ i, role, oy }) }
  pairs.forEach((p, i) => { put(p.a, 'R', i, 'src', autoPos[p.b].y); put(p.b, 'L', i, 'tgt', autoPos[p.a].y) })
  for (const [, arr] of bySide) { arr.sort((x, y) => x.oy - y.oy); arr.forEach((it, i) => autoFrac.set(it.i + '|' + it.role, (i + 1) / (arr.length + 1))) }
}

// ── Reactive state: node positions + edge shaping + mode ─────────────────────
const pos = reactive({})   // node key → {x,y} centre
const wp = reactive({})    // edge id "a|b" → {x,y} bend point (custom layout only)
const ends = reactive({})  // edge id → { a:side, b:side } forced dock sides T/R/B/L
const mode = ref('auto')
const dirty = ref(false)
const edgeId = p => p.a + '|' + p.b
// Saved layout shape is { nodes:{key:{x,y}}, edges:{id:{x?,y?,a?,b?}} }; tolerate
// the old flat { key:{x,y} } (nodes only) too.
const savedNodes = props.layout ? (props.layout.nodes || props.layout) : null
const savedEdges = (props.layout && props.layout.edges) || {}
const hasSaved = !!(savedNodes && Object.keys(savedNodes).length)
function clearEdges() { for (const k of Object.keys(wp)) delete wp[k]; for (const k of Object.keys(ends)) delete ends[k] }
function applyAuto() { mode.value = 'auto'; clearEdges(); for (const n of nodes) pos[n.key] = { ...autoPos[n.key] } }
function applyLayout(nodesL, edgesL) {
  mode.value = 'custom'; clearEdges()
  for (const n of nodes) pos[n.key] = nodesL[n.key] ? { ...nodesL[n.key] } : { ...autoPos[n.key] }
  for (const p of pairs) {
    const e = edgesL[edgeId(p)]; if (!e) continue
    if (e.x != null && e.y != null) wp[edgeId(p)] = { x: e.x, y: e.y }
    if (e.a || e.b) ends[edgeId(p)] = { a: e.a || null, b: e.b || null }
  }
}
if (hasSaved) applyLayout(savedNodes, savedEdges); else applyAuto()

// ── Edge routing + viewBox (reactive on positions) ──────────────────────────
const P = p => `${p.x.toFixed(1)},${p.y.toFixed(1)}`
// Border point + outward normal of a rect toward (tx,ty).
function border(c, w, h, tx, ty) {
  const dx = tx - c.x, dy = ty - c.y
  if (!dx && !dy) return { p: { x: c.x, y: c.y }, n: { x: 1, y: 0 } }
  const sx = dx ? (w / 2) / Math.abs(dx) : Infinity
  const sy = dy ? (h / 2) / Math.abs(dy) : Infinity
  const s = Math.min(sx, sy)
  return { p: { x: c.x + dx * s, y: c.y + dy * s }, n: sx < sy ? { x: Math.sign(dx), y: 0 } : { x: 0, y: Math.sign(dy) } }
}
function anchorSide(c, w, h, side, f) {
  const t = f - 0.5
  return side === 'R' ? { x: c.x + w / 2, y: c.y + t * (h - 8) } : { x: c.x - w / 2, y: c.y + t * (h - 8) }
}
// Centre of a chosen box side + its outward normal (for forced dock sides).
function sideAnchor(c, w, h, side) {
  if (side === 'T') return { p: { x: c.x, y: c.y - h / 2 }, n: { x: 0, y: -1 } }
  if (side === 'B') return { p: { x: c.x, y: c.y + h / 2 }, n: { x: 0, y: 1 } }
  if (side === 'L') return { p: { x: c.x - w / 2, y: c.y }, n: { x: -1, y: 0 } }
  return { p: { x: c.x + w / 2, y: c.y }, n: { x: 1, y: 0 } } // 'R'
}

const layoutData = computed(() => {
  const bb = []
  const edges = pairs.map((p, i) => {
    const S = nodeByKey.get(p.a), T = nodeByKey.get(p.b), cS = pos[p.a], cT = pos[p.b]
    const on = (p.a === props.current && p.arrowEnd) || (p.b === props.current && p.arrowStart)
    const id = edgeId(p)
    let a, c1, c2, b
    if (mode.value === 'auto') {
      a = anchorSide(cS, S.w, S.h, 'R', autoFrac.get(i + '|src'))
      b = anchorSide(cT, T.w, T.h, 'L', autoFrac.get(i + '|tgt'))
      const dx = b.x - a.x
      const hl = Math.max(30, Math.abs(dx) * 0.5)
      const span = rankOf.get(p.b) - rankOf.get(p.a)
      const lift = span > 1 ? Math.min(38 + span * 22, 132) : 0
      c1 = { x: a.x + hl, y: a.y - lift }; c2 = { x: b.x - hl, y: b.y - lift }
    } else {
      // Forced dock side (from dragging an end) or auto border toward the other box.
      const ed = ends[id]
      const A = ed?.a ? sideAnchor(cS, S.w, S.h, ed.a) : border(cS, S.w, S.h, cT.x, cT.y)
      const B = ed?.b ? sideAnchor(cT, T.w, T.h, ed.b) : border(cT, T.w, T.h, cS.x, cS.y)
      a = A.p; b = B.p
      const hl = Math.max(24, Math.hypot(b.x - a.x, b.y - a.y) * 0.35)
      c1 = { x: a.x + A.n.x * hl, y: a.y + A.n.y * hl }; c2 = { x: b.x + B.n.x * hl, y: b.y + B.n.y * hl }
      // Bend the curve through a dragged waypoint: shift both controls so the
      // Bézier midpoint (t=0.5) lands on it. dM = 0.75·dControls → dControls = 4/3·dM.
      const w = wp[id]
      if (w) {
        const m0x = (a.x + 3 * c1.x + 3 * c2.x + b.x) / 8, m0y = (a.y + 3 * c1.y + 3 * c2.y + b.y) / 8
        const sx = (w.x - m0x) * 4 / 3, sy = (w.y - m0y) * 4 / 3
        c1 = { x: c1.x + sx, y: c1.y + sy }; c2 = { x: c2.x + sx, y: c2.y + sy }
      }
    }
    bb.push(a, b, c1, c2)
    return {
      d: `M${P(a)} C${P(c1)} ${P(c2)} ${P(b)}`, on, arrowStart: p.arrowStart, arrowEnd: p.arrowEnd, id,
      endA: { x: a.x, y: a.y, nodeKey: p.a }, endB: { x: b.x, y: b.y, nodeKey: p.b },
    }
  })
  let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity
  for (const n of nodes) { const c = pos[n.key]; minX = Math.min(minX, c.x - n.w / 2); maxX = Math.max(maxX, c.x + n.w / 2); minY = Math.min(minY, c.y - n.h / 2); maxY = Math.max(maxY, c.y + n.h / 2) }
  for (const p of bb) { minX = Math.min(minX, p.x); maxX = Math.max(maxX, p.x); minY = Math.min(minY, p.y); maxY = Math.max(maxY, p.y) }
  return { edges, minX, minY, maxX, maxY }
})
const edges = computed(() => layoutData.value.edges)
// The viewBox is NOT bound live to the content: refitting it mid-drag would shift
// the cursor→SVG mapping and make the dragged node run away. We refit only on
// init, drag-end and reset; during a drag it stays frozen (stable dragging).
const vb = ref('0 0 400 260')
function refit() {
  const L = layoutData.value, PAD = 20
  vb.value = `${(L.minX - PAD).toFixed(1)} ${(L.minY - PAD).toFixed(1)} ${(L.maxX - L.minX + 2 * PAD).toFixed(1)} ${(L.maxY - L.minY + 2 * PAD).toFixed(1)}`
}
refit()

// ── Node visuals ─────────────────────────────────────────────────────────────
function nodeStyle(n) {
  const c = toneColor(n.tone)
  if (n.key === props.current) return { fill: c, stroke: c, strokeWidth: '2.5' }
  if (reachable.has(n.key)) return { fill: c + '22', stroke: c, strokeWidth: '1.5' }
  return { fill: 'var(--clr-surface-2)', stroke: 'var(--clr-border)', strokeWidth: '1' }
}
function nodeText(n) {
  if (n.key === props.current) return '#fff'
  if (reachable.has(n.key)) return toneColor(n.tone)
  return 'var(--clr-text-3)'
}

// ── Drag (editors) + click-to-advance ────────────────────────────────────────
const svgEl = ref(null)
let drag = null
function svgPt(e) {
  const r = svgEl.value.getBoundingClientRect()
  const [x0, y0, w, h] = vb.value.split(' ').map(Number)
  return { x: x0 + (e.clientX - r.left) / r.width * w, y: y0 + (e.clientY - r.top) / r.height * h }
}
function startDrag(e) {
  window.addEventListener('mousemove', onMove)
  window.addEventListener('mouseup', onUp)
  e.preventDefault()
}
function onDown(n, e) {
  if (!canEdit.value) return
  const p = svgPt(e)
  drag = { kind: 'node', key: n.key, moved: false, ox: p.x - pos[n.key].x, oy: p.y - pos[n.key].y }
  startDrag(e)
}
function onLineDown(ed, e) {
  if (!canEdit.value) return
  drag = { kind: 'line', id: ed.id, moved: false }
  startDrag(e)
}
function onEndDown(ed, end, e) {
  if (!canEdit.value) return
  drag = { kind: 'end', id: ed.id, end, nodeKey: end === 'a' ? ed.endA.nodeKey : ed.endB.nodeKey, moved: false }
  startDrag(e)
}
// Nearest box side to a point (rasterised dock): compare normalised offsets.
function nearestSide(nodeKey, px, py) {
  const c = pos[nodeKey], n = nodeByKey.get(nodeKey)
  const dx = (px - c.x) / (n.w / 2), dy = (py - c.y) / (n.h / 2)
  return Math.abs(dx) >= Math.abs(dy) ? (dx >= 0 ? 'R' : 'L') : (dy >= 0 ? 'B' : 'T')
}
function onMove(e) {
  if (!drag) return
  const p = svgPt(e)
  if (drag.kind === 'node') {
    const nx = p.x - drag.ox, ny = p.y - drag.oy
    if (Math.hypot(nx - pos[drag.key].x, ny - pos[drag.key].y) > 2) drag.moved = true
    pos[drag.key] = { x: nx, y: ny }
  } else if (drag.kind === 'line') {
    drag.moved = true                       // grab the line anywhere to bend it
    wp[drag.id] = { x: p.x, y: p.y }
  } else {                                  // 'end' — snap the dock to the nearest side
    drag.moved = true
    ends[drag.id] = { ...(ends[drag.id] || { a: null, b: null }), [drag.end]: nearestSide(drag.nodeKey, p.x, p.y) }
  }
  if (drag.moved) { mode.value = 'custom'; dirty.value = true }
}
function onUp() {
  window.removeEventListener('mousemove', onMove)
  window.removeEventListener('mouseup', onUp)
  const d = drag; drag = null
  if (!d) return
  if (d.moved) refit()                                    // re-fit the frozen viewBox once the drag ends
  else if (d.kind === 'node') advance(nodeByKey.get(d.key)) // press without drag = click
}
function onClick(n) { if (!canEdit.value) advance(n) } // editors advance via onUp (drag path)
function advance(n) { if (!props.readOnly && reachable.has(n.key)) emit('advance', n.key) }

function onSave() {
  const nodesOut = {}, edgesOut = {}
  for (const n of nodes) nodesOut[n.key] = { x: Math.round(pos[n.key].x), y: Math.round(pos[n.key].y) }
  for (const p of pairs) {
    const id = edgeId(p), e = {}
    if (wp[id]) { e.x = Math.round(wp[id].x); e.y = Math.round(wp[id].y) }
    if (ends[id]?.a) e.a = ends[id].a
    if (ends[id]?.b) e.b = ends[id].b
    if (Object.keys(e).length) edgesOut[id] = e
  }
  emit('saveLayout', { nodes: nodesOut, edges: edgesOut })
  dirty.value = false
}
function onReset() {
  applyAuto()
  refit()
  dirty.value = false
  emit('resetLayout')
}

function onKey(e) { if (e.key === 'Escape') emit('close') }
window.addEventListener('keydown', onKey)
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey)
  window.removeEventListener('mousemove', onMove)
  window.removeEventListener('mouseup', onUp)
})
</script>

<style scoped>
.sf-overlay { position: fixed; inset: 0; z-index: 1000; display: flex; align-items: center; justify-content: center; background: rgba(0,0,0,0.32); }
.sf-pop { width: 640px; max-width: calc(100vw - 32px); background: var(--clr-surface); border: 1px solid var(--clr-border-light); border-radius: var(--r-lg); box-shadow: var(--sh-modal); overflow: hidden; }
.sf-head { display: flex; align-items: center; justify-content: space-between; padding: 12px 16px; border-bottom: 1px solid var(--clr-border-light); }
.sf-title { font-size: 13px; font-weight: 700; color: var(--clr-text); }
.sf-x { width: 26px; height: 26px; display: inline-flex; align-items: center; justify-content: center; font-size: 18px; line-height: 1; color: var(--clr-text-3); border-radius: var(--r-sm); background: none; }
.sf-x:hover { background: var(--clr-surface-2); color: var(--clr-text); }
.sf-canvas { background: var(--clr-bg); padding: 18px; display: flex; justify-content: center; }
.sf-svg { display: block; width: 100%; max-height: 60vh; height: auto; }
.sf-svg.editing { cursor: default; }
.sf-link { fill: none; stroke: var(--clr-border); stroke-width: 1.6; }
.sf-link.on { stroke: var(--clr-accent); stroke-width: 2.2; }
/* Invisible grab targets (no visible dots): wide stroke over the line = bend,
   circle at each end = re-dock. */
.sf-hit-line { fill: none; stroke: transparent; stroke-width: 14; pointer-events: stroke; cursor: grab; }
.sf-hit-line:active { cursor: grabbing; }
.sf-hit-end { fill: transparent; pointer-events: all; cursor: grab; }
.sf-hit-end:active { cursor: grabbing; }
.sf-node text { font-size: 12px; pointer-events: none; user-select: none; }
.sf-node.reachable { cursor: pointer; }
.sf-node.draggable { cursor: grab; }
.sf-node.draggable:active { cursor: grabbing; }
.sf-node.reachable rect { transition: filter 0.12s; }
.sf-node.reachable:hover rect { filter: brightness(1.12); }
.sf-node.current rect { animation: sf-currentpulse 2s ease-in-out infinite; }
.sf-foot { display: flex; align-items: center; gap: 12px; padding: 10px 16px; border-top: 1px solid var(--clr-border-light); }
.sf-hint { margin: 0; flex: 1; font-size: 12px; color: var(--clr-text-3); }
.sf-hint strong { color: var(--clr-text-2); font-weight: 600; }
.sf-actions { display: inline-flex; gap: 8px; flex-shrink: 0; }
.sf-btn { font-size: 12px; font-weight: 600; color: var(--clr-text-2); background: var(--clr-surface-2); border-radius: var(--r-md); padding: 6px 12px; }
.sf-btn:hover { background: var(--clr-surface); color: var(--clr-text); }
.sf-btn.primary { color: #fff; background: var(--clr-accent); }
.sf-btn.primary:hover { background: var(--clr-accent-hover); }
.sf-btn:disabled { opacity: 0.5; cursor: default; }
.sf-btn.primary:disabled { background: var(--clr-accent); }
@keyframes sf-currentpulse { 0%, 100% { filter: brightness(1.04); } 50% { filter: brightness(1.16); } }
</style>
