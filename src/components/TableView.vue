<template>
  <div class="tv">
    <table class="tv-table">
      <thead>
        <tr>
          <th v-for="c in cols" :key="c.key" :class="{ on: sortKey === c.key, num: c.num }" @click="sortBy(c.key)">
            {{ c.label }}<span v-if="sortKey === c.key" class="tv-arrow">{{ sortDir > 0 ? '▲' : '▼' }}</span>
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="m in rows" :key="m.id" @click="$emit('edit', m)">
          <td class="tv-title">
            <span class="tv-dot" :style="{ background: m.color || areaColor(m) || '#8a8a8e' }"></span>{{ m.title }}
          </td>
          <td>{{ typeLabel(m) }}</td>
          <td>{{ areaName(m) }}</td>
          <td class="num">{{ m.when || dateOf(m) }}</td>
          <td>{{ m.maturity ? MATURITY_STAGES[m.maturity - 1] : '' }}</td>
          <td class="num">{{ m.progress != null ? m.progress + '%' : '' }}</td>
        </tr>
      </tbody>
    </table>
    <div v-if="!rows.length" class="tv-empty">No artifacts yet.</div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { store, itemTypes, MATURITY_STAGES, MONTHS } from '../stores/useAppStore.js'

defineEmits(['edit'])

const cols = [
  { key: 'title', label: 'Title' },
  { key: 'type', label: 'Type' },
  { key: 'area', label: 'Area' },
  { key: 'date', label: 'Date', num: true },
  { key: 'maturity', label: 'Maturity' },
  { key: 'progress', label: 'Progress', num: true },
]

const sortKey = ref('date')
const sortDir = ref(1)
function sortBy(k) {
  if (sortKey.value === k) sortDir.value *= -1
  else { sortKey.value = k; sortDir.value = 1 }
}

function typeLabel(m) {
  const k = m.typeKey || m.kind || 'milestone'
  return itemTypes.list.find(t => t.key === k)?.label || k
}
function areaName(m) { return store.swimlanes.find(s => s.id === m.swimlaneId)?.name || '' }
function areaColor(m) { return store.swimlanes.find(s => s.id === m.swimlaneId)?.color }
function dateOf(m) { return m.year && m.swimlaneId ? `${MONTHS[(m.month || 1) - 1]} ${m.year}` : '' }

function keyOf(m, k) {
  if (k === 'title') return (m.title || '').toLowerCase()
  if (k === 'type') return typeLabel(m).toLowerCase()
  if (k === 'area') return areaName(m).toLowerCase()
  if (k === 'date') return m.when || `${m.year || 0}-${String(m.month || 0).padStart(2, '0')}`
  if (k === 'maturity') return m.maturity || 0
  if (k === 'progress') return m.progress ?? -1
  return ''
}
const rows = computed(() =>
  [...store.milestones].sort((a, b) => {
    const x = keyOf(a, sortKey.value), y = keyOf(b, sortKey.value)
    return (x < y ? -1 : x > y ? 1 : 0) * sortDir.value
  }))
</script>

<style scoped>
.tv { max-width: 1100px; margin: 0 auto; padding: 18px; }
.tv-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.tv-table th {
  text-align: left; padding: 8px 12px; color: var(--clr-text-3); font-weight: 700;
  font-size: 11px; text-transform: uppercase; letter-spacing: 0.4px;
  border-bottom: 1px solid var(--clr-border); cursor: pointer; white-space: nowrap; user-select: none;
}
.tv-table th.on { color: var(--clr-accent); }
.tv-table th.num, .tv-table td.num { text-align: right; }
.tv-arrow { font-size: 9px; margin-left: 4px; }
.tv-table td { padding: 9px 12px; border-bottom: 1px solid var(--clr-border-light); color: var(--clr-text); }
.tv-table tbody tr { cursor: pointer; }
.tv-table tbody tr:hover { background: var(--clr-bg); }
.tv-title { font-weight: 600; }
.tv-dot { display: inline-block; width: 9px; height: 9px; border-radius: 50%; margin-right: 8px; vertical-align: middle; }
.tv-empty { text-align: center; color: var(--clr-text-3); padding: 50px 0; }
</style>
