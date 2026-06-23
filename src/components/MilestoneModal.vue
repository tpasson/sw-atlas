<template>
  <Teleport to="body">
    <Transition name="modal">
      <div class="backdrop" @click.self="$emit('close')">
        <Transition name="modal-panel" appear>
          <div class="panel">
            <!-- Header -->
            <div class="panel-header">
              <div class="panel-meta">
                <span class="panel-badge" :style="{ background: swimlane?.color }">
                  {{ swimlane?.name }}
                </span>
                <span v-if="subLane" class="panel-sub">{{ subLane.name }}</span>
                <span class="panel-month">{{ displayMonth }}</span>
              </div>
              <h2 class="panel-title">{{ mode === 'edit' ? 'Edit Milestone' : 'New Milestone' }}</h2>
              <div class="panel-actions-top">
                <button v-if="mode === 'edit'" type="button" class="icon-act danger" title="Delete milestone" @click="remove">
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6M10 11v6M14 11v6"/>
                  </svg>
                </button>
                <button type="button" class="icon-act primary" :title="mode === 'edit' ? 'Save' : 'Create'" @click="submit">
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                    <path d="M3 8.5L6.5 12L13 4.5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                  </svg>
                </button>
                <button type="button" class="icon-act" title="Close" @click="$emit('close')">
                  <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                    <path d="M1 1l12 12M13 1L1 13" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/>
                  </svg>
                </button>
              </div>
            </div>

            <!-- Form -->
            <form class="panel-form" @submit.prevent="submit">
              <div class="field">
                <label class="field-label">Title <span class="req">*</span></label>
                <input
                  v-model="form.title"
                  class="field-input"
                  placeholder="Short description of the milestone"
                  autocomplete="off"
                  required
                  ref="titleInput"
                />
              </div>

              <div class="two-col">
                <div class="field">
                  <label class="field-label">Type</label>
                  <div class="seg">
                    <button type="button" class="seg-btn" :class="{ on: form.kind === 'milestone' }" @click="form.kind = 'milestone'">Milestone</button>
                    <button type="button" class="seg-btn" :class="{ on: form.kind === 'event' }" @click="form.kind = 'event'">Event</button>
                  </div>
                </div>
                <div class="field">
                  <label class="field-label">
                    Marker
                    <button
                      v-if="form.kind === 'event'"
                      type="button"
                      class="mini-toggle"
                      :class="{ on: markerOn }"
                      @click="markerOn = !markerOn"
                    >{{ markerOn ? 'On' : 'Off' }}</button>
                  </label>
                  <div v-if="form.kind !== 'event' || markerOn" class="marker-row">
                    <button
                      v-for="o in markerOptions"
                      :key="o.shape"
                      type="button"
                      class="marker-btn"
                      :class="{ on: form.marker === o.shape }"
                      :title="o.shape"
                      @click="form.marker = o.shape"
                    >
                      <MarkerIcon :shape="o.shape" :fill="o.fill" :color="form.marker === o.shape ? (swimlane?.color || '#0A84FF') : '#9aa0a6'" :size="16" />
                    </button>
                  </div>
                  <span v-else class="field-hint">No marker on this event</span>
                </div>
              </div>

              <div class="two-col">
                <div class="field">
                  <label class="field-label">What</label>
                  <textarea v-model="form.what" class="field-textarea" rows="3" placeholder="What will be achieved?"></textarea>
                </div>
                <div class="field">
                  <label class="field-label">Why</label>
                  <textarea v-model="form.why" class="field-textarea" rows="3" placeholder="Why is this important?"></textarea>
                </div>
              </div>

              <div class="two-col">
                <div class="field">
                  <label class="field-label">Where</label>
                  <textarea v-model="form.how" class="field-textarea" rows="3" placeholder="Where will it take place?"></textarea>
                </div>
                <div class="field">
                  <label class="field-label">Who</label>
                  <input v-model="form.who" class="field-input" placeholder="Responsible person / team"/>
                </div>
              </div>

              <div v-if="form.kind === 'event'" class="two-col">
                <div class="field">
                  <label class="field-label">Start <span class="req">*</span></label>
                  <input v-model="form.startDate" type="date" class="field-input field-date" />
                </div>
                <div class="field">
                  <label class="field-label">End</label>
                  <input v-model="form.endDate" type="date" class="field-input field-date" :min="form.startDate" />
                </div>
              </div>
              <div v-else class="field">
                <label class="field-label">Date</label>
                <input v-model="form.when" type="date" class="field-input field-date" />
              </div>
              <p v-if="dateError" class="field-error">{{ dateError }}</p>

              <!-- Milestone Links -->
              <div class="field">
                <label class="field-label">
                  Depends on
                  <span v-if="localLinkedIds.size > 0" class="link-count">{{ localLinkedIds.size }}</span>
                </label>
                <div class="ms-picker">
                  <div class="picker-search">
                    <svg width="13" height="13" viewBox="0 0 13 13" fill="none" class="search-icon">
                      <circle cx="5.5" cy="5.5" r="4" stroke="currentColor" stroke-width="1.5"/>
                      <path d="M9 9l2.5 2.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                    </svg>
                    <input
                      v-model="pickerSearch"
                      class="picker-input"
                      placeholder="Search milestones…"
                      autocomplete="off"
                    />
                    <button
                      v-if="pickerSearch"
                      type="button"
                      class="picker-clear"
                      @click="pickerSearch = ''"
                    >×</button>
                  </div>
                  <div class="picker-list">
                    <template v-for="group in pickerGroups" :key="group.swimlane.id + '-' + (group.subLane?.id ?? 'root')">
                      <div class="picker-group-header">
                        <span class="picker-group-dot" :style="{ background: group.swimlane.color }"></span>
                        {{ group.swimlane.name }}{{ group.subLane ? ' · ' + group.subLane.name : '' }}
                      </div>
                      <button
                        v-for="m in group.milestones"
                        :key="m.id"
                        type="button"
                        class="picker-item"
                        :class="{ 'picker-active': localLinkedIds.has(m.id) }"
                        :style="localLinkedIds.has(m.id) ? activePickerStyle(group.swimlane.color) : {}"
                        @click="toggleLocalLink(m.id)"
                      >
                        <span class="picker-dot" :style="{ background: group.swimlane.color }"></span>
                        <div class="picker-info">
                          <span class="picker-title">{{ m.title }}</span>
                          <span class="picker-meta">{{ MONTHS[m.month - 1] }} {{ m.year !== year ? m.year : '' }}</span>
                        </div>
                        <svg v-if="localLinkedIds.has(m.id)" class="picker-check" width="14" height="14" viewBox="0 0 14 14" fill="none">
                          <path d="M2.5 7L5.5 10L11.5 4" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/>
                        </svg>
                      </button>
                    </template>
                    <div v-if="pickerGroups.length === 0" class="picker-empty">
                      {{ pickerSearch ? 'No milestones match your search' : 'No other milestones yet' }}
                    </div>
                  </div>
                </div>
              </div>

              <!-- Group membership -->
              <div v-if="groups.list.length" class="field">
                <label class="field-label">
                  Groups
                  <span v-if="localGroupIds.size > 0" class="link-count">{{ localGroupIds.size }}</span>
                </label>
                <div class="grp-chips">
                  <button
                    v-for="g in groups.list"
                    :key="g.id"
                    type="button"
                    class="grp-chip"
                    :class="{ on: localGroupIds.has(g.id) }"
                    @click="toggleLocalGroup(g.id)"
                  >
                    <span class="grp-dot" :style="{ background: g.color }"></span>{{ g.name }}
                  </button>
                </div>
              </div>

              <!-- Enter-to-save (actions live in the header) -->
              <button type="submit" class="hidden-submit" tabindex="-1" aria-hidden="true"></button>
            </form>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { reactive, ref, computed, onMounted, watch } from 'vue'
