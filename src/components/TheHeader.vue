<template>
  <header class="header">
    <div class="header-inner">
      <div class="header-left">
        <button class="brand-btn" title="About ATLAS" @click="$emit('about')">
          <div class="brand-icon">
            <svg width="22" height="22" viewBox="0 0 22 22" fill="none">
              <rect x="2" y="2" width="8" height="8" rx="2" fill="white" opacity="0.9"/>
              <rect x="12" y="2" width="8" height="8" rx="2" fill="white" opacity="0.6"/>
              <rect x="2" y="12" width="8" height="8" rx="2" fill="white" opacity="0.6"/>
              <rect x="12" y="12" width="8" height="8" rx="2" fill="white" opacity="0.35"/>
            </svg>
          </div>
          <div class="brand-text">
            <span class="brand-title">ATLAS</span>
            <span class="brand-ver">v{{ version }}</span>
          </div>
        </button>

        <div class="today-chip">
          <span class="today-date">{{ todayLabel }}</span>
          <span class="today-cw">CW {{ todayWeek }}</span>
        </div>


        <div v-if="store.view === 'timeline'" class="year-nav">
          <button class="year-btn" :title="store.granularity === 'month' ? 'Previous month' : 'Previous year'" @click="store.granularity === 'month' ? prevMonth() : $emit('prev-year')">
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
              <path d="M9 11L5 7l4-4" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
          <span class="year-label">{{ store.granularity === 'month' ? MONTHS[store.viewMonth - 1] + ' ' + year : year }}</span>
          <button class="year-btn" :title="store.granularity === 'month' ? 'Next month' : 'Next year'" @click="store.granularity === 'month' ? nextMonth() : $emit('next-year')">
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
              <path d="M5 3l4 4-4 4" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
          <button
            class="year-btn gran-toggle"
            :title="store.granularity === 'month' ? 'Switch to year view' : 'Switch to month view (day columns)'"
            @click="setGranularity(store.granularity === 'month' ? 'year' : 'month')"
          >{{ store.granularity === 'month' ? 'Year' : 'Month' }}</button>
        </div>

        <div v-if="store.view === 'timeline'" class="zoom-nav">
          <button class="year-btn" :disabled="zoom <= 0.6" @click="$emit('zoom-out')">
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
              <path d="M2 6h8" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/>
            </svg>
          </button>
          <span class="year-label">{{ Math.round(zoom * 100) }}%</span>
          <button class="year-btn" :disabled="zoom >= 2" @click="$emit('zoom-in')">
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
              <path d="M6 2v8M2 6h8" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/>
            </svg>
          </button>
        </div>
      </div>

      <div class="header-center">
        <div class="baseline-ctl">
          <button class="year-btn" :disabled="!canPrevBaseline" title="Oldest baseline" @click="jumpBaseline(-1)">
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
              <path d="M6.5 11L2.5 7l4-4M11.5 11L7.5 7l4-4" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
          <button class="year-btn" :disabled="!canPrevBaseline" title="Older" @click="stepBaseline(-1)">
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
              <path d="M9 11L5 7l4-4" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
          <div class="bl-dd" ref="blRef">
            <button class="bl-select" :class="{ open: blOpen }" title="View a saved snapshot (the plan as it was then)" @click="blOpen = !blOpen">
              <span class="bl-cur">{{ currentLabel }}</span>
              <svg class="bl-chevron" width="11" height="11" viewBox="0 0 12 12" fill="none">
                <path d="M2.5 4.5L6 8l3.5-3.5" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </button>
            <div v-if="blOpen" class="bl-menu">
              <button class="bl-opt" :class="{ active: !baselines.activeId }" @click="pickBaseline(null)">
                <span class="bl-opt-name">Live</span>
                <span class="bl-opt-sub">current</span>
              </button>
              <button v-for="b in baselines.list" :key="b.id" class="bl-opt" :class="{ active: baselines.activeId === b.id }" @click="pickBaseline(b.id)">
                <span class="bl-opt-name">{{ b.name }}</span>
                <span class="bl-opt-sub">{{ b.itemCount }} items</span>
              </button>
            </div>
          </div>
          <button class="year-btn" :disabled="!canNextBaseline" title="Newer" @click="stepBaseline(1)">
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
              <path d="M5 3l4 4-4 4" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
          <button class="year-btn" :disabled="!canNextBaseline" title="Live (most current)" @click="jumpBaseline(1)">
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
              <path d="M2.5 3l4 4-4 4M7.5 3l4 4-4 4" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
        </div>
      </div>

      <div class="header-right">
        <span
          v-if="riskWarnings.length"
          class="risk-hdr"
          title="Milestones at risk (a blocker is scheduled later)"
          @mouseenter="hoverRisk = true"
          @mouseleave="hoverRisk = false"
        >
          <AlertTriangle :size="14" />
          {{ riskWarnings.length }}
          <div v-if="hoverRisk" class="risk-pop">
            <div class="risk-pop-title">At risk ({{ riskWarnings.length }})</div>
            <div v-for="w in riskWarnings" :key="w.item.id" class="risk-pop-row" @click.stop="focusRisk(w.item.id)">
              <span class="risk-pop-name">{{ w.item.title }}</span>
              <span class="risk-pop-sub">late: {{ w.lateDeps.map(d => d.title).join(', ') }}</span>
            </div>
          </div>
        </span>

        <span
          v-if="lateItems.length"
          class="risk-hdr"
          title="Overdue items (progress set, past their date)"
          @mouseenter="hoverLate = true"
          @mouseleave="hoverLate = false"
        >
          <Clock :size="14" />
          {{ lateItems.length }}
          <div v-if="hoverLate" class="risk-pop">
            <div class="risk-pop-title">Overdue ({{ lateItems.length }})</div>
            <div v-for="m in lateItems" :key="m.id" class="risk-pop-row" @click.stop="focusRisk(m.id)">
              <span class="risk-pop-name">{{ m.title }}</span>
              <span class="risk-pop-sub">{{ m.progress }}% done</span>
            </div>
          </div>
        </span>

        <span v-if="baselines.activeId" class="view-pill" title="Viewing a saved baseline — editing is disabled">
          <span class="view-dot"></span>
          Viewing baseline
        </span>

        <template v-if="authenticated">
          <!-- Project switcher -->
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
                  <span class="bl-opt-sub">{{ currentPublic?.ownerName || 'guest' }}</span>
                </div>
                <div class="proj-divider"></div>
              </template>

              <!-- Your personal space (your /{username} home plan). -->
              <div class="proj-sec">Your area</div>
              <button
                v-if="homeWs"
                class="bl-opt"
                :class="{ active: homeWs.slug === workspace.slug }"
                @click="goProject(homeWs.slug)"
              >
                <span class="proj-glyph proj-glyph-home"><User :size="13" /></span>
                <span class="bl-opt-name">{{ homeWs.name }}</span>
                <span class="bl-opt-sub">just you</span>
              </button>

              <!-- Collaborative projects — invite others to work together. -->
              <div class="proj-sec">Projects<span class="proj-sec-hint">invite people to collaborate</span></div>
              <button
                v-for="p in projects"
                :key="p.slug"
                class="bl-opt"
                :class="{ active: p.slug === workspace.slug }"
                @click="goProject(p.slug)"
              >
                <span class="proj-glyph proj-glyph-team"><Users :size="13" /></span>
                <span class="bl-opt-name">{{ p.name }}</span>
                <span class="bl-opt-sub">{{ p.role }}</span>
              </button>
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

              <!-- Manage the project you're currently in. -->
              <template v-if="currentIsProject">
                <div class="proj-divider"></div>
                <div class="proj-sec">Manage “{{ currentProjectName }}”</div>
                <template v-if="workspace.role === 'owner'">
                  <button class="bl-opt proj-action" @click="inviteToProject"><UserPlus :size="14" /> Invite people…</button>
                  <button class="bl-opt proj-action" @click="renameCurrent"><Pencil :size="14" /> Rename…</button>
                  <button class="bl-opt proj-del" @click="deleteCurrent"><Trash2 :size="14" /> Delete project</button>
                </template>
                <button v-else class="bl-opt proj-leave" @click="leaveCurrent"><LogOut :size="14" /> Leave project</button>
              </template>
            </div>
          </div>

          <!-- Editable workspace (owner/editor): status + save baseline. Settings,
               theme, account live in the left activity rail now. -->
          <template v-if="canEdit">
            <span v-if="!baselines.activeId" class="edit-pill"><span class="edit-dot"></span>Editing</span>
            <button v-if="!baselines.activeId" class="hdr-icon-btn" title="Save current plan as a baseline" @click="onSaveBaseline"><Bookmark :size="16" /></button>
          </template>
          <span v-else class="view-pill" title="You are viewing this plan read-only"><span class="view-dot"></span>Read only</span>
        </template>

        <template v-else>
          <span v-if="workspace.slug && workspace.slug !== 'default'" class="view-pill" title="Public plan">
            {{ workspace.slug }}’s plan
          </span>
        </template>
      </div>
    </div>
    <div class="header-trim"></div>
  </header>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { Sun, Moon, AlertTriangle, Clock, Settings, Bookmark, User, Users, UserPlus, Plus, Pencil, Trash2, LogOut, Eye, Globe } from 'lucide-vue-next'
