<template>
  <div class="table-wrap" ref="wrapEl">
    <table
      class="ms-table"
      :class="['bm-' + settings.items.borderMode, { 'has-focus': !!activeMs, 'is-readonly': props.readOnly }]"
      :style="{
        width: tableWidth + 'px',
        '--it-font': settings.items.fontSize + 'px',
        '--it-weight': settings.items.fontWeight,
        '--it-pad': settings.items.padding + 'px',
        '--it-radius': settings.items.radius + 'px',
        '--it-ring': settings.items.border + 'px',
        '--it-label-y': settings.items.labelOffset + 'px',
        '--it-gap': settings.items.iconGap + 'px',
      }"
    >
      <thead>
        <tr class="head-months" ref="headRowEl">
          <th class="th-area">Area</th>
          <th class="th-sub">Sub-Area</th>
          <th
            v-for="(m, i) in MONTHS"
            :key="i"
            class="th-month"
            :class="{ 'th-current': isCurrentMonth(i + 1) }"
            :style="{ width: COL_W + 'px' }"
          >
            {{ m }}
          </th>
          <th class="th-gutter" :style="{ width: gutterW + 'px' }"></th>
        </tr>
        <tr v-if="settings.weekNumbers.enabled" class="head-weeks" :style="{ '--wk-top': monthHeadH + 'px' }">
          <th class="th-area wk-corner"></th>
          <th class="th-sub wk-corner"><span class="wk-kw">CW</span></th>
          <th class="wk-cell" colspan="13">
            <span
              v-for="w in weekTicks"
              :key="w.n + '-' + w.x"
              class="wk-num"
              :style="{ left: w.x + 'px' }"
            >{{ w.n }}</span>
          </th>
        </tr>
      </thead>
      <tbody>
        <template v-for="g in grid" :key="rowKey(g.row)">
          <tr>
            <td
              v-if="g.row.showLaneCell"
              :rowspan="g.row.rowspan"
              class="td-area"
              :style="{ '--lane': g.row.swimlane.color }"
            >
              <div class="area-label">
                <span class="area-dot" :style="{ background: g.row.swimlane.color }"></span>
                <span class="area-name">{{ g.row.swimlane.name }}</span>
              </div>
            </td>

            <td class="td-sub">
              <span class="sub-name">{{ g.row.subLane?.name ?? '' }}</span>
            </td>

            <!-- One track per row spanning all 12 fixed-width months. Items are
                 positioned by date and stacked into lanes to avoid collisions. -->
            <td
              class="track"
              colspan="13"
              :style="{ height: g.trackHeight + 'px', '--col': COL_W + 'px' }"
              @click="props.readOnly ? null : onTrackClick(g.row, $event)"
              @mousemove="onTrackMove(rowKey(g.row), $event)"
              @mouseleave="addHint.key = null"
            >
              <template v-if="currentMonthIndex >= 0">
                <div
                  v-if="settings.monthHighlight.enabled"
                  class="month-now"
                  :style="{ left: currentMonthIndex * COL_W + 'px', width: COL_W + 'px', background: monthHlColor }"
                ></div>
              </template>

              <!-- Crisp gridlines (week lines first so month lines paint on top) -->
              <div
                v-for="ln in weekLineDivs"
                :key="'wl' + ln.key"
                class="grid-line"
                :style="{ left: ln.left + 'px', width: ln.w + 'px', background: ln.color }"
              ></div>
              <div
                v-for="ln in monthLineDivs"
                :key="'ml' + ln.key"
                class="grid-line"
                :style="{ left: ln.left + 'px', width: ln.w + 'px', background: ln.color }"
              ></div>
              <div
                class="grid-line"
                :style="{ left: closeLine.left + 'px', width: closeLine.w + 'px', background: closeLine.color }"
              ></div>

              <template v-if="currentMonthIndex >= 0">
                <div
                  v-if="settings.dayLine.enabled"
                  class="day-line"
                  :style="{ left: todayX + 'px', width: settings.dayLine.width + 'px', background: dayLineColor }"
                ></div>
              </template>

              <template v-for="it in g.items" :key="it.key">
                <!-- Milestone: marker is the anchor at its day, label flows right -->
                <div
                  v-if="it.type === 'point'"
                  class="mk-item"
                  :data-item-id="it.m.id"
                  :class="[chipState(it.m), baselineClass(it.m)]"
                  :style="{ left: (it.x - 9 - settings.items.padding) + 'px', top: (it.lane * g.laneH + g.vOffset) + 'px', color: g.row.swimlane.color }"
                  @mouseenter="hoveredMs = it.m"
                  @mouseleave="hoveredMs = null"
                  @click.stop="onChipClick($event, it.m, g.row.swimlane.color)"
                  @dblclick.stop="!props.readOnly && !it.m.sourceSystem && $emit('edit-milestone', it.m)"
                >
                  <MarkerIcon :shape="markerOf(it.m)" :color="g.row.swimlane.color" :size="settings.items.markerSize" :stroke-width="settings.items.markerStroke" :fill="markerFill(markerOf(it.m))" class="mk-icon" />
                  <span v-if="it.m.sourceSystem" class="chip-lock" title="Synced — read-only">🔒</span><span class="mk-label">{{ it.m.title }}</span><AlertTriangle v-if="riskIds.has(it.m.id)" class="risk-badge" :size="settings.items.markerSize" :stroke-width="settings.items.markerStroke" color="#FF3B30" />
                </div>

                <!-- Event: bar from start to end date -->
                <div
                  v-else-if="it.type === 'bar'"
                  class="event-bar"
                  :data-item-id="it.m.id"
                  :class="[chipState(it.m), baselineClass(it.m), { draggable: !props.readOnly && !it.m.sourceSystem }]"
                  :style="barStyleFull(it, g.row.swimlane.color, g.laneH, g.vOffset)"
                  @mouseenter="hoveredMs = it.m"
                  @mouseleave="hoveredMs = null"
                  @pointerdown="startDrag($event, it, 'move')"
                  @click.stop="onChipClick($event, it.m, g.row.swimlane.color)"
                  @dblclick.stop="!props.readOnly && !it.m.sourceSystem && $emit('edit-milestone', it.m)"
                >
                  <span v-if="it.continuesLeft" class="bar-arrow">◀</span>
                  <span class="bar-title" :class="{ 'bar-title-out': it.labelOutside }">
                    <MarkerIcon
                      v-if="it.m.marker && it.m.marker !== 'bar'"
                      :shape="it.m.marker"
                      :fill="markerFill(it.m.marker)"
                      :color="g.row.swimlane.color"
                      :size="settings.items.markerSize"
                      :stroke-width="settings.items.markerStroke"
                      class="bar-marker"
                    />
                    <span v-if="it.m.sourceSystem" class="chip-lock" title="Synced — read-only">🔒</span>{{ it.m.title }}<AlertTriangle v-if="riskIds.has(it.m.id)" class="risk-badge" :size="settings.items.markerSize" :stroke-width="settings.items.markerStroke" color="#FF3B30" />
                  </span>
                  <span v-if="it.continuesRight" class="bar-arrow">▶</span>
                  <template v-if="!props.readOnly && !it.m.sourceSystem">
                    <span v-if="!it.continuesLeft" class="bar-handle bar-handle-l" @pointerdown.stop="startDrag($event, it, 'resize-l')" @click.stop @dblclick.stop></span>
                    <span v-if="!it.continuesRight" class="bar-handle bar-handle-r" @pointerdown.stop="startDrag($event, it, 'resize-r')" @click.stop @dblclick.stop></span>
                  </template>
                </div>

                <!-- Baseline ghost (old/removed position) -->
                <div
                  v-else
                  class="mk-item ghost"
                  :class="[it.ghostType === 'removed' ? 'ghost-removed' : 'ghost-moved']"
                  :style="{ left: (it.x - 9 - settings.items.padding) + 'px', top: (it.lane * g.laneH + g.vOffset) + 'px' }"
                  :title="it.ghostType === 'removed' ? 'Removed since baseline' : 'Baseline position (moved)'"
                >
                  <span class="mk-label">{{ it.gh.title }}</span>
                </div>
              </template>

              <span
                v-if="!props.readOnly && addHint.key === rowKey(g.row) && !hoveredMs"
                class="track-add-hint"
                :style="{ left: addHint.x + 'px' }"
              >+</span>
            </td>
          </tr>
        </template>

        <tr v-if="tableRows.length === 0">
          <td :colspan="15" class="empty-state">
            No areas defined yet. Click "Settings" to get started.
          </td>
        </tr>
      </tbody>
    </table>

    <div class="track-spacer"></div>

    <div class="legend">
      <span v-for="(m, i) in settings.markers" :key="m.shape + i" class="legend-item">
        <MarkerIcon :shape="m.shape" color="#8a8a8e" :size="settings.items.markerSize" :stroke-width="settings.items.markerStroke" :fill="m.fill" /> {{ m.label }}
      </span>
      <span class="legend-item"><span class="legend-bar"></span> {{ settings.eventLabel }}</span>
    </div>
  </div>

  <!-- Tooltip -->
  <Teleport to="body">
    <Transition name="tooltip">
      <div
        v-if="tooltip.visible && tooltip.ms"
        class="ms-tooltip"
        :style="tooltipStyle"
        @click.stop
      >
        <div class="tooltip-header">
          <span class="tooltip-dot" :style="{ background: tooltip.color }"></span>
          <span class="tooltip-title">{{ tooltip.ms.title }}</span>
        </div>
        <div class="tooltip-fields">
          <div v-if="tooltip.ms.what" class="tooltip-field">
            <span class="tf-label">What</span>
            <span class="tf-val">{{ tooltip.ms.what }}</span>
          </div>
          <div v-if="tooltip.ms.why" class="tooltip-field">
            <span class="tf-label">Why</span>
            <span class="tf-val">{{ tooltip.ms.why }}</span>
          </div>
          <div v-if="tooltip.ms.how" class="tooltip-field">
            <span class="tf-label">Where</span>
            <span class="tf-val">{{ tooltip.ms.how }}</span>
          </div>
          <div v-if="tooltip.ms.who" class="tooltip-field">
            <span class="tf-label">Who</span>
            <span class="tf-val">{{ tooltip.ms.who }}</span>
          </div>
          <div v-if="tooltip.ms.when" class="tooltip-field">
            <span class="tf-label">When</span>
            <span class="tf-val">{{ formatDate(tooltip.ms.when) }}</span>
          </div>
          <div v-if="tooltip.ms.endDate" class="tooltip-field">
            <span class="tf-label">End</span>
            <span class="tf-val">{{ formatDate(tooltip.ms.endDate) }}</span>
          </div>
          <div v-if="tooltip.ms.sourceSystem" class="tooltip-field">
            <span class="tf-label">Source</span>
            <span class="tf-val">🔒 Synced from {{ tooltip.ms.sourceSystem }} (read-only)</span>
          </div>
        </div>
        <div v-if="linkedMilestones.length > 0" class="tooltip-links">
          <span class="tl-label">Prerequisites</span>
          <div class="tl-items">
            <div v-for="lm in linkedMilestones.slice(0, 10)" :key="lm.id" class="tl-item" @click.stop="selectFromTooltip(lm, $event)">
              <span class="tl-dot" :style="{ background: swimlaneColor(lm.swimlaneId) }"></span>
              <span class="tl-title">{{ lm.title }}</span>
              <span v-if="lm.when || lm.startDate" class="tl-date">{{ formatDate(lm.when || lm.startDate) }}</span>
            </div>
            <div v-if="linkedMilestones.length > 10" class="tl-more">
              +{{ linkedMilestones.length - 10 }} more
            </div>
          </div>
        </div>

        <div v-if="parentMilestones.length > 0" class="tooltip-links">
          <span class="tl-label">Required by</span>
          <div class="tl-items">
            <div v-for="pm in parentMilestones.slice(0, 10)" :key="pm.id" class="tl-item" @click.stop="selectFromTooltip(pm, $event)">
              <span class="tl-dot" :style="{ background: swimlaneColor(pm.swimlaneId) }"></span>
              <span class="tl-title">{{ pm.title }}</span>
              <span v-if="pm.when || pm.startDate" class="tl-date">{{ formatDate(pm.when || pm.startDate) }}</span>
            </div>
            <div v-if="parentMilestones.length > 10" class="tl-more">
              +{{ parentMilestones.length - 10 }} more
            </div>
          </div>
        </div>
        <div v-if="riskByItem[tooltip.ms.id] && riskByItem[tooltip.ms.id].length" class="tooltip-risk">
          <span class="tr-label">⚠ Late prerequisites</span>
          <div class="tr-items">
            <div v-for="p in riskByItem[tooltip.ms.id]" :key="p.id" class="tl-item" @click.stop="selectFromTooltip(p, $event)">
              <span class="tl-dot" :style="{ background: swimlaneColor(p.swimlaneId) }"></span>
              <span class="tl-title">{{ p.title }}</span>
              <span v-if="p.when || p.startDate" class="tl-date">{{ formatDate(p.when || p.startDate) }}</span>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>

  <!-- Live preview while dragging / resizing an event bar -->
  <Teleport to="body">
    <div v-if="dragTip" class="drag-tip" :style="dragTipStyle">
      <div class="dt-range">{{ formatShort(dragTip.s) }} → {{ formatShort(dragTip.en) }}</div>
      <div class="dt-meta">
        <span class="dt-days">{{ dragTip.days }} {{ dragTip.days === 1 ? 'day' : 'days' }}</span>
        <span v-if="dragTip.deltaDays" class="dt-delta" :class="{ neg: dragTip.deltaDays < 0 }">
          {{ dragTip.deltaDays > 0 ? '+' : '' }}{{ dragTip.deltaDays }}d
        </span>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch, nextTick } from 'vue'