import { useAppStore, MONTHS, store, groups, settings } from '../stores/useAppStore.js'
import MarkerIcon from './MarkerIcon.vue'

const props = defineProps({
  mode:      { type: String,  default: 'add' },
  swimlane:  { type: Object,  default: null },
  subLane:   { type: Object,  default: null },
  month:     { type: Number,  default: 1 },
  year:      { type: Number,  default: 2026 },
  date:      { type: String,  default: null },
  milestone: { type: Object,  default: null },
})

const emit = defineEmits(['close'])
const { addMilestone, updateMilestone, deleteMilestone, addLink, removeLink, dependsOnIds, itemGroupIds, setItemGroups } = useAppStore()

// Marker shapes offered in the picker = the active legend markers (+ the item's
// own marker if it was removed from the active set, so it stays selectable).
const markerOptions = computed(() => {
  const opts = settings.markers.map(m => ({ shape: m.shape, fill: !!m.fill }))
  const cur = props.milestone?.marker
  if (cur && cur !== 'bar' && !opts.some(o => o.shape === cur)) opts.push({ shape: cur, fill: false })
  return opts
})

// Events have an optional marker (off by default); milestones always have one.
const markerOn = ref(!!(props.milestone?.marker && props.milestone.marker !== 'bar'))

const defaultDate = props.date || `${props.year}-${String(props.month).padStart(2,'0')}-01`

