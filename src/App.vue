<template>
  <div class="app">
    <div class="app-shell">
    <ActivityRail
      @manage="openManage"
      @settings="openGeneralSettings()"
      @about="aboutOpen = true"
      @login="loginOpen = true"
      @logout="onLogout"
    />
    <div class="app-main">
    <LandingPage v-if="workspace.mode === 'landing'" @about="aboutOpen = true" />
    <template v-else>
    <TheHeader
      :year="store.year"
      :zoom="zoom"
      :authenticated="session.authenticated"
      @prev-year="prevYear"
      @next-year="nextYear"
      @manage="openManage"
      @about="aboutOpen = true"
      @login="loginOpen = true"
      @logout="onLogout"
      @zoom-in="zoom = Math.min(2, Math.round((zoom + 0.1) * 10) / 10)"
      @zoom-out="zoom = Math.max(0.6, Math.round((zoom - 0.1) * 10) / 10)"
    />

    <div v-if="!session.ready" class="app-state">
      <div class="app-state-card"><p>Loading…</p></div>
    </div>

    <div v-else-if="session.error" class="app-state">
      <div class="app-state-card">
        <template v-if="session.error === 'auth-required'">
          <h2>ATLAS is private</h2>
          <p>Public access is currently disabled. Log in as an editor to view the plan.</p>
          <button class="state-btn" @click="loginOpen = true">Log in</button>
        </template>
        <template v-else-if="session.error === 'private'">
          <h2>This plan is private</h2>
          <p>The owner hasn’t made <strong>{{ workspace.slug }}</strong> public.<span v-if="!session.authenticated"> Log in if it’s yours.</span></p>
          <button v-if="!session.authenticated" class="state-btn" @click="loginOpen = true">Log in</button>
        </template>
        <template v-else-if="session.error === 'not-found'">
          <h2>No plan here</h2>
          <p>There’s no workspace at <strong>/{{ workspace.slug }}</strong>.</p>
        </template>
        <template v-else>
          <h2>Couldn’t load ATLAS</h2>
          <p>{{ session.error }}</p>
          <button class="state-btn" @click="retry">Retry</button>
        </template>
      </div>
    </div>

    <template v-else>
      <ExplorerView
        v-if="store.view === 'explorer'"
        :read-only="readOnly"
        @edit="openEdit"
        @add="openAddType"
      />
      <SourceControlView
        v-else-if="store.view === 'scm'"
        :read-only="readOnly"
      />
      <ChangeRequestsView
        v-else-if="store.view === 'cr'"
        @propose-new="openProposeNew"
      />
      <SettingsView v-else-if="store.view === 'project-settings'" key="ps" scope="project" />
      <SettingsView v-else-if="store.view === 'settings'" key="gs" scope="general" />
      <MilestoneTable
        v-else
        :zoom="zoom"
        :read-only="readOnly"
        @add-milestone="openAdd"
        @edit-milestone="openEdit"
        @show-history="openEdit($event, 'history')"
        @manage="openManage"
      />
    </template>

    <FacetFilter v-if="session.ready && !session.error && store.view === 'timeline'" />
    <GroupLegend v-if="session.ready && !session.error && store.view === 'timeline'" :read-only="readOnly" />
    </template>
    </div>
    </div>

    <Transition name="modal">
      <MilestoneModal
        v-if="modal.show"
        :mode="modal.mode"
        :swimlane="modal.swimlane"
        :sub-lane="modal.subLane"
        :month="modal.month"
        :year="modal.year"
        :date="modal.date"
        :milestone="modal.milestone"
        :initial-type="modal.initialType"
        :initial-tab="modal.initialTab"
        :propose-mode="modal.proposeMode"
        @close="modal.show = false"
      />
    </Transition>


    <Transition name="modal">
      <LoginModal v-if="loginOpen" @close="loginOpen = false" />
    </Transition>

    <Transition name="modal">
      <AboutModal v-if="aboutOpen" @close="aboutOpen = false" />
    </Transition>

    <UserProfilePopover />
  </div>
</template>