import { useAppStore, baselines, store, MONTHS, settings, toggleTheme, riskWarnings, lateItems, ui, session, workspace, canEditWorkspace, createProject, loadMyWorkspaces } from '../stores/useAppStore.js'
import { api } from '../api.js'
import { APP_VERSION } from '../version.js'

defineProps({
  year: Number,
  zoom: { type: Number, default: 1 },
  authenticated: { type: Boolean, default: false },
})
const emit = defineEmits(['prev-year', 'next-year', 'manage', 'zoom-in', 'zoom-out', 'login', 'logout', 'about'])

const version = APP_VERSION

const canEdit = computed(() => canEditWorkspace())

// Project switcher
const projOpen = ref(false)
const projRef = ref(null)
const currentProjectName = computed(() => {
  const cur = workspace.myWorkspaces.find(p => p.slug === workspace.slug)
  return cur?.name || workspace.slug || 'Plan'
})
// Split the switcher into your personal home plan vs. collaborative projects.
const homeWs = computed(() => workspace.myWorkspaces.find(p => p.slug === workspace.ownSlug))
const projects = computed(() => workspace.myWorkspaces.filter(p => p.slug !== workspace.ownSlug))
const currentIsProject = computed(() =>
  !!workspace.slug && workspace.slug !== workspace.ownSlug && projects.value.some(p => p.slug === workspace.slug))

