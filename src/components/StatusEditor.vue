<template>
  <div class="se">
    <template v-if="statuses.length">
    <div class="se-title">
      Statuses <span class="se-hint">workflow — tone sets the meaning, the colour swatch overrides it · the top one is the start</span>
    </div>

    <div v-for="(s, i) in statuses" :key="s._uid" class="se-row">
      <button type="button" class="se-start" :class="{ on: i === 0 }" title="Start status for new items" @click="makeStart(i)">start</button>
      <select class="ti se-tone" v-model="s.tone" :style="{ boxShadow: 'inset 4px 0 0 ' + toneColor(s.tone) }" title="Tone — the meaning (drives logic like ‘ready’)">
        <option v-for="t in STATUS_TONES" :key="t.key" :value="t.key">{{ t.label }}</option>
      </select>
      <span class="se-colorwrap">
        <input type="color" class="se-color" :value="statusColor(s)" @input="s.color = $event.target.value" title="Colour — overrides the tone colour" />
        <button v-if="s.color" type="button" class="se-color-x" title="Reset to the tone colour" @click="s.color = ''">↺</button>
      </span>
      <input class="ti se-label" v-model="s.label" placeholder="Status name (e.g. Approved)" @input="onLabel(s)" />
      <div class="se-to">
        <span class="se-arrow" title="Can transition to">→</span>
        <button
          v-for="o in others(i)" :key="o._uid" type="button"
          class="se-chip" :class="{ on: (s.to || []).includes(o.key) }"
          :title="'Allow moving to ' + (o.label || o.key)"
          @click="toggleTo(s, o.key)"
        >{{ o.label || o.key }}</button>
        <span v-if="!others(i).length" class="se-empty">add more statuses to set transitions</span>
      </div>
      <button v-if="statuses.length > 1" class="se-x" @click="remove(i)" title="Remove status">×</button>
    </div>

    <button class="link se-add" @click="add">+ Status</button>
    </template>

    <button v-else type="button" class="se-enable" @click="enable">
      + Enable status workflow <span class="se-enable-hint">seeds the standard To&nbsp;Do&nbsp;→&nbsp;Done set</span>
    </button>
  </div>
</template>

<script setup>
import { STATUS_TONES, toneColor, statusColor, DEFAULT_STATUSES } from '../stores/useAppStore.js'

const props = defineProps({ statuses: { type: Array, required: true } })

let uidN = 0
const uid = () => 'st' + (++uidN) + Math.random().toString(36).slice(2, 6)
const slugify = (s) => (s || '').toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '')

// Give existing statuses a render key + remember which had a hand-edited key.
for (const s of props.statuses) { if (!s._uid) s._uid = uid(); if (s._keyTouched == null) s._keyTouched = true }

function uniqueKey(base, self) {
  const taken = new Set(props.statuses.filter(x => x !== self).map(x => x.key))
  let k = base || 'status', n = 2
  while (taken.has(k)) k = `${base}-${n++}`
  return k
}
function onLabel(s) { if (!s._keyTouched) s.key = uniqueKey(slugify(s.label), s) }
function others(i) { return props.statuses.filter((_, j) => j !== i) }
function toggleTo(s, key) {
  if (!Array.isArray(s.to)) s.to = []
  const idx = s.to.indexOf(key)
  if (idx === -1) s.to.push(key); else s.to.splice(idx, 1)
}
function makeStart(i) {
  if (i === 0) return
  const [s] = props.statuses.splice(i, 1)
  props.statuses.unshift(s)
}
function add() {
  props.statuses.push({ _uid: uid(), key: uniqueKey('status'), label: '', tone: 'neutral', color: '', to: [], _keyTouched: false })
}
function enable() {
  for (const s of DEFAULT_STATUSES) props.statuses.push({ key: s.key, label: s.label, tone: s.tone, color: s.color || '', to: [...s.to], _uid: uid(), _keyTouched: true })
}
function remove(i) {
  if (props.statuses.length <= 1) return // a workflow must keep at least one status
  const removed = props.statuses[i]
  props.statuses.splice(i, 1)
  for (const s of props.statuses) if (Array.isArray(s.to)) s.to = s.to.filter(k => k !== removed.key)
}
</script>

<style scoped>
.se { display: flex; flex-direction: column; gap: 6px; padding-left: 10px; border-left: 2px solid var(--clr-border-light); }
.se-title { font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.4px; color: var(--clr-text-3); display: flex; align-items: baseline; gap: 8px; }
.se-hint { font-size: 9px; font-weight: 600; text-transform: none; letter-spacing: 0.2px; opacity: 0.75; }
.se-row { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.ti { border: 1px solid var(--clr-border); border-radius: var(--r-sm); padding: 6px 9px; font-size: 13px; color: var(--clr-text); background: var(--clr-bg); }
.ti:focus { outline: none; border-color: var(--clr-accent); }
.se-tone { width: 124px; flex-shrink: 0; }
.se-colorwrap { display: inline-flex; align-items: center; gap: 3px; flex-shrink: 0; }
.se-color { width: 30px; height: 30px; padding: 0; border: 1px solid var(--clr-border); border-radius: var(--r-sm); background: none; cursor: pointer; }
.se-color-x { width: 22px; height: 26px; font-size: 13px; color: var(--clr-text-3); background: none; border-radius: var(--r-sm); }
.se-color-x:hover { color: var(--clr-text); background: var(--clr-surface-2); }
.se-label { flex: 1; min-width: 130px; }
.se-start { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.4px; color: var(--clr-text-3);
  border: 1px solid var(--clr-border); border-radius: 100px; padding: 4px 9px; background: var(--clr-bg); flex-shrink: 0; }
.se-start.on { color: var(--clr-accent); border-color: var(--clr-accent); background: rgba(0,113,227,0.08); }
.se-to { display: flex; align-items: center; flex-wrap: wrap; gap: 5px; }
.se-arrow { color: var(--clr-text-3); font-size: 13px; }
.se-chip { font-size: 11px; font-weight: 600; color: var(--clr-text-2); border: 1px solid var(--clr-border); border-radius: 100px; padding: 3px 9px; background: var(--clr-bg); }
.se-chip.on { color: var(--clr-accent); border-color: var(--clr-accent); background: rgba(0,113,227,0.08); }
.se-empty { font-size: 11px; color: var(--clr-text-3); }
.se-x { width: 26px; height: 26px; flex-shrink: 0; display: inline-flex; align-items: center; justify-content: center; font-size: 17px; line-height: 1; color: var(--clr-text-3); background: none; border-radius: var(--r-sm); }
.se-x:hover { color: var(--clr-danger); background: rgba(255,59,48,0.08); }
.se-add { align-self: flex-start; }
.se-off { margin-left: auto; font-size: 11px; font-weight: 600; color: var(--clr-danger); background: none; text-transform: none; letter-spacing: 0; }
.se-off:hover { text-decoration: underline; }
.se-enable { align-self: flex-start; display: inline-flex; align-items: baseline; gap: 8px; color: var(--clr-accent); font-size: 13px; font-weight: 600; background: none; padding: 4px 6px; border-radius: var(--r-sm); }
.se-enable:hover { text-decoration: underline; }
.se-enable-hint { font-size: 11px; font-weight: 400; color: var(--clr-text-3); }
.link { background: none; color: var(--clr-accent); font-size: 13px; font-weight: 600; padding: 4px 6px; border-radius: var(--r-sm); }
.link:hover { text-decoration: underline; }
</style>
