<template>
  <div class="crv">
    <div class="crv-head">
      <ClipboardCheck :size="18" />
      <h1 class="crv-title">Change Requests</h1>
      <span v-if="pending.length" class="crv-count">{{ pending.length }} pending</span>
      <button v-if="canPropose" class="crv-new" @click="$emit('propose-new')">+ Propose new item</button>
    </div>

    <div class="crv-body">
      <p v-if="!changeRequests.list.length" class="crv-empty">
        No change requests yet. Open a milestone and choose <strong>Propose change</strong> to suggest one —
        the project owner can then approve it onto the timeline.
      </p>

      <div v-for="cr in changeRequests.list" :key="cr.id" class="cr" :class="cr.status">
        <div class="cr-top">
          <span class="cr-status" :class="cr.status">{{ cr.status }}</span>
          <span class="cr-kind">{{ cr.kind === 'create' ? 'New item' : 'Edit' }}<template v-if="cr.targetTitle || cr.payload?.title">: <strong>{{ cr.targetTitle || cr.payload?.title }}</strong></template></span>
          <span class="cr-by">by {{ cr.authorName || 'someone' }} · {{ fmt(cr.createdAt) }}</span>
        </div>

        <p v-if="cr.note" class="cr-note">“{{ cr.note }}”</p>

        <dl v-if="changeRows(cr).length" class="cr-diff">
          <template v-for="r in changeRows(cr)" :key="r.label">
            <dt>{{ r.label }}</dt>
            <dd>
              <span v-if="r.from" class="cr-from">{{ r.from }}</span>
              <span v-if="r.from" class="cr-arrow">→</span>
              <span class="cr-to">{{ r.to || '—' }}</span>
            </dd>
          </template>
        </dl>
        <p v-else class="cr-nochange">No field changes.</p>

        <div v-if="cr.status === 'pending' && canAdminWorkspace()" class="cr-actions">
          <button class="cr-btn cr-approve" :disabled="busy === cr.id" @click="decide(cr, true)">Approve</button>
          <button class="cr-btn cr-reject" :disabled="busy === cr.id" @click="decide(cr, false)">Reject</button>
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
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ClipboardCheck } from 'lucide-vue-next'
import { changeRequests, decideChangeRequest, canAdminWorkspace, store, itemTypeByKey, MATURITY_STAGES, MONTHS, memberName, session, workspace } from '../stores/useAppStore.js'

defineEmits(['propose-new'])

const pending = computed(() => changeRequests.list.filter(c => c.status === 'pending'))
const canPropose = computed(() => session.authenticated && !!workspace.role)
const busy = ref(null)

async function decide(cr, approve) {
  let note = ''
  if (!approve) { note = prompt('Reason for rejecting (optional):') || ''; if (note === null) return }
  busy.value = cr.id
  try { await decideChangeRequest(cr.id, approve, note) }
  catch (e) { alert(e?.message || 'Could not update the change request') }
  finally { busy.value = null }
}

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

// Field-level diff between the live item (or empty, for a new item) and the proposal.
function changeRows(cr) {
  const p = cr.payload || {}
  const cur = cr.kind === 'create' ? {} : (store.milestones.find(m => m.id === cr.targetItemId) || {})
  const rows = []
  const push = (label, a, b) => { if (norm(a) !== norm(b)) rows.push({ label, from: a, to: b }) }
  push('Title', cur.title, p.title)
  push('Type', typeLabel(cur.typeKey || cur.kind), typeLabel(p.typeKey || p.kind))
  push('Area', laneName(cur.swimlaneId), laneName(p.swimlaneId))
  push('Date', dateOf(cur), dateOf(p))
  push('Progress', pct(cur.progress), pct(p.progress))
  push('Maturity', maturity(cur.maturity), maturity(p.maturity))
  push('Who', memberName(cur.assigneeId), memberName(p.assigneeId))
  push('What', cur.what, p.what)
  push('Why', cur.why, p.why)
  push('Where', cur.how, p.how)
  const t = itemTypeByKey(p.typeKey || p.kind)
  for (const f of (t?.fields || [])) push(f.label || f.key, cur.data?.[f.key], p.data?.[f.key])
  return rows
}
</script>

<style scoped>
.crv { flex: 1; min-height: 0; display: flex; flex-direction: column; }
.crv-head { display: flex; align-items: center; gap: 9px; padding: 16px 24px 12px; border-bottom: 1px solid var(--clr-border-light); color: var(--clr-text); flex-shrink: 0; }
.crv-title { font-size: 17px; font-weight: 700; }
.crv-count { margin-left: 4px; font-size: 11px; font-weight: 700; color: #fff; background: #FF9F0A; border-radius: 100px; padding: 2px 9px; }
.crv-new { margin-left: auto; font-size: 13px; font-weight: 600; color: #fff; background: var(--clr-accent); border-radius: var(--r-md); padding: 7px 14px; }
.crv-new:hover { background: var(--clr-accent-hover); }
.crv-body { flex: 1; min-height: 0; overflow-y: auto; padding: 20px 24px 40px; }
.crv-empty { max-width: 640px; font-size: 13px; color: var(--clr-text-3); line-height: 1.6; }

.cr { max-width: 720px; margin: 0 auto 14px; border: 1px solid var(--clr-border-light); border-radius: var(--r-md); padding: 14px 16px; }
.cr.pending { border-left: 3px solid #FF9F0A; }
.cr.approved { border-left: 3px solid #30D158; }
.cr.rejected { border-left: 3px solid #FF3B30; opacity: 0.75; }

.cr-top { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
.cr-status { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.4px; border-radius: 100px; padding: 2px 8px; }
.cr-status.pending { color: #b7791f; background: rgba(255,159,10,0.16); }
.cr-status.approved { color: #1a7f37; background: rgba(48,209,88,0.16); }
.cr-status.rejected { color: #c0392b; background: rgba(255,59,48,0.14); }
.cr-kind { font-size: 13px; color: var(--clr-text); }
.cr-by { margin-left: auto; font-size: 11px; color: var(--clr-text-3); }

.cr-note { font-size: 13px; color: var(--clr-text-2); font-style: italic; margin: 9px 0 4px; }

.cr-diff { display: grid; grid-template-columns: 96px 1fr; gap: 5px 12px; margin: 10px 0 4px; }
.cr-diff dt { font-size: 12px; font-weight: 600; color: var(--clr-text-3); }
.cr-diff dd { font-size: 13px; color: var(--clr-text); display: flex; align-items: center; gap: 7px; flex-wrap: wrap; }
.cr-from { color: var(--clr-text-3); text-decoration: line-through; }
.cr-arrow { color: var(--clr-text-3); }
.cr-to { color: var(--clr-text); font-weight: 500; }
.cr-nochange { font-size: 12px; color: var(--clr-text-3); margin: 8px 0 0; }

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
