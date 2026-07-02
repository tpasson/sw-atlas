<template>
  <div v-if="workspace.myWorkspaces.length" class="bl-dd proj-dd" ref="projRef">
    <button class="bl-select" :class="{ open: projOpen, guest: isForeign }" title="Switch plan" @click="toggleProj">
      <Eye v-if="isForeign" :size="13" class="bl-guest-ic" />
      <span class="bl-cur">{{ currentProjectName }}</span>
      <svg class="bl-chevron" width="11" height="11" viewBox="0 0 12 12" fill="none"><path d="M2.5 4.5L6 8l3.5-3.5" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/></svg>
    </button>
    <div v-if="projOpen" class="bl-menu proj-menu">
      <!-- Foreign plan: you're a guest here (not in your workspaces). -->
      <template v-if="isForeign">
        <div class="proj-sec">Viewing<span class="proj-sec-hint">a plan that isn't yours</span></div>
        <div class="bl-opt active proj-guest">
          <span class="proj-glyph proj-glyph-guest"><Eye :size="13" /></span>
          <span class="bl-opt-name">{{ currentPublic?.name || currentProjectName }}</span>
          <span class="bl-opt-sub">{{ currentPublic?.ownerName || 'Guest' }}</span>
        </div>
        <div class="proj-divider"></div>
      </template>

      <!-- Your personal space (your /{username} home plan). -->
      <div class="proj-sec">Your area</div>
      <button v-if="homeWs" class="bl-opt" :class="{ active: homeWs.slug === workspace.slug }" @click="goProject(homeWs.slug)">
        <span class="proj-glyph proj-glyph-home"><User :size="13" /></span>
        <span class="bl-opt-name">{{ homeWs.name }}</span>
        <span class="bl-opt-sub">Just you</span>
      </button>

      <!-- Collaborative projects — invite others to work together. -->
      <div class="proj-sec">Projects<span class="proj-sec-hint">invite people to collaborate</span></div>
      <div v-for="p in projects" :key="p.slug" class="proj-row" :class="{ active: p.slug === workspace.slug }">
        <button class="proj-main" @click="goProject(p.slug)">
          <span class="proj-glyph proj-glyph-team"><Users :size="13" /></span>
          <span class="bl-opt-name">{{ p.name }}</span>
        </button>
        <div class="proj-acts">
          <template v-if="p.role === 'owner'">
            <button class="proj-act-ic" title="Invite people" @click.stop="inviteProj(p)"><UserPlus :size="15" /></button>
            <button class="proj-act-ic" title="Rename" @click.stop="renameProj(p)"><Pencil :size="15" /></button>
            <button class="proj-act-ic danger" title="Delete project" @click.stop="deleteProj(p)"><Trash2 :size="15" /></button>
          </template>
          <button v-else class="proj-act-ic" title="Leave project" @click.stop="leaveProj(p)"><LogOut :size="15" /></button>
        </div>
      </div>
      <p v-if="!projects.length" class="proj-empty">No projects yet — create one to invite collaborators.</p>
      <button class="bl-opt proj-new" @click="newProject"><Plus :size="14" /> New project</button>

      <!-- Other public plans on this instance — read-only, hop between them. -->
      <template v-if="otherPublic.length">
        <div class="proj-divider"></div>
        <div class="proj-sec">Public plans<span class="proj-sec-hint">read-only</span></div>
        <button v-for="p in otherPublic" :key="p.slug" class="bl-opt" @click="goProject(p.slug)">
          <span class="proj-glyph proj-glyph-public"><Globe :size="13" /></span>
          <span class="bl-opt-name">{{ p.name || p.slug }}</span>
          <span class="bl-opt-sub">{{ p.ownerName }}</span>
        </button>
      </template>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { User, Users, UserPlus, Plus, Pencil, Trash2, LogOut, Eye, Globe } from 'lucide-vue-next'
import { workspace, createProject, loadMyWorkspaces } from '../stores/useAppStore.js'
import { api } from '../api.js'

