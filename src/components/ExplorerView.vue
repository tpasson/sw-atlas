<template>
  <div class="explorer">
    <div class="ev-bar">
      <div class="ev-modes">
        <button v-for="m in MODES" :key="m.key" class="ev-mode" :class="{ on: mode === m.key }" @click="setMode(m.key)">{{ m.label }}</button>
      </div>
    </div>

    <div v-if="mode === 'table'" class="ev-page"><TableView @edit="$emit('edit', $event)" /></div>
    <div v-else-if="mode === 'board'" class="ev-page"><BoardView :read-only="readOnly" @edit="$emit('edit', $event)" /></div>

    <!-- VS-Code layout: type tree on the left, the selected item's content in the centre. -->
    <div v-else class="ev-split">
      <aside class="ev-tree" :style="{ width: treeWidth + 'px' }">
        <div class="ev-tree-head">Explorer</div>
        <div v-for="g in folders" :key="g.key" class="ex-node">
          <div class="ex-row" @click="toggle(g.key)">
            <svg class="ex-chev" :class="{ open: isOpen(g.key) }" width="10" height="10" viewBox="0 0 10 10" fill="none"><path d="M3 1.5L6.5 5L3 8.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
            <MarkerIcon :shape="g.type.icon || 'l:Diamond'" :color="g.type.color || '#8a8a8e'" :size="14" :fill="true" />
            <span class="ex-row-label">{{ g.type.label }}</span>
            <span class="ex-row-count">{{ g.items.length }}</span>
            <button v-if="!readOnly" class="ex-row-add" :title="'New ' + g.type.label" @click.stop="$emit('add', g.type)">+</button>
          </div>
          <div v-if="isOpen(g.key)" class="ex-children">
            <button
              v-for="m in g.items"
              :key="m.id"
              class="ex-leaf"
              :class="{ on: selectedId === m.id }"
              @click="selectedId = m.id"
            >
              <span class="ex-leaf-dot" :style="{ background: m.color || laneColor(m) || '#8a8a8e' }"></span>
              <span class="ex-leaf-title">{{ m.title }}</span>
            </button>
            <div v-if="!g.items.length" class="ex-leaf-empty">— empty —</div>
          </div>
        </div>

        <!-- Source control: mirrored repo content is NOT mixed in with milestones.
             It lives in its own SCM → repo → category (issue/release/…) → items tree. -->
        <div v-if="scmRepos.length" class="ex-node">
          <div class="ex-row" @click="toggle('scm')">
            <svg class="ex-chev" :class="{ open: isOpen('scm') }" width="10" height="10" viewBox="0 0 10 10" fill="none"><path d="M3 1.5L6.5 5L3 8.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
            <GitPullRequest :size="14" />
            <span class="ex-row-label">SCM</span>
            <span class="ex-row-count">{{ scmCount }}</span>
          </div>
          <div v-if="isOpen('scm')" class="ex-children">
            <div v-for="repo in scmRepos" :key="repo.id" class="ex-node scm-repo">
              <div class="ex-row" @click="toggle('scm:' + repo.id)">
                <svg class="ex-chev" :class="{ open: isOpen('scm:' + repo.id) }" width="10" height="10" viewBox="0 0 10 10" fill="none"><path d="M3 1.5L6.5 5L3 8.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
                <span class="ex-leaf-dot" :style="{ background: repo.color || '#6E5494' }"></span>
                <span class="ex-row-label">{{ repo.name }}</span>
                <span class="ex-row-count">{{ repo.count }}</span>
              </div>
              <div v-if="isOpen('scm:' + repo.id)" class="ex-children">
                <div v-for="cat in repo.categories" :key="cat.id" class="ex-node scm-cat">
                  <div class="ex-row" @click="toggle('scm:' + repo.id + ':' + cat.id)">
                    <svg class="ex-chev" :class="{ open: isOpen('scm:' + repo.id + ':' + cat.id) }" width="10" height="10" viewBox="0 0 10 10" fill="none"><path d="M3 1.5L6.5 5L3 8.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
                    <span class="ex-row-label scm-cat-label">{{ cat.name }}</span>
                    <span class="ex-row-count">{{ cat.items.length }}</span>
                  </div>
                  <div v-if="isOpen('scm:' + repo.id + ':' + cat.id)" class="ex-children">
                    <button
                      v-for="m in cat.items"
                      :key="m.id"
                      class="ex-leaf scm-leaf"
                      :class="{ on: selectedId === m.id }"
                      @click="selectedId = m.id"
                    >
                      <span class="ex-leaf-dot" :style="{ background: m.color || '#8a8a8e' }"></span>
                      <span class="ex-leaf-title">{{ m.title }}</span>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="!folders.length && !scmRepos.length" class="ev-tree-blank">No artifacts yet.</div>
      </aside>

      <div class="ev-resizer" :class="{ dragging: resizing }" title="Drag to resize · double-click to reset" @mousedown.prevent="startResize" @dblclick="resetWidth"></div>

      <section class="ev-detail">
        <ItemDetail v-if="selected" :item="selected" :read-only="readOnly" @edit="$emit('edit', selected)" />
        <div v-else class="ev-detail-empty">
          <p>Select an item in the tree to see its content.</p>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { GitPullRequest } from 'lucide-vue-next'
