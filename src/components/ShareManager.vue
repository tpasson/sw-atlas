<template>
  <div class="card">
    <p class="section-label">Shared schedules</p>
    <p class="card-hint">
      This is <strong>not</strong> the same as “Make this plan public” above. Here you offer a
      curated <strong>slice</strong> of your plan that other people <strong>subscribe</strong> to —
      your milestones then appear <em>inside their own timeline</em>, read-only and kept in sync
      (this is how a shared “team” account aggregates several people's milestones).
      Use <strong>List on server</strong> so other users here can find &amp; subscribe to it; use
      <strong>Links</strong> for someone on a different ATLAS instance.
    </p>

    <!-- existing scopes -->
    <div v-if="scopes.length" class="scope-list">
      <div v-for="sc in scopes" :key="sc.id" class="scope">
        <div class="scope-head">
          <div class="scope-meta">
            <span class="scope-name">{{ sc.name }}</span>
            <span class="scope-sub">
              {{ sc.detailLevel === 'full' ? 'full detail' : 'timing only' }}
              · {{ sc.lanes.length }} area{{ sc.lanes.length === 1 ? '' : 's' }}
              · {{ sharedCount(sc) }} milestone{{ sharedCount(sc) === 1 ? '' : 's' }}
              · {{ sc.tokenCount }} active link(s)
            </span>
          </div>
          <div class="scope-actions">
            <button
              class="link-btn"
              :class="{ on: sc.published }"
              :title="sc.published ? 'Listed — other users on this server can subscribe to this slice. Click to unlist.' : 'List this slice so other users on this server can find and subscribe to it'"
              @click="onTogglePublish(sc)"
            >{{ sc.published ? 'Listed ✓' : 'List on server' }}</button>
            <button class="link-btn" @click="toggleTokens(sc)">{{ openScope === sc.id ? 'Hide links' : 'Links' }}</button>
            <button class="link-btn danger" @click="onDeleteScope(sc)">Delete</button>
          </div>
        </div>

        <!-- tokens / subscribe links for this scope -->
        <div v-if="openScope === sc.id" class="tokens">
          <div v-if="newSecret" class="secret-box">
            <p class="secret-title">Subscribe link — copy it now, it won't be shown again:</p>
            <code class="secret-code">{{ newSecret.code }}</code>
            <div class="secret-actions">
              <button class="btn-add" @click="copy(newSecret.code)">{{ copied ? 'Copied ✓' : 'Copy link' }}</button>
              <span class="secret-raw">token: <code>{{ newSecret.token }}</code></span>
            </div>
          </div>

          <div v-if="tokens.length" class="token-list">
            <div v-for="t in tokens" :key="t.id" class="token-row" :class="{ revoked: t.revoked }">
              <span class="token-label">{{ t.label || '(no label)' }}</span>
              <span class="token-sub">
                {{ t.revoked ? 'revoked' : (t.lastAccessedAt ? 'last sync ' + fmt(t.lastAccessedAt) : 'never synced') }}
              </span>
              <button v-if="!t.revoked" class="link-btn danger" @click="onRevoke(t)">Revoke</button>
            </div>
          </div>
          <div v-else class="empty-sm">No links yet.</div>

          <div class="token-new">
            <input v-model="newLabel" class="field-input sm" placeholder="Who is this link for? (e.g. Team B – Bob)" @keyup.enter="onCreateToken(sc)" />
            <button class="btn-add" :disabled="busy" @click="onCreateToken(sc)">Create subscribe link</button>
          </div>
        </div>
      </div>
    </div>
    <div v-else class="empty">No shared schedules yet — create one below.</div>

    <!-- new scope -->
    <div class="new-scope">
      <p class="section-label" style="margin-top: 18px">New shared schedule</p>
      <div class="ns-row">
        <input v-model="form.name" class="field-input" placeholder="Name (e.g. External Q3 view)" />
        <select v-model="form.detailLevel" class="field-input sm">
          <option value="timing">Timing only</option>
          <option value="full">Full detail</option>
        </select>
      </div>

      <p class="card-hint">
        Check a whole area to share it (incl. items added later), or expand and pick
        individual milestones. Unchecking a milestone under a checked area excludes it.
      </p>

      <div class="tree">
        <div v-for="sw in swimlanes" :key="sw.id" class="tree-lane">
          <label class="tree-row">
            <input type="checkbox" :checked="laneChecked.has(sw.id)"
                   :indeterminate.prop="laneState(sw.id) === 'some'"
                   @change="toggleLane(sw.id)" />
            <span class="dot" :style="{ background: sw.color }"></span>
            <span class="tree-name">{{ sw.name }}</span>
            <button class="link-btn" @click.prevent="toggleExpand(sw.id)">{{ expanded.has(sw.id) ? '−' : '+' }}</button>
          </label>
          <div v-if="expanded.has(sw.id)" class="tree-items">
            <label v-for="m in itemsOf(sw.id)" :key="m.id" class="tree-row sub">
              <input type="checkbox" :checked="isItemChecked(m)" @change="toggleItem(m)" />
              <span class="tree-name">{{ m.title }}</span>
            </label>
            <div v-if="!itemsOf(sw.id).length" class="empty-sm">No milestones in this area.</div>
          </div>
        </div>
      </div>

      <button class="btn-add" :disabled="busy || !canCreate" @click="onCreateScope">Create shared schedule</button>
      <p v-if="error" class="data-msg err">{{ error }}</p>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, computed, onMounted } from 'vue'
