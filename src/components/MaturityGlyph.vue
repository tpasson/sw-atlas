<template>
  <span class="mat-glyph" :class="variant" :style="{ '--mc': color, '--cell': size + 'px' }" :title="title">
    <span v-for="i in 4" :key="i" class="cell" :class="{ on: i <= level }"></span>
  </span>
</template>

<script setup>
// 4-stage maturity indicator that mirrors the ATLAS logo (four squares).
// level 1..4 = Concept · Design · Production · Series (0 = none). Filled = reached.
defineProps({
  level: { type: Number, default: 1 },          // 0..4
  variant: { type: String, default: 'grid' },   // 'grid' (2x2 logo) | 'row' (1x4 meter)
  color: { type: String, default: 'currentColor' },
  title: { type: String, default: '' },
  size: { type: Number, default: 5 },           // cell size in px
})
</script>

<style scoped>
.mat-glyph { display: inline-grid; gap: calc(var(--cell, 5px) * 0.3); vertical-align: middle; flex-shrink: 0; }
.mat-glyph.grid { grid-template-columns: repeat(2, var(--cell, 5px)); }
.mat-glyph.row  { grid-template-columns: repeat(4, var(--cell, 5px)); }
.cell {
  width: var(--cell, 5px); height: var(--cell, 5px);
  border-radius: calc(var(--cell, 5px) * 0.28); box-sizing: border-box;
  border: 1px solid var(--mc);
}
.cell.on { background: var(--mc); }
.cell:not(.on) { opacity: 0.35; }
</style>
