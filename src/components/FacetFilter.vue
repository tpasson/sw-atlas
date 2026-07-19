<template>
  <!-- Bottom-left filter dock: a free-text query that highlights matching timeline
       items (the rest dim) with type-ahead suggestions, plus the hover facets
       (Groups / Type / Maturity / Assignee). -->
  <div class="facet-dock">
    <div class="fd-search">
      <svg width="13" height="13" viewBox="0 0 13 13" fill="none" class="fd-search-ico"><circle cx="5.5" cy="5.5" r="4" stroke="currentColor" stroke-width="1.5"/><path d="M9 9l2.5 2.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
      <input
        v-model="query"
        class="fd-query"
        placeholder="Search items…"
        autocomplete="off"
        @focus="qFocus = true"
        @blur="qFocus = false"
        @keydown.enter="suggestions[0] && pick(suggestions[0])"
        @keydown.esc="query = ''"
      />
      <span v-if="tokens.length" class="fd-qcount">{{ queryMatches.length }}</span>
      <button v-if="query" type="button" class="fd-clear" title="Clear" @click="query = ''">×</button>

      <div v-if="qFocus && tokens.length" class="fd-suggest">
        <button
          v-for="m in suggestions"
          :key="m.id"
          type="button"
          class="fd-sug"
          @mousedown.prevent="pick(m)"
        >
          <MarkerIcon :shape="iconOf(m)" :color="colorOf(m)" :size="12" :fill="fillOf(m)" class="fd-sug-ico" />
          <span class="fd-sug-title">{{ m.title }}</span>
          <span class="fd-sug-meta">{{ sugMeta(m) }}</span>
        </button>
        <div v-if="!suggestions.length" class="fd-sug-empty">No matches</div>
        <div v-else-if="queryMatches.length > suggestions.length" class="fd-sug-more">+{{ queryMatches.length - suggestions.length }} more highlighted on the timeline</div>
      </div>
    </div>

    <span class="fd-sep"></span>

    <select class="fd-dim" v-model="dim" title="Filter dimension">
      <option value="group">Groups</option>
      <option value="type">Type</option>
      <option value="maturity">Maturity</option>
      <option value="status">Status</option>
      <option v-if="workspace.members.length" value="assignee">Assignee</option>
    </select>
    <div class="fd-values">
      <span
        v-for="f in facets"
        :key="f.key"
        class="fd-chip"
        @mouseenter="ui.highlightIds = f.ids"
        @mouseleave="ui.highlightIds = queryIds"
      >
        <span v-if="f.color" class="fd-dot" :style="{ background: f.color }"></span>
        <span class="fd-label" :title="f.label">{{ f.label }}</span>
        <span class="fd-count">{{ f.ids.size }}</span>
      </span>
      <span v-if="!facets.length" class="fd-empty">No {{ dimLabel }} to filter by yet</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onUnmounted } from 'vue'
import { store, groups, ui, itemTypes, itemTypeByKey, itemStatus, statusColor, memberName, MATURITY_STAGES, MONTHS, workspace } from '../stores/useAppStore.js'
import MarkerIcon from './MarkerIcon.vue'

const emit = defineEmits(['open-item'])

const dim = ref('group')
const dimLabel = computed(() => (dim.value === 'group' ? 'groups' : dim.value === 'type' ? 'types' : dim.value === 'assignee' ? 'assignees' : dim.value === 'status' ? 'statuses' : 'maturity stages'))

// Only items that actually sit on the timeline (have an area). Off-timeline
// backlog items are managed in the Explorer and must not appear in / dim the timeline.
const timelineItems = computed(() => store.milestones.filter(m => m.swimlaneId))

// ── Free-text query → highlight matches, dim the rest ────────────────────────
const query = ref('')
const qFocus = ref(false)
const tokens = computed(() => query.value.trim().toLowerCase().split(/\s+/).filter(Boolean))