import { store } from '../stores/useAppStore.js'
import { api } from '../api.js'

// Synced/mirrored lanes are read-only references — they can't be re-shared.
const swimlanes = computed(() => store.swimlanes.filter(sw => !sw.sourceSystem))
const itemsOf = (laneId) => store.milestones.filter(m => m.swimlaneId === laneId)

const scopes = ref([])
const openScope = ref(null)
const tokens = ref([])
const newSecret = ref(null)
const newLabel = ref('')
const copied = ref(false)
const busy = ref(false)
const error = ref(null)

const expanded = reactive(new Set())
const laneChecked = reactive(new Set())
const itemChecked = reactive(new Set())
const itemExcluded = reactive(new Set())

const form = reactive({ name: '', detailLevel: 'timing' })
const canCreate = computed(() =>
  form.name.trim() && (laneChecked.size > 0 || itemChecked.size > 0))

function fmt(s) {
  try { return new Date(s).toLocaleString() } catch { return s }
}

// How many milestones a scope actually shares: everything in its whole-area
// includes plus its explicitly-picked items, minus any excluded ones. (Mirrors
// the backend's ResolveScopePlan, so a whole-area scope no longer reads "0 items".)
function sharedCount(sc) {
  const lanes = new Set(sc.lanes || [])
  const picked = new Set(sc.items || [])
  const excluded = new Set(sc.excludes || [])
  let n = 0
  for (const m of store.milestones) {
    if (excluded.has(m.id)) continue
    if (lanes.has(m.swimlaneId) || picked.has(m.id)) n++
  }
  return n
}

async function load() {
  try {
    const r = await api.listShareScopes()
    scopes.value = r.scopes || []
  } catch (e) { error.value = e.message || 'Failed to load' }
}
onMounted(load)

// ── selection tree ──────────────────────────────────────────────────────────
function toggleExpand(id) { expanded.has(id) ? expanded.delete(id) : expanded.add(id) }

function toggleLane(id) {
  if (laneChecked.has(id)) {
    laneChecked.delete(id)
    for (const m of itemsOf(id)) itemExcluded.delete(m.id)
  } else {
    laneChecked.add(id)
    for (const m of itemsOf(id)) { itemChecked.delete(m.id); itemExcluded.delete(m.id) }
  }
}
function isItemChecked(m) {
  return laneChecked.has(m.swimlaneId) ? !itemExcluded.has(m.id) : itemChecked.has(m.id)
}
function toggleItem(m) {
  if (laneChecked.has(m.swimlaneId)) {
    itemExcluded.has(m.id) ? itemExcluded.delete(m.id) : itemExcluded.add(m.id)
  } else {
    itemChecked.has(m.id) ? itemChecked.delete(m.id) : itemChecked.add(m.id)
  }
}
function laneState(id) {
  if (laneChecked.has(id)) {
    return itemsOf(id).some(m => itemExcluded.has(m.id)) ? 'some' : 'all'
  }
  return itemsOf(id).some(m => itemChecked.has(m.id)) ? 'some' : 'none'
}

function resetForm() {
  form.name = ''; form.detailLevel = 'timing'
  laneChecked.clear(); itemChecked.clear(); itemExcluded.clear()
}

// ── actions ───────────────────────────────────────────────────────────────
async function onCreateScope() {
  error.value = null; busy.value = true
  try {
    await api.createShareScope({
      name: form.name.trim(),
      detailLevel: form.detailLevel,
      lanes: [...laneChecked],
      items: [...itemChecked],
      excludes: [...itemExcluded],
    })
    resetForm()
    await load()
  } catch (e) { error.value = e.message || 'Create failed' }
  busy.value = false
}

async function onDeleteScope(sc) {
  if (!confirm(`Delete shared schedule "${sc.name}"? Existing subscribe links stop working.`)) return
  try {
    await api.deleteShareScope(sc.id)
    if (openScope.value === sc.id) openScope.value = null
    await load()
  } catch (e) { error.value = e.message || 'Delete failed' }
}

async function onTogglePublish(sc) {
  try {
    await api.setShareScopePublished(sc.id, !sc.published)
    sc.published = !sc.published
  } catch (e) { error.value = e.message || 'Could not change publish state' }
}