import { useAppStore, MONTHS, store, baselineDiff, settings, groups, ui, riskIds, riskByItem } from '../stores/useAppStore.js'
import { AlertTriangle } from 'lucide-vue-next'
import MarkerIcon from './MarkerIcon.vue'

// Fixed geometry — months are a fixed width so date math is exact regardless of
// label length (labels overflow freely to the right).
const AREA_W = 168
const SUB_W = 148
const MIN_COL_W = 72
// Lane heights scale with the item appearance. Event rows are slightly taller
// (events carry a real border); milestone-only rows are tighter.
// Per-lane height = real pill height + ITEM_AIR px breathing room top & bottom.
// Pill height = tallest content (marker or text, plus a small line-height/icon
// overshoot) + vertical padding (+ border on events). The item is centred in
// its lane so any estimation error splits evenly between top and bottom.
const ITEM_AIR = 2
const LH_FUDGE = 2
const msPillH = computed(() =>
  Math.round(Math.max(settings.items.fontSize, settings.items.markerSize) + settings.items.padding * 2 + LH_FUDGE)
)
const eventPillH = computed(() =>
  Math.round(Math.max(settings.items.fontSize, settings.items.markerSize) + settings.items.padding * 2 + settings.items.border * 2 + LH_FUDGE)
)
const msLaneH = computed(() => Math.max(20, msPillH.value + ITEM_AIR * 2))
const eventLaneH = computed(() => Math.max(20, eventPillH.value + ITEM_AIR * 2))
const DAYS_PER_COL = 30.4 // avg days/month for px↔day conversion when dragging
// Reserved space right of December so end-of-year labels flow right into a
// defined gutter instead of creating dead horizontal scroll space.
const RIGHT_PAD = 200

