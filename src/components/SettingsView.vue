<template>
  <!-- Full-page Settings: a section sidebar + content pane. The active section is
       remembered (store + URL), so switching to another view and back returns here.
       Workspace-config sections reuse ManageModal in embedded (no-modal) mode. -->
  <div class="sv">
    <div class="sv-bar"><component :is="scopeIcon" :size="17" class="sv-icon" /><h1 class="sv-title">{{ title }}</h1></div>
    <div class="sv-split">
      <aside class="sv-nav">
        <div v-for="(grp, gi) in groups" :key="grp.label || gi" class="sv-group">
          <div v-if="grp.label" class="sv-group-head">{{ grp.label }}</div>
          <button
            v-for="s in grp.items" :key="s.key"
            type="button" class="sv-link" :class="{ on: active === s.key }"
            @click="setSettingsSection(s.key)"
          ><component :is="s.icon" :size="15" class="sv-link-icon" /><span>{{ s.label }}</span></button>
        </div>
      </aside>

      <div class="sv-content">
        <ManageModal v-if="isWorkspaceSection" embedded :initial-tab="active" @close="() => {}" />
        <div v-else class="sv-pane">
          <AccountManager v-if="active === 'account'" />
          <div v-else-if="active === 'appearance'" class="card">
            <span class="section-label">Appearance</span>
            <div class="row-between">
              <div class="setting-info">
                <span class="setting-name">Dark mode</span>
                <span class="setting-desc">Per-browser preference — not shared with the workspace.</span>
              </div>
              <button type="button" class="sv-toggle" :class="{ on: isDark }" role="switch" :aria-checked="isDark" @click="toggleTheme"><span class="sv-knob"></span></button>
            </div>
          </div>
          <ServerSettings v-else-if="active === 'server'" />
          <UsersManager v-else-if="active === 'users'" />
          <DisplaySettings v-else-if="active === 'display'" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { SlidersHorizontal, Settings, Rows3, Boxes, Shapes, Workflow, GitCompare, Database, Share2, Users, CircleUser, Palette, Server, UserCog, Monitor } from 'lucide-vue-next'
import { ui, setSettingsSection, session, workspace, settings, toggleTheme, canEditWorkspace, canAdminWorkspace, useAppStore, PROJECT_SECTIONS } from '../stores/useAppStore.js'
import ManageModal from './ManageModal.vue'
import AccountManager from './AccountManager.vue'
import ServerSettings from './ServerSettings.vue'
import UsersManager from './UsersManager.vue'
import DisplaySettings from './DisplaySettings.vue'

// scope is fixed per view: 'project' (its own view) vs 'general' — so the two are
// genuinely separate (no project menus while configuring the server, and vice versa).
const props = defineProps({ scope: { type: String, default: 'project' } })

const isDemo = import.meta.env.VITE_DEMO
const canEdit = computed(() => isDemo || canEditWorkspace())
const canAdmin = computed(() => isDemo || canAdminWorkspace())
const isOwner = computed(() => isDemo || workspace.role === 'owner')
const siteAdmin = computed(() => session.role === 'admin')
const isDark = computed(() => settings.theme === 'dark')

const scope = computed(() => props.scope)
const scopeIcon = computed(() => (props.scope === 'project' ? SlidersHorizontal : Settings))
const projectName = computed(() => {
  const w = (workspace.myWorkspaces || []).find(x => x.slug === workspace.slug)
  return w?.name || workspace.slug || 'this project'
})
const title = computed(() => (scope.value === 'project' ? `Settings for project: ${projectName.value}` : 'Settings'))

