<template>
  <!-- Inline "content" pane for a selected artifact (VS-Code editor-style). -->
  <article class="id">
    <header class="id-head">
      <div class="id-titlewrap">
        <MarkerIcon :shape="type?.icon || item.marker || 'l:Diamond'" :color="item.color || '#8a8a8e'" :size="20" :fill="type?.fill !== false" />
        <h1 class="id-title">{{ item.title }}</h1>
      </div>
      <button v-if="!readOnly" class="id-edit" @click="$emit('edit')">Edit</button>
    </header>

    <div class="id-chips">
      <span class="id-chip">{{ type?.label || item.typeKey || item.kind }}</span>
      <span v-if="status" class="id-status" :style="statusStyle">{{ status.label }}</span>
      <button v-if="!item.sourceSystem" class="id-hist-btn" @click="showHistory = !showHistory"><History :size="13" /> {{ showHistory ? 'Hide history' : 'History' }} · v{{ item.version || 1 }}</button>
      <button v-if="item.assigneeId" type="button" class="id-chip id-assignee" title="View profile" @click.stop="openProfile(memberById(item.assigneeId), $event)"><span class="id-av">{{ initials(item.assigneeId) }}</span>{{ memberName(item.assigneeId) }}</button>
    </div>

    <div v-if="!item.sourceSystem" class="id-attrib">
      <span v-if="item.createdBy">Added by <strong>{{ who(item.createdBy) }}</strong><span v-if="item.createdAt"> · {{ fmtStamp(item.createdAt) }}</span></span>
      <span v-if="item.updatedBy && (item.version || 1) > 1">Last edit by <strong>{{ who(item.updatedBy) }}</strong><span v-if="item.updatedAt"> · {{ fmtStamp(item.updatedAt) }}</span></span>
    </div>

    <dl class="id-meta">
      <template v-for="r in metaRows" :key="r.k">
        <dt>{{ r.k }}</dt><dd>{{ r.v }}</dd>
      </template>
    </dl>

    <section v-if="fieldRows.length" class="id-block">
      <h2 class="id-h2">Fields</h2>
      <dl class="id-meta">
        <template v-for="f in fieldRows" :key="f.k"><dt>{{ f.k }}</dt><dd :class="{ 'id-empty': !f.v }">{{ f.v || '—' }}</dd></template>
      </dl>
    </section>

    <section v-for="d in descriptions" :key="d.k" class="id-block">
      <h2 class="id-h2">{{ d.label }}</h2>
      <p class="id-text">{{ d.v }}</p>
    </section>

    <section v-if="showHistory" class="id-block">
      <h2 class="id-h2">Version history</h2>
      <ItemHistory :key="item.id" :item-id="item.id" />
    </section>
  </article>
</template>

<script setup>
import { ref, computed } from 'vue'
import { History } from 'lucide-vue-next'
import { itemTypes, MATURITY_STAGES, store, MONTHS, memberName, memberInitials, memberById, openProfile, itemStatus, toneColor } from '../stores/useAppStore.js'

const status = computed(() => itemStatus(props.item))
const statusStyle = computed(() => {
  if (!status.value) return {}
  const c = toneColor(status.value.tone)
  return { color: c, background: c + '22', borderColor: c + '66' }
})
import MarkerIcon from './MarkerIcon.vue'
import ItemHistory from './ItemHistory.vue'

const props = defineProps({ item: { type: Object, required: true }, readOnly: { type: Boolean, default: false } })
defineEmits(['edit'])