function laneOf(m) { return store.swimlanes.find(s => s.id === m.swimlaneId) }
function iconOf(m) { return itemTypeByKey(m.typeKey || m.kind)?.icon || 'l:Diamond' }
function fillOf(m) { return itemTypeByKey(m.typeKey || m.kind)?.fill !== false }
function colorOf(m) { const st = itemStatus(m); return st ? statusColor(st) : '#8a8a8e' }
function dateOf(m) {
  if (m.startDate && m.endDate) return `${m.startDate} ${m.endDate}`
  return m.when || (m.year ? `${MONTHS[(m.month || 1) - 1]} ${m.year}` : '')
}
// Everything worth searching for one item, lower-cased into one string.
function haystack(m) {
  const lane = laneOf(m)
  const t = itemTypeByKey(m.typeKey || m.kind)
  const st = itemStatus(m)
  const parts = [m.title, lane?.name, lane?.subLanes?.find(s => s.id === m.subLaneId)?.name,
    t?.label, st?.label, m.assigneeId ? memberName(m.assigneeId) : '',
    m.maturity ? MATURITY_STAGES[m.maturity - 1] : '', dateOf(m)]
  for (const v of Object.values(m.data || {})) if (v != null) parts.push(Array.isArray(v) ? v.join(' ') : String(v))
  return parts.filter(Boolean).join(' ').toLowerCase()
}
const queryMatches = computed(() => {
  if (!tokens.value.length) return []
  return timelineItems.value.filter(m => { const h = haystack(m); return tokens.value.every(tk => h.includes(tk)) })
})
const queryIds = computed(() => tokens.value.length ? new Set(queryMatches.value.map(m => m.id)) : null)
const suggestions = computed(() => queryMatches.value.slice(0, 8))

function sugMeta(m) {
  const st = itemStatus(m)
  return [laneOf(m)?.name, st?.label].filter(Boolean).join(' · ')
}
function pick(m) { qFocus.value = false; emit('open-item', m) }

// The query drives the persistent highlight; hovering a facet chip temporarily
// overrides it and mouseleave restores the query result (see the template).
watch(queryIds, (ids) => { ui.highlightIds = ids })

const facets = computed(() => {
  const items = timelineItems.value
  const tlIds = new Set(items.map(m => m.id))
  if (dim.value === 'group') {
    return groups.list
      .map(g => ({ key: g.id, label: g.name, color: g.color, ids: new Set((g.itemIds || []).filter(id => tlIds.has(id))) }))
      .filter(f => f.ids.size)
  }
  if (dim.value === 'type') {
    const by = new Map()
    for (const m of items) { const k = m.typeKey || m.kind || 'milestone'; if (!by.has(k)) by.set(k, new Set()); by.get(k).add(m.id) }
    return itemTypes.list.filter(t => by.get(t.key)?.size).map(t => ({ key: t.key, label: t.label, color: t.color, ids: by.get(t.key) }))
  }
  if (dim.value === 'status') {
    const by = new Map()
    for (const m of items) { const st = itemStatus(m); if (!st) continue; if (!by.has(st.key)) by.set(st.key, { label: st.label, color: statusColor(st), ids: new Set() }); by.get(st.key).ids.add(m.id) }
    return [...by.entries()].map(([key, v]) => ({ key, label: v.label, color: v.color, ids: v.ids })).filter(f => f.ids.size)
  }
  if (dim.value === 'assignee') {
    return workspace.members
      .map(mb => ({ key: mb.userId, label: mb.username, color: '', ids: new Set(items.filter(m => m.assigneeId === mb.userId).map(m => m.id)) }))
      .filter(f => f.ids.size)
  }
  return MATURITY_STAGES
    .map((s, i) => ({ key: i + 1, label: s, color: '', ids: new Set(items.filter(m => m.maturity === i + 1).map(m => m.id)) }))
    .filter(f => f.ids.size)
})

// Don't leave the timeline dimmed if this dock unmounts (view switch).
onUnmounted(() => { ui.highlightIds = null })
</script>

