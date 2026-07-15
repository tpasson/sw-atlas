<template>
  <div class="card-stack">
    <div class="card">
      <span class="section-label">Item types</span>
      <p class="card-hint">
        Every timeline item is a type — its <strong>icon &amp; colour come from the type</strong>, and the legend
        fills itself from the types in use. Pick an icon (the key is auto-filled). Built-in types can be renamed,
        restyled and given fields too; only their key &amp; behaviour stay fixed.
      </p>

      <div v-for="(t, ti) in types" :key="t._uid" class="tm-type" :class="{ 'tm-builtin': t.builtin }">
        <div class="tm-head">
          <input class="ti tm-grow" v-model="t.label" :placeholder="t.builtin ? 'Name' : 'Type name (e.g. Bug)'" @input="onLabel(t)" />

          <span v-if="t.builtin" class="tm-fam" :title="'Built-in behaviour: ' + t.family">{{ familyShort(t.family) }}</span>
          <select v-else class="ti" v-model="t.family" style="width:128px" title="How it behaves">
            <option value="timeline-point">Timeline · point</option>
            <option value="timeline-range">Timeline · range</option>
            <option value="work-item">Backlog item</option>
            <option value="container">Folder</option>
          </select>

          <div class="tm-iconwrap">
            <button type="button" class="tm-iconbtn" :title="'Icon: ' + prettyShape(t.icon || 'l:Diamond')" @click="toggleIcon(ti)">
              <MarkerIcon :shape="t.icon || 'l:Diamond'" :color="t.color || '#8a8a8e'" :size="18" :fill="t.fill" />
              <svg width="9" height="9" viewBox="0 0 10 10" fill="none"><path d="M2 3.5L5 6.5L8 3.5" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/></svg>
            </button>
            <div v-if="iconOpen === ti" class="tm-iconpop">
              <input v-model="iconSearch" class="ti tm-iconsearch" placeholder="Search icons… (bug, flag, star, server…)" @click.stop />
              <div class="tm-icongrid">
                <button v-for="s in iconChoices" :key="s" type="button" class="tm-iconopt" :class="{ on: s === t.icon }" :title="prettyShape(s)" @click="pickIcon(t, s)">
                  <MarkerIcon :shape="s" :color="t.color || '#8a8a8e'" :size="16" :fill="t.fill" />
                </button>
                <span v-if="!iconChoices.length" class="card-hint">No icons match “{{ iconSearch }}”.</span>
              </div>
            </div>
          </div>

          <input type="color" class="tm-color" :value="t.color || '#0A84FF'" @input="t.color = $event.target.value" title="Colour" />
          <button type="button" class="tm-fill" :class="{ on: t.fill }" title="Filled vs outline icon" @click="t.fill = !t.fill">Fill</button>

          <label v-if="!t.builtin" class="tm-key">key<input class="tm-keyin" v-model="t.key" :disabled="t._persisted" @input="t._keyTouched = true"
            :title="t._persisted ? 'Type key is locked once saved — it identifies existing items; to change it, remove this type and add a new one' : 'Auto-filled from the name'" /></label>
          <span v-else class="tm-tag" title="Built-in — can't be deleted">built-in</span>
          <button v-if="!t.builtin" class="link danger" @click="removeType(ti)">Remove</button>
        </div>

        <div class="tm-fields">
          <div v-for="(f, fi) in t.fields" :key="fi" class="tm-field">
            <input class="ti tm-grow" v-model="f.label" placeholder="Field name (e.g. Severity)" @input="onFieldLabel(t, f)" />
            <select class="ti tm-ftype" v-model="f.type" :disabled="f._persisted"
              :title="f._persisted ? 'Field type is locked once saved — to change it, remove this field and add a new one' : 'Field type'">
              <option value="text">Text</option>
              <option value="number">Number</option>
              <option value="select">Select (one)</option>
              <option value="multiselect">Multi-select</option>
              <option value="date">Date</option>
              <option value="reference">Reference</option>
            </select>
            <input
              v-if="f.type === 'select' || f.type === 'multiselect'"
              class="ti tm-grow"
              :value="(f.options || []).join(', ')"
              placeholder="comma, separated, options"
              @change="f.options = $event.target.value.split(',').map(s => s.trim()).filter(Boolean)"
            />
            <template v-if="f.type === 'reference'">
              <select class="ti tm-ftype" :class="{ 'tm-need': !f.refType }" v-model="f.refType" :disabled="f._persisted && f._hadRefType"
                :title="f._persisted && f._hadRefType ? 'Target type is locked once saved — to change it, remove this field and add a new one' : 'References items of this type — required'">
                <option value="">— pick target type —</option>
                <option v-for="rt in refTypeOptions" :key="rt.key" :value="rt.key">{{ rt.label }}</option>
              </select>
              <button type="button" class="tm-toggle" :class="{ on: f.refMulti }" :disabled="f._persisted && f._hadRefType" title="Allow multiple references" @click="f.refMulti = !f.refMulti">Multiple</button>
            </template>
            <button type="button" class="tm-toggle" :class="{ on: f.required }" title="Must be filled in" @click="f.required = !f.required">Required</button>
            <label class="tm-key">key<input class="tm-keyin" v-model="f.key" :disabled="f._persisted" @input="f._keyTouched = true"
              :title="f._persisted ? 'Field key is locked once saved — it stores the item values; to change it, remove this field and add a new one' : 'Auto-filled from the field name'" /></label>
            <button class="tm-x" @click="removeField(t, fi)" title="Remove field">×</button>
          </div>
          <button class="link tm-addfield" @click="addField(t)">+ Field</button>
        </div>
        <div class="tm-wfrow">
          <span class="tm-wflabel">Workflow</span>
          <select class="ti tm-wfsel" v-model="t._wf">
            <option value="">None</option>
            <option v-for="w in workflowOptions" :key="w.key" :value="w.key">Shared · {{ w.label || w.key }}</option>
            <option value="__custom">Custom (own statuses)</option>
          </select>
          <span v-if="t._wf && t._wf !== '__custom'" class="tm-wfnote">→ edit “{{ wfLabel(t._wf) }}” under Shared workflows below</span>
        </div>
        <StatusEditor v-if="t._wf === '__custom'" :statuses="t.statuses" />
      </div>

      <button class="link" @click="addType">+ Add type</button>

      <p v-if="msg" class="data-msg" :class="{ ok: okMsg, err: !okMsg }">{{ msg }}</p>
    </div>

    <div class="card">
      <span class="section-label">Shared workflows</span>
      <p class="card-hint">
        Define a status flow <strong>once</strong> and point many types at it — editing the workflow updates
        every type that uses it, and its status-flow diagram is arranged just once, for all of them.
      </p>
      <div v-for="(w, wi) in workflowDrafts" :key="w._uid" class="tm-type">
        <div class="tm-head">
          <input class="ti tm-grow" v-model="w.label" placeholder="Workflow name (e.g. Standard)" @input="onWfLabel(w)" />
          <label class="tm-key">key<input class="tm-keyin" v-model="w.key" :disabled="w._persisted"
            :title="w._persisted ? 'Workflow key is locked once saved — types reference it' : 'Auto-filled from the name'" /></label>
          <button class="link danger" @click="removeWorkflow(wi)">Remove</button>
        </div>
        <StatusEditor :statuses="w.statuses" />
      </div>
      <button class="link" @click="addWorkflow">+ Add workflow</button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { itemTypes, saveItemTypes, workflows, saveWorkflows, DEFAULT_STATUSES, MARKER_LIBRARY, store } from '../stores/useAppStore.js'