const props = defineProps({
  zoom: { type: Number, default: 1 },
  readOnly: { type: Boolean, default: false },
})
const emit = defineEmits(['add-milestone', 'edit-milestone'])
const { getLinkedIds, dependsOnIds, dependentIds, updateMilestone } = useAppStore()

// Months fill the available width at 100% zoom; the zoom control then widens the
// months (horizontal detail) WITHOUT scaling the header height or fonts.
const wrapEl = ref(null)
const wrapW = ref(1200)
const headRowEl = ref(null)     // months header row — measured to stick the week row below it
const monthHeadH = ref(44)
let resizeObs = null
const baseColW = computed(() => Math.max(MIN_COL_W, (wrapW.value - AREA_W - SUB_W) / 12))
const COL_W = computed(() => baseColW.value * props.zoom)
const MONTHS_W = computed(() => COL_W.value * 12)
// The right gutter grows to fill the viewport when zoomed out, so the grid/rows
// reach the edge instead of leaving the page background showing.
const gutterW = computed(() => Math.max(RIGHT_PAD, wrapW.value - AREA_W - SUB_W - MONTHS_W.value))
const tableWidth = computed(() => AREA_W + SUB_W + MONTHS_W.value + gutterW.value)

// ── Rows ────────────────────────────────────────────────────────────────────
const tableRows = computed(() => {
  const rows = []
  for (const sw of store.swimlanes) {
    if (sw.subLanes.length === 0) {
      rows.push({ swimlane: sw, subLane: null, showLaneCell: true, rowspan: 1 })
    } else {
      sw.subLanes.forEach((sub, i) => {
        rows.push({ swimlane: sw, subLane: sub, showLaneCell: i === 0, rowspan: sw.subLanes.length })
      })
    }
  }
  return rows
})
function rowKey(row) {
  return `${row.swimlane.id}-${row.subLane?.id ?? 'root'}`
}

// ── Date → pixel mapping ──────────────────────────────────────────────────────
function ymOf(dateStr) {
  const [y, mo, day] = dateStr.split('-').map(Number)
  return { y, mo, day: day || 1 }
}
function daysInMonth(y, mo) {
  return new Date(y, mo, 0).getDate()
}
// x within the month track (0..MONTHS_W) for a date in the viewed year, clamped.
function dateX(dateStr) {
  if (!dateStr) return 0
  const { y, mo, day } = ymOf(dateStr)
  if (y < store.year) return 0
  if (y > store.year) return MONTHS_W.value
  return ((mo - 1) + (day - 1) / daysInMonth(store.year, mo)) * COL_W.value
}
// Same mapping as dateX but for a Date object (used for week ticks).
function dateXOf(dt) {
  const y = dt.getFullYear(), mo = dt.getMonth() + 1, day = dt.getDate()
  if (y < store.year) return 0
  if (y > store.year) return MONTHS_W.value
  return ((mo - 1) + (day - 1) / daysInMonth(store.year, mo)) * COL_W.value
}
function isoWeek(dt) {
  const d = new Date(Date.UTC(dt.getFullYear(), dt.getMonth(), dt.getDate()))
  const dayNum = (d.getUTCDay() + 6) % 7
  d.setUTCDate(d.getUTCDate() - dayNum + 3) // shift to the week's Thursday
  const firstThursday = new Date(Date.UTC(d.getUTCFullYear(), 0, 4))
  const fDayNum = (firstThursday.getUTCDay() + 6) % 7
  firstThursday.setUTCDate(firstThursday.getUTCDate() - fDayNum + 3)
  return 1 + Math.round((d - firstThursday) / (7 * 86400000))
}
// Inverse of dateXOf: pixel x within the months area → a Date in the viewed year.
function xToDate(x) {
  const clamped = Math.max(0, Math.min(MONTHS_W.value - 0.001, x))
  const mi = Math.floor(clamped / COL_W.value)
  const frac = (clamped - mi * COL_W.value) / COL_W.value
  const dim = daysInMonth(store.year, mi + 1)
  const day = Math.min(dim, Math.floor(frac * dim) + 1)
  return new Date(store.year, mi, day)
}
function mondayOf(dt) {
  const off = (dt.getDay() + 6) % 7
  return new Date(dt.getFullYear(), dt.getMonth(), dt.getDate() - off)
}
function fmtDate(dt) {
  const m = String(dt.getMonth() + 1).padStart(2, '0')
  const d = String(dt.getDate()).padStart(2, '0')
  return `${dt.getFullYear()}-${m}-${d}`
}
function anchorDate(m) {
  return m.when || `${m.year}-${String(m.month).padStart(2, '0')}-01`
}
function estTextW(t) {
  // Scale with the configured font size (~0.58 × fontSize per character).
  return Math.ceil((t ? t.length : 0) * settings.items.fontSize * 0.58)
}
function markerOf(m) {
  if (m.marker && m.marker !== 'bar') return m.marker
  return m.kind === 'event' ? 'flag' : 'diamond'
}
// Whether a marker shape should be rendered filled (from its legend definition).
// Compare normalised so legacy names ("flag") match Lucide ids ("l:Flag").
const SHAPE_ALIAS = { diamond: 'Diamond', circle: 'Circle', cone: 'Triangle', triangleDown: 'Triangle', flag: 'Flag', square: 'Square', star: 'Star', hexagon: 'Hexagon', pentagon: 'Pentagon' }
function normShape(s) {
  if (!s) return ''
  return s.startsWith('l:') ? s.slice(2) : (SHAPE_ALIAS[s] || s)
}
function markerFill(shape) {
  const n = normShape(shape)
  const m = settings.markers.find(x => normShape(x.shape) === n)
  return !!(m && m.fill)
}
function isBar(m) {
  return m.kind === 'event' && m.startDate && m.endDate && m.endDate > m.startDate
}
function barInfo(m) {
  const s = ymOf(m.startDate)
  const e = ymOf(m.endDate)
  if (s.y > store.year || e.y < store.year) return null
  return {
    startX: dateX(m.startDate),
    endX: dateX(m.endDate),
    continuesLeft: s.y < store.year,
    continuesRight: e.y > store.year,
  }
}

