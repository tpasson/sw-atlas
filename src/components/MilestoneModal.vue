<template>
  <Teleport to="body">
    <Transition name="modal">
      <div class="backdrop">
        <Transition name="modal-panel" appear>
          <div class="panel">
            <!-- Header -->
            <div class="panel-header">
              <div class="panel-meta">
                <span v-if="swimlane" class="panel-badge" :style="{ background: swimlane.color }">
                  {{ swimlane.name }}
                </span>
                <span v-if="subLane" class="panel-sub">{{ subLane.name }}</span>
                <span v-if="mode === 'add'" class="panel-month">{{ displayMonth }}</span>
                <button v-if="mode === 'edit' && milestone && !milestone.sourceSystem" type="button" class="panel-ver" :class="{ on: tab === 'history' }" title="View version history" @click="tab = 'history'">v{{ milestone.version || 1 }} <History :size="11" /></button>
                <span v-if="readOnly" class="ro-badge"><Lock :size="11" :stroke-width="2.5" /> Read-only</span>
              </div>
              <div class="panel-actions-top">
                <button v-if="canPropose && !proposing" type="button" class="propose-act" @click="proposing = true">{{ mode === 'add' ? 'Propose new item' : 'Propose change' }}</button>
                <button v-if="mode === 'edit' && !readOnly" type="button" class="icon-act danger" title="Delete milestone" @click="remove">
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6M10 11v6M14 11v6"/>
                  </svg>
                </button>
                <button v-if="!readOnly || proposing" type="button" class="icon-act primary" :title="proposing ? 'Submit proposal' : (mode === 'edit' ? 'Save' : 'Create')" @click="submit">
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

            <div v-if="mode === 'edit' && milestone && !milestone.sourceSystem" class="panel-attrib">
              <span v-if="milestone.createdBy">Added by <strong>{{ who(milestone.createdBy) }}</strong><span v-if="milestone.createdAt"> · {{ fmtStamp(milestone.createdAt) }}</span></span>
              <span v-if="milestone.updatedBy && (milestone.version || 1) > 1">Last edit by <strong>{{ who(milestone.updatedBy) }}</strong><span v-if="milestone.updatedAt"> · {{ fmtStamp(milestone.updatedAt) }}</span></span>
            </div>

            <div v-if="proposing" class="propose-banner">
              <span class="pb-text">{{ mode === 'add' ? 'Proposing a new item' : 'Proposing a change' }} — the owner must approve it before it goes live.</span>
              <input v-model="proposeNote" class="pb-note" placeholder="Reason (optional)" />
            </div>

            <!-- Form -->
            <form class="panel-form" @submit.prevent="submit">
              <fieldset class="ms-group" :disabled="formLocked">
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

              <div v-if="isTimelineType" class="field">
                <label class="field-label">Area</label>
                <select class="field-input" :disabled="formLocked" v-model="form.swimlaneId">
                  <option value="">— No area (off-timeline) —</option>
                  <option v-for="sw in timelineLanes" :key="sw.id" :value="sw.id">{{ sw.name }}</option>
                </select>
              </div>

              <div v-if="isTimelineType && chosenLaneSubs.length" class="field">
                <label class="field-label">Sub-area</label>
                <select class="field-input" :disabled="formLocked" v-model="form.subLaneId">
                  <option value="">— Top of area —</option>
                  <option v-for="sub in chosenLaneSubs" :key="sub.id" :value="sub.id">{{ sub.name }}</option>
                </select>
              </div>

              <div class="field">
                <label class="field-label">Type</label>
                <div class="type-row">
                  <span class="type-ico"><MarkerIcon :shape="currentType?.icon || 'l:Diamond'" :color="currentType?.color || swimlane?.color || '#8a8a8e'" :size="18" :fill="currentType?.fill !== false" /></span>
                  <select class="field-input" :value="form.typeKey" :disabled="formLocked" @change="applyType($event.target.value)">
                    <option v-for="t in itemTypes.list" :key="t.key" :value="t.key">{{ t.label }}</option>
                  </select>
                </div>
                <p class="type-hint">The icon comes from the type — set it under Settings → Types.</p>
              </div>

              <!-- Workflow status: allowed values come from the type's status set. -->
              <div v-if="typeStatuses.length" class="field">
                <label class="field-label">Status</label>
                <div class="ms-status">
                  <span class="ms-status-dot" :style="{ background: toneColor(currentStatusTone) }"></span>
                  <select class="field-input" :disabled="formLocked" v-model="form.status">
                    <option v-for="s in selectableStatuses" :key="s.key" :value="s.key">{{ s.label }}</option>
                  </select>
                </div>
              </div>

              <!-- Type-specific fields: schema comes from the selected item type. -->
              <div v-if="currentTypeFields.length" class="field type-fields">
                <label class="field-label">Fields</label>
                <div v-for="f in currentTypeFields" :key="f.key" class="tf-row">
                  <label class="tf-label">{{ f.label || f.key }}<span v-if="f.required" class="tf-req" title="Required">*</span></label>
                  <select v-if="f.type === 'select'" class="field-input" :class="{ 'tf-invalid': invalidFields.includes(f.key) }" :disabled="formLocked" v-model="form.data[f.key]">
                    <option value="">—</option>
                    <option v-for="o in (f.options || [])" :key="o" :value="o">{{ o }}</option>
                  </select>
                  <div v-else-if="f.type === 'multiselect'" class="tf-checks" :class="{ 'tf-invalid': invalidFields.includes(f.key) }">
                    <label v-for="o in (f.options || [])" :key="o" class="tf-check">
                      <input type="checkbox" :disabled="formLocked" :checked="Array.isArray(form.data[f.key]) && form.data[f.key].includes(o)" @change="toggleMulti(f.key, o, $event.target.checked)" /> {{ o }}
                    </label>
                    <span v-if="!(f.options || []).length" class="tf-empty">No options defined.</span>
                  </div>
                  <input v-else-if="f.type === 'number'" type="number" class="field-input" :class="{ 'tf-invalid': invalidFields.includes(f.key) }" :disabled="formLocked" v-model="form.data[f.key]" />
                  <input v-else-if="f.type === 'date'" type="date" class="field-input" :class="{ 'tf-invalid': invalidFields.includes(f.key) }" :disabled="formLocked" v-model="form.data[f.key]" />
                  <select v-else-if="f.type === 'reference' && !f.refMulti" class="field-input" :class="{ 'tf-invalid': invalidFields.includes(f.key) }" :disabled="formLocked" v-model="form.data[f.key]">
                    <option value="">—</option>
                    <option v-for="r in refItems(f)" :key="r.id" :value="r.id">{{ r.title }}</option>
                  </select>
                  <div v-else-if="f.type === 'reference'" class="tf-checks" :class="{ 'tf-invalid': invalidFields.includes(f.key) }">
                    <label v-for="r in refItems(f)" :key="r.id" class="tf-check">
                      <input type="checkbox" :disabled="formLocked" :checked="Array.isArray(form.data[f.key]) && form.data[f.key].includes(r.id)" @change="toggleMulti(f.key, r.id, $event.target.checked)" /> {{ r.title }}
                    </label>
                    <span v-if="!refItems(f).length" class="tf-empty">No {{ refTypeLabel(f.refType) }} items to reference.</span>
                  </div>
                  <input v-else type="text" class="field-input" :class="{ 'tf-invalid': invalidFields.includes(f.key) }" :disabled="formLocked" v-model="form.data[f.key]" />
                  <div v-if="f.type === 'reference' && selectedRefs(f).length" class="tf-pins">
                    <span class="tf-pins-lbl" title="Pin a reference to the target's current version, or keep it tracking the latest">Version</span>
                    <button
                      v-for="id in selectedRefs(f)" :key="id" type="button"
                      class="tf-pin" :class="{ on: isPinned(f.key, id) }" :disabled="formLocked"
                      :title="isPinned(f.key, id) ? refTitle(id) + ' pinned to v' + pinnedVer(f.key, id) + ' — click to track latest' : 'Pin ' + refTitle(id) + ' to its current version'"
                      @click="togglePin(f.key, id)"
                    >{{ refTitle(id) }}: {{ isPinned(f.key, id) ? 'v' + pinnedVer(f.key, id) : 'latest' }}</button>
                  </div>
                </div>
              </div>


              <div class="field">
                <label class="field-label">
                  Maturity
                  <span v-if="form.maturity" class="mat-current">{{ MATURITY_STAGES[form.maturity - 1] }}</span>
                </label>
                <div class="maturity-row">
                  <button
                    type="button"
                    class="maturity-btn"
                    :class="{ on: !form.maturity }"
                    title="No maturity"
                    @click="form.maturity = null"
                  >
                    <MaturityGlyph :level="0" variant="grid" :color="!form.maturity ? '#0A84FF' : '#9aa0a6'" />
                    <span class="maturity-lbl">None</span>
                  </button>
                  <button
                    v-for="(s, i) in MATURITY_STAGES"
                    :key="s"
                    type="button"
                    class="maturity-btn"
                    :class="{ on: form.maturity === i + 1 }"
                    :title="s"
                    @click="form.maturity = i + 1"
                  >
                    <MaturityGlyph :level="i + 1" variant="grid" :color="form.maturity === i + 1 ? (form.color || swimlane?.color || '#0A84FF') : '#9aa0a6'" />
                    <span class="maturity-lbl">{{ s }}</span>
                  </button>
                </div>
              </div>

              <div class="field">
                <label class="field-label">
                  Progress
                  <span v-if="form.progress != null" class="mat-current">{{ form.progress }}%</span>
                </label>
                <div class="progress-row">
                  <button
                    type="button"
                    class="maturity-btn"
                    :class="{ on: form.progress == null }"
                    @click="form.progress = null"
                  >None</button>
                  <input
                    type="range" min="0" max="100" step="1"
                    class="progress-slider"
                    :value="form.progress ?? 0"
                    @input="form.progress = +$event.target.value"
                  />
                </div>
              </div>
              </fieldset>

              <div class="ms-tabs" role="tablist">
                <button type="button" class="ms-tab" :class="{ active: tab === 'details' }" @click="tab = 'details'">Details</button>
                <button type="button" class="ms-tab" :class="{ active: tab === 'deps' }" @click="tab = 'deps'">Dependencies</button>
                <button type="button" class="ms-tab" :class="{ active: tab === 'groups' }" @click="tab = 'groups'">Groups</button>
                <button v-if="mode === 'edit' && !milestone?.sourceSystem" type="button" class="ms-tab" :class="{ active: tab === 'history' }" @click="tab = 'history'">History</button>
              </div>

              <!-- The History tab is read-only display, so it's never form-disabled
                   (you can browse versions even when the rest is read-only). -->
              <fieldset class="ms-tab-body" :disabled="formLocked && tab !== 'history'">
              <div v-show="tab === 'history'" class="ms-panel ms-history">
                <ItemHistory v-if="milestone" :key="milestone.id" :item-id="milestone.id" />
              </div>
              <div v-show="tab === 'details'" class="ms-panel">
              <div class="two-col">
                <div class="field">
                  <label class="field-label">What</label>
                  <textarea v-model="form.what" class="field-textarea" :rows="readOnly ? 12 : 3" placeholder="What will be achieved?"></textarea>
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
                  <select class="field-input" :disabled="formLocked" v-model="form.assigneeId">
                    <option :value="null">Unassigned</option>
                    <option v-for="mb in workspace.members" :key="mb.userId" :value="mb.userId">{{ mb.username }}</option>
                  </select>
                </div>
              </div>

              <template v-if="isTimelineType">
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
                  <label class="field-label">When</label>
                  <input v-model="form.when" type="date" class="field-input field-date" />
                </div>
              </template>
              <p v-if="dateError" class="field-error">{{ dateError }}</p>
              </div>

              <div v-show="tab === 'deps'" class="ms-panel">
              <div class="field">
                <label class="field-label">Relationship</label>
                <select v-model="relType" class="field-input" :disabled="formLocked">
                  <option v-for="r in RELATIONSHIP_TYPES" :key="r.key" :value="r.key">{{ r.label }} ↔ {{ r.inverse }}</option>
                </select>
              </div>
              <!-- The two directions of the selected relationship (side by side) -->
              <div class="two-col dep-cols">
              <div class="field">
                <label class="field-label">
                  {{ relDef.label }}
                  <span v-if="localLinkedIds.size > 0" class="link-count link-toggle" :class="{ on: showOnly1 }" :title="showOnly1 ? 'Show all' : 'Show only selected'" @click.prevent.stop="showOnly1 = !showOnly1">{{ localLinkedIds.size }}</span>
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

              <!-- Required by (reverse dependency) -->
              <div class="field">
                <label class="field-label">
                  {{ relDef.inverse }}
                  <span v-if="localDependentIds.size > 0" class="link-count link-toggle" :class="{ on: showOnly2 }" :title="showOnly2 ? 'Show all' : 'Show only selected'" @click.prevent.stop="showOnly2 = !showOnly2">{{ localDependentIds.size }}</span>
                </label>
                <div class="ms-picker">
                  <div class="picker-search">
                    <svg width="13" height="13" viewBox="0 0 13 13" fill="none" class="search-icon">
                      <circle cx="5.5" cy="5.5" r="4" stroke="currentColor" stroke-width="1.5"/>
                      <path d="M9 9l2.5 2.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                    </svg>
                    <input
                      v-model="pickerSearch2"
                      class="picker-input"
                      placeholder="Search milestones…"
                      autocomplete="off"
                    />
                    <button
                      v-if="pickerSearch2"
                      type="button"
                      class="picker-clear"
                      @click="pickerSearch2 = ''"
                    >×</button>
                  </div>
                  <div class="picker-list">
                    <template v-for="group in pickerGroups2" :key="'rb-' + group.swimlane.id + '-' + (group.subLane?.id ?? 'root')">
                      <div class="picker-group-header">
                        <span class="picker-group-dot" :style="{ background: group.swimlane.color }"></span>
                        {{ group.swimlane.name }}{{ group.subLane ? ' · ' + group.subLane.name : '' }}
                      </div>
                      <button
                        v-for="m in group.milestones"
                        :key="m.id"
                        type="button"
                        class="picker-item"
                        :class="{ 'picker-active': localDependentIds.has(m.id) }"
                        :style="localDependentIds.has(m.id) ? activePickerStyle(group.swimlane.color) : {}"
                        @click="toggleLocalDependent(m.id)"
                      >
                        <span class="picker-dot" :style="{ background: group.swimlane.color }"></span>
                        <div class="picker-info">
                          <span class="picker-title">{{ m.title }}</span>
                          <span class="picker-meta">{{ MONTHS[m.month - 1] }} {{ m.year !== year ? m.year : '' }}</span>
                        </div>
                        <svg v-if="localDependentIds.has(m.id)" class="picker-check" width="14" height="14" viewBox="0 0 14 14" fill="none">
                          <path d="M2.5 7L5.5 10L11.5 4" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/>
                        </svg>
                      </button>
                    </template>
                    <div v-if="pickerGroups2.length === 0" class="picker-empty">
                      {{ pickerSearch2 ? 'No milestones match your search' : 'No other milestones yet' }}
                    </div>
                  </div>
                </div>
              </div>
              </div>
              </div>

              <div v-show="tab === 'groups'" class="ms-panel">
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
              </div>
              </fieldset>

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
import { useAppStore, MONTHS, MATURITY_STAGES, store, groups, swatchColors, stripMarkdown, itemTypes, itemTypeByKey, RELATIONSHIP_TYPES, workspace, session, baselines, canEditWorkspace, proposeChange, proposeCreate, memberName, STATUS_TONES, toneColor, parseRef } from '../stores/useAppStore.js'