import MarkerIcon from './MarkerIcon.vue'
import StatusEditor from './StatusEditor.vue'

let uid = 0
const nextUid = () => `t${uid++}`
// One unified list. Built-ins are editable (label/icon/colour/fill/fields) but
// keep their key + behaviour; only custom types are removable / re-keyed.
// _wf drives the workflow choice: '' = none, '__custom' = own inline statuses,
// otherwise a shared workflow key.
const types = ref(itemTypes.list.map(t => ({
  _uid: nextUid(), builtin: !!t.builtin, _persisted: true, _keyTouched: true,
  key: t.key, label: t.label, family: t.family, icon: t.icon, color: t.color || '', fill: t.fill !== false,
  _wf: t.workflowKey ? t.workflowKey : ((t.statuses || []).length ? '__custom' : ''),
  fields: (t.fields || []).map(f => ({ key: f.key, label: f.label, type: f.type, _persisted: true, _hadRefType: !!f.refType, options: [...(f.options || [])], required: !!f.required, refType: f.refType || '', refMulti: !!f.refMulti, _keyTouched: true })),
  statuses: (t.statuses || []).map(s => ({ key: s.key, label: s.label, tone: s.tone || 'neutral', to: [...(s.to || [])], _keyTouched: true })),
})))

// Editable drafts of the shared workflows (mirrors workflows.list). Its saved
// layout rides along untouched through the round-trip.
const workflowDrafts = ref(workflows.list.map(w => ({
  _uid: nextUid(), _persisted: true, _keyTouched: true, key: w.key, label: w.label, layout: w.layout,
  statuses: (w.statuses || []).map(s => ({ key: s.key, label: s.label, tone: s.tone || 'neutral', to: [...(s.to || [])], _keyTouched: true })),
})))
const workflowOptions = computed(() => workflowDrafts.value.filter(w => w.key))
function wfLabel(key) { return workflowDrafts.value.find(w => w.key === key)?.label || key }

