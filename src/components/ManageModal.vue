<template>
  <Teleport to="body">
    <Transition name="modal">
      <div class="backdrop" @click.self="$emit('close')">
        <Transition name="modal-panel" appear>
          <div class="panel">
            <!-- Header -->
            <div class="panel-header">
              <h2 class="panel-title">Settings</h2>
              <button class="btn-close" @click="$emit('close')">
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                  <path d="M1 1l12 12M13 1L1 13" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/>
                </svg>
              </button>
            </div>

            <!-- Tabs -->
            <div class="tabs" role="tablist">
              <button class="tab" :class="{ active: tab === 'areas' }" @click="tab = 'areas'">Areas</button>
              <button class="tab" :class="{ active: tab === 'display' }" @click="tab = 'display'">Display</button>
              <button class="tab" :class="{ active: tab === 'legend' }" @click="tab = 'legend'">Icons</button>
              <button class="tab" :class="{ active: tab === 'baselines' }" @click="tab = 'baselines'">Baselines</button>
            </div>

            <div class="panel-body">
              <!-- ───────────────── AREAS ───────────────── -->
              <section v-show="tab === 'areas'" class="tab-pane">
                <div class="card">
                  <p class="section-label">Add new area</p>
                  <div class="add-lane-form">
                    <input
                      v-model="newLane.name"
                      class="field-input grow"
                      placeholder="Area name"
                      @keyup.enter="doAddSwimlane"
                    />
                    <div class="color-row">
                      <button
                        v-for="c in swatchColors"
                        :key="c"
                        class="color-swatch"
                        :class="{ selected: newLane.color === c }"
                        :style="{ background: c }"
                        @click="newLane.color = c"
                      >
                        <svg v-if="newLane.color === c" width="10" height="10" viewBox="0 0 10 10" fill="none">
                          <path d="M1.5 5l2.5 2.5 4.5-5" stroke="white" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                        </svg>
                      </button>
                      <input type="color" class="color-custom" v-model="newLane.color" title="Custom colour" />
                    </div>
                    <button class="btn-add" :disabled="!newLane.name.trim()" @click="doAddSwimlane">
                      + Add area
                    </button>
                  </div>
                </div>

                <div class="card">
                  <div class="row-between">
                    <p class="section-label" style="margin: 0">Shared palette</p>
                    <button class="link-btn" @click="resetPalette">Reset to defaults</button>
                  </div>
                  <p class="card-hint">These colours appear in every area colour picker — for all editors. Remove any (incl. defaults) or add your own.</p>
                  <div class="palette-row">
                    <span v-for="c in palette.colors" :key="c" class="pal-chip" :style="{ background: c }" :title="c">
                      <button class="pal-x" @click="removePaletteColor(c)" title="Remove colour">×</button>
                    </span>
                    <span v-if="palette.colors.length === 0" class="pal-empty">Palette is empty — add a colour or reset.</span>
                  </div>
                  <div class="palette-add">
                    <input type="color" class="color-custom" v-model="customNew" />
                    <button class="btn-add" @click="addPaletteColor(customNew)">+ Add to palette</button>
                  </div>
                </div>

                <p class="section-label">Existing areas</p>
                <div v-if="store.swimlanes.length === 0" class="empty">
                  No areas defined yet — add one above.
                </div>

                <div v-else class="lanes-grid">
                  <div v-for="(sw, si) in store.swimlanes" :key="sw.id" class="lane-item">
                    <!-- Row 1: name + actions -->
                    <div class="lane-header">
                      <span class="lane-dot" :style="{ background: sw.color }"></span>

                      <input
                        v-if="editing.id === sw.id && editing.type === 'lane'"
                        :ref="el => { if (el) el.focus() }"
                        class="inline-input"
                        v-model="editing.name"
                        @blur="saveEdit(sw)"
                        @keyup.enter="saveEdit(sw)"
                        @keyup.escape="cancelEdit"
                      />
                      <span v-else class="lane-name" @dblclick="startEdit(sw)">{{ sw.name }}</span>

                      <div class="lane-actions">
                        <button class="icon-btn" title="Move up" @click="moveSwimlane(sw.id, -1)" :disabled="si === 0">
                          <svg width="13" height="13" viewBox="0 0 13 13" fill="none">
                            <path d="M6.5 10V3M3 6.5l3.5-3.5 3.5 3.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                          </svg>
                        </button>
                        <button class="icon-btn" title="Move down" @click="moveSwimlane(sw.id, 1)" :disabled="si === store.swimlanes.length - 1">
                          <svg width="13" height="13" viewBox="0 0 13 13" fill="none">
                            <path d="M6.5 3v7M10 6.5L6.5 10 3 6.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                          </svg>
                        </button>
                        <button class="icon-btn danger" title="Delete area" @click="doDeleteSwimlane(sw.id)">
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6M10 11v6M14 11v6"/>
                          </svg>
                        </button>
                      </div>
                    </div>

                    <!-- Row 2: color swatches -->
                    <div class="lane-colors">
                      <button
                        v-for="c in swatchColors"
                        :key="c"
                        class="color-swatch sm"
                        :class="{ selected: sw.color === c }"
                        :style="{ background: c }"
                        @click="updateSwimlane(sw.id, { color: c })"
                      >
                        <svg v-if="sw.color === c" width="7" height="7" viewBox="0 0 10 10" fill="none">
                          <path d="M1.5 5l2.5 2.5 4.5-5" stroke="white" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
                        </svg>
                      </button>
                      <input type="color" class="color-custom sm" :value="sw.color" @change="updateSwimlane(sw.id, { color: $event.target.value })" title="Custom colour" />
                    </div>

                    <!-- Row 3: sub-areas -->
                    <div class="sublanes">
                      <div v-for="sub in sw.subLanes" :key="sub.id" class="sublane-item">
                        <span class="sublane-dot" :style="{ background: sw.color }"></span>
                        <input
                          v-if="editing.id === sub.id && editing.type === 'sub'"
                          :ref="el => { if (el) el.focus() }"
                          class="inline-input sm"
                          v-model="editing.name"
                          @blur="saveSubEdit(sw.id, sub)"
                          @keyup.enter="saveSubEdit(sw.id, sub)"
                          @keyup.escape="cancelEdit"
                        />
                        <span v-else class="sublane-name" @dblclick="startSubEdit(sub)">{{ sub.name }}</span>
                        <button class="icon-btn danger" @click="doDeleteSubLane(sw.id, sub.id, sub.name)">
                          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6M10 11v6M14 11v6"/>
                          </svg>
                        </button>
                      </div>

                      <div class="add-sub-row">
                        <input
                          v-model="newSubs[sw.id]"
                          class="field-input sm"
                          :placeholder="`Add sub-area…`"
                          @keyup.enter="doAddSubLane(sw.id)"
                        />
                        <button
                          class="btn-add-sub"
                          :disabled="!newSubs[sw.id]?.trim()"
                          @click="doAddSubLane(sw.id)"
                        >+</button>
                      </div>
                    </div>
                  </div>
                </div>
              </section>

              <!-- ───────────────── DISPLAY ───────────────── -->
              <section v-show="tab === 'display'" class="tab-pane">
                <div class="card">
                  <div class="row-between">
                    <div class="setting-info">
                      <span class="setting-name">Public read access</span>
                      <span class="setting-desc">When off, only logged-in editors can view ATLAS</span>
                    </div>
                    <button class="toggle" :class="{ active: session.publicReadEnabled }" @click="togglePublicRead">
                      <span class="toggle-knob"></span>
                    </button>
                  </div>
                </div>

                <div class="card">
                  <p class="section-label">Today indicator</p>
                  <div class="row-between">
                    <div class="setting-info">
                      <span class="setting-name">Month highlight</span>
                      <span class="setting-desc">Tint the column of the current month</span>
                    </div>
                    <button class="toggle" :class="{ active: settings.monthHighlight.enabled }" @click="settings.monthHighlight.enabled = !settings.monthHighlight.enabled">
                      <span class="toggle-knob"></span>
                    </button>
                  </div>
                  <div v-if="settings.monthHighlight.enabled" class="opt-row">
                    <label class="opt">Color <input type="color" v-model="settings.monthHighlight.color" /></label>
                    <label class="opt">Opacity
                      <input type="range" min="0" max="0.3" step="0.01" v-model.number="settings.monthHighlight.opacity" />
                      <span class="opt-val">{{ Math.round(settings.monthHighlight.opacity * 100) }}%</span>
                    </label>
                  </div>

                  <div class="row-between">
                    <div class="setting-info">
                      <span class="setting-name">Today line</span>
                      <span class="setting-desc">Vertical line at today's exact date</span>
                    </div>
                    <button class="toggle" :class="{ active: settings.dayLine.enabled }" @click="settings.dayLine.enabled = !settings.dayLine.enabled">
                      <span class="toggle-knob"></span>
                    </button>
                  </div>
                  <div v-if="settings.dayLine.enabled" class="opt-row">
                    <label class="opt">Color <input type="color" v-model="settings.dayLine.color" /></label>
                    <label class="opt">Opacity
                      <input type="range" min="0" max="1" step="0.05" v-model.number="settings.dayLine.opacity" />
                      <span class="opt-val">{{ Math.round(settings.dayLine.opacity * 100) }}%</span>
                    </label>
                    <label class="opt">Width
                      <input type="range" min="0.5" max="6" step="0.5" v-model.number="settings.dayLine.width" />
                      <span class="opt-val">{{ settings.dayLine.width }}px</span>
                    </label>
                  </div>
                </div>

                <div class="card">
                  <div class="row-between">
                    <div class="setting-info">
                      <span class="setting-name">Calendar weeks (CW)</span>
                      <span class="setting-desc">Show ISO week numbers under the months</span>
                    </div>
                    <button class="toggle" :class="{ active: settings.weekNumbers.enabled }" @click="settings.weekNumbers.enabled = !settings.weekNumbers.enabled">
                      <span class="toggle-knob"></span>
                    </button>
                  </div>
                </div>

                <div class="card">
                  <p class="section-label">Gridlines</p>

                  <div class="row-between">
                    <div class="setting-info">
                      <span class="setting-name">Month lines</span>
                      <span class="setting-desc">Vertical separators between months</span>
                    </div>
                    <button class="toggle" :class="{ active: settings.monthLines.enabled }" @click="settings.monthLines.enabled = !settings.monthLines.enabled">
                      <span class="toggle-knob"></span>
                    </button>
                  </div>
                  <div v-if="settings.monthLines.enabled" class="opt-row">
                    <label class="opt">Color <input type="color" v-model="settings.monthLines.color" /></label>
                    <label class="opt">Opacity
                      <input type="range" min="0" max="1" step="0.02" v-model.number="settings.monthLines.opacity" />
                      <span class="opt-val">{{ Math.round(settings.monthLines.opacity * 100) }}%</span>
                    </label>
                    <label class="opt">Width
                      <input type="range" min="0.5" max="6" step="0.5" v-model.number="settings.monthLines.width" />
                      <span class="opt-val">{{ settings.monthLines.width }}px</span>
                    </label>
                  </div>

                  <div class="row-between">
                    <div class="setting-info">
                      <span class="setting-name">Week lines</span>
                      <span class="setting-desc">Fine vertical lines at each calendar week</span>
                    </div>
                    <button class="toggle" :class="{ active: settings.weekLines.enabled }" @click="settings.weekLines.enabled = !settings.weekLines.enabled">
                      <span class="toggle-knob"></span>
                    </button>
                  </div>
                  <div v-if="settings.weekLines.enabled" class="opt-row">
                    <label class="opt">Color <input type="color" v-model="settings.weekLines.color" /></label>
                    <label class="opt">Opacity
                      <input type="range" min="0" max="1" step="0.02" v-model.number="settings.weekLines.opacity" />
                      <span class="opt-val">{{ Math.round(settings.weekLines.opacity * 100) }}%</span>
                    </label>
                    <label class="opt">Width
                      <input type="range" min="0.5" max="6" step="0.5" v-model.number="settings.weekLines.width" />
                      <span class="opt-val">{{ settings.weekLines.width }}px</span>
                    </label>
                  </div>
                </div>

                <div class="card">
                  <p class="section-label">Items (milestones &amp; events)</p>
                  <p class="card-hint">Appearance of markers and event bars, incl. the spacing of the hover outline.</p>
                  <div class="row-between">
                    <span class="setting-name">Item border</span>
                    <div class="seg-mini">
                      <button type="button" :class="{ on: settings.items.borderMode === 'always' }" @click="settings.items.borderMode = 'always'">Always</button>
                      <button type="button" :class="{ on: settings.items.borderMode === 'hover' }" @click="settings.items.borderMode = 'hover'">On hover</button>
                      <button type="button" :class="{ on: settings.items.borderMode === 'off' }" @click="settings.items.borderMode = 'off'">Off</button>
                    </div>
                  </div>
                  <div class="opt-row">
                    <label class="opt">Font size
                      <input type="range" min="9" max="18" step="0.5" v-model.number="settings.items.fontSize" />
                      <span class="opt-val">{{ settings.items.fontSize }}px</span>
                    </label>
                    <label class="opt">Font weight
                      <input type="range" min="300" max="700" step="100" v-model.number="settings.items.fontWeight" />
                      <span class="opt-val">{{ ({ 300: 'Light', 400: 'Regular', 500: 'Medium', 600: 'Semibold', 700: 'Bold' })[settings.items.fontWeight] || settings.items.fontWeight }}</span>
                    </label>
                    <label class="opt">Padding
                      <input type="range" min="0" max="12" step="1" v-model.number="settings.items.padding" />
                      <span class="opt-val">{{ settings.items.padding }}px</span>
                    </label>
                  </div>
                  <div class="opt-row">
                    <label class="opt">Corner radius
                      <input type="range" min="0" max="20" step="1" v-model.number="settings.items.radius" />
                      <span class="opt-val">{{ settings.items.radius }}px</span>
                    </label>
                    <label class="opt">Border width
                      <input type="range" min="0" max="5" step="0.5" v-model.number="settings.items.border" />
                      <span class="opt-val">{{ settings.items.border }}px</span>
                    </label>
                    <label class="opt">Icon gap
                      <input type="range" min="0" max="12" step="1" v-model.number="settings.items.iconGap" />
                      <span class="opt-val">{{ settings.items.iconGap }}px</span>
                    </label>
                  </div>
                  <div class="opt-row">
                    <label class="opt">Label offset
                      <input type="range" min="-4" max="4" step="1" v-model.number="settings.items.labelOffset" />
                      <span class="opt-val">{{ settings.items.labelOffset > 0 ? '+' : '' }}{{ settings.items.labelOffset }}px</span>
                    </label>
                    <span class="opt-note">−  higher · +  lower (per-browser text alignment)</span>
                  </div>
                  <div class="opt-row">
                    <label class="opt">Label fit buffer
                      <input type="range" min="-20" max="40" step="2" v-model.number="settings.items.labelBuffer" />
                      <span class="opt-val">{{ settings.items.labelBuffer > 0 ? '+' : '' }}{{ settings.items.labelBuffer }}px</span>
                    </label>
                    <span class="opt-note">event title inside vs. right of the bar (− = fits tighter)</span>
                  </div>
                  <div class="opt-row">
                    <label class="opt">Event fill
                      <input type="range" min="0" max="1" step="0.05" v-model.number="settings.items.eventOpacity" />
                      <span class="opt-val">{{ Math.round(settings.items.eventOpacity * 100) }}%</span>
                    </label>
                  </div>
                </div>
              </section>

              <!-- ───────────────── LEGEND ───────────────── -->
              <section v-show="tab === 'legend'" class="tab-pane">
                <div class="card">
                  <p class="section-label">Icons &amp; markers</p>
                  <p class="card-hint">Up to 8 marker types — shown in the legend and offered when picking an item's marker. Plus the event-bar label.</p>
                  <div class="opt-row">
                    <label class="opt">Marker size
                      <input type="range" min="10" max="22" step="1" v-model.number="settings.items.markerSize" />
                      <span class="opt-val">{{ settings.items.markerSize }}px</span>
                    </label>
                    <label class="opt">Line thickness
                      <input type="range" min="1" max="3" step="0.25" v-model.number="settings.items.markerStroke" />
                      <span class="opt-val">{{ settings.items.markerStroke }}</span>
                    </label>
                  </div>
                  <div class="legend-edit">
                    <div v-for="(m, i) in settings.markers" :key="i" class="leg-row">
                      <span class="leg-ico"><MarkerIcon :shape="m.shape" :fill="m.fill" color="#8a8a8e" :size="14" /></span>
                      <input class="field-input sm" v-model="m.label" />
                      <button class="fill-toggle" :class="{ on: m.fill }" title="Toggle filled icon" @click="m.fill = !m.fill">Fill</button>
                      <button class="icon-btn danger" :disabled="settings.markers.length <= 1" title="Remove marker" @click="removeMarker(i)">
                        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                          <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6M10 11v6M14 11v6"/>
                        </svg>
                      </button>
                    </div>
                    <label class="leg-row">
                      <span class="leg-ico"><span class="leg-bar"></span></span>
                      <input class="field-input sm" v-model="settings.eventLabel" />
                    </label>
                  </div>

                  <div v-if="settings.markers.length < 8" class="marker-add">
                    <input v-model="iconSearch" class="field-input sm" placeholder="Search markers… (save, server, cloud, bug, database …)" />
                    <div class="marker-grid">
                      <button v-for="s in availableShapes" :key="s" type="button" class="marker-add-btn" :title="prettyShape(s)" @click="addMarker(s)">
                        <MarkerIcon :shape="s" color="#8a8a8e" :size="16" />
                      </button>
                      <span v-if="!availableShapes.length" class="card-hint">No markers match “{{ iconSearch }}”.</span>
                    </div>
                  </div>
                  <p v-else class="card-hint">Maximum of 8 markers reached.</p>
                </div>
              </section>

              <!-- ───────────────── BASELINES ───────────────── -->
              <section v-show="tab === 'baselines'" class="tab-pane">
                <div class="card">
                  <p class="section-label">Baselines</p>
                  <p class="card-hint">Saved snapshots of the plan. Select one in the header to compare against Live.</p>
                  <div v-if="baselines.list.length === 0" class="empty">No baselines yet — save one from the header.</div>
                  <div v-else class="bl-list">
                    <div v-for="b in baselines.list" :key="b.id" class="bl-row">
                      <div class="bl-meta">
                        <span class="bl-name">{{ b.name }}</span>
                        <span class="bl-sub">{{ b.itemCount }} items · {{ formatDate(b.createdAt) }}</span>
                      </div>
                      <button class="icon-btn danger" @click="onDeleteBaseline(b)" title="Delete baseline">
                        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                          <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6M10 11v6M14 11v6"/>
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              </section>
            </div>

            <div class="panel-footer">
              <button class="reset-btn" @click="onResetSettings">Reset to defaults</button>
              <button class="btn-primary" @click="$emit('close')">Done</button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { reactive, ref, computed } from 'vue'
