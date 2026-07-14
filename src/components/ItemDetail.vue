<template>
  <!-- Inline "content" pane for a selected artifact (VS-Code editor-style). -->
  <article class="id">
    <header class="id-head">
      <div class="id-titlewrap">
        <MarkerIcon :shape="type?.icon || item.marker || 'l:Diamond'" :color="type?.color || '#8a8a8e'" :size="20" :fill="type?.fill !== false" />
        <h1 class="id-title">{{ item.title }}</h1>
      </div>
      <div class="id-actions">
        <button class="id-act" :class="{ done: copied === 'link' }" title="Copy a stable link to this item" @click="copy('link')">{{ copied === 'link' ? 'Copied' : 'Copy link' }}</button>
        <button class="id-act" :class="{ done: copied === 'json' }" title="Copy this item as JSON" @click="copy('json')">{{ copied === 'json' ? 'Copied' : 'JSON' }}</button>
        <button class="id-act" :class="{ done: copied === 'yaml' }" title="Copy this item as YAML" @click="copy('yaml')">{{ copied === 'yaml' ? 'Copied' : 'YAML' }}</button>
        <button v-if="!readOnly" class="id-edit" @click="$emit('edit')">Edit</button>
      </div>
    </header>

    <div class="id-chips">
      <span class="id-chip">{{ type?.label || item.typeKey || item.kind }}</span>
      <span v-if="status" class="id-status" :style="statusStyle">{{ status.label }}</span>
      <button v-if="!item.sourceSystem && !pinnedVersion" class="id-hist-btn" @click="showHistory = !showHistory"><History :size="13" /> {{ showHistory ? 'Hide history' : 'History' }} · v{{ item.version || 1 }}</button>
      <span v-else-if="pinnedVersion" class="id-chip id-pin">Pinned · v{{ pinnedVersion }}</span>
      <button v-if="item.assigneeId" type="button" class="id-chip id-assignee" title="View profile" @click.stop="openProfile(memberById(item.assigneeId), $event)"><span class="id-av">{{ initials(item.assigneeId) }}</span>{{ memberName(item.assigneeId) }}</button>
    </div>

    <div v-if="!item.sourceSystem" class="id-attrib">
      <span v-if="item.createdBy">Added by <strong>{{ who(item.createdBy) }}</strong><span v-if="item.createdAt"> · {{ fmtStamp(item.createdAt) }}</span></span>
      <span v-if="item.updatedBy && (item.version || 1) > 1">Last edit by <strong>{{ who(item.updatedBy) }}</strong><span v-if="item.updatedAt"> · {{ fmtStamp(item.updatedAt) }}</span></span>
    </div>

    <dl class="id-meta">
      <template v-for="r in metaRows" :key="r.k">
        <dt>{{ r.k }}</dt><dd :class="{ 'id-empty': !r.v }">{{ r.v || '—' }}</dd>
      </template>
    </dl>

    <section v-if="fieldRows.length" class="id-block">
      <h2 class="id-h2">Fields</h2>
      <dl class="id-meta">
        <template v-for="f in fieldRows" :key="f.k"><dt>{{ f.k }}</dt><dd :class="{ 'id-empty': !f.v }">{{ f.v || '—' }}</dd></template>
      </dl>
    </section>

    <section class="id-block">
      <h2 class="id-h2">Details</h2>
      <dl class="id-meta">
        <template v-for="d in details" :key="d.k"><dt>{{ d.k }}</dt><dd :class="{ 'id-empty': !d.v }">{{ d.v || '—' }}</dd></template>
      </dl>
    </section>

    <section class="id-block">
      <h2 class="id-h2">Dependencies</h2>
      <ul v-if="dependencies.length" class="id-deps">
        <li v-for="(d, i) in dependencies" :key="i"><span class="id-dep-rel">{{ d.rel }}</span> {{ d.title }}</li>
      </ul>
      <p v-else class="id-text id-empty">None.</p>
    </section>

    <section class="id-block">
      <h2 class="id-h2">Groups</h2>
      <div v-if="itemGroups.length" class="id-groups">
        <span v-for="g in itemGroups" :key="g.id" class="id-group" :style="{ background: (g.color || '#888') + '22', color: g.color || '#888' }">{{ g.name }}</span>
      </div>
      <p v-else class="id-text id-empty">None.</p>
    </section>

    <section v-if="showHistory" class="id-block">
      <h2 class="id-h2">Version history</h2>
      <ItemHistory :key="item.id" :item-id="item.id" />
    </section>
  </article>