function who(id) { return id ? (memberName(id) || 'someone') : 'system' }
function fmtStamp(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  return isNaN(d) ? '' : d.toLocaleString('en-GB', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}
import MaturityGlyph from './MaturityGlyph.vue'
import MarkerIcon from './MarkerIcon.vue'
import ItemHistory from './ItemHistory.vue'
import { Lock, History } from 'lucide-vue-next'

const props = defineProps({
  mode:      { type: String,  default: 'add' },
  swimlane:  { type: Object,  default: null },
  subLane:   { type: Object,  default: null },
  month:     { type: Number,  default: 1 },
  year:      { type: Number,  default: 2026 },
  date:      { type: String,  default: null },
  milestone: { type: Object,  default: null },
  initialType: { type: String, default: '' }, // preselect a type (Explorer "+ New")
  initialTab: { type: String, default: 'details' }, // open straight on a tab (e.g. "history")
  proposeMode: { type: Boolean, default: false }, // open straight in "propose a new item" mode
})

const emit = defineEmits(['close'])
const { addMilestone, updateMilestone, deleteMilestone, addLink, removeLink, itemGroupIds, setItemGroups } = useAppStore()

const TABS = ['details', 'deps', 'groups', 'history']
const tab = ref(props.mode === 'edit' && TABS.includes(props.initialTab) ? props.initialTab : 'details')
const invalidFields = ref([]) // keys of empty required fields, framed red
const isFieldEmpty = (v) => v == null || v === '' || (Array.isArray(v) && v.length === 0)

// The form is read-only when you can't edit content here: a source-synced item,
// a baseline (historical) view, or you're a viewer/non-member.
const readOnly = computed(() => !!props.milestone?.sourceSystem || !!baselines.activeId || !canEditWorkspace())

// Members who can't edit directly (or want to go through review) can PROPOSE a
// change to an existing item; the owner approves it. Not for synced/baseline items.
const proposing = ref(!!props.proposeMode)
const proposeNote = ref('')
const canPropose = computed(() =>
  (props.mode === 'edit' || props.mode === 'add') && session.authenticated && !!workspace.role &&
  !baselines.activeId && !props.milestone?.sourceSystem)
// Effective lock for the form fields: read-only unless we're actively proposing.
const formLocked = computed(() => readOnly.value && !proposing.value)

// Marker shapes offered in the picker = the active legend markers (+ the item's
// own marker if it was removed from the active set, so it stays selectable).
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
  swimlaneId: props.milestone?.swimlaneId ?? (props.swimlane?.id || ''),
  subLaneId: props.milestone?.subLaneId ?? (props.subLane?.id || ''),
  title:  props.milestone?.title ?? '',
  kind:   props.milestone?.kind ?? 'milestone',
  typeKey: props.milestone?.typeKey ?? (props.initialType || props.milestone?.kind || 'milestone'),
  data:   { ...(props.milestone?.data || {}) },
  assigneeId: props.milestone?.assigneeId ?? null,
  status: props.milestone?.status ?? '',
  what:   props.milestone?.sourceSystem ? stripMarkdown(props.milestone?.what || '') : (props.milestone?.what ?? ''),
  why:    props.milestone?.why   ?? '',
  how:    props.milestone?.how   ?? '',
  who:    props.milestone?.who   ?? '',
  when:   props.milestone?.when ?? defaultDate,
  startDate: props.milestone?.startDate ?? defaultDate,
  endDate:   props.milestone?.endDate ?? addDays(defaultDate, 7),
  color:  props.milestone?.color ?? null,
  maturity: props.milestone?.maturity ?? null,
  progress: props.milestone?.progress ?? null,
  scmUrl: props.milestone?.scmUrl ?? '',
})