import { useAppStore, PRESET_COLORS, swatchColors, palette, baselines, store, session, settings, resetSettings, MARKER_LIBRARY } from '../stores/useAppStore.js'
import MarkerIcon from './MarkerIcon.vue'

const SHAPE_NAMES = { diamond: 'Diamond', circle: 'Circle', cone: 'Cone', flag: 'Flag', square: 'Square', triangleDown: 'Triangle', star: 'Star', hexagon: 'Hexagon', pentagon: 'Pentagon' }
function prettyShape(s) {
  if (!s.startsWith('l:')) return SHAPE_NAMES[s] || s
  return s.slice(2).replace(/([a-z0-9])([A-Z])/g, '$1 $2')
}
const iconSearch = ref('')
const availableShapes = computed(() => {
  const used = new Set(settings.markers.map(m => m.shape))
  const q = iconSearch.value.trim().toLowerCase()
  const out = []
  for (const s of MARKER_LIBRARY) {
    if (used.has(s)) continue
    if (q && !prettyShape(s).toLowerCase().includes(q) && !s.toLowerCase().includes(q)) continue
    out.push(s)
    if (out.length >= 120) break   // cap rendered icons; refine via search
  }
  return out
})
function addMarker(shape) {
  if (settings.markers.length >= 8) return
  settings.markers.push({ shape, label: prettyShape(shape), fill: true })
}
function removeMarker(i) {
  if (settings.markers.length > 1) settings.markers.splice(i, 1)
}

