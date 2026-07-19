<template>
  <Teleport to="body" :disabled="embedded">
    <Transition :name="embedded ? 'none' : 'modal'">
      <div :class="embedded ? 'embed-host' : 'backdrop'">
        <Transition :name="embedded ? 'none' : 'modal-panel'" :appear="!embedded">
          <div class="panel" :class="{ embedded }">
            <!-- Header -->
            <div class="panel-header">
              <div class="panel-meta">
                <button v-if="typeStatuses.length" type="button" class="ms-status-chip ms-status-chip-sm" :style="{ '--chip': currentStatusColor }" :title="formLocked ? 'View the status flow' : 'Open the status flow to change status'" @click="tab = 'flow'">
                  <MarkerIcon :shape="currentType?.icon || 'l:Diamond'" :color="currentStatusColor" :size="14" :fill="currentType?.fill !== false" class="ms-status-ico" />
                  <span class="ms-status-lbl">{{ currentStatusLabel }}</span>
                  <Workflow :size="12" class="ms-status-flowico" />
                </button>
                <span v-if="mode === 'add'" class="panel-month">{{ displayMonth }}</span>
                <button v-if="mode === 'edit' && milestone && !milestone.sourceSystem" type="button" class="panel-ver" :class="{ on: tab === 'history' }" title="View version history" @click="tab = 'history'">v{{ milestone.version || 1 }} <History :size="11" /></button>
                <span v-if="readOnly" class="ro-badge"><Lock :size="11" :stroke-width="2.5" /> Read-only</span>
                <span v-if="mode === 'edit' && milestone && !milestone.sourceSystem" class="panel-attrib-inline">
                  <template v-if="milestone.updatedBy && (milestone.version || 1) > 1">Last edit by <strong>{{ who(milestone.updatedBy) }}</strong><span v-if="milestone.updatedAt"> · {{ fmtStamp(milestone.updatedAt) }}</span></template>
                  <template v-else-if="milestone.createdBy">Added by <strong>{{ who(milestone.createdBy) }}</strong><span v-if="milestone.createdAt"> · {{ fmtStamp(milestone.createdAt) }}</span></template>
                </span>
              </div>
              <div class="panel-actions-top">
                <template v-if="mode === 'edit' && milestone">
                  <button type="button" class="icon-act" :class="{ done: copied === 'link' }" :title="copied === 'link' ? 'Copied' : 'Copy link to this view'" @click="copy('link')"><Check v-if="copied === 'link'" :size="15" :stroke-width="2.5" /><Link2 v-else :size="15" /></button>
                  <button type="button" class="icon-act" :class="{ on: viewFormat === 'form' }" title="Normal view" @click="setFormat('form')"><AlignLeft :size="15" /></button>
                  <button type="button" class="icon-act" :class="{ on: viewFormat === 'json' }" title="View as JSON" @click="setFormat('json')"><Braces :size="15" /></button>
                  <button type="button" class="icon-act" :class="{ on: viewFormat === 'yaml' }" title="View as YAML" @click="setFormat('yaml')"><FileText :size="15" /></button>
                </template>
                <button v-if="canPropose && !proposing && viewVersion == null && viewFormat === 'form'" type="button" class="propose-act" @click="proposing = true; editing = true">{{ mode === 'add' ? 'Propose new item' : 'Propose change' }}</button>
                <button v-if="mode === 'edit' && !readOnly && editable && viewVersion == null" type="button" class="icon-act danger" title="Delete" @click="remove">
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6M10 11v6M14 11v6"/>
                  </svg>
                </button>
                <button v-if="embedded && !editable && !readOnly && viewVersion == null" type="button" class="icon-act primary" title="Edit" @click="setFormat('form'); editing = true"><Pencil :size="15" /></button>
                <button v-if="editable && viewVersion == null" type="button" class="icon-act primary" :class="{ saved: justSaved }" :title="proposing ? 'Submit proposal' : (mode === 'edit' ? 'Save' : 'Create')" @click="onSave"><Check :size="16" :stroke-width="2.5" /></button>
                <button v-if="!embedded" type="button" class="icon-act" title="Close" @click="$emit('close')">
                  <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                    <path d="M1 1l12 12M13 1L1 13" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/>
                  </svg>
                </button>
              </div>
            </div>

            <div v-if="proposing" class="propose-banner">
              <span class="pb-text">{{ mode === 'add' ? 'Proposing a new item' : 'Proposing a change' }} — the owner must approve it before it goes live.</span>
              <input v-model="proposeNote" class="pb-note" placeholder="Reason (optional)" />
            </div>

            <div v-if="viewVersion != null" class="version-banner">
              <span class="vb-text"><History :size="13" /> Viewing <strong>version {{ viewVersion }}</strong> — a read-only snapshot.</span>
              <button type="button" class="vb-back" @click="backToLatest">Back to latest (v{{ headVersion }})</button>
            </div>

            <!-- Form -->
            <form class="panel-form" :class="{ 'read-mode': formLocked }" @submit.prevent="submit">
              <!-- JSON / YAML view replaces the FIELDS block; the tabs (flow, console…)
                   below stay visible. -->
              <div v-if="viewFormat !== 'form'" class="ms-code-view"><pre class="ms-code">{{ formattedText }}</pre></div>
              <fieldset v-else class="ms-group" :disabled="formLocked">
              <div class="field span2">
                <label class="field-label">Title <span class="req">*</span></label>
                <input
                  v-model="form.title"
                  class="field-input"
                  placeholder="Short description of the milestone"
                  autocomplete="off"
                  required
                  ref="titleInput"
                />
              </div>

              <div v-if="isTimelineType" class="field span2">
                <label class="field-label">Area</label>
                <select class="field-input" :disabled="formLocked" v-model="form.swimlaneId">
                  <option value="">— No area (off-timeline) —</option>
                  <option v-for="sw in timelineLanes" :key="sw.id" :value="sw.id">{{ sw.name }}</option>
                </select>
              </div>

              <div v-if="isTimelineType && chosenLaneSubs.length" class="field">
                <label class="field-label">Sub-area</label>
                <select class="field-input" :disabled="formLocked" v-model="form.subLaneId">
                  <option value="">— Top of area —</option>
                  <option v-for="sub in chosenLaneSubs" :key="sub.id" :value="sub.id">{{ sub.name }}</option>
                </select>
              </div>

              <div class="field">
                <label class="field-label">Type</label>
                <div class="type-row">
                  <span class="type-ico"><MarkerIcon :shape="currentType?.icon || 'l:Diamond'" :color="currentType?.color || swimlane?.color || '#8a8a8e'" :size="18" :fill="currentType?.fill !== false" /></span>
                  <select v-if="mode === 'add' && !formLocked" class="field-input" :value="form.typeKey" @change="applyType($event.target.value)">
                    <option v-for="t in itemTypes.list" :key="t.key" :value="t.key">{{ t.label }}</option>
                  </select>
                  <span v-else class="type-static">{{ currentType?.label || form.typeKey }}</span>
                </div>
                <p class="type-hint">{{ mode === 'add' && !formLocked ? 'The icon comes from the type — set it under Settings → Types.' : 'The type is fixed once an item is created.' }}</p>
              </div>

              <!-- Type-specific fields: schema comes from the selected item type. -->
              <div v-if="currentTypeFields.length" class="field type-fields span2">
                <label class="field-label">Fields</label>
                <dl v-if="!editable" class="read-fields">
                  <template v-for="f in readFieldRows" :key="f.key">
                    <dt>{{ f.label }}</dt>
                    <dd v-if="f.refs" :class="{ 'read-empty': !f.refs.length }">
                      <span v-if="f.refs.length" class="read-refs">
                        <span v-for="r in f.refs" :key="r.id + ':' + (r.version || '')" class="read-pill" :class="{ missing: !r.exists }" :style="{ color: r.color, background: r.color + '22' }" @click="r.exists && openRef(r)"><span class="read-pill-dot" :style="{ background: r.dot }"></span><MarkerIcon :shape="r.icon" :color="r.color" :size="12" :fill="r.fill" />{{ r.title }}<span v-if="r.version" class="read-pill-ver">v{{ r.version }}</span></span>
                      </span>
                      <template v-else>—</template>
                    </dd>
                    <dd v-else :class="{ 'read-empty': !f.v, 'read-prose': f.prose }">{{ f.v || '—' }}</dd>
                  </template>
                </dl>
                <div v-for="f in (editable ? currentTypeFields : [])" :key="f.key" class="tf-row">
                  <label class="tf-label">{{ f.label || f.key }}<span v-if="f.required" class="tf-req" title="Required">*</span></label>
                  <select v-if="f.type === 'select'" class="field-input" :class="{ 'tf-invalid': invalidFields.includes(f.key) }" :disabled="formLocked" v-model="form.data[f.key]">
                    <option value="">—</option>
                    <option v-for="o in (f.options || [])" :key="o" :value="o">{{ o }}</option>
                  </select>
                  <div v-else-if="f.type === 'multiselect'" class="tf-checks" :class="{ 'tf-invalid': invalidFields.includes(f.key) }">
                    <label v-for="o in (f.options || [])" :key="o" class="tf-check">
                      <input type="checkbox" :disabled="formLocked" :checked="Array.isArray(form.data[f.key]) && form.data[f.key].includes(o)" @change="toggleMulti(f.key, o, $event.target.checked)" /> {{ o }}
                    </label>
                    <span v-if="!(f.options || []).length" class="tf-empty">No options defined.</span>
                  </div>
                  <input v-else-if="f.type === 'number'" type="number" class="field-input" :class="{ 'tf-invalid': invalidFields.includes(f.key) }" :disabled="formLocked" v-model="form.data[f.key]" />
                  <input v-else-if="f.type === 'date'" type="date" class="field-input" :class="{ 'tf-invalid': invalidFields.includes(f.key) }" :disabled="formLocked" v-model="form.data[f.key]" />
                  <select v-else-if="f.type === 'reference' && !f.refMulti" class="field-input" :class="{ 'tf-invalid': invalidFields.includes(f.key) }" :disabled="formLocked" v-model="form.data[f.key]">
                    <option value="">—</option>
                    <option v-for="r in refItems(f)" :key="r.id" :value="r.id">{{ r.title }}</option>
                  </select>
                  <div v-else-if="f.type === 'reference'" class="tf-multiref" :class="{ 'tf-invalid': invalidFields.includes(f.key) }" @focusout="onComboBlur">
                    <input
                      class="field-input" :disabled="formLocked" v-model="refSearch[f.key]"
                      :placeholder="refItems(f).length ? 'Search ' + refTypeLabel(f.refType) + ' to add…' : ''"
                      @focus="refOpen = f.key"
                    />
                    <div v-if="refOpen === f.key && refItems(f).length" class="tf-combo-list">
                      <button
                        v-for="r in filteredRefItems(f)" :key="r.id" type="button"
                        class="tf-combo-opt" :class="{ on: isSelected(f, r.id) }"
                        @mousedown.prevent @click="toggleMulti(f.key, r.id, !isSelected(f, r.id))"
                      ><span class="tf-combo-check">{{ isSelected(f, r.id) ? '✓' : '' }}</span>{{ r.title }}</button>
                      <div v-if="!filteredRefItems(f).length" class="tf-combo-empty">No matches</div>
                      <div v-else-if="moreCount(f)" class="tf-combo-more">+{{ moreCount(f) }} more — keep typing to narrow</div>
                    </div>
                    <span v-if="!refItems(f).length" class="tf-empty">{{ refHint(f) }}</span>
                  </div>
                  <textarea v-else-if="f.type === 'textarea'" class="field-textarea" :class="{ 'tf-invalid': invalidFields.includes(f.key) }" :rows="readOnly ? 8 : 2" :disabled="formLocked" v-model="form.data[f.key]"></textarea>
                  <input v-else type="text" class="field-input" :class="{ 'tf-invalid': invalidFields.includes(f.key) }" :disabled="formLocked" v-model="form.data[f.key]" />
                  <span v-if="f.type === 'reference' && !f.refMulti && !refItems(f).length" class="tf-refhint">{{ refHint(f) }}</span>
                  <div v-if="f.type === 'reference' && selectedRefs(f).length" class="tf-pins">
                    <span class="tf-pins-lbl" title="Track the latest revision, or pin each reference to a specific version">Version</span>
                    <span v-for="id in selectedRefs(f)" :key="id" class="tf-pinitem">
                      <span class="tf-pin-name">{{ refTitle(id) }}</span>
                      <select class="tf-pin-sel" :class="{ on: isPinned(f.key, id) }" :disabled="formLocked" v-model="refPins[f.key][id]">
                        <option :value="''">latest (v{{ refHead(id) }})</option>
                        <option v-for="v in refVersions(id)" :key="v" :value="v">v{{ v }}</option>
                      </select>
                      <button v-if="f.refMulti" type="button" class="tf-pin-x" :disabled="formLocked" title="Remove reference" @click="toggleMulti(f.key, id, false)">×</button>
                    </span>
                  </div>
                </div>
              </div>

              <!-- Exclusive resource (#128): a capacity-1 backlog item can be used
                   by many timeline items, but never by two whose bookings overlap
                   in time (e.g. a machine can't be on two sites at once). -->
              <div v-if="!isTimelineType" class="field span2 excl-block">
                <label class="field-label">Exclusive resource</label>
                <span v-if="!editable" class="read-val">{{ exclusiveSummary }}</span>
                <div v-else class="excl-edit">
                  <div class="excl-modes">
                    <label v-for="m in EXCL_MODES" :key="m.key" class="excl-mode" :class="{ on: exclusive.mode === m.key }">
                      <input type="radio" :value="m.key" v-model="exclusive.mode" :disabled="formLocked" />
                      <span>{{ m.label }}</span>
                    </label>
                  </div>
                  <p class="excl-hint">{{ EXCL_MODES.find(m => m.key === exclusive.mode)?.hint }}</p>
                  <div v-if="exclusive.mode !== 'off'" class="excl-buffer">
                    <label class="excl-buf">Setup days before <input type="number" min="0" class="field-input excl-num" :disabled="formLocked" v-model.number="exclusive.before" /></label>
                    <label class="excl-buf">after <input type="number" min="0" class="field-input excl-num" :disabled="formLocked" v-model.number="exclusive.after" /></label>
                  </div>
                </div>
              </div>

              <div class="field">
                <label class="field-label">
                  Maturity
                  <span v-if="form.maturity && editable" class="mat-current">{{ MATURITY_STAGES[form.maturity - 1] }}</span>
                </label>
                <span v-if="!editable" class="read-val">{{ form.maturity ? MATURITY_STAGES[form.maturity - 1] : '—' }}</span>
                <div v-else class="maturity-row">
                  <button
                    type="button"
                    class="maturity-btn"
                    :class="{ on: !form.maturity }"
                    title="No maturity"
                    @click="form.maturity = null"
                  >
                    <MaturityGlyph :level="0" variant="grid" :color="!form.maturity ? 'var(--clr-text-2)' : '#9aa0a6'" />
                    <span class="maturity-lbl">None</span>
                  </button>
                  <button
                    v-for="(s, i) in MATURITY_STAGES"
                    :key="s"
                    type="button"
                    class="maturity-btn"
                    :class="{ on: form.maturity === i + 1 }"
                    :title="s"
                    @click="form.maturity = i + 1"
                  >
                    <MaturityGlyph :level="i + 1" variant="grid" :color="form.maturity === i + 1 ? 'var(--clr-text-2)' : '#9aa0a6'" />
                    <span class="maturity-lbl">{{ s }}</span>
                  </button>
                </div>
              </div>

              <div class="field">
                <label class="field-label">
                  Progress
                  <span v-if="form.progress != null && editable" class="mat-current">{{ form.progress }}%</span>
                </label>
                <span v-if="!editable" class="read-val">{{ form.progress != null ? form.progress + '%' : '—' }}</span>
                <div v-else class="progress-row">
                  <button
                    type="button"
                    class="maturity-btn"
                    :class="{ on: form.progress == null }"
                    @click="form.progress = null"
                  >None</button>
                  <input
                    type="range" min="0" max="100" step="1"
                    class="progress-slider"
                    :value="form.progress ?? 0"
                    @input="form.progress = +$event.target.value"
                  />
                </div>
              </div>
              <!-- Assignment + scheduling live in the top block now; the tabs below
                   are for relations, flow & history. -->
              <div class="field">
                <label class="field-label">Assigned to</label>
                <select class="field-input" :disabled="formLocked" v-model="form.assigneeId">
                  <option :value="null">Unassigned</option>
                  <option v-for="mb in workspace.members" :key="mb.userId" :value="mb.userId">{{ mb.username }}</option>
                </select>
              </div>
              <template v-if="isTimelineType">
                <div v-if="form.kind === 'event'" class="span2 two-col">
                  <div class="field">
                    <label class="field-label">Start <span class="req">*</span></label>
                    <input v-model="form.startDate" type="date" class="field-input field-date" />
                  </div>
                  <div class="field">
                    <label class="field-label">End</label>
                    <input v-model="form.endDate" type="date" class="field-input field-date" :min="form.startDate" />
                  </div>
                </div>
                <div v-else class="field">
                  <label class="field-label">When</label>
                  <input v-model="form.when" type="date" class="field-input field-date" />
                </div>
              </template>
              <p v-if="dateError" class="field-error span2">{{ dateError }}</p>
              </fieldset>

              <div class="ms-tabs" role="tablist">
                <button v-if="typeStatuses.length" type="button" class="ms-tab" :class="{ active: tab === 'flow' }" @click="tab = 'flow'">Flow</button>
                <button v-if="currentSchedulable" type="button" class="ms-tab" :class="{ active: tab === 'deps' }" @click="tab = 'deps'">Dependencies</button>
                <button type="button" class="ms-tab" :class="{ active: tab === 'uses' }" @click="tab = 'uses'">Uses</button>
                <button v-if="currentSchedulable" type="button" class="ms-tab" :class="{ active: tab === 'groups' }" @click="tab = 'groups'">Groups</button>
                <button v-if="mode === 'edit' && !milestone?.sourceSystem" type="button" class="ms-tab" :class="{ active: tab === 'history' }" @click="tab = 'history'">History</button>
              </div>

              <!-- The History tab is read-only display, so it's never form-disabled
                   (you can browse versions even when the rest is read-only). -->
              <fieldset class="ms-tab-body" :disabled="formLocked && tab !== 'history'">
              <div v-show="tab === 'history'" class="ms-panel ms-history">
                <ItemHistory v-if="milestone && tab === 'history'" :key="milestone.id" :item-id="milestone.id" :current-version="viewVersion || headVersion" @select="onSelectVersion" />
              </div>
              <div v-show="tab === 'flow'" class="ms-panel ms-flow">
                <StatusFlow v-if="typeStatuses.length" inline :statuses="typeStatuses" :current="form.status" :version="milestone?.version || 1" :read-only="readOnly || viewVersion != null" :viewing-version="viewVersion || 0" :arrangeable="false" :layout="currentType?.layout" @advance="onFlowAdvance" @back-to-latest="backToLatest" />
              </div>

              <div v-show="tab === 'deps' && !editable" class="ms-panel">
                <ul v-if="readDeps.length" class="read-deps">
                  <li v-for="g in readDeps" :key="g.rel">
                    <span class="read-dep-rel">{{ g.rel }}</span>
                    <span class="read-refs">
                      <span v-for="d in g.items" :key="d.id" class="read-pill" :class="{ 'pill-conflict': pillConflict(d.id) }" :style="{ color: d.color, background: d.color + '22' }" @click="openRef({ id: d.id, version: d.version, exists: true })"><span class="read-pill-dot" :style="{ background: d.dot }"></span><MarkerIcon :shape="d.icon" :color="d.color" :size="12" :fill="d.fill" />{{ d.title }}<span v-if="d.version" class="read-pill-ver">v{{ d.version }}</span><AlertTriangle v-if="pillConflict(d.id)" :size="12" :stroke-width="2.4" color="#FF3B30" :title="pillConflict(d.id)" /></span>
                    </span>
                  </li>
                </ul>
                <p v-else class="read-none">No dependencies.</p>
              </div>

              <div v-show="tab === 'deps' && editable" class="ms-panel">
              <div class="field">
                <label class="field-label">Relationship</label>
                <select v-model="relType" class="field-input" :disabled="formLocked">
                  <option v-for="r in availableRelTypes" :key="r.key" :value="r.key">{{ r.label }} ↔ {{ r.inverse }}</option>
                </select>
              </div>
              <!-- The two directions of the selected relationship (side by side) -->
              <div class="two-col dep-cols">
              <div class="field">
                <label class="field-label">
                  {{ relDef.label }}
                  <span v-if="localLinkedIds.size > 0" class="link-count link-toggle" :class="{ on: showOnly1 }" :title="showOnly1 ? 'Show all' : 'Show only selected'" @click.prevent.stop="showOnly1 = !showOnly1">{{ localLinkedIds.size }}</span>
                </label>
                <div class="ms-picker">
                  <div class="picker-search">
                    <svg width="13" height="13" viewBox="0 0 13 13" fill="none" class="search-icon">
                      <circle cx="5.5" cy="5.5" r="4" stroke="currentColor" stroke-width="1.5"/>
                      <path d="M9 9l2.5 2.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                    </svg>
                    <input
                      v-model="pickerSearch"
                      class="picker-input"
                      placeholder="Search milestones…"
                      autocomplete="off"
                    />
                    <button
                      v-if="pickerSearch"
                      type="button"
                      class="picker-clear"
                      @click="pickerSearch = ''"
                    >×</button>
                  </div>
                  <div class="picker-list">
                    <template v-for="group in pickerGroups" :key="group.swimlane.id + '-' + (group.subLane?.id ?? 'root')">
                      <div class="picker-group-header">
                        <span class="picker-group-dot" :style="{ background: group.swimlane.color }"></span>
                        {{ group.swimlane.name }}{{ group.subLane ? ' · ' + group.subLane.name : '' }}
                      </div>
                      <button
                        v-for="m in group.milestones"
                        :key="m.id"
                        type="button"
                        class="picker-item"
                        :class="{ 'picker-active': localLinkedIds.has(m.id) }"
                        :style="localLinkedIds.has(m.id) ? activePickerStyle(group.swimlane.color) : {}"
                        @click="toggleLocalLink(m.id)"
                      >
                        <span class="picker-dot" :style="{ background: group.swimlane.color }"></span>
                        <div class="picker-info">
                          <span class="picker-title">{{ m.title }}</span>
                          <span class="picker-meta">{{ MONTHS[m.month - 1] }} {{ m.year !== year ? m.year : '' }}</span>
                        </div>
                        <svg v-if="localLinkedIds.has(m.id)" class="picker-check" width="14" height="14" viewBox="0 0 14 14" fill="none">
                          <path d="M2.5 7L5.5 10L11.5 4" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/>
                        </svg>
                      </button>
                    </template>
                    <div v-if="pickerGroups.length === 0" class="picker-empty">
                      {{ pickerSearch ? 'No milestones match your search' : 'No other milestones yet' }}
                    </div>
                  </div>
                </div>
              </div>

              <!-- Required by (reverse dependency) -->
              <div class="field">
                <label class="field-label">
                  {{ relDef.inverse }}
                  <span v-if="localDependentIds.size > 0" class="link-count link-toggle" :class="{ on: showOnly2 }" :title="showOnly2 ? 'Show all' : 'Show only selected'" @click.prevent.stop="showOnly2 = !showOnly2">{{ localDependentIds.size }}</span>
                </label>
                <div class="ms-picker">
                  <div class="picker-search">
                    <svg width="13" height="13" viewBox="0 0 13 13" fill="none" class="search-icon">
                      <circle cx="5.5" cy="5.5" r="4" stroke="currentColor" stroke-width="1.5"/>
                      <path d="M9 9l2.5 2.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                    </svg>
                    <input
                      v-model="pickerSearch2"
                      class="picker-input"
                      placeholder="Search milestones…"
                      autocomplete="off"
                    />
                    <button
                      v-if="pickerSearch2"
                      type="button"
                      class="picker-clear"
                      @click="pickerSearch2 = ''"
                    >×</button>
                  </div>
                  <div class="picker-list">
                    <template v-for="group in pickerGroups2" :key="'rb-' + group.swimlane.id + '-' + (group.subLane?.id ?? 'root')">
                      <div class="picker-group-header">
                        <span class="picker-group-dot" :style="{ background: group.swimlane.color }"></span>
                        {{ group.swimlane.name }}{{ group.subLane ? ' · ' + group.subLane.name : '' }}
                      </div>
                      <button
                        v-for="m in group.milestones"
                        :key="m.id"
                        type="button"
                        class="picker-item"
                        :class="{ 'picker-active': localDependentIds.has(m.id) }"
                        :style="localDependentIds.has(m.id) ? activePickerStyle(group.swimlane.color) : {}"
                        @click="toggleLocalDependent(m.id)"
                      >
                        <span class="picker-dot" :style="{ background: group.swimlane.color }"></span>
                        <div class="picker-info">
                          <span class="picker-title">{{ m.title }}</span>
                          <span class="picker-meta">{{ MONTHS[m.month - 1] }} {{ m.year !== year ? m.year : '' }}</span>
                        </div>
                        <svg v-if="localDependentIds.has(m.id)" class="picker-check" width="14" height="14" viewBox="0 0 14 14" fill="none">
                          <path d="M2.5 7L5.5 10L11.5 4" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/>
                        </svg>
                      </button>
                    </template>
                    <div v-if="pickerGroups2.length === 0" class="picker-empty">
                      {{ pickerSearch2 ? 'No milestones match your search' : 'No other milestones yet' }}
                    </div>
                  </div>
                </div>
              </div>
              </div>
              </div>

              <div v-show="tab === 'uses' && !editable" class="ms-panel">
                <ul v-if="readUses.length" class="read-deps">
                  <li v-for="g in readUses" :key="g.rel">
                    <span class="read-dep-rel">{{ g.rel }}</span>
                    <span class="read-refs">
                      <span v-for="d in g.items" :key="d.id" class="read-pill" :class="{ 'pill-conflict': pillConflict(d.id) }" :style="{ color: d.color, background: d.color + '22' }" @click="openRef({ id: d.id, version: d.version, exists: true })"><span class="read-pill-dot" :style="{ background: d.dot }"></span><MarkerIcon :shape="d.icon" :color="d.color" :size="12" :fill="d.fill" />{{ d.title }}<span v-if="d.version" class="read-pill-ver">v{{ d.version }}</span><AlertTriangle v-if="pillConflict(d.id)" :size="12" :stroke-width="2.4" color="#FF3B30" :title="pillConflict(d.id)" /></span>
                    </span>
                  </li>
                </ul>
                <p v-else class="read-none">Uses nothing yet.</p>
              </div>

              <div v-show="tab === 'uses' && editable" class="ms-panel">
              <p class="uses-hint">Backlog items this one <strong>uses</strong> (components / sub-artifacts). Timeline items aren't selectable here.</p>
              <div class="two-col dep-cols">
              <div class="field">
                <label class="field-label">
                  Uses
                  <span v-if="localUsesIds.size > 0" class="link-count link-toggle" :class="{ on: showOnlyU1 }" :title="showOnlyU1 ? 'Show all' : 'Show only selected'" @click.prevent.stop="showOnlyU1 = !showOnlyU1">{{ localUsesIds.size }}</span>
                </label>
                <div class="ms-picker">
                  <div class="picker-search">
                    <svg width="13" height="13" viewBox="0 0 13 13" fill="none" class="search-icon"><circle cx="5.5" cy="5.5" r="4" stroke="currentColor" stroke-width="1.5"/><path d="M9 9l2.5 2.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
                    <input v-model="usesSearch" class="picker-input" placeholder="Search backlog items…" autocomplete="off" />
                    <button v-if="usesSearch" type="button" class="picker-clear" @click="usesSearch = ''">×</button>
                  </div>
                  <div class="picker-list">
                    <template v-for="group in usesGroups" :key="'u-' + group.swimlane.id">
                      <div v-for="m in group.milestones" :key="m.id" class="picker-row">
                        <button
                          type="button" class="picker-item"
                          :class="{ 'picker-active': localUsesIds.has(m.id) }"
                          :style="localUsesIds.has(m.id) ? activePickerStyle(usesDot(m)) : {}"
                          @click="toggleUses(m.id)"
                        >
                          <span class="picker-dot" :style="{ background: usesDot(m) }"></span>
                          <div class="picker-info"><span class="picker-title">{{ m.title }}</span><span class="picker-meta">{{ usesTypeLabel(m) }}</span></div>
                          <svg v-if="localUsesIds.has(m.id)" class="picker-check" width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M2.5 7L5.5 10L11.5 4" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/></svg>
                        </button>
                        <select v-if="localUsesIds.has(m.id)" class="picker-ver" v-model="usesPins[m.id]" title="Pin a version (or track the latest)" @click.stop @mousedown.stop>
                          <option :value="''">latest</option>
                          <option v-for="v in refVersions(m.id)" :key="v" :value="v">v{{ v }}</option>
                        </select>
                      </div>
                    </template>
                    <div v-if="usesGroups.length === 0" class="picker-empty">{{ usesSearch ? 'No backlog items match' : 'No backlog items to reference' }}</div>
                  </div>
                </div>
              </div>
              <div class="field">
                <label class="field-label">
                  Used by
                  <span v-if="localUsedByIds.size > 0" class="link-count link-toggle" :class="{ on: showOnlyU2 }" :title="showOnlyU2 ? 'Show all' : 'Show only selected'" @click.prevent.stop="showOnlyU2 = !showOnlyU2">{{ localUsedByIds.size }}</span>
                </label>
                <div class="ms-picker">
                  <div class="picker-search">
                    <svg width="13" height="13" viewBox="0 0 13 13" fill="none" class="search-icon"><circle cx="5.5" cy="5.5" r="4" stroke="currentColor" stroke-width="1.5"/><path d="M9 9l2.5 2.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
                    <input v-model="usesSearch2" class="picker-input" placeholder="Search backlog items…" autocomplete="off" />
                    <button v-if="usesSearch2" type="button" class="picker-clear" @click="usesSearch2 = ''">×</button>
                  </div>
                  <div class="picker-list">
                    <template v-for="group in usesGroups2" :key="'ub-' + group.swimlane.id">
                      <button
                        v-for="m in group.milestones" :key="m.id" type="button" class="picker-item"
                        :class="{ 'picker-active': localUsedByIds.has(m.id) }"
                        :style="localUsedByIds.has(m.id) ? activePickerStyle(usesDot(m)) : {}"
                        @click="toggleUsedBy(m.id)"
                      >
                        <span class="picker-dot" :style="{ background: usesDot(m) }"></span>
                        <div class="picker-info"><span class="picker-title">{{ m.title }}</span><span class="picker-meta">{{ usesTypeLabel(m) }}</span></div>
                        <svg v-if="localUsedByIds.has(m.id)" class="picker-check" width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M2.5 7L5.5 10L11.5 4" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/></svg>
                      </button>
                    </template>
                    <div v-if="usesGroups2.length === 0" class="picker-empty">{{ usesSearch2 ? 'No backlog items match' : 'No backlog items to reference' }}</div>
                  </div>
                </div>
              </div>
              </div>
              </div>

              <div v-show="tab === 'groups' && !editable" class="ms-panel">
                <div v-if="readItemGroups.length" class="read-groups">
                  <span v-for="g in readItemGroups" :key="g.id" class="read-group" :style="{ background: (g.color || '#888') + '22', color: g.color || '#888' }">{{ g.name }}</span>
                </div>
                <p v-else class="read-none">No groups.</p>
              </div>

              <div v-show="tab === 'groups' && editable" class="ms-panel">
              <!-- Group membership -->
              <div v-if="groups.list.length" class="field">
                <label class="field-label">
                  Groups
                  <span v-if="localGroupIds.size > 0" class="link-count">{{ localGroupIds.size }}</span>
                </label>
                <div class="grp-chips">
                  <button
                    v-for="g in groups.list"
                    :key="g.id"
                    type="button"
                    class="grp-chip"
                    :class="{ on: localGroupIds.has(g.id) }"
                    @click="toggleLocalGroup(g.id)"
                  >
                    <span class="grp-dot" :style="{ background: g.color }"></span>{{ g.name }}
                  </button>
                </div>
              </div>
              </div>
              </fieldset>

              <!-- Enter-to-save (actions live in the header) -->
              <button type="submit" class="hidden-submit" tabindex="-1" aria-hidden="true"></button>
            </form>

            <!-- Always-visible item console: the status flow teleports its terminal
                 here, so it stays on screen across every tab. -->
            <div v-if="typeStatuses.length" id="modal-console-dock" class="modal-console-dock"></div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { reactive, ref, computed, onMounted, watch } from 'vue'
import { useAppStore, MONTHS, MATURITY_STAGES, store, groups, swatchColors, stripMarkdown, itemTypes, itemTypeByKey, RELATIONSHIP_TYPES, workspace, session, baselines, canEditWorkspace, canProposeChanges, proposeChange, proposeCreate, memberName, memberInitials, memberById, openProfile, STATUS_TONES, toneColor, statusColor, parseRef, itemLink, itemStatus, isSchedulableItem, ui, pushNav, checkResourceConflicts, resourceConflicts } from '../stores/useAppStore.js'

function who(id) { return id ? (memberName(id) || 'someone') : 'system' }
function fmtStamp(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  return isNaN(d) ? '' : d.toLocaleString('en-GB', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}
import { api } from '../api.js'
import MaturityGlyph from './MaturityGlyph.vue'
import MarkerIcon from './MarkerIcon.vue'
import ItemHistory from './ItemHistory.vue'
import StatusFlow from './StatusFlow.vue'
import { Lock, History, Workflow, Link2, Braces, FileText, Pencil, Check, AlignLeft, AlertTriangle } from 'lucide-vue-next'

const props = defineProps({
  mode:      { type: String,  default: 'add' },
  swimlane:  { type: Object,  default: null },
  subLane:   { type: Object,  default: null },
  month:     { type: Number,  default: 1 },
  year:      { type: Number,  default: 2026 },
  date:      { type: String,  default: null },
  milestone: { type: Object,  default: null },
  initialType: { type: String, default: '' }, // preselect a type (Explorer "+ New")
  initialTab: { type: String, default: 'flow' }, // open straight on a tab (e.g. "history")
  proposeMode: { type: Boolean, default: false }, // open straight in "propose a new item" mode
  embedded: { type: Boolean, default: false }, // render inline (Explorer pane), not as a pop-up
})

const emit = defineEmits(['close'])
const { addMilestone, updateMilestone, deleteMilestone, addLink, removeLink, itemGroupIds, setItemGroups } = useAppStore()

const TABS = ['flow', 'deps', 'uses', 'groups', 'history']
const tab = ref(props.mode === 'edit' && TABS.includes(props.initialTab) ? props.initialTab : 'flow')
const invalidFields = ref([]) // keys of empty required fields, framed red
const isFieldEmpty = (v) => v == null || v === '' || (Array.isArray(v) && v.length === 0)

// The form is read-only when you can't edit content here: a source-synced item,
// a baseline (historical) view, or you're a viewer/non-member.
const readOnly = computed(() => !!props.milestone?.sourceSystem || !!baselines.activeId || !canEditWorkspace())

// Members who can't edit directly (or want to go through review) can PROPOSE a
// change to an existing item; the owner approves it. Not for synced/baseline items.
const proposing = ref(!!props.proposeMode)
const proposeNote = ref('')
const canPropose = computed(() =>
  (props.mode === 'edit' || props.mode === 'add') &&
  !baselines.activeId && !props.milestone?.sourceSystem && canProposeChanges())
// Previewing a historical version: the whole form shows that snapshot, read-only,
// until "Back to latest". null = the live/head version.
const viewVersion = ref(null)
const headVersion = computed(() => props.milestone?.version || 1)
// Read vs edit. The pop-up (timeline) always opens straight in edit; the embedded
// (Explorer) view opens in READ and flips to edit via the pencil. Same layout in
// both modes — only the fields switch between display and input.
const editing = ref(!props.embedded)
const editable = computed(() => proposing.value || (editing.value && !readOnly.value && viewVersion.value == null))
// Effective lock for the form fields (read-mode = display, not inputs).
const formLocked = computed(() => !editable.value)

// Marker shapes offered in the picker = the active legend markers (+ the item's
// own marker if it was removed from the active set, so it stays selectable).
const defaultDate = props.date || `${props.year}-${String(props.month).padStart(2,'0')}-01`

function addDays(dateStr, n) {
  if (!dateStr) return ''
  const [y, m, d] = dateStr.split('-').map(Number)
  const dt = new Date(y, m - 1, d + n)
  const mm = String(dt.getMonth() + 1).padStart(2, '0')
  const dd = String(dt.getDate()).padStart(2, '0')
  return `${dt.getFullYear()}-${mm}-${dd}`
}

const displayMonth = computed(() => {
  const base = form.kind === 'event' ? form.startDate : form.when
  if (!base) return `${MONTHS[props.month - 1]} ${props.year}`
  const [y, m] = base.split('-').map(Number)
  return `${MONTHS[m - 1]} ${y}`
})

const form = reactive({
  swimlaneId: props.milestone?.swimlaneId ?? (props.swimlane?.id || ''),
  subLaneId: props.milestone?.subLaneId ?? (props.subLane?.id || ''),
  title:  props.milestone?.title ?? '',
  kind:   props.milestone?.kind ?? 'milestone',
  typeKey: props.milestone?.typeKey ?? (props.initialType || props.milestone?.kind || 'milestone'),
  // Description fields (what/why/how) are ordinary type fields now → they live in
  // data. Mirrored items keep the markdown-stripped body in data.what.
  data:   (() => { const d = { ...(props.milestone?.data || {}) }; if (props.milestone?.sourceSystem && typeof d.what === 'string') d.what = stripMarkdown(d.what); return d })(),
  assigneeId: props.milestone?.assigneeId ?? null,
  status: props.milestone?.status ?? '',
  when:   props.milestone?.when ?? defaultDate,
  startDate: props.milestone?.startDate ?? defaultDate,
  endDate:   props.milestone?.endDate ?? addDays(defaultDate, 7),
  color:  props.milestone?.color ?? null,
  maturity: props.milestone?.maturity ?? null,
  progress: props.milestone?.progress ?? null,
  scmUrl: props.milestone?.scmUrl ?? '',
})

// Load a snapshot (a history version, or the live item) into the form fields so the
// whole item reflects it. Used by the History tab's version preview.
function applyToForm(src) {
  form.swimlaneId = src?.swimlaneId ?? ''
  form.subLaneId = src?.subLaneId ?? null
  form.title = src?.title ?? ''
  form.kind = src?.kind ?? 'milestone'
  form.typeKey = src?.typeKey ?? src?.kind ?? 'milestone'
  form.data = { ...(src?.data || {}) }
  form.assigneeId = src?.assigneeId ?? null
  form.status = src?.status ?? ''
  form.when = src?.when ?? ''
  form.startDate = src?.startDate ?? ''
  form.endDate = src?.endDate ?? ''
  form.color = src?.color ?? null
  form.maturity = src?.maturity ?? null
  form.progress = src?.progress ?? null
  form.scmUrl = src?.scmUrl ?? ''
}
function onSelectVersion(version, snapshot) {
  if (version === headVersion.value || !snapshot) { backToLatest(); return }
  viewVersion.value = version
  applyToForm(snapshot)
  // Reflect the viewed version in the URL + shared nav state so the link carries it
  // and a fresh open lands right back here (handled by the watch below).
  if (props.embedded && props.milestone) { ui.explorerItemVersion = version; pushNav({ view: 'explorer', item: props.milestone.id, version }) }
}
function backToLatest() {
  viewVersion.value = null
  applyToForm(props.milestone)
  if (props.embedded && props.milestone) { ui.explorerItemVersion = null; pushNav({ view: 'explorer', item: props.milestone.id }) }
}
// Fetch a revision's snapshot and show it (used for URL / back-forward driven views).
async function loadVersion(version) {
  if (!props.milestone || !version || version >= headVersion.value) { viewVersion.value = null; applyToForm(props.milestone); return }
  try {
    const rev = await api.getRevision(props.milestone.id, version)
    const snap = typeof rev.snapshot === 'string' ? JSON.parse(rev.snapshot) : rev.snapshot
    viewVersion.value = version
    applyToForm(snap)
  } catch { /* ignore */ }
}
// The Explorer's version (from the URL / back-forward) drives which snapshot shows.
// Clicks set it themselves (guarded here to avoid a redundant re-fetch).
watch(() => ui.explorerItemVersion, (v) => {
  if (!props.embedded) return
  if ((v || null) === (viewVersion.value || null)) return
  if (v && v < headVersion.value) loadVersion(v)
  else { viewVersion.value = null; applyToForm(props.milestone) }
}, { immediate: true })

// Reference fields can pin a target to a specific version, stored as "id@vN".
// Keep form.data on bare ids (so the pickers bind cleanly) and track pinned
// versions separately in refPins; both are re-encoded on save. Decode any pins
// already stored so editing an item preserves them.
const refPins = reactive({}) // { [fieldKey]: { [id]: version } }
for (const f of (itemTypeByKey(form.typeKey)?.fields || [])) {
  if (f.type !== 'reference') continue
  refPins[f.key] = {} // always present so the version <select>'s v-model path is safe
  const raw = form.data[f.key]
  const entries = Array.isArray(raw) ? raw : (raw ? [raw] : [])
  const bare = []
  for (const e of entries) {
    const { id, version } = parseRef(e)
    if (!id) continue
    bare.push(id)
    if (version) refPins[f.key][id] = version
  }
  form.data[f.key] = f.refMulti ? bare : (bare[0] || '')
}

// Lanes you can place a new item / proposal in (mirrored Git lanes excluded).
const timelineLanes = computed(() => store.swimlanes.filter(s => !s.sourceSystem))
const chosenLaneSubs = computed(() => store.swimlanes.find(s => s.id === form.swimlaneId)?.subLanes || [])
// Changing the Area clears the sub-area selection (it belonged to the old lane).
watch(() => form.swimlaneId, () => { form.subLaneId = '' })
// Clear a required field's red frame as soon as it's filled in.
watch(() => form.data, () => {
  if (invalidFields.value.length) invalidFields.value = invalidFields.value.filter(k => isFieldEmpty(form.data[k]))
}, { deep: true })

// Type-specific field schema for the selected type.
const currentType = computed(() => itemTypeByKey(form.typeKey))
// timeline-family types sit on a lane/date; work-item & container types don't.
const isTimelineType = computed(() => {
  const f = currentType.value?.family
  return !f || f === 'timeline-point' || f === 'timeline-range'
})
const currentTypeFields = computed(() => currentType.value?.fields || [])
const currentTypeLabel = computed(() => currentType.value?.label || 'Type')

// Exclusive-resource config (#128) — only meaningful for backlog items. Stored in
// item.data._exclusive; injected into the save payload by encodedData().
const EXCL_MODES = [
  { key: 'off',   label: 'Off',   hint: 'No limit — any number of timeline items can use this at the same time.' },
  { key: 'warn',  label: 'Warn',  hint: 'Overlapping bookings are flagged with a warning, but still allowed.' },
  { key: 'block', label: 'Block', hint: 'Overlapping bookings are rejected — only one timeline item can use this at a time.' },
]
const exclusive = reactive({
  mode:   props.milestone?.data?._exclusive?.mode || 'off',
  before: props.milestone?.data?._exclusive?.before ?? 0,
  after:  props.milestone?.data?._exclusive?.after ?? 0,
})
const exclusiveSummary = computed(() => {
  if (exclusive.mode === 'off') return '—'
  const label = exclusive.mode === 'block' ? 'Block overlaps' : 'Warn on overlaps'
  const b = Math.max(0, +exclusive.before || 0), a = Math.max(0, +exclusive.after || 0)
  return (b || a) ? `${label} · buffer ${b}d before / ${a}d after` : label
})
const typeStatuses = computed(() => currentType.value?.statuses || [])
const currentStatusTone = computed(() => (typeStatuses.value.find(s => s.key === form.status)?.tone) || 'neutral')
const currentStatusColor = computed(() => statusColor(typeStatuses.value.find(s => s.key === form.status)))
const currentStatusLabel = computed(() => typeStatuses.value.find(s => s.key === form.status)?.label || form.status || 'Set status')
// Advancing from the status flow updates the form and — for a live item — persists
// the status right away, so a status change sticks without a manual save. (When
// adding or proposing, the item isn't live yet, so it's just staged.)
function onFlowAdvance(key) {
  // Status is a live quick-action — allowed even in read mode (auto-saved), as long
  // as the item isn't truly read-only (viewer / baseline / version preview).
  if (readOnly.value || viewVersion.value != null) return
  form.status = key
  if (props.mode === 'edit' && props.milestone && !proposing.value) updateMilestone(props.milestone.id, { status: key })
}

// ── Read-mode displays (pills, groups) + Copy link / JSON / YAML export ───────
// These mirror the old ItemDetail read view so the same layout serves both modes.
function laneNameOf(id) { return store.swimlanes.find(s => s.id === id)?.name || '' }
function dateStrOf(it) {
  if (!it) return ''
  if (it.startDate && it.endDate) return `${it.startDate} → ${it.endDate}`
  return it.when || (it.year ? `${MONTHS[(it.month || 1) - 1]} ${it.year}` : '')
}
function initials(id) { return memberInitials(id) }
function itemPill(id) {
  const target = store.milestones.find(m => m.id === id)
  const t = target ? itemTypeByKey(target.typeKey || target.kind || 'milestone') : null
  const st = target ? itemStatus(target) : null
  const lane = target ? store.swimlanes.find(s => s.id === target.swimlaneId)?.color : null
  return {
    title: target?.title || id, exists: !!target, head: target?.version,
    icon: t?.icon || 'l:Diamond', fill: t?.fill !== false, color: t?.color || '#8a8a8e',
    dot: st ? statusColor(st) : (lane || '#8a8a8e'),
  }
}
function refDisplay(entry) {
  const { id, version } = parseRef(entry)
  const target = store.milestones.find(m => m.id === id)
  const title = target?.title || id
  if (!version) return title
  const head = target?.version
  return head && head > version ? `${title} · v${version} (latest v${head})` : `${title} · v${version}`
}
function openRef(r) {
  if (!r || r.exists === false || !r.id) return
  ui.explorerItemId = r.id
  ui.explorerItemVersion = r.version || null
  pushNav({ view: 'explorer', item: r.id, version: r.version || null })
}
// Field values as read-display: reference fields become clickable pills, the rest text.
const readFieldRows = computed(() => (currentType.value?.fields || []).map(f => {
  if (f.type !== 'reference') {
    const v = form.data?.[f.key]
    return { key: f.key, label: f.label || f.key, prose: f.type === 'textarea', v: Array.isArray(v) ? v.join(', ') : (v == null ? '' : String(v)) }
  }
  const v = form.data?.[f.key]
  const ids = Array.isArray(v) ? v : (v ? [v] : [])
  return { key: f.key, label: f.label || f.key, refs: ids.map(entry => { const { id, version } = parseRef(entry); return { id, version, ...itemPill(id) } }) }
}))
// All links touching this item, grouped by relationship (read-mode Dependencies).
const readDependencyGroups = computed(() => {
  const id = props.milestone?.id
  if (!id) return []
  const byId = new Map(store.milestones.map(m => [m.id, m]))
  const relLabel = (rel, fwd) => { const r = RELATIONSHIP_TYPES.find(x => x.key === rel); return fwd ? (r?.label || rel) : (r?.inverse || rel) }
  const out = []
  for (const l of store.links) {
    const relKey = l.rel || 'depends-on'
    if (l.a === id && byId.has(l.b)) out.push({ relKey, rel: relLabel(l.rel, true), id: l.b, version: l.version ?? null, ...itemPill(l.b) })
    else if (l.b === id && byId.has(l.a)) out.push({ relKey, rel: relLabel(l.rel, false), id: l.a, version: l.version ?? null, ...itemPill(l.a) })
  }
  const m = new Map()
  for (const d of out) { if (!m.has(d.rel)) m.set(d.rel, { rel: d.rel, relKey: d.relKey, items: [] }); m.get(d.rel).items.push(d) }
  return [...m.values()]
})
const readDeps = computed(() => readDependencyGroups.value.filter(g => g.relKey !== 'uses'))
const readUses = computed(() => readDependencyGroups.value.filter(g => g.relKey === 'uses'))
// When viewing an exclusive resource, flag the users that over-book it (#128).
function pillConflict(itemId) {
  const rid = props.milestone?.id
  if (!rid) return ''
  const list = (resourceConflicts.value[itemId] || []).filter(c => c.resourceId === rid)
  if (!list.length) return ''
  return list.map(c => `Overlaps “${c.otherTitle}” (${c.when})`).join('\n')
}
const readItemGroups = computed(() => groups.list.filter(g => (g.itemIds || []).includes(props.milestone?.id)))

const rawStatusLabel = computed(() => typeStatuses.value.find(s => s.key === (props.milestone?.status))?.label || props.milestone?.status || '')
// JSON/YAML view mode: the fields view (form), or the item rendered as JSON / YAML.
// Reflected in the URL (?fmt=) so a shared link opens the same view.
const viewFormat = ref('form') // 'form' | 'json' | 'yaml'
function setFormat(f) {
  viewFormat.value = f
  if (props.embedded && props.milestone) pushNav({ view: 'explorer', item: props.milestone.id, version: viewVersion.value || null, fmt: f })
}
const linkUrl = computed(() => itemLink(props.milestone?.id, viewVersion.value || null, viewFormat.value))
const formattedText = computed(() =>
  viewFormat.value === 'json' ? JSON.stringify(exportObj.value, null, 2)
  : viewFormat.value === 'yaml' ? toYaml(exportObj.value) : '')
function exportFieldValue(f) {
  const v = props.milestone?.data?.[f.key]
  if (f.type === 'reference') {
    const ids = Array.isArray(v) ? v : (v ? [v] : [])
    const names = ids.map(refDisplay)
    return f.refMulti ? names : (names[0] || '')
  }
  return Array.isArray(v) ? v : (v == null ? '' : v)
}
const exportObj = computed(() => {
  const it = props.milestone || {}
  const t = currentType.value
  const o = { id: it.id, type: t?.label || it.typeKey || it.kind || 'item', title: it.title || '', status: rawStatusLabel.value, version: viewVersion.value || it.version || 1 }
  const area = laneNameOf(it.swimlaneId); if (area) o.area = area
  const d = dateStrOf(it); if (d) o.date = d
  if (it.maturity) o.maturity = MATURITY_STAGES[it.maturity - 1]
  if (it.progress != null) o.progress = it.progress
  if (it.assigneeId) o.assignee = memberName(it.assigneeId) || it.assigneeId
  const fields = {}
  for (const f of (t?.fields || [])) fields[f.label || f.key] = exportFieldValue(f)
  if (Object.keys(fields).length) o.fields = fields
  return o
})
function yamlScalar(v) {
  if (v == null) return '""'
  if (typeof v === 'number' || typeof v === 'boolean') return String(v)
  const s = String(v)
  const needsQuote = s === '' || /^[\s>|@`"'#%&*!?[\]{},-]/.test(s) || /:\s|\s#|[\n:]/.test(s) || /\s$/.test(s) || /^(true|false|null|yes|no|~)$/i.test(s) || /^-?\d/.test(s)
  return needsQuote ? JSON.stringify(s) : s
}
function toYaml(obj, indent = 0) {
  const pad = '  '.repeat(indent); const lines = []
  for (const [k, v] of Object.entries(obj)) {
    if (Array.isArray(v)) {
      if (!v.length) { lines.push(`${pad}${k}: []`); continue }
      lines.push(`${pad}${k}:`); for (const el of v) lines.push(`${pad}  - ${yamlScalar(el)}`)
    } else if (v && typeof v === 'object') {
      const entries = Object.entries(v)
      if (!entries.length) { lines.push(`${pad}${k}: {}`); continue }
      lines.push(`${pad}${k}:`); lines.push(toYaml(v, indent + 1))
    } else lines.push(`${pad}${k}: ${yamlScalar(v)}`)
  }
  return lines.join('\n')
}
const copied = ref('')
let copiedTimer = null
async function copy(kind) {
  const text = kind === 'link' ? linkUrl.value : kind === 'json' ? JSON.stringify(exportObj.value, null, 2) : toYaml(exportObj.value)
  try { await navigator.clipboard.writeText(text) }
  catch {
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

// Switching the item type derives its rendering kind, and (for custom types)
// seeds the marker/colour and any new field slots.
function applyType(key) {
  const t = itemTypeByKey(key)
  form.typeKey = key
  if (!t) return
  if (t.builtin) {
    form.kind = key === 'event' ? 'event' : key === 'point' ? 'point' : 'milestone'
  } else {
    form.kind = t.family === 'timeline-range' ? 'event' : 'milestone'
    if (t.color) form.color = t.color
  }
  // Off-timeline types (backlog / folder) never carry a lane.
  if (t.family === 'work-item' || t.family === 'container') { form.swimlaneId = ''; form.subLaneId = '' }
  for (const f of (t.fields || [])) {
    if (!(f.key in form.data)) form.data[f.key] = (f.type === 'multiselect' || (f.type === 'reference' && f.refMulti)) ? [] : ''
    if (f.type === 'reference' && !refPins[f.key]) refPins[f.key] = {}
  }
  // Default to the type's start status if the current one isn't valid for this type.
  const sts = t.statuses || []
  if (!sts.length) form.status = ''
  else if (!sts.some(s => s.key === form.status)) form.status = sts[0].key
}

// Items a reference field can point at: every item of the field's target type,
// except this item itself.
function refItems(f) {
  if (!f.refType) return []
  return store.milestones.filter(m => (m.typeKey || m.kind) === f.refType && m.id !== props.milestone?.id)
}
function refTypeLabel(key) { return itemTypeByKey(key)?.label || key || 'referenced' }

// Multi-reference picker: a searchable, capped dropdown instead of one checkbox
// per candidate — so it stays usable with hundreds of referenceable items.
const refSearch = reactive({}) // { [fieldKey]: query }
const refOpen = ref('')        // fieldKey whose dropdown is open
const REF_LIST_CAP = 50
function isSelected(f, id) {
  const v = form.data[f.key]
  return Array.isArray(v) ? v.includes(id) : v === id
}
function matchedRefItems(f) {
  const q = (refSearch[f.key] || '').trim().toLowerCase()
  const items = refItems(f)
  return q ? items.filter(r => (r.title || '').toLowerCase().includes(q)) : items
}
function filteredRefItems(f) { return matchedRefItems(f).slice(0, REF_LIST_CAP) }
function moreCount(f) { return Math.max(0, matchedRefItems(f).length - REF_LIST_CAP) }
function onComboBlur() { setTimeout(() => { refOpen.value = '' }, 120) }
// Why a reference field shows no options: either it has no target type configured
// (a mis-set field) or that type simply has no items yet.
function refHint(f) {
  if (!f.refType) return 'This reference field has no target type set — fix it in Manage → Item types.'
  return `No ${refTypeLabel(f.refType)} items to reference yet.`
}

// Version pinning for reference fields (see refPins above).
function selectedRefs(f) {
  const v = form.data[f.key]
  return Array.isArray(v) ? v.filter(Boolean) : (v ? [v] : [])
}
function refTitle(id) { return store.milestones.find(m => m.id === id)?.title || id }
function isPinned(key, id) { return !!(refPins[key] && refPins[key][id]) }
function pinnedVer(key, id) { return refPins[key] && refPins[key][id] }
// A referenced item's current (head) version, and the full list of pickable
// versions (head → 1) so you can pin to any past revision, not just the latest.
function refHead(id) { return store.milestones.find(m => m.id === id)?.version || 1 }
function refVersions(id) {
  const out = []
  for (let v = refHead(id); v >= 1; v--) out.push(v)
  return out
}
// Re-encode reference values with their pinned versions for the save payload.
function encodedData() {
  const out = { ...form.data }
  for (const f of (itemTypeByKey(form.typeKey)?.fields || [])) {
    if (f.type !== 'reference') continue
    const pins = refPins[f.key] || {}
    const enc = (id) => (id && pins[id] ? `${id}@v${pins[id]}` : id)
    const v = out[f.key]
    out[f.key] = Array.isArray(v) ? v.map(enc) : enc(v)
  }
  // Exclusive-resource config (#128) — backlog items only.
  if (!isTimelineType.value && exclusive.mode && exclusive.mode !== 'off') {
    out._exclusive = { mode: exclusive.mode, before: Math.max(0, +exclusive.before || 0), after: Math.max(0, +exclusive.after || 0) }
  } else {
    delete out._exclusive
  }
  return out
}

// Toggle one option of a multi-select field on/off.
function toggleMulti(key, opt, checked) {
  const arr = Array.isArray(form.data[key]) ? [...form.data[key]] : []
  const i = arr.indexOf(opt)
  if (checked && i === -1) arr.push(opt)
  else if (!checked && i !== -1) arr.splice(i, 1)
  form.data[key] = arr
}

// Keep an event's end date on/after its start so the picker opens in the right
// month instead of defaulting to today/a past date.
watch(() => form.startDate, (s) => {
  if (form.kind !== 'event' || !s) return
  if (!form.endDate || form.endDate < s) form.endDate = addDays(s, 7)
})

const dateError = computed(() => {
  if (form.kind === 'event' && form.startDate && form.endDate && form.endDate < form.startDate) {
    return 'End date must be on or after the start date'
  }
  return ''
})

// Typed relationships (R1): one relationship kind is edited at a time. `edges`
// is the working copy of every link touching this item; the pickers operate on
// the selected rel and the diff is applied on save (preserving cancel).
const relType = ref('depends-on')
const relDef = computed(() => RELATIONSHIP_TYPES.find(r => r.key === relType.value) || RELATIONSHIP_TYPES[0])
// Scheduling relations (Blocked by / Blocks) only make sense between timeline
// items. When this item is off-timeline (backlog/container) they're not offered;
// cross-family links use traceability relations (Uses / Used by, Implements, …).
const currentSchedulable = computed(() => {
  const fam = itemTypeByKey(form.typeKey)?.family
  return fam !== 'work-item' && fam !== 'container'
})
// 'uses' has its own dedicated tab (backlog-only) — keep it out of the general
// relationship dropdown so there's a single home for it.
const availableRelTypes = computed(() => RELATIONSHIP_TYPES.filter(r => r.key !== 'uses' && (!r.scheduling || currentSchedulable.value)))
watch(availableRelTypes, (types) => {
  if (!types.some(r => r.key === relType.value)) relType.value = types[0]?.key || 'relates-to'
}, { immediate: true })
// Backlog/container items don't sit in the timeline: the Dependencies (scheduling)
// and Groups (timeline legend/highlight) tabs are hidden for them — their
// relationships live in the Uses tab.
watch(currentSchedulable, (ok) => { if (!ok && (tab.value === 'deps' || tab.value === 'groups')) tab.value = 'flow' }, { immediate: true })
const SELF = props.milestone?.id || '__NEW__'
const originalEdges = (props.milestone ? store.links.filter(l => l.a === SELF || l.b === SELF) : [])
  .map(l => ({ a: l.a, b: l.b, rel: l.rel || 'depends-on', version: l.version ?? null }))
const edges = ref(originalEdges.map(e => ({ ...e })))

// Pinned versions for outgoing "uses" links (this item uses X at version N).
// Kept separate from edge membership so the version <select> can v-model it.
const usesPins = reactive({}) // { [usedItemId]: version }
for (const l of originalEdges) if (l.rel === 'uses' && l.a === SELF && l.version != null) usesPins[l.b] = l.version

// Sets for the SELECTED relationship — drive the two pickers.
const localLinkedIds = computed(() => new Set(edges.value.filter(e => e.a === SELF && e.rel === relType.value).map(e => e.b)))
const localDependentIds = computed(() => new Set(edges.value.filter(e => e.b === SELF && e.rel === relType.value).map(e => e.a)))

// The two directions of a relationship are mutually exclusive for one pair
// (prevents A↔A cycles within the same rel).
function toggleLocalLink(id) {
  const rel = relType.value
  const had = edges.value.some(e => e.a === SELF && e.b === id && e.rel === rel)
  edges.value = edges.value.filter(e => !(e.rel === rel && ((e.a === SELF && e.b === id) || (e.a === id && e.b === SELF))))
  if (!had) edges.value.push({ a: SELF, b: id, rel })
}
function toggleLocalDependent(id) {
  const rel = relType.value
  const had = edges.value.some(e => e.a === id && e.b === SELF && e.rel === rel)
  edges.value = edges.value.filter(e => !(e.rel === rel && ((e.a === id && e.b === SELF) || (e.a === SELF && e.b === id))))
  if (!had) edges.value.push({ a: id, b: SELF, rel })
}

// Group membership (applied on save).
const localGroupIds = ref(new Set(props.milestone ? itemGroupIds(props.milestone.id) : []))
function toggleLocalGroup(id) {
  const next = new Set(localGroupIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  localGroupIds.value = next
}

function activePickerStyle(color) {
  if (!color) return {}
  const r = parseInt(color.slice(1, 3), 16)
  const g = parseInt(color.slice(3, 5), 16)
  const b = parseInt(color.slice(5, 7), 16)
  return {
    borderLeft: `2px solid rgba(${r},${g},${b},0.5)`,
  }
}

// Milestone picker search + grouping
const pickerSearch = ref('')
const pickerSearch2 = ref('')

function buildPickerGroups(query, onlyIds, backlogOnly = false) {
  const q = (query || '').toLowerCase()
  const groups = []
  // Uses tab: off-timeline (backlog) items only, in one flat "Backlog" group.
  if (backlogOnly) {
    const backlog = store.milestones.filter(m => {
      if (m.id === props.milestone?.id) return false
      if (m.swimlaneId || m.sourceSystem) return false
      if (onlyIds && !onlyIds.has(m.id)) return false
      if (q && !m.title.toLowerCase().includes(q)) return false
      return true
    })
    if (backlog.length) groups.push({ swimlane: { id: '__backlog__', name: 'Backlog', color: '#8E8E93' }, subLane: null, milestones: backlog })
    return groups
  }
  for (const sw of store.swimlanes) {
    const subs = sw.subLanes.length ? sw.subLanes : [null]
    for (const sub of subs) {
      const mils = store.milestones.filter(m => {
        if (m.id === props.milestone?.id) return false
        if (m.swimlaneId !== sw.id) return false
        if (m.subLaneId !== (sub?.id ?? null)) return false
        if (onlyIds && !onlyIds.has(m.id)) return false
        if (q && !m.title.toLowerCase().includes(q)) return false
        return true
      })
      if (mils.length) groups.push({ swimlane: sw, subLane: sub, milestones: mils })
    }
  }
  return groups
}
// Clicking the count badge filters the picker to only the selected items.
const showOnly1 = ref(false)
const showOnly2 = ref(false)
const pickerGroups = computed(() => buildPickerGroups(pickerSearch.value, showOnly1.value && localLinkedIds.value.size ? localLinkedIds.value : null))
const pickerGroups2 = computed(() => buildPickerGroups(pickerSearch2.value, showOnly2.value && localDependentIds.value.size ? localDependentIds.value : null))

// ── Uses tab (#86): reference Backlog items only. Same edge store, rel='uses',
// with a backlog-only picker. Two directions: "Uses" (this → backlog) and
// "Used by" (backlog → this).
const localUsesIds = computed(() => new Set(edges.value.filter(e => e.a === SELF && e.rel === 'uses').map(e => e.b)))
const localUsedByIds = computed(() => new Set(edges.value.filter(e => e.b === SELF && e.rel === 'uses').map(e => e.a)))
function toggleUses(id) {
  const had = edges.value.some(e => e.a === SELF && e.b === id && e.rel === 'uses')
  edges.value = edges.value.filter(e => !(e.rel === 'uses' && ((e.a === SELF && e.b === id) || (e.a === id && e.b === SELF))))
  if (!had) edges.value.push({ a: SELF, b: id, rel: 'uses' })
  else delete usesPins[id]
}
function toggleUsedBy(id) {
  const had = edges.value.some(e => e.a === id && e.b === SELF && e.rel === 'uses')
  edges.value = edges.value.filter(e => !(e.rel === 'uses' && ((e.a === id && e.b === SELF) || (e.a === SELF && e.b === id))))
  if (!had) edges.value.push({ a: id, b: SELF, rel: 'uses' })
}
function usesDot(m) { return itemTypeByKey(m.typeKey || m.kind)?.color || '#8a8a8e' }
function usesTypeLabel(m) { return itemTypeByKey(m.typeKey || m.kind)?.label || (m.typeKey || m.kind || '') }
const usesSearch = ref(''); const usesSearch2 = ref('')
const showOnlyU1 = ref(false); const showOnlyU2 = ref(false)
const usesGroups = computed(() => buildPickerGroups(usesSearch.value, showOnlyU1.value && localUsesIds.value.size ? localUsesIds.value : null, true))
const usesGroups2 = computed(() => buildPickerGroups(usesSearch2.value, showOnlyU2.value && localUsedByIds.value.size ? localUsedByIds.value : null, true))

const titleInput = ref(null)
onMounted(() => {
  // Preselect the Explorer's chosen type (sets kind / marker / colour / fields).
  if (props.mode === 'add' && props.initialType) applyType(props.initialType)
  // A status-typed item always has a status — default to the start if unset.
  if (typeStatuses.value.length && !typeStatuses.value.some(s => s.key === form.status)) {
    form.status = typeStatuses.value[0].key
  }
  // Open straight in the JSON/YAML view when the shared link asked for it.
  if (props.embedded && typeof window !== 'undefined') {
    const f = new URLSearchParams(window.location.search).get('fmt')
    if (f === 'json' || f === 'yaml') viewFormat.value = f
  }
  titleInput.value?.focus()
})

// Resolve the placeholder self-id to a real item id. Outgoing "uses" links carry
// a pinned version. Shared by syncLinks (live save) and the change-request payload.
function resolveLinks(msId) {
  return edges.value.map(e => ({
    a: e.a === SELF ? msId : e.a,
    b: e.b === SELF ? msId : e.b,
    rel: e.rel,
    version: (e.rel === 'uses' && e.a === SELF && usesPins[e.b]) ? Number(usesPins[e.b]) : null,
  }))
}

function syncLinks(msId) {
  // Diff the resolved working edges against the originals (keyed a|b|rel).
  const key = (e) => `${e.a}|${e.b}|${e.rel}`
  const want = new Map(resolveLinks(msId).map(e => [key(e), e]))
  const orig = new Map(originalEdges.map(e => [key(e), e]))
  for (const [k, e] of want) {
    const o = orig.get(k)
    if (!o || (o.version ?? null) !== (e.version ?? null)) addLink(e.a, e.b, e.rel, e.version) // new link or version change → upsert
  }
  for (const [k, e] of orig) if (!want.has(k)) removeLink(e.a, e.b, e.rel)
}

const justSaved = ref(false)
let savedTimer = null
function flashSaved() { justSaved.value = true; clearTimeout(savedTimer); savedTimer = setTimeout(() => { justSaved.value = false }, 1600) }

// Block-mode conflicts this save would introduce (#128). Only timeline items book
// resources; the candidate window uses the item's proposed dates and its current
// (possibly just-edited) set of "uses" links.
function blockingConflicts(id, payload) {
  if (!isTimelineType.value) return []
  const start = payload.startDate || payload.when
  const end   = payload.endDate || payload.startDate || payload.when
  if (!start) return []
  const resourceIds = edges.value.filter(e => e.a === SELF && (e.rel || 'depends-on') === 'uses').map(e => e.b)
  if (!resourceIds.length) return []
  return checkResourceConflicts({ id, start, end, resourceIds }).filter(c => c.mode === 'block')
}

function submit(keepOpen = false) {
  if (formLocked.value) return false // view-only and not proposing
  if (dateError.value) return false // the date field (with its error) lives in the always-visible top block
  if (!form.title.trim()) return false

  // Enforce mandatory type fields: frame the empty ones in red instead of a message.
  // (The type fields are always visible in the top block, so no tab switch needed.)
  invalidFields.value = currentTypeFields.value.filter(f => f.required && isFieldEmpty(form.data[f.key])).map(f => f.key)
  if (invalidFields.value.length) return false

  const isEvent = form.kind === 'event'
  // Grid position derives from the start (event) or the date (milestone).
  const base = isEvent ? (form.startDate || form.when) : form.when
  let month = props.month
  let year  = props.year
  if (base) {
    const parts = base.split('-')
    year  = parseInt(parts[0], 10)
    month = parseInt(parts[1], 10)
  }

  const payload = {
    swimlaneId: form.swimlaneId || '', // "" = off-timeline artifact (no lane)
    subLaneId:  form.subLaneId || null,
    year,
    month,
    title:      form.title.trim(),
    kind:       form.kind,
    typeKey:    form.typeKey,
    data:       encodedData(),
    assigneeId: form.assigneeId || null,
    status:     form.status || '',
    marker:     null, // the icon now comes from the item's type, not a per-item marker
    when:       isEvent ? (form.startDate || null) : (form.when || null),
    startDate:  isEvent ? (form.startDate || null) : null,
    endDate:    isEvent ? (form.endDate || null) : null,
    color:      null, // per-item colour removed — icon inherits the area/type colour
    maturity:   form.maturity || null,
    progress:   form.progress,
    scmUrl:     form.scmUrl.trim() || null,
  }
  // Proposing → submit a change request instead of touching the live plan. The
  // payload carries the item's full desired link set so dependencies edited in the
  // proposal are applied when the owner approves it (resolved against the item id).
  if (proposing.value) {
    let done
    if (props.mode === 'add') {
      const nid = crypto.randomUUID()
      done = proposeCreate({ ...payload, id: nid, links: resolveLinks(nid) }, proposeNote.value.trim())
    } else {
      done = proposeChange(props.milestone.id, { ...payload, links: resolveLinks(props.milestone.id) }, proposeNote.value.trim())
    }
    done.catch(e => alert(e?.message || 'Could not submit the proposal'))
    if (!props.embedded) emit('close')
    return true
  }
  // Block-mode exclusive resources (#128): refuse the save if this timeline item's
  // booking would overlap another user of a capacity-1 resource it uses.
  const blocked = blockingConflicts(props.mode === 'edit' ? props.milestone.id : SELF, payload)
  if (blocked.length) {
    const c = blocked[0]
    alert(`${c.resourceTitle} is already booked ${c.when} by “${c.otherTitle}”.\nMove the dates or free the resource before saving.`)
    return false
  }
  if (props.mode === 'edit') {
    updateMilestone(props.milestone.id, payload)
    syncLinks(props.milestone.id)
    setItemGroups(props.milestone.id, [...localGroupIds.value])
    if (keepOpen) { flashSaved(); return true } // saved in place — keep it open for more edits
  } else {
    const newMs = addMilestone(payload)
    syncLinks(newMs.id)
    setItemGroups(newMs.id, [...localGroupIds.value])
  }
  if (!props.embedded) emit('close')
  return true
}

// Header Save: pop-up saves (keeping edit open for edits / closing on create); the
// embedded view saves and flips back to read.
function onSave() {
  if (props.embedded) {
    if (submit(true)) { editing.value = false; proposing.value = false }
  } else {
    submit(props.mode === 'edit' && !proposing.value)
  }
}

function remove() {
  if (!props.milestone) return
  const label = form.title.trim() || props.milestone.title || 'this item'
  if (!confirm(`Delete "${label}"? This can't be undone.`)) return
  deleteMilestone(props.milestone.id)
  emit('close')
}
</script>

<style scoped>
.backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.45);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
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
  max-width: 1120px;
  max-height: 92vh;
  box-shadow: var(--sh-modal);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* Embedded (Explorer pane): fill the container instead of floating as a pop-up. */
.embed-host { flex: 1; min-width: 0; display: flex; flex-direction: column; min-height: 0; }
.panel.embedded { max-width: none; max-height: none; height: 100%; border-radius: 0; box-shadow: none; background: transparent; }
.embed-done { font-size: 13px; font-weight: 600; color: var(--clr-text); background: var(--clr-surface-2); border-radius: var(--r-md); padding: 7px 15px; transition: background 0.15s; }
.embed-done:hover { background: var(--clr-border-light); }

.panel-header {
  padding: 20px 20px 0;
  position: relative;
  flex-shrink: 0;
}

.panel-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
  padding-right: 300px; /* keep clear of the top-right action icons (copy/json/yaml + save) */
}

.panel-badge {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 9px;
  border-radius: 100px;
  color: #fff;
  letter-spacing: 0.2px;
}

.panel-sub { font-size: 12px; color: var(--clr-text-2); font-weight: 500; }
.panel-month { font-size: 12px; color: var(--clr-text-3); }
.panel-attrib { display: flex; flex-wrap: wrap; gap: 3px 18px; padding: 8px 20px 4px; font-size: 12px; color: var(--clr-text-3); }
.panel-attrib strong { color: var(--clr-text-2); font-weight: 600; }
.panel-attrib-inline { margin-left: 6px; font-size: 12px; color: var(--clr-text-3); white-space: nowrap; }
.panel-attrib-inline strong { color: var(--clr-text-2); font-weight: 600; }
.panel-ver { display: inline-flex; align-items: center; gap: 4px; font-size: 11px; font-weight: 700; color: var(--clr-text-2); background: var(--clr-surface-2); border-radius: 100px; padding: 3px 10px; cursor: pointer; transition: background 0.12s, color 0.12s; }
.panel-ver:hover, .panel-ver.on { background: rgba(0,113,227,0.12); color: var(--clr-accent); }

.panel-actions-top {
  position: absolute;
  top: 16px; right: 16px;
  display: flex; gap: 8px;
}
.icon-act {
  width: 32px; height: 32px;
  display: flex; align-items: center; justify-content: center;
  background: var(--clr-surface-2);
  border-radius: 50%;
  color: var(--clr-text-2);
  transition: background 0.15s, color 0.15s;
}
.icon-act:hover { background: var(--clr-border-light); color: var(--clr-text); }
.icon-act.primary { background: var(--clr-accent); color: #fff; }
.icon-act.primary:hover { background: var(--clr-accent-hover); }
.icon-act.saved { background: #30D158; color: #06310f; }
.icon-act.done { background: #30D158; color: #06310f; }
.icon-act.on { background: rgba(0,113,227,0.14); color: var(--clr-accent); }
.propose-act { font-size: 12px; font-weight: 600; color: var(--clr-accent); background: rgba(0,113,227,0.08); border-radius: 100px; padding: 6px 13px; }
.propose-act:hover { background: rgba(0,113,227,0.16); }
.save-act { display: inline-flex; align-items: center; gap: 5px; font-size: 13px; font-weight: 600; color: #fff; background: var(--clr-accent); border-radius: var(--r-md); padding: 7px 15px; transition: background 0.15s; }
.save-act:hover { background: var(--clr-accent-hover); }
.save-act.saved { background: #30D158; color: #06310f; }

.propose-banner { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; padding: 9px 14px; margin: 0 0 4px; background: rgba(255,159,10,0.12); border: 1px solid rgba(255,159,10,0.3); border-radius: var(--r-md); }
.pb-text { font-size: 12px; font-weight: 600; color: #b7791f; }
.pb-note { flex: 1; min-width: 160px; border: 1px solid var(--clr-border); border-radius: var(--r-sm); padding: 6px 9px; font-size: 13px; color: var(--clr-text); background: var(--clr-bg); }
.pb-note:focus { outline: none; border-color: var(--clr-accent); }

.version-banner { display: inline-flex; align-self: flex-start; max-width: calc(100% - 40px); align-items: center; gap: 16px; flex-wrap: wrap; padding: 8px 8px 8px 14px; margin: 0 20px 8px; background: rgba(0,113,227,0.1); border: 1px solid rgba(0,113,227,0.3); border-radius: var(--r-md); }
.vb-text { display: inline-flex; align-items: center; gap: 6px; font-size: 12.5px; font-weight: 600; color: var(--clr-accent); }
.vb-back { font-size: 12px; font-weight: 600; color: #fff; background: var(--clr-accent); border-radius: var(--r-md); padding: 6px 13px; white-space: nowrap; }
.vb-back:hover { background: var(--clr-accent-hover); }
.icon-act.danger { background: rgba(255,59,48,0.1); color: var(--clr-danger); }
.icon-act.danger:hover { background: rgba(255,59,48,0.18); }

.hidden-submit { position: absolute; width: 0; height: 0; padding: 0; margin: 0; border: 0; opacity: 0; pointer-events: none; }

.panel-form {
  padding: 0 20px 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
}

/* JSON / YAML code view — replaces the FIELDS block (tabs/flow/console stay below). */
.ms-code-view { min-height: 260px; max-height: 60vh; overflow: auto; background: var(--clr-bg); border: 1px solid var(--clr-border-light); border-radius: var(--r-md); padding: 12px 14px; }
.ms-code { margin: 0; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12.5px; line-height: 1.6; color: var(--clr-text); white-space: pre-wrap; word-break: break-word; }

/* Always-visible console dock at the very bottom of the modal (teleport target). */
.modal-console-dock { flex-shrink: 0; background: var(--clr-bg); border-top: 1px solid var(--clr-border); }
.modal-console-dock:empty { display: none; }

.two-col { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }

.ms-tabs { display: flex; gap: 2px; border-bottom: 1px solid var(--clr-border-light); margin: 2px 0 0; }
.ms-tab {
  padding: 8px 14px; font-size: 13.5px; font-weight: 600;
  color: var(--clr-text-3); background: none;
  border-bottom: 2px solid transparent; margin-bottom: -1px;
  cursor: pointer; transition: color 0.12s, border-color 0.12s;
}
.ms-tab:hover { color: var(--clr-text-2); }
.ms-tab.active { color: var(--clr-accent); border-bottom-color: var(--clr-accent); }
.ms-tab-body { height: 360px; display: flex; flex-direction: column; } /* fixed so the modal is the same height on every tab */
.ms-flow { padding: 0; }
/* fieldsets are only used to disable the whole form in read-only mode — strip their chrome */
.panel-form fieldset { border: 0; margin: 0; padding: 0; min-width: 0; }
/* Two-column field grid: compact fields pair up, wide ones (Title, description,
   date ranges) span the full width. Wraps gracefully when a field is hidden. */
.ms-group { display: flex; flex-wrap: wrap; gap: 14px 18px; align-items: start; border: 0; padding: 0; margin: 0; min-width: 0; }
.ms-group > .field { flex: 1 1 calc(50% - 9px); min-width: 210px; }
.ms-group > .span2 { flex: 1 1 100%; }
/* The description fields render last (so Sub-area|Type and the other short fields
   pair up cleanly above them) without reordering the DOM. */
.ms-group > .type-fields { order: 1; }
/* read-only view: keep the disabled controls fully legible (no browser dimming) */
.panel-form fieldset:disabled :disabled { opacity: 1; cursor: default; }
.panel-form fieldset:disabled .field-input,
.panel-form fieldset:disabled .field-textarea,
.panel-form fieldset:disabled .field-date {
  color: var(--clr-text); -webkit-text-fill-color: var(--clr-text); background: var(--clr-bg);
}
/* Read-mode: the same fields render as clean display text — no borders, no input
   chrome — so the layout is identical to edit mode, only the affordance changes. */
.panel-form.read-mode .field-input,
.panel-form.read-mode .field-textarea,
.panel-form.read-mode .field-date {
  border-color: transparent; background: transparent; box-shadow: none;
  padding-top: 2px; padding-bottom: 2px; padding-left: 0; padding-right: 0;
  appearance: none; -webkit-appearance: none; font-weight: 500;
}
.panel-form.read-mode .field-input::-webkit-calendar-picker-indicator,
.panel-form.read-mode .field-input::-webkit-inner-spin-button { display: none; }
.panel-form.read-mode .field-label { color: var(--clr-text-3); }
/* Read-mode is compact: label sits to the LEFT of the value (one line per field),
   with tighter gaps, so the fields don't push the Flow below the fold. */
.panel-form.read-mode .ms-group { gap: 8px 30px; }
.panel-form.read-mode .field:not(.type-fields) { flex-direction: row; align-items: baseline; gap: 14px; }
.panel-form.read-mode .field:not(.type-fields) > .field-label { flex: 0 0 104px; margin: 0; padding-top: 2px; }
.panel-form.read-mode .field:not(.type-fields) > .field-input,
.panel-form.read-mode .field:not(.type-fields) > .field-date,
.panel-form.read-mode .field:not(.type-fields) > .read-val { flex: 1 1 auto; width: auto; min-width: 0; }
.panel-form.read-mode .type-fields { gap: 8px; }
.panel-form.read-mode .type-hint { display: none; }
/* Read-mode display bits (maturity/progress value, type-field list, ref pills). */
.read-val { font-size: 14px; font-weight: 500; color: var(--clr-text); padding: 2px 0; }
.read-fields { display: grid; grid-template-columns: 120px 1fr; gap: 6px 14px; margin-top: 4px; }
.read-fields dt { font-size: 12px; font-weight: 600; color: var(--clr-text-3); }
.read-fields dd { font-size: 13.5px; color: var(--clr-text); }
.read-fields dd.read-empty { color: var(--clr-text-3); }
.read-fields dd.read-prose { white-space: pre-wrap; line-height: 1.5; }
.read-refs { display: flex; flex-wrap: wrap; gap: 6px; }
.read-pill { display: inline-flex; align-items: center; gap: 6px; font-size: 12px; font-weight: 600; border-radius: 100px; padding: 4px 11px; cursor: pointer; transition: filter 0.12s; }
.read-pill:hover:not(.missing) { filter: brightness(1.18); }
.read-pill.missing { cursor: default; }
.read-pill.pill-conflict { box-shadow: inset 0 0 0 1.5px #FF3B30; }
.read-pill-dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.read-pill-ver { font-weight: 500; color: var(--clr-text-3); }
.read-deps { list-style: none; display: flex; flex-direction: column; gap: 8px; }
.read-deps li { display: flex; align-items: flex-start; gap: 8px; font-size: 13px; }
.read-dep-rel { flex-shrink: 0; padding-top: 5px; min-width: 96px; font-size: 12px; font-weight: 500; color: var(--clr-text-3); }
.read-groups { display: flex; flex-wrap: wrap; gap: 6px; }
.read-group { font-size: 12px; font-weight: 600; border-radius: 100px; padding: 3px 10px; border: 1px solid currentColor; }
.read-none { font-size: 13px; color: var(--clr-text-3); }
.ro-badge { display: inline-flex; align-items: center; gap: 4px; font-size: 11px; font-weight: 600; color: var(--clr-text-3); }
.ms-panel { display: flex; flex-direction: column; gap: 14px; flex: 1; min-height: 0; overflow-y: auto; }
.scm-hint { font-size: 12.5px; color: var(--clr-text-3); line-height: 1.45; margin-top: -1px; }

/* Dependencies tab: let the two pickers grow to fill the tab height instead of leaving whitespace below */
.uses-hint { font-size: 12px; color: var(--clr-text-3); margin: 0 0 10px; }
.uses-hint strong { color: var(--clr-text-2); font-weight: 700; }
/* Uses picker: version dropdown inline in each selected row. */
.picker-row { display: flex; align-items: center; }
.picker-row .picker-item { flex: 1; min-width: 0; width: auto; }
.picker-ver { flex-shrink: 0; margin-right: 8px; padding: 3px 6px; font-size: 11px; font-variant-numeric: tabular-nums;
  color: var(--clr-text-2); background: var(--clr-bg); border: 1px solid var(--clr-border); border-radius: var(--r-sm); }
.picker-ver:focus { outline: none; border-color: var(--clr-accent); }
.dep-cols { flex: 1; min-height: 0; grid-template-rows: minmax(0, 1fr); }
.dep-cols .field { min-height: 0; }
.dep-cols .ms-picker { display: flex; flex-direction: column; flex: 1; min-height: 0; }
.dep-cols .picker-list { max-height: none; flex: 1; min-height: 0; }

.field { display: flex; flex-direction: column; gap: 5px; min-width: 0; }

.field-label {
  font-size: 11.5px; font-weight: 600;
  color: var(--clr-text-2);
  text-transform: uppercase; letter-spacing: 0.4px;
  display: flex; align-items: center; gap: 6px;
}
.req { color: var(--clr-danger); }

.link-count {
  font-size: 10px; font-weight: 700;
  background: var(--clr-accent);
  color: #fff;
  padding: 1px 6px;
  border-radius: 100px;
  letter-spacing: 0;
}
.link-toggle { cursor: pointer; transition: background 0.12s, color 0.12s, box-shadow 0.12s; }
/* Filter toggle: outline (hollow) while ALL are shown, filled while filtered to the selected ones */
.link-count.link-toggle { background: transparent; color: var(--clr-accent); box-shadow: inset 0 0 0 1.5px var(--clr-accent); }
.link-count.link-toggle.on { background: var(--clr-accent); color: #fff; box-shadow: none; }

.type-fields .tf-row { display: flex; align-items: center; gap: 10px; margin-top: 8px; }
.type-fields .tf-label { font-size: 12px; color: var(--clr-text-2); min-width: 120px; flex-shrink: 0; }
.type-fields .tf-row .field-input { flex: 1; }
.type-fields .tf-req { color: var(--clr-danger); margin-left: 2px; }
.type-fields .tf-checks { flex: 1; display: flex; flex-wrap: wrap; gap: 6px 14px; align-items: center; }
.type-fields .tf-check { display: inline-flex; align-items: center; gap: 4px; font-size: 13px; color: var(--clr-text); }
.type-fields .tf-empty { font-size: 12px; color: var(--clr-text-3); }
.type-fields .field-input.tf-invalid { border-color: var(--clr-danger); box-shadow: 0 0 0 2px rgba(255,59,48,0.18); }
.type-fields .tf-checks.tf-invalid { border: 1px solid var(--clr-danger); border-radius: var(--r-md); padding: 6px 8px; box-shadow: 0 0 0 2px rgba(255,59,48,0.18); }

/* Exclusive-resource settings (#128) */
.excl-edit { display: flex; flex-direction: column; gap: 6px; }
.excl-modes { display: inline-flex; gap: 4px; background: var(--clr-bg-2); border-radius: var(--r-md); padding: 3px; width: max-content; }
.excl-mode { display: inline-flex; align-items: center; gap: 5px; padding: 4px 12px; border-radius: calc(var(--r-md) - 2px); font-size: 13px; font-weight: 600; color: var(--clr-text-2); cursor: pointer; }
.excl-mode input { position: absolute; opacity: 0; width: 0; height: 0; }
.excl-mode.on { background: var(--clr-accent); color: #fff; }
.excl-hint { margin: 0; font-size: 12px; color: var(--clr-text-3); }
.excl-buffer { display: flex; align-items: center; gap: 16px; }
.excl-buf { display: inline-flex; align-items: center; gap: 6px; font-size: 13px; color: var(--clr-text-2); }
.excl-num { width: 64px; }
.type-fields .tf-row { flex-wrap: wrap; } /* lets the version-pin strip drop to its own line */
.type-fields .tf-pins { flex-basis: 100%; display: flex; flex-wrap: wrap; align-items: center; gap: 6px 14px; padding-left: 130px; }
.type-fields .tf-refhint { flex-basis: 100%; padding-left: 130px; font-size: 12px; color: var(--clr-warning, #FF9F0A); }
.type-fields .tf-pins-lbl { font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.3px; color: var(--clr-text-3); }
.type-fields .tf-pinitem { display: inline-flex; align-items: center; gap: 6px; }
.type-fields .tf-pin-name { font-size: 12px; color: var(--clr-text-2); }
.type-fields .tf-pin-sel { padding: 4px 8px; font-size: 12px; color: var(--clr-text); background: var(--clr-bg); border: 1px solid var(--clr-border); border-radius: var(--r-sm); }
.type-fields .tf-pin-sel:focus { outline: none; border-color: var(--clr-accent); }
.type-fields .tf-pin-sel.on { color: var(--clr-warning, #FF9F0A); border-color: var(--clr-warning, #FF9F0A); }
.type-fields .tf-pin-sel:disabled { opacity: 0.6; cursor: default; }
.type-fields .tf-pin-x { width: 22px; height: 22px; flex-shrink: 0; display: inline-flex; align-items: center; justify-content: center; font-size: 15px; line-height: 1; color: var(--clr-text-3); background: none; border-radius: var(--r-sm); }
.type-fields .tf-pin-x:hover { color: var(--clr-danger); background: rgba(255,59,48,0.08); }

/* Searchable multi-reference picker (scales to many candidates). */
.type-fields .tf-multiref { flex: 1; min-width: 0; position: relative; }
.type-fields .tf-multiref .field-input { width: 100%; }
.type-fields .tf-multiref.tf-invalid .field-input { border-color: var(--clr-danger); box-shadow: 0 0 0 2px rgba(255,59,48,0.18); }
.type-fields .tf-combo-list { position: absolute; z-index: 30; top: calc(100% + 4px); left: 0; right: 0; max-height: 220px; overflow-y: auto;
  background: var(--clr-surface, var(--clr-bg)); border: 1px solid var(--clr-border); border-radius: var(--r-md); box-shadow: 0 10px 28px rgba(0,0,0,0.28); padding: 4px; }
.type-fields .tf-combo-opt { display: flex; align-items: center; gap: 8px; width: 100%; text-align: left; padding: 6px 10px; font-size: 13px; color: var(--clr-text); background: none; border-radius: var(--r-sm); }
.type-fields .tf-combo-opt:hover { background: var(--clr-surface-2); }
.type-fields .tf-combo-opt.on { color: var(--clr-accent); font-weight: 600; }
.type-fields .tf-combo-check { width: 12px; flex-shrink: 0; color: var(--clr-accent); }
.type-fields .tf-combo-empty, .type-fields .tf-combo-more { padding: 6px 10px; font-size: 12px; color: var(--clr-text-3); }
/* Status shown as a coloured chip (tint = its status colour); click opens the flow. */
.ms-status-chip {
  display: inline-flex; align-items: center; gap: 8px; align-self: flex-start;
  font-size: 13.5px; font-weight: 600; color: var(--chip, var(--clr-text));
  padding: 8px 12px; border-radius: var(--r-md);
  border: 1px solid color-mix(in srgb, var(--chip, #8a8a8e) 50%, transparent);
  background: color-mix(in srgb, var(--chip, #8a8a8e) 12%, transparent);
  cursor: pointer; transition: background 0.14s, border-color 0.14s;
}
.ms-status-chip:hover { background: color-mix(in srgb, var(--chip, #8a8a8e) 20%, transparent); }
.ms-status-dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.ms-status-ico { flex-shrink: 0; }
.ms-status-lbl { color: var(--clr-text); }
.ms-status-flowico { margin-left: 2px; opacity: 0.7; color: var(--chip); }
/* Compact variant for the header (top of the modal). */
.ms-status-chip-sm { padding: 4px 10px; font-size: 12.5px; border-radius: 100px; }
.ms-status-chip-sm .ms-status-dot { width: 8px; height: 8px; }

.field-input,
.field-textarea {
  border: 1.5px solid var(--clr-border);
  border-radius: var(--r-md);
  padding: 9px 12px;
  font-size: 14px;
  color: var(--clr-text);
  background: var(--clr-bg);
  transition: border-color 0.15s, box-shadow 0.15s;
  resize: none;
  outline: none;
  width: 100%;
}
.field-input:focus,
.field-textarea:focus {
  border-color: var(--clr-accent);
  box-shadow: 0 0 0 3px rgba(0,113,227,0.12);
  background: var(--clr-surface);
}
.field-input::placeholder,
.field-textarea::placeholder { color: var(--clr-text-3); }
/* Description textareas start small and grow with their content (no giant empty
   boxes). field-sizing is progressive — browsers without it fall back to `rows`. */
.field-textarea { field-sizing: content; min-height: 2.6em; max-height: 260px; overflow-y: auto; }

.field-date { cursor: pointer; }

/* Type segmented control + marker picker (P1) */
.seg { display: flex; gap: 0; border: 1.5px solid var(--clr-border); border-radius: var(--r-md); overflow: hidden; }
.seg-btn {
  flex: 1; padding: 8px 10px; font-size: 13px; font-weight: 500;
  color: var(--clr-text-2); background: var(--clr-bg); transition: background 0.12s, color 0.12s;
}
.seg-btn + .seg-btn { border-left: 1.5px solid var(--clr-border); }
.seg-btn.on { background: var(--clr-accent); color: #fff; }

.type-row { display: flex; align-items: center; gap: 9px; }
.type-ico { flex-shrink: 0; display: inline-flex; }
/* Fixed type: plain read-only text — no field box, no chevron, so nobody mistakes
   it for a control (it can't change once the item exists). */
.type-static { font-size: 14px; font-weight: 600; color: var(--clr-text); cursor: default; user-select: none; }
.type-row .field-input { flex: 1; }
.type-hint { font-size: 11px; color: var(--clr-text-3); margin-top: 5px; }
.marker-row { display: flex; gap: 6px; }
.marker-btn {
  width: 34px; height: 34px;
  display: flex; align-items: center; justify-content: center;
  border: 1.5px solid var(--clr-border); border-radius: var(--r-md);
  background: var(--clr-bg); cursor: pointer; transition: border-color 0.12s, background 0.12s;
}
.marker-btn:hover { background: var(--clr-surface-2); }
.marker-btn.on { border-color: var(--clr-accent); box-shadow: 0 0 0 3px rgba(0,113,227,0.12); }

.color-row { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
.color-swatch {
  width: 22px; height: 22px; border-radius: 6px; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
  border: 2px solid transparent; cursor: pointer;
  color: #fff; font-size: 11px; font-weight: 700;
}
.color-swatch.selected { border-color: var(--clr-text); }
.color-custom {
  width: 30px; height: 24px; padding: 0; border: 1.5px solid var(--clr-border);
  border-radius: 6px; background: none; cursor: pointer;
}

.maturity-row { display: flex; flex-wrap: wrap; gap: 6px; }
.maturity-btn {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 6px 10px; border-radius: var(--r-md);
  border: 1.5px solid var(--clr-border); background: var(--clr-surface);
  font-size: 12.5px; color: var(--clr-text-2); cursor: pointer;
  transition: border-color 0.12s, background 0.12s, color 0.12s;
}
.maturity-btn.on { border-color: var(--clr-accent); color: var(--clr-text); background: rgba(0,113,227,0.06); }
.maturity-lbl { white-space: nowrap; }
.mat-current { font-size: 11px; font-weight: 600; color: var(--clr-accent); padding-left: 6px; }
.progress-row { display: flex; align-items: center; gap: 12px; }
.progress-slider { flex: 1; }

.field-error { font-size: 12.5px; color: var(--clr-danger); margin: -6px 0 0; }

/* ── Milestone Picker ───────────────────────────────────────────────── */
.ms-picker {
  border: 1.5px solid var(--clr-border);
  border-radius: var(--r-md);
  overflow: hidden;
  background: var(--clr-bg);
}

.picker-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--clr-border-light);
}

.search-icon { color: var(--clr-text-3); flex-shrink: 0; }

.picker-input {
  flex: 1;
  border: none;
  background: transparent;
  outline: none;
  font-size: 13px;
  color: var(--clr-text);
  min-width: 0;
}
.picker-input::placeholder { color: var(--clr-text-3); }

.picker-clear {
  font-size: 16px;
  color: var(--clr-text-3);
  line-height: 1;
  padding: 0 2px;
  transition: color 0.1s;
}
.picker-clear:hover { color: var(--clr-text); }

.picker-list {
  max-height: 210px;
  overflow-y: auto;
}

.picker-group-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 12px 4px;
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--clr-text-3);
  position: sticky;
  top: 0;
  background: var(--clr-bg);
  z-index: 1;
}

.picker-group-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.picker-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 7px 12px;
  cursor: pointer;
  transition: background 0.12s;
  text-align: left;
  background: none;
  border-left: 2px solid transparent;
}
.picker-item:hover { background: rgba(120,120,128,0.2); }

.picker-dot {
  width: 7px; height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}

.picker-info {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.picker-title {
  font-size: 13px;
  color: var(--clr-text);
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.picker-meta {
  font-size: 11px;
  color: var(--clr-text-3);
  white-space: nowrap;
  flex-shrink: 0;
}

.picker-check { color: var(--clr-accent); flex-shrink: 0; }

.picker-empty {
  padding: 20px;
  text-align: center;
  font-size: 13px;
  color: var(--clr-text-3);
}

/* ── Group chips ─────────────────────────────────────────────────────── */
.grp-chips { display: flex; flex-wrap: wrap; gap: 8px; }
.grp-chip {
  display: inline-flex; align-items: center; gap: 7px;
  padding: 6px 12px;
  font-size: 13px; color: var(--clr-text-2);
  background: var(--clr-bg);
  border: 1.5px solid var(--clr-border);
  border-radius: 100px;
  transition: border-color 0.12s, background 0.12s, color 0.12s;
}
.grp-chip:hover { background: var(--clr-surface-2); }
.grp-chip.on { border-color: var(--clr-accent); background: rgba(0,113,227,0.08); color: var(--clr-text); }
.grp-dot { width: 9px; height: 9px; border-radius: 50%; flex-shrink: 0; }

/* ── Actions ─────────────────────────────────────────────────────────── */
.panel-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 4px;
  padding-top: 16px;
  border-top: 1px solid var(--clr-border-light);
  flex-shrink: 0;
}

.actions-right { display: flex; gap: 8px; margin-left: auto; }

.btn-primary {
  padding: 9px 20px;
  font-size: 14px; font-weight: 600;
  color: #fff;
  background: var(--clr-accent);
  border-radius: var(--r-md);
  transition: background 0.15s, transform 0.1s;
}
.btn-primary:hover { background: var(--clr-accent-hover); }
.btn-primary:active { transform: scale(0.98); }

.btn-ghost {
  padding: 9px 16px;
  font-size: 14px; font-weight: 500;
  color: var(--clr-text-2);
  background: transparent;
  border-radius: var(--r-md);
  transition: background 0.15s;
}
.btn-ghost:hover { background: var(--clr-surface-2); }

.btn-danger {
  display: flex; align-items: center; gap: 6px;
  padding: 9px 14px;
  font-size: 13px; font-weight: 500;
  color: var(--clr-danger);
  background: rgba(255,59,48,0.07);
  border-radius: var(--r-md);
  transition: background 0.15s;
}
.btn-danger:hover { background: rgba(255,59,48,0.14); }
</style>
