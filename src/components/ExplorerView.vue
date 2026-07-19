<template>
  <div class="explorer">
    <div v-if="ui.explorerMode === 'table'" class="ev-page"><TableView @edit="$emit('edit', $event)" /></div>
    <div v-else-if="ui.explorerMode === 'board'" class="ev-page"><BoardView :read-only="readOnly" @edit="$emit('edit', $event)" /></div>

    <!-- VS-Code layout: type tree on the left, the selected item's content in the centre. -->
    <div v-else class="ev-split">
      <aside class="ev-tree" :style="{ width: treeWidth + 'px' }">
        <div class="ev-tree-head">
          <span>Explorer</span>
          <div class="ev-sortbtns">
            <button v-for="o in SORT_OPTIONS" :key="o.key" type="button" class="ev-sortbtn" :class="{ on: sortKey === o.key }" :title="'Sort: ' + o.label" @click="sortKey = o.key"><component :is="o.icon" :size="14" :stroke-width="2.2" /></button>
          </div>
        </div>
        <template v-for="g in groupedFolders" :key="g.key">
        <div v-if="g.showHeader" class="ex-group-head">{{ g.groupLabel }}</div>
        <div class="ex-node">
          <div class="ex-row" @click="toggle(g.key)">
            <svg class="ex-chev" :class="{ open: isOpen(g.key) }" width="10" height="10" viewBox="0 0 10 10" fill="none"><path d="M3 1.5L6.5 5L3 8.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
            <MarkerIcon :shape="g.type.icon || 'l:Diamond'" :color="g.type.color || '#8a8a8e'" :size="14" :fill="g.type.fill !== false" />
            <span class="ex-row-label" :style="{ color: g.type.color || '#8a8a8e' }">{{ g.type.label }}</span>
            <span class="ex-row-count">{{ g.items.length }}</span>
            <button v-if="!readOnly" class="ex-row-add" :title="'New ' + g.type.label" @click.stop="$emit('add', g.type)">+</button>
          </div>
          <div v-if="isOpen(g.key)" class="ex-children">
            <button
              v-for="m in g.items"
              :key="m.id"
              class="ex-leaf"
              :class="{ on: selectedId === m.id }"
              @click="selectLeaf(m.id)"
            >
              <MarkerIcon class="ex-leaf-ico" :shape="g.type.icon || 'l:Diamond'" :color="dotColor(m)" :size="13" :fill="g.type.fill !== false" />
              <span class="ex-leaf-title">{{ m.title }}</span>
              <span class="ex-leaf-ver">v{{ m.version || 1 }}</span>
            </button>
            <div v-if="!g.items.length" class="ex-leaf-empty">— empty —</div>
          </div>
        </div>
        </template>

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
                      @click="selectLeaf(m.id)"
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

        <!-- Change Requests: proposals about items — their own branch, by status. -->
        <div v-if="crGroups.length" class="ex-node">
          <div class="ex-row" @click="toggle('cr')">
            <svg class="ex-chev" :class="{ open: isOpen('cr') }" width="10" height="10" viewBox="0 0 10 10" fill="none"><path d="M3 1.5L6.5 5L3 8.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
            <ClipboardCheck :size="14" />
            <span class="ex-row-label">Change Requests</span>
            <span class="ex-row-count">{{ crCount }}</span>
          </div>
          <div v-if="isOpen('cr')" class="ex-children">
            <div v-for="grp in crGroups" :key="grp.key" class="ex-node">
              <div class="ex-row" @click="toggle('cr:' + grp.key)">
                <svg class="ex-chev" :class="{ open: isOpen('cr:' + grp.key) }" width="10" height="10" viewBox="0 0 10 10" fill="none"><path d="M3 1.5L6.5 5L3 8.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
                <span class="ex-leaf-dot" :style="{ background: grp.color }"></span>
                <span class="ex-row-label">{{ grp.label }}</span>
                <span class="ex-row-count">{{ grp.items.length }}</span>
              </div>
              <div v-if="isOpen('cr:' + grp.key)" class="ex-children">
                <button
                  v-for="cr in grp.items"
                  :key="cr.id"
                  class="ex-leaf"
                  :class="{ on: ui.focusCrId === cr.id }"
                  @click="openChangeRequest(cr.id)"
                >
                  <span class="ex-leaf-dot" :style="{ background: grp.color }"></span>
                  <span class="ex-leaf-title"><span class="ex-cr-kind">{{ cr.kind === 'create' ? '＋' : '↻' }}</span>{{ crTitle(cr) }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <div v-if="!folders.length && !scmRepos.length && !crGroups.length" class="ev-tree-blank">No artifacts yet.</div>
      </aside>

      <div class="ev-resizer" :class="{ dragging: resizing }" title="Drag to resize · double-click to reset" @mousedown.prevent="startResize" @dblclick="resetWidth"></div>

      <section class="ev-detail" :class="{ embedded: !!selected }">
        <MilestoneModal
          v-if="selected"
          :key="selected.id"
          embedded
          mode="edit"
          :milestone="selected"
          :swimlane="editSwimlane"
          :sub-lane="editSubLane"
        />
        <div v-else class="ev-detail-empty">
          <p>Select an item in the tree to see its content.</p>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { GitPullRequest, ClipboardCheck, ArrowDownAZ, ArrowDownZA, Clock, ArrowDown10, ArrowDown01 } from 'lucide-vue-next'
import { store, itemTypes, itemStatus, statusColor, ui, pushNav, workspace, changeRequests, openChangeRequest, setExplorerMode } from '../stores/useAppStore.js'
import MarkerIcon from './MarkerIcon.vue'
import TableView from './TableView.vue'
import BoardView from './BoardView.vue'
import MilestoneModal from './MilestoneModal.vue'

defineProps({ readOnly: { type: Boolean, default: false } })
defineEmits(['edit', 'add'])


// Item ordering within each folder — stable (so nothing jumps when you add one).
const SORT_KEY = 'atlas-explorer-sort'
const SORT_OPTIONS = [
  { key: 'name', label: 'Name (A–Z)', icon: ArrowDownAZ },
  { key: 'name-desc', label: 'Name (Z–A)', icon: ArrowDownZA },
  { key: 'updated', label: 'Recently updated', icon: Clock },
  { key: 'version', label: 'Version (high–low)', icon: ArrowDown10 },
  { key: 'version-asc', label: 'Version (low–high)', icon: ArrowDown01 },
]
const sortKey = ref(SORT_OPTIONS.some(o => o.key === localStorage.getItem(SORT_KEY)) ? localStorage.getItem(SORT_KEY) : 'name')
watch(sortKey, (v) => { try { localStorage.setItem(SORT_KEY, v) } catch { /* ignore */ } })
function sortItems(items) {
  const name = (m) => (m.title || '').toLowerCase()
  const by = sortKey.value
  return [...items].sort((a, b) => {
    if (by === 'name-desc') return name(b).localeCompare(name(a))
    if (by === 'updated') return (b.updatedAt || b.createdAt || '').localeCompare(a.updatedAt || a.createdAt || '') || name(a).localeCompare(name(b))
    if (by === 'version') return ((b.version || 1) - (a.version || 1)) || name(a).localeCompare(name(b))
    if (by === 'version-asc') return ((a.version || 1) - (b.version || 1)) || name(a).localeCompare(name(b))
    return name(a).localeCompare(name(b)) // name A–Z
  })
}


// One folder per type: every defined type (so you can add into empty ones) plus
// any orphan type keys that still have items. Folders stay alphabetical and items
// follow the chosen sort — both stable, so adding an item never reshuffles things.
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
  const out = [...byKey.values()].filter(g => g.items.length > 0 || (g.type && !g.type.builtin))
  for (const g of out) g.items = sortItems(g.items)
  return out.sort((a, b) => (a.type.label || '').localeCompare(b.type.label || ''))
})