defineEmits(['close'])

const { addSwimlane, updateSwimlane, deleteSwimlane, moveSwimlane, addSubLane, updateSubLane, deleteSubLane, setPublicRead, addPaletteColor, removePaletteColor, resetPalette, deleteBaseline } = useAppStore()

const tab = ref('areas')
const customNew = ref('#0A84FF')

function formatDate(s) {
  if (!s) return ''
  return new Date(s).toLocaleDateString('en-US', { day: 'numeric', month: 'short', year: 'numeric' })
}
async function onDeleteBaseline(b) {
  if (confirm(`Delete baseline "${b.name}"?`)) await deleteBaseline(b.id)
}

function onResetSettings() {
  if (confirm('Reset all view settings (today indicator, gridlines, calendar weeks, item style, legend labels) to defaults?')) {
    resetSettings()
  }
}

async function togglePublicRead() {
  try {
    await setPublicRead(!session.publicReadEnabled)
  } catch (e) {
    alert('Could not change setting: ' + (e.message || 'error'))
  }
}

const newLane = reactive({ name: '', color: PRESET_COLORS[0] })
const newSubs = reactive({})
const editing = reactive({ id: null, type: null, name: '' })

function doAddSwimlane() {
  if (!newLane.name.trim()) return
  addSwimlane(newLane.name.trim(), newLane.color)
  newLane.name = ''
}

