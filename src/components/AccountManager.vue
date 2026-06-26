<template>
  <div class="card">
    <p class="section-label">Account</p>
    <p class="card-hint">
      Signed in as <strong>{{ session.username }}</strong><span v-if="session.role"> · {{ session.role }}</span>.
    </p>

    <p class="field-label">Change your password</p>
    <div class="acc-row">
      <input
        v-model="pw"
        class="field-input"
        type="password"
        placeholder="New password"
        autocomplete="new-password"
        @keyup.enter="onChange"
      />
      <button class="btn-add" :disabled="busy || !pw" @click="onChange">Update</button>
    </div>
    <p v-if="msg" class="data-msg" :class="msg.type">{{ msg.text }}</p>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { api } from '../api.js'
import { session } from '../stores/useAppStore.js'

const pw = ref('')
const busy = ref(false)
const msg = ref(null)

async function onChange() {
  msg.value = null; busy.value = true
  try {
    await api.changeOwnPassword(pw.value)
    pw.value = ''
    msg.value = { type: 'ok', text: 'Your password was updated.' }
  } catch (e) {
    msg.value = { type: 'err', text: e.message || 'Update failed' }
  }
  busy.value = false
}
</script>

<style scoped>
.acc-row { display: flex; gap: 8px; margin: 6px 0 0; }
.field-input { flex: 1; padding: 7px 10px; font-size: 13px; color: var(--clr-text);
  background: var(--clr-surface); border: 1px solid var(--clr-border-light); border-radius: var(--r-md); }
.field-label { font-size: 12px; font-weight: 600; color: var(--clr-text-2); margin: 10px 0 0; }
.card-hint { font-size: 12px; color: var(--clr-text-3); margin: 0 0 4px; line-height: 1.45; }
.btn-add { padding: 8px 14px; font-size: 13px; font-weight: 600; white-space: nowrap;
  color: var(--clr-accent); background: rgba(0,113,227,0.08); border-radius: var(--r-md); transition: background 0.15s; }
.btn-add:hover:not(:disabled) { background: rgba(0,113,227,0.14); }
.btn-add:disabled { opacity: 0.4; cursor: not-allowed; }
.data-msg { font-size: 13px; margin: 8px 0 0; }
.data-msg.ok { color: var(--clr-accent); }
.data-msg.err { color: var(--clr-danger); }
</style>