// Reference fields can pin a target to a specific version, stored as "id@vN".
// Keep form.data on bare ids (so the pickers bind cleanly) and track pinned
// versions separately in refPins; both are re-encoded on save. Decode any pins
// already stored so editing an item preserves them.
const refPins = reactive({}) // { [fieldKey]: { [id]: version } }
for (const f of (itemTypeByKey(form.typeKey)?.fields || [])) {
  if (f.type !== 'reference') continue
  const raw = form.data[f.key]
  const entries = Array.isArray(raw) ? raw : (raw ? [raw] : [])
  const bare = []
  for (const e of entries) {
    const { id, version } = parseRef(e)
    if (!id) continue
    bare.push(id)
    if (version) (refPins[f.key] || (refPins[f.key] = {}))[id] = version
  }
  form.data[f.key] = f.refMulti ? bare : (bare[0] || '')
}

// Lanes you can place a new item / proposal in (mirrored Git lanes excluded).
const timelineLanes = computed(() => store.swimlanes.filter(s => !s.sourceSystem))
const chosenLaneSubs = computed(() => store.swimlanes.find(s => s.id === form.swimlaneId)?.subLanes || [])
// Changing the Area clears the sub-area selection (it belonged to the old lane).
watch(() => form.swimlaneId, () => { form.subLaneId = '' })
// Clear a required field's red frame as soon as it's filled in.
watch(() => form.data, () => {
  if (invalidFields.value.length) invalidFields.value = invalidFields.value.filter(k => isFieldEmpty(form.data[k]))
}, { deep: true })

