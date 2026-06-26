<template>
  <div class="app">
    <TheHeader
      :year="store.year"
      :zoom="zoom"
      :authenticated="session.authenticated"
      @prev-year="prevYear"
      @next-year="nextYear"
      @manage="manageOpen = true"
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

    <MilestoneTable
      v-else
      :zoom="zoom"
      :read-only="readOnly"
      @add-milestone="openAdd"
      @edit-milestone="openEdit"
    />

    <GroupLegend v-if="session.ready && !session.error" :read-only="readOnly" />

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
        @close="modal.show = false"
      />
    </Transition>

    <Transition name="modal">
      <ManageModal v-if="manageOpen" @close="manageOpen = false" />
    </Transition>

    <Transition name="modal">
      <LoginModal v-if="loginOpen" @close="loginOpen = false" />
    </Transition>

    <Transition name="modal">
      <AboutModal v-if="aboutOpen" @close="aboutOpen = false" />
    </Transition>
  </div>
</template>

<script setup>
import { reactive, ref, computed, onMounted } from 'vue'
import { useAppStore, store, session, workspace, baselines, initApp } from './stores/useAppStore.js'
import TheHeader from './components/TheHeader.vue'
import MilestoneTable from './components/MilestoneTable.vue'
import MilestoneModal from './components/MilestoneModal.vue'
import ManageModal from './components/ManageModal.vue'
import LoginModal from './components/LoginModal.vue'
import GroupLegend from './components/GroupLegend.vue'
import AboutModal from './components/AboutModal.vue'

const { prevYear, nextYear, logout } = useAppStore()

const manageOpen = ref(false)
const loginOpen = ref(false)
const aboutOpen = ref(false)
const zoom = ref(1)

// Editing is unlocked only when an authenticated user is viewing their OWN
// workspace — never on someone else's plan, and never while viewing a saved
// baseline (a historical snapshot, not the live/head plan).
const readOnly = computed(() => !session.authenticated || !workspace.isOwn || !!baselines.activeId)

onMounted(initApp)

async function retry() {
  await initApp()
}

async function onLogout() {
  manageOpen.value = false
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
})

function openAdd({ swimlane, subLane, month, date }) {
  Object.assign(modal, {
    show: true, mode: 'add', swimlane, subLane, month, year: store.year, date: date || null, milestone: null,
  })
}

function openEdit(milestone) {
  const swimlane = store.swimlanes.find(s => s.id === milestone.swimlaneId)
  const subLane = swimlane?.subLanes.find(s => s.id === milestone.subLaneId) ?? null
  Object.assign(modal, {
    show: true, mode: 'edit', swimlane, subLane,
    month: milestone.month, year: milestone.year, date: null, milestone,
  })
}
</script>

<style>
.app {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}
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
