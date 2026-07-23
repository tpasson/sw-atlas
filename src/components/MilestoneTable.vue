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
          <th class="th-area" :style="{ width: AREA_W + 'px', minWidth: AREA_W + 'px' }">Area</th>
          <th class="th-sub" :style="{ left: AREA_W + 'px', width: SUB_W + 'px', minWidth: SUB_W + 'px' }">Sub-Area</th>
          <th
            v-for="(m, i) in columns"
            :key="i"
            class="th-month"
            :class="{ 'th-current': isCurrentCol(i) }"
            :style="{ width: COL_W + 'px' }"
          >
            {{ m }}
          </th>
          <th class="th-gutter" :style="{ width: gutterW + 'px' }"></th>
        </tr>
        <tr v-if="settings.weekNumbers.enabled && granularity !== 'month'" class="head-weeks" :style="{ '--wk-top': monthHeadH + 'px' }">
          <th class="th-area wk-corner" :style="{ width: AREA_W + 'px', minWidth: AREA_W + 'px' }"></th>
          <th class="th-sub wk-corner" :style="{ left: AREA_W + 'px', width: SUB_W + 'px', minWidth: SUB_W + 'px' }"><span class="wk-kw">CW</span></th>
          <th class="wk-cell" :colspan="trackColspan">
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
                <span class="area-name" :title="g.row.swimlane.name">{{ g.row.swimlane.name }}</span>
              </div>
            </td>

            <td class="td-sub" :style="{ left: AREA_W + 'px' }">
              <span class="sub-name" :title="g.row.subLane?.name">{{ g.row.subLane?.name ?? '' }}</span>
            </td>

            <!-- One track per row spanning all 12 fixed-width months. Items are
                 positioned by date and stacked into lanes to avoid collisions. -->
            <td
              class="track"
              :colspan="trackColspan"
              :style="{ height: g.trackHeight + 'px', '--col': COL_W + 'px' }"
              @click="props.readOnly ? null : onTrackClick(g.row, $event)"
              @mousemove="onTrackMove(rowKey(g.row), $event)"
              @mouseleave="addHint.key = null"
            >
              <template v-if="currentColIndex >= 0">
                <div
                  v-if="settings.monthHighlight.enabled"
                  class="month-now"
                  :style="{ left: currentColIndex * COL_W + 'px', width: COL_W + 'px', background: monthHlColor }"
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

              <template v-if="currentColIndex >= 0">
                <div
                  v-if="settings.dayLine.enabled"
                  class="day-line"
                  :style="{ left: todayX + 'px', width: settings.dayLine.width + 'px', background: dayLineColor }"
                ></div>
              </template>

              <template v-for="it in g.items" :key="it.key">
                <!-- Rail dot: compact density mode collapses markers to a tick row -->
                <div
                  v-if="it.type === 'point' && it.rail"
                  class="rail-dot"
                  :data-item-id="it.m.id"
                  :class="chipState(it.m)"
                  :style="{ left: (it.x - 5) + 'px', top: (it.lane * g.laneH + (g.laneH - 10) / 2) + 'px', background: itemColor(it.m) }"
                  :title="it.m.title"
                  @mouseenter="hoveredMs = it.m"
                  @mouseleave="hoveredMs = null"
                  @click.stop="onChipClick($event, it.m, g.row.swimlane.color)"
                  @dblclick.stop="onEdit(it.m)"
                ></div>

                <!-- Milestone: marker is the anchor at its day, label flows right -->
                <div
                  v-else-if="it.type === 'point'"
                  class="mk-item"
                  :data-item-id="it.m.id"
                  :class="chipState(it.m)"
                  :style="{ left: (it.x - 9 - settings.items.padding - dotExtra(it.m)) + 'px', top: (it.lane * g.laneH + g.vOffset) + 'px', color: 'var(--clr-text)', '--it-status': itemColor(it.m) }"
                  @mouseenter="hoveredMs = it.m"
                  @mouseleave="hoveredMs = null"
                  @click.stop="onChipClick($event, it.m, g.row.swimlane.color)"
                  @dblclick.stop="onEdit(it.m)"
                >
                  <MarkerIcon :shape="markerOf(it.m)" :color="itemColor(it.m)" :size="settings.items.markerSize" :stroke-width="settings.items.markerStroke" :fill="markerFillFor(it.m)" class="mk-icon" />
                  <span v-if="it.m.sourceSystem" class="chip-lock" title="Synced — read-only"><Lock :size="10" :stroke-width="2.5" /></span><span class="mk-label">{{ it.m.title }}</span><MaturityGlyph v-if="it.m.maturity" :level="it.m.maturity" variant="grid" :size="matSize" :color="'var(--clr-text-2)'" :title="maturityTitle(it.m.maturity)" class="mk-mat" /><AlertTriangle v-if="riskIds.has(it.m.id)" class="risk-badge" :size="settings.items.markerSize" :stroke-width="settings.items.markerStroke" color="#FF3B30" /><Clock v-if="lateIds.has(it.m.id)" class="late-badge" title="Overdue" :size="settings.items.markerSize" :stroke-width="settings.items.markerStroke" color="#FF3B30" /><AlertTriangle v-if="resourceConflictIds.has(it.m.id)" class="conflict-badge" :title="conflictTitle(it.m.id)" :size="settings.items.markerSize" :stroke-width="settings.items.markerStroke" color="#FF3B30" />
                </div>

                <!-- Cluster: collapsed overflow markers; click to expand the list -->
                <div
                  v-else-if="it.type === 'cluster'"
                  class="mk-cluster"
                  :style="{ left: (it.x - 9 - settings.items.padding) + 'px', top: (it.lane * g.laneH + g.vOffset) + 'px' }"
                  :title="it.members.length + ' more — click to expand'"
                  @click.stop="openCluster($event, it)"
                >+{{ it.members.length }}</div>

                <!-- Event: bar from start to end date -->
                <div
                  v-else-if="it.type === 'bar'"
                  class="event-bar"
                  :data-item-id="it.m.id"
                  :class="[chipState(it.m), { draggable: !props.readOnly && !it.m.sourceSystem }]"
                  :style="barStyleFull(it, itemColor(it.m), g.laneH, g.vOffset)"
                  @mouseenter="hoveredMs = it.m"
                  @mouseleave="hoveredMs = null"
                  @pointerdown="startDrag($event, it, 'move')"
                  @click.stop="onChipClick($event, it.m, g.row.swimlane.color)"
                  @dblclick.stop="onEdit(it.m)"
                >
                  <span v-if="it.continuesLeft" class="bar-arrow">◀</span>
                  <span class="bar-title" :class="{ 'bar-title-out': it.labelOutside }">
                    <MarkerIcon
                      :shape="markerOf(it.m)"
                      :fill="markerFillFor(it.m)"
                      :color="itemColor(it.m)"
                      :size="settings.items.markerSize"
                      :stroke-width="settings.items.markerStroke"
                      class="bar-marker"
                    />
                    <span v-if="it.m.sourceSystem" class="chip-lock" title="Synced — read-only"><Lock :size="10" :stroke-width="2.5" /></span>{{ it.m.title }}<MaturityGlyph v-if="it.m.maturity" :level="it.m.maturity" variant="grid" :size="matSize" :color="'var(--clr-text-2)'" :title="maturityTitle(it.m.maturity)" class="mk-mat" /><AlertTriangle v-if="riskIds.has(it.m.id)" class="risk-badge" :size="settings.items.markerSize" :stroke-width="settings.items.markerStroke" color="#FF3B30" /><Clock v-if="lateIds.has(it.m.id)" class="late-badge" title="Overdue" :size="settings.items.markerSize" :stroke-width="settings.items.markerStroke" color="#FF3B30" /><AlertTriangle v-if="resourceConflictIds.has(it.m.id)" class="conflict-badge" :title="conflictTitle(it.m.id)" :size="settings.items.markerSize" :stroke-width="settings.items.markerStroke" color="#FF3B30" />
                  </span>
                  <span v-if="it.continuesRight" class="bar-arrow">▶</span>
                  <template v-if="!props.readOnly && !it.m.sourceSystem">
                    <span v-if="!it.continuesLeft" class="bar-handle bar-handle-l" @pointerdown.stop="startDrag($event, it, 'resize-l')" @click.stop @dblclick.stop></span>
                    <span v-if="!it.continuesRight" class="bar-handle bar-handle-r" @pointerdown.stop="startDrag($event, it, 'resize-r')" @click.stop @dblclick.stop></span>
                  </template>
                </div>
              </template>

              <span
                v-if="!props.readOnly && !g.row.swimlane.sourceSystem && addHint.key === rowKey(g.row) && !hoveredMs && !tooltip.visible"
                class="track-add-hint"
                :style="{ left: addHint.x + 'px' }"
              >+</span>
            </td>
          </tr>
        </template>

        <tr v-if="tableRows.length === 0">
          <td :colspan="15" class="empty-state">
            <template v-if="!props.readOnly">
              No areas defined yet — <button class="empty-link" @click="emit('manage', 'areas')">open Project settings</button> to get started.
            </template>
            <template v-else>
              No areas defined yet.
            </template>
          </td>
        </tr>
      </tbody>
    </table>

    <div class="track-spacer"></div>

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
          <MarkerIcon :shape="markerOf(tooltip.ms)" :color="itemColor(tooltip.ms)" :size="14" :fill="markerFillFor(tooltip.ms)" class="tooltip-ico" />
          <span class="tooltip-title">{{ tooltip.ms.title }}</span>
          <button v-if="!tooltip.ms.sourceSystem && (!props.readOnly || canProposeChanges())" type="button" class="tooltip-edit" @click.stop="props.readOnly ? (emit('propose-milestone', tooltip.ms), tooltip.visible = false) : onEdit(tooltip.ms)">{{ props.readOnly ? 'Propose change' : 'Edit' }}</button>
        </div>
        <div class="tooltip-fields">
          <div v-if="tooltip.ms.assigneeId && memberName(tooltip.ms.assigneeId)" class="tooltip-field">
            <span class="tf-label">Who</span>
            <button type="button" class="tf-val tf-who" title="View profile" @click.stop="openProfile(memberById(tooltip.ms.assigneeId), $event)">{{ memberName(tooltip.ms.assigneeId) }}</button>
          </div>
          <div v-if="tooltip.ms.data?.what" class="tooltip-field">
            <span class="tf-label">What</span>
            <span class="tf-val tf-clamp">{{ tooltip.ms.sourceSystem ? stripMarkdown(tooltip.ms.data.what) : tooltip.ms.data.what }}</span>
          </div>
          <div v-if="tooltip.ms.data?.why" class="tooltip-field">
            <span class="tf-label">Why</span>
            <span class="tf-val">{{ tooltip.ms.data.why }}</span>
          </div>
          <div v-if="tooltip.ms.data?.how" class="tooltip-field">
            <span class="tf-label">Where</span>
            <span class="tf-val">{{ tooltip.ms.data.how }}</span>
          </div>
          <div v-if="tooltip.ms.kind === 'event'" class="tooltip-field tooltip-field-dates">
            <span class="tf-label">Start</span>
            <span class="tf-val">{{ formatDate(tooltip.ms.startDate || tooltip.ms.when) }}</span>
            <template v-if="tooltip.ms.endDate">
              <span class="tf-label tfd-end-label">End</span>
              <span class="tf-val">{{ formatDate(tooltip.ms.endDate) }}</span>
            </template>
          </div>
          <div v-else-if="tooltip.ms.when" class="tooltip-field">
            <span class="tf-label">When</span>
            <span class="tf-val">{{ formatDate(tooltip.ms.when) }}</span>
          </div>
        </div>
        <div v-if="tooltip.ms.progress != null" class="tooltip-progress">
          <span class="tp-label">Progress</span>
          <span class="tp-bar"><span class="tp-fill" :style="{ width: tooltip.ms.progress + '%', background: tooltip.color }"></span></span>
          <span class="tp-pct">{{ tooltip.ms.progress }}%</span>
        </div>
        <div v-if="lateIds.has(tooltip.ms.id)" class="tooltip-late">
          <Clock :size="12" :stroke-width="2.2" />
          <span>Overdue — not finished past its date</span>
        </div>
        <div v-for="c in itemConflicts(tooltip.ms.id)" :key="c.resourceId + ':' + c.otherId" class="tooltip-conflict">
          <AlertTriangle :size="12" :stroke-width="2.2" />
          <span>Conflicts with <strong>{{ c.otherTitle }}</strong> over <strong>{{ c.resourceTitle }}</strong> ({{ c.when }})</span>
        </div>
        <div v-if="linkedMilestones.length > 0" class="tooltip-links">
          <span class="tl-label">Blocked by</span>
          <div class="tl-items">
            <div v-for="lm in linkedMilestones.slice(0, 10)" :key="lm.id" class="tl-item" :class="{ 'tl-late': lateBlockerIds.has(lm.id) }" @click.stop="selectFromTooltip(lm, $event)">
              <MarkerIcon :shape="markerOf(lm)" :color="itemColor(lm)" :size="12" :fill="markerFillFor(lm)" class="tl-ico" />
              <span class="tl-title">{{ lm.title }}</span>
              <AlertTriangle v-if="lateBlockerIds.has(lm.id)" class="tl-late-mark" :size="12" :stroke-width="2.5" title="Late — still blocking past its date" />
              <span v-if="lm.when || lm.startDate" class="tl-date">{{ formatDate(lm.when || lm.startDate) }}</span>
            </div>
            <div v-if="linkedMilestones.length > 10" class="tl-more">
              +{{ linkedMilestones.length - 10 }} more
            </div>
          </div>
        </div>

        <div v-if="parentMilestones.length > 0" class="tooltip-links">
          <span class="tl-label">Blocks</span>
          <div class="tl-items">
            <div v-for="pm in parentMilestones.slice(0, 10)" :key="pm.id" class="tl-item" @click.stop="selectFromTooltip(pm, $event)">
              <MarkerIcon :shape="markerOf(pm)" :color="itemColor(pm)" :size="12" :fill="markerFillFor(pm)" class="tl-ico" />
              <span class="tl-title">{{ pm.title }}</span>
              <span v-if="pm.when || pm.startDate" class="tl-date">{{ formatDate(pm.when || pm.startDate) }}</span>
            </div>
            <div v-if="parentMilestones.length > 10" class="tl-more">
              +{{ parentMilestones.length - 10 }} more
            </div>
          </div>
        </div>
        <div v-if="usesMilestones.length > 0" class="tooltip-links">
          <span class="tl-label">Uses</span>
          <div class="tl-items">
            <div v-for="um in usesMilestones.slice(0, 10)" :key="um.id" class="tl-item" @click.stop="openRelated(um, $event)">
              <MarkerIcon :shape="markerOf(um)" :color="itemColor(um)" :size="12" :fill="markerFillFor(um)" class="tl-ico" />
              <span class="tl-title">{{ um.title }}<span v-if="um._pinV" class="tl-ver">· v{{ um._pinV }}</span></span>
              <span class="tl-date">{{ backlogMeta(um) }}</span>
            </div>
            <div v-if="usesMilestones.length > 10" class="tl-more">+{{ usesMilestones.length - 10 }} more</div>
          </div>
        </div>

        <div v-if="usedByMilestones.length > 0" class="tooltip-links">
          <span class="tl-label">Used by</span>
          <div class="tl-items">
            <div v-for="ub in usedByMilestones.slice(0, 10)" :key="ub.id" class="tl-item" @click.stop="openRelated(ub, $event)">
              <MarkerIcon :shape="markerOf(ub)" :color="itemColor(ub)" :size="12" :fill="markerFillFor(ub)" class="tl-ico" />
              <span class="tl-title">{{ ub.title }}<span v-if="ub._pinV" class="tl-ver">· v{{ ub._pinV }}</span></span>
              <span class="tl-date">{{ backlogMeta(ub) }}</span>
            </div>
            <div v-if="usedByMilestones.length > 10" class="tl-more">+{{ usedByMilestones.length - 10 }} more</div>
          </div>
        </div>

        <div v-if="!tooltip.ms.sourceSystem" class="tooltip-meta">
          <div v-if="tooltip.ms.createdBy" class="tm-row"><User :size="12" :stroke-width="2.2" /><span>Added by {{ whoName(tooltip.ms.createdBy) }}<span v-if="tooltip.ms.createdAt" class="tm-when"> · {{ fmtStamp(tooltip.ms.createdAt) }}</span></span></div>
          <div v-if="tooltip.ms.updatedBy && (tooltip.ms.version || 1) > 1" class="tm-row"><Pencil :size="12" :stroke-width="2.2" /><span>Edited by {{ whoName(tooltip.ms.updatedBy) }}<span v-if="tooltip.ms.updatedAt" class="tm-when"> · {{ fmtStamp(tooltip.ms.updatedAt) }}</span></span></div>
          <button type="button" class="tm-row tm-ver-row tm-ver-btn" title="View version history" @click.stop="openHistory(tooltip.ms)">
            <span class="tm-ver-label">Version</span>
            <span class="tm-ver">v{{ tooltip.ms.version || 1 }} <History :size="11" /></span>
          </button>
        </div>
      </div>
    </Transition>
  </Teleport>

  <!-- Cluster popover: the markers collapsed under a "+N" chip -->
  <Teleport to="body">
    <div v-if="clusterPop.visible" class="cluster-backdrop" @click="clusterPop.visible = false" @wheel="clusterPop.visible = false"></div>
    <div v-if="clusterPop.visible" class="cluster-pop" :style="{ left: clusterPop.x + 'px', top: clusterPop.y + 'px' }">
      <div class="cl-head">{{ clusterPop.items.length }} milestones</div>
      <div class="cl-list">
        <div v-for="m in clusterPop.items" :key="m.id" class="tl-item" @click.stop="pickCluster(m)">
          <span class="tl-dot" :style="{ background: m.color || swimlaneColor(m.swimlaneId) }"></span>
          <span class="tl-title">{{ m.title }}</span>
          <span class="tl-date">{{ MONTHS[(m.month || 1) - 1] }}</span>
        </div>
      </div>
    </div>
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
import { useAppStore, MONTHS, MATURITY_STAGES, store, viewItems, settings, groups, ui, riskIds, riskByItem, lateIds, stripMarkdown, memberName, memberById, openProfile, itemTypeByKey, itemStatus, statusColor, toneColor, canProposeChanges, resourceConflicts, resourceConflictIds, checkResourceConflicts } from '../stores/useAppStore.js'
import { AlertTriangle, Clock, Lock, User, Pencil, History } from 'lucide-vue-next'
import MarkerIcon from './MarkerIcon.vue'
import MaturityGlyph from './MaturityGlyph.vue'

