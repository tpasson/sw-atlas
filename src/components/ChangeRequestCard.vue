<template>
  <div class="crc" :class="[cr.status, { open: expanded, 'crc-focus': focused }]" :data-cr="cr.id">
    <button type="button" class="crc-head" @click="$emit('toggle', cr.id)">
      <span class="crc-kind" :class="cr.kind" :title="cr.kind === 'create' ? 'New item' : 'Edit'">{{ cr.kind === 'create' ? '＋' : '↻' }}</span>
      <span class="crc-title">{{ title }}</span>
      <span class="crc-status" :class="cr.status">{{ cr.status }}</span>
      <span class="crc-by">{{ cr.authorName || 'someone' }} · {{ fmt(cr.createdAt) }}</span>
      <ChevronRight class="crc-chev" :class="{ open: expanded }" :size="15" />
    </button>

    <div v-if="expanded" class="crc-body">
      <p v-if="cr.note" class="cr-note">“{{ cr.note }}”</p>

      <dl v-if="rows.length" class="cr-diff">
        <template v-for="r in rows" :key="r.label">
          <dt>{{ r.label }}</dt>
          <dd>
            <span v-if="r.from" class="cr-from">{{ r.from }}</span>
            <span v-if="r.from" class="cr-arrow">→</span>
            <span class="cr-to">{{ r.to || '—' }}</span>
          </dd>
        </template>
      </dl>
      <p v-else-if="!deps || (!deps.added.length && !deps.removed.length)" class="cr-nochange">No field changes.</p>

      <div v-if="deps && (deps.added.length || deps.removed.length)" class="cr-deps">
        <div class="cr-deps-h">Dependencies</div>
        <div v-for="(d, i) in deps.added" :key="'a'+i" class="cr-dep add"><span class="cr-dep-op">+</span> {{ d.rel }} <strong>{{ d.title }}</strong></div>
        <div v-for="(d, i) in deps.removed" :key="'r'+i" class="cr-dep rem"><span class="cr-dep-op">−</span> {{ d.rel }} <strong>{{ d.title }}</strong></div>
      </div>

      <div v-if="cr.status === 'pending' && canAdminWorkspace()" class="cr-actions">
        <button class="cr-btn cr-approve" :disabled="busy" @click="decide(true)">Approve</button>
        <button class="cr-btn cr-reject" :disabled="busy" @click="decide(false)">Reject</button>
      </div>
      <div v-else-if="cr.status !== 'pending'" class="cr-decided">
        {{ cr.status === 'approved' ? 'Approved' : 'Rejected' }}
        <template v-if="cr.deciderName">by {{ cr.deciderName }}</template>
        <template v-if="cr.decidedAt"> · {{ fmt(cr.decidedAt) }}</template>
        <span v-if="cr.decisionNote" class="cr-dnote">— {{ cr.decisionNote }}</span>
      </div>
      <div v-else class="cr-waiting">Waiting for the project owner to decide.</div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ChevronRight } from 'lucide-vue-next'
import { decideChangeRequest, canAdminWorkspace, store, itemTypeByKey, MATURITY_STAGES, MONTHS, memberName, RELATIONSHIP_TYPES } from '../stores/useAppStore.js'

const props = defineProps({
  cr: { type: Object, required: true },
  expanded: { type: Boolean, default: false },
  focused: { type: Boolean, default: false },
})
defineEmits(['toggle'])

const busy = ref(false)
async function decide(approve) {
  let note = ''
  if (!approve) { note = prompt('Reason for rejecting (optional):') || ''; if (note === null) return }
  busy.value = true
  try { await decideChangeRequest(props.cr.id, approve, note) }
  catch (e) { alert(e?.message || 'Could not update the change request') }
  finally { busy.value = false }
}

const title = computed(() => props.cr.targetTitle || props.cr.payload?.title || 'Untitled')

