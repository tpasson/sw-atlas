<template>
  <div class="card-stack">
    <!-- Create -->
    <div class="card">
      <p class="section-label">Add user</p>
      <p class="card-hint">
        Each user gets their own private workspace and lands on it when they log in.
        Editors manage their own plan; admins additionally manage accounts here.
      </p>
      <div class="user-new">
        <input v-model="form.username" class="field-input" placeholder="Username" autocomplete="off" @keyup.enter="onCreate" />
        <input v-model="form.password" class="field-input" type="password" placeholder="Password" autocomplete="new-password" @keyup.enter="onCreate" />
        <select v-model="form.role" class="field-select">
          <option value="editor">Editor</option>
          <option value="admin">Admin</option>
        </select>
        <button class="btn-add" :disabled="busy || !form.username.trim() || !form.password" @click="onCreate">Add</button>
      </div>
      <p v-if="msg" class="data-msg" :class="msg.type">{{ msg.text }}</p>
    </div>

    <!-- List -->
    <div class="card">
      <p class="section-label">Accounts</p>
      <div v-if="users.length" class="user-list">
        <div v-for="u in users" :key="u.id" class="user-row">
          <div class="user-meta">
            <span class="user-name">
              {{ u.username }}
              <span v-if="u.username === session.username" class="you">you</span>
            </span>
            <span class="user-sub">created {{ formatDate(u.createdAt) }}</span>
          </div>
          <div class="user-actions">
            <select
              class="field-select sm"
              :value="u.role"
              :disabled="busy"
              @change="onRole(u, $event.target.value)"
            >
              <option value="editor">Editor</option>
              <option value="admin">Admin</option>
            </select>
            <button class="link-btn" :disabled="busy" @click="onResetPassword(u)">Reset password</button>
            <button
              class="link-btn danger"
              :disabled="busy || u.username === session.username"
              :title="u.username === session.username ? 'You cannot delete your own account' : ''"
              @click="onDelete(u)"
            >Delete</button>
          </div>
        </div>
      </div>
      <div v-else class="empty">No accounts yet.</div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { api } from '../api.js'
import { session } from '../stores/useAppStore.js'

const users = ref([])
const form = reactive({ username: '', password: '', role: 'editor' })
const busy = ref(false)
const msg = ref(null)

function formatDate(s) {
  if (!s) return '—'
  try { return new Date(s).toLocaleDateString() } catch { return '—' }
}

async function load() {
  try { users.value = (await api.listUsers()).users || [] }
  catch (e) { msg.value = { type: 'err', text: e.message || 'Failed to load' } }
}
onMounted(load)

async function onCreate() {
  msg.value = null; busy.value = true
  try {
    await api.createUser({ username: form.username.trim(), password: form.password, role: form.role })
    msg.value = { type: 'ok', text: `User "${form.username.trim().toLowerCase()}" created.` }
    form.username = ''; form.password = ''; form.role = 'editor'
    await load()
  } catch (e) { msg.value = { type: 'err', text: e.message || 'Create failed' } }
  busy.value = false
}

async function onRole(u, role) {
  if (role === u.role) return
  msg.value = null; busy.value = true
  try { await api.setUserRole(u.id, role); await load() }
  catch (e) { msg.value = { type: 'err', text: e.message || 'Could not change role' }; await load() }
  busy.value = false
}

async function onResetPassword(u) {
  const pw = window.prompt(`Set a new password for "${u.username}":`)
  if (!pw) return
  msg.value = null; busy.value = true
  try { await api.setUserPassword(u.id, pw); msg.value = { type: 'ok', text: `Password updated for "${u.username}".` } }
  catch (e) { msg.value = { type: 'err', text: e.message || 'Reset failed' } }
  busy.value = false
}

async function onDelete(u) {
  if (!confirm(`Delete user "${u.username}" and their entire workspace? This cannot be undone.`)) return
  msg.value = null; busy.value = true
  try { await api.deleteUser(u.id); msg.value = { type: 'ok', text: `User "${u.username}" deleted.` }; await load() }
  catch (e) { msg.value = { type: 'err', text: e.message || 'Delete failed' } }
  busy.value = false
}
</script>

<style scoped>
.user-new { display: flex; gap: 8px; margin: 10px 0 0; flex-wrap: wrap; }
.field-input { flex: 1; min-width: 120px; padding: 7px 10px; font-size: 13px; color: var(--clr-text);
  background: var(--clr-surface); border: 1px solid var(--clr-border-light); border-radius: var(--r-md); }
.field-select { padding: 7px 10px; font-size: 13px; color: var(--clr-text);
  background: var(--clr-surface); border: 1px solid var(--clr-border-light); border-radius: var(--r-md); }
.field-select.sm { padding: 5px 8px; font-size: 12px; }
.user-list { display: flex; flex-direction: column; gap: 6px; }
.user-row { display: flex; align-items: center; justify-content: space-between; gap: 10px;
  border: 1px solid var(--clr-border-light); border-radius: var(--r-md); padding: 8px 12px; }
.user-name { font-size: 13.5px; font-weight: 600; color: var(--clr-text); }
.you { display: inline-block; margin-left: 6px; padding: 1px 6px; font-size: 10.5px; font-weight: 600;
  color: var(--clr-accent); background: rgba(0,113,227,0.10); border-radius: 999px; vertical-align: middle; }
.user-sub { display: block; font-size: 11.5px; color: var(--clr-text-3); margin-top: 1px; }
.user-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
.btn-add { padding: 8px 14px; font-size: 13px; font-weight: 600; white-space: nowrap;
  color: var(--clr-accent); background: rgba(0,113,227,0.08); border-radius: var(--r-md); transition: background 0.15s; }
.btn-add:hover:not(:disabled) { background: rgba(0,113,227,0.14); }
.btn-add:disabled { opacity: 0.4; cursor: not-allowed; }
.link-btn { background: none; font-size: 12px; font-weight: 600; color: var(--clr-accent); padding: 4px 6px; border-radius: var(--r-sm); }
.link-btn.danger { color: var(--clr-danger); }
.link-btn:hover:not(:disabled) { text-decoration: underline; }
.link-btn:disabled { opacity: 0.4; cursor: not-allowed; }
/* .empty + .data-msg + .card-hint come from the shared Settings template in style.css. */
</style>