function maturityTitle(level) {
  return level ? `Maturity: ${MATURITY_STAGES[level - 1]} (${level}/4)` : ''
}

// Maturity glyph scales with the same size slider as the marker / warning / time
// icons, so every item icon stays visually consistent. Cell ≈ markerSize / 2.3 →
// the 2×2 grid matches the marker's height.
const matSize = computed(() => Math.max(3, Math.round(settings.items.markerSize * 0.42)))
// Colour on the timeline encodes STATUS — the marker carries its status colour,
// To Do included (the neutral status colour, not the lane colour). Area comes from
// the row, type from the icon shape, so colour means exactly one thing: the state.
function itemColor(m) {
  const st = itemStatus(m)
  return st ? statusColor(st) : toneColor('neutral')
}
function dotExtra() { return 0 } // status dot removed; kept as a no-op for the geometry calls

// Fixed geometry — months are a fixed width so date math is exact regardless of
// label length (labels overflow freely to the right).
const clampW = (v, d) => Math.min(280, Math.max(150, v ?? d)) // frozen columns: 150–280px
const AREA_W = computed(() => clampW(settings.layout?.areaWidth, 168))
const SUB_W = computed(() => clampW(settings.layout?.subAreaWidth, 240))
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
const msLaneH = computed(() => Math.max(20, msPillH.value + (settings.items.margin ?? ITEM_AIR) * 2))
const eventLaneH = computed(() => Math.max(20, eventPillH.value + (settings.items.margin ?? ITEM_AIR) * 2))
const DAYS_PER_COL = 30.4 // avg days/month for px↔day conversion when dragging
// Reserved space right of December so end-of-year labels flow right into a
// defined gutter instead of creating dead horizontal scroll space.
const RIGHT_PAD = 200

