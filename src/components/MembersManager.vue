<template>
  <div class="card-stack">
    <div class="card">
      <span class="section-label">Project</span>
      <div class="mm-invite">
        <input class="mm-in mm-name-in" v-model="projectName" placeholder="Project name" autocomplete="off" @keyup.enter="rename" />
        <button class="mm-btn" @click="rename">Rename</button>
      </div>
      <div class="row-between">
        <span class="setting-desc">Deleting removes the project and all of its data — this can't be undone.</span>
        <button class="mm-del" @click="del">Delete project</button>
      </div>
      <div v-if="projMsg" class="data-msg" :class="{ ok: projOk, err: !projOk }">{{ projMsg }}</div>
    </div>

    <div class="card">
      <span class="section-label">Members</span>
      <p class="card-hint">
        Invite people to collaborate on this project. <strong>Editors</strong> can change the plan;
        <strong>viewers</strong> can only read it (even when it's private). Only the owner manages members.
      </p>

      <div class="mm-invite">
        <input class="mm-in mm-name-in" v-model="inviteName" placeholder="username" autocomplete="off" @keyup.enter="invite" />
        <select class="mm-in mm-role" v-model="inviteRole">
          <option value="editor">Editor</option>
          <option value="viewer">Viewer</option>
        </select>
        <button class="mm-btn" @click="invite">Invite</button>
      </div>

      <div v-for="m in members" :key="m.userId" class="mm-row">
        <button type="button" class="mm-name" title="View profile" @click="openProfile(m, $event)">
          {{ personName(m) }}<span v-if="personName(m) !== m.username" class="mm-user">@{{ m.username }}</span>
        </button>
        <select class="mm-in mm-role" :value="m.role" @change="changeRole(m, $event.target.value)">
          <option value="owner">Owner</option>
          <option value="editor">Editor</option>
          <option value="viewer">Viewer</option>
        </select>
        <button class="mm-x" title="Remove from project" @click="remove(m)">×</button>
      </div>
      <div v-if="!members.length" class="empty">No members yet.</div>

      <div v-if="msg" class="data-msg" :class="{ ok: okMsg, err: !okMsg }">{{ msg }}</div>
      <p class="card-hint">Role changes take effect on the member's next page load.</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'
import { workspace, openProfile, personName } from '../stores/useAppStore.js'

const props = defineProps({ slug: { type: String, required: true } })

const projectName = ref(workspace.myWorkspaces.find(p => p.slug === props.slug)?.name || props.slug)
const projMsg = ref('')
const projOk = ref(false)

async function rename() {
  const name = projectName.value.trim()
  if (!name) return
  try {
    await api.renameProject(props.slug, name)
    const cur = workspace.myWorkspaces.find(p => p.slug === props.slug)
    if (cur) cur.name = name
    projOk.value = true; projMsg.value = 'Renamed.'
  } catch (e) { projOk.value = false; projMsg.value = e?.message || 'Rename failed' }
}
async function del() {
  if (!confirm(`Delete the project "${projectName.value}" and all of its data? This can't be undone.`)) return
  try {
    await api.deleteProject(props.slug)
    window.location.assign('/')
  } catch (e) { projOk.value = false; projMsg.value = e?.message || 'Delete failed' }
}

const members = ref([])
const inviteName = ref('')
const inviteRole = ref('editor')
const msg = ref('')
const okMsg = ref(false)

function ok(m) { okMsg.value = true; msg.value = m }
function fail(e) { okMsg.value = false; msg.value = e?.message || 'Something went wrong' }

async function load() {
  try { members.value = await api.listMembers(props.slug) } catch (e) { fail(e) }
}
onMounted(load)

async function invite() {
  const name = inviteName.value.trim()
  if (!name) return
  try {
    await api.inviteMember(props.slug, name, inviteRole.value)
    ok(`Invited ${name} as ${inviteRole.value}.`)
    inviteName.value = ''
    await load()
  } catch (e) { fail(e) }
}
async function changeRole(m, role) {
  if (role === m.role) return
  try { await api.setMemberRole(props.slug, m.userId, role); ok(`${m.username} is now ${role}.`) } catch (e) { fail(e) }
  await load()
}
async function remove(m) {
  if (!confirm(`Remove ${m.username} from this project?`)) return
  try { await api.removeMember(props.slug, m.userId); ok(`Removed ${m.username}.`); await load() } catch (e) { fail(e) }
}
</script>

<style scoped>
.mm-invite { display: flex; gap: 8px; align-items: center; }
.mm-row { display: flex; gap: 8px; align-items: center; padding: 6px 0; border-top: 1px solid var(--clr-border-light); }
.mm-name { flex: 1; min-width: 0; text-align: left; font-size: 14px; font-weight: 600; color: var(--clr-text);
  background: none; border: none; cursor: pointer; padding: 2px 4px; margin: -2px -4px; border-radius: 6px; transition: background 0.15s; }
.mm-name:hover { background: var(--clr-surface-2); }
.mm-user { margin-left: 6px; font-size: 12px; font-weight: 500; color: var(--clr-text-3); }
.mm-name-in { flex: 1; }
.mm-in {
  border: 1px solid var(--clr-border); border-radius: var(--r-sm);
  padding: 7px 10px; font-size: 13px; color: var(--clr-text); background: var(--clr-bg);
}
.mm-in:focus { outline: none; border-color: var(--clr-accent); }
.mm-role { width: 100px; }
.mm-btn { background: var(--clr-accent); color: #fff; border-radius: var(--r-md); padding: 7px 14px; font-weight: 600; font-size: 13px; }
.mm-btn:hover { background: var(--clr-accent-hover); }
.mm-del { background: rgba(255,59,48,0.1); color: var(--clr-danger); border-radius: var(--r-md); padding: 7px 14px; font-weight: 600; font-size: 13px; flex-shrink: 0; }
.mm-del:hover { background: rgba(255,59,48,0.18); }
.mm-x {
  width: 26px; height: 26px; border-radius: 50%; flex-shrink: 0;
  display: inline-flex; align-items: center; justify-content: center;
  font-size: 16px; line-height: 1; color: var(--clr-text-3); background: transparent;
}
.mm-x:hover { background: rgba(255,59,48,0.12); color: var(--clr-danger); }
</style>