function addDays(dateStr, n) {
  if (!dateStr) return ''
  const [y, m, d] = dateStr.split('-').map(Number)
  const dt = new Date(y, m - 1, d + n)
  const mm = String(dt.getMonth() + 1).padStart(2, '0')
  const dd = String(dt.getDate()).padStart(2, '0')
  return `${dt.getFullYear()}-${mm}-${dd}`
}

const displayMonth = computed(() => {
  const base = form.kind === 'event' ? form.startDate : form.when
  if (!base) return `${MONTHS[props.month - 1]} ${props.year}`
  const [y, m] = base.split('-').map(Number)
  return `${MONTHS[m - 1]} ${y}`
})

const form = reactive({
  title:  props.milestone?.title ?? '',
  kind:   props.milestone?.kind ?? 'milestone',
  marker: props.milestone?.marker && props.milestone.marker !== 'bar' ? props.milestone.marker : (settings.markers[0]?.shape || 'l:Flag'),
  what:   props.milestone?.what  ?? '',
  why:    props.milestone?.why   ?? '',
  how:    props.milestone?.how   ?? '',
  who:    props.milestone?.who   ?? '',
  when:   props.milestone?.when ?? defaultDate,
  startDate: props.milestone?.startDate ?? defaultDate,
  endDate:   props.milestone?.endDate ?? addDays(defaultDate, 7),
})

// Keep an event's end date on/after its start so the picker opens in the right
// month instead of defaulting to today/a past date.
watch(() => form.startDate, (s) => {
  if (form.kind !== 'event' || !s) return
  if (!form.endDate || form.endDate < s) form.endDate = addDays(s, 7)
})

const dateError = computed(() => {
  if (form.kind === 'event' && form.startDate && form.endDate && form.endDate < form.startDate) {
    return 'End date must be on or after the start date'
  }
  return ''
})

// Local link state — Set of milestone IDs linked to this milestone
const localLinkedIds = ref(new Set(
  props.milestone ? dependsOnIds(props.milestone.id) : []
))