const showHistory = ref(false)
function who(id) { return id ? (memberName(id) || 'someone') : 'system' }
function fmtStamp(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  return isNaN(d) ? '' : d.toLocaleString('en-GB', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

const type = computed(() => itemTypes.list.find(t => t.key === (props.item.typeKey || props.item.kind || 'milestone')))
const laneName = computed(() => store.swimlanes.find(s => s.id === props.item.swimlaneId)?.name || '')
const dateStr = computed(() => {
  const it = props.item
  if (it.startDate && it.endDate) return `${it.startDate} → ${it.endDate}`
  return it.when || (it.year && it.swimlaneId ? `${MONTHS[(it.month || 1) - 1]} ${it.year}` : '')
})
function initials(id) { return memberInitials(id) }

const metaRows = computed(() => [
  { k: 'Area', v: laneName.value },
  { k: 'Date', v: dateStr.value },
  { k: 'Maturity', v: props.item.maturity ? MATURITY_STAGES[props.item.maturity - 1] : '' },
  { k: 'Progress', v: props.item.progress != null ? props.item.progress + '%' : '' },
].filter(r => r.v))

// Show every field the type defines — empty ones included, so it's clear which
// fields exist and which are still blank (rendered as "—" in the template).
const fieldDisplay = (v) => Array.isArray(v) ? v.join(', ') : (v == null ? '' : String(v))
const fieldRows = computed(() =>
  (type.value?.fields || []).map(f => ({ k: f.label || f.key, v: fieldDisplay(props.item.data?.[f.key]) })))

// "Who" is now the assignee (shown as a chip up top), so it's no longer a
// free-text description here.
const descriptions = computed(() => [
  { k: 'what', label: 'What', v: props.item.what },
  { k: 'why', label: 'Why', v: props.item.why },
  { k: 'how', label: 'How', v: props.item.how },
].filter(d => d.v && String(d.v).trim()))
</script>

<style scoped>
.id { max-width: 720px; padding: 28px 32px; }
.id-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }
.id-titlewrap { display: flex; align-items: center; gap: 10px; min-width: 0; }
.id-title { font-size: 22px; font-weight: 700; color: var(--clr-text); line-height: 1.25; }
.id-edit { flex-shrink: 0; background: var(--clr-accent); color: #fff; border-radius: var(--r-md); padding: 7px 16px; font-weight: 600; font-size: 13px; }
.id-edit:hover { background: var(--clr-accent-hover); }

.id-chips { display: flex; flex-wrap: wrap; gap: 8px; margin: 12px 0 4px; }
.id-chip { font-size: 12px; font-weight: 600; color: var(--clr-text-2); background: var(--clr-surface-2); border-radius: 100px; padding: 4px 11px; }
.id-status { font-size: 12px; font-weight: 700; border-radius: 100px; padding: 4px 11px; border: 1px solid transparent; }
.id-assignee { display: inline-flex; align-items: center; gap: 6px; border: none; cursor: pointer; transition: filter 0.15s; }
.id-assignee:hover { filter: brightness(0.95); }

.id-attrib { display: flex; flex-wrap: wrap; align-items: center; gap: 6px 16px; margin: 12px 0 2px; font-size: 12px; color: var(--clr-text-3); }
.id-attrib strong { color: var(--clr-text-2); font-weight: 600; }
.id-hist-btn { display: inline-flex; align-items: center; gap: 5px; font-size: 12px; font-weight: 600; color: var(--clr-accent); background: var(--clr-surface-2); border-radius: 100px; padding: 4px 11px; }
.id-hist-btn:hover { background: var(--clr-surface); }
.id-av { width: 18px; height: 18px; border-radius: 50%; background: var(--clr-accent); color: #fff; font-size: 10px; font-weight: 700; display: inline-flex; align-items: center; justify-content: center; }

.id-meta { display: grid; grid-template-columns: 110px 1fr; gap: 6px 14px; margin: 16px 0; }
.id-meta dt { font-size: 12px; color: var(--clr-text-3); font-weight: 600; }
.id-meta dd { font-size: 13px; color: var(--clr-text); }
.id-meta dd.id-empty { color: var(--clr-text-3); }

.id-block { margin-top: 20px; }
.id-h2 { font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-text-3); margin-bottom: 7px; }
.id-text { font-size: 14px; color: var(--clr-text); line-height: 1.6; white-space: pre-wrap; }
.id-link { font-size: 13px; color: var(--clr-accent); word-break: break-all; }
.id-link:hover { text-decoration: underline; }
</style>
