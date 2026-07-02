<template>
  <div class="tab-pane">
    <div class="card">
      <p class="section-label">Instance</p>
      <p class="card-hint">General server settings — applied to the whole instance.</p>
      <div class="ss-inrow">
        <input v-model="name" class="field-input" placeholder="Instance name (optional)" @keyup.enter="save" />
        <button class="btn-add" :disabled="busy" @click="save">Save</button>
      </div>
      <p v-if="msg" class="data-msg" :class="msg.type">{{ msg.text }}</p>
    </div>

    <div class="card">
      <p class="section-label">Limits</p>
      <p class="card-hint">
        Abuse guards, applied to the whole instance. <strong>0 = unlimited</strong> for any field.
      </p>
      <div class="ss-limit">
        <div class="ss-limit-info"><span class="setting-name">Writes / minute per user</span>
          <span class="setting-desc">Throttles edits per account; UI bursts are fine, loops get blocked.</span></div>
        <input v-model.number="limits.writesPerMinute" type="number" min="0" class="field-input sm" />
      </div>
      <div class="ss-limit">
        <div class="ss-limit-info"><span class="setting-name">Items per plan</span>
          <span class="setting-desc">Caps native items in a single plan/workspace.</span></div>
        <input v-model.number="limits.maxItemsPerPlan" type="number" min="0" class="field-input sm" />
      </div>
      <div class="ss-limit">
        <div class="ss-limit-info"><span class="setting-name">Projects per user</span>
          <span class="setting-desc">Collaborative projects a user can own (home plan excluded).</span></div>
        <input v-model.number="limits.maxProjectsPerUser" type="number" min="0" class="field-input sm" />
      </div>
      <button class="btn-add" :disabled="busyLim" @click="saveLimits">Save limits</button>
      <p v-if="limMsg" class="data-msg" :class="limMsg.type">{{ limMsg.text }}</p>
    </div>

    <div class="card">
      <p class="section-label">Statistics</p>
      <div class="ss-grid">
        <div class="ss-stat"><span class="ss-num">{{ stats.users ?? '—' }}</span><span class="ss-lbl">Users</span></div>
        <div class="ss-stat"><span class="ss-num">{{ stats.workspaces ?? '—' }}</span><span class="ss-lbl">Plans</span></div>
        <div class="ss-stat"><span class="ss-num">{{ stats.items ?? '—' }}</span><span class="ss-lbl">Items</span></div>
      </div>
      <div class="ss-rows">
        <div class="ss-row"><span>Version</span><span>{{ stats.version || '—' }}</span></div>
        <div class="ss-row"><span>Uptime</span><span>{{ uptime }}</span></div>
        <div v-if="stats.startedAt" class="ss-row"><span>Started</span><span>{{ fmtStarted }}</span></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { api } from '../api.js'

const name = ref('')
const stats = reactive({})
const busy = ref(false)
const msg = ref(null)

const limits = reactive({ writesPerMinute: 0, maxItemsPerPlan: 0, maxProjectsPerUser: 0 })
const busyLim = ref(false)
const limMsg = ref(null)

const uptime = computed(() => {
  const s = stats.uptimeSeconds || 0
  const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600), m = Math.floor((s % 3600) / 60)
  if (d) return `${d}d ${h}h ${m}m`
  if (h) return `${h}h ${m}m`
  return `${m}m`
})
const fmtStarted = computed(() => { try { return new Date(stats.startedAt).toLocaleString() } catch { return stats.startedAt } })

async function load() {
  try {
    const r = await api.getServerInfo()
    Object.assign(stats, r.stats || {})
    name.value = (r.settings && r.settings.name) || ''
  } catch (e) { msg.value = { type: 'err', text: e.message || 'Failed to load' } }
  try { Object.assign(limits, await api.getLimits()) } catch { /* keep zeros */ }
}
onMounted(load)

async function saveLimits() {
  limMsg.value = null; busyLim.value = true
  try {
    const clean = await api.setLimits({
      writesPerMinute: Math.max(0, limits.writesPerMinute | 0),
      maxItemsPerPlan: Math.max(0, limits.maxItemsPerPlan | 0),
      maxProjectsPerUser: Math.max(0, limits.maxProjectsPerUser | 0),
    })
    Object.assign(limits, clean)
    limMsg.value = { type: 'ok', text: 'Limits saved.' }
  } catch (e) { limMsg.value = { type: 'err', text: e.message || 'Save failed' } }
  busyLim.value = false
}

async function save() {
  msg.value = null; busy.value = true
  try {
    await api.setServerSettings({ name: name.value.trim() })
    msg.value = { type: 'ok', text: 'Saved.' }
  } catch (e) { msg.value = { type: 'err', text: e.message || 'Save failed' } }
  busy.value = false
}
</script>

<style scoped>
.tab-pane { display: flex; flex-direction: column; gap: 14px; }
.ss-inrow { display: flex; gap: 8px; margin: 6px 0 0; }
.field-input { flex: 1; min-width: 0; padding: 7px 10px; font-size: 13px; color: var(--clr-text);
  background: var(--clr-surface); border: 1px solid var(--clr-border-light); border-radius: var(--r-md); }
.btn-add { padding: 8px 14px; font-size: 13px; font-weight: 600; white-space: nowrap;
  color: var(--clr-accent); background: rgba(0,113,227,0.08); border-radius: var(--r-md); }
.btn-add:disabled { opacity: 0.5; cursor: not-allowed; }
.ss-grid { display: flex; gap: 10px; margin-top: 8px; }
.ss-stat { flex: 1; display: flex; flex-direction: column; align-items: center; gap: 3px;
  padding: 14px 8px; background: var(--clr-surface-2); border-radius: var(--r-md); }
.ss-num { font-size: 22px; font-weight: 700; color: var(--clr-text); letter-spacing: -0.5px; }
.ss-lbl { font-size: 10.5px; text-transform: uppercase; letter-spacing: 0.4px; color: var(--clr-text-3); }
.ss-limit { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-top: 10px; }
.ss-limit-info { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.field-input.sm { flex: 0 0 84px; max-width: 84px; text-align: right; }
.ss-rows { margin-top: 12px; display: flex; flex-direction: column; }
.ss-row { display: flex; justify-content: space-between; font-size: 13px; color: var(--clr-text-2);
  padding: 8px 0; border-top: 1px solid var(--clr-border-light); }
</style>