const props = defineProps({
  zoom: { type: Number, default: 1 },
  readOnly: { type: Boolean, default: false },
})
const emit = defineEmits(['add-milestone', 'edit-milestone', 'propose-milestone', 'show-history', 'manage'])
const { getLinkedIds, dependsOnIds, dependentIds, updateMilestone, setView } = useAppStore()

// Months fill the available width at 100% zoom; the zoom control then widens the
// months (horizontal detail) WITHOUT scaling the header height or fonts.
const wrapEl = ref(null)
const wrapW = ref(1200)
const headRowEl = ref(null)     // months header row — measured to stick the week row below it
const monthHeadH = ref(44)
let resizeObs = null
// View granularity: 'year' = 12 month columns, 'month' = one month of day columns.
const granularity = computed(() => store.granularity)
const viewMonth = computed(() => store.viewMonth)
const unitCount = computed(() => granularity.value === 'month' ? daysInMonth(store.year, viewMonth.value) : 12)
const minColW = computed(() => granularity.value === 'month' ? 26 : MIN_COL_W)
// Column header labels: month names, or day numbers 1..N for the focused month.
const columns = computed(() =>
  granularity.value === 'month'
    ? Array.from({ length: unitCount.value }, (_, i) => String(i + 1))
    : MONTHS)
