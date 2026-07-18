<template>
  <div class="crv">
    <div class="crv-head">
      <ClipboardCheck :size="18" />
      <h1 class="crv-title">Change Requests</h1>
      <span v-if="pending.length" class="crv-count">{{ pending.length }} pending</span>

      <div class="crv-views">
        <button type="button" class="crv-vbtn" :class="{ on: viewMode === 'list' }" title="List" @click="setView('list')"><List :size="15" :stroke-width="2.2" /></button>
        <button type="button" class="crv-vbtn" :class="{ on: viewMode === 'board' }" title="Board" @click="setView('board')"><Columns3 :size="15" :stroke-width="2.2" /></button>
      </div>

      <button v-if="canPropose" class="crv-new" @click="$emit('propose-new')">+ Propose new item</button>
    </div>

    <div class="crv-body">
      <p v-if="!changeRequests.list.length" class="crv-empty">
        No change requests yet. Open a milestone and choose <strong>Propose change</strong> to suggest one —
        the project owner can then approve it onto the timeline.
      </p>

      <!-- List: status sections stacked vertically -->
      <div v-else-if="viewMode === 'list'" class="crv-list">
        <section v-for="g in groups" :key="g.key" class="crv-sec">
          <div class="crv-sec-h"><span class="crv-dot" :style="{ background: g.color }" /> {{ g.label }} <span class="crv-sec-n">{{ g.items.length }}</span></div>
          <ChangeRequestCard v-for="cr in g.items" :key="cr.id" :cr="cr" :expanded="expanded.has(cr.id)" :focused="ui.focusCrId === cr.id" @toggle="toggle" />
        </section>
      </div>

      <!-- Board: status columns side by side -->
      <div v-else class="crv-board">
        <section v-for="g in groups" :key="g.key" class="crv-col">
          <div class="crv-sec-h"><span class="crv-dot" :style="{ background: g.color }" /> {{ g.label }} <span class="crv-sec-n">{{ g.items.length }}</span></div>
          <div class="crv-col-body">
            <ChangeRequestCard v-for="cr in g.items" :key="cr.id" :cr="cr" :expanded="expanded.has(cr.id)" :focused="ui.focusCrId === cr.id" @toggle="toggle" />
            <p v-if="!g.items.length" class="crv-col-empty">None</p>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import { ClipboardCheck, List, Columns3 } from 'lucide-vue-next'
import { changeRequests, session, workspace, ui } from '../stores/useAppStore.js'
import ChangeRequestCard from './ChangeRequestCard.vue'

defineEmits(['propose-new'])

const VIEW_KEY = 'atlas-cr-view'
const viewMode = ref(['list', 'board'].includes(localStorage.getItem(VIEW_KEY)) ? localStorage.getItem(VIEW_KEY) : 'list')
function setView(v) { viewMode.value = v; try { localStorage.setItem(VIEW_KEY, v) } catch { /* ignore */ } }

const pending = computed(() => changeRequests.list.filter(c => c.status === 'pending'))
const canPropose = computed(() => session.authenticated && !!workspace.role)

const STATUS = [
  { key: 'pending', label: 'Pending', color: '#FF9F0A' },
  { key: 'approved', label: 'Approved', color: '#30D158' },
  { key: 'rejected', label: 'Rejected', color: '#FF3B30' },
]
// List hides empty sections; Board keeps all three columns for a stable layout.
const groups = computed(() => {
  const all = STATUS.map(s => ({ ...s, items: changeRequests.list.filter(c => c.status === s.key) }))
  return viewMode.value === 'board' ? all : all.filter(g => g.items.length)
})

// Which cards are expanded. Pending cards start open (they need a decision);
// decided ones start collapsed to keep the page compact.
const expanded = ref(new Set(pending.value.map(c => c.id)))
function toggle(id) { const s = expanded.value; s.has(id) ? s.delete(id) : s.add(id); expanded.value = new Set(s) }
// Seed pending → expanded once the list first arrives (it may load after mount).
let seeded = false
watch(() => changeRequests.list.length, (n) => {
  if (!seeded && n) { seeded = true; expanded.value = new Set([...expanded.value, ...pending.value.map(c => c.id)]) }
}, { immediate: true })

// Opened from the Explorer → expand, scroll to, and highlight the focused CR.
function focusCr() {
  if (!ui.focusCrId) return
  expanded.value = new Set(expanded.value).add(ui.focusCrId)
  nextTick(() => {
    const el = document.querySelector(`[data-cr="${ui.focusCrId}"]`)
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' })
  })
}
watch(() => ui.focusCrId, focusCr)
onMounted(focusCr)
</script>

<style scoped>
.crv { flex: 1; min-height: 0; display: flex; flex-direction: column; }
.crv-head { display: flex; align-items: center; gap: 9px; height: 56px; box-sizing: border-box; padding: 0 24px; border-bottom: 1px solid var(--clr-border-light); color: var(--clr-text); flex-shrink: 0; }
.crv-title { font-size: 17px; font-weight: 700; }
.crv-count { margin-left: 4px; font-size: 11px; font-weight: 700; color: #fff; background: #FF9F0A; border-radius: 100px; padding: 2px 9px; }

.crv-views { margin-left: auto; display: inline-flex; gap: 2px; background: var(--clr-surface-2); border-radius: var(--r-md); padding: 2px; }
.crv-vbtn { width: 28px; height: 24px; display: inline-flex; align-items: center; justify-content: center; border-radius: var(--r-sm); color: var(--clr-text-3); background: none; transition: background 0.12s, color 0.12s; }
.crv-vbtn:hover { color: var(--clr-text); }
.crv-vbtn.on { background: var(--clr-surface); color: var(--clr-accent); box-shadow: var(--sh-sm, 0 1px 2px rgba(0,0,0,0.08)); }

.crv-new { font-size: 13px; font-weight: 600; color: #fff; background: var(--clr-accent); border-radius: var(--r-md); padding: 7px 14px; }
.crv-new:hover { background: var(--clr-accent-hover); }

.crv-body { flex: 1; min-height: 0; overflow-y: auto; padding: 18px 24px 40px; }
.crv-empty { max-width: 640px; font-size: 13px; color: var(--clr-text-3); line-height: 1.6; }

.crv-sec-h { display: flex; align-items: center; gap: 7px; font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.4px; color: var(--clr-text-3); margin-bottom: 8px; }
.crv-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.crv-sec-n { font-weight: 600; color: var(--clr-text-3); background: var(--clr-surface-2); border-radius: 100px; padding: 0 7px; font-size: 11px; }

/* List */
.crv-list { max-width: 760px; margin: 0 auto; }
.crv-sec { margin-bottom: 22px; }
.crv-sec :deep(.crc) { margin-bottom: 8px; }

/* Board */
.crv-board { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16px; align-items: start; }
.crv-col-body { display: flex; flex-direction: column; gap: 8px; }
.crv-col-empty { font-size: 12px; color: var(--clr-text-3); padding: 6px 2px; }
</style>