// Type-specific field schema for the selected type.
const currentType = computed(() => itemTypeByKey(form.typeKey))
// timeline-family types sit on a lane/date; work-item & container types don't.
const isTimelineType = computed(() => {
  const f = currentType.value?.family
  return !f || f === 'timeline-point' || f === 'timeline-range'
})
const currentTypeFields = computed(() => currentType.value?.fields || [])
const currentTypeLabel = computed(() => currentType.value?.label || 'Type')
const typeStatuses = computed(() => currentType.value?.statuses || [])
const currentStatusTone = computed(() => (typeStatuses.value.find(s => s.key === form.status)?.tone) || 'neutral')
// While editing, offer only the current status plus its allowed transitions
// (empty transitions = any). Creating an item can start in any status.
const selectableStatuses = computed(() => {
  const all = typeStatuses.value
  if (!all.length) return []
  if (props.mode === 'add') return all
  const cur = all.find(s => s.key === form.status)
  if (!cur || !(cur.to && cur.to.length)) return all
  const allowed = new Set([cur.key, ...cur.to])
  return all.filter(s => allowed.has(s.key))
})

// Switching the item type derives its rendering kind, and (for custom types)
// seeds the marker/colour and any new field slots.
function applyType(key) {
  const t = itemTypeByKey(key)
  form.typeKey = key
  if (!t) return
  if (t.builtin) {
    form.kind = key === 'event' ? 'event' : key === 'point' ? 'point' : 'milestone'
  } else {
    form.kind = t.family === 'timeline-range' ? 'event' : 'milestone'
    if (t.color) form.color = t.color
  }
  // Off-timeline types (backlog / folder) never carry a lane.
  if (t.family === 'work-item' || t.family === 'container') { form.swimlaneId = ''; form.subLaneId = '' }
  for (const f of (t.fields || [])) {
    if (!(f.key in form.data)) form.data[f.key] = (f.type === 'multiselect' || (f.type === 'reference' && f.refMulti)) ? [] : ''
  }
  // Default to the type's start status if the current one isn't valid for this type.
  const sts = t.statuses || []
  if (!sts.length) form.status = ''
  else if (!sts.some(s => s.key === form.status)) form.status = sts[0].key
}

