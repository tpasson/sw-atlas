<template>
  <!-- Configurable hover-filter (replaces the old fixed Groups bar): pick a
       dimension, hover one of its values to highlight matching items on the
       timeline (the rest dim). Values scroll horizontally when there are many. -->
  <div class="facet-dock">
    <select class="fd-dim" v-model="dim" title="Filter dimension">
      <option value="group">Groups</option>
      <option value="type">Type</option>
      <option value="maturity">Maturity</option>
      <option v-if="workspace.members.length" value="assignee">Assignee</option>
    </select>
    <div class="fd-values">
      <span
        v-for="f in facets"
        :key="f.key"
        class="fd-chip"
        @mouseenter="ui.highlightIds = f.ids"
        @mouseleave="ui.highlightIds = null"
      >
        <span v-if="f.color" class="fd-dot" :style="{ background: f.color }"></span>
        <span class="fd-label" :title="f.label">{{ f.label }}</span>
        <span class="fd-count">{{ f.ids.size }}</span>
      </span>
      <span v-if="!facets.length" class="fd-empty">No {{ dimLabel }} to filter by yet</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onUnmounted } from 'vue'
import { store, groups, ui, itemTypes, MATURITY_STAGES, workspace } from '../stores/useAppStore.js'

const dim = ref('group')
const dimLabel = computed(() => (dim.value === 'group' ? 'groups' : dim.value === 'type' ? 'types' : dim.value === 'assignee' ? 'assignees' : 'maturity stages'))

const facets = computed(() => {
  if (dim.value === 'group') {
    return groups.list.map(g => ({ key: g.id, label: g.name, color: g.color, ids: new Set(g.itemIds || []) }))
  }
  if (dim.value === 'type') {
    const by = new Map()
    for (const m of store.milestones) {
      const k = m.typeKey || m.kind || 'milestone'
      if (!by.has(k)) by.set(k, new Set())
      by.get(k).add(m.id)
    }
    return itemTypes.list
      .filter(t => by.get(t.key)?.size)
      .map(t => ({ key: t.key, label: t.label, color: t.color, ids: by.get(t.key) }))
  }
  if (dim.value === 'assignee') {
    return workspace.members
      .map(mb => ({ key: mb.userId, label: mb.username, color: '', ids: new Set(store.milestones.filter(m => m.assigneeId === mb.userId).map(m => m.id)) }))
      .filter(f => f.ids.size)
  }
  // maturity
  return MATURITY_STAGES
    .map((s, i) => ({ key: i + 1, label: s, color: '', ids: new Set(store.milestones.filter(m => m.maturity === i + 1).map(m => m.id)) }))
    .filter(f => f.ids.size)
})

// Don't leave the timeline dimmed if this dock unmounts (view switch).
onUnmounted(() => { ui.highlightIds = null })
</script>

<style scoped>
.facet-dock {
  position: fixed;
  bottom: 14px; left: 18px;
  display: flex; align-items: center; gap: 10px;
  max-width: 56vw;
  min-height: 40px; box-sizing: border-box;
  padding: 5px 12px 5px 8px;
  background: var(--clr-glass);
  backdrop-filter: blur(10px); -webkit-backdrop-filter: blur(10px);
  border: 1px solid var(--clr-border);
  border-radius: 100px;
  box-shadow: var(--sh-lg), 0 0 0 1px rgba(0,0,0,0.03);
  z-index: 50;
}
.fd-dim {
  flex-shrink: 0;
  border: 1px solid var(--clr-border); border-radius: 100px;
  padding: 4px 10px; font-size: 12px; font-weight: 600;
  color: var(--clr-text); background: var(--clr-bg);
}
.fd-dim:focus { outline: none; border-color: var(--clr-accent); }

.fd-values {
  display: flex; align-items: center; gap: 6px;
  overflow-x: auto; min-width: 0;
  scrollbar-width: thin; scrollbar-color: var(--clr-border) transparent;
}
.fd-values::-webkit-scrollbar { height: 5px; }
.fd-values::-webkit-scrollbar-thumb { background: var(--clr-border); border-radius: 3px; }

.fd-chip {
  display: inline-flex; align-items: center; gap: 6px; flex-shrink: 0;
  padding: 3px 8px; border-radius: 100px;
  font-size: 12px; color: var(--clr-text-2);
  cursor: default; transition: background 0.12s, color 0.12s;
}
.fd-chip:hover { background: var(--clr-surface-2); color: var(--clr-text); }
.fd-dot { width: 9px; height: 9px; border-radius: 50%; flex-shrink: 0; }
.fd-label { white-space: nowrap; max-width: 160px; overflow: hidden; text-overflow: ellipsis; }
.fd-count { font-size: 10px; color: var(--clr-text-3); background: var(--clr-bg); border-radius: 100px; padding: 0 6px; }
.fd-empty { font-size: 12px; color: var(--clr-text-3); white-space: nowrap; }
</style>
