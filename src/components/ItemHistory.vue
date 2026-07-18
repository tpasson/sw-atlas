<template>
  <!-- Version history for one artifact. Picking a version loads that snapshot into
       the whole item form (read-only), so you see the version in the item itself —
       no separate field panel. Each card shows who/when + what changed. -->
  <div class="ih">
    <div v-if="loading" class="ih-empty">Loading history…</div>
    <div v-else-if="!rows.length" class="ih-empty">No history yet.</div>

    <div
      v-for="r in rows"
      :key="r.version"
      role="button"
      tabindex="0"
      class="ih-rev"
      :class="{ on: selected === r.version }"
      @click="select(r.version)"
      @keydown.enter="select(r.version)"
    >
      <span class="ih-v">v{{ r.version }}</span>
      <span class="ih-dot" :style="{ background: r.statusColor || 'var(--clr-text-3)' }"></span>
      <span class="ih-status">{{ r.statusLabel || '—' }}</span>
      <span v-if="r.isHead" class="ih-badge">Latest</span>
      <span class="ih-changes" :class="{ created: r.created }">{{ r.summary }}</span>
      <span class="ih-meta">{{ who(r.editedBy) }} · <span :title="fmtAbs(r.editedAt)">{{ rel(r.editedAt) }}</span></span>
      <button type="button" class="ih-copy" :class="{ done: copiedV === r.version }" :title="copiedV === r.version ? 'Copied' : 'Copy link to this version'" @click.stop="copyLink(r.version, r.isHead)"><Check v-if="copiedV === r.version" :size="13" :stroke-width="2.5" /><Link2 v-else :size="13" /></button>
      <span v-if="selected === r.version" class="ih-view">Viewing</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { api } from '../api.js'
import { Link2, Check } from 'lucide-vue-next'
import { itemTypes, MATURITY_STAGES, MONTHS, memberName, store, statusColor, itemLink } from '../stores/useAppStore.js'

const props = defineProps({
  itemId: { type: String, required: true },
  currentVersion: { type: Number, default: 0 }, // highlight the version shown in the form
})
const emit = defineEmits(['select'])

const revisions = ref([])
const snaps = ref({})       // version → snapshot
const loading = ref(true)
const selected = ref(null)

onMounted(async () => {
  try {
    revisions.value = (await api.listRevisions(props.itemId)) || []
    // Fetch every snapshot once so the list can show status + a change summary,
    // and so selecting a version is instant.
    const got = await Promise.all(revisions.value.map(r =>
      api.getRevision(props.itemId, r.version).then(rev => rev?.snapshot || null).catch(() => null)))
    const map = {}
    revisions.value.forEach((r, i) => { map[r.version] = got[i] })
    snaps.value = map
    if (revisions.value.length) selected.value = props.currentVersion || revisions.value[0].version
  } catch { revisions.value = [] }
  loading.value = false
})

// Keep the highlight in sync if the form is put back to another version elsewhere.
watch(() => props.currentVersion, (v) => { if (v) selected.value = v })

function select(version) {
  selected.value = version
  emit('select', version, snaps.value[version] || null)
}

// Copy a deep link to this specific version (the head version links to "latest").
const copiedV = ref(null)
let copyTimer = null
async function copyLink(version, isHead) {
  const url = itemLink(props.itemId, isHead ? null : version)
  try { await navigator.clipboard.writeText(url) }
  catch {
    const ta = document.createElement('textarea')
    ta.value = url; ta.style.position = 'fixed'; ta.style.opacity = '0'
    document.body.appendChild(ta); ta.select()
    try { document.execCommand('copy') } catch { /* ignore */ }
    document.body.removeChild(ta)
  }
  copiedV.value = version
  clearTimeout(copyTimer)
  copyTimer = setTimeout(() => { copiedV.value = null }, 1400)
}