</template>

<script setup>
import { ref, computed } from 'vue'
import { History } from 'lucide-vue-next'
import { itemTypes, MATURITY_STAGES, store, MONTHS, memberName, memberInitials, memberById, openProfile, itemStatus, toneColor, groups, RELATIONSHIP_TYPES, itemLink, parseRef } from '../stores/useAppStore.js'

const status = computed(() => itemStatus(props.item))
const statusStyle = computed(() => {
  if (!status.value) return {}
  const c = toneColor(status.value.tone)
  return { color: c, background: c + '22', borderColor: c + '66' }
})
import MarkerIcon from './MarkerIcon.vue'
import ItemHistory from './ItemHistory.vue'

const props = defineProps({
  item: { type: Object, required: true },
  readOnly: { type: Boolean, default: false },
  pinnedVersion: { type: Number, default: 0 }, // >0 = showing an older revision's snapshot
})
defineEmits(['edit'])

const showHistory = ref(false)
function who(id) { return id ? (memberName(id) || 'someone') : 'system' }
function fmtStamp(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  return isNaN(d) ? '' : d.toLocaleString('en-GB', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

const type = computed(() => itemTypes.list.find(t => t.key === (props.item.typeKey || props.item.kind || 'milestone')))
const laneName = computed(() => store.swimlanes.find(s => s.id === props.item.swimlaneId)?.name || '')
const dateStr = computed(() => {
  const it = props.item
  if (it.startDate && it.endDate) return `${it.startDate} → ${it.endDate}`
  return it.when || (it.year && it.swimlaneId ? `${MONTHS[(it.month || 1) - 1]} ${it.year}` : '')
})
function initials(id) { return memberInitials(id) }

const metaRows = computed(() => {
  const it = props.item
  const rows = []
  if (laneName.value) rows.push({ k: 'Area', v: laneName.value })
  if (dateStr.value) rows.push({ k: 'Date', v: dateStr.value })
  rows.push({ k: 'Maturity', v: it.maturity ? MATURITY_STAGES[it.maturity - 1] : '' })
  rows.push({ k: 'Progress', v: it.progress != null ? it.progress + '%' : '' })
  return rows
})

// Details, dependencies and groups — always shown, so the detail view is complete.
const details = computed(() => [
  { k: 'What', v: props.item.what },
  { k: 'Why', v: props.item.why },
  { k: 'How', v: props.item.how },
])
const dependencies = computed(() => {
  const id = props.item.id
  const byId = new Map(store.milestones.map(m => [m.id, m]))
  const relLabel = (rel, forward) => {
    const r = RELATIONSHIP_TYPES.find(x => x.key === rel)
    return forward ? (r?.label || rel) : (r?.inverse || rel)
  }
  const out = []
  for (const l of store.links) {
    if (l.a === id && byId.has(l.b)) out.push({ rel: relLabel(l.rel, true), title: byId.get(l.b).title })
    else if (l.b === id && byId.has(l.a)) out.push({ rel: relLabel(l.rel, false), title: byId.get(l.a).title })
  }
  return out
})
const itemGroups = computed(() => groups.list.filter(g => (g.itemIds || []).includes(props.item.id)))

// Show every field the type defines — empty ones included, so it's clear which
// fields exist and which are still blank (rendered as "—" in the template).
// Reference fields resolve item ids to their titles.
// Resolve one reference entry to a display string. A pinned reference ("id@vN")
// shows its version; if the target has since advanced, that's made explicit so a
// stale pin is never mistaken for the latest.
function refDisplay(entry) {
  const { id, version } = parseRef(entry)
  const target = store.milestones.find(m => m.id === id)
  const title = target?.title || id
  if (!version) return title
  const head = target?.version
  return head && head > version ? `${title} · v${version} (latest v${head})` : `${title} · v${version}`
}
function fieldValue(f) {
  const v = props.item.data?.[f.key]
  if (f.type === 'reference') {
    const ids = Array.isArray(v) ? v : (v ? [v] : [])
    return ids.map(refDisplay).join(', ')
  }
  return Array.isArray(v) ? v.join(', ') : (v == null ? '' : String(v))
}
const fieldRows = computed(() =>
  (type.value?.fields || []).map(f => ({ k: f.label || f.key, v: fieldValue(f) })))

// ── Stable link + JSON/YAML export ───────────────────────────────────────────
const linkUrl = computed(() => itemLink(props.item.id, props.pinnedVersion || null))

// A clean, human-readable projection of the item (labels resolved, ids hidden)
// for the JSON/YAML "views" — the same object serialized either way.
function exportFieldValue(f) {
  const v = props.item.data?.[f.key]
  if (f.type === 'reference') {
    const ids = Array.isArray(v) ? v : (v ? [v] : [])
    const names = ids.map(refDisplay)
    return f.refMulti ? names : (names[0] || '')
  }
  if (Array.isArray(v)) return v
  return v == null ? '' : v
}
const exportObj = computed(() => {
  const it = props.item
  const o = {
    id: it.id,
    type: type.value?.label || it.typeKey || it.kind || 'item',
    title: it.title || '',
    status: status.value?.label || it.status || '',
    version: props.pinnedVersion || it.version || 1,
  }
  if (laneName.value) o.area = laneName.value
  if (dateStr.value) o.date = dateStr.value
  if (it.maturity) o.maturity = MATURITY_STAGES[it.maturity - 1]
  if (it.progress != null) o.progress = it.progress
  if (it.assigneeId) o.assignee = memberName(it.assigneeId) || it.assigneeId
  const fields = {}
  for (const f of (type.value?.fields || [])) fields[f.label || f.key] = exportFieldValue(f)
  if (Object.keys(fields).length) o.fields = fields
  const det = {}
  if (it.what) det.what = it.what
  if (it.why) det.why = it.why
  if (it.how) det.how = it.how
  if (Object.keys(det).length) o.details = det
  return o
})

// Minimal, dependency-free YAML for the export shape (scalars, arrays of scalars,
// one nested object level). Scalars are quoted whenever they could be misparsed.
function yamlScalar(v) {
  if (v == null) return '""'
  if (typeof v === 'number' || typeof v === 'boolean') return String(v)
  const s = String(v)
  const needsQuote = s === '' || /^[\s>|@`"'#%&*!?[\]{},-]/.test(s) || /:\s|\s#|[\n:]/.test(s) ||
    /\s$/.test(s) || /^(true|false|null|yes|no|~)$/i.test(s) || /^-?\d/.test(s)
  return needsQuote ? JSON.stringify(s) : s
}
function toYaml(obj, indent = 0) {
  const pad = '  '.repeat(indent)
  const lines = []
  for (const [k, v] of Object.entries(obj)) {
    if (Array.isArray(v)) {
      if (!v.length) { lines.push(`${pad}${k}: []`); continue }
      lines.push(`${pad}${k}:`)
      for (const el of v) lines.push(`${pad}  - ${yamlScalar(el)}`)
    } else if (v && typeof v === 'object') {
      const entries = Object.entries(v)
      if (!entries.length) { lines.push(`${pad}${k}: {}`); continue }
      lines.push(`${pad}${k}:`)
      lines.push(toYaml(v, indent + 1))
    } else {
      lines.push(`${pad}${k}: ${yamlScalar(v)}`)
    }
  }
  return lines.join('\n')
}

const copied = ref('')
let copiedTimer = null
async function copy(kind) {
  const text = kind === 'link' ? linkUrl.value
    : kind === 'json' ? JSON.stringify(exportObj.value, null, 2)
    : toYaml(exportObj.value)
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    const ta = document.createElement('textarea')
    ta.value = text; ta.style.position = 'fixed'; ta.style.opacity = '0'
    document.body.appendChild(ta); ta.select()
    try { document.execCommand('copy') } catch { /* ignore */ }
    document.body.removeChild(ta)
  }
  copied.value = kind
  if (copiedTimer) clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => { copied.value = '' }, 1400)
}

</script>

<style scoped>
.id { max-width: 720px; padding: 28px 32px; }
.id-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }
.id-titlewrap { display: flex; align-items: center; gap: 10px; min-width: 0; }
.id-title { font-size: 22px; font-weight: 700; color: var(--clr-text); line-height: 1.25; }
.id-actions { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
.id-act { font-size: 12px; font-weight: 600; color: var(--clr-text-2); background: var(--clr-surface-2); border-radius: var(--r-md); padding: 6px 11px; transition: background 0.12s, color 0.12s; }
.id-act:hover { background: var(--clr-surface); color: var(--clr-text); }
.id-act.done { color: #fff; background: var(--clr-positive, #30D158); }
.id-edit { flex-shrink: 0; background: var(--clr-accent); color: #fff; border-radius: var(--r-md); padding: 7px 16px; font-weight: 600; font-size: 13px; }
.id-edit:hover { background: var(--clr-accent-hover); }
.id-pin { color: #fff; background: var(--clr-warning, #FF9F0A); }

.id-chips { display: flex; flex-wrap: wrap; gap: 8px; margin: 12px 0 4px; }
.id-chip { font-size: 12px; font-weight: 600; color: var(--clr-text-2); background: var(--clr-surface-2); border-radius: 100px; padding: 4px 11px; }
.id-status { font-size: 12px; font-weight: 700; border-radius: 100px; padding: 4px 11px; border: 1px solid transparent; }
.id-assignee { display: inline-flex; align-items: center; gap: 6px; border: none; cursor: pointer; transition: filter 0.15s; }
.id-assignee:hover { filter: brightness(0.95); }

.id-attrib { display: flex; flex-wrap: wrap; align-items: center; gap: 6px 16px; margin: 12px 0 2px; font-size: 12px; color: var(--clr-text-3); }
.id-attrib strong { color: var(--clr-text-2); font-weight: 600; }
.id-hist-btn { display: inline-flex; align-items: center; gap: 5px; font-size: 12px; font-weight: 600; color: var(--clr-accent); background: var(--clr-surface-2); border-radius: 100px; padding: 4px 11px; }
.id-hist-btn:hover { background: var(--clr-surface); }
.id-av { width: 18px; height: 18px; border-radius: 50%; background: var(--clr-accent); color: #fff; font-size: 10px; font-weight: 700; display: inline-flex; align-items: center; justify-content: center; }

.id-meta { display: grid; grid-template-columns: 110px 1fr; gap: 6px 14px; margin: 16px 0; }
.id-meta dt { font-size: 12px; color: var(--clr-text-3); font-weight: 600; }
.id-meta dd { font-size: 13px; color: var(--clr-text); }
.id-meta dd.id-empty { color: var(--clr-text-3); }

.id-block { margin-top: 20px; }
.id-h2 { font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-text-3); margin-bottom: 7px; }
.id-text { font-size: 14px; color: var(--clr-text); line-height: 1.6; white-space: pre-wrap; }
.id-deps { list-style: none; display: flex; flex-direction: column; gap: 6px; }
.id-deps li { font-size: 13px; color: var(--clr-text); }
.id-dep-rel { display: inline-block; min-width: 96px; font-size: 11px; font-weight: 700; color: var(--clr-text-3); text-transform: uppercase; letter-spacing: 0.3px; }
.id-groups { display: flex; flex-wrap: wrap; gap: 6px; }
.id-group { font-size: 12px; font-weight: 600; border-radius: 100px; padding: 3px 10px; border: 1px solid currentColor; }
.id-link { font-size: 13px; color: var(--clr-accent); word-break: break-all; }
.id-link:hover { text-decoration: underline; }
</style>
