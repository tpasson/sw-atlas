<template>
  <!-- Bottom-right legend: auto-derived from the timeline item TYPES actually in
       use. Each type shows its own SYMBOL (the flag for events, the diamond for
       milestones, …) in neutral grey — on the timeline colour means status, not
       type, so the legend stays a pure shape key. Plus the maturity stages. -->
  <div class="legend-dock">
    <span v-for="t in legendTypes" :key="t.key" class="legend-item">
      <MarkerIcon :shape="t.icon || 'l:Diamond'" color="#8a8a8e" :size="settings.items.markerSize" :stroke-width="settings.items.markerStroke" :fill="t.fill !== false" />
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
</style>