// Types a reference field can point at (any defined type).
const refTypeOptions = computed(() => types.value.filter(x => x.key).map(x => ({ key: x.key, label: x.label || x.key })))

const msg = ref('')
const okMsg = ref(false)

function familyShort(f) {
  return f === 'timeline-range' ? 'range' : f === 'work-item' ? 'work' : f === 'container' ? 'folder' : 'point'
}

// ── Keys: derived from the label, kept unique, editable for power users ────────
function slugify(s) {
  return (s || '').toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '')
}
function uniqueKey(base, taken) {
  const root = base || 'type'
  let k = root, n = 2
  while (taken.has(k)) k = `${root}-${n++}`
  return k
}
function onLabel(t) {
  if (t.builtin || t._keyTouched) return
  const taken = new Set(types.value.filter(x => x !== t).map(x => x.key))
  t.key = uniqueKey(slugify(t.label), taken)
}
function onFieldLabel(t, f) {
  if (f._keyTouched) return
  const taken = new Set(t.fields.filter(x => x !== f).map(x => x.key))
  f.key = uniqueKey(slugify(f.label), taken)
}
function addType() {
  types.value.push({ _uid: nextUid(), builtin: false, _persisted: false, _keyTouched: false, key: '', label: '', family: 'timeline-point', icon: 'l:Diamond', color: '', fill: true, fields: [], statuses: [] })
}
function addField(t) {
  // A brand-new field: its type/target stay editable until the first save locks them.
  t.fields.push({ key: '', label: '', type: 'text', _persisted: false, _hadRefType: false, options: [], required: false, refType: '', refMulti: false, _keyTouched: false })
}

// ── Shared workflows ──────────────────────────────────────────────────────────
function onWfLabel(w) {
  if (w._keyTouched) return
  const taken = new Set(workflowDrafts.value.filter(x => x !== w).map(x => x.key))
  w.key = uniqueKey(slugify(w.label), taken)
}
function addWorkflow() {
  workflowDrafts.value.push({
    _uid: nextUid(), _persisted: false, _keyTouched: false, key: '', label: '', layout: undefined,
    statuses: DEFAULT_STATUSES.map(s => ({ key: s.key, label: s.label, tone: s.tone, to: [...s.to], _keyTouched: true })),
  })
}
function removeWorkflow(wi) {
  const w = workflowDrafts.value[wi]
  const users = types.value.filter(t => t._wf === w.key)
  if (users.length && !confirm(`Remove the “${w.label || w.key}” workflow?\n\n${users.length} type${users.length > 1 ? 's' : ''} use it and will fall back to no statuses until you pick another.`)) return
  for (const t of types.value) if (t._wf === w.key) t._wf = ''
  workflowDrafts.value.splice(wi, 1)
}