async function toggleTokens(sc) {
  newSecret.value = null
  if (openScope.value === sc.id) { openScope.value = null; return }
  openScope.value = sc.id
  await loadTokens(sc.id)
}
async function loadTokens(scopeId) {
  try {
    const r = await api.listShareTokens(scopeId)
    tokens.value = r.tokens || []
  } catch (e) { error.value = e.message || 'Failed to load links' }
}

async function onCreateToken(sc) {
  error.value = null; busy.value = true
  try {
    const r = await api.createShareToken(sc.id, newLabel.value.trim())
    const code = btoa(JSON.stringify({ u: location.origin, t: r.secret, n: sc.name }))
    newSecret.value = { code, token: r.secret }
    newLabel.value = ''
    await loadTokens(sc.id)
    await load() // refresh token counts
  } catch (e) { error.value = e.message || 'Create link failed' }
  busy.value = false
}

async function onRevoke(t) {
  try { await api.revokeShareToken(t.id); await loadTokens(openScope.value); await load() }
  catch (e) { error.value = e.message || 'Revoke failed' }
}

async function copy(text) {
  try { await navigator.clipboard.writeText(text); copied.value = true; setTimeout(() => (copied.value = false), 1500) }
  catch { /* clipboard unavailable */ }
}
</script>

<style scoped>
.scope-list { display: flex; flex-direction: column; gap: 8px; margin-top: 10px; }
.scope { border: 1px solid var(--clr-border-light); border-radius: var(--r-md); padding: 10px 12px; }
.scope-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.scope-name { font-size: 13.5px; font-weight: 600; color: var(--clr-text); }
.scope-sub { display: block; font-size: 11.5px; color: var(--clr-text-3); margin-top: 1px; }
.scope-actions { display: flex; gap: 8px; flex-shrink: 0; }
.btn-add {
  padding: 8px 14px; font-size: 13px; font-weight: 600; white-space: nowrap;
  color: var(--clr-accent); background: rgba(0,113,227,0.08);
  border-radius: var(--r-md); transition: background 0.15s;
}
.btn-add:hover:not(:disabled) { background: rgba(0,113,227,0.14); }
.btn-add:disabled { opacity: 0.4; cursor: not-allowed; }
.link-btn { background: none; font-size: 12px; font-weight: 600; color: var(--clr-accent); padding: 4px 6px; border-radius: var(--r-sm); }
.link-btn.danger { color: var(--clr-danger); }
.link-btn.on { color: var(--clr-success, #30A46C); }
.link-btn:hover:not(:disabled) { text-decoration: underline; }
.link-btn:disabled { opacity: 0.4; cursor: not-allowed; }

.tokens { margin-top: 10px; padding-top: 10px; border-top: 1px solid var(--clr-border-light); }
.token-list { display: flex; flex-direction: column; gap: 4px; margin-bottom: 8px; }
.token-row { display: flex; align-items: center; gap: 8px; font-size: 12.5px; }
.token-row.revoked { opacity: 0.5; }
.token-label { font-weight: 600; color: var(--clr-text); }
.token-sub { color: var(--clr-text-3); flex: 1; }
.token-new { display: flex; gap: 8px; }

.secret-box { background: var(--clr-surface-2); border-radius: var(--r-md); padding: 10px 12px; margin-bottom: 10px; }
.secret-title { font-size: 12px; font-weight: 600; color: var(--clr-text); margin-bottom: 6px; }
.secret-code { display: block; font-size: 11px; word-break: break-all; color: var(--clr-text-2); background: var(--clr-surface); padding: 6px 8px; border-radius: var(--r-sm); }
.secret-actions { display: flex; align-items: center; gap: 12px; margin-top: 8px; }
.secret-raw { font-size: 11px; color: var(--clr-text-3); }

.tree { margin: 8px 0 12px; max-height: 260px; overflow-y: auto; border: 1px solid var(--clr-border-light); border-radius: var(--r-md); padding: 6px; }
.tree-row { display: flex; align-items: center; gap: 8px; padding: 4px 6px; cursor: pointer; font-size: 13px; color: var(--clr-text); }
.tree-row.sub { padding-left: 26px; font-size: 12.5px; color: var(--clr-text-2); }
.tree-items { margin-bottom: 2px; }
.tree-name { flex: 1; }
.dot { width: 10px; height: 10px; border-radius: 3px; flex-shrink: 0; }

.ns-row { display: flex; gap: 8px; margin-bottom: 6px; }
.field-input {
  width: 100%; border: 1.5px solid var(--clr-border); border-radius: var(--r-md);
  padding: 9px 12px; font-size: 14px; color: var(--clr-text);
  background: var(--clr-surface); outline: none; transition: border-color 0.15s;
}
.field-input:focus { border-color: var(--clr-accent); }
.field-input.sm { width: auto; max-width: 160px; padding: 7px 10px; font-size: 13px; }
.empty-sm { font-size: 12.5px; color: var(--clr-text-3); padding: 4px 0; }
</style>