<script setup>
import { reactive, ref, computed, onMounted, watch } from 'vue'
import { useAppStore, store, session, workspace, baselines, initApp, canEditWorkspace, openSettings, openProjectSettings, openGeneralSettings } from './stores/useAppStore.js'
import TheHeader from './components/TheHeader.vue'
import ActivityRail from './components/ActivityRail.vue'
import MilestoneTable from './components/MilestoneTable.vue'
import ExplorerView from './components/ExplorerView.vue'
import SourceControlView from './components/SourceControlView.vue'
import ChangeRequestsView from './components/ChangeRequestsView.vue'
import MilestoneModal from './components/MilestoneModal.vue'
import SettingsView from './components/SettingsView.vue'
import LoginModal from './components/LoginModal.vue'
import GroupLegend from './components/GroupLegend.vue'
import FacetFilter from './components/FacetFilter.vue'
import AboutModal from './components/AboutModal.vue'
import LandingPage from './components/LandingPage.vue'
import UserProfilePopover from './components/UserProfilePopover.vue'

const { prevYear, nextYear, logout } = useAppStore()

// The rail's "Project settings" gear (no arg) reopens the last project section;
// callers passing a specific tab (e.g. "Invite people…" → members) jump there.
function openManage(tab) {
  if (typeof tab === 'string') openSettings(tab)
  else openProjectSettings()
}
const loginOpen = ref(false)
const aboutOpen = ref(false)
// Zoom level persists across reloads (clamped to the same 0.6–2 range the +/- use).
const ZOOM_KEY = 'atlas-zoom'
const zoom = ref((() => { const v = parseFloat(localStorage.getItem(ZOOM_KEY)); return v >= 0.6 && v <= 2 ? v : 1 })())
watch(zoom, (v) => { try { localStorage.setItem(ZOOM_KEY, String(v)) } catch { /* ignore */ } })

// Editing is unlocked only when an authenticated user is viewing their OWN
// workspace — never on someone else's plan, and never while viewing a saved
// baseline (a historical snapshot, not the live/head plan).
const readOnly = computed(() => !canEditWorkspace() || !!baselines.activeId)

onMounted(initApp)

async function retry() {
  await initApp()
}

async function onLogout() {
  await logout()
}

const modal = reactive({
  show: false,
  mode: 'add',
  swimlane: null,
  subLane: null,
  month: 1,
  year: store.year,
  date: null,
  milestone: null,
  initialType: '',
  initialTab: 'details',
  proposeMode: false,
})

function openAdd({ swimlane, subLane, month, date }) {
  Object.assign(modal, {
    show: true, mode: 'add', swimlane, subLane, month, year: store.year, date: date || null, milestone: null, initialType: '', initialTab: 'details', proposeMode: false,
  })
}

// Add an off-timeline artifact from the Explorer (no lane; type preselected).
function openAddType(type) {
  Object.assign(modal, {
    show: true, mode: 'add', swimlane: null, subLane: null,
    month: new Date().getMonth() + 1, year: store.year, date: null, milestone: null, initialType: type.key, initialTab: 'details', proposeMode: false,
  })
}

// Propose a brand-new item (a create change request) — opens the add dialog
// already in propose mode; the proposer picks the target Area.
function openProposeNew() {
  Object.assign(modal, {
    show: true, mode: 'add', swimlane: null, subLane: null,
    month: new Date().getMonth() + 1, year: store.year, date: null, milestone: null, initialType: '', initialTab: 'details', proposeMode: true,
  })
}

// Open the item dialog. `tab` lets us land straight on a tab (e.g. "history"
// from the timeline's version chip) so history is shown IN the item's window.
function openEdit(milestone, tab = 'details') {
  const swimlane = store.swimlanes.find(s => s.id === milestone.swimlaneId)
  const subLane = swimlane?.subLanes.find(s => s.id === milestone.subLaneId) ?? null
  Object.assign(modal, {
    show: true, mode: 'edit', swimlane, subLane,
    month: milestone.month, year: milestone.year, date: null, milestone, initialType: '', initialTab: tab, proposeMode: false,
  })
}
</script>

<style>
.app {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}
.app-shell { display: flex; min-height: 100vh; }
.app-main { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.app-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
}
.app-state-card { text-align: center; max-width: 380px; }
.app-state-card h2 { font-size: 18px; font-weight: 700; margin-bottom: 8px; color: var(--clr-text); }
.app-state-card p { font-size: 14px; color: var(--clr-text-2); margin-bottom: 16px; line-height: 1.5; }
.state-btn {
  padding: 9px 20px; font-size: 14px; font-weight: 600; color: #fff;
  background: var(--clr-accent); border-radius: var(--r-md); transition: background 0.15s;
}
.state-btn:hover { background: var(--clr-accent-hover); }
</style>
