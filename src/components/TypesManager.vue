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

          <label v-if="!t.builtin" class="tm-key" title="Auto-filled from the name">key<input class="tm-keyin" v-model="t.key" @input="t._keyTouched = true" /></label>
          <span v-else class="tm-tag" title="Built-in — can't be deleted">built-in</span>
          <button v-if="!t.builtin" class="link danger" @click="types.splice(ti, 1)">Remove</button>
        </div>

        <div class="tm-fields">
          <div v-for="(f, fi) in t.fields" :key="fi" class="tm-field">
            <input class="ti tm-grow" v-model="f.label" placeholder="Field name (e.g. Severity)" @input="onFieldLabel(t, f)" />
            <select class="ti tm-ftype" v-model="f.type" title="Field type">
              <option value="text">Text</option>
              <option value="number">Number</option>
              <option value="select">Select (one)</option>
              <option value="multiselect">Multi-select</option>
              <option value="date">Date</option>
            </select>
            <input
              v-if="f.type === 'select' || f.type === 'multiselect'"
              class="ti tm-grow"
              :value="(f.options || []).join(', ')"
              placeholder="comma, separated, options"
              @change="f.options = $event.target.value.split(',').map(s => s.trim()).filter(Boolean)"
            />
            <button type="button" class="tm-toggle" :class="{ on: f.required }" title="Must be filled in" @click="f.required = !f.required">Required</button>
            <label class="tm-key" title="Auto-filled from the field name">key<input class="tm-keyin" v-model="f.key" @input="f._keyTouched = true" /></label>
            <button class="tm-x" @click="t.fields.splice(fi, 1)" title="Remove field">×</button>
          </div>
          <button class="link tm-addfield" @click="addField(t)">+ Field</button>
        </div>
        <StatusEditor :statuses="t.statuses" />
      </div>

      <button class="link" @click="addType">+ Add type</button>

      <div class="row-between">
        <span v-if="msg" class="data-msg" :class="{ ok: okMsg, err: !okMsg }">{{ msg }}</span>
        <span v-else></span>
        <button class="tm-save" @click="save">Save types</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { itemTypes, saveItemTypes, MARKER_LIBRARY } from '../stores/useAppStore.js'
import MarkerIcon from './MarkerIcon.vue'
import StatusEditor from './StatusEditor.vue'

let uid = 0
const nextUid = () => `t${uid++}`
// One unified list. Built-ins are editable (label/icon/colour/fill/fields) but
// keep their key + behaviour; only custom types are removable / re-keyed.
const types = ref(itemTypes.list.map(t => ({
  _uid: nextUid(), builtin: !!t.builtin, _keyTouched: true,
  key: t.key, label: t.label, family: t.family, icon: t.icon, color: t.color || '', fill: t.fill !== false,
  fields: (t.fields || []).map(f => ({ key: f.key, label: f.label, type: f.type, options: [...(f.options || [])], required: !!f.required, _keyTouched: true })),
  statuses: (t.statuses || []).map(s => ({ key: s.key, label: s.label, tone: s.tone || 'neutral', to: [...(s.to || [])], _keyTouched: true })),
})))

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
  types.value.push({ _uid: nextUid(), builtin: false, _keyTouched: false, key: '', label: '', family: 'timeline-point', icon: 'l:Diamond', color: '', fill: true, fields: [], statuses: [] })
}
function addField(t) {
  t.fields.push({ key: '', label: '', type: 'text', options: [], required: false, _keyTouched: false })
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
    out.push({ key: fk, label: f.label || fk, type: f.type, required: !!f.required, options: (f.type === 'select' || f.type === 'multiselect') ? (f.options || []) : [] })
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
async function save() {
  const typeKeys = new Set(types.value.filter(t => t.builtin).map(t => t.key))
  const payload = []
  for (const t of types.value) {
    if (t.builtin) {
      payload.push({ key: t.key, label: t.label || t.key, family: t.family, icon: t.icon || 'l:Diamond', color: t.color || '', fill: t.fill, fields: cleanFields(t.fields), statuses: cleanStatuses(t.statuses) })
      continue
    }
    if (!t.label && !t.key && !t.fields.length) continue
    const key = uniqueKey(t.key || slugify(t.label), typeKeys)
    typeKeys.add(key)
    payload.push({ key, label: t.label || key, family: t.family, icon: t.icon || 'l:Diamond', color: t.color || '', fill: t.fill, fields: cleanFields(t.fields), statuses: cleanStatuses(t.statuses) })
  }
  try {
    await saveItemTypes(payload)
    okMsg.value = true
    msg.value = 'Saved — icons, colours and the legend update everywhere.'
  } catch {
    fail('Save failed.')
  }
}
function fail(m) { okMsg.value = false; msg.value = m }
</script>

<style scoped>
.tm-type { border: 1px solid var(--clr-border-light); border-radius: var(--r-md); padding: 10px; display: flex; flex-direction: column; gap: 8px; }
.tm-builtin { background: var(--clr-surface-2); }
.tm-head { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; }
.tm-fam { font-size: 11px; color: var(--clr-text-3); background: var(--clr-bg); border-radius: 100px; padding: 4px 10px; white-space: nowrap; }
.tm-tag { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.4px; color: var(--clr-text-3); }
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

.tm-save { background: var(--clr-accent); color: #fff; border-radius: var(--r-md); padding: 8px 16px; font-weight: 600; }
.tm-save:hover { background: var(--clr-accent-hover); }
</style>