function fmt(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  return isNaN(d) ? '' : d.toLocaleDateString('en-US', { day: 'numeric', month: 'short', year: 'numeric' })
}

const norm = v => (v == null ? '' : String(v).trim())
function typeLabel(key) { return itemTypeByKey(key)?.label || key || '' }
function laneName(id) { return store.swimlanes.find(s => s.id === id)?.name || '' }
function dateOf(it) {
  if (!it) return ''
  if (it.startDate && it.endDate) return `${it.startDate} → ${it.endDate}`
  return it.when || (it.year ? `${MONTHS[(it.month || 1) - 1]} ${it.year}` : '')
}
const pct = v => (v != null && v !== '' ? `${v}%` : '')
const maturity = v => (v ? MATURITY_STAGES[v - 1] : '')
function statusLabel(it) {
  if (!it || !it.status) return ''
  const t = itemTypeByKey(it.typeKey || it.kind)
  return (t?.statuses || []).find(s => s.key === it.status)?.label || it.status
}

// Field-level diff between the live item (or empty, for a new item) and the proposal.
const rows = computed(() => {
  const cr = props.cr
  const p = cr.payload || {}
  const cur = cr.kind === 'create' ? {} : (store.milestones.find(m => m.id === cr.targetItemId) || {})
  const out = []
  const push = (label, a, b) => { if (norm(a) !== norm(b)) out.push({ label, from: a, to: b }) }
  push('Title', cur.title, p.title)
  push('Type', typeLabel(cur.typeKey || cur.kind), typeLabel(p.typeKey || p.kind))
  push('Area', laneName(cur.swimlaneId), laneName(p.swimlaneId))
  push('Date', dateOf(cur), dateOf(p))
  push('Status', statusLabel(cur), statusLabel(p))
  push('Progress', pct(cur.progress), pct(p.progress))
  push('Maturity', maturity(cur.maturity), maturity(p.maturity))
  push('Assigned to', memberName(cur.assigneeId), memberName(p.assigneeId))
  const t = itemTypeByKey(p.typeKey || p.kind)
  for (const f of (t?.fields || [])) push(f.label || f.key, cur.data?.[f.key], p.data?.[f.key])
  return out
})

// Dependency changes: the payload carries the item's full desired link set, diffed
// against the item's current links so the approver sees exactly what changes.
function itemTitle(id) { return store.milestones.find(m => m.id === id)?.title || id }
function linkFace(l, selfId) {
  const other = l.a === selfId ? l.b : l.a
  const rt = RELATIONSHIP_TYPES.find(r => r.key === (l.rel || 'depends-on'))
  const rel = l.a === selfId ? (rt?.label || l.rel) : (rt?.inverse || l.rel)
  return { rel, title: itemTitle(other) }
}
const deps = computed(() => {
  const cr = props.cr
  const p = cr.payload || {}
  if (!Array.isArray(p.links)) return null // proposal predates dependency support
  const selfId = cr.kind === 'create' ? p.id : cr.targetItemId
  const key = l => `${l.a}|${l.b}|${l.rel || 'depends-on'}`
  const proposed = new Map(p.links.map(l => [key(l), l]))
  const current = cr.kind === 'create'
    ? new Map()
    : new Map(store.links.filter(l => l.a === selfId || l.b === selfId).map(l => [key(l), l]))
  const added = [], removed = []
  for (const [k, l] of proposed) if (!current.has(k)) added.push(linkFace(l, selfId))
  for (const [k, l] of current) if (!proposed.has(k)) removed.push(linkFace(l, selfId))
  return { added, removed }
})
</script>