// Items a reference field can point at: every item of the field's target type,
// except this item itself.
function refItems(f) {
  if (!f.refType) return []
  return store.milestones.filter(m => (m.typeKey || m.kind) === f.refType && m.id !== props.milestone?.id)
}
function refTypeLabel(key) { return itemTypeByKey(key)?.label || key || 'referenced' }

// Version pinning for reference fields (see refPins above).
function selectedRefs(f) {
  const v = form.data[f.key]
  return Array.isArray(v) ? v.filter(Boolean) : (v ? [v] : [])
}
function refTitle(id) { return store.milestones.find(m => m.id === id)?.title || id }
function isPinned(key, id) { return !!(refPins[key] && refPins[key][id]) }
function pinnedVer(key, id) { return refPins[key] && refPins[key][id] }
// Toggle a reference between tracking latest (head) and pinned-to-current-version.
function togglePin(key, id) {
  if (formLocked.value) return
  if (isPinned(key, id)) delete refPins[key][id]
  else (refPins[key] || (refPins[key] = {}))[id] = store.milestones.find(m => m.id === id)?.version || 1
}
// Re-encode reference values with their pinned versions for the save payload.
function encodedData() {
  const out = { ...form.data }
  for (const f of (itemTypeByKey(form.typeKey)?.fields || [])) {
    if (f.type !== 'reference') continue
    const pins = refPins[f.key] || {}
    const enc = (id) => (id && pins[id] ? `${id}@v${pins[id]}` : id)
    const v = out[f.key]
    out[f.key] = Array.isArray(v) ? v.map(enc) : enc(v)
  }
  return out
}

