<template>
  <div class="card">
    <p class="section-label">Account</p>
    <p class="card-hint">
      Signed in as <strong>{{ session.username }}</strong><span v-if="session.role"> · {{ session.role }}</span>.
    </p>

    <p class="field-label">Your profile</p>
    <p class="card-hint">Optional — shown when someone opens your profile. Your email stays hidden from anonymous visitors.</p>
    <div class="acc-grid">
      <input v-model="firstName" class="field-input" placeholder="First name" autocomplete="given-name" />
      <input v-model="lastName" class="field-input" placeholder="Last name" autocomplete="family-name" />
    </div>
    <div class="acc-row">
      <input v-model="email" class="field-input" type="email" placeholder="Email" autocomplete="email" @keyup.enter="onSaveProfile" />
      <button class="btn-add" :disabled="busy" @click="onSaveProfile">Save</button>
    </div>

    <p class="field-label">Change your username</p>
    <p class="card-hint">
      This also changes your plan URL — from <code>/{{ session.username }}</code> to
      <code>/your-new-name</code>. Existing links to your plan will stop working.
    </p>
    <div class="acc-row">
      <input
        v-model="uname"
        class="field-input"
        placeholder="New username"
        autocomplete="off"
        @keyup.enter="onRename"
      />
      <button class="btn-add" :disabled="busy || !uname.trim()" @click="onRename">Rename</button>
    </div>

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
const uname = ref('')
const firstName = ref(session.firstName || '')
const lastName = ref(session.lastName || '')
const email = ref(session.email || '')
const busy = ref(false)
const msg = ref(null)

async function onSaveProfile() {
  msg.value = null; busy.value = true
  try {
    const data = { firstName: firstName.value.trim(), lastName: lastName.value.trim(), email: email.value.trim() }
    await api.updateOwnProfile(data)
    session.firstName = data.firstName; session.lastName = data.lastName; session.email = data.email
    msg.value = { type: 'ok', text: 'Your profile was saved.' }
  } catch (e) {
    msg.value = { type: 'err', text: e.message || 'Save failed' }
  }
  busy.value = false
}

async function onRename() {
  const next = uname.value.trim().toLowerCase()
  if (!next || next === session.username) return
  msg.value = null; busy.value = true
  try {
    const res = await api.renameOwnUsername(next)
    // Our username + plan URL changed; reload at the new slug with the fresh cookie.
    window.location.assign('/' + (res?.slug || next))
  } catch (e) {
    msg.value = { type: 'err', text: e.message || 'Rename failed' }
    busy.value = false
  }
}

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
.acc-grid { display: flex; gap: 8px; margin: 6px 0 0; }
.acc-grid .field-input { flex: 1; min-width: 0; }
.field-input { flex: 1; padding: 7px 10px; font-size: 13px; color: var(--clr-text);
  background: var(--clr-surface); border: 1px solid var(--clr-border-light); border-radius: var(--r-md); }
.field-label { font-size: 12px; font-weight: 600; color: var(--clr-text-2); margin: 10px 0 0; }
/* .card-hint comes from the shared Settings template in style.css. */
.btn-add { padding: 8px 14px; font-size: 13px; font-weight: 600; white-space: nowrap;
  color: var(--clr-accent); background: rgba(0,113,227,0.08); border-radius: var(--r-md); transition: background 0.15s; }
.btn-add:hover:not(:disabled) { background: rgba(0,113,227,0.14); }
.btn-add:disabled { opacity: 0.4; cursor: not-allowed; }
/* .data-msg comes from the shared Settings template in style.css. */
</style>