<style scoped>
.crc { border: 1px solid var(--clr-border-light); border-radius: var(--r-md); overflow: hidden; background: var(--clr-surface); }
.crc.pending { border-left: 3px solid #FF9F0A; }
.crc.approved { border-left: 3px solid #30D158; }
.crc.rejected { border-left: 3px solid #FF3B30; }
.crc.rejected:not(.open) { opacity: 0.7; }
.crc.crc-focus { box-shadow: 0 0 0 2px var(--clr-accent); opacity: 1; }

.crc-head { display: flex; align-items: center; gap: 9px; width: 100%; text-align: left; padding: 9px 12px; background: none; cursor: pointer; transition: background 0.12s; }
.crc-head:hover { background: var(--clr-surface-2); }
.crc-kind { flex-shrink: 0; width: 18px; height: 18px; display: inline-flex; align-items: center; justify-content: center; border-radius: 5px; font-size: 12px; font-weight: 700; }
.crc-kind.create { color: #1a7f37; background: rgba(48,209,88,0.16); }
.crc-kind.edit { color: var(--clr-accent); background: rgba(0,113,227,0.12); }
.crc-title { font-size: 13px; font-weight: 600; color: var(--clr-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; min-width: 0; }
.crc-status { flex-shrink: 0; font-size: 9px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.4px; border-radius: 100px; padding: 2px 7px; }
.crc-status.pending { color: #b7791f; background: rgba(255,159,10,0.16); }
.crc-status.approved { color: #1a7f37; background: rgba(48,209,88,0.16); }
.crc-status.rejected { color: #c0392b; background: rgba(255,59,48,0.14); }
.crc-by { flex-shrink: 0; margin-left: auto; font-size: 11px; color: var(--clr-text-3); white-space: nowrap; }
.crc-chev { flex-shrink: 0; color: var(--clr-text-3); transition: transform 0.15s; }
.crc-chev.open { transform: rotate(90deg); }

.crc-body { padding: 4px 14px 14px 14px; border-top: 1px solid var(--clr-border-light); }

.cr-note { font-size: 13px; color: var(--clr-text-2); font-style: italic; margin: 10px 0 4px; }

.cr-diff { display: grid; grid-template-columns: 96px 1fr; gap: 5px 12px; margin: 10px 0 4px; }
.cr-diff dt { font-size: 12px; font-weight: 600; color: var(--clr-text-3); }
.cr-diff dd { font-size: 13px; color: var(--clr-text); display: flex; align-items: center; gap: 7px; flex-wrap: wrap; }
.cr-from { color: var(--clr-text-3); text-decoration: line-through; }
.cr-arrow { color: var(--clr-text-3); }
.cr-to { color: var(--clr-text); font-weight: 500; }
.cr-nochange { font-size: 12px; color: var(--clr-text-3); margin: 10px 0 0; }

.cr-deps { margin: 10px 0 2px; }
.cr-deps-h { font-size: 12px; font-weight: 600; color: var(--clr-text-3); margin-bottom: 4px; }
.cr-dep { font-size: 13px; color: var(--clr-text); display: flex; align-items: center; gap: 6px; padding: 1px 0; }
.cr-dep-op { display: inline-block; width: 12px; text-align: center; font-weight: 700; }
.cr-dep.add .cr-dep-op { color: #1a7f37; }
.cr-dep.rem { color: var(--clr-text-3); }
.cr-dep.rem strong { text-decoration: line-through; }
.cr-dep.rem .cr-dep-op { color: #c0392b; }

.cr-actions { display: flex; gap: 8px; margin-top: 12px; }
.cr-btn { font-size: 13px; font-weight: 600; border-radius: var(--r-md); padding: 7px 16px; }
.cr-approve { background: #30D158; color: #06310f; }
.cr-approve:hover:not(:disabled) { filter: brightness(1.05); }
.cr-reject { background: var(--clr-surface-2); color: #c0392b; }
.cr-reject:hover:not(:disabled) { background: rgba(255,59,48,0.1); }
.cr-btn:disabled { opacity: 0.5; }

.cr-decided { margin-top: 10px; font-size: 12px; color: var(--clr-text-3); }
.cr-dnote { color: var(--clr-text-2); }
.cr-waiting { margin-top: 10px; font-size: 12px; color: var(--clr-text-3); }
</style>
