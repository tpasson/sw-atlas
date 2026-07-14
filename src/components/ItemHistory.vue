<template>
  <!-- Version history for one artifact: pick a version on the left, see that
       snapshot's values on the right. Read-only. -->
  <div class="ih">
    <div class="ih-list">
      <div v-if="loading" class="ih-empty">Loading history…</div>
      <div v-else-if="!revisions.length" class="ih-empty">No history yet.</div>
      <button
        v-for="r in revisions"
        :key="r.version"
        type="button"
        class="ih-rev"
        :class="{ on: selected === r.version }"
        @click="select(r.version)"
      >
        <span class="ih-v">v{{ r.version }}</span>
        <span class="ih-who">{{ who(r.editedBy) }}</span>
        <span class="ih-at">{{ fmt(r.editedAt) }}</span>
      </button>
    </div>

    <div v-if="snapshot" class="ih-snap">
      <div class="ih-snap-head">Version {{ snapshot.version }}</div>
      <dl class="ih-fields">
        <template v-for="f in snapFields" :key="f.k"><dt>{{ f.k }}</dt><dd>{{ f.v }}</dd></template>
      </dl>
      <div v-for="d in snapDescriptions" :key="d.k" class="ih-desc">
        <span class="ih-desc-h">{{ d.label }}</span>
        <p>{{ d.v }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'
import { itemTypes, MATURITY_STAGES, MONTHS, memberName, store } from '../stores/useAppStore.js'

const props = defineProps({ itemId: { type: String, required: true } })

const revisions = ref([])
const loading = ref(true)
const selected = ref(null)
const snapshot = ref(null)

onMounted(async () => {
  try {
    revisions.value = (await api.listRevisions(props.itemId)) || []
    if (revisions.value.length) select(revisions.value[0].version)
  } catch { revisions.value = [] }
  loading.value = false
})

async function select(version) {
  selected.value = version
  snapshot.value = null
  try {
    const rev = await api.getRevision(props.itemId, version)
    snapshot.value = { version: rev.version, ...(rev.snapshot || {}) }
  } catch { snapshot.value = null }
}

function who(id) { return id ? (memberName(id) || 'someone') : 'system' }
function fmt(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  return isNaN(d) ? '' : d.toLocaleString('en-GB', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

const snapType = computed(() => snapshot.value && itemTypes.list.find(t => t.key === (snapshot.value.typeKey || snapshot.value.kind)))
const snapFields = computed(() => {
  const s = snapshot.value
  if (!s) return []
  const laneName = store.swimlanes.find(l => l.id === s.swimlaneId)?.name || ''
  const date = s.startDate && s.endDate ? `${s.startDate} → ${s.endDate}` : (s.when || (s.year ? `${MONTHS[(s.month || 1) - 1]} ${s.year}` : ''))
  const rows = [
    { k: 'Title', v: s.title },
    { k: 'Area', v: laneName },
    { k: 'Date', v: date },
    { k: 'Maturity', v: s.maturity ? MATURITY_STAGES[s.maturity - 1] : '' },
    { k: 'Progress', v: s.progress != null ? s.progress + '%' : '' },
    { k: 'Assignee', v: s.assigneeId ? memberName(s.assigneeId) : '' },
  ]
  for (const f of (snapType.value?.fields || [])) {
    const val = s.data?.[f.key]
    if (val != null && String(val).trim() !== '') rows.push({ k: f.label || f.key, v: val })
  }
  return rows.filter(r => r.v != null && String(r.v).trim() !== '')
})
const snapDescriptions = computed(() => {
  const s = snapshot.value
  if (!s) return []
  return [
    { k: 'what', label: 'What', v: s.what },
    { k: 'why', label: 'Why', v: s.why },
    { k: 'how', label: 'Where', v: s.how },
    { k: 'who', label: 'Who', v: s.who },
  ].filter(d => d.v && String(d.v).trim())
})
</script>

<style scoped>
.ih { display: flex; gap: 16px; }
.ih-list { width: 210px; flex-shrink: 0; display: flex; flex-direction: column; gap: 2px; max-height: 340px; overflow-y: auto; }
.ih-empty { font-size: 12px; color: var(--clr-text-3); padding: 6px 0; }
.ih-rev { display: grid; grid-template-columns: auto 1fr; column-gap: 8px; text-align: left; padding: 6px 9px; border-radius: 7px; background: none; }
.ih-rev:hover { background: var(--clr-surface-2); }
.ih-rev.on { background: rgba(0,113,227,0.1); }
.ih-v { grid-row: span 2; align-self: center; font-size: 11px; font-weight: 700; color: var(--clr-accent); }
.ih-who { font-size: 12px; color: var(--clr-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.ih-at { font-size: 10px; color: var(--clr-text-3); }
.ih-snap { flex: 1; min-width: 0; border-left: 1px solid var(--clr-border-light); padding-left: 16px; }
.ih-snap-head { font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-text-3); margin-bottom: 8px; }
.ih-fields { display: grid; grid-template-columns: 90px 1fr; gap: 5px 12px; }
.ih-fields dt { font-size: 12px; color: var(--clr-text-3); font-weight: 600; }
.ih-fields dd { font-size: 13px; color: var(--clr-text); }
.ih-desc { margin-top: 12px; }
.ih-desc-h { font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-text-3); }
.ih-desc p { font-size: 13px; color: var(--clr-text); line-height: 1.5; white-space: pre-wrap; margin-top: 3px; }
</style>
