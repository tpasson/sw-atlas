<template>
  <component
    v-if="comp"
    :is="comp"
    :size="px"
    :stroke-width="strokeWidth"
    :style="{ color, fill: fill ? color : 'none' }"
    class="marker-icon"
  />
</template>

<script setup>
import { computed } from 'vue'
import { LUCIDE_MARKERS } from '../lucideMarkers.js'

const props = defineProps({
  shape: { type: String, default: 'l:Flag' }, // "l:LucideName" (legacy geometric names are mapped)
  color: { type: String, default: '#8a8a8e' },
  size: { type: [Number, String], default: 12 },
  strokeWidth: { type: [Number, String], default: 2.25 },
  fill: { type: Boolean, default: false },
})

// Map the old hand-drawn shape names to their Lucide equivalents.
const LEGACY = {
  diamond: 'Diamond', circle: 'Circle', cone: 'Triangle', triangleDown: 'Triangle',
  flag: 'Flag', square: 'Square', star: 'Star', hexagon: 'Hexagon', pentagon: 'Pentagon',
}

const comp = computed(() => {
  const s = props.shape || ''
  const name = s.startsWith('l:') ? s.slice(2) : (LEGACY[s] || 'Flag')
  return LUCIDE_MARKERS[name] || LUCIDE_MARKERS.Flag || null
})
const px = computed(() => Number(props.size) || 12)
</script>

<style scoped>
.marker-icon { flex-shrink: 0; display: inline-block; vertical-align: middle; }
</style>
