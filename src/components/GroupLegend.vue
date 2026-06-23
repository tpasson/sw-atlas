<template>
  <div v-if="!readOnly || groups.list.length" class="group-legend">
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

      <button v-if="!readOnly" class="gl-add" @click="add">
        + {{ groups.list.length ? 'Group' : 'Add group' }}
      </button>
      <span v-else-if="groups.list.length === 0" class="gl-empty">No groups</span>
    </div>
  </div>
</template>

<script setup>
import { groups, ui, swatchColors, useAppStore } from '../stores/useAppStore.js'

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
.group-legend {
  position: fixed;
  bottom: 14px; left: 18px;
  display: flex; align-items: center; gap: 12px;
  max-width: 60vw;
  min-height: 40px; box-sizing: border-box;
  padding: 0 16px;
  background: var(--clr-glass);
  backdrop-filter: blur(10px); -webkit-backdrop-filter: blur(10px);
  border: 1px solid var(--clr-border);
  border-radius: 100px;
  box-shadow: var(--sh-lg), 0 0 0 1px rgba(0,0,0,0.03);
  z-index: 50;
}
.gl-title {
  font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px;
  color: var(--clr-text-3); flex-shrink: 0;
}
.gl-items { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }

.gl-chip {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 3px 6px 3px 8px;
  border-radius: 100px;
  font-size: 12px; color: var(--clr-text-2);
  cursor: default;
  transition: background 0.12s, color 0.12s;
}
.gl-chip:hover, .gl-chip.hot { background: var(--clr-surface-2); color: var(--clr-text); }
.gl-dot { width: 9px; height: 9px; border-radius: 50%; flex-shrink: 0; }
.gl-name { white-space: nowrap; }

.gl-x {
  width: 15px; height: 15px; border-radius: 50%;
  display: inline-flex; align-items: center; justify-content: center;
  font-size: 13px; line-height: 1; color: var(--clr-text-3);
  background: transparent;
  opacity: 0; transition: opacity 0.12s, color 0.12s, background 0.12s;
}
.gl-chip:hover .gl-x { opacity: 1; }
.gl-x:hover { background: rgba(255,59,48,0.12); color: var(--clr-danger); }

.gl-add {
  font-size: 12px; font-weight: 600; color: var(--clr-accent);
  background: rgba(0,113,227,0.08);
  padding: 4px 11px; border-radius: 100px;
  transition: background 0.12s;
}
.gl-add:hover { background: rgba(0,113,227,0.16); }
.gl-empty { font-size: 12px; color: var(--clr-text-3); }
</style>