// Toggle one option of a multi-select field on/off.
function toggleMulti(key, opt, checked) {
  const arr = Array.isArray(form.data[key]) ? [...form.data[key]] : []
  const i = arr.indexOf(opt)
  if (checked && i === -1) arr.push(opt)
  else if (!checked && i !== -1) arr.splice(i, 1)
  form.data[key] = arr
}

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

// Typed relationships (R1): one relationship kind is edited at a time. `edges`
// is the working copy of every link touching this item; the pickers operate on
// the selected rel and the diff is applied on save (preserving cancel).
const relType = ref('depends-on')
const relDef = computed(() => RELATIONSHIP_TYPES.find(r => r.key === relType.value) || RELATIONSHIP_TYPES[0])
const SELF = props.milestone?.id || '__NEW__'
const originalEdges = (props.milestone ? store.links.filter(l => l.a === SELF || l.b === SELF) : [])
  .map(l => ({ a: l.a, b: l.b, rel: l.rel || 'depends-on' }))
const edges = ref(originalEdges.map(e => ({ ...e })))

// Sets for the SELECTED relationship — drive the two pickers.
const localLinkedIds = computed(() => new Set(edges.value.filter(e => e.a === SELF && e.rel === relType.value).map(e => e.b)))
const localDependentIds = computed(() => new Set(edges.value.filter(e => e.b === SELF && e.rel === relType.value).map(e => e.a)))

// The two directions of a relationship are mutually exclusive for one pair
// (prevents A↔A cycles within the same rel).
function toggleLocalLink(id) {
  const rel = relType.value
  const had = edges.value.some(e => e.a === SELF && e.b === id && e.rel === rel)
  edges.value = edges.value.filter(e => !(e.rel === rel && ((e.a === SELF && e.b === id) || (e.a === id && e.b === SELF))))
  if (!had) edges.value.push({ a: SELF, b: id, rel })
}
function toggleLocalDependent(id) {
  const rel = relType.value
  const had = edges.value.some(e => e.a === id && e.b === SELF && e.rel === rel)
  edges.value = edges.value.filter(e => !(e.rel === rel && ((e.a === id && e.b === SELF) || (e.a === SELF && e.b === id))))
  if (!had) edges.value.push({ a: id, b: SELF, rel })
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
    borderLeft: `2px solid rgba(${r},${g},${b},0.5)`,
  }
}

// Milestone picker search + grouping
const pickerSearch = ref('')
const pickerSearch2 = ref('')

function buildPickerGroups(query, onlyIds) {
  const q = (query || '').toLowerCase()
  const groups = []
  for (const sw of store.swimlanes) {
    const subs = sw.subLanes.length ? sw.subLanes : [null]
    for (const sub of subs) {
      const mils = store.milestones.filter(m => {
        if (m.id === props.milestone?.id) return false
        if (m.swimlaneId !== sw.id) return false
        if (m.subLaneId !== (sub?.id ?? null)) return false
        if (onlyIds && !onlyIds.has(m.id)) return false
        if (q && !m.title.toLowerCase().includes(q)) return false
        return true
      })
      if (mils.length) groups.push({ swimlane: sw, subLane: sub, milestones: mils })
    }
  }
  return groups
}
// Clicking the count badge filters the picker to only the selected items.
const showOnly1 = ref(false)
const showOnly2 = ref(false)
const pickerGroups = computed(() => buildPickerGroups(pickerSearch.value, showOnly1.value && localLinkedIds.value.size ? localLinkedIds.value : null))
const pickerGroups2 = computed(() => buildPickerGroups(pickerSearch2.value, showOnly2.value && localDependentIds.value.size ? localDependentIds.value : null))

const titleInput = ref(null)
onMounted(() => {
  // Preselect the Explorer's chosen type (sets kind / marker / colour / fields).
  if (props.mode === 'add' && props.initialType) applyType(props.initialType)
  // A status-typed item always has a status — default to the start if unset.
  if (typeStatuses.value.length && !typeStatuses.value.some(s => s.key === form.status)) {
    form.status = typeStatuses.value[0].key
  }
  titleInput.value?.focus()
})

function syncLinks(msId) {
  // Resolve the placeholder self-id, then diff the working edges against the
  // originals (keyed a|b|rel) and apply add/remove for the differences.
  const resolve = (e) => ({ a: e.a === SELF ? msId : e.a, b: e.b === SELF ? msId : e.b, rel: e.rel })
  const key = (e) => `${e.a}|${e.b}|${e.rel}`
  const want = new Map(edges.value.map(resolve).map(e => [key(e), e]))
  const orig = new Map(originalEdges.map(e => [key(e), e]))
  for (const [k, e] of want) if (!orig.has(k)) addLink(e.a, e.b, e.rel)
  for (const [k, e] of orig) if (!want.has(k)) removeLink(e.a, e.b, e.rel)
}