function toggleLocalLink(id) {
  const next = new Set(localLinkedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  localLinkedIds.value = next
}

// Group membership (applied on save).
const localGroupIds = ref(new Set(props.milestone ? itemGroupIds(props.milestone.id) : []))
function toggleLocalGroup(id) {
  const next = new Set(localGroupIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  localGroupIds.value = next
}

function activePickerStyle(color) {
  if (!color) return {}
  const r = parseInt(color.slice(1, 3), 16)
  const g = parseInt(color.slice(3, 5), 16)
  const b = parseInt(color.slice(5, 7), 16)
  return {
    background: `rgba(${r},${g},${b},0.08)`,
    borderLeft: `2px solid rgba(${r},${g},${b},0.5)`,
  }
}

// Milestone picker search + grouping
const pickerSearch = ref('')

const pickerGroups = computed(() => {
  const q = pickerSearch.value.toLowerCase()
  const groups = []
  for (const sw of store.swimlanes) {
    const subs = sw.subLanes.length ? sw.subLanes : [null]
    for (const sub of subs) {
      const mils = store.milestones.filter(m => {
        if (m.id === props.milestone?.id) return false
        if (m.swimlaneId !== sw.id) return false
        if (m.subLaneId !== (sub?.id ?? null)) return false
        if (q && !m.title.toLowerCase().includes(q)) return false
        return true
      })
      if (mils.length) groups.push({ swimlane: sw, subLane: sub, milestones: mils })
    }
  }
  return groups
})

const titleInput = ref(null)
onMounted(() => titleInput.value?.focus())

function syncLinks(msId) {
  const current = dependsOnIds(msId)
  for (const id of localLinkedIds.value) {
    if (!current.has(id)) addLink(msId, id)
  }
  for (const id of current) {
    if (!localLinkedIds.value.has(id)) removeLink(msId, id)
  }
}

function submit() {
  if (!form.title.trim() || dateError.value) return

  const isEvent = form.kind === 'event'
  // Grid position derives from the start (event) or the date (milestone).
  const base = isEvent ? (form.startDate || form.when) : form.when
  let month = props.month
  let year  = props.year
  if (base) {
    const parts = base.split('-')
    year  = parseInt(parts[0], 10)
    month = parseInt(parts[1], 10)
  }

  const payload = {
    swimlaneId: props.swimlane?.id,
    subLaneId:  props.subLane?.id ?? null,
    year,
    month,
    title:      form.title.trim(),
    what:       form.what,
    why:        form.why,
    how:        form.how,
    who:        form.who,
    kind:       form.kind,
    marker:     (form.kind === 'event' && !markerOn.value) ? null : form.marker,
    when:       isEvent ? (form.startDate || null) : (form.when || null),
    startDate:  isEvent ? (form.startDate || null) : null,
    endDate:    isEvent ? (form.endDate || null) : null,
  }
  if (props.mode === 'edit') {
    updateMilestone(props.milestone.id, payload)
    syncLinks(props.milestone.id)
    setItemGroups(props.milestone.id, [...localGroupIds.value])
  } else {
    const newMs = addMilestone(payload)
    syncLinks(newMs.id)
    setItemGroups(newMs.id, [...localGroupIds.value])
  }
  emit('close')
}

function remove() {
  if (!props.milestone) return
  const label = form.title.trim() || props.milestone.title || 'this item'
  if (!confirm(`Delete "${label}"? This can't be undone.`)) return
  deleteMilestone(props.milestone.id)
  emit('close')
}
</script>

<style scoped>
.backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.45);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.panel {
  background: var(--clr-surface);
  border-radius: var(--r-xl);
  width: 100%;
  max-width: 600px;
  max-height: 92vh;
  box-shadow: var(--sh-modal);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-header {
  padding: 20px 20px 0;
  position: relative;
  flex-shrink: 0;
}

.panel-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  flex-wrap: wrap;
  padding-right: 110px; /* keep clear of the top-right action icons */
}

.panel-badge {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 9px;
  border-radius: 100px;
  color: #fff;
  letter-spacing: 0.2px;
}

.panel-sub { font-size: 12px; color: var(--clr-text-2); font-weight: 500; }
.panel-month { font-size: 12px; color: var(--clr-text-3); }

.panel-title {
  font-size: 20px;
  font-weight: 700;
  letter-spacing: -0.4px;
  color: var(--clr-text);
  margin-bottom: 20px;
}

.panel-actions-top {
  position: absolute;
  top: 16px; right: 16px;
  display: flex; gap: 8px;
}
.icon-act {
  width: 32px; height: 32px;
  display: flex; align-items: center; justify-content: center;
  background: var(--clr-surface-2);
  border-radius: 50%;
  color: var(--clr-text-2);
  transition: background 0.15s, color 0.15s;
}
.icon-act:hover { background: var(--clr-border-light); color: var(--clr-text); }
.icon-act.primary { background: var(--clr-accent); color: #fff; }
.icon-act.primary:hover { background: var(--clr-accent-hover); }
.icon-act.danger { background: rgba(255,59,48,0.1); color: var(--clr-danger); }
.icon-act.danger:hover { background: rgba(255,59,48,0.18); }

.hidden-submit { position: absolute; width: 0; height: 0; padding: 0; margin: 0; border: 0; opacity: 0; pointer-events: none; }

.panel-form {
  padding: 0 20px 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  overflow-y: auto;
}

.two-col { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }

.field { display: flex; flex-direction: column; gap: 5px; }

.field-label {
  font-size: 11.5px; font-weight: 600;
  color: var(--clr-text-2);
  text-transform: uppercase; letter-spacing: 0.4px;
  display: flex; align-items: center; gap: 6px;
}
.req { color: var(--clr-danger); }

.link-count {
  font-size: 10px; font-weight: 700;
  background: var(--clr-accent);
  color: #fff;
  padding: 1px 6px;
  border-radius: 100px;
  letter-spacing: 0;
}

.field-input,
.field-textarea {
  border: 1.5px solid var(--clr-border);
  border-radius: var(--r-md);
  padding: 9px 12px;
  font-size: 14px;
  color: var(--clr-text);
  background: var(--clr-bg);
  transition: border-color 0.15s, box-shadow 0.15s;
  resize: none;
  outline: none;
  width: 100%;
}
.field-input:focus,
.field-textarea:focus {
  border-color: var(--clr-accent);
  box-shadow: 0 0 0 3px rgba(0,113,227,0.12);
  background: var(--clr-surface);
}
.field-input::placeholder,
.field-textarea::placeholder { color: var(--clr-text-3); }

.field-date { cursor: pointer; }

/* Type segmented control + marker picker (P1) */
.seg { display: flex; gap: 0; border: 1.5px solid var(--clr-border); border-radius: var(--r-md); overflow: hidden; }
.seg-btn {
  flex: 1; padding: 8px 10px; font-size: 13px; font-weight: 500;
  color: var(--clr-text-2); background: var(--clr-bg); transition: background 0.12s, color 0.12s;
}
.seg-btn + .seg-btn { border-left: 1.5px solid var(--clr-border); }
.seg-btn.on { background: var(--clr-accent); color: #fff; }

.marker-row { display: flex; gap: 6px; }
.marker-btn {
  width: 34px; height: 34px;
  display: flex; align-items: center; justify-content: center;
  border: 1.5px solid var(--clr-border); border-radius: var(--r-md);
  background: var(--clr-bg); cursor: pointer; transition: border-color 0.12s, background 0.12s;
}
.marker-btn:hover { background: var(--clr-surface-2); }
.marker-btn.on { border-color: var(--clr-accent); box-shadow: 0 0 0 3px rgba(0,113,227,0.12); }

.mini-toggle {
  margin-left: auto;
  font-size: 10px; font-weight: 700; letter-spacing: 0.3px;
  padding: 2px 9px; border-radius: 100px;
  border: 1px solid var(--clr-border);
  color: var(--clr-text-3); background: var(--clr-bg);
}
.mini-toggle.on { background: var(--clr-accent); color: #fff; border-color: var(--clr-accent); }
.field-hint { font-size: 12.5px; color: var(--clr-text-3); padding: 7px 2px; }

.field-error { font-size: 12.5px; color: var(--clr-danger); margin: -6px 0 0; }

/* ── Milestone Picker ───────────────────────────────────────────────── */
.ms-picker {
  border: 1.5px solid var(--clr-border);
  border-radius: var(--r-md);
  overflow: hidden;
  background: var(--clr-bg);
}

.picker-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--clr-border-light);
}

.search-icon { color: var(--clr-text-3); flex-shrink: 0; }

.picker-input {
  flex: 1;
  border: none;
  background: transparent;
  outline: none;
  font-size: 13px;
  color: var(--clr-text);
  min-width: 0;
}
.picker-input::placeholder { color: var(--clr-text-3); }

.picker-clear {
  font-size: 16px;
  color: var(--clr-text-3);
  line-height: 1;
  padding: 0 2px;
  transition: color 0.1s;
}
.picker-clear:hover { color: var(--clr-text); }

.picker-list {
  max-height: 210px;
  overflow-y: auto;
}

.picker-group-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 12px 4px;
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--clr-text-3);
  position: sticky;
  top: 0;
  background: var(--clr-bg);
  z-index: 1;
}