// ── Per-row layout: gather items, estimate extents, pack into lanes ───────────
function rowItems(swId, subId) {
  const items = []
  for (const m of store.milestones) {
    if (m.swimlaneId !== swId || (m.subLaneId ?? null) !== subId) continue
    if (isBar(m)) {
      const info = barInfo(m)
      if (!info) continue
      const width = Math.max(info.endX - info.startX, 16)
      const labelW = estTextW(m.title)
      const hasMarker = !!(m.marker && m.marker !== 'bar')
      // Account for the icon AND the bar padding it needs around it.
      const iconW = hasMarker ? (settings.items.markerSize + settings.items.iconGap + settings.items.padding + 8) : 0
      // If the icon + title don't fit, the whole icon+title unit moves to the
      // right of the bar (tight together); otherwise it sits inside the bar.
      const labelOutside = iconW + labelW + settings.items.labelBuffer > width
      const x1 = labelOutside ? info.startX + width + 10 + iconW + labelW : info.startX + width
      items.push({
        key: m.id, m, type: 'bar', x: info.startX, width, labelOutside,
        x0: info.startX, x1,
        continuesLeft: info.continuesLeft, continuesRight: info.continuesRight,
      })
    } else {
      const ad = anchorDate(m)
      if (Number(ad.slice(0, 4)) !== store.year) continue
      const x = dateX(ad)
      const labelW = estTextW(m.title)
      const pad = settings.items.padding
      items.push({
        key: m.id, m, type: 'point', x,
        x0: x - 6 - pad,
        x1: x + 16 + labelW + pad,
      })
    }
  }

  const d = baselineDiff.value
  if (d.active) {
    for (const gh of d.ghosts) {
      const ghSw = gh.swimlaneId, ghSub = gh.subLaneId ?? null
      if (ghSw !== swId || ghSub !== subId) continue
      const ad = gh.when || `${gh.year}-${String(gh.month).padStart(2, '0')}-01`
      if (Number(ad.slice(0, 4)) !== store.year) continue
      const x = dateX(ad)
      const labelW = estTextW(gh.title)
      const pad = settings.items.padding
      items.push({ key: 'g-' + gh.id, gh, type: 'ghost', ghostType: gh.ghostType, x, x0: x - 6 - pad, x1: x + 16 + labelW + pad })
    }
  }

  // Greedy lane packing: each item drops to the first lane whose last item ended
  // (plus a small gap) before this one starts.
  items.sort((a, b) => a.x0 - b.x0)
  const laneRight = []
  const GAP = 6
  for (const it of items) {
    let lane = laneRight.findIndex(r => it.x0 >= r + GAP)
    if (lane === -1) { lane = laneRight.length; laneRight.push(0) }
    laneRight[lane] = it.x1
    it.lane = lane
  }
  return { items, laneCount: Math.max(1, laneRight.length) }
}

const grid = computed(() =>
  tableRows.value.map(row => {
    const { items, laneCount } = rowItems(row.swimlane.id, row.subLane?.id ?? null)
    // Tight rows: only as tall as needed; event rows a touch taller than milestone-only rows.
    const isEvent = items.some(i => i.type === 'bar')
    const laneH = isEvent ? eventLaneH.value : msLaneH.value
    const vOffset = Math.round((laneH - (isEvent ? eventPillH.value : msPillH.value)) / 2)
    return { row, items, laneCount, laneH, vOffset, trackHeight: laneCount * laneH }
  })
)

function baselineClass(m) {
  const s = baselineDiff.value.status[m.id]
  return s === 'added' ? 'bl-added' : s === 'moved' ? 'bl-moved' : ''
}

const currentMonthIndex = computed(() => {
  const now = new Date()
  return now.getFullYear() === store.year ? now.getMonth() : -1
})
function isCurrentMonth(month) {
  return currentMonthIndex.value === month - 1
}
const todayX = computed(() => {
  if (currentMonthIndex.value < 0) return -1
  const now = new Date()
  return (now.getMonth() + (now.getDate() - 1) / daysInMonth(store.year, now.getMonth() + 1)) * COL_W.value
})
const monthHlColor = computed(() => hexAlpha(settings.monthHighlight.color, settings.monthHighlight.opacity))
const dayLineColor = computed(() => hexAlpha(settings.dayLine.color, settings.dayLine.opacity))

// ISO week numbers across the viewed year, thinned out when columns get narrow.
const weekTicks = computed(() => {
  if (!settings.weekNumbers.enabled) return []
  const year = store.year
  const weekPx = MONTHS_W.value / 52.1429
  const step = Math.max(1, Math.ceil(22 / weekPx))
  const out = []
  const offset = (new Date(year, 0, 1).getDay() + 6) % 7 // days from Monday
  let monday = new Date(year, 0, 1 - offset)              // Monday of the week containing Jan 1
  const lastDay = new Date(year, 11, 31)
  let i = 0
  while (monday <= lastDay) {
    if (i % step === 0) {
      const next = new Date(monday.getFullYear(), monday.getMonth(), monday.getDate() + 7)
      const xMid = (dateXOf(monday) + dateXOf(next)) / 2
      if (xMid >= 4 && xMid <= MONTHS_W.value - 2) out.push({ n: isoWeek(monday), x: xMid })
    }
    monday = new Date(monday.getFullYear(), monday.getMonth(), monday.getDate() + 7)
    i++
  }
  return out
})

// Gridlines are rendered as real, pixel-snapped <div> rectangles (not gradient
// backgrounds) — wide gradient backgrounds get rasterised into a scaled texture
// by Chromium/Edge and turn fuzzy. Solid divs stay crisp at any zoom / DPR.
function lineDivs(xs, color, w) {
  const width = Math.max(1, Math.round(w))
  return xs.map((x, i) => ({ key: i, left: Math.round(x - width / 2), w: width, color }))
}

// Month separator lines — interior boundaries only (the December edge is drawn
// once by the always-present closing line below, so it never doubles up).
const monthLineDivs = computed(() => {
  const s = settings.monthLines
  if (!s || !s.enabled) return []
  const xs = []
  for (let i = 1; i <= 11; i++) xs.push(i * COL_W.value)
  return lineDivs(xs, hexAlpha(s.color, s.opacity), s.width)
})

