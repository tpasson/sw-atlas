<template>
  <!-- Compact bottom-right legend: marker shapes + the event bar + maturity
       stages. (The old Groups bar was removed — group/dimension selection will
       return as a configurable hover-filter; see the initiative issues.) -->
  <div class="legend-dock">
    <span v-for="(m, i) in settings.markers" :key="m.shape + i" class="legend-item">
      <MarkerIcon :shape="m.shape" color="#8a8a8e" :size="settings.items.markerSize" :stroke-width="settings.items.markerStroke" :fill="m.fill" /> {{ m.label }}
    </span>
    <span class="legend-item"><span class="legend-bar"></span> {{ settings.eventLabel }}</span>
    <span class="legend-sep"></span>
    <span v-for="(s, i) in MATURITY_STAGES" :key="s" class="legend-item">
      <MaturityGlyph :level="i + 1" variant="grid" color="#8a8a8e" :title="s" /> {{ s }}
    </span>
  </div>
</template>

<script setup>
import { settings, MATURITY_STAGES } from '../stores/useAppStore.js'
import MarkerIcon from './MarkerIcon.vue'
import MaturityGlyph from './MaturityGlyph.vue'

// readOnly is accepted (passed by App.vue) but the legend is purely informational.
defineProps({ readOnly: { type: Boolean, default: false } })
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
}
.legend-item { display: inline-flex; align-items: center; gap: 5px; white-space: nowrap; }
.legend-sep { width: 1px; height: 18px; background: var(--clr-border); flex-shrink: 0; }
.legend-bar { width: 18px; height: 10px; border-radius: 3px; background: rgba(120,120,128,0.3); border: 1px solid rgba(120,120,128,0.55); }
</style>