function doAddSubLane(swimlaneId) {
  const name = newSubs[swimlaneId]?.trim()
  if (!name) return
  addSubLane(swimlaneId, name)
  newSubs[swimlaneId] = ''
}

function doDeleteSwimlane(id) {
  if (confirm('Delete area and all its milestones?')) deleteSwimlane(id)
}

function doDeleteSubLane(swimlaneId, subId, name) {
  if (confirm(`Delete sub-area "${name}" and all its milestones?`)) deleteSubLane(swimlaneId, subId)
}

function startEdit(sw) {
  editing.id = sw.id; editing.type = 'lane'; editing.name = sw.name
}
function saveEdit(sw) {
  if (editing.name.trim()) updateSwimlane(sw.id, { name: editing.name.trim() })
  cancelEdit()
}
function startSubEdit(sub) {
  editing.id = sub.id; editing.type = 'sub'; editing.name = sub.name
}
function saveSubEdit(swimlaneId, sub) {
  if (editing.name.trim()) updateSubLane(swimlaneId, sub.id, editing.name.trim())
  cancelEdit()
}
function cancelEdit() {
  editing.id = null; editing.type = null; editing.name = ''
}
</script>

<style scoped>
.backdrop {
  position: fixed;
  inset: 0;
  /* Light, un-blurred scrim so timeline changes (gridlines, today marker, …)
     stay visible behind the Settings panel while you adjust them. */
  background: rgba(0,0,0,0.12);
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.panel {
  background: var(--clr-surface);
  border-radius: var(--r-xl);
  width: 100%;
  max-width: 960px;
  height: min(88vh, 760px);
  box-shadow: var(--sh-modal);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px 20px 14px;
  flex-shrink: 0;
}

.panel-title {
  font-size: 18px; font-weight: 700; letter-spacing: -0.3px; color: var(--clr-text);
}

.btn-close {
  width: 30px; height: 30px;
  display: flex; align-items: center; justify-content: center;
  background: var(--clr-surface-2);
  border-radius: 50%;
  color: var(--clr-text-2);
  transition: background 0.15s;
}
.btn-close:hover { background: var(--clr-border-light); }

/* ── Tabs ────────────────────────────────────────────────────────────── */
.tabs {
  display: flex;
  gap: 2px;
  padding: 0 16px;
  border-bottom: 1px solid var(--clr-border-light);
  flex-shrink: 0;
}
.tab {
  padding: 11px 16px;
  font-size: 13px; font-weight: 600;
  color: var(--clr-text-2);
  background: none;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  transition: color 0.15s, border-color 0.15s;
}
.tab:hover { color: var(--clr-text); }
.tab.active { color: var(--clr-accent); border-bottom-color: var(--clr-accent); }

.link-btn { background: none; font-size: 12px; font-weight: 600; color: var(--clr-accent); padding: 2px 4px; }
.link-btn:hover { text-decoration: underline; }

/* ── Body / panes ────────────────────────────────────────────────────── */
.panel-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 18px 20px;
}
.tab-pane { display: flex; flex-direction: column; gap: 14px; }

