<template>
  <!-- Kanban by maturity stage (Concept → Series). Drag a card to advance it. -->
  <div class="bv">
    <div
      v-for="col in columns"
      :key="col.key"
      class="bv-col"
      :class="{ over: dragOver === col.key }"
      @dragover.prevent="dragOver = col.key"
      @dragleave="dragOver = (dragOver === col.key ? null : dragOver)"
      @drop="onDrop(col.key)"
    >
      <div class="bv-col-head">
        <span class="bv-col-name">{{ col.label }}</span>
        <span class="bv-col-n">{{ col.items.length }}</span>
      </div>
      <div class="bv-cards">
        <div
          v-for="m in col.items"
          :key="m.id"
          class="bv-card"
          :draggable="!readOnly"
          :style="{ borderLeftColor: m.color || areaColor(m) || '#8a8a8e' }"
          @dragstart="dragId = m.id"
          @dragend="dragId = null; dragOver = null"
          @click="$emit('edit', m)"
        >
          <span class="bv-card-title">{{ m.title }}</span>
          <div class="bv-card-foot">
            <span class="bv-card-meta">{{ typeLabel(m) }}<template v-if="m.when"> · {{ m.when }}</template></span>
            <span v-if="m.assigneeId" class="bv-av" :title="memberName(m.assigneeId)" @click.stop="openProfile(memberById(m.assigneeId), $event)">{{ memberInitials(m.assigneeId) }}</span>
          </div>
        </div>
        <div v-if="!col.items.length" class="bv-col-empty">—</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { store, itemTypes, MATURITY_STAGES, useAppStore, memberInitials, memberName, memberById, openProfile } from '../stores/useAppStore.js'

const props = defineProps({ readOnly: { type: Boolean, default: false } })
defineEmits(['edit'])

const { updateMilestone } = useAppStore()
const dragId = ref(null)
const dragOver = ref(null)

const COLS = [{ key: 0, label: 'No stage' }, ...MATURITY_STAGES.map((s, i) => ({ key: i + 1, label: s }))]

function typeLabel(m) {
  const k = m.typeKey || m.kind || 'milestone'
  return itemTypes.list.find(t => t.key === k)?.label || k
}
function areaColor(m) { return store.swimlanes.find(s => s.id === m.swimlaneId)?.color }

const columns = computed(() => COLS.map(c => ({ ...c, items: store.milestones.filter(m => (m.maturity || 0) === c.key) })))

function onDrop(key) {
  dragOver.value = null
  if (props.readOnly || !dragId.value) return
  updateMilestone(dragId.value, { maturity: key === 0 ? null : key })
  dragId.value = null
}
</script>

<style scoped>
.bv { display: flex; gap: 14px; padding: 18px; overflow-x: auto; align-items: flex-start; min-height: 60vh; }
.bv-col { flex: 0 0 240px; background: var(--clr-bg); border: 1px solid var(--clr-border-light); border-radius: var(--r-lg); padding: 10px; transition: background 0.12s, border-color 0.12s; }
.bv-col.over { border-color: var(--clr-accent); background: var(--clr-surface-2); }
.bv-col-head { display: flex; align-items: center; gap: 8px; padding: 2px 4px 10px; }
.bv-col-name { font-weight: 700; font-size: 13px; color: var(--clr-text); }
.bv-col-n { font-size: 11px; color: var(--clr-text-3); background: var(--clr-surface-2); border-radius: 100px; padding: 1px 8px; }
.bv-cards { display: flex; flex-direction: column; gap: 8px; }
.bv-card {
  background: var(--clr-surface); border: 1px solid var(--clr-border-light); border-left: 3px solid #8a8a8e;
  border-radius: var(--r-md); padding: 9px 11px; cursor: pointer;
  display: flex; flex-direction: column; gap: 3px; box-shadow: var(--sh-sm);
  transition: box-shadow 0.12s, transform 0.06s;
}
.bv-card:hover { box-shadow: var(--sh-md); }
.bv-card:active { transform: scale(0.99); }
.bv-card-title { font-size: 13px; font-weight: 600; color: var(--clr-text); }
.bv-card-foot { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.bv-card-meta { font-size: 11px; color: var(--clr-text-3); }
.bv-av {
  width: 20px; height: 20px; border-radius: 50%; flex-shrink: 0; cursor: pointer;
  display: inline-flex; align-items: center; justify-content: center;
  font-size: 10px; font-weight: 700; color: #fff; background: var(--clr-accent);
}
.bv-av:hover { filter: brightness(1.1); }
.bv-col-empty { font-size: 12px; color: var(--clr-text-3); text-align: center; padding: 10px 0; }
</style>
