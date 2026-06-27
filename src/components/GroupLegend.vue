<template>
  <!-- Unified bottom dock: groups (left, scroll when many) + marker/maturity
       legend (right, pinned). One flex row, so groups can never overlap the
       legend — they scroll instead. -->
  <div class="legend-dock">
    <div v-if="!readOnly || groups.list.length" class="ld-groups">
      <span class="gl-title">Groups</span>
      <div class="gl-items">
        <span
          v-for="g in groups.list"
          :key="g.id"
          class="gl-chip"
          :class="{ hot: ui.hoverGroupId === g.id }"
          @mouseenter="ui.hoverGroupId = g.id"
          @mouseleave="ui.hoverGroupId = null"
        >
          <span class="gl-dot" :style="{ background: g.color }"></span>
          <span class="gl-name" :title="!readOnly ? 'Double-click to rename' : ''" @dblclick="!readOnly && rename(g)">{{ g.name }}</span>
          <button v-if="!readOnly" class="gl-x" title="Delete group" @click="del(g)">×</button>
        </span>
        <span v-if="readOnly && groups.list.length === 0" class="gl-empty">No groups</span>
      </div>
      <button v-if="!readOnly" class="gl-add" @click="add">+ {{ groups.list.length ? 'Group' : 'Add group' }}</button>
    </div>

    <div class="ld-legend">
      <span v-for="(m, i) in settings.markers" :key="m.shape + i" class="legend-item">
        <MarkerIcon :shape="m.shape" color="#8a8a8e" :size="settings.items.markerSize" :stroke-width="settings.items.markerStroke" :fill="m.fill" /> {{ m.label }}
      </span>
      <span class="legend-item"><span class="legend-bar"></span> {{ settings.eventLabel }}</span>
      <span class="legend-sep"></span>
      <span v-for="(s, i) in MATURITY_STAGES" :key="s" class="legend-item">
        <MaturityGlyph :level="i + 1" variant="grid" color="#8a8a8e" :title="s" /> {{ s }}
      </span>
    </div>
  </div>
</template>

<script setup>
import { groups, ui, settings, swatchColors, MATURITY_STAGES, useAppStore } from '../stores/useAppStore.js'
import MarkerIcon from './MarkerIcon.vue'
import MaturityGlyph from './MaturityGlyph.vue'

defineProps({ readOnly: { type: Boolean, default: false } })

const { addGroup, deleteGroup, updateGroup } = useAppStore()

function add() {
  const name = prompt('Group name:')
  if (!name || !name.trim()) return
  const pal = swatchColors.value
  const color = pal.length ? pal[groups.list.length % pal.length] : '#0A84FF'
  addGroup(name.trim(), color)
}
function rename(g) {
  const n = prompt('Rename group:', g.name)
  if (n && n.trim()) updateGroup(g.id, { name: n.trim() })
}
function del(g) {
  if (confirm(`Delete group "${g.name}"?`)) deleteGroup(g.id)
}
</script>

<style scoped>
.legend-dock {
  position: fixed;
  bottom: 14px; left: 18px; right: 18px;
  display: flex; align-items: center; gap: 14px;
  min-height: 40px; box-sizing: border-box;
  padding: 0 16px;
  background: var(--clr-glass);
  backdrop-filter: blur(10px); -webkit-backdrop-filter: blur(10px);
  border: 1px solid var(--clr-border);
  border-radius: 100px;
  box-shadow: var(--sh-lg), 0 0 0 1px rgba(0,0,0,0.03);
  z-index: 50;
}

/* Groups zone: takes the free space; its chips scroll horizontally when many. */
.ld-groups { display: flex; align-items: center; gap: 10px; flex: 1 1 auto; min-width: 0; }
.gl-title {
  font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px;
  color: var(--clr-text-3); flex-shrink: 0;
}
.gl-items {
  display: flex; align-items: center; gap: 6px;
  flex-wrap: nowrap; overflow-x: auto; min-width: 0; flex: 1 1 auto;
  scrollbar-width: thin; scrollbar-color: var(--clr-border) transparent;
}
.gl-items::-webkit-scrollbar { height: 6px; }
.gl-items::-webkit-scrollbar-thumb { background: var(--clr-border); border-radius: 3px; }

.gl-chip {
  display: inline-flex; align-items: center; gap: 6px; flex-shrink: 0;
  padding: 3px 6px 3px 8px;
  border-radius: 100px;
  font-size: 12px; color: var(--clr-text-2);
  cursor: default;
  transition: background 0.12s, color 0.12s;
}
.gl-chip:hover, .gl-chip.hot { background: var(--clr-surface-2); color: var(--clr-text); }
.gl-dot { width: 9px; height: 9px; border-radius: 50%; flex-shrink: 0; }
.gl-name { white-space: nowrap; max-width: 180px; overflow: hidden; text-overflow: ellipsis; }

.gl-x {
  width: 15px; height: 15px; border-radius: 50%;
  display: inline-flex; align-items: center; justify-content: center;
  font-size: 13px; line-height: 1; color: var(--clr-text-3);
  background: transparent; flex-shrink: 0;
  opacity: 0; transition: opacity 0.12s, color 0.12s, background 0.12s;
}
.gl-chip:hover .gl-x { opacity: 1; }
.gl-x:hover { background: rgba(255,59,48,0.12); color: var(--clr-danger); }

.gl-add {
  flex-shrink: 0;
  font-size: 12px; font-weight: 600; color: var(--clr-accent);
  background: rgba(0,113,227,0.08);
  padding: 4px 11px; border-radius: 100px;
  transition: background 0.12s;
}
.gl-add:hover { background: rgba(0,113,227,0.16); }
.gl-empty { font-size: 12px; color: var(--clr-text-3); }

/* Legend zone: pinned to the right, never shrinks/overlaps. */
.ld-legend {
  display: flex; align-items: center; gap: 14px; flex-shrink: 0; margin-left: auto;
  font-size: 11px; color: var(--clr-text-2);
}
.legend-item { display: inline-flex; align-items: center; gap: 5px; white-space: nowrap; }
.legend-sep { width: 1px; height: 18px; background: var(--clr-border); flex-shrink: 0; }
.legend-bar { width: 18px; height: 10px; border-radius: 3px; background: rgba(120,120,128,0.3); border: 1px solid rgba(120,120,128,0.55); }
</style>
