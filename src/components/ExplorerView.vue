<template>
  <div class="explorer">
    <div class="ex-head">
      <h2 class="ex-h">Explorer</h2>
      <span class="ex-sub">All artifacts, grouped by type</span>
    </div>

    <div v-for="g in folders" :key="g.key" class="ex-folder">
      <div class="ex-folder-head">
        <MarkerIcon :shape="g.type.icon || 'l:Diamond'" :color="g.type.color || '#8a8a8e'" :size="15" :fill="true" />
        <span class="ex-folder-name">{{ g.type.label }}</span>
        <span class="ex-count">{{ g.items.length }}</span>
        <button v-if="!readOnly && offTimeline(g.type)" class="ex-add" @click="$emit('add', g.type)">+ New</button>
      </div>
      <div class="ex-items">
        <button v-for="m in g.items" :key="m.id" class="ex-item" @click="$emit('edit', m)">
          <span class="ex-dot" :style="{ background: m.color || laneColor(m) || '#8a8a8e' }"></span>
          <span class="ex-title">{{ m.title }}</span>
          <span class="ex-meta">{{ meta(m) }}</span>
        </button>
        <div v-if="!g.items.length" class="ex-empty">No {{ g.type.label.toLowerCase() }} yet — “+ New” to add one.</div>
      </div>
    </div>

    <div v-if="!folders.length" class="ex-blank">No artifacts yet. Add items on the timeline, or define backlog types in Settings → Types.</div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { store, itemTypes, MONTHS } from '../stores/useAppStore.js'
import MarkerIcon from './MarkerIcon.vue'

defineProps({ readOnly: { type: Boolean, default: false } })
defineEmits(['edit', 'add'])

function offTimeline(t) {
  return t.family === 'work-item' || t.family === 'container'
}

// One folder per type: every defined type (so you can add into empty ones) plus
// any orphan type keys that still have items. Sorted by item count.
const folders = computed(() => {
  const byKey = new Map()
  for (const t of itemTypes.list) byKey.set(t.key, { key: t.key, type: t, items: [] })
  for (const m of store.milestones) {
    const k = m.typeKey || m.kind || 'milestone'
    let g = byKey.get(k)
    if (!g) { g = { key: k, type: { key: k, label: k, icon: 'l:Diamond', color: '' }, items: [] }; byKey.set(k, g) }
    g.items.push(m)
  }
  return [...byKey.values()]
    .filter(g => g.items.length > 0 || (g.type && !g.type.builtin))
    .sort((a, b) => (b.items.length - a.items.length) || a.type.label.localeCompare(b.type.label))
})

function laneColor(m) {
  return store.swimlanes.find(s => s.id === m.swimlaneId)?.color
}
function meta(m) {
  const lane = store.swimlanes.find(s => s.id === m.swimlaneId)
  const date = m.when || (m.year && m.swimlaneId ? `${MONTHS[(m.month || 1) - 1]} ${m.year}` : '')
  return [lane?.name, date].filter(Boolean).join(' · ')
}
</script>

<style scoped>
.explorer { max-width: 920px; margin: 0 auto; padding: 24px 18px 80px; }
.ex-head { margin-bottom: 18px; }
.ex-h { font-size: 20px; font-weight: 700; color: var(--clr-text); }
.ex-sub { font-size: 13px; color: var(--clr-text-3); }

.ex-folder { margin-bottom: 18px; border: 1px solid var(--clr-border-light); border-radius: var(--r-lg); overflow: hidden; background: var(--clr-surface); }
.ex-folder-head { display: flex; align-items: center; gap: 8px; padding: 10px 14px; background: var(--clr-bg); border-bottom: 1px solid var(--clr-border-light); }
.ex-folder-name { font-weight: 600; font-size: 14px; color: var(--clr-text); }
.ex-count { font-size: 12px; color: var(--clr-text-3); background: var(--clr-surface-2); border-radius: 100px; padding: 1px 8px; }
.ex-add { margin-left: auto; font-size: 12px; font-weight: 600; color: var(--clr-accent); background: rgba(0,113,227,0.08); padding: 4px 10px; border-radius: 100px; }
.ex-add:hover { background: rgba(0,113,227,0.16); }

.ex-items { display: flex; flex-direction: column; }
.ex-item { display: flex; align-items: center; gap: 10px; padding: 9px 14px; text-align: left; background: none; border-bottom: 1px solid var(--clr-border-light); transition: background 0.12s; }
.ex-item:last-child { border-bottom: none; }
.ex-item:hover { background: var(--clr-bg); }
.ex-dot { width: 9px; height: 9px; border-radius: 50%; flex-shrink: 0; }
.ex-title { font-size: 14px; color: var(--clr-text); flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ex-meta { font-size: 12px; color: var(--clr-text-3); flex-shrink: 0; }
.ex-empty { padding: 12px 14px; font-size: 13px; color: var(--clr-text-3); }
.ex-blank { text-align: center; color: var(--clr-text-3); padding: 60px 0; }
</style>