import { store, itemTypes } from '../stores/useAppStore.js'
import MarkerIcon from './MarkerIcon.vue'
import TableView from './TableView.vue'
import BoardView from './BoardView.vue'
import ItemDetail from './ItemDetail.vue'

defineProps({ readOnly: { type: Boolean, default: false } })
defineEmits(['edit', 'add'])

const MODES = [{ key: 'folders', label: 'Tree' }, { key: 'table', label: 'Table' }, { key: 'board', label: 'Board' }]
const MODE_KEY = 'atlas-explorer-mode'
const mode = ref(['folders', 'table', 'board'].includes(localStorage.getItem(MODE_KEY)) ? localStorage.getItem(MODE_KEY) : 'folders')
function setMode(k) { mode.value = k; try { localStorage.setItem(MODE_KEY, k) } catch { /* ignore */ } }


// One folder per type: every defined type (so you can add into empty ones) plus
// any orphan type keys that still have items. Sorted by item count.
const folders = computed(() => {
  const byKey = new Map()
  for (const t of itemTypes.list) byKey.set(t.key, { key: t.key, type: t, items: [] })
  for (const m of store.milestones) {
    if (m.sourceSystem) continue // mirrored SCM items get their own tree, below
    const k = m.typeKey || m.kind || 'milestone'
    let g = byKey.get(k)
    if (!g) { g = { key: k, type: { key: k, label: k, icon: 'l:Diamond', color: '' }, items: [] }; byKey.set(k, g) }
    g.items.push(m)
  }
  return [...byKey.values()]
    .filter(g => g.items.length > 0 || (g.type && !g.type.builtin))
    .sort((a, b) => (b.items.length - a.items.length) || a.type.label.localeCompare(b.type.label))
})

// SCM tree: one node per connected repo (source swimlane), split into its
// categories (the repo's sub-lanes: Releases / Tags / Issues / Pull requests),
// with the mirrored items underneath.
const scmRepos = computed(() =>
  store.swimlanes
    .filter(sw => sw.sourceSystem)
    .map(sw => {
      const mine = store.milestones.filter(m => m.swimlaneId === sw.id)
      const categories = (sw.subLanes || [])
        .map(sl => ({ id: sl.id, name: sl.name, items: mine.filter(m => m.subLaneId === sl.id) }))
        .filter(c => c.items.length)
      const orphans = mine.filter(m => !m.subLaneId)
      if (orphans.length) categories.push({ id: '_root', name: 'Other', items: orphans })
      return { id: sw.id, name: sw.name, color: sw.color, categories, count: mine.length }
    })
    .filter(r => r.count > 0))
const scmCount = computed(() => scmRepos.value.reduce((n, r) => n + r.count, 0))

function laneColor(m) { return store.swimlanes.find(s => s.id === m.swimlaneId)?.color }

// Tree expand/collapse (types open by default; collapse-set tracks closed ones).
const collapsed = ref(new Set())
function isOpen(key) { return !collapsed.value.has(key) }
function toggle(key) {
  const s = new Set(collapsed.value)
  s.has(key) ? s.delete(key) : s.add(key)
  collapsed.value = s
}

// Selected item → its content shows in the centre pane. Resolve by id so it stays
// in sync with edits and clears if the item is deleted.
const selectedId = ref(null)
const selected = computed(() => store.milestones.find(m => m.id === selectedId.value) || null)

// Draggable divider: resize the tree column (persisted; double-click resets).
const TREE_W_KEY = 'atlas-explorer-tree-w'
const DEFAULT_W = 280
function clampW(w) { return Math.min(560, Math.max(190, w || DEFAULT_W)) }
const treeWidth = ref(clampW(parseInt(localStorage.getItem(TREE_W_KEY) || '', 10)))
const resizing = ref(false)
function startResize(e) {
  resizing.value = true
  const startX = e.clientX
  const startW = treeWidth.value
  function onMove(ev) { treeWidth.value = clampW(startW + (ev.clientX - startX)) }
  function onUp() {
    resizing.value = false
    try { localStorage.setItem(TREE_W_KEY, String(treeWidth.value)) } catch { /* ignore */ }
    window.removeEventListener('mousemove', onMove)
    window.removeEventListener('mouseup', onUp)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }
  window.addEventListener('mousemove', onMove)
  window.addEventListener('mouseup', onUp)
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}
function resetWidth() {
  treeWidth.value = DEFAULT_W
  try { localStorage.setItem(TREE_W_KEY, String(DEFAULT_W)) } catch { /* ignore */ }
}
</script>