// A clean closing line at the right edge of December — always present so the
// grid is properly closed even when month lines are turned off.
const closeLine = computed(() => {
  const s = settings.monthLines
  const on = !!(s && s.enabled)
  const w = Math.max(1, Math.round(on ? s.width : 1))
  const color = on ? hexAlpha(s.color, s.opacity) : '#E5E5EA'
  return { left: Math.round(MONTHS_W.value - w / 2), w, color }
})

// Fine vertical week gridlines, one per Monday so they align with the CW numbers.
const weekLineDivs = computed(() => {
  const s = settings.weekLines
  if (!s || !s.enabled) return []
  const year = store.year
  const offset = (new Date(year, 0, 1).getDay() + 6) % 7
  let monday = new Date(year, 0, 1 - offset)
  const last = new Date(year, 11, 31)
  const xs = []
  while (monday <= last) {
    const x = dateXOf(monday)
    if (x > 0.5 && x < MONTHS_W.value - 0.5) xs.push(x)
    monday = new Date(monday.getFullYear(), monday.getMonth(), monday.getDate() + 7)
  }
  return lineDivs(xs, hexAlpha(s.color, s.opacity), s.width)
})

function onTrackClick(row, e) {
  const rect = e.currentTarget.getBoundingClientRect()
  const x = e.clientX - rect.left
  if (x > MONTHS_W.value) return   // ignore clicks in the empty right gutter
  if (settings.weekNumbers.enabled) {
    // Week-specific: prefill the new item with the clicked week's Monday.
    let monday = mondayOf(xToDate(x))
    if (monday.getFullYear() < store.year) monday = new Date(store.year, 0, 1)
    emit('add-milestone', { swimlane: row.swimlane, subLane: row.subLane, month: monday.getMonth() + 1, date: fmtDate(monday) })
  } else {
    const month = Math.min(12, Math.max(1, Math.floor(x / COL_W.value) + 1))
    emit('add-milestone', { swimlane: row.swimlane, subLane: row.subLane, month })
  }
}

// "+" add-hint follows the cursor — snapped to the hovered week (CW on) or month.
const addHint = reactive({ key: null, x: 0 })
function onTrackMove(key, e) {
  if (props.readOnly) return
  const rect = e.currentTarget.getBoundingClientRect()
  const x = e.clientX - rect.left
  if (x > MONTHS_W.value) { addHint.key = null; return }   // empty right gutter
  addHint.key = key
  if (settings.weekNumbers.enabled) {
    const monday = mondayOf(xToDate(x))
    const next = new Date(monday.getFullYear(), monday.getMonth(), monday.getDate() + 7)
    const c = (dateXOf(monday) + dateXOf(next)) / 2
    addHint.x = Math.max(6, Math.min(MONTHS_W.value - 6, c))
  } else {
    const mi = Math.min(11, Math.max(0, Math.floor(x / COL_W.value)))
    addHint.x = (mi + 0.5) * COL_W.value
  }
}

// ── Bar styling + drag (day-precise) ──────────────────────────────────────────
function hexAlpha(hex, alpha) {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return `rgba(${r},${g},${b},${alpha})`
}
function barStyle(it, color, laneH, vOffset = ITEM_AIR) {
  return {
    left: it.x + 'px',
    top: (it.lane * laneH + vOffset) + 'px',
    width: it.width + 'px',
    background: hexAlpha(color, settings.items.eventOpacity),
    borderColor: hexAlpha(color, Math.min(1, settings.items.eventOpacity * 2.5)),
    color,
  }
}

const dragState = ref(null)      // { id, mode, startClientX, dx }
const suppressClick = ref(false)

function addDaysToDate(dateStr, days) {
  const [y, m, d] = dateStr.split('-').map(Number)
  const dt = new Date(y, m - 1, d + days)
  const mm = String(dt.getMonth() + 1).padStart(2, '0')
  const dd = String(dt.getDate()).padStart(2, '0')
  return `${dt.getFullYear()}-${mm}-${dd}`
}
function startDrag(e, it, mode) {
  if (props.readOnly || it.m.sourceSystem) return
  dragState.value = { id: it.m.id, mode, startClientX: e.clientX, dx: 0, cx: e.clientX, cy: e.clientY }
  document.body.style.userSelect = 'none'
  window.addEventListener('pointermove', onDragMove)
  window.addEventListener('pointerup', onDragEnd, { once: true })
}
function onDragMove(e) {
  const ds = dragState.value
  if (!ds) return
  ds.dx = e.clientX - ds.startClientX // layout px (no CSS zoom; months scale via COL_W)
  ds.cx = e.clientX
  ds.cy = e.clientY
}
function onDragEnd() {
  window.removeEventListener('pointermove', onDragMove)
  document.body.style.userSelect = ''
  const ds = dragState.value
  dragState.value = null
  if (!ds) return
  const deltaDays = Math.round(ds.dx / (COL_W.value / DAYS_PER_COL))
  if (deltaDays !== 0) {
    const m = store.milestones.find(x => x.id === ds.id)
    if (m && m.startDate && m.endDate) {
      let s = m.startDate
      let en = m.endDate
      if (ds.mode === 'move') {
        s = addDaysToDate(s, deltaDays)
        en = addDaysToDate(en, deltaDays)
      } else if (ds.mode === 'resize-r') {
        en = addDaysToDate(en, deltaDays)
        if (en < s) en = s
      } else if (ds.mode === 'resize-l') {
        s = addDaysToDate(s, deltaDays)
        if (s > en) s = en
      }
      const [yy, mm] = s.split('-').map(Number)
      updateMilestone(ds.id, { startDate: s, endDate: en, when: s, year: yy, month: mm })
    }
    suppressClick.value = true
    setTimeout(() => { suppressClick.value = false }, 60)
  }
}
function barStyleFull(it, color, laneH, vOffset) {
  const base = barStyle(it, color, laneH, vOffset)
  const ds = dragState.value
  if (ds && ds.id === it.m.id && ds.dx !== 0) {
    if (ds.mode === 'move') {
      base.transform = `translateX(${ds.dx}px)`
    } else if (ds.mode === 'resize-r') {
      base.width = Math.max(8, it.width + ds.dx) + 'px'
    } else if (ds.mode === 'resize-l') {
      base.transform = `translateX(${ds.dx}px)`
      base.width = Math.max(8, it.width - ds.dx) + 'px'
    }
    base.opacity = '0.9'
    base.zIndex = 6
  }
  return base
}

