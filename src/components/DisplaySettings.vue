<template>
  <div class="tab-pane">
    <div class="card">
      <p class="section-label">Item icons</p>
      <p class="card-hint">Each item's icon &amp; colour come from its <strong>type</strong> (Types). These control how big they render.</p>
      <div class="opt-row">
        <label class="opt">Icon size
          <input type="range" min="10" max="22" step="1" v-model.number="settings.items.markerSize" />
          <span class="opt-val">{{ settings.items.markerSize }}px</span>
        </label>
        <label class="opt">Line thickness
          <input type="range" min="1" max="3" step="0.25" v-model.number="settings.items.markerStroke" />
          <span class="opt-val">{{ settings.items.markerStroke }}</span>
        </label>
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
      <div class="row-between">
        <span class="setting-name">Density
          <span class="setting-sub">how many markers stack in one spot</span>
        </span>
        <div class="seg-mini">
          <button type="button" :class="{ on: settings.items.density === 'stack' }" @click="settings.items.density = 'stack'" title="Stack them all (tallest)">Stack</button>
          <button type="button" :class="{ on: settings.items.density === 'cluster' }" @click="settings.items.density = 'cluster'" title="Cap the stack, collapse the rest into a +N chip">Cluster</button>
          <button type="button" :class="{ on: settings.items.density === 'rail' }" @click="settings.items.density = 'rail'" title="Collapse markers to a single tick row">Rail</button>
        </div>
      </div>
      <div v-if="settings.items.density === 'cluster'" class="opt-row">
        <label class="opt">Max rows before “+N”
          <input type="range" min="2" max="6" step="1" v-model.number="settings.items.densityRows" />
          <span class="opt-val">{{ settings.items.densityRows }}</span>
        </label>
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
        <label class="opt">Row margin
          <input type="range" min="0" max="20" step="1" v-model.number="settings.items.margin" />
          <span class="opt-val">{{ settings.items.margin }}px</span>
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

    <div class="card">
      <p class="section-label">Layout</p>
      <p class="card-hint">Width of the frozen Area and Sub-Area columns. Longer names truncate with a hover tooltip.</p>
      <div class="opt-row">
        <label class="opt">Area width
          <input type="range" min="150" max="280" step="2" v-model.number="settings.layout.areaWidth" />
          <span class="opt-val">{{ settings.layout.areaWidth }}px</span>
        </label>
      </div>
      <div class="opt-row">
        <label class="opt">Sub-area width
          <input type="range" min="150" max="280" step="2" v-model.number="settings.layout.subAreaWidth" />
          <span class="opt-val">{{ settings.layout.subAreaWidth }}px</span>
        </label>
      </div>
    </div>

    <button class="ds-reset" @click="onReset">Reset display to defaults</button>
  </div>
</template>

<script setup>
import { settings, resetSettings } from '../stores/useAppStore.js'
function onReset() {
  if (confirm('Reset all display settings (icons, gridlines, calendar weeks, item style, layout) to defaults for every dashboard?')) {
    resetSettings()
  }
}
</script>

<style scoped>
.tab-pane { display: flex; flex-direction: column; gap: 14px; }
.setting-sub { display: block; font-size: 11.5px; font-weight: 400; color: var(--clr-text-3); margin-top: 1px; }
.toggle { width: 42px; height: 26px; border-radius: 13px; background: var(--clr-border); position: relative; transition: background 0.22s; flex-shrink: 0; }
.toggle.active { background: var(--clr-accent); }
.toggle-knob { position: absolute; width: 20px; height: 20px; border-radius: 50%; background: white; top: 3px; left: 3px; transition: transform 0.22s; box-shadow: 0 1px 4px rgba(0,0,0,0.22); }
.toggle.active .toggle-knob { transform: translateX(16px); }
.opt-row { display: flex; flex-wrap: wrap; gap: 16px; align-items: center; }
.opt { display: inline-flex; align-items: center; gap: 6px; font-size: 12px; color: var(--clr-text-2); }
.opt input[type="color"] { width: 30px; height: 22px; padding: 0; border: 1px solid var(--clr-border); border-radius: 6px; background: none; cursor: pointer; }
.opt input[type="range"] { width: 92px; }
.opt-val { font-size: 11px; color: var(--clr-text-3); min-width: 34px; }
.opt-note { font-size: 11px; color: var(--clr-text-3); }
.seg-mini { display: inline-flex; border: 1px solid var(--clr-border); border-radius: var(--r-md); overflow: hidden; }
.seg-mini button { padding: 5px 11px; font-size: 12px; font-weight: 500; color: var(--clr-text-2); background: var(--clr-bg); transition: background 0.12s, color 0.12s; }
.seg-mini button + button { border-left: 1px solid var(--clr-border); }
.seg-mini button:hover:not(.on) { background: var(--clr-surface-2); }
.seg-mini button.on { background: var(--clr-accent); color: #fff; }
.ds-reset { align-self: flex-start; font-size: 12.5px; font-weight: 600; color: var(--clr-text-3);
  background: none; padding: 6px 2px; transition: color 0.15s; }
.ds-reset:hover { color: var(--clr-danger); }
</style>
