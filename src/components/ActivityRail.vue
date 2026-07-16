<template>
  <!-- VS-Code-style activity rail: views on top, account/settings/theme bottom. -->
  <nav class="rail">
    <div class="rail-top">
      <button class="rail-btn" :class="{ on: !inPlan }" title="All plans" @click="goHome">
        <Home :size="20" />
      </button>
    </div>
    <div v-if="inPlan" class="rail-group rail-views">
      <button class="rail-btn" :class="{ on: store.view === 'timeline' }" title="Timeline" @click="setView('timeline')">
        <CalendarDays :size="20" />
      </button>
      <button class="rail-btn" :class="{ on: store.view === 'explorer' }" title="Explorer" @click="setView('explorer')">
        <LayoutGrid :size="20" />
      </button>
      <button class="rail-btn" :class="{ on: store.view === 'scm' }" title="Source Control" @click="setView('scm')">
        <GitPullRequest :size="20" />
      </button>
      <button v-if="session.authenticated" class="rail-btn" :class="{ on: store.view === 'cr' }" title="Change Requests" @click="setView('cr')">
        <ClipboardCheck :size="20" />
        <span v-if="pendingCRCount" class="rail-badge">{{ pendingCRCount > 9 ? '9+' : pendingCRCount }}</span>
      </button>
      <button v-if="canEdit" class="rail-btn" :class="{ on: inProject }" title="Project settings" @click="$emit('manage')">
        <SlidersHorizontal :size="20" />
      </button>
    </div>

    <div class="rail-group rail-bottom">
      <button class="rail-btn" :title="settings.theme === 'dark' ? 'Light mode' : 'Dark mode'" @click="toggleTheme">
        <Sun v-if="settings.theme === 'dark'" :size="19" />
        <Moon v-else :size="19" />
      </button>

      <button v-if="session.authenticated" class="rail-btn" :class="{ on: inGeneral }" title="Settings" @click="$emit('settings')">
        <Settings :size="19" />
      </button>

      <div v-if="session.authenticated" class="rail-user" ref="userRef">
        <button class="rail-av" :title="session.username" @click="userOpen = !userOpen">{{ initials }}</button>
        <div v-if="userOpen" class="rail-menu">
          <div class="rail-menu-name">{{ session.username }}</div>
          <button @click="userOpen = false; $emit('about')">About ATLAS</button>
          <button @click="userOpen = false; $emit('logout')">Log out</button>
        </div>
      </div>
      <button v-else class="rail-btn" title="Log in" @click="$emit('login')">
        <KeyRound :size="19" />
      </button>
    </div>
  </nav>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Home, CalendarDays, LayoutGrid, GitPullRequest, ClipboardCheck, Sun, Moon, Settings, SlidersHorizontal, KeyRound } from 'lucide-vue-next'
import { store, session, settings, workspace, toggleTheme, useAppStore, canEditWorkspace, pendingCRCount, goHomeView } from '../stores/useAppStore.js'

defineEmits(['manage', 'settings', 'admin', 'login', 'logout', 'about'])

const { setView } = useAppStore()
// The discovery directory (all plans) lives at the bare root — switch to it
// client-side (no full reload → no white flash).
function goHome() { goHomeView() }
const canEdit = computed(() => canEditWorkspace())
const inPlan = computed(() => workspace.mode === 'plan')
// The two settings entries are now separate views.
const inProject = computed(() => store.view === 'project-settings')
const inGeneral = computed(() => store.view === 'settings')
const initials = computed(() => (session.username || '?').trim().charAt(0).toUpperCase() || '?')

const userOpen = ref(false)
const userRef = ref(null)
function onDocClick(e) { if (userRef.value && !userRef.value.contains(e.target)) userOpen.value = false }
function onKey(e) { if (e.key === 'Escape') userOpen.value = false }
onMounted(() => { document.addEventListener('click', onDocClick); document.addEventListener('keydown', onKey) })
onUnmounted(() => { document.removeEventListener('click', onDocClick); document.removeEventListener('keydown', onKey) })
</script>

<style scoped>
.rail {
  width: 52px; flex-shrink: 0;
  background: var(--clr-header);
  display: flex; flex-direction: column; align-items: center;
  padding: 0; gap: 0;
  position: sticky; top: 0; height: 100vh;
  box-shadow: 1px 0 0 rgba(255,255,255,0.06);
  z-index: 101;
}
/* Home sits in a zone the height of the header (so it lines up vertically with the
   brand), but the rail stays a clean icon column — no horizontal divider that
   would clash with the content's header/table lines. */
.rail-top {
  height: 64px; flex-shrink: 0; width: 100%;
  display: flex; align-items: center; justify-content: center;
}
.rail-group { display: flex; flex-direction: column; align-items: center; gap: 6px; }
.rail-views { padding-top: 4px; }
.rail-bottom { margin-top: auto; padding-bottom: 10px; }

.rail-btn {
  width: 40px; height: 40px; border-radius: 10px;
  display: inline-flex; align-items: center; justify-content: center;
  color: rgba(255,255,255,0.6); background: transparent; position: relative;
  transition: background 0.15s, color 0.15s;
}
.rail-btn:hover { background: rgba(255,255,255,0.10); color: #fff; }
.rail-btn.on { color: #fff; background: rgba(255,255,255,0.14); }
.rail-btn.on::before {
  content: ''; position: absolute; left: -10px; top: 9px; bottom: 9px;
  width: 3px; border-radius: 2px; background: var(--clr-accent);
}
.rail-badge {
  position: absolute; top: 2px; right: 2px;
  min-width: 15px; height: 15px; padding: 0 3px; box-sizing: border-box;
  border-radius: 100px; background: #FF3B30; color: #fff;
  font-size: 9px; font-weight: 700; line-height: 15px; text-align: center;
}

.rail-user { position: relative; }
.rail-av {
  width: 32px; height: 32px; border-radius: 50%;
  background: rgba(255,255,255,0.16); color: #fff;
  font-size: 12px; font-weight: 700;
  display: inline-flex; align-items: center; justify-content: center;
  transition: background 0.15s;
}
.rail-av:hover { background: rgba(255,255,255,0.26); }
.rail-menu {
  position: absolute; left: 46px; bottom: 0;
  background: var(--clr-surface); border: 1px solid var(--clr-border);
  border-radius: var(--r-md); box-shadow: var(--sh-lg);
  min-width: 168px; padding: 4px; z-index: 200;
}
.rail-menu-name {
  padding: 8px 10px; font-size: 12px; font-weight: 700; color: var(--clr-text-3);
  border-bottom: 1px solid var(--clr-border-light); margin-bottom: 4px;
}
.rail-menu button {
  display: block; width: 100%; text-align: left;
  padding: 7px 10px; font-size: 13px; color: var(--clr-text);
  border-radius: var(--r-sm); background: transparent;
}
.rail-menu button:hover { background: var(--clr-bg); }
</style>