<style scoped>
.facet-dock {
  position: fixed;
  bottom: 14px; left: 70px;
  display: flex; align-items: center; gap: 10px;
  max-width: 62vw;
  min-height: 40px; box-sizing: border-box;
  padding: 5px 12px 5px 8px;
  background: var(--clr-glass);
  backdrop-filter: blur(10px); -webkit-backdrop-filter: blur(10px);
  border: 1px solid var(--clr-border);
  border-radius: 100px;
  box-shadow: var(--sh-lg), 0 0 0 1px rgba(0,0,0,0.03);
  z-index: 50;
}

.fd-search { position: relative; display: flex; align-items: center; gap: 6px; flex-shrink: 0; padding-left: 4px; }
.fd-search-ico { color: var(--clr-text-3); flex-shrink: 0; }
.fd-query { width: 150px; border: none; background: none; outline: none; font-size: 12.5px; color: var(--clr-text); }
.fd-query::placeholder { color: var(--clr-text-3); }
.fd-qcount { font-size: 10px; font-weight: 700; color: var(--clr-accent); background: rgba(0,113,227,0.12); border-radius: 100px; padding: 1px 7px; }
.fd-clear { width: 18px; height: 18px; display: inline-flex; align-items: center; justify-content: center; font-size: 15px; line-height: 1; color: var(--clr-text-3); border-radius: 50%; background: none; }
.fd-clear:hover { background: var(--clr-surface-2); color: var(--clr-text); }

/* Type-ahead suggestions, opening upward above the dock. */
.fd-suggest {
  position: absolute; bottom: calc(100% + 10px); left: -8px;
  width: 300px; max-height: 320px; overflow-y: auto;
  background: var(--clr-surface); border: 1px solid var(--clr-border-light);
  border-radius: var(--r-md); box-shadow: var(--sh-lg); padding: 5px;
}
.fd-sug { display: flex; align-items: center; gap: 8px; width: 100%; text-align: left; padding: 6px 8px; border-radius: var(--r-sm); background: none; }
.fd-sug:hover { background: var(--clr-surface-2); }
.fd-sug-ico { flex-shrink: 0; }
.fd-sug-title { font-size: 13px; color: var(--clr-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; min-width: 0; }
.fd-sug-meta { margin-left: auto; flex-shrink: 0; font-size: 11px; color: var(--clr-text-3); white-space: nowrap; }
.fd-sug-empty { font-size: 12px; color: var(--clr-text-3); padding: 8px; }
.fd-sug-more { font-size: 11px; color: var(--clr-text-3); padding: 6px 8px 3px; }

.fd-sep { width: 1px; height: 20px; background: var(--clr-border); flex-shrink: 0; }

.fd-dim {
  flex-shrink: 0;
  border: 1px solid var(--clr-border); border-radius: 100px;
  padding: 4px 10px; font-size: 12px; font-weight: 600;
  color: var(--clr-text); background: var(--clr-bg);
}
.fd-dim:focus { outline: none; border-color: var(--clr-accent); }

.fd-values {
  display: flex; align-items: center; gap: 6px;
  overflow-x: auto; min-width: 0;
  scrollbar-width: thin; scrollbar-color: var(--clr-border) transparent;
}
.fd-values::-webkit-scrollbar { height: 5px; }
.fd-values::-webkit-scrollbar-thumb { background: var(--clr-border); border-radius: 3px; }

.fd-chip {
  display: inline-flex; align-items: center; gap: 6px; flex-shrink: 0;
  padding: 3px 8px; border-radius: 100px;
  font-size: 12px; color: var(--clr-text-2);
  cursor: default; transition: background 0.12s, color 0.12s;
}
.fd-chip:hover { background: var(--clr-surface-2); color: var(--clr-text); }
.fd-dot { width: 9px; height: 9px; border-radius: 50%; flex-shrink: 0; }
.fd-label { white-space: nowrap; max-width: 160px; overflow: hidden; text-overflow: ellipsis; }
.fd-count { font-size: 10px; color: var(--clr-text-3); background: var(--clr-bg); border-radius: 100px; padding: 0 6px; }
.fd-empty { font-size: 12px; color: var(--clr-text-3); white-space: nowrap; }
</style>