// ── Guardrails: schema changes must not silently drop existing item data ────────
// A saved field's TYPE is immutable (locked in the UI) — changing it would
// reinterpret stored values and break references. To change a field's type you
// remove it and add a new one, which surfaces the data impact via removeField().
const hasVal = (v) => !(v == null || v === '' || (Array.isArray(v) && v.length === 0))
function itemsWithField(typeKey, fieldKey) {
  return store.milestones.filter(m => (m.typeKey || m.kind || 'milestone') === typeKey && hasVal(m.data?.[fieldKey]))
}
function itemsOfType(typeKey) {
  return store.milestones.filter(m => (m.typeKey || m.kind || 'milestone') === typeKey)
}
function removeField(t, fi) {
  const f = t.fields[fi]
  const n = f._persisted ? itemsWithField(t.key, f.key).length : 0
  if (n && !confirm(`Remove “${f.label || f.key}”?\n\n${n} “${t.label || t.key}” item${n > 1 ? 's' : ''} hold a value in it. The value is NOT deleted — it stays stored but hidden, and reappears if you re-add a field with the same key.`)) return
  t.fields.splice(fi, 1)
}
function removeType(ti) {
  const t = types.value[ti]
  const n = itemsOfType(t.key).length
  if (n && !confirm(`Remove the “${t.label || t.key}” type?\n\n${n} item${n > 1 ? 's' : ''} use it. The items are kept, but they lose this type's fields, colour and workflow until a matching type exists again.`)) return
  types.value.splice(ti, 1)
}

// ── Icon picker ───────────────────────────────────────────────────────────────
const SHAPE_NAMES = { diamond: 'Diamond', circle: 'Circle', cone: 'Cone', flag: 'Flag', square: 'Square', triangleDown: 'Triangle', star: 'Star', hexagon: 'Hexagon', pentagon: 'Pentagon' }
function prettyShape(s) {
  if (!s) return 'Diamond'
  if (!s.startsWith('l:')) return SHAPE_NAMES[s] || s
  return s.slice(2).replace(/([a-z0-9])([A-Z])/g, '$1 $2')
}
const iconOpen = ref(null)
const iconSearch = ref('')
function toggleIcon(ti) { iconOpen.value = iconOpen.value === ti ? null : ti; iconSearch.value = '' }
function pickIcon(t, s) { t.icon = s; iconOpen.value = null }
const iconChoices = computed(() => {
  const q = iconSearch.value.trim().toLowerCase()
  const out = []
  for (const s of MARKER_LIBRARY) {
    if (q && !prettyShape(s).toLowerCase().includes(q) && !s.toLowerCase().includes(q)) continue
    out.push(s)
    if (out.length >= 90) break
  }
  return out
})
function onDocClick(e) { if (!e.target.closest('.tm-iconwrap')) iconOpen.value = null }
onMounted(() => document.addEventListener('click', onDocClick))
onUnmounted(() => document.removeEventListener('click', onDocClick))

