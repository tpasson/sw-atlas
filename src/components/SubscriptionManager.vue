<template>
  <div class="card">
    <p class="section-label">Subscriptions</p>
    <p class="card-hint">
      Subscribe to a colleague's published schedule — paste the subscribe link they sent.
      Their milestones appear as read-only areas you can link your own work to, kept fresh
      automatically. Hide or reorder them in the <strong>Areas</strong> tab.
    </p>

    <div class="sub-new">
      <input v-model="code" class="field-input" placeholder="Paste subscribe link…" @keyup.enter="onSubscribe" />
      <button class="btn-add" :disabled="busy || !code.trim()" @click="onSubscribe">Subscribe</button>
    </div>
    <p v-if="msg" class="data-msg" :class="msg.type">{{ msg.text }}</p>

    <div v-if="subs.length" class="sub-list">
      <div v-for="s in subs" :key="s.id" class="sub-row">
        <div class="sub-meta">
          <span class="sub-name">{{ s.sourceLabel }}</span>
          <span class="sub-sub" :class="{ err: (s.lastStatus || '').startsWith('error') }">{{ statusText(s) }}</span>
        </div>
        <div class="sub-actions">
          <button class="link-btn" :disabled="busy" @click="onSync(s)">Sync now</button>
          <button class="link-btn danger" :disabled="busy" @click="onRemove(s)">Remove</button>
        </div>
      </div>
    </div>
    <div v-else class="empty">No subscriptions yet.</div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'
import { loadPlan } from '../stores/useAppStore.js'

const subs = ref([])
const code = ref('')
const busy = ref(false)
const msg = ref(null)

function statusText(s) {
  const when = s.lastSyncedAt ? new Date(s.lastSyncedAt).toLocaleString() : 'never'
  return `${s.lastStatus || '—'} · last sync ${when}`
}

async function load() {
  try { subs.value = (await api.listSubscriptions()).subscriptions || [] }
  catch (e) { msg.value = { type: 'err', text: e.message || 'Failed to load' } }
}
onMounted(load)

async function onSubscribe() {
  msg.value = null; busy.value = true
  try {
    const s = await api.createSubscription({ code: code.value.trim() })
    code.value = ''
    await load(); await loadPlan()
    msg.value = (s.lastStatus || '').startsWith('ok')
      ? { type: 'ok', text: `Subscribed to "${s.sourceLabel}".` }
      : { type: 'err', text: `Subscribed, but first sync said: ${s.lastStatus}` }
  } catch (e) { msg.value = { type: 'err', text: e.message || 'Subscribe failed' } }
  busy.value = false
}

async function onSync(s) {
  msg.value = null; busy.value = true
  try { await api.syncSubscription(s.id); await load(); await loadPlan() }
  catch (e) { msg.value = { type: 'err', text: e.message || 'Sync failed' } }
  busy.value = false
}

async function onRemove(s) {
  if (!confirm(`Remove subscription "${s.sourceLabel}"? Its mirrored areas disappear.`)) return
  busy.value = true
  try { await api.deleteSubscription(s.id); await load(); await loadPlan() }
  catch (e) { msg.value = { type: 'err', text: e.message || 'Remove failed' } }
  busy.value = false
}
</script>

<style scoped>
.sub-new { display: flex; gap: 8px; margin: 10px 0; }
.field-input { flex: 1; padding: 7px 10px; font-size: 13px; color: var(--clr-text);
  background: var(--clr-surface); border: 1px solid var(--clr-border-light); border-radius: var(--r-md); }
.sub-list { display: flex; flex-direction: column; gap: 6px; }
.sub-row { display: flex; align-items: center; justify-content: space-between; gap: 10px;
  border: 1px solid var(--clr-border-light); border-radius: var(--r-md); padding: 8px 12px; }
.sub-name { font-size: 13.5px; font-weight: 600; color: var(--clr-text); }
.sub-sub { display: block; font-size: 11.5px; color: var(--clr-text-3); margin-top: 1px; }
.sub-sub.err { color: var(--clr-danger); }
.sub-actions { display: flex; gap: 8px; flex-shrink: 0; }
.btn-add {
  padding: 8px 14px; font-size: 13px; font-weight: 600; white-space: nowrap;
  color: var(--clr-accent); background: rgba(0,113,227,0.08);
  border-radius: var(--r-md); transition: background 0.15s;
}
.btn-add:hover:not(:disabled) { background: rgba(0,113,227,0.14); }
.btn-add:disabled { opacity: 0.4; cursor: not-allowed; }
.link-btn { background: none; font-size: 12px; font-weight: 600; color: var(--clr-accent); padding: 4px 6px; border-radius: var(--r-sm); }
.link-btn.danger { color: var(--clr-danger); }
.link-btn:hover:not(:disabled) { text-decoration: underline; }
.link-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.empty { font-size: 12.5px; color: var(--clr-text-3); padding: 8px 0; }
.data-msg { font-size: 13px; margin: 6px 0; }
.data-msg.ok { color: var(--clr-accent); }
.data-msg.err { color: var(--clr-danger); }
</style>