// A "foreign" plan is one you're viewing but don't belong to (public/guest).
const isForeign = computed(() => !!workspace.slug && !workspace.myWorkspaces.some(p => p.slug === workspace.slug))

// Public plans (loaded lazily when the switcher opens) so you can hop between the
// schedules available to you — and clearly see when you're a guest on one.
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
function inviteToProject() {
  projOpen.value = false
  emit('manage', 'members') // open Settings straight on the Members tab
}
async function renameCurrent() {
  const name = prompt('Rename project:', currentProjectName.value)
  if (!name || !name.trim()) return
  try { await api.renameProject(workspace.slug, name.trim()); await loadMyWorkspaces() } catch (e) { alert(e?.message || 'Rename failed') }
}
async function deleteCurrent() {
  projOpen.value = false
  if (!confirm(`Delete “${currentProjectName.value}” and all of its data? This can’t be undone.`)) return
  try { await api.deleteProject(workspace.slug); window.location.assign('/') } catch (e) { alert(e?.message || 'Could not delete the project') }
}
async function leaveCurrent() {
  projOpen.value = false
  if (!confirm('Leave this project? You will lose access until an owner re-invites you.')) return
  try { await api.leaveProject(workspace.slug); window.location.assign('/') } catch (e) { alert(e?.message || 'Could not leave the project') }
}

// Account menu (username avatar → log out / about).
const userOpen = ref(false)
const userRef = ref(null)
const initials = computed(() => (session.username || '?').trim().charAt(0).toUpperCase() || '?')

const { selectBaseline, createBaseline, setGranularity, prevMonth, nextMonth, setView } = useAppStore()