// ── Save: built-ins keep key/family; custom get auto/deduped keys ─────────────
function cleanFields(fields) {
  const fieldKeys = new Set()
  const out = []
  for (const f of fields) {
    if (!f.label && !f.key) continue
    const fk = uniqueKey(f.key || slugify(f.label), fieldKeys)
    fieldKeys.add(fk)
    const o = { key: fk, label: f.label || fk, type: f.type, required: !!f.required, options: (f.type === 'select' || f.type === 'multiselect') ? (f.options || []) : [] }
    if (f.type === 'reference') { o.refType = f.refType || ''; o.refMulti = !!f.refMulti }
    out.push(o)
  }
  return out
}
function cleanStatuses(statuses) {
  const keys = new Set()
  const out = []
  for (const s of (statuses || [])) {
    if (!s.label && !s.key) continue
    const k = uniqueKey(s.key || slugify(s.label), keys)
    keys.add(k)
    out.push({ key: k, label: s.label || k, tone: s.tone || 'neutral', to: [...(s.to || [])] })
  }
  const valid = new Set(out.map(s => s.key))
  for (const s of out) s.to = s.to.filter(k => valid.has(k) && k !== s.key)
  return out
}
// Collect the shared workflows into a persistable payload (keyed, deduped).
function cleanWorkflows() {
  const keys = new Set()
  const out = []
  for (const w of workflowDrafts.value) {
    if (!w.label && !w.key && !w.statuses.length) continue
    const k = uniqueKey(w.key || slugify(w.label), keys)
    keys.add(k)
    const o = { key: k, label: w.label || k, statuses: cleanStatuses(w.statuses) }
    if (w.layout) o.layout = w.layout
    out.push(o)
  }
  return out
}
async function save() {
  const wfPayload = cleanWorkflows()
  const validWf = new Set(wfPayload.map(w => w.key))
  const typeKeys = new Set(types.value.filter(t => t.builtin).map(t => t.key))
  const payload = []
  for (const t of types.value) {
    // Map the workflow choice: shared key, own inline statuses, or none.
    const wfKey = (t._wf && t._wf !== '__custom' && validWf.has(t._wf)) ? t._wf : ''
    const statuses = t._wf === '__custom' ? cleanStatuses(t.statuses) : []
    if (t.builtin) {
      payload.push({ key: t.key, label: t.label || t.key, family: t.family, icon: t.icon || 'l:Diamond', color: t.color || '', fill: t.fill, fields: cleanFields(t.fields), workflowKey: wfKey, statuses })
      continue
    }
    if (!t.label && !t.key && !t.fields.length) continue
    const key = uniqueKey(t.key || slugify(t.label), typeKeys)
    typeKeys.add(key)
    payload.push({ key, label: t.label || key, family: t.family, icon: t.icon || 'l:Diamond', color: t.color || '', fill: t.fill, fields: cleanFields(t.fields), workflowKey: wfKey, statuses })
  }
  try {
    // Workflows first so the keys types reference already exist server-side.
    if (wfPayload.length || workflows.list.length) await saveWorkflows(wfPayload)
    await saveItemTypes(payload)
    // Everything is now persisted → lock structural props (keys, type, ref target).
    for (const t of types.value) {
      t._persisted = true
      for (const f of (t.fields || [])) { f._persisted = true; f._hadRefType = !!f.refType }
    }
    for (const w of workflowDrafts.value) w._persisted = true
    okMsg.value = true
    msg.value = 'Saved — icons, colours, workflows and the legend update everywhere.'
    return true
  } catch {
    fail('Save failed.')
    return false
  }
}
function fail(m) { okMsg.value = false; msg.value = m }

// The hosting modal drives persistence: its "Done" button calls save().
defineExpose({ save })
</script>

<style scoped>
.tm-type { border: 1px solid var(--clr-border-light); border-radius: var(--r-md); padding: 10px; display: flex; flex-direction: column; gap: 8px; }
.tm-builtin { background: var(--clr-surface-2); }
.tm-head { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; }
.tm-fam { font-size: 11px; color: var(--clr-text-3); background: var(--clr-bg); border-radius: 100px; padding: 4px 10px; white-space: nowrap; }
.tm-tag { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.4px; color: var(--clr-text-3); }
.tm-wfrow { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; padding-left: 10px; border-left: 2px solid var(--clr-border-light); }
.tm-wflabel { font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.4px; color: var(--clr-text-3); }
.tm-wfsel { min-width: 200px; }
.tm-wfnote { font-size: 11px; color: var(--clr-text-3); }
.tm-fields { display: flex; flex-direction: column; gap: 6px; padding-left: 10px; border-left: 2px solid var(--clr-border-light); }
.tm-field { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; padding: 3px 0; }
.tm-grow { flex: 1; min-width: 120px; }
.tm-ftype { width: 132px; flex-shrink: 0; }
.tm-addfield { margin-top: 2px; }