const trackColspan = computed(() => unitCount.value + 1) // columns + right gutter

// At 100% zoom the months fill only ~90% of the available width (not 100%), so there
// is a bit of breathing room to the right of DEC by default. The zoom control scales
// up from there. Effectively what used to be "90%" is now the "100%" baseline.
const FILL_FACTOR = 0.9
const baseColW = computed(() => Math.max(minColW.value, FILL_FACTOR * (wrapW.value - AREA_W.value - SUB_W.value) / unitCount.value))
const COL_W = computed(() => baseColW.value * props.zoom)
// Total width of the time track (kept named MONTHS_W; it is the column area width
// in both granularities).
const MONTHS_W = computed(() => COL_W.value * unitCount.value)
// The right gutter grows to fill the viewport when zoomed out, so the grid/rows
// reach the edge instead of leaving the page background showing.
// The gutter must be wide enough to (a) fill the viewport and (b) hold the
// right-most item LABEL — labels overflow freely to the right of their marker, so
// items near the end of the timeline would otherwise be clipped at the table edge.
const gutterW = computed(() => Math.max(
  RIGHT_PAD,
  wrapW.value - AREA_W.value - SUB_W.value - MONTHS_W.value,
  maxLabelRight.value - MONTHS_W.value + 32,
))
const tableWidth = computed(() => AREA_W.value + SUB_W.value + MONTHS_W.value + gutterW.value)