const emit = defineEmits(['manage'])

const projOpen = ref(false)
const projRef = ref(null)

const currentProjectName = computed(() => {
  const cur = workspace.myWorkspaces.find(p => p.slug === workspace.slug)
  if (cur) return cur.name
  return workspace.slug || 'My plans'
})
const homeWs = computed(() => workspace.myWorkspaces.find(p => p.slug === workspace.ownSlug))
const projects = computed(() => workspace.myWorkspaces.filter(p => p.slug !== workspace.ownSlug))
const isForeign = computed(() => !!workspace.slug && !workspace.myWorkspaces.some(p => p.slug === workspace.slug))

const publicPlans = ref([])
async function loadPublicPlans() {
  try { publicPlans.value = (await api.listPublicWorkspaces()).workspaces || [] } catch { /* ignore */ }
}
const currentPublic = computed(() => publicPlans.value.find(p => p.slug === workspace.slug) || null)
const otherPublic = computed(() => {
  const mine = new Set(workspace.myWorkspaces.map(p => p.slug))
  return publicPlans.value.filter(p => !mine.has(p.slug) && p.slug !== workspace.slug)
})
function toggleProj() {
  projOpen.value = !projOpen.value
  if (projOpen.value) loadPublicPlans()
}

function goProject(slug) {
  projOpen.value = false
  window.location.assign('/' + encodeURIComponent(slug))
}
async function newProject() {
  projOpen.value = false
  const name = prompt('New project name:')
  if (!name || !name.trim()) return
  try { await createProject(name.trim()) } catch (e) { alert(e?.message || 'Could not create the project') }
}
// Per-project actions (each works by slug, so they run inline from the list).
function inviteProj(p) {
  projOpen.value = false
  if (p.slug === workspace.slug) { emit('manage', 'members'); return } // members panel of the open project
  goProject(p.slug) // otherwise switch into it to manage members
}
async function renameProj(p) {
  const name = prompt('Rename project:', p.name)
  if (!name || !name.trim()) return
  try { await api.renameProject(p.slug, name.trim()); await loadMyWorkspaces() } catch (e) { alert(e?.message || 'Rename failed') }
}
async function deleteProj(p) {
  if (!confirm(`Delete “${p.name}” and all of its data? This can’t be undone.`)) return
  try {
    await api.deleteProject(p.slug)
    if (p.slug === workspace.slug) window.location.assign('/')
    else await loadMyWorkspaces()
  } catch (e) { alert(e?.message || 'Could not delete the project') }
}
async function leaveProj(p) {
  if (!confirm('Leave this project? You will lose access until an owner re-invites you.')) return
  try {
    await api.leaveProject(p.slug)
    if (p.slug === workspace.slug) window.location.assign('/')
    else await loadMyWorkspaces()
  } catch (e) { alert(e?.message || 'Could not leave the project') }
}

function onDocClick(e) { if (projRef.value && !projRef.value.contains(e.target)) projOpen.value = false }
function onKeyDown(e) { if (e.key === 'Escape') projOpen.value = false }
onMounted(() => { document.addEventListener('click', onDocClick); document.addEventListener('keydown', onKeyDown) })
onUnmounted(() => { document.removeEventListener('click', onDocClick); document.removeEventListener('keydown', onKeyDown) })
</script>