.picker-group-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.picker-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 7px 12px;
  cursor: pointer;
  transition: background 0.12s;
  text-align: left;
  border-left: 2px solid transparent;
}
.picker-item:hover { background: var(--clr-surface-2); }

.picker-dot {
  width: 7px; height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}

.picker-info {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.picker-title {
  font-size: 13px;
  color: var(--clr-text);
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.picker-meta {
  font-size: 11px;
  color: var(--clr-text-3);
  white-space: nowrap;
  flex-shrink: 0;
}

.picker-check { color: var(--clr-accent); flex-shrink: 0; }

.picker-empty {
  padding: 20px;
  text-align: center;
  font-size: 13px;
  color: var(--clr-text-3);
}

/* ── Group chips ─────────────────────────────────────────────────────── */
.grp-chips { display: flex; flex-wrap: wrap; gap: 8px; }
.grp-chip {
  display: inline-flex; align-items: center; gap: 7px;
  padding: 6px 12px;
  font-size: 13px; color: var(--clr-text-2);
  background: var(--clr-bg);
  border: 1.5px solid var(--clr-border);
  border-radius: 100px;
  transition: border-color 0.12s, background 0.12s, color 0.12s;
}
.grp-chip:hover { background: var(--clr-surface-2); }
.grp-chip.on { border-color: var(--clr-accent); background: rgba(0,113,227,0.08); color: var(--clr-text); }
.grp-dot { width: 9px; height: 9px; border-radius: 50%; flex-shrink: 0; }

/* ── Actions ─────────────────────────────────────────────────────────── */
.panel-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 4px;
  padding-top: 16px;
  border-top: 1px solid var(--clr-border-light);
  flex-shrink: 0;
}

.actions-right { display: flex; gap: 8px; margin-left: auto; }

.btn-primary {
  padding: 9px 20px;
  font-size: 14px; font-weight: 600;
  color: #fff;
  background: var(--clr-accent);
  border-radius: var(--r-md);
  transition: background 0.15s, transform 0.1s;
}
.btn-primary:hover { background: var(--clr-accent-hover); }
.btn-primary:active { transform: scale(0.98); }

.btn-ghost {
  padding: 9px 16px;
  font-size: 14px; font-weight: 500;
  color: var(--clr-text-2);
  background: transparent;
  border-radius: var(--r-md);
  transition: background 0.15s;
}
.btn-ghost:hover { background: var(--clr-surface-2); }

.btn-danger {
  display: flex; align-items: center; gap: 6px;
  padding: 9px 14px;
  font-size: 13px; font-weight: 500;
  color: var(--clr-danger);
  background: rgba(255,59,48,0.07);
  border-radius: var(--r-md);
  transition: background 0.15s;
}
.btn-danger:hover { background: rgba(255,59,48,0.14); }
</style>
