<template>
  <!-- Bottom-right legend: auto-derived from the timeline item TYPES actually in
       use (point types → their icon, range types → the bar) plus maturity stages.
       No manual marker palette anymore — icons live on the types. -->
  <div class="legend-dock">
    <span v-for="t in legendTypes" :key="t.key" class="legend-item">
      <span v-if="t.family === 'timeline-range'" class="legend-bar" :style="{ background: barFill(t.color), borderColor: barBorder(t.color) }"></span>
      <MarkerIcon v-else :shape="t.icon || 'l:Diamond'" :color="t.color || '#8a8a8e'" :size="settings.items.markerSize" :stroke-width="settings.items.markerStroke" :fill="t.fill !== false" />
      {{ t.label }}
    </span>
    <span v-if="legendTypes.length" class="legend-sep"></span>
    <span v-for="(s, i) in MATURITY_STAGES" :key="s" class="legend-item">
      <MaturityGlyph :level="i + 1" variant="grid" color="#8a8a8e" :title="s" /> {{ s }}
    </span>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { settings, MATURITY_STAGES, store, itemTypes } from '../stores/useAppStore.js'
import MarkerIcon from './MarkerIcon.vue'
import MaturityGlyph from './MaturityGlyph.vue'

// readOnly is accepted (passed by App.vue) but the legend is purely informational.
defineProps({ readOnly: { type: Boolean, default: false } })

const legendTypes = computed(() => {
  const present = new Set()
  for (const m of store.milestones) {
    if (m.sourceSystem) continue // mirrored SCM items have their own view
    present.add(m.typeKey || m.kind || 'milestone')
  }
  return itemTypes.list.filter(t =>
    (t.family === 'timeline-point' || t.family === 'timeline-range') && present.has(t.key))
})

function hexA(hex, a) {
  const h = (hex || '').replace('#', '')
  if (h.length === 3 || h.length === 6) {
    const n = h.length === 3 ? h.split('').map(x => x + x).join('') : h
    const r = parseInt(n.slice(0, 2), 16), g = parseInt(n.slice(2, 4), 16), b = parseInt(n.slice(4, 6), 16)
    return `rgba(${r},${g},${b},${a})`
  }
  return hex || `rgba(120,120,128,${a})`
}
function barFill(c) { return c ? hexA(c, 0.3) : 'rgba(120,120,128,0.3)' }
function barBorder(c) { return c ? hexA(c, 0.6) : 'rgba(120,120,128,0.55)' }
</script>

<style scoped>
.legend-dock {
  position: fixed;
  bottom: 14px; right: 18px;
  display: inline-flex; align-items: center; gap: 14px;
  min-height: 40px; box-sizing: border-box;
  padding: 0 16px;
  background: var(--clr-glass);
  backdrop-filter: blur(10px); -webkit-backdrop-filter: blur(10px);
  border: 1px solid var(--clr-border);
  border-radius: 100px;
  box-shadow: var(--sh-lg), 0 0 0 1px rgba(0,0,0,0.03);
  font-size: 11px; color: var(--clr-text-2);
  z-index: 50;
  max-width: 70vw; overflow-x: auto;
}
.legend-item { display: inline-flex; align-items: center; gap: 5px; white-space: nowrap; }
.legend-sep { width: 1px; height: 18px; background: var(--clr-border); flex-shrink: 0; }
.legend-bar { width: 18px; height: 10px; border-radius: 3px; background: rgba(120,120,128,0.3); border: 1px solid rgba(120,120,128,0.55); flex-shrink: 0; }
</style>