function who(id) { return id ? (memberName(id) || 'someone') : 'system' }
function fmtAbs(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  return isNaN(d) ? '' : d.toLocaleString('en-GB', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}
function rel(iso) {
  if (!iso) return ''
  const d = new Date(iso); if (isNaN(d)) return ''
  const s = Math.max(0, Math.floor((Date.now() - d.getTime()) / 1000))
  if (s < 60) return 'just now'
  const m = Math.floor(s / 60); if (m < 60) return `${m} min ago`
  const h = Math.floor(m / 60); if (h < 24) return `${h} h ago`
  const days = Math.floor(h / 24); if (days < 30) return `${days} d ago`
  return fmtAbs(iso)
}

function typeOf(s) { return s && itemTypes.list.find(t => t.key === (s.typeKey || s.kind)) }
function statusLabelOf(s) {
  if (!s) return ''
  return (typeOf(s)?.statuses || []).find(st => st.key === s.status)?.label || s.status || ''
}
function statusColorOf(s) {
  if (!s || !s.status) return null
  const st = (typeOf(s)?.statuses || []).find(x => x.key === s.status)
  return st ? statusColor(st) : null
}

const norm = v => (v == null ? '' : String(v))
const dateKey = s => (s.startDate && s.endDate) ? `${s.startDate}→${s.endDate}` : (s.when || (s.year ? `${s.year}-${s.month}` : ''))
// A short "what changed vs. the previous version" summary.
function summarize(cur, prev) {
  if (!cur) return ''
  if (!prev) return 'Created'
  const c = []
  const t = typeOf(cur)
  if (norm(cur.title) !== norm(prev.title)) c.push('Title')
  if (norm(cur.status) !== norm(prev.status)) c.push('Status')
  if (norm(cur.swimlaneId) !== norm(prev.swimlaneId) || norm(cur.subLaneId) !== norm(prev.subLaneId)) c.push('Area')
  if (dateKey(cur) !== dateKey(prev)) c.push('Date')
  if (norm(cur.maturity) !== norm(prev.maturity)) c.push('Maturity')
  if (norm(cur.progress) !== norm(prev.progress)) c.push('Progress')
  if (norm(cur.assigneeId) !== norm(prev.assigneeId)) c.push('Assignee')
  for (const f of (t?.fields || [])) {
    if (norm(cur.data?.[f.key] ?? cur[f.key]) !== norm(prev.data?.[f.key] ?? prev[f.key])) c.push(f.label || f.key)
  }
  if (!c.length) return 'No field changes'
  return 'Changed ' + c.join(', ')
}

const rows = computed(() => revisions.value.map((r, i) => {
  const snap = snaps.value[r.version] || null
  const prev = revisions.value[i + 1] ? snaps.value[revisions.value[i + 1].version] : null // older neighbour
  return {
    version: r.version, editedBy: r.editedBy, editedAt: r.editedAt,
    statusLabel: statusLabelOf(snap), statusColor: statusColorOf(snap),
    summary: summarize(snap, prev), created: !prev, isHead: i === 0,
  }
}))
</script>

<style scoped>
.ih { display: flex; flex-direction: column; gap: 6px; }
.ih-empty { font-size: 12px; color: var(--clr-text-3); padding: 6px 0; }
/* Each revision is a single wide row — uses the full pane width. */
.ih-rev {
  display: flex; align-items: center; gap: 12px; cursor: pointer;
  text-align: left; padding: 7px 10px 7px 16px; border-radius: var(--r-md);
  border: 1px solid var(--clr-border-light); background: var(--clr-surface);
  transition: border-color 0.13s, background 0.13s;
}
.ih-rev:hover { background: var(--clr-surface-2); }
.ih-rev.on { border-color: var(--clr-accent); background: rgba(0,113,227,0.07); }
.ih-v { flex-shrink: 0; min-width: 32px; font-size: 12px; font-weight: 700; color: var(--clr-accent); }
.ih-dot { flex-shrink: 0; width: 9px; height: 9px; border-radius: 50%; }
.ih-status { flex-shrink: 0; font-size: 13px; font-weight: 600; color: var(--clr-text); }
.ih-badge { flex-shrink: 0; font-size: 9px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.4px; color: #1a7f37; background: rgba(48,209,88,0.16); border-radius: 100px; padding: 1px 7px; }
.ih-changes { flex: 1 1 auto; min-width: 0; font-size: 12.5px; color: var(--clr-text-2); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.ih-changes.created { color: #1a7f37; font-weight: 600; }
.ih-meta { flex-shrink: 0; font-size: 11.5px; color: var(--clr-text-3); white-space: nowrap; }
.ih-copy { flex-shrink: 0; width: 26px; height: 26px; display: inline-flex; align-items: center; justify-content: center; border-radius: var(--r-sm); color: var(--clr-text-3); background: none; transition: background 0.12s, color 0.12s; }
.ih-copy:hover { background: var(--clr-border-light); color: var(--clr-text); }
.ih-copy.done { color: #1a7f37; }
.ih-view { flex-shrink: 0; font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.4px; color: var(--clr-accent); }
</style>