// Live drag/resize preview shown next to the cursor while dragging an event bar.
const dragTip = computed(() => {
  const ds = dragState.value
  if (!ds) return null
  const m = store.milestones.find(x => x.id === ds.id)
  if (!m || !m.startDate || !m.endDate) return null
  const deltaDays = Math.round(ds.dx / (COL_W.value / DAYS_PER_COL))
  let s = m.startDate, en = m.endDate
  if (ds.mode === 'move') { s = addDaysToDate(s, deltaDays); en = addDaysToDate(en, deltaDays) }
  else if (ds.mode === 'resize-r') { en = addDaysToDate(en, deltaDays); if (en < s) en = s }
  else if (ds.mode === 'resize-l') { s = addDaysToDate(s, deltaDays); if (s > en) s = en }
  return { mode: ds.mode, s, en, days: durationDays(s, en), deltaDays, x: ds.cx, y: ds.cy }
})
const dragTipStyle = computed(() => {
  if (!dragTip.value) return {}
  const margin = 12, w = 170
  const left = Math.min(dragTip.value.x + 16, window.innerWidth - w - margin)
  return { position: 'fixed', left: Math.max(margin, left) + 'px', top: (dragTip.value.y + 18) + 'px', zIndex: 10000 }
})

// ── Dependency highlighting ───────────────────────────────────────────────────
const hoveredMs = ref(null)
const selectedMs = ref(null)
const activeMs = computed(() => hoveredMs.value ?? selectedMs.value)
const relatedIds = computed(() => activeMs.value ? getLinkedIds(activeMs.value.id) : new Set())
const linkedMilestones = computed(() => {
  if (!selectedMs.value) return []
  const deps = dependsOnIds(selectedMs.value.id)
  return store.milestones.filter(m => deps.has(m.id))
})
const parentMilestones = computed(() => {
  if (!selectedMs.value) return []
  const parents = dependentIds(selectedMs.value.id)
  return store.milestones.filter(m => parents.has(m.id))
})
function swimlaneColor(swimlaneId) {
  return store.swimlanes.find(s => s.id === swimlaneId)?.color ?? '#888'
}
function formatDate(dateStr) {
  if (!dateStr) return ''
  const [y, m, day] = dateStr.split('-').map(Number)
  return new Date(y, m - 1, day).toLocaleDateString('en-US', { day: 'numeric', month: 'long', year: 'numeric' })
}
function formatShort(dateStr) {
  if (!dateStr) return ''
  const [y, m, day] = dateStr.split('-').map(Number)
  return new Date(y, m - 1, day).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}
function durationDays(s, en) {
  const [y1, m1, d1] = s.split('-').map(Number)
  const [y2, m2, d2] = en.split('-').map(Number)
  return Math.round((new Date(y2, m2 - 1, d2) - new Date(y1, m1 - 1, d1)) / 86400000) + 1
}
// Members of the group currently hovered in the left legend.
const groupMemberIds = computed(() => {
  if (!ui.hoverGroupId) return null
  const g = groups.list.find(x => x.id === ui.hoverGroupId)
  return new Set(g ? (g.itemIds || []) : [])
})

function chipState(m) {
  // Hovering a group in the legend highlights its members and dims the rest.
  if (groupMemberIds.value) {
    return groupMemberIds.value.has(m.id) ? 'chip-related' : 'chip-dimmed'
  }
  if (!activeMs.value) return ''
  if (m.id === activeMs.value.id) return 'chip-active'
  if (relatedIds.value.has(m.id)) return 'chip-related'
  return 'chip-dimmed'
}

// ── Tooltip ───────────────────────────────────────────────────────────────────
const tooltip = reactive({ visible: false, ms: null, x: 0, chipTop: 0, chipBottom: 0, color: '' })
const tooltipStyle = computed(() => {
  const margin = 12
  const tipW = 296
  const estimatedHeight = 340
  const left = Math.min(tooltip.x, window.innerWidth - tipW - margin)
  const spaceBelow = window.innerHeight - tooltip.chipBottom
  const openUp = spaceBelow < estimatedHeight + margin
  return {
    position: 'fixed',
    ...(openUp
      ? { bottom: `${window.innerHeight - tooltip.chipTop + 8}px`, top: 'auto' }
      : { top: `${tooltip.chipBottom + 8}px`, bottom: 'auto' }),
    left: `${Math.max(margin, left)}px`,
    width: `${tipW}px`,
    zIndex: 9999,
  }
})
function onChipClick(e, m, color) {
  if (suppressClick.value) return
  if (selectedMs.value?.id === m.id) {
    selectedMs.value = null
    tooltip.visible = false
    return
  }
  selectedMs.value = m
  const rect = e.currentTarget.getBoundingClientRect()
  tooltip.x = rect.left
  tooltip.chipTop = rect.top
  tooltip.chipBottom = rect.bottom
  tooltip.ms = m
  tooltip.color = color
  tooltip.visible = true
}

// Click a Prerequisite / Required-by / Late row → swap the popup to that item,
// anchored to the real item on the timeline (scrolled into view) when possible.
function selectFromTooltip(m, e) {
  const fallback = e.currentTarget.getBoundingClientRect()
  selectedMs.value = m
  tooltip.ms = m
  tooltip.color = swimlaneColor(m.swimlaneId)
  tooltip.visible = true
  let rect = fallback
  const el = wrapEl.value && wrapEl.value.querySelector(`[data-item-id="${m.id}"]`)
  if (el) {
    el.scrollIntoView({ block: 'nearest', inline: 'center' })
    rect = el.getBoundingClientRect()
  }
  tooltip.x = rect.left
  tooltip.chipTop = rect.top
  tooltip.chipBottom = rect.bottom
}

// Open an item's popup by id (e.g. from the header risk list), switching the
// year and scrolling it into view if needed.
function focusItem(id) {
  const m = store.milestones.find(x => x.id === id)
  if (!m) return
  if (m.year && m.year !== store.year) store.year = m.year
  const place = () => {
    selectedMs.value = m
    tooltip.ms = m
    tooltip.color = swimlaneColor(m.swimlaneId)
    tooltip.visible = true
    const el = wrapEl.value && wrapEl.value.querySelector(`[data-item-id="${id}"]`)
    if (el) {
      el.scrollIntoView({ block: 'nearest', inline: 'center' })
      const r = el.getBoundingClientRect()
      tooltip.x = r.left; tooltip.chipTop = r.top; tooltip.chipBottom = r.bottom
      return true
    }
    return false
  }
  nextTick(() => {
    if (place()) return
    // item may have only just re-rendered (e.g. after a year switch) — retry once
    requestAnimationFrame(() => {
      if (!place()) { tooltip.x = window.innerWidth / 2 - 148; tooltip.chipTop = 110; tooltip.chipBottom = 130 }
    })
  })
}
watch(() => ui.focusItemId, (id) => {
  if (!id) return
  focusItem(id)
  ui.focusItemId = null
})
function closeTooltip() {
  tooltip.visible = false
  selectedMs.value = null
}
function onDocumentClick() {
  if (tooltip.visible) closeTooltip()
}
function onKeyDown(e) {
  if (e.key === 'Escape') closeTooltip()
}
onMounted(() => {
  document.addEventListener('click', onDocumentClick)
  document.addEventListener('keydown', onKeyDown)
  if (wrapEl.value) {
    const measure = () => {
      wrapW.value = wrapEl.value.clientWidth
      if (headRowEl.value) monthHeadH.value = headRowEl.value.offsetHeight
    }
    measure()
    resizeObs = new ResizeObserver(measure)
    resizeObs.observe(wrapEl.value)
    if (headRowEl.value) resizeObs.observe(headRowEl.value)
  }
})
onUnmounted(() => {
  document.removeEventListener('click', onDocumentClick)
  document.removeEventListener('keydown', onKeyDown)
  if (resizeObs) resizeObs.disconnect()
})
</script>

