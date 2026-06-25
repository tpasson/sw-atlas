<template>
  <div class="card">
    <p class="section-label">GitHub sources</p>
    <p class="card-hint">
      Pull a repository's releases, tags, issues and pull requests in as a read-only area,
      placed on the timeline by date. Each item links back to GitHub. Hide or reorder the
      area in the <strong>Areas</strong> tab. Public repos need no token.
    </p>

    <div class="gh-new">
      <input v-model="url" class="field-input" placeholder="https://github.com/owner/repo" @keyup.enter="onAdd" />

      <div class="gh-types">
        <label class="gh-chk"><input type="checkbox" v-model="inc.releases" /> Releases</label>
        <label class="gh-chk"><input type="checkbox" v-model="inc.tags" /> Tags</label>
        <label class="gh-chk"><input type="checkbox" v-model="inc.issues" /> Issues</label>
        <label class="gh-chk"><input type="checkbox" v-model="inc.prs" /> Pull requests</label>
      </div>

      <div class="gh-filters">
        <label class="gh-chk"><input type="checkbox" v-model="filters.stableOnly" /> Stable releases only</label>
        <label class="gh-sel">Issues/PRs
          <select v-model="filters.stateFilter">
            <option value="all">all</option>
            <option value="open">open</option>
            <option value="closed">closed</option>
          </select>
        </label>
        <label class="gh-sel">Since <input type="date" v-model="filters.sinceDate" class="gh-inline" /></label>
        <label class="gh-sel" title="Keep only the N most recent of each type (0 = all)">Max/type
          <input type="number" min="0" step="1" v-model.number="filters.maxPerType" class="gh-num" />
        </label>
      </div>

      <button v-if="!showToken" class="link-btn" @click="showToken = true">+ Add token (private repos)</button>
      <input v-else v-model="token" class="field-input" type="password" autocomplete="off"
             placeholder="GitHub personal access token (optional)" />

      <button class="btn-add" :disabled="busy || !url.trim()" @click="onAdd">
        {{ busy ? 'Connecting…' : 'Add repository' }}
      </button>
    </div>
    <p v-if="msg" class="data-msg" :class="msg.type">{{ msg.text }}</p>

    <div v-if="sources.length" class="gh-list">
      <div v-for="s in sources" :key="s.id" class="gh-row">
        <div class="gh-meta">
          <span class="gh-name">{{ s.owner }}/{{ s.repo }}</span>
          <span class="gh-kinds">{{ kinds(s) }}</span>
          <span class="gh-sub" :class="{ err: (s.lastStatus || '').startsWith('error') }">{{ statusText(s) }}</span>
        </div>
        <div class="gh-actions">
          <button class="link-btn" :disabled="busy" @click="onSync(s)">Sync now</button>
          <button class="link-btn danger" :disabled="busy" @click="onRemove(s)">Remove</button>
        </div>
      </div>
    </div>
    <div v-else class="empty">No repositories connected yet.</div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { api } from '../api.js'
import { loadPlan } from '../stores/useAppStore.js'

const sources = ref([])
const url = ref('')
const token = ref('')
const showToken = ref(false)
const inc = reactive({ releases: true, tags: false, issues: false, prs: false })
const filters = reactive({ stableOnly: false, stateFilter: 'all', sinceDate: '', maxPerType: 0 })
const busy = ref(false)
const msg = ref(null)

function kinds(s) {
  const k = []
  if (s.includeReleases) k.push('releases')
  if (s.includeTags) k.push('tags')
  if (s.includeIssues) k.push('issues')
  if (s.includePrs) k.push('PRs')
  return k.join(' · ')
}
function statusText(s) {
  const when = s.lastSyncedAt ? new Date(s.lastSyncedAt).toLocaleString() : 'never'
  return `${s.lastStatus || '—'} · ${when}`
}

async function load() {
  try { sources.value = (await api.listGitHubSources()).sources || [] }
  catch (e) { msg.value = { type: 'err', text: e.message || 'Failed to load' } }
}
onMounted(load)