// Arrow order left→right: oldest … newest … Live (rightmost = most current).
const blSeq = computed(() => {
  const ids = baselines.list.map(b => b.id)
  ids.reverse()   // baselines.list is newest-first → make it oldest-first
  ids.push(null)  // null = Live (head) sits at the far right
  return ids
})
const blIndex = computed(() => {
  const i = blSeq.value.indexOf(baselines.activeId || null)
  return i < 0 ? blSeq.value.length - 1 : i
})
const canPrevBaseline = computed(() => blIndex.value > 0)                       // left = older
const canNextBaseline = computed(() => blIndex.value < blSeq.value.length - 1)  // right = newer / Live
function stepBaseline(dir) {
  const i = blIndex.value + dir
  if (i >= 0 && i < blSeq.value.length) selectBaseline(blSeq.value[i])
}
function jumpBaseline(dir) {
  selectBaseline(blSeq.value[dir < 0 ? 0 : blSeq.value.length - 1])
}

const hoverRisk = ref(false)
const hoverLate = ref(false)
function focusRisk(id) { ui.focusItemId = id; hoverRisk.value = false; hoverLate.value = false }

// Today's date + ISO calendar week, shown next to the brand.
function isoWeek(dt) {
  const d = new Date(Date.UTC(dt.getFullYear(), dt.getMonth(), dt.getDate()))
  const dayNum = (d.getUTCDay() + 6) % 7
  d.setUTCDate(d.getUTCDate() - dayNum + 3)
  const firstThursday = new Date(Date.UTC(d.getUTCFullYear(), 0, 4))
  const fDayNum = (firstThursday.getUTCDay() + 6) % 7
  firstThursday.setUTCDate(firstThursday.getUTCDate() - fDayNum + 3)
  return 1 + Math.round((d - firstThursday) / (7 * 86400000))
}
const today = new Date()
const todayLabel = today.toLocaleDateString('en-US', { day: 'numeric', month: 'short', year: 'numeric' })
const todayWeek = isoWeek(today)

// Custom baseline dropdown (the native <select> popup can't be styled).
const blOpen = ref(false)
const blRef = ref(null)
const currentLabel = computed(() => {
  if (!baselines.activeId) return 'Live'
  const b = baselines.list.find(x => x.id === baselines.activeId)
  return b ? b.name : 'Live'
})
function pickBaseline(id) { selectBaseline(id); blOpen.value = false }
function onDocClick(e) {
  if (blRef.value && !blRef.value.contains(e.target)) blOpen.value = false
  if (projRef.value && !projRef.value.contains(e.target)) projOpen.value = false
  if (userRef.value && !userRef.value.contains(e.target)) userOpen.value = false
}
function onKeyDown(e) { if (e.key === 'Escape') { blOpen.value = false; projOpen.value = false; userOpen.value = false } }
onMounted(() => { document.addEventListener('click', onDocClick); document.addEventListener('keydown', onKeyDown) })
onUnmounted(() => { document.removeEventListener('click', onDocClick); document.removeEventListener('keydown', onKeyDown) })
function nowStamp() {
  const d = new Date()
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}.${p(d.getMonth() + 1)}.${p(d.getDate())} - ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}
async function onSaveBaseline() {
  const name = prompt('Baseline name:', nowStamp())
  if (name && name.trim()) await createBaseline(name.trim())
}
</script>

<style scoped>
.header {
  background: var(--clr-header);
  position: sticky;
  top: 0;
  z-index: 100;
  box-shadow: 0 1px 0 rgba(255,255,255,0.06), var(--sh-md);
}

.header-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  height: 64px;
  gap: 16px;
}

.header-left { display: flex; align-items: center; gap: 12px; flex: 1; min-width: 0; }
.header-center { display: flex; align-items: center; flex-shrink: 0; }
.header-right { display: flex; align-items: center; gap: 8px; flex: 1; min-width: 0; justify-content: flex-end; }