const groups = computed(() => {
  const filt = (items) => items.filter(i => i.show)
  if (scope.value === 'project') {
    const ws = filt([
      { key: 'areas', label: 'Areas', icon: Rows3, show: canEdit.value },
      { key: 'groups', label: 'Groups', icon: Boxes, show: canEdit.value },
      { key: 'types', label: 'Types', icon: Shapes, show: canAdmin.value },
      { key: 'workflows', label: 'Workflows', icon: Workflow, show: canAdmin.value },
      { key: 'baselines', label: 'Baselines', icon: GitCompare, show: canEdit.value },
      { key: 'data', label: 'Data', icon: Database, show: canEdit.value },
      { key: 'sharing', label: 'Sharing', icon: Share2, show: canAdmin.value && !isDemo },
      { key: 'members', label: 'Members', icon: Users, show: isOwner.value && !isDemo },
    ])
    return ws.length ? [{ label: null, items: ws }] : []
  }
  const out = []
  const you = filt([
    { key: 'account', label: 'Account', icon: CircleUser, show: session.authenticated },
    { key: 'appearance', label: 'Appearance', icon: Palette, show: true },
  ])
  if (you.length) out.push({ label: 'You', items: you })
  const inst = filt([
    { key: 'server', label: 'Server', icon: Server, show: siteAdmin.value },
    { key: 'users', label: 'Users', icon: UserCog, show: siteAdmin.value },
    { key: 'display', label: 'Display', icon: Monitor, show: siteAdmin.value },
  ])
  if (inst.length) out.push({ label: 'Instance', items: inst })
  return out
})
const allKeys = computed(() => groups.value.flatMap(g => g.items.map(i => i.key)))
// Falls back to the first available section if the remembered one isn't allowed here.
const active = computed(() => (allKeys.value.includes(ui.settingsSection) ? ui.settingsSection : (allKeys.value[0] || 'appearance')))
const isWorkspaceSection = computed(() => PROJECT_SECTIONS.includes(active.value))

const { setView } = useAppStore()
onMounted(() => {
  // No sections available here for this user (e.g. a viewer deep-linking project
  // settings) → leave the settings view entirely.
  if (!allKeys.value.length) { setView('timeline'); return }
  if (!allKeys.value.includes(ui.settingsSection)) setSettingsSection(allKeys.value[0])
})
</script>

<style scoped>
.sv { flex: 1; min-height: 0; display: flex; flex-direction: column; }
.sv-bar { height: 56px; box-sizing: border-box; display: flex; align-items: center; padding: 0 24px; border-bottom: 1px solid var(--clr-border-light); flex-shrink: 0; }
.sv-icon { color: var(--clr-text-2); margin-right: 10px; flex-shrink: 0; }
.sv-title { font-size: 17px; font-weight: 700; color: var(--clr-text); }
.sv-split { flex: 1; min-height: 0; display: flex; }
.sv-nav { width: 220px; flex-shrink: 0; overflow-y: auto; padding: 14px 10px; border-right: 1px solid var(--clr-border-light); }
.sv-group { margin-bottom: 14px; }
.sv-group-head { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.6px; color: var(--clr-text-3); padding: 4px 10px; }
.sv-link { display: flex; align-items: center; gap: 9px; width: 100%; text-align: left; font-size: 13px; font-weight: 500; color: var(--clr-text-2); background: none; border-radius: var(--r-sm); padding: 7px 10px; }
.sv-link-icon { flex-shrink: 0; opacity: 0.85; }
.sv-link:hover { background: var(--clr-surface-2); color: var(--clr-text); }
.sv-link.on { background: rgba(0,113,227,0.1); color: var(--clr-accent); font-weight: 600; }
.sv-content { flex: 1; min-height: 0; display: flex; }
.sv-pane { flex: 1; min-height: 0; overflow-y: auto; padding: 20px 24px; display: flex; flex-direction: column; gap: 14px; }
.sv-toggle { width: 42px; height: 24px; border-radius: 100px; background: var(--clr-border); position: relative; transition: background 0.15s; flex-shrink: 0; }
.sv-toggle.on { background: var(--clr-accent); }
.sv-knob { position: absolute; top: 2px; left: 2px; width: 20px; height: 20px; border-radius: 50%; background: #fff; transition: transform 0.15s; box-shadow: 0 1px 2px rgba(0,0,0,0.2); }
.sv-toggle.on .sv-knob { transform: translateX(18px); }
</style>