.ti {
  border: 1px solid var(--clr-border); border-radius: var(--r-sm);
  padding: 6px 9px; font-size: 13px; color: var(--clr-text); background: var(--clr-bg);
}
.ti:focus { outline: none; border-color: var(--clr-accent); }
.tm-color { width: 30px; height: 30px; border: none; background: none; padding: 0; cursor: pointer; flex-shrink: 0; }
.tm-fill, .tm-toggle { font-size: 12px; font-weight: 600; color: var(--clr-text-3); border: 1px solid var(--clr-border); border-radius: var(--r-sm); padding: 5px 10px; background: var(--clr-bg); white-space: nowrap; flex-shrink: 0; }
.tm-fill.on, .tm-toggle.on { color: var(--clr-accent); border-color: var(--clr-accent); background: rgba(0,113,227,0.08); }
.tm-x { width: 26px; height: 26px; flex-shrink: 0; display: inline-flex; align-items: center; justify-content: center; font-size: 17px; line-height: 1; color: var(--clr-text-3); background: none; border-radius: var(--r-sm); }
.tm-x:hover { color: var(--clr-danger); background: rgba(255,59,48,0.08); }

.tm-key { display: inline-flex; align-items: center; gap: 4px; font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.4px; color: var(--clr-text-3); flex-shrink: 0; }
.tm-keyin { width: 78px; border: 1px solid var(--clr-border-light); border-radius: var(--r-sm); padding: 4px 7px; font-size: 12px; color: var(--clr-text-2); background: var(--clr-surface-2); text-transform: none; letter-spacing: 0; font-weight: 400; }
.tm-keyin:focus { outline: none; border-color: var(--clr-accent); background: var(--clr-bg); }
.tm-keyin:disabled { opacity: 0.6; cursor: not-allowed; }

.tm-iconwrap { position: relative; flex-shrink: 0; }
.tm-iconbtn { display: inline-flex; align-items: center; gap: 4px; padding: 5px 8px; border: 1px solid var(--clr-border); border-radius: var(--r-sm); background: var(--clr-bg); color: var(--clr-text-3); }
.tm-iconbtn:hover { border-color: var(--clr-accent); }
.tm-iconpop { position: absolute; top: calc(100% + 5px); left: 0; z-index: 30; width: 248px; padding: 8px; background: var(--clr-surface); border: 1px solid var(--clr-border); border-radius: var(--r-md); box-shadow: var(--sh-lg); }
.tm-iconsearch { width: 100%; box-sizing: border-box; margin-bottom: 8px; }
.tm-icongrid { display: grid; grid-template-columns: repeat(7, 1fr); gap: 3px; max-height: 200px; overflow-y: auto; }
.tm-iconopt { display: inline-flex; align-items: center; justify-content: center; height: 30px; border-radius: var(--r-sm); background: transparent; }
.tm-iconopt:hover { background: var(--clr-surface-2); }
.tm-iconopt.on { background: rgba(0,113,227,0.14); box-shadow: inset 0 0 0 1px var(--clr-accent); }

.link { background: none; color: var(--clr-accent); font-size: 13px; font-weight: 600; padding: 4px 6px; align-self: flex-start; border-radius: var(--r-sm); }
.link:hover { text-decoration: underline; }
.link.danger { color: var(--clr-danger); }
.tm-need { border-color: var(--clr-danger); box-shadow: 0 0 0 2px rgba(255,59,48,0.16); }
.tm-ftype:disabled { cursor: not-allowed; opacity: 0.65; }
.tm-toggle:disabled { cursor: not-allowed; opacity: 0.5; }
</style>