async function onAdd() {
  msg.value = null; busy.value = true
  try {
    const s = await api.createGitHubSource({
      url: url.value.trim(), token: token.value.trim(),
      includeReleases: inc.releases, includeTags: inc.tags,
      includeIssues: inc.issues, includePrs: inc.prs,
      stableOnly: filters.stableOnly, stateFilter: filters.stateFilter,
      sinceDate: filters.sinceDate, maxPerType: Number(filters.maxPerType) || 0,
    })
    url.value = ''; token.value = ''; showToken.value = false
    await load(); await loadPlan()
    msg.value = (s.lastStatus || '').startsWith('ok')
      ? { type: 'ok', text: `Connected ${s.owner}/${s.repo} — ${s.lastStatus}.` }
      : { type: 'err', text: `Connected, but sync said: ${s.lastStatus}` }
  } catch (e) { msg.value = { type: 'err', text: e.message || 'Connect failed' } }
  busy.value = false
}

async function onSync(s) {
  msg.value = null; busy.value = true
  try {
    const r = await api.syncGitHubSource(s.id); await load(); await loadPlan()
    if ((r.lastStatus || '').startsWith('error')) msg.value = { type: 'err', text: r.lastStatus }
  } catch (e) { msg.value = { type: 'err', text: e.message || 'Sync failed' } }
  busy.value = false
}

async function onRemove(s) {
  if (!confirm(`Remove ${s.owner}/${s.repo}? Its mirrored area disappears.`)) return
  busy.value = true
  try { await api.deleteGitHubSource(s.id); await load(); await loadPlan() }
  catch (e) { msg.value = { type: 'err', text: e.message || 'Remove failed' } }
  busy.value = false
}
</script>

<style scoped>
.gh-new { display: flex; flex-direction: column; gap: 8px; margin: 10px 0; }
.field-input { padding: 7px 10px; font-size: 13px; color: var(--clr-text);
  background: var(--clr-surface); border: 1px solid var(--clr-border-light); border-radius: var(--r-md); }
.gh-types { display: flex; flex-wrap: wrap; gap: 6px 16px; }
.gh-chk { display: inline-flex; align-items: center; gap: 6px; font-size: 13px; color: var(--clr-text-2); cursor: pointer; }
.gh-chk input { accent-color: var(--clr-accent); }
.gh-filters { display: flex; flex-wrap: wrap; align-items: center; gap: 8px 16px; padding-top: 2px; }
.gh-sel { display: inline-flex; align-items: center; gap: 6px; font-size: 13px; color: var(--clr-text-2); }
.gh-sel select, .gh-inline, .gh-num {
  font-size: 12.5px; color: var(--clr-text); background: var(--clr-surface);
  border: 1px solid var(--clr-border-light); border-radius: var(--r-sm); padding: 3px 6px;
}
.gh-num { width: 56px; }
.gh-list { display: flex; flex-direction: column; gap: 6px; }
.gh-row { display: flex; align-items: center; justify-content: space-between; gap: 10px;
  border: 1px solid var(--clr-border-light); border-radius: var(--r-md); padding: 8px 12px; }
.gh-name { font-size: 13.5px; font-weight: 600; color: var(--clr-text); }
.gh-kinds { display: block; font-size: 11px; color: var(--clr-text-2); margin-top: 1px; }
.gh-sub { display: block; font-size: 11.5px; color: var(--clr-text-3); margin-top: 1px; }
.gh-sub.err { color: var(--clr-danger); }
.gh-actions { display: flex; gap: 8px; flex-shrink: 0; }
.btn-add {
  align-self: flex-start;
  padding: 8px 14px; font-size: 13px; font-weight: 600; white-space: nowrap;
  color: var(--clr-accent); background: rgba(0,113,227,0.08);
  border-radius: var(--r-md); transition: background 0.15s;
}
.btn-add:hover:not(:disabled) { background: rgba(0,113,227,0.14); }
.btn-add:disabled { opacity: 0.4; cursor: not-allowed; }
.link-btn { align-self: flex-start; background: none; font-size: 12px; font-weight: 600; color: var(--clr-accent); padding: 4px 6px; border-radius: var(--r-sm); }
.link-btn.danger { color: var(--clr-danger); }
.link-btn:hover:not(:disabled) { text-decoration: underline; }
.link-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.empty { font-size: 12.5px; color: var(--clr-text-3); padding: 8px 0; }
.data-msg { font-size: 13px; margin: 6px 0; }
.data-msg.ok { color: var(--clr-accent); }
.data-msg.err { color: var(--clr-danger); }
</style>