function submit() {
  if (formLocked.value) return // view-only and not proposing
  if (dateError.value) { tab.value = 'details'; return } // surface the date error
  if (!form.title.trim()) return

  // Enforce mandatory type fields: frame the empty ones in red instead of a message.
  invalidFields.value = currentTypeFields.value.filter(f => f.required && isFieldEmpty(form.data[f.key])).map(f => f.key)
  if (invalidFields.value.length) { tab.value = 'details'; return }

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
    swimlaneId: form.swimlaneId || '', // "" = off-timeline artifact (no lane)
    subLaneId:  form.subLaneId || null,
    year,
    month,
    title:      form.title.trim(),
    what:       form.what,
    why:        form.why,
    how:        form.how,
    who:        form.who,
    kind:       form.kind,
    typeKey:    form.typeKey,
    data:       encodedData(),
    assigneeId: form.assigneeId || null,
    status:     form.status || '',
    marker:     null, // the icon now comes from the item's type, not a per-item marker
    when:       isEvent ? (form.startDate || null) : (form.when || null),
    startDate:  isEvent ? (form.startDate || null) : null,
    endDate:    isEvent ? (form.endDate || null) : null,
    color:      null, // per-item colour removed — icon inherits the area/type colour
    maturity:   form.maturity || null,
    progress:   form.progress,
    scmUrl:     form.scmUrl.trim() || null,
  }
  // Proposing → submit a change request instead of touching the live plan.
  if (proposing.value) {
    const done = props.mode === 'add'
      ? proposeCreate({ ...payload, id: crypto.randomUUID() }, proposeNote.value.trim())
      : proposeChange(props.milestone.id, payload, proposeNote.value.trim())
    done.catch(e => alert(e?.message || 'Could not submit the proposal'))
    emit('close')
    return
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
  max-width: 960px;
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
  margin-bottom: 14px;
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
.panel-attrib { display: flex; flex-wrap: wrap; gap: 3px 18px; padding: 8px 20px 4px; font-size: 12px; color: var(--clr-text-3); }
.panel-attrib strong { color: var(--clr-text-2); font-weight: 600; }
.panel-ver { display: inline-flex; align-items: center; gap: 4px; font-size: 11px; font-weight: 700; color: var(--clr-text-2); background: var(--clr-surface-2); border-radius: 100px; padding: 3px 10px; cursor: pointer; transition: background 0.12s, color 0.12s; }
.panel-ver:hover, .panel-ver.on { background: rgba(0,113,227,0.12); color: var(--clr-accent); }

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
.propose-act { font-size: 12px; font-weight: 600; color: var(--clr-accent); background: rgba(0,113,227,0.08); border-radius: 100px; padding: 6px 13px; }
.propose-act:hover { background: rgba(0,113,227,0.16); }

.propose-banner { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; padding: 9px 14px; margin: 0 0 4px; background: rgba(255,159,10,0.12); border: 1px solid rgba(255,159,10,0.3); border-radius: var(--r-md); }
.pb-text { font-size: 12px; font-weight: 600; color: #b7791f; }
.pb-note { flex: 1; min-width: 160px; border: 1px solid var(--clr-border); border-radius: var(--r-sm); padding: 6px 9px; font-size: 13px; color: var(--clr-text); background: var(--clr-bg); }
.pb-note:focus { outline: none; border-color: var(--clr-accent); }
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

.ms-tabs { display: flex; gap: 2px; border-bottom: 1px solid var(--clr-border-light); margin: 2px 0 0; }
.ms-tab {
  padding: 8px 14px; font-size: 13.5px; font-weight: 600;
  color: var(--clr-text-3); background: none;
  border-bottom: 2px solid transparent; margin-bottom: -1px;
  cursor: pointer; transition: color 0.12s, border-color 0.12s;
}
.ms-tab:hover { color: var(--clr-text-2); }
.ms-tab.active { color: var(--clr-accent); border-bottom-color: var(--clr-accent); }
.ms-tab-body { height: 320px; display: flex; flex-direction: column; } /* fixed so the modal is the same height on every tab */
/* fieldsets are only used to disable the whole form in read-only mode — strip their chrome */
.panel-form fieldset { border: 0; margin: 0; padding: 0; min-width: 0; }
.ms-group { display: flex; flex-direction: column; gap: 14px; }
/* read-only view: keep the disabled controls fully legible (no browser dimming) */
.panel-form fieldset:disabled :disabled { opacity: 1; cursor: default; }
.panel-form fieldset:disabled .field-input,
.panel-form fieldset:disabled .field-textarea,
.panel-form fieldset:disabled .field-date {
  color: var(--clr-text); -webkit-text-fill-color: var(--clr-text); background: var(--clr-bg);
}
.ro-badge { display: inline-flex; align-items: center; gap: 4px; font-size: 11px; font-weight: 600; color: var(--clr-text-3); }
.ms-panel { display: flex; flex-direction: column; gap: 14px; flex: 1; min-height: 0; overflow-y: auto; }
.scm-hint { font-size: 12.5px; color: var(--clr-text-3); line-height: 1.45; margin-top: -1px; }

/* Dependencies tab: let the two pickers grow to fill the tab height instead of leaving whitespace below */
.dep-cols { flex: 1; min-height: 0; grid-template-rows: minmax(0, 1fr); }
.dep-cols .field { min-height: 0; }
.dep-cols .ms-picker { display: flex; flex-direction: column; flex: 1; min-height: 0; }
.dep-cols .picker-list { max-height: none; flex: 1; min-height: 0; }

.field { display: flex; flex-direction: column; gap: 5px; min-width: 0; }

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
.link-toggle { cursor: pointer; transition: background 0.12s, color 0.12s, box-shadow 0.12s; }
/* Filter toggle: outline (hollow) while ALL are shown, filled while filtered to the selected ones */
.link-count.link-toggle { background: transparent; color: var(--clr-accent); box-shadow: inset 0 0 0 1.5px var(--clr-accent); }
.link-count.link-toggle.on { background: var(--clr-accent); color: #fff; box-shadow: none; }

.type-fields .tf-row { display: flex; align-items: center; gap: 10px; margin-top: 8px; }
.type-fields .tf-label { font-size: 12px; color: var(--clr-text-2); min-width: 120px; flex-shrink: 0; }
.type-fields .tf-row .field-input { flex: 1; }
.type-fields .tf-req { color: var(--clr-danger); margin-left: 2px; }
.type-fields .tf-checks { flex: 1; display: flex; flex-wrap: wrap; gap: 6px 14px; align-items: center; }
.type-fields .tf-check { display: inline-flex; align-items: center; gap: 4px; font-size: 13px; color: var(--clr-text); }
.type-fields .tf-empty { font-size: 12px; color: var(--clr-text-3); }
.type-fields .field-input.tf-invalid { border-color: var(--clr-danger); box-shadow: 0 0 0 2px rgba(255,59,48,0.18); }
.type-fields .tf-checks.tf-invalid { border: 1px solid var(--clr-danger); border-radius: var(--r-md); padding: 6px 8px; box-shadow: 0 0 0 2px rgba(255,59,48,0.18); }
.type-fields .tf-row { flex-wrap: wrap; } /* lets the version-pin strip drop to its own line */
.type-fields .tf-pins { flex-basis: 100%; display: flex; flex-wrap: wrap; align-items: center; gap: 6px; padding-left: 130px; }
.type-fields .tf-pins-lbl { font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.3px; color: var(--clr-text-3); }
.type-fields .tf-pin { font-size: 11px; font-weight: 600; color: var(--clr-text-2); background: var(--clr-bg); border: 1px solid var(--clr-border); border-radius: 100px; padding: 3px 10px; }
.type-fields .tf-pin.on { color: var(--clr-warning, #FF9F0A); border-color: var(--clr-warning, #FF9F0A); background: rgba(255,159,10,0.1); }
.type-fields .tf-pin:hover:not(:disabled) { border-color: var(--clr-text-3); }
.type-fields .tf-pin:disabled { opacity: 0.6; cursor: default; }
.ms-status { display: flex; align-items: center; gap: 8px; }
.ms-status-dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }

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

.type-row { display: flex; align-items: center; gap: 9px; }
.type-ico { flex-shrink: 0; display: inline-flex; }
.type-row .field-input { flex: 1; }
.type-hint { font-size: 11px; color: var(--clr-text-3); margin-top: 5px; }
.marker-row { display: flex; gap: 6px; }
.marker-btn {
  width: 34px; height: 34px;
  display: flex; align-items: center; justify-content: center;
  border: 1.5px solid var(--clr-border); border-radius: var(--r-md);
  background: var(--clr-bg); cursor: pointer; transition: border-color 0.12s, background 0.12s;
}
.marker-btn:hover { background: var(--clr-surface-2); }
.marker-btn.on { border-color: var(--clr-accent); box-shadow: 0 0 0 3px rgba(0,113,227,0.12); }

.color-row { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
.color-swatch {
  width: 22px; height: 22px; border-radius: 6px; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
  border: 2px solid transparent; cursor: pointer;
  color: #fff; font-size: 11px; font-weight: 700;
}
.color-swatch.selected { border-color: var(--clr-text); }
.color-custom {
  width: 30px; height: 24px; padding: 0; border: 1.5px solid var(--clr-border);
  border-radius: 6px; background: none; cursor: pointer;
}

.maturity-row { display: flex; flex-wrap: wrap; gap: 6px; }
.maturity-btn {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 6px 10px; border-radius: var(--r-md);
  border: 1.5px solid var(--clr-border); background: var(--clr-surface);
  font-size: 12.5px; color: var(--clr-text-2); cursor: pointer;
  transition: border-color 0.12s, background 0.12s, color 0.12s;
}
.maturity-btn.on { border-color: var(--clr-accent); color: var(--clr-text); background: rgba(0,113,227,0.06); }
.maturity-lbl { white-space: nowrap; }
.mat-current { font-size: 11px; font-weight: 600; color: var(--clr-accent); padding-left: 6px; }
.progress-row { display: flex; align-items: center; gap: 12px; }
.progress-slider { flex: 1; }

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
  background: none;
  border-left: 2px solid transparent;
}
.picker-item:hover { background: rgba(120,120,128,0.2); }

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