<style scoped>
.explorer { flex: 1; min-height: 0; display: flex; flex-direction: column; }

.ev-bar { display: flex; justify-content: flex-end; padding: 9px 18px; border-bottom: 1px solid var(--clr-border-light); flex-shrink: 0; }
.ev-modes { display: inline-flex; gap: 2px; background: var(--clr-surface-2); border-radius: 100px; padding: 2px; }
.ev-mode { font-size: 12px; font-weight: 600; color: var(--clr-text-2); background: transparent; border-radius: 100px; padding: 5px 14px; transition: background 0.12s, color 0.12s; }
.ev-mode.on { background: var(--clr-surface); color: var(--clr-text); box-shadow: var(--sh-sm); }
.ev-mode:hover:not(.on) { color: var(--clr-text); }

.ev-page { flex: 1; min-height: 0; overflow: auto; }

.ev-split { flex: 1; min-height: 0; display: flex; }
.ev-tree {
  flex-shrink: 0; overflow-y: auto;
  background: var(--clr-bg); padding: 8px 0;
}
.ev-resizer { width: 7px; flex-shrink: 0; cursor: col-resize; position: relative; background: transparent; }
.ev-resizer::before {
  content: ''; position: absolute; top: 0; bottom: 0; left: 50%; transform: translateX(-50%);
  width: 1px; background: var(--clr-border-light); transition: background 0.12s, width 0.12s;
}
.ev-resizer:hover::before, .ev-resizer.dragging::before { width: 2px; background: var(--clr-accent); }
.ev-tree-head { font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.6px; color: var(--clr-text-3); padding: 4px 14px 8px; }
.ev-tree-blank { padding: 14px; font-size: 13px; color: var(--clr-text-3); }

.ex-row { display: flex; align-items: center; gap: 7px; padding: 5px 12px; cursor: pointer; user-select: none; transition: background 0.1s; }
.ex-row:hover { background: var(--clr-surface-2); }
.ex-chev { color: var(--clr-text-3); flex-shrink: 0; transition: transform 0.12s; }
.ex-chev.open { transform: rotate(90deg); }
.ex-row-label { font-weight: 600; font-size: 13px; color: var(--clr-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.ex-row-count { font-size: 10px; color: var(--clr-text-3); background: var(--clr-surface); border-radius: 100px; padding: 1px 7px; }
.ex-row-add { margin-left: auto; width: 20px; height: 20px; border-radius: 5px; flex-shrink: 0; display: inline-flex; align-items: center; justify-content: center; font-size: 15px; line-height: 1; color: var(--clr-accent); background: rgba(0,113,227,0.08); }
.ex-row-add:hover { background: rgba(0,113,227,0.18); }

.ex-children { display: flex; flex-direction: column; }
.ex-leaf { display: flex; align-items: center; gap: 8px; text-align: left; background: none; padding: 5px 12px 5px 31px; transition: background 0.1s; position: relative; }
.ex-leaf:hover { background: var(--clr-surface-2); }
.ex-leaf.on { background: var(--clr-surface-2); }
.ex-leaf.on::before { content: ''; position: absolute; left: 0; top: 3px; bottom: 3px; width: 2px; background: var(--clr-accent); }
.ex-leaf-dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.ex-leaf-title { font-size: 13px; color: var(--clr-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ex-leaf-empty { padding: 4px 12px 6px 31px; font-size: 12px; color: var(--clr-text-3); }

/* SCM sub-tree: repo (level 1) → category (level 2) → items (leaves). */
.scm-repo > .ex-row { padding-left: 27px; }
.scm-cat > .ex-row { padding-left: 44px; }
.scm-cat-label { text-transform: uppercase; font-size: 11px; font-weight: 700; letter-spacing: 0.4px; color: var(--clr-text-2); }
.scm-leaf { padding-left: 61px; }

.ev-detail { flex: 1; min-width: 0; overflow-y: auto; }
.ev-detail-empty { height: 100%; display: flex; align-items: center; justify-content: center; color: var(--clr-text-3); font-size: 14px; }
</style>