// ── Rows ────────────────────────────────────────────────────────────────────
const tableRows = computed(() => {
  const rows = []
  for (const sw of store.swimlanes) {
    if (sw.hidden) continue // consumer-hidden (e.g. a subscribed lane)
    if (sw.sourceSystem) continue // mirrored Git repos live in Source Control, not on the timeline
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
// Whether a (year, month) falls inside the visible window — the whole year, or
// just the focused month in day-granularity.
function inWindow(y, mo) {
  if (y !== store.year) return false
  return granularity.value === 'year' || mo === viewMonth.value
}
// x within the time track (0..MONTHS_W) for a date in view, clamped to the edges.
function dateX(dateStr) {
  if (!dateStr) return 0
  const { y, mo, day } = ymOf(dateStr)
  if (granularity.value === 'month') {
    if (y < store.year || (y === store.year && mo < viewMonth.value)) return 0
    if (y > store.year || (y === store.year && mo > viewMonth.value)) return MONTHS_W.value
    return (day - 1) * COL_W.value
  }
  if (y < store.year) return 0
  if (y > store.year) return MONTHS_W.value
  return ((mo - 1) + (day - 1) / daysInMonth(store.year, mo)) * COL_W.value
}
// Same mapping for a Date object (week ticks, year-granularity only).
function dateXOf(dt) {
  return dateX(fmtDate(dt))
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
  if (granularity.value === 'month') {
    const di = Math.floor(clamped / COL_W.value)
    const dim = daysInMonth(store.year, viewMonth.value)
    return new Date(store.year, viewMonth.value - 1, Math.min(dim, di + 1))
  }
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
// Measure a label's pixel width with the ACTUAL item font (weight + size + the
// app's system stack) instead of estimating — bars decide inside/outside layout
// from this, so a character-count guess left short bars looking cramped.
const LABEL_FONT_FAMILY = "-apple-system, BlinkMacSystemFont, 'Helvetica Neue', Arial, sans-serif"
let _measureCtx = null
function estTextW(t) {
  if (!t) return 0
  if (typeof document !== 'undefined') {
    if (!_measureCtx) _measureCtx = document.createElement('canvas').getContext('2d')
    if (_measureCtx) {
      _measureCtx.font = `${settings.items.fontWeight} ${settings.items.fontSize}px ${LABEL_FONT_FAMILY}`
      return Math.ceil(_measureCtx.measureText(t).width)
    }
  }
  return Math.ceil(t.length * settings.items.fontSize * 0.58) // fallback (no DOM)
}
// The icon & its fill now come from the item's TYPE (single source of truth);
// the Diamond is just a safety net for an item whose type can't be resolved.
function markerOf(m) {
  const t = itemTypeByKey(m.typeKey || m.kind || 'milestone')
  return (t && t.icon) || 'l:Diamond'
}
function markerFillFor(m) {
  const t = itemTypeByKey(m.typeKey || m.kind || 'milestone')
  return !(t && t.fill === false)
}
// Only timeline-family types belong on the timeline. Work-item / container types
// (e.g. a "Task" backlog item) live in the Explorer and must never render here,
// even if they somehow carry a lane + date.
function isTimelineItem(m) {
  const fam = itemTypeByKey(m.typeKey || m.kind || 'milestone')?.family
  return !fam || fam === 'timeline-point' || fam === 'timeline-range'
}
function isBar(m) {
  return m.kind === 'event' && m.startDate && m.endDate && m.endDate > m.startDate
}
function barInfo(m) {
  if (granularity.value === 'month') {
    const mm = String(viewMonth.value).padStart(2, '0')
    const dim = daysInMonth(store.year, viewMonth.value)
    const winStart = `${store.year}-${mm}-01`
    const winEnd = `${store.year}-${mm}-${String(dim).padStart(2, '0')}`
    if (m.endDate < winStart || m.startDate > winEnd) return null
    return {
      startX: dateX(m.startDate),
      endX: dateX(m.endDate),
      continuesLeft: m.startDate < winStart,
      continuesRight: m.endDate > winEnd,
    }
  }
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
function rowItems(row) {
  const swId = row.swimlane.id
  const subId = row.subLane?.id ?? null
  const subIds = (row.swimlane.subLanes || []).map(s => s.id)
  // The first sub-lane row also "catches" items whose sub-lane is null/unknown,
  // so a freshly-created (e.g. approved-CR) item never silently disappears.
  const isFirstSub = !!row.subLane && row.showLaneCell
  const items = []
  for (const m of viewItems.value) {
    if (!isTimelineItem(m)) continue // off-timeline types (backlog/folder) never show here
    if (m.swimlaneId !== swId) continue
    const mSub = m.subLaneId ?? null
    const here = subId === null
      ? true
      : (mSub === subId || (isFirstSub && (mSub === null || !subIds.includes(mSub))))
    if (!here) continue
    if (isBar(m)) {
      const info = barInfo(m)
      if (!info) continue
      const width = Math.max(info.endX - info.startX, 16)
      const labelW = estTextW(m.title)
      const hasMarker = true // the type's icon always renders at the bar start now
      // Account for the icon AND the bar padding it needs around it.
      const iconW = hasMarker ? (settings.items.markerSize + settings.items.iconGap + settings.items.padding + 8 + dotExtra(m)) : 0
      // Badges that trail the title and also need room: the maturity glyph (a 2×2
      // grid, ≈2.3× its cell size) and the risk warning triangle. Without these the
      // glyph could overflow a bar the title only just fit into.
      const trailW = (m.maturity ? Math.ceil(matSize.value * 2.3) + settings.items.iconGap + 5 : 0)
        + (riskIds.value.has(m.id) ? settings.items.markerSize + settings.items.iconGap : 0)
        + (lateIds.value.has(m.id) ? settings.items.markerSize + settings.items.iconGap : 0)
        + (resourceConflictIds.value.has(m.id) ? settings.items.markerSize + settings.items.iconGap : 0)
      // If the icon + title (+ trailing badges) don't fit, the whole unit moves to
      // the right of the bar (tight together); otherwise it sits inside the bar.
      const labelOutside = iconW + labelW + trailW + settings.items.labelBuffer > width
      const x1 = labelOutside ? info.startX + width + 10 + iconW + labelW + trailW : info.startX + width
      items.push({
        key: m.id, m, type: 'bar', x: info.startX, width, labelOutside,
        x0: info.startX, x1,
        continuesLeft: info.continuesLeft, continuesRight: info.continuesRight,
      })
    } else {
      const ad = anchorDate(m)
      { const w = ymOf(ad); if (!inWindow(w.y, w.mo)) continue }
      const x = dateX(ad)
      const labelW = estTextW(m.title)
      const pad = settings.items.padding
      // The maturity glyph / risk badge trail the title here too — count them so
      // lane packing reserves the full extent.
      const trailW = (m.maturity ? Math.ceil(matSize.value * 2.3) + settings.items.iconGap + 5 : 0)
        + (riskIds.value.has(m.id) ? settings.items.markerSize + settings.items.iconGap : 0)
        + (lateIds.value.has(m.id) ? settings.items.markerSize + settings.items.iconGap : 0)
        + (resourceConflictIds.value.has(m.id) ? settings.items.markerSize + settings.items.iconGap : 0)
      items.push({
        key: m.id, m, type: 'point', x,
        x0: x - 6 - pad - dotExtra(m),
        x1: x + 16 + labelW + trailW + pad,
      })
    }
  }

  return packLanes(items, settings.items.density, settings.items.densityRows)
}

const PACK_GAP = 6

// Stack (default): each item drops to the first lane that's free where it starts.
function packStack(items) {
  const laneRight = []
  for (const it of items) {
    let lane = laneRight.findIndex(r => it.x0 >= r + PACK_GAP)
    if (lane === -1) { lane = laneRight.length; laneRight.push(0) }
    laneRight[lane] = it.x1
    it.lane = lane
  }
  return { items, laneCount: Math.max(1, laneRight.length) }
}

// Rail: collapse point markers onto one tick row (below any bars/ghosts), shown
// as small dots — keeps the row 1 lane tall no matter how dense it is.
function packRail(items) {
  const normal = items.filter(i => i.type !== 'point')
  const pts = items.filter(i => i.type === 'point')
  const { laneCount: base } = packStack(normal)
  const railLane = normal.length ? base : 0
  for (const it of pts) { it.lane = railLane; it.rail = true }
  return { items, laneCount: Math.max(1, railLane + (pts.length ? 1 : 0)) }
}

// Cluster: cap the stack at K lanes — show K-1 individually, collapse the rest at
// that spot into a "+N" chip (click to expand). Only for pure point rows.
function packCluster(items, maxLanes) {
  const K = Math.max(2, maxLanes || 3)
  const indiv = K - 1
  const laneRight = new Array(indiv).fill(-Infinity)
  const clusters = []
  const out = []
  const CLUSTER_W = 30, MERGE = 12
  let maxLane = 0
  for (const it of items) {
    let lane = -1
    for (let i = 0; i < indiv; i++) { if (it.x0 >= laneRight[i] + PACK_GAP) { lane = i; break } }
    if (lane !== -1) {
      laneRight[lane] = it.x1; it.lane = lane; out.push(it); maxLane = Math.max(maxLane, lane)
    } else {
      let c = clusters.length ? clusters[clusters.length - 1] : null
      if (!c || it.x > c.x1 + MERGE) {
        c = { type: 'cluster', key: 'cl-' + it.key, x: it.x, x0: it.x - 8, x1: it.x + CLUSTER_W, lane: indiv, members: [] }
        clusters.push(c); out.push(c)
      }
      c.members.push(it)
      c.x1 = Math.max(c.x1, it.x + CLUSTER_W)
    }
  }
  if (clusters.length) maxLane = Math.max(maxLane, indiv)
  return { items: out, laneCount: maxLane + 1 }
}

function packLanes(items, mode, maxLanes) {
  items.sort((a, b) => a.x0 - b.x0)
  if (mode === 'cluster' && items.length && items.every(i => i.type === 'point')) return packCluster(items, maxLanes)
  if (mode === 'rail') return packRail(items)
  return packStack(items)
}

const grid = computed(() =>
  tableRows.value.map(row => {
    const { items, laneCount } = rowItems(row)
    // Tight rows: only as tall as needed; event rows a touch taller than milestone-only rows.
    const isEvent = items.some(i => i.type === 'bar')
    const laneH = isEvent ? eventLaneH.value : msLaneH.value
    const vOffset = Math.round((laneH - (isEvent ? eventPillH.value : msPillH.value)) / 2)
    return { row, items, laneCount, laneH, vOffset, trackHeight: laneCount * laneH }
  })
)

// Right-most pixel reached by any item (marker + overflowing label) across all
// rows — drives the gutter width so long labels at the timeline end aren't clipped.
const maxLabelRight = computed(() => {
  let m = 0
  for (const g of grid.value) {
    for (const it of g.items) {
      if (it.x1 > m) m = it.x1
    }
  }
  return m
})

// Index (0-based) of the column representing "now": the current month in
// year-granularity, or the current day in month-granularity. -1 when off-screen.
const currentColIndex = computed(() => {
  const now = new Date()
  if (now.getFullYear() !== store.year) return -1
  if (granularity.value === 'month') {
    return now.getMonth() + 1 === viewMonth.value ? now.getDate() - 1 : -1
  }
  return now.getMonth()
})
function isCurrentCol(i) {
  return currentColIndex.value === i
}
const todayX = computed(() => {
  if (currentColIndex.value < 0) return -1
  return dateX(fmtDate(new Date()))
})
const monthHlColor = computed(() => hexAlpha(settings.monthHighlight.color, settings.monthHighlight.opacity))
const dayLineColor = computed(() => hexAlpha(settings.dayLine.color, settings.dayLine.opacity))

// ISO week numbers across the viewed year, thinned out when columns get narrow.
const weekTicks = computed(() => {
  if (granularity.value === 'month') return [] // day columns already label every day
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
  for (let i = 1; i < unitCount.value; i++) xs.push(i * COL_W.value) // between every column
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
  if (granularity.value === 'month') return []
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
  // A popup is open → this click just dismisses it (don't also trigger "+" add).
  if (tooltip.visible) { closeTooltip(); return }
  // Synced (read-only) lanes only get their items from their source.
  if (row.swimlane.sourceSystem) return
  const rect = e.currentTarget.getBoundingClientRect()
  const x = e.clientX - rect.left
  if (x > MONTHS_W.value) return   // ignore clicks in the empty right gutter
  if (granularity.value === 'month') {
    const d = xToDate(x)
    emit('add-milestone', { swimlane: row.swimlane, subLane: row.subLane, month: viewMonth.value, date: fmtDate(d) })
    return
  }
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
  if (granularity.value === 'month') {
    const di = Math.min(unitCount.value - 1, Math.max(0, Math.floor(x / COL_W.value)))
    addHint.x = (di + 0.5) * COL_W.value
    return
  }
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
    color: 'var(--clr-text)', // title stays off-white; the status colour lives in the bar fill/border
    '--it-status': color,     // the active outline (border mode / selection) uses the status colour
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
      // Block-mode exclusive resources (#128): a drag that would over-book a
      // capacity-1 resource is refused — the bar snaps back (no update applied).
      const resourceIds = store.links.filter(l => l.a === ds.id && (l.rel || 'depends-on') === 'uses').map(l => l.b)
      const blocked = resourceIds.length
        ? checkResourceConflicts({ id: ds.id, start: s, end: en, resourceIds }).filter(c => c.mode === 'block')
        : []
      if (blocked.length) {
        const c = blocked[0]
        alert(`${c.resourceTitle} is already booked ${c.when} by “${c.otherTitle}”.\nThe move was reverted.`)
      } else {
        const [yy, mm] = s.split('-').map(Number)
        updateMilestone(ds.id, { startDate: s, endDate: en, when: s, year: yy, month: mm })
      }
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
// Uses / Used-by (composition edge to backlog items) — shown in the tooltip too.
const usesMilestones = computed(() => {
  if (!selectedMs.value) return []
  const id = selectedMs.value.id
  const byId = new Map(store.milestones.map(m => [m.id, m]))
  return store.links.filter(l => l.a === id && l.rel === 'uses' && byId.has(l.b))
    .map(l => ({ ...byId.get(l.b), _pinV: l.version ?? null }))
})
const usedByMilestones = computed(() => {
  if (!selectedMs.value) return []
  const id = selectedMs.value.id
  const byId = new Map(store.milestones.map(m => [m.id, m]))
  return store.links.filter(l => l.b === id && l.rel === 'uses' && byId.has(l.a))
    .map(l => ({ ...byId.get(l.a), _pinV: l.version ?? null }))
})
function backlogDot(m) { return itemTypeByKey(m.typeKey || m.kind)?.color || '#8a8a8e' }
function backlogMeta(m) { return itemTypeByKey(m.typeKey || m.kind)?.label || (m.typeKey || m.kind || '') }
// Open a related item: timeline items scroll into view; off-timeline (backlog)
// items open in the Explorer (set the view first, then focus once it's mounted).
function openRelated(m, e) {
  if (m.swimlaneId) { selectFromTooltip(m, e); return }
  tooltip.visible = false
  ui.explorerItemId = m.id
  ui.explorerItemVersion = m._pinV || null
  setView('explorer') // pushes {view:explorer, item:m.id}; ExplorerView opens it
}
function swimlaneColor(swimlaneId) {
  return store.swimlanes.find(s => s.id === swimlaneId)?.color ?? '#888'
}
function formatDate(dateStr) {
  if (!dateStr) return ''
  const [y, m, day] = dateStr.split('-').map(Number)
  return new Date(y, m - 1, day).toLocaleDateString('en-US', { day: 'numeric', month: 'long', year: 'numeric' })
}
// Attribution helpers: resolve a member id to a name and format an ISO timestamp.
function whoName(id) { return id ? (memberName(id) || 'someone') : '' }
function fmtStamp(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  return isNaN(d) ? '' : d.toLocaleDateString('en-US', { day: 'numeric', month: 'short', year: 'numeric' })
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
function chipState(m) {
  // Hovering a facet (filter dock) highlights its matching items, dims the rest.
  if (ui.highlightIds) {
    return ui.highlightIds.has(m.id) ? 'chip-related' : 'chip-dimmed'
  }
  if (!activeMs.value) return ''
  if (m.id === activeMs.value.id) return 'chip-active'
  if (relatedIds.value.has(m.id)) return 'chip-related'
  return 'chip-dimmed'
}

// ── Tooltip ───────────────────────────────────────────────────────────────────
const tooltip = reactive({ visible: false, ms: null, x: 0, chipTop: 0, chipBottom: 0, color: '' })

// Version history opened from the tooltip's version row → handled at the app
// level (a top-level overlay), so it reliably covers the whole screen.
function openHistory(m) { emit('show-history', m); tooltip.visible = false }
const tooltipStyle = computed(() => {
  const margin = 12
  const tipW = 400
  const left = Math.min(tooltip.x, window.innerWidth - tipW - margin)
  const spaceBelow = window.innerHeight - tooltip.chipBottom - margin
  const spaceAbove = tooltip.chipTop - margin
  // Open toward the side with more room when below is tight; cap the height to the
  // available space and scroll, so the tooltip is never cut off on a small screen.
  const openUp = spaceBelow < 340 && spaceAbove > spaceBelow
  const maxH = Math.max(180, (openUp ? spaceAbove : spaceBelow) - 8)
  return {
    position: 'fixed',
    ...(openUp
      ? { bottom: `${window.innerHeight - tooltip.chipTop + 8}px`, top: 'auto' }
      : { top: `${tooltip.chipBottom + 8}px`, bottom: 'auto' }),
    left: `${Math.max(margin, left)}px`,
    width: `${tipW}px`,
    maxHeight: `${maxH}px`,
    overflowY: 'auto',
    zIndex: 9999,
  }
})
// Which of this item's blockers are themselves late — marked inline (⚠) in the
// "Blocked by" list instead of a separate section.
const lateBlockerIds = computed(() => {
  const id = tooltip.ms?.id
  const list = (id && riskByItem.value[id]) ? riskByItem.value[id] : []
  return new Set(list.map(p => p.id))
})

// Exclusive-resource over-bookings for an item (#128) — tooltip / badge detail.
function itemConflicts(id) { return resourceConflicts.value[id] || [] }
function conflictTitle(id) {
  const list = itemConflicts(id)
  if (!list.length) return ''
  return list.map(c => `Conflicts with “${c.otherTitle}” over ${c.resourceTitle} (${c.when})`).join('\n')
}

function onEdit(m) {
  // Opening the editor: dismiss the info tooltip (the dblclick's .stop would
  // otherwise leave it open behind/beside the modal).
  tooltip.visible = false
  selectedMs.value = null
  emit('edit-milestone', m)
}

// Cluster "+N" popover (density: cluster mode)
const clusterPop = reactive({ visible: false, x: 0, y: 0, items: [] })
function openCluster(ev, it) {
  const r = ev.currentTarget.getBoundingClientRect()
  clusterPop.items = it.members.map(x => x.m)
  clusterPop.x = Math.min(r.left, window.innerWidth - 280)
  clusterPop.y = r.bottom + 6
  clusterPop.visible = true
  tooltip.visible = false
}
function pickCluster(m) {
  clusterPop.visible = false
  onEdit(m)
}

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
  tooltip.color = m.color || swimlaneColor(m.swimlaneId)
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
    tooltip.color = m.color || swimlaneColor(m.swimlaneId)
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
  text-align: left;
}
.th-sub {
  position: sticky; z-index: 30; /* left offset is bound to the Area width inline */
  text-align: left;
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
.area-label { display: flex; align-items: center; min-width: 0; }
.area-name { font-size: 13px; font-weight: 600; color: var(--clr-text); letter-spacing: -0.1px;
  min-width: 0; overflow-wrap: anywhere; }

.td-sub {
  position: sticky; z-index: 10; /* left offset is bound to the Area width inline */
  background: var(--clr-surface-2);
  border-bottom: 1px solid var(--clr-border-light);
  border-right: 1px solid var(--clr-border-light);
  padding: 4px 14px;
  vertical-align: middle;
  white-space: nowrap;
  overflow: hidden;
  user-select: none; -webkit-user-select: none;
}
.sub-name { display: block; font-size: 12px; color: var(--clr-text-2); font-weight: 500;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

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
  -webkit-user-select: none;
  border-radius: var(--it-radius, 6px);
  z-index: 2;                 /* keep markers/labels above the "+" add hint */
  /* No filter transition: animating grayscale/brightness re-rasterises the inline
     content (marker + badges) each frame → sub-pixel "wiggle". */
  transition: opacity 0.18s ease, box-shadow 0.18s ease;
}
.mk-item:hover { filter: brightness(0.9); }
.mk-icon { flex-shrink: 0; }
.mk-label, .bar-title {
  text-box-trim: trim-both;
  text-box-edge: cap alphabetic;
}
.mk-label { position: relative; top: var(--it-label-y, 1px); color: inherit; }
.chip-lock { display: inline-flex; align-items: center; margin-right: 3px; opacity: 0.55; flex-shrink: 0; }

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
  -webkit-user-select: none;
  transition: box-shadow 0.18s ease, opacity 0.18s ease;
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

/* --- Dependency states --- */
.chip-active {
  box-shadow: 0 0 0 var(--it-ring, 2px) var(--it-status, currentColor), 0 2px 10px rgba(0,0,0,0.14);
  filter: brightness(1.05);
  border-radius: var(--it-radius, 6px);
  z-index: 5;
}
.chip-related {
  box-shadow: 0 0 0 var(--it-ring, 2px) var(--it-status, currentColor);
  filter: brightness(1.03);
  opacity: 1 !important;
  border-radius: var(--it-radius, 6px);
}
.chip-dimmed { opacity: 0.32 !important; filter: grayscale(0.3) !important; }

/* --- Item border mode: always / on-hover / off (width = --it-ring) --- */
/* Milestones (ring drawn in the item's STATUS colour) */
.bm-always .mk-item { box-shadow: 0 0 0 var(--it-ring, 2px) var(--it-status, currentColor); }
.bm-off .mk-item,
.bm-off .mk-item.chip-active,
.bm-off .mk-item.chip-related { box-shadow: none !important; }
/* Events (bar outline) */
.bm-hover .event-bar,
.bm-off .event-bar { border-color: transparent !important; }
.bm-hover .event-bar:hover,
.bm-hover .event-bar.chip-active,
.bm-hover .event-bar.chip-related { border-color: var(--it-status, currentColor) !important; }

/* --- Empty state --- */
.empty-state { text-align: center; padding: 80px 20px; color: var(--clr-text-3); font-size: 14px; }
.empty-link { background: none; color: var(--clr-accent); font-size: 14px; font-weight: 600; cursor: pointer; padding: 0; }
.empty-link:hover { text-decoration: underline; }

/* --- Legend --- */
/* The bottom legend + groups now live in the unified GroupLegend.vue dock. */

/* --- Tooltip --- */
.ms-tooltip {
  background: var(--clr-glass);
  backdrop-filter: blur(36px) saturate(2);
  -webkit-backdrop-filter: blur(36px) saturate(2);
  border: 1px solid var(--clr-border-light);
  border-radius: var(--r-lg);
  box-shadow: 0 8px 32px rgba(0,0,0,0.18);
  overflow: hidden;
}
.tooltip-progress { display: flex; align-items: center; gap: 11px; padding: 9px 14px; border-top: 1px solid var(--clr-border-light); }
.tp-label { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-text-3); flex-shrink: 0; }
.tp-bar { flex: 1; height: 6px; border-radius: 3px; background: rgba(127,127,127,0.22); overflow: hidden; }
.tp-fill { display: block; height: 100%; border-radius: 3px; transition: width 0.2s; }
.tp-pct { font-size: 12px; font-weight: 600; color: var(--clr-text); font-variant-numeric: tabular-nums; flex-shrink: 0; min-width: 34px; text-align: right; }
.tooltip-meta { padding: 8px 14px 10px; border-top: 1px solid var(--clr-border-light); display: flex; flex-direction: column; gap: 4px; }
.tm-row { display: flex; align-items: center; gap: 6px; font-size: 11px; color: var(--clr-text-3); }
.tm-row svg { flex-shrink: 0; }
.tm-when { color: var(--clr-text-3); }
.tm-ver-row { justify-content: space-between; }
.tm-ver-btn { width: 100%; background: none; border: none; cursor: pointer; padding: 1px 0; border-radius: 6px; transition: color 0.12s; }
.tm-ver-btn:hover { color: var(--clr-text); }
.tm-ver-btn:hover .tm-ver { background: rgba(0,113,227,0.12); color: var(--clr-accent); }
.tm-ver-label { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; }
.tm-ver { display: inline-flex; align-items: center; gap: 4px; font-size: 10px; font-weight: 700; color: var(--clr-text-2); background: var(--clr-surface-2); border-radius: 100px; padding: 2px 8px; transition: background 0.12s, color 0.12s; }
.tooltip-header {
  display: flex; align-items: center; gap: 8px;
  padding: 12px 14px 10px;
  border-bottom: 1px solid var(--clr-border-light);
}
.tooltip-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.tooltip-title { font-size: 13px; font-weight: 600; color: var(--clr-text); letter-spacing: -0.1px; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tooltip-edit { margin-left: auto; flex-shrink: 0; font-size: 12px; font-weight: 600; color: #fff; background: var(--clr-accent); border-radius: var(--r-md); padding: 4px 13px; transition: background 0.14s; }
.tooltip-edit:hover { background: var(--clr-accent-hover); }
.tooltip-fields { padding: 10px 14px 8px; display: flex; flex-direction: column; gap: 7px; }
.tooltip-field { display: grid; grid-template-columns: 64px 1fr; gap: 8px; align-items: baseline; }
/* Event dates: Start on the left, End right-aligned to the tooltip edge — both
   labels sit right next to their date (equal gap). */
.tooltip-field-dates { display: flex; align-items: baseline; gap: 8px; }
.tooltip-field-dates > .tf-label { flex-shrink: 0; }
.tfd-end-label { margin-left: auto; }
.tf-label { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-text-3); }
.tf-val { font-size: 12.5px; color: var(--clr-text); line-height: 1.45; }
.tf-who { background: none; border: none; padding: 0; cursor: pointer; text-align: left; font: inherit; }
.tf-who:hover { color: var(--clr-accent); text-decoration: underline; }
.tf-clamp { display: -webkit-box; -webkit-line-clamp: 6; -webkit-box-orient: vertical; overflow: hidden; }
/* Source-control section (its own divided block, below WHEN) */
.tooltip-scm { padding: 9px 14px; border-top: 1px solid var(--clr-border-light); display: flex; flex-direction: column; gap: 8px; }
.scm-row { display: flex; align-items: center; gap: 10px; min-width: 0; }
.scm-label { flex: 0 0 44px; font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-text-3); }
.scm-synced { display: inline-flex; align-items: center; gap: 6px; font-size: 12.5px; color: var(--clr-text-2); }

/* Density: rail dots + cluster chips */
.rail-dot {
  position: absolute;
  width: 10px; height: 10px; border-radius: 50%;
  box-shadow: 0 0 0 2px var(--clr-bg);
  cursor: pointer; z-index: 3;
  transition: transform 0.12s;
}
.rail-dot:hover { transform: scale(1.45); }
.mk-cluster {
  position: absolute;
  display: inline-flex; align-items: center; justify-content: center;
  height: 18px; padding: 0 7px;
  font-size: 11px; font-weight: 700;
  color: var(--clr-text-2); background: var(--clr-surface-2);
  border: 1px solid var(--clr-border-light); border-radius: 100px;
  cursor: pointer; z-index: 4; user-select: none; -webkit-user-select: none;
  transition: background 0.12s, color 0.12s, border-color 0.12s;
}
.mk-cluster:hover { background: var(--clr-accent); color: #fff; border-color: var(--clr-accent); }
.cluster-backdrop { position: fixed; inset: 0; z-index: 9998; }
.cluster-pop {
  position: fixed; z-index: 9999;
  min-width: 220px; max-width: 280px; max-height: 320px; overflow-y: auto;
  background: var(--clr-glass);
  backdrop-filter: blur(36px) saturate(2); -webkit-backdrop-filter: blur(36px) saturate(2);
  border: 1px solid var(--clr-border-light); border-radius: var(--r-lg);
  box-shadow: 0 8px 32px rgba(0,0,0,0.18);
  padding: 8px;
}
.cl-head { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-text-3); padding: 2px 4px 6px; }
.cl-list { display: flex; flex-direction: column; gap: 2px; }
.tooltip-links { padding: 8px 14px 10px; border-top: 1px solid var(--clr-border-light); }
.tl-label { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-text-3); display: block; margin-bottom: 6px; }
.tl-items { display: flex; flex-direction: column; gap: 5px; }
.tl-item { display: flex; align-items: center; gap: 6px; min-width: 0; cursor: pointer; padding: 2px 4px; margin: 0 -4px; border-radius: 5px; transition: background 0.1s; }
.tl-item:hover { background: var(--clr-surface-2); }
.tl-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.tl-ico, .tooltip-ico { flex-shrink: 0; }
.tl-title { font-size: 12.5px; color: var(--clr-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.tl-ver { color: var(--clr-text-3); font-variant-numeric: tabular-nums; }
.tl-date { font-size: 11px; color: var(--clr-text-3); white-space: nowrap; margin-left: auto; padding-left: 8px; flex-shrink: 0; }
/* A late blocker is flagged inline with a ⚠ (no separate section). */
.tl-late-mark { flex-shrink: 0; color: var(--clr-danger, #FF3B30); }
.tl-item.tl-late .tl-title { color: var(--clr-danger, #FF3B30); }
.tl-more { font-size: 11px; color: var(--clr-text-3); padding-left: 12px; }

/* Dependency-risk badge + tooltip section */
.risk-badge, .late-badge, .conflict-badge { flex-shrink: 0; margin-left: 3px; align-self: center; vertical-align: middle; }
.tooltip-late { display: flex; align-items: center; gap: 6px; padding: 8px 14px 10px;
  font-size: 10px; font-weight: 700; color: #FF3B30; text-transform: uppercase; letter-spacing: 0.5px; }
.tooltip-conflict { display: flex; align-items: flex-start; gap: 6px; padding: 6px 14px; color: #FF3B30;
  font-size: 11.5px; line-height: 1.35; }
.tooltip-conflict strong { font-weight: 700; }
.mk-mat { margin-left: 5px; }
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
