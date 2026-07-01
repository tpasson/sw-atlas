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
}
onMounted(load)

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
.ss-rows { margin-top: 12px; display: flex; flex-direction: column; }
.ss-row { display: flex; justify-content: space-between; font-size: 13px; color: var(--clr-text-2);
  padding: 8px 0; border-top: 1px solid var(--clr-border-light); }
</style>
