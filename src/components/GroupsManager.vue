<template>
  <div class="card-stack">
    <div class="card">
      <p class="section-label">Groups</p>
      <p class="card-hint">
        Groups tag items across areas — the legend and filters use them. Assign items
        to a group from an item's <strong>Groups</strong> tab.
      </p>

      <div v-if="groups.list.length" class="gm-list">
        <div v-for="g in groups.list" :key="g.id" class="gm-row">
          <input
            type="color"
            class="gm-color"
            :value="g.color || '#0A84FF'"
            title="Colour"
            @input="updateGroup(g.id, { color: $event.target.value })"
          />
          <input
            class="field-input"
            :value="g.name"
            @change="updateGroup(g.id, { name: ($event.target.value || '').trim() || 'Group' })"
          />
          <span class="gm-count">{{ (g.itemIds || []).length }} item{{ (g.itemIds || []).length === 1 ? '' : 's' }}</span>
          <button class="link-btn danger" @click="onDelete(g)">Delete</button>
        </div>
      </div>
      <div v-else class="empty">No groups yet.</div>

      <div class="gm-new">
        <input v-model="newName" class="field-input" placeholder="New group name" @keyup.enter="onAdd" />
        <button class="btn-add" :disabled="!newName.trim()" @click="onAdd">Add group</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { groups, useAppStore } from '../stores/useAppStore.js'

const { addGroup, updateGroup, deleteGroup } = useAppStore()
const newName = ref('')

function onAdd() {
  const n = newName.value.trim()
  if (!n) return
  addGroup(n)
  newName.value = ''
}
function onDelete(g) {
  if (!confirm(`Delete group "${g.name}"? Items keep their data — they're just untagged from this group.`)) return
  deleteGroup(g.id)
}
</script>

<style scoped>
.gm-list { display: flex; flex-direction: column; gap: 6px; }
.gm-row { display: flex; align-items: center; gap: 8px; }
.gm-color { width: 30px; height: 30px; flex-shrink: 0; padding: 0; background: none;
  border: 1px solid var(--clr-border-light); border-radius: var(--r-sm); cursor: pointer; }
.gm-count { font-size: 12px; color: var(--clr-text-3); white-space: nowrap; flex-shrink: 0; }
.gm-new { display: flex; gap: 8px; margin-top: 4px; }
.gm-row .field-input, .gm-new .field-input { flex: 1; min-width: 0; padding: 7px 10px; font-size: 13px;
  color: var(--clr-text); background: var(--clr-surface); border: 1px solid var(--clr-border-light); border-radius: var(--r-md); }
.btn-add { padding: 8px 14px; font-size: 13px; font-weight: 600; white-space: nowrap;
  color: var(--clr-accent); background: rgba(0,113,227,0.08); border-radius: var(--r-md); }
.btn-add:disabled { opacity: 0.4; cursor: not-allowed; }
.link-btn { background: none; font-size: 12px; font-weight: 600; color: var(--clr-accent); padding: 4px 6px; border-radius: var(--r-sm); flex-shrink: 0; }
.link-btn.danger { color: var(--clr-danger); }
.link-btn:hover { text-decoration: underline; }
</style>