.card {
  background: var(--clr-bg);
  border: 1px solid var(--clr-border-light);
  border-radius: var(--r-lg);
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.card-hint { font-size: 12px; color: var(--clr-text-3); margin-top: -6px; }

.section-label {
  font-size: 11px; font-weight: 700;
  text-transform: uppercase; letter-spacing: 0.5px;
  color: var(--clr-text-3);
}

.row-between {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.setting-info { display: flex; flex-direction: column; gap: 2px; }
.setting-name { font-size: 14px; font-weight: 600; color: var(--clr-text); }
.setting-desc { font-size: 12px; color: var(--clr-text-3); }

/* ── Toggle ──────────────────────────────────────────────────────────── */
.toggle {
  width: 42px; height: 26px;
  border-radius: 13px;
  background: var(--clr-border);
  position: relative;
  transition: background 0.22s;
  flex-shrink: 0;
}
.toggle.active { background: var(--clr-accent); }
.toggle-knob {
  position: absolute;
  width: 20px; height: 20px;
  border-radius: 50%;
  background: white;
  top: 3px; left: 3px;
  transition: transform 0.22s;
  box-shadow: 0 1px 4px rgba(0,0,0,0.22);
}
.toggle.active .toggle-knob { transform: translateX(16px); }

/* ── Today-indicator option rows ─────────────────────────────────────── */
.opt-row { display: flex; flex-wrap: wrap; gap: 16px; align-items: center; }
.opt { display: inline-flex; align-items: center; gap: 6px; font-size: 12px; color: var(--clr-text-2); }
.opt input[type="color"] { width: 30px; height: 22px; padding: 0; border: 1px solid var(--clr-border); border-radius: 6px; background: none; cursor: pointer; }
.opt input[type="range"] { width: 92px; }
.opt-val { font-size: 11px; color: var(--clr-text-3); min-width: 34px; }
.opt-note { font-size: 11px; color: var(--clr-text-3); }

/* mini segmented control (e.g. item border mode) */
.seg-mini { display: inline-flex; border: 1px solid var(--clr-border); border-radius: var(--r-md); overflow: hidden; }
.seg-mini button { padding: 5px 11px; font-size: 12px; font-weight: 500; color: var(--clr-text-2); background: var(--clr-bg); transition: background 0.12s, color 0.12s; }
.seg-mini button + button { border-left: 1px solid var(--clr-border); }
.seg-mini button:hover:not(.on) { background: var(--clr-surface-2); }
.seg-mini button.on { background: var(--clr-accent); color: #fff; }

/* ── Add area form ───────────────────────────────────────────────────── */
.add-lane-form { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; }

.color-row { display: flex; gap: 6px; flex-wrap: wrap; }

.color-swatch {
  width: 24px; height: 24px;
  border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer;
  transition: transform 0.12s, box-shadow 0.12s;
  flex-shrink: 0;
}
.color-swatch:hover { transform: scale(1.15); }
.color-swatch.selected { box-shadow: 0 0 0 2px var(--clr-surface), 0 0 0 4px currentColor; }
.color-swatch.sm { width: 18px; height: 18px; }

/* Free colour picker (any colour, not just presets) */
.color-custom {
  width: 24px; height: 24px;
  padding: 0;
  border: 1px solid var(--clr-border);
  border-radius: 50%;
  background: none;
  cursor: pointer;
  flex-shrink: 0;
  -webkit-appearance: none; appearance: none;
  overflow: hidden;
}
.color-custom.sm { width: 18px; height: 18px; }
.color-custom::-webkit-color-swatch-wrapper { padding: 0; }
.color-custom::-webkit-color-swatch { border: none; border-radius: 50%; }
.color-custom::-moz-color-swatch { border: none; border-radius: 50%; }

/* Shared palette manager */
.palette-row { display: flex; flex-wrap: wrap; gap: 9px; align-items: center; min-height: 24px; }
.pal-chip {
  position: relative;
  width: 24px; height: 24px;
  border-radius: 50%;
  box-shadow: inset 0 0 0 1px rgba(0,0,0,0.12);
}
.pal-x {
  position: absolute; top: -5px; right: -5px;
  width: 15px; height: 15px;
  border-radius: 50%;
  background: var(--clr-surface);
  border: 1px solid var(--clr-border);
  color: var(--clr-text-2);
  font-size: 12px; line-height: 1;
  display: flex; align-items: center; justify-content: center;
  opacity: 0; transition: opacity 0.12s;
}
.pal-chip:hover .pal-x { opacity: 1; }
.pal-x:hover { color: var(--clr-danger); border-color: var(--clr-danger); }
.pal-empty { font-size: 12px; color: var(--clr-text-3); }
.palette-add { display: flex; align-items: center; gap: 10px; }

/* Baselines list */
.bl-list { display: flex; flex-direction: column; gap: 8px; }
.bl-row {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  padding: 10px 12px;
  background: var(--clr-surface);
  border: 1px solid var(--clr-border-light);
  border-radius: var(--r-md);
}
.bl-meta { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.bl-name { font-size: 13px; font-weight: 600; color: var(--clr-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.bl-sub { font-size: 11.5px; color: var(--clr-text-3); }

.field-input {
  border: 1.5px solid var(--clr-border);
  border-radius: var(--r-md);
  padding: 9px 12px;
  font-size: 14px;
  color: var(--clr-text);
  background: var(--clr-surface);
  outline: none;
  width: 100%;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.field-input.grow { flex: 1 1 200px; width: auto; }
.field-input:focus {
  border-color: var(--clr-accent);
  box-shadow: 0 0 0 3px rgba(0,113,227,0.12);
  background: var(--clr-surface);
}
.field-input.sm { padding: 6px 10px; font-size: 13px; }
.field-input::placeholder { color: var(--clr-text-3); }

.btn-add {
  padding: 9px 16px;
  font-size: 13px; font-weight: 600;
  color: var(--clr-accent);
  background: rgba(0,113,227,0.08);
  border-radius: var(--r-md);
  transition: background 0.15s;
  white-space: nowrap;
}
.btn-add:hover:not(:disabled) { background: rgba(0,113,227,0.14); }
.btn-add:disabled { opacity: 0.4; cursor: not-allowed; }

/* ── Legend label editor ─────────────────────────────────────────────── */
.legend-edit { display: flex; flex-direction: column; gap: 8px; }
.leg-row { display: flex; align-items: center; gap: 10px; }
.leg-ico { width: 20px; display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; }
.leg-row .field-input { flex: 1; width: auto; }
.leg-bar { width: 18px; height: 10px; border-radius: 3px; background: rgba(120,120,128,0.3); border: 1px solid rgba(120,120,128,0.55); }

/* Add-marker shape picker (searchable) */
.marker-add { margin-top: 10px; }
.marker-grid {
  display: flex; flex-wrap: wrap; gap: 6px;
  max-height: 180px; overflow-y: auto;
  margin-top: 8px;
}
.marker-add-btn {
  width: 32px; height: 32px;
  display: inline-flex; align-items: center; justify-content: center;
  border: 1px solid var(--clr-border); border-radius: var(--r-sm);
  background: var(--clr-bg);
  transition: background 0.12s, border-color 0.12s;
}
.marker-add-btn:hover { background: var(--clr-surface-2); border-color: var(--clr-accent); }

.fill-toggle {
  flex-shrink: 0;
  font-size: 11px; font-weight: 600;
  padding: 4px 9px; border-radius: 100px;
  border: 1px solid var(--clr-border);
  color: var(--clr-text-2); background: var(--clr-bg);
  transition: background 0.12s, color 0.12s, border-color 0.12s;
}
.fill-toggle.on { background: var(--clr-accent); color: #fff; border-color: var(--clr-accent); }

/* ── Areas grid ──────────────────────────────────────────────────────── */
.empty { font-size: 13px; color: var(--clr-text-3); text-align: center; padding: 28px 0; }

.lanes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 12px;
  align-items: start;
}

.lane-item {
  background: var(--clr-surface);
  border: 1px solid var(--clr-border-light);
  border-radius: var(--r-lg);
  overflow: hidden;
}

.lane-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 10px 10px 12px;
  border-bottom: 1px solid var(--clr-border-light);
}

.lane-dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }

.lane-name {
  flex: 1;
  font-size: 13px; font-weight: 600; color: var(--clr-text);
  cursor: pointer;
  padding: 2px 4px;
  border-radius: var(--r-xs);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  transition: background 0.12s;
}
.lane-name:hover { background: var(--clr-border-light); }

.inline-input {
  flex: 1;
  border: 1.5px solid var(--clr-accent);
  border-radius: var(--r-sm);
  padding: 3px 8px;
  font-size: 13px; font-weight: 600; color: var(--clr-text);
  background: var(--clr-surface); outline: none;
  box-shadow: 0 0 0 3px rgba(0,113,227,0.12);
  min-width: 0;
}
.inline-input.sm { font-size: 12.5px; font-weight: 500; }

.lane-actions { display: flex; gap: 2px; flex-shrink: 0; }

.icon-btn {
  width: 28px; height: 28px;
  display: flex; align-items: center; justify-content: center;
  background: transparent;
  border-radius: var(--r-sm);
  color: var(--clr-text-2);
  transition: background 0.12s, color 0.12s;
}
.icon-btn:hover:not(:disabled) { background: var(--clr-border-light); color: var(--clr-text); }
.icon-btn.danger:hover:not(:disabled) { background: rgba(255,59,48,0.1); color: var(--clr-danger); }
.icon-btn:disabled { opacity: 0.3; cursor: not-allowed; }

.lane-colors {
  display: flex;
  gap: 5px;
  flex-wrap: wrap;
  padding: 8px 12px;
  border-bottom: 1px solid var(--clr-border-light);
}

.sublanes {
  padding: 8px 12px 10px;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.sublane-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 8px;
  background: var(--clr-bg);
  border-radius: var(--r-sm);
  border: 1px solid var(--clr-border-light);
}

.sublane-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; opacity: 0.7; }

.sublane-name {
  flex: 1;
  font-size: 12.5px; color: var(--clr-text-2);
  cursor: pointer;
  padding: 1px 3px;
  border-radius: var(--r-xs);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  transition: background 0.12s;
}
.sublane-name:hover { background: var(--clr-border-light); }

.add-sub-row { display: flex; gap: 6px; margin-top: 2px; }

.btn-add-sub {
  width: 32px; height: 32px;
  flex-shrink: 0;
  background: rgba(0,113,227,0.08);
  color: var(--clr-accent);
  border-radius: var(--r-md);
  font-size: 18px;
  display: flex; align-items: center; justify-content: center;
  transition: background 0.15s;
}
.btn-add-sub:hover:not(:disabled) { background: rgba(0,113,227,0.14); }
.btn-add-sub:disabled { opacity: 0.35; cursor: not-allowed; }

/* ── Footer ──────────────────────────────────────────────────────────── */
.panel-footer {
  padding: 14px 20px;
  border-top: 1px solid var(--clr-border-light);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.reset-btn {
  background: none;
  font-size: 13px; font-weight: 600;
  color: var(--clr-text-2);
  padding: 6px 4px;
  transition: color 0.15s;
}
.reset-btn:hover { color: var(--clr-danger); }

.btn-primary {
  padding: 9px 24px;
  font-size: 14px; font-weight: 600;
  color: #fff;
  background: var(--clr-accent);
  border-radius: var(--r-md);
  transition: background 0.15s;
}
.btn-primary:hover { background: var(--clr-accent-hover); }

/* ── Mobile: full-screen sheet ───────────────────────────────────────── */
@media (max-width: 600px) {
  .backdrop { padding: 0; align-items: stretch; }
  .panel { max-width: 100%; height: 100%; border-radius: 0; }
  .lanes-grid { grid-template-columns: 1fr; }
}
</style>