<style scoped>
.table-wrap {
  overflow: auto;
  height: calc(100vh - 68px);
  background: var(--clr-surface);
}

.ms-table {
  table-layout: fixed;          /* fixed month widths — content never resizes columns */
  border-collapse: separate;
  border-spacing: 0;
  background: var(--clr-surface);
}

/* --- Header row --- */
thead tr { background: var(--clr-header); }
thead th {
  position: sticky;
  top: 0;
  z-index: 20;
  padding: 14px 12px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.5px;
  text-transform: uppercase;
  color: rgba(255,255,255,0.55);
  white-space: nowrap;
  border-bottom: 1px solid rgba(255,255,255,0.06);
  background: var(--clr-header);
  user-select: none;
}
.th-area {
  position: sticky; left: 0; z-index: 30;
  width: 168px; min-width: 168px; text-align: left;
}
.th-sub {
  position: sticky; left: 168px; z-index: 30;
  width: 148px; min-width: 148px; text-align: left;
  border-left: 1px solid rgba(255,255,255,0.06);
}
.th-month { text-align: center; }
.th-current { color: rgba(0,180,255,0.85) !important; }

/* --- Calendar-week header row --- */
.head-weeks th {
  position: sticky;             /* pin the whole week row below the month row */
  top: var(--wk-top, 44px);
  height: 22px;
  padding: 0 12px;
  z-index: 19;
}
.head-weeks .th-area, .head-weeks .th-sub { z-index: 29; }
.wk-corner { text-align: right; }
.wk-kw { font-size: 10px; font-weight: 700; color: rgba(255,255,255,0.4); letter-spacing: 0.5px; }
/* sticky (from .head-weeks th) already establishes the containing block for the absolute numbers */
.wk-cell { overflow: hidden; padding: 0 !important; }
.wk-num {
  position: absolute; top: 50%;
  transform: translate(-50%, -50%);
  font-size: 10px; font-weight: 600;
  color: rgba(255,255,255,0.42);
  pointer-events: none; white-space: nowrap;
}

/* --- Sticky lane columns --- */
.td-area {
  position: sticky; left: 0; z-index: 10;
  background: var(--clr-surface);
  border-bottom: 1px solid var(--clr-border-light);
  border-right: 3px solid var(--lane);
  padding: 4px 14px;
  vertical-align: middle;
  user-select: none; -webkit-user-select: none;
}
.area-label { display: flex; align-items: center; gap: 8px; }
.area-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.area-name { font-size: 13px; font-weight: 600; color: var(--clr-text); letter-spacing: -0.1px; }

.td-sub {
  position: sticky; left: 168px; z-index: 10;
  background: var(--clr-surface-2);
  border-bottom: 1px solid var(--clr-border-light);
  border-right: 1px solid var(--clr-border-light);
  padding: 4px 14px;
  vertical-align: middle;
  white-space: nowrap;
  user-select: none; -webkit-user-select: none;
}
.sub-name { font-size: 12px; color: var(--clr-text-2); font-weight: 500; }

/* --- Track (the 12-month area for one row) --- */
.track {
  position: relative;
  vertical-align: top;
  overflow: hidden;             /* clip lines/labels to the reserved right gutter */
  border-bottom: 1px solid var(--clr-border-light);
  cursor: pointer;
}

/* Crisp vertical gridlines (solid rectangles, pixel-snapped). */
.grid-line {
  position: absolute;
  top: 0; bottom: 0;
  pointer-events: none;
}
.is-readonly .track { cursor: default; }

.month-now {
  position: absolute; top: 0; bottom: 0;
  pointer-events: none;
}
.day-line {
  position: absolute; top: 0; bottom: 0;
  transform: translateX(-50%);
  pointer-events: none;
  z-index: 1;
}

.track-add-hint {
  position: absolute; top: 50%;
  transform: translate(-50%, -50%);
  font-size: 18px; font-weight: 400; color: var(--clr-text-3);
  line-height: 1; pointer-events: none;
  z-index: 1;                 /* stays behind markers & event bars */
}
.is-readonly .track-add-hint { display: none; }

/* Breathing room at the bottom so the fixed legend never covers the last rows. */
.track-spacer { height: 76px; }

/* --- Milestone (marker + floating label) --- */
.mk-item {
  position: absolute;
  display: inline-flex;
  align-items: center;
  gap: var(--it-gap, 4px);
  padding: var(--it-pad, 4px) calc(var(--it-pad, 4px) + 3px);
  font-size: var(--it-font, 12px);
  line-height: 1;
  font-weight: var(--it-weight, 500);
  white-space: nowrap;
  cursor: pointer;
  user-select: none;
  border-radius: var(--it-radius, 6px);
  z-index: 2;                 /* keep markers/labels above the "+" add hint */
  transition: filter 0.18s ease, opacity 0.18s ease, box-shadow 0.18s ease;
}
.mk-item:hover { filter: brightness(0.9); }
.mk-icon { flex-shrink: 0; }
.mk-label, .bar-title {
  text-box-trim: trim-both;
  text-box-edge: cap alphabetic;
}
.mk-label { position: relative; top: var(--it-label-y, 1px); color: inherit; }
.chip-lock { margin-right: 2px; font-size: 9px; opacity: 0.85; }

/* --- Event bars --- */
.event-bar {
  position: absolute;
  box-sizing: border-box;
  display: inline-flex;
  align-items: center;
  gap: var(--it-gap, 5px);
  padding: var(--it-pad, 4px) calc(var(--it-pad, 4px) + 8px);
  border: var(--it-ring, 2px) solid;
  border-radius: var(--it-radius, 6px);
  font-size: var(--it-font, 12px);
  line-height: 1;
  font-weight: var(--it-weight, 500);
  /* keep the pill height even when the title sits outside (out of flow) */
  min-height: calc(var(--it-font, 12px) + var(--it-pad, 4px) * 2 + var(--it-ring, 2px) * 2);
  white-space: nowrap;
  overflow: visible;            /* short bars show their title to the right instead */
  cursor: pointer;
  z-index: 3;
  user-select: none;
  transition: filter 0.18s ease, box-shadow 0.18s ease, opacity 0.18s ease;
}
.event-bar:hover { filter: brightness(0.97); }
.bar-title { display: inline-flex; align-items: center; gap: var(--it-gap, 4px); overflow: visible; flex-shrink: 0; position: relative; top: var(--it-label-y, 1px); }
/* Short event: render the title just to the right of the bar (out of flow). */
.bar-title.bar-title-out {
  position: absolute;
  left: 100%;
  top: calc(50% + var(--it-label-y, 1px));
  transform: translateY(-50%);
  margin-left: 7px;
}
.bar-arrow { font-size: 9px; opacity: 0.7; flex-shrink: 0; }
.bar-marker { flex-shrink: 0; }
.event-bar.draggable { cursor: grab; }
.event-bar.draggable:active { cursor: grabbing; }
.bar-handle { position: absolute; top: 0; height: 100%; width: 9px; cursor: ew-resize; z-index: 6; }
.bar-handle-l { left: 0; }
.bar-handle-r { right: 0; }