<style scoped>
.bl-dd { position: relative; }
.bl-select {
  display: inline-flex; align-items: center; justify-content: space-between; gap: 8px;
  height: 32px; box-sizing: border-box; width: 210px; padding: 0 12px 0 14px;
  background: rgba(255,255,255,0.08); border: 1px solid rgba(255,255,255,0.12);
  color: #fff; border-radius: 100px; font-size: 13px; font-weight: 500; cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}
.bl-select:hover, .bl-select.open { background: rgba(255,255,255,0.14); border-color: rgba(255,255,255,0.22); }
.bl-select.guest { border-color: rgba(255,149,0,0.5); background: rgba(255,149,0,0.12); }
.bl-select.guest:hover, .bl-select.guest.open { background: rgba(255,149,0,0.2); }
.bl-guest-ic { flex-shrink: 0; color: #FF9F0A; margin-right: -2px; }
.bl-cur { overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.bl-chevron { flex-shrink: 0; opacity: 0.7; transition: transform 0.18s; }
.bl-select.open .bl-chevron { transform: rotate(180deg); }
.bl-menu {
  position: absolute; top: calc(100% + 6px); right: 0; left: auto;
  min-width: 320px; max-width: 400px; max-height: 360px; overflow-y: auto;
  background: var(--clr-surface); border: 1px solid var(--clr-border-light);
  border-radius: var(--r-lg); box-shadow: var(--sh-modal); padding: 6px; z-index: 200;
}
.bl-opt {
  display: flex; align-items: baseline; justify-content: space-between; gap: 12px;
  width: 100%; text-align: left; padding: 8px 10px; border-radius: 8px;
  background: none; cursor: pointer; transition: background 0.12s;
}
.bl-opt:hover { background: var(--clr-surface-2); }
.bl-opt.active { background: rgba(0,113,227,0.1); }
.bl-opt-name { font-size: 13px; font-weight: 500; color: var(--clr-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.bl-opt.active .bl-opt-name { color: var(--clr-accent); font-weight: 600; }
.bl-opt-sub { font-size: 11px; color: var(--clr-text-3); white-space: nowrap; flex-shrink: 0; }
.proj-menu .bl-opt { align-items: center; justify-content: flex-start; }
.proj-menu .bl-opt .bl-opt-sub { margin-left: auto; }
.proj-row { display: flex; align-items: center; gap: 4px; border-radius: 8px; padding-right: 4px; transition: background 0.12s; }
.proj-row:hover { background: var(--clr-surface-2); }
.proj-row.active { background: rgba(0,113,227,0.1); }
.proj-main { flex: 1; min-width: 0; display: flex; align-items: center; gap: 12px; background: none; cursor: pointer; padding: 8px 6px 8px 10px; text-align: left; }
.proj-row.active .bl-opt-name { color: var(--clr-accent); font-weight: 600; }
.proj-acts { display: flex; gap: 1px; flex-shrink: 0; }
.proj-act-ic { width: 27px; height: 27px; display: inline-flex; align-items: center; justify-content: center;
  border-radius: 6px; color: var(--clr-text-3); background: none; transition: background 0.12s, color 0.12s; }
.proj-act-ic:hover { background: var(--clr-border-light); color: var(--clr-text); }
.proj-act-ic.danger:hover { background: rgba(255,59,48,0.1); color: var(--clr-danger); }
.proj-sec { padding: 8px 10px 4px; font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-text-3); display: flex; align-items: baseline; gap: 8px; }
.proj-sec-hint { font-size: 9px; font-weight: 600; letter-spacing: 0.2px; text-transform: none; color: var(--clr-text-3); opacity: 0.75; }
.proj-glyph { width: 22px; height: 22px; border-radius: 6px; flex-shrink: 0; display: inline-flex; align-items: center; justify-content: center; }
.proj-glyph-home { background: rgba(0,113,227,0.12); color: var(--clr-accent); }
.proj-glyph-team { background: var(--clr-surface-2); color: var(--clr-text-2); }
.proj-glyph-guest { background: rgba(255,149,0,0.16); color: #E8890C; }
.proj-glyph-public { background: var(--clr-surface-2); color: var(--clr-text-3); }
.proj-guest { cursor: default; }
.proj-empty { padding: 2px 10px 6px; font-size: 12px; color: var(--clr-text-3); line-height: 1.45; }
.proj-divider { height: 1px; background: var(--clr-border-light); margin: 6px 4px; }
.proj-new { gap: 8px; font-size: 13px; color: var(--clr-accent); font-weight: 600; }
</style>