.hdr-icon-btn {
  width: 32px; height: 32px; flex-shrink: 0;
  display: inline-flex; align-items: center; justify-content: center;
  border-radius: 100px;
  color: rgba(255,255,255,0.8);
  background: rgba(255,255,255,0.08);
  border: 1px solid rgba(255,255,255,0.12);
  transition: background 0.15s, color 0.15s;
}
.hdr-icon-btn:hover { background: rgba(255,255,255,0.14); color: #fff; }

/* Account avatar + its menu (replaces the username chip + Log out button). */
.user-av {
  width: 30px; height: 30px; border-radius: 50%; flex-shrink: 0;
  display: inline-flex; align-items: center; justify-content: center;
  background: rgba(255,255,255,0.16); color: #fff;
  font-size: 12px; font-weight: 700;
  transition: background 0.15s;
}
.user-av:hover { background: rgba(255,255,255,0.26); }
.user-menu { min-width: 168px; }
.user-menu-name {
  padding: 8px 12px 7px; font-size: 12px; font-weight: 700; color: var(--clr-text-3);
  border-bottom: 1px solid var(--clr-border-light); margin-bottom: 4px;
}

/* Free up width on narrower screens so header zones never overlap. */
@media (max-width: 1200px) {
  .today-chip { display: none; }
}

.risk-hdr {
  position: relative;
  display: inline-flex; align-items: center; gap: 5px;
  height: 32px; box-sizing: border-box; padding: 0 12px;
  border-radius: 100px;
  font-size: 13px; font-weight: 700;
  color: #ffd2cf;
  background: rgba(255,69,58,0.18);
  border: 1px solid rgba(255,69,58,0.42);
  cursor: default;
}
.risk-pop {
  position: absolute; top: 100%; right: 0;
  padding: 10px 8px 8px;
  min-width: 240px; max-width: 360px; max-height: 320px; overflow-y: auto;
  background: var(--clr-surface);
  border: 1px solid var(--clr-border-light);
  border-radius: var(--r-lg);
  box-shadow: var(--sh-modal);
  z-index: 200; text-align: left;
}
.risk-pop-title { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-danger); padding: 0 6px 6px; }
.risk-pop-row { display: flex; flex-direction: column; gap: 1px; padding: 5px 6px; border-radius: 6px; cursor: pointer; }
.risk-pop-row:hover { background: var(--clr-surface-2); }
.risk-pop-name { font-size: 12.5px; font-weight: 600; color: var(--clr-text); }
.risk-pop-sub { font-size: 11px; color: var(--clr-text-3); }

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.brand-btn {
  display: flex;
  align-items: center;
  gap: 12px;
  background: none;
  padding: 4px;
  margin: -4px;
  border-radius: 10px;
  transition: background 0.15s;
}
.brand-btn:hover { background: rgba(255,255,255,0.07); }

.brand-ver {
  font-size: 10px;
  font-weight: 500;
  color: rgba(255,255,255,0.4);
  letter-spacing: 0.3px;
}

.brand-icon {
  width: 38px;
  height: 38px;
  background: rgba(255,255,255,0.08);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(255,255,255,0.1);
  flex-shrink: 0;
}

.brand-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.brand-title {
  font-size: 15px;
  font-weight: 600;
  color: #FFFFFF;
  letter-spacing: -0.2px;
}

.today-chip {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-left: 12px;
  border-left: 1px solid rgba(255,255,255,0.12);
  flex-shrink: 0;
}
.today-date {
  font-size: 13px;
  font-weight: 500;
  color: rgba(255,255,255,0.82);
  white-space: nowrap;
}
.today-cw {
  font-size: 11px;
  font-weight: 700;
  color: rgba(255,255,255,0.72);
  background: rgba(255,255,255,0.1);
  border: 1px solid rgba(255,255,255,0.12);
  padding: 2px 8px;
  border-radius: 100px;
  white-space: nowrap;
}

@media (max-width: 880px) {
  .today-chip { display: none; }
}

.zoom-nav {
  display: flex;
  align-items: center;
  gap: 2px;
  height: 32px;
  box-sizing: border-box;
  background: rgba(255,255,255,0.08);
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 100px;
  padding: 0 3px;
  flex-shrink: 0;
}

.year-nav {
  display: flex;
  align-items: center;
  gap: 2px;
  height: 32px;
  box-sizing: border-box;
  background: rgba(255,255,255,0.08);
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 100px;
  padding: 0 3px;
  flex-shrink: 0;
}

.year-btn {
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border-radius: 100px;
  color: rgba(255,255,255,0.7);
  transition: background 0.15s, color 0.15s;
}

.year-btn:hover:not(:disabled) {
  background: rgba(255,255,255,0.12);
  color: #FFFFFF;
}
.year-btn:disabled { opacity: 0.3; cursor: default; }