// Group the type-folders by the type's behaviour family — Timeline / Backlog /
// Structure — in a fixed order (stable). Adaptive: headers only appear when more
// than one family is present; a single-family workspace stays flat.
const GROUP_ORDER = ['timeline', 'backlog', 'structure']
const GROUP_LABEL = { timeline: 'Timeline', backlog: 'Backlog', structure: 'Structure' }
function familyGroup(fam) {
  if (fam === 'work-item') return 'backlog'
  if (fam === 'container') return 'structure'
  return 'timeline' // timeline-point / timeline-range (+ unknown legacy → timeline)
}
const folderGroups = computed(() => {
  const byGroup = { timeline: [], backlog: [], structure: [] }
  for (const f of folders.value) byGroup[familyGroup(f.type?.family)].push(f)
  return GROUP_ORDER.map(k => ({ key: k, label: GROUP_LABEL[k], folders: byGroup[k] })).filter(s => s.folders.length)
})
const showGroups = computed(() => folderGroups.value.length > 1)
// Flattened list with a header flag on the first folder of each group (one folder
// block in the template, no duplication). Flat when only one family exists.
const groupedFolders = computed(() => {
  if (!showGroups.value) return folders.value
  const out = []
  for (const grp of folderGroups.value) grp.folders.forEach((f, i) => out.push({ ...f, groupLabel: grp.label, showHeader: i === 0 }))
  return out
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

// Change Requests: their own Explorer branch, grouped by status (not artifacts).
const CR_STATUS = [
  { key: 'pending', label: 'Pending', color: '#FF9F0A' },
  { key: 'approved', label: 'Approved', color: '#30D158' },
  { key: 'rejected', label: 'Rejected', color: '#FF3B30' },
]
const crGroups = computed(() =>
  CR_STATUS
    .map(s => ({ ...s, items: changeRequests.list.filter(c => c.status === s.key) }))
    .filter(g => g.items.length))
const crCount = computed(() => changeRequests.list.length)
function crTitle(cr) { return cr.targetTitle || (cr.payload && cr.payload.title) || 'Untitled' }

function laneColor(m) { return store.swimlanes.find(s => s.id === m.swimlaneId)?.color }
// Leaf dot: the item's status tone (a quick at-a-glance state), else its area colour.
function dotColor(m) {
  const s = itemStatus(m)
  return s ? statusColor(s) : (laneColor(m) || '#8a8a8e')
}

// Tree expand/collapse (types open by default; collapse-set tracks closed ones).
// Folder collapse state persists per workspace (survives view switches + reloads).
const collapseKey = () => `atlas-explorer-collapsed:${workspace.slug || 'default'}`
function loadCollapsed() {
  try { const a = JSON.parse(localStorage.getItem(collapseKey()) || '[]'); return new Set(Array.isArray(a) ? a : []) } catch { return new Set() }
}
const collapsed = ref(loadCollapsed())
// The workspace slug loads async, so the initial ref may have read the "default"
// key. Re-load once the real slug arrives so a saved collapse state is restored.
watch(() => workspace.slug, () => { collapsed.value = loadCollapsed() })
function isOpen(key) { return !collapsed.value.has(key) }
function toggle(key) {
  const s = new Set(collapsed.value)
  s.has(key) ? s.delete(key) : s.add(key)
  collapsed.value = s
  try { localStorage.setItem(collapseKey(), JSON.stringify([...s])) } catch { /* ignore */ }
}

// Selected item → its content shows in the centre pane. Resolve by id so it stays
// in sync with edits and clears if the item is deleted.
// The open item lives in the store (ui.explorerItemId), which mirrors the URL —
// so browser back/forward restores the selection. Reads/writes go through it.
const selectedId = computed({ get: () => ui.explorerItemId, set: (v) => { ui.explorerItemId = v } })
const selected = computed(() => store.milestones.find(m => m.id === selectedId.value) || null)

// The Explorer detail pane IS the item form (embedded): it opens in read mode and
// flips to edit in place — one consistent layout. These feed the form its context.
const editSwimlane = computed(() => store.swimlanes.find(s => s.id === selected.value?.swimlaneId) || null)
const editSubLane = computed(() => editSwimlane.value?.subLanes?.find(s => s.id === selected.value?.subLaneId) || null)

// Clicking an item in the tree opens its live/head content (drops any version view)
// and pushes a history entry (so Back returns to the previously viewed item).
function selectLeaf(id) {
  ui.explorerItemVersion = null
  ui.explorerItemId = id
  pushNav({ view: 'explorer', item: id })
}

// React to the open item changing (tree click, deep-link, back/forward, cross-view
// jump): switch to the tree layout and expand its folder. The embedded item form
// itself reads ?v={n} and shows that revision — no separate pinned view.
watch([() => ui.explorerItemId, () => ui.explorerItemVersion], ([id]) => {
  if (!id) return
  setExplorerMode('folders')
  const m = store.milestones.find(x => x.id === id)
  const key = m ? (m.typeKey || m.kind || 'milestone') : null
  if (key && collapsed.value.has(key)) { const s = new Set(collapsed.value); s.delete(key); collapsed.value = s }
}, { immediate: true })

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
.ev-tree-head { display: flex; align-items: center; justify-content: space-between; font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.6px; color: var(--clr-text-3); padding: 4px 10px 8px 14px; }
.ev-sortbtns { display: inline-flex; gap: 2px; }
.ev-sortbtn { width: 24px; height: 24px; display: inline-flex; align-items: center; justify-content: center; border-radius: var(--r-sm); color: var(--clr-text-3); background: none; transition: background 0.12s, color 0.12s; }
.ev-sortbtn:hover { background: var(--clr-surface-2); color: var(--clr-text); }
.ev-sortbtn.on { background: var(--clr-accent); color: #fff; }
.ev-tree-blank { padding: 14px; font-size: 13px; color: var(--clr-text-3); }
.ex-group-head { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.6px; color: var(--clr-text-3); padding: 12px 10px 3px 14px; }
.ex-group-head:first-child { padding-top: 2px; }

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
.ex-leaf-ico { flex-shrink: 0; }
.ex-leaf-title { flex: 1; min-width: 0; font-size: 13px; color: var(--clr-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ex-cr-kind { display: inline-block; width: 13px; margin-right: 3px; color: var(--clr-text-3); font-weight: 700; }
.ex-leaf-ver { flex-shrink: 0; font-size: 11px; color: var(--clr-text-3); font-variant-numeric: tabular-nums; }
.ex-leaf-empty { padding: 4px 12px 6px 31px; font-size: 12px; color: var(--clr-text-3); }

/* SCM sub-tree: repo (level 1) → category (level 2) → items (leaves). */
.scm-repo > .ex-row { padding-left: 27px; }
.scm-cat > .ex-row { padding-left: 44px; }
.scm-cat-label { text-transform: uppercase; font-size: 11px; font-weight: 700; letter-spacing: 0.4px; color: var(--clr-text-2); }
.scm-leaf { padding-left: 61px; }

.ev-detail { flex: 1; min-width: 0; overflow-y: auto; }
/* When showing the embedded item form (read or edit), it manages its own scroll +
   console dock, so the pane is a fixed-height flex host. */
.ev-detail.embedded { overflow: hidden; display: flex; }
.ev-detail-empty { height: 100%; display: flex; align-items: center; justify-content: center; color: var(--clr-text-3); font-size: 14px; }
</style>
