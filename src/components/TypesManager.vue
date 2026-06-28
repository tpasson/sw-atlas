<template>
  <div class="card-stack">
    <div class="card">
      <span class="section-label">Item types</span>
      <p class="card-hint">
        Define your own item types (e.g. <em>Bug</em>, <em>Release</em>, <em>Feature</em>) with their own fields.
        New items can be created as any type; a type's fields then appear in the item dialog.
        Built-in types can't be edited.
      </p>

      <div class="tm-builtins">
        <span v-for="t in builtins" :key="t.key" class="tm-chip">{{ t.label }} · {{ familyShort(t.family) }}</span>
      </div>

      <div v-for="(t, ti) in custom" :key="ti" class="tm-type">
        <div class="tm-head">
          <input class="ti" v-model="t.label" placeholder="Label (e.g. Bug)" style="flex:1;min-width:110px" />
          <input class="ti" v-model="t.key" placeholder="key" style="width:96px" />
          <select class="ti" v-model="t.family" style="width:96px">
            <option value="timeline-point">Point</option>
            <option value="timeline-range">Range</option>
          </select>
          <input class="ti" v-model="t.icon" placeholder="l:Bug" style="width:96px" />
          <input type="color" class="tm-color" :value="t.color || '#0A84FF'" @input="t.color = $event.target.value" title="Colour" />
          <button class="link danger" @click="custom.splice(ti, 1)">Remove</button>
        </div>

        <div class="tm-fields">
          <div v-for="(f, fi) in t.fields" :key="fi" class="tm-field">
            <input class="ti" v-model="f.label" placeholder="Field label" style="flex:1" />
            <input class="ti" v-model="f.key" placeholder="key" style="width:90px" />
            <select class="ti" v-model="f.type" style="width:96px">
              <option value="text">Text</option>
              <option value="number">Number</option>
              <option value="select">Select</option>
              <option value="date">Date</option>
            </select>
            <input
              v-if="f.type === 'select'"
              class="ti"
              :value="(f.options || []).join(', ')"
              placeholder="opt1, opt2, …"
              style="flex:1"
              @input="f.options = $event.target.value.split(',').map(s => s.trim()).filter(Boolean)"
            />
            <button class="link danger" @click="t.fields.splice(fi, 1)" title="Remove field">×</button>
          </div>
          <button class="link" @click="t.fields.push({ key: '', label: '', type: 'text', options: [] })">+ Field</button>
        </div>
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
import { ref, computed } from 'vue'
import { itemTypes, saveItemTypes } from '../stores/useAppStore.js'

const builtins = computed(() => itemTypes.list.filter(t => t.builtin))
const custom = ref(itemTypes.list.filter(t => !t.builtin).map(t => ({
  key: t.key, label: t.label, family: t.family, icon: t.icon, color: t.color,
  fields: (t.fields || []).map(f => ({ key: f.key, label: f.label, type: f.type, options: [...(f.options || [])] })),
})))

const msg = ref('')
const okMsg = ref(false)

function familyShort(f) {
  return f === 'timeline-range' ? 'range' : f === 'work-item' ? 'work' : f === 'container' ? 'folder' : 'point'
}
function addType() {
  custom.value.push({ key: '', label: '', family: 'timeline-point', icon: 'l:Diamond', color: '', fields: [] })
}

async function save() {
  const keys = new Set()
  for (const t of custom.value) {
    if (!/^[a-z0-9_-]+$/i.test(t.key || '')) { fail(`Each type needs a simple key (letters/digits/-/_). Got "${t.key}".`); return }
    if (keys.has(t.key)) { fail(`Duplicate type key "${t.key}".`); return }
    keys.add(t.key)
    for (const f of t.fields) {
      if (!/^[a-z0-9_-]+$/i.test(f.key || '')) { fail(`Field in "${t.label || t.key}" needs a simple key.`); return }
    }
  }
  try {
    await saveItemTypes(custom.value.map(t => ({
      key: t.key, label: t.label || t.key, family: t.family, icon: t.icon || 'l:Diamond', color: t.color || '',
      fields: t.fields.map(f => ({ key: f.key, label: f.label || f.key, type: f.type, options: f.type === 'select' ? (f.options || []) : [] })),
    })))
    okMsg.value = true
    msg.value = 'Saved — pick the type when adding an item.'
  } catch {
    fail('Save failed.')
  }
}
function fail(m) { okMsg.value = false; msg.value = m }
</script>

<style scoped>
.tm-builtins { display: flex; flex-wrap: wrap; gap: 6px; }
.tm-chip { font-size: 12px; color: var(--clr-text-3); background: var(--clr-surface-2); border-radius: 100px; padding: 3px 10px; }

.tm-type { border: 1px solid var(--clr-border-light); border-radius: var(--r-md); padding: 10px; display: flex; flex-direction: column; gap: 8px; }
.tm-head { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; }
.tm-fields { display: flex; flex-direction: column; gap: 6px; padding-left: 10px; border-left: 2px solid var(--clr-border-light); }
.tm-field { display: flex; gap: 6px; align-items: center; }

.ti {
  border: 1px solid var(--clr-border); border-radius: var(--r-sm);
  padding: 6px 9px; font-size: 13px; color: var(--clr-text); background: var(--clr-bg);
}
.ti:focus { outline: none; border-color: var(--clr-accent); }
.tm-color { width: 30px; height: 30px; border: none; background: none; padding: 0; cursor: pointer; flex-shrink: 0; }

.link { background: none; color: var(--clr-accent); font-size: 13px; font-weight: 600; padding: 4px 6px; align-self: flex-start; border-radius: var(--r-sm); }
.link:hover { text-decoration: underline; }
.link.danger { color: var(--clr-danger); }

.tm-save { background: var(--clr-accent); color: #fff; border-radius: var(--r-md); padding: 8px 16px; font-weight: 600; }
.tm-save:hover { background: var(--clr-accent-hover); }
</style>