/* Keep items absolutely positioned even when highlighted (no row growth). */
.mk-item.chip-active, .mk-item.chip-related, .mk-item.chip-dimmed,
.event-bar.chip-active, .event-bar.chip-related, .event-bar.chip-dimmed { position: absolute; }
.event-bar.chip-active { box-shadow: none !important; filter: brightness(0.93) !important; z-index: 4; }
.event-bar.chip-related { box-shadow: none; filter: brightness(0.98); }

/* --- Baseline diff overlay (P2) --- */
.bl-added { box-shadow: 0 0 0 2px #30D158; border-radius: 6px; }
.bl-moved { box-shadow: 0 0 0 2px #FF9F0A; border-radius: 6px; }
.ghost { opacity: 0.6; }
.ghost .mk-label { color: var(--clr-text-3); border-bottom: 1px dashed var(--clr-border); }
.ghost-removed .mk-label { text-decoration: line-through; color: #c0392b; }

/* --- Dependency states --- */
.chip-active {
  box-shadow: 0 0 0 var(--it-ring, 2px) currentColor, 0 2px 10px rgba(0,0,0,0.14);
  filter: brightness(1.05);
  border-radius: var(--it-radius, 6px);
  z-index: 5;
}
.chip-related {
  box-shadow: 0 0 0 var(--it-ring, 2px) currentColor;
  filter: brightness(1.03);
  opacity: 1 !important;
  border-radius: var(--it-radius, 6px);
}
.chip-dimmed { opacity: 0.32 !important; filter: grayscale(0.3) !important; }

/* --- Item border mode: always / on-hover / off (width = --it-ring) --- */
/* Milestones (ring drawn in the item colour) */
.bm-always .mk-item { box-shadow: 0 0 0 var(--it-ring, 2px) currentColor; }
.bm-off .mk-item,
.bm-off .mk-item.chip-active,
.bm-off .mk-item.chip-related { box-shadow: none !important; }
/* Events (bar outline) */
.bm-hover .event-bar,
.bm-off .event-bar { border-color: transparent !important; }
.bm-hover .event-bar:hover,
.bm-hover .event-bar.chip-active,
.bm-hover .event-bar.chip-related { border-color: currentColor !important; }

/* --- Empty state --- */
.empty-state { text-align: center; padding: 80px 20px; color: var(--clr-text-3); font-size: 14px; }

/* --- Legend --- */
.legend {
  position: fixed;
  bottom: 14px; right: 18px;
  display: flex; gap: 14px; align-items: center;
  min-height: 40px; box-sizing: border-box;
  padding: 0 16px;
  background: var(--clr-glass);
  backdrop-filter: blur(10px); -webkit-backdrop-filter: blur(10px);
  border: 1px solid var(--clr-border);
  border-radius: 100px;
  box-shadow: var(--sh-lg), 0 0 0 1px rgba(0,0,0,0.03);
  font-size: 11px; color: var(--clr-text-2);
  z-index: 50;
}
.legend-item { display: inline-flex; align-items: center; gap: 5px; }
.legend-bar { width: 18px; height: 10px; border-radius: 3px; background: rgba(120,120,128,0.3); border: 1px solid rgba(120,120,128,0.55); }

/* --- Tooltip --- */
.ms-tooltip {
  background: var(--clr-glass);
  backdrop-filter: blur(28px) saturate(1.8);
  -webkit-backdrop-filter: blur(28px) saturate(1.8);
  border: 1px solid var(--clr-border-light);
  border-radius: var(--r-lg);
  box-shadow: 0 8px 32px rgba(0,0,0,0.18);
  overflow: hidden;
}
.tooltip-header {
  display: flex; align-items: center; gap: 8px;
  padding: 12px 14px 10px;
  border-bottom: 1px solid var(--clr-border-light);
}
.tooltip-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.tooltip-title { font-size: 13px; font-weight: 600; color: var(--clr-text); letter-spacing: -0.1px; }
.tooltip-fields { padding: 10px 14px 8px; display: flex; flex-direction: column; gap: 7px; }
.tooltip-field { display: grid; grid-template-columns: 38px 1fr; gap: 8px; align-items: baseline; }
.tf-label { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-text-3); }
.tf-val { font-size: 12.5px; color: var(--clr-text); line-height: 1.45; }
.tooltip-links { padding: 8px 14px 10px; border-top: 1px solid var(--clr-border-light); }
.tl-label { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-text-3); display: block; margin-bottom: 6px; }
.tl-items { display: flex; flex-direction: column; gap: 5px; }
.tl-item { display: flex; align-items: center; gap: 6px; min-width: 0; cursor: pointer; padding: 2px 4px; margin: 0 -4px; border-radius: 5px; transition: background 0.1s; }
.tl-item:hover { background: var(--clr-surface-2); }
.tl-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.tl-title { font-size: 12.5px; color: var(--clr-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.tl-date { font-size: 11px; color: var(--clr-text-3); white-space: nowrap; margin-left: auto; padding-left: 8px; flex-shrink: 0; }
.tl-more { font-size: 11px; color: var(--clr-text-3); padding-left: 12px; }

/* Dependency-risk badge + tooltip section */
.risk-badge { flex-shrink: 0; margin-left: 3px; }
.tooltip-risk { padding: 8px 14px 10px; border-top: 1px solid var(--clr-border-light); }
.tr-label { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-danger); display: block; margin-bottom: 6px; }
.tr-items { display: flex; flex-direction: column; gap: 4px; }
.tr-item { font-size: 12.5px; color: var(--clr-text); }

/* --- Drag/resize live preview --- */
.drag-tip {
  background: rgba(28,28,30,0.92);
  color: #fff;
  border-radius: 8px;
  padding: 7px 10px;
  pointer-events: none;
  box-shadow: 0 6px 20px rgba(0,0,0,0.28);
  backdrop-filter: blur(8px); -webkit-backdrop-filter: blur(8px);
  white-space: nowrap;
}
.dt-range { font-size: 12.5px; font-weight: 600; letter-spacing: -0.1px; }
.dt-meta { display: flex; gap: 8px; align-items: center; margin-top: 3px; }
.dt-days { font-size: 11px; color: rgba(255,255,255,0.7); }
.dt-delta { font-size: 11px; font-weight: 700; color: #6ee7a0; background: rgba(48,209,88,0.18); padding: 1px 6px; border-radius: 100px; }
.dt-delta.neg { color: #fda4a4; background: rgba(255,69,58,0.18); }
</style>