/* Year/Month granularity toggle: a compact text button beside the date stepper. */
.gran-toggle {
  width: auto;
  padding: 0 9px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.2px;
  margin-left: 2px;
  color: rgba(255,255,255,0.85);
}

/* Timeline ↔ Explorer view toggle. */
.view-toggle {
  display: inline-flex;
  background: rgba(255,255,255,0.08);
  border-radius: 100px;
  padding: 2px;
  flex-shrink: 0;
}
.vt-btn {
  font-size: 12px;
  font-weight: 600;
  color: rgba(255,255,255,0.7);
  background: transparent;
  border-radius: 100px;
  padding: 4px 12px;
  transition: background 0.15s, color 0.15s;
}
.vt-btn.on { background: rgba(255,255,255,0.16); color: #FFFFFF; }
.vt-btn:hover:not(.on) { color: #FFFFFF; }

.year-label {
  font-size: 13px;
  font-weight: 600;
  color: #FFFFFF;
  padding: 0 8px;
  letter-spacing: -0.2px;
  min-width: 40px;
  text-align: center;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  justify-content: flex-end;
}

.edit-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  box-sizing: border-box;
  padding: 0 14px;
  font-size: 13px;
  font-weight: 600;
  color: rgba(255,255,255,0.85);
  background: rgba(48,209,88,0.16);
  border: 1px solid rgba(48,209,88,0.3);
  border-radius: 100px;
}
.edit-dot {
  width: 7px; height: 7px; border-radius: 50%;
  background: #30D158;
}
.user-chip {
  display: inline-flex;
  align-items: center;
  height: 32px;
  box-sizing: border-box;
  padding: 0 12px;
  font-size: 12.5px;
  font-weight: 600;
  color: rgba(255,255,255,0.92);
  background: rgba(255,255,255,0.10);
  border: 1px solid rgba(255,255,255,0.18);
  border-radius: 100px;
}

.view-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  box-sizing: border-box;
  padding: 0 14px;
  font-size: 13px;
  font-weight: 600;
  color: rgba(255,255,255,0.9);
  background: rgba(255,159,10,0.18);
  border: 1px solid rgba(255,159,10,0.4);
  border-radius: 100px;
}
.view-dot { width: 7px; height: 7px; border-radius: 50%; background: #FF9F0A; }

.btn-manage {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  height: 32px;
  box-sizing: border-box;
  padding: 0 14px;
  font-size: 13px;
  font-weight: 500;
  color: rgba(255,255,255,0.85);
  background: rgba(255,255,255,0.08);
  border: 1px solid rgba(255,255,255,0.12);
  border-radius: 100px;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}

.btn-manage:hover {
  background: rgba(255,255,255,0.14);
  color: #FFFFFF;
  border-color: rgba(255,255,255,0.2);
}

.baseline-ctl { display: flex; align-items: center; gap: 8px; }
.bl-dd { position: relative; }
.bl-select {
  display: inline-flex; align-items: center; justify-content: space-between; gap: 8px;
  height: 32px; box-sizing: border-box;
  width: 210px;
  padding: 0 12px 0 14px;
  background: rgba(255,255,255,0.08);
  border: 1px solid rgba(255,255,255,0.12);
  color: #fff; border-radius: 100px;
  font-size: 13px; font-weight: 500; cursor: pointer;
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
  position: absolute;
  top: calc(100% + 6px);
  left: 50%; transform: translateX(-50%);
  min-width: 240px; max-width: 340px;
  max-height: 340px; overflow-y: auto;
  background: var(--clr-surface);
  border: 1px solid var(--clr-border-light);
  border-radius: var(--r-lg);
  box-shadow: var(--sh-modal);
  padding: 6px;
  z-index: 200;
}
.bl-opt {
  display: flex; align-items: baseline; justify-content: space-between; gap: 12px;
  width: 100%; text-align: left;
  padding: 8px 10px; border-radius: 8px;
  background: none; cursor: pointer;
  transition: background 0.12s;
}
.bl-opt:hover { background: var(--clr-surface-2); }
.bl-opt.active { background: rgba(0,113,227,0.1); }
.bl-opt-name { font-size: 13px; font-weight: 500; color: var(--clr-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.bl-opt.active .bl-opt-name { color: var(--clr-accent); font-weight: 600; }
.bl-opt-sub { font-size: 11px; color: var(--clr-text-3); white-space: nowrap; flex-shrink: 0; }

/* Project switcher: separate the personal home plan from collaborative projects. */
.proj-menu { min-width: 264px; }
.proj-menu .bl-opt { align-items: center; justify-content: flex-start; }
.proj-menu .bl-opt .bl-opt-sub { margin-left: auto; text-transform: capitalize; }
.proj-sec {
  display: flex; align-items: baseline; gap: 6px;
  padding: 9px 10px 4px;
  font-size: 10px; font-weight: 700; letter-spacing: 0.6px; text-transform: uppercase;
  color: var(--clr-text-3);
}
.proj-sec-hint { font-size: 9px; font-weight: 600; letter-spacing: 0.2px; text-transform: none; color: var(--clr-text-3); opacity: 0.75; }
.proj-glyph {
  width: 22px; height: 22px; border-radius: 6px; flex-shrink: 0;
  display: inline-flex; align-items: center; justify-content: center;
}
.proj-glyph-home { background: rgba(0,113,227,0.12); color: var(--clr-accent); }
.proj-glyph-team { background: var(--clr-surface-2); color: var(--clr-text-2); }
.proj-glyph-guest { background: rgba(255,149,0,0.16); color: #E8890C; }
.proj-glyph-public { background: var(--clr-surface-2); color: var(--clr-text-3); }
.proj-guest { cursor: default; }
.proj-empty { padding: 2px 10px 6px; font-size: 12px; color: var(--clr-text-3); line-height: 1.45; }
.proj-divider { height: 1px; background: var(--clr-border-light); margin: 6px 4px; }
.proj-new, .proj-action, .proj-leave, .proj-del { gap: 8px; font-size: 13px; color: var(--clr-text-2); }
.proj-new { color: var(--clr-accent); font-weight: 600; }
.proj-del { color: #F85149; }
.proj-del:hover { background: rgba(248,81,73,0.10); }

.bl-diff { display: inline-flex; gap: 4px; position: relative; }
.bd { cursor: default; }

.diff-pop {
  position: absolute;
  top: 100%;
  right: 0;
  padding: 10px 8px 8px;        /* top padding bridges the gap so hover stays open */
  min-width: 210px;
  max-width: 340px;
  max-height: 320px;
  overflow-y: auto;
  background: var(--clr-surface);
  border: 1px solid var(--clr-border-light);
  border-radius: var(--r-lg);
  box-shadow: var(--sh-modal);
  z-index: 200;
  text-align: left;
}
.diff-pop-title {
  font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px;
  color: var(--clr-text-3); padding: 0 6px 6px;
}
.diff-pop-row {
  display: flex; align-items: baseline; justify-content: space-between; gap: 12px;
  padding: 4px 6px; border-radius: 6px;
}
.diff-pop-row:hover { background: var(--clr-surface-2); }
.diff-pop-name { font-size: 12.5px; color: var(--clr-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.diff-pop-meta { font-size: 11px; color: var(--clr-text-3); white-space: nowrap; flex-shrink: 0; }
.bd { font-size: 11px; font-weight: 700; padding: 2px 7px; border-radius: 100px; }
.bd-add { background: rgba(48,209,88,0.22); color: #6ee7a0; }
.bd-move { background: rgba(255,159,10,0.22); color: #fdba74; }
.bd-rem { background: rgba(255,69,58,0.22); color: #fda4a4; }
.bl-btn {
  display: inline-flex; align-items: center;
  height: 32px; box-sizing: border-box;
  padding: 0 14px; font-size: 13px; font-weight: 500;
  color: rgba(255,255,255,0.85); background: rgba(255,255,255,0.08);
  border: 1px solid rgba(255,255,255,0.12); border-radius: 100px;
  transition: background 0.15s, color 0.15s;
}
.bl-btn:hover { background: rgba(255,255,255,0.14); color: #fff; }

.header-trim {
  height: 2px;
  background: linear-gradient(90deg,
    transparent 0%,
    rgba(255,255,255,0.06) 20%,
    rgba(0,113,227,0.5) 50%,
    rgba(255,255,255,0.06) 80%,
    transparent 100%
  );
}
</style>
