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
                  <button v-if="canComment" type="button" class="icon-act" title="Write a comment" @click="focusComment"><MessageSquarePlus :size="15" /></button>
                  <button type="button" class="icon-act" :class="{ done: copied === 'link' }" :title="copied === 'link' ? 'Copied' : 'Copy link to this view'" @click="copy('link')"><Check v-if="copied === 'link'" :size="15" :stroke-width="2.5" /><Link2 v-else :size="15" /></button>
                  <!-- View formats live behind one menu — JSON/YAML are occasional
                       tools, not everyday buttons. -->
                  <div class="fmt-menu">
                    <button type="button" class="icon-act" :class="{ on: fmtOpen || viewFormat !== 'form' }" title="View format" @click="fmtOpen = !fmtOpen"><MoreHorizontal :size="15" /></button>
                    <div v-if="fmtOpen" class="fmt-bg" @click="fmtOpen = false"></div>
                    <div v-if="fmtOpen" class="fmt-pop">
                      <button type="button" class="fmt-opt" :class="{ on: viewFormat === 'form' }" @click="setFormat('form'); fmtOpen = false"><AlignLeft :size="13" /> Normal view</button>
                      <button type="button" class="fmt-opt" :class="{ on: viewFormat === 'json' }" @click="setFormat('json'); fmtOpen = false"><Braces :size="13" /> JSON</button>
                      <button type="button" class="fmt-opt" :class="{ on: viewFormat === 'yaml' }" @click="setFormat('yaml'); fmtOpen = false"><FileText :size="13" /> YAML</button>
                    </div>
                  </div>
                </template>
                <button v-if="canPropose && !proposing && viewVersion == null && viewFormat === 'form'" type="button" class="propose-act" @click="proposing = true; editing = true">{{ mode === 'add' ? 'Propose new item' : 'Propose change' }}</button>
                <button v-if="mode === 'edit' && !readOnly && editable && viewVersion == null" type="button" class="icon-act danger" title="Delete" @click="remove">
                  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6M10 11v6M14 11v6"/>
                  </svg>
                </button>
                <button v-if="embedded && !editable && !readOnly && viewVersion == null" type="button" class="icon-act primary" title="Edit" @click="setFormat('form'); editing = true"><Pencil :size="15" /></button>
                <button v-if="embedded && editable && mode === 'edit'" type="button" class="icon-act" title="Cancel — discard changes" @click="cancelEdit">
                  <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                    <path d="M1 1l12 12M13 1L1 13" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/>
                  </svg>
                </button>
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
            <form class="panel-form" :class="{ 'read-mode': formLocked }" @submit.prevent="onSave">
              <!-- JSON / YAML view replaces the FIELDS block AND the tabs below —
                   their content (uses, referenced by, history) is serialized into
                   the export instead. -->
              <div v-if="viewFormat !== 'form'" class="ms-code-view" :class="{ full: embedded }">
                <button type="button" class="code-copy" :class="{ done: copied === viewFormat }" :title="copied === viewFormat ? 'Copied' : 'Copy ' + viewFormat.toUpperCase()" @click="copy(viewFormat)">
                  <Check v-if="copied === viewFormat" :size="14" :stroke-width="2.5" /><CopyIcon v-else :size="14" />
                </button>
                <pre class="ms-code">{{ formattedText }}</pre>
              </div>
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
                  <span v-else class="type-static" :style="{ color: currentType?.color || swimlane?.color || '#8a8a8e' }">{{ currentType?.label || form.typeKey }}</span>
                </div>
                <p class="type-hint">{{ mode === 'add' && !formLocked ? 'The icon comes from the type — set it under Settings → Types.' : 'The type is fixed once an item is created.' }}</p>
              </div>

              <!-- Type-specific fields: schema comes from the selected item type. -->
              <div v-if="currentTypeFields.length" class="field type-fields span2">
                <label class="field-label">Fields</label>
                <!-- Read mode hides empty fields — a row of "—" carries no
                     information; a dim toggle brings them back on demand. -->
                <dl v-if="!editable" class="read-fields">
                  <template v-for="f in visibleReadFieldRows" :key="f.key">
                    <dt>{{ f.label }}</dt>
                    <dd v-if="f.refs" :class="{ 'read-empty': !f.refs.length }">
                      <span v-if="f.refs.length" class="read-refs">
                        <span v-for="r in f.refs" :key="r.id + ':' + (r.version || '')" class="read-pill" :class="{ missing: !r.exists }" :style="{ color: r.color, background: r.color + '22' }" @click="r.exists && openRef(r)"><MarkerIcon :shape="r.icon" :color="r.dot" :size="12" :fill="r.fill" />{{ r.title }}<span v-if="r.version" class="read-pill-ver">v{{ r.version }}</span></span>
                      </span>
                      <template v-else>—</template>
                    </dd>
                    <dd v-else :class="{ 'read-empty': !f.v, 'read-prose': f.prose }">{{ f.v || '—' }}</dd>
                  </template>
                </dl>
                <span v-if="!editable && emptyFieldCount" class="rf-toggle" role="button" @click="showEmptyFields = !showEmptyFields">
                  {{ showEmptyFields ? 'hide empty fields' : `${emptyFieldCount} empty ${emptyFieldCount === 1 ? 'field' : 'fields'} · show` }}
                </span>
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
              <div v-if="!isTimelineType && (editable || exclusive.mode !== 'off')" class="field span2 excl-block">
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
                <span v-if="!editable" class="read-val" :class="{ qa: quickEditable }" :role="quickEditable ? 'button' : null" :title="quickEditable ? 'Click to change' : null" @click="toggleQa('maturity')">{{ form.maturity ? MATURITY_STAGES[form.maturity - 1] : '—' }}<Pencil v-if="quickEditable" class="qa-pen" :size="11" /></span>
                <div v-if="!editable && qaOpen === 'maturity'" class="qa-wrap">
                  <div class="qa-bg" @click="qaOpen = null"></div>
                  <div class="qa-pop">
                    <span class="qa-opt" :class="{ on: !form.maturity }" role="button" @click="setQuick('maturity', null)">None</span>
                    <span v-for="(s, i) in MATURITY_STAGES" :key="s" class="qa-opt" :class="{ on: form.maturity === i + 1 }" role="button" @click="setQuick('maturity', i + 1)">{{ s }}</span>
                  </div>
                </div>
                <div v-if="editable" class="maturity-row">
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
                <span v-if="!editable" class="read-val" :class="{ qa: quickEditable }" :role="quickEditable ? 'button' : null" :title="quickEditable ? 'Click to change' : null" @click="toggleQa('progress')">{{ form.progress != null ? form.progress + '%' : '—' }}<Pencil v-if="quickEditable" class="qa-pen" :size="11" /></span>
                <div v-if="!editable && qaOpen === 'progress'" class="qa-wrap">
                  <div class="qa-bg" @click="qaOpen = null"></div>
                  <div class="qa-pop">
                    <span class="qa-opt" :class="{ on: form.progress == null }" role="button" @click="setQuick('progress', null)">None</span>
                    <span v-for="p in [0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100]" :key="p" class="qa-opt num" :class="{ on: form.progress === p }" role="button" @click="setQuick('progress', p)">{{ p }}%</span>
                  </div>
                </div>
                <div v-if="editable" class="progress-row">
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
                <span v-if="!editable" class="read-val" :class="{ qa: quickEditable }" :role="quickEditable ? 'button' : null" :title="quickEditable ? 'Click to change' : null" @click="toggleQa('assignee')">{{ form.assigneeId ? (memberName(form.assigneeId) || 'someone') : 'Unassigned' }}<Pencil v-if="quickEditable" class="qa-pen" :size="11" /></span>
                <div v-if="!editable && qaOpen === 'assignee'" class="qa-wrap">
                  <div class="qa-bg" @click="qaOpen = null"></div>
                  <div class="qa-pop">
                    <span class="qa-opt" :class="{ on: !form.assigneeId }" role="button" @click="setQuick('assigneeId', null)">Unassigned</span>
                    <span v-for="mb in workspace.members" :key="mb.userId" class="qa-opt" :class="{ on: form.assigneeId === mb.userId }" role="button" @click="setQuick('assigneeId', mb.userId)">{{ mb.username }}</span>
                  </div>
                </div>
                <select v-if="editable" class="field-input" :disabled="formLocked" v-model="form.assigneeId">
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

              <div v-if="viewFormat === 'form'" class="ms-tabs" role="tablist">
                <button v-if="typeStatuses.length" type="button" class="ms-tab" :class="{ active: tab === 'flow' }" @click="tab = 'flow'"><Workflow :size="13" :stroke-width="2" />Flow</button>
                <button v-if="currentSchedulable" type="button" class="ms-tab" :class="{ active: tab === 'deps' }" @click="tab = 'deps'"><CalendarClock :size="13" :stroke-width="2" />Scheduling</button>
                <button type="button" class="ms-tab" :class="{ active: tab === 'uses' }" @click="tab = 'uses'"><Boxes :size="13" :stroke-width="2" />Uses</button>
                <button v-if="mode === 'edit'" type="button" class="ms-tab" :class="{ active: tab === 'structure' }" @click="tab = 'structure'"><Network :size="13" :stroke-width="2" />Graph</button>
                <button v-if="mode === 'edit'" type="button" class="ms-tab" title="Bill of Material" :class="{ active: tab === 'bom' }" @click="tab = 'bom'"><Table2 :size="13" :stroke-width="2" />BOM</button>
                <button type="button" class="ms-tab" :class="{ active: tab === 'refby' }" @click="tab = 'refby'"><Link2 :size="13" :stroke-width="2" />Referenced by</button>
                <button v-if="currentSchedulable" type="button" class="ms-tab" :class="{ active: tab === 'groups' }" @click="tab = 'groups'"><Layers :size="13" :stroke-width="2" />Groups</button>
                <button v-if="mode === 'edit' && !milestone?.sourceSystem" type="button" class="ms-tab" :class="{ active: tab === 'history' }" @click="tab = 'history'"><History :size="13" :stroke-width="2" />History</button>
                <button v-if="typeStatuses.length" type="button" class="ms-tab" :class="{ active: tab === 'log' }" @click="tab = 'log'"><Terminal :size="13" :stroke-width="2" />Log</button>
              </div>

              <!-- The History tab is read-only display, so it's never form-disabled
                   (you can browse versions even when the rest is read-only). -->
              <fieldset v-if="viewFormat === 'form'" class="ms-tab-body" :disabled="formLocked && tab !== 'history' && tab !== 'structure' && tab !== 'bom' && tab !== 'log'">
              <div v-show="tab === 'history'" class="ms-panel ms-history">
                <ItemHistory v-if="milestone && tab === 'history'" :key="milestone.id" :item-id="milestone.id" :current-version="viewVersion || headVersion" @select="onSelectVersion" />
              </div>
              <!-- Log: the real version history as terminal lines, with the status
                   console (teleported out of the Flow diagram) docked below it. -->
              <div v-show="tab === 'log'" class="ms-panel ms-log">
                <!-- ONE terminal: history lines on top, the live status console
                     (teleported from the Flow diagram) as the prompt at the end. -->
                <div ref="logHistEl" class="log-hist">
                  <div v-for="e in logDisplay" :key="e.key" class="log-line">
                    <template v-if="e.kind === 'revgroup' && e.items.length === 1">
                      <span class="log-prompt">atlas:history</span><span class="log-time">[{{ fmtStamp(e.first.time) }}]</span><span class="log-gt">&gt;</span><span class="log-text">saved <strong>v{{ e.first.version }}</strong> · {{ who(e.first.by) }}</span>
                    </template>
                    <template v-else-if="e.kind === 'revgroup'">
                      <span class="log-prompt">atlas:history</span><span class="log-time">[{{ revRange(e) }}]</span><span class="log-gt">&gt;</span><span class="log-text">saved <button type="button" class="log-ref" title="Open the History tab" @click="tab = 'history'">v{{ e.first.version }} → v{{ e.last.version }}</button> · {{ e.sameBy !== null ? who(e.sameBy) : 'several editors' }} <span class="log-dim">({{ e.items.length }} saves)</span></span>
                    </template>
                    <template v-else-if="e.kind === 'comment'">
                      <span class="log-prompt cmt">atlas:comment</span><span class="log-time">[{{ fmtStamp(e.time) }}]</span><span class="log-gt cmt">&gt;</span><span class="log-text"><strong>{{ who(e.by) }}</strong>: <template v-for="(p, i) in commentParts(e.body)" :key="i"><button v-if="p.id" type="button" class="log-ref" @click="openRef({ id: p.id, exists: true })">{{ p.title }}</button><span v-else>{{ p.text }}</span></template></span>
                      <button v-if="canDeleteComment(e)" type="button" class="log-del" title="Delete comment" @click="removeComment(e.id)">×</button>
                    </template>
                    <template v-else>
                      <span class="log-prompt mnt">atlas:mention</span><span class="log-time">[{{ fmtStamp(e.time) }}]</span><span class="log-gt mnt">&gt;</span><span class="log-text"><strong>{{ who(e.by) }}</strong> on <button type="button" class="log-ref" @click="openRef({ id: e.host, exists: true })">{{ hostTitle(e.host) }}</button>: <template v-for="(p, i) in commentParts(e.body)" :key="i"><button v-if="p.id" type="button" class="log-ref" @click="openRef({ id: p.id, exists: true })">{{ p.title }}</button><span v-else>{{ p.text }}</span></template></span>
                    </template>
                  </div>
                  <div v-if="!logEntries.length" class="log-line log-none">no history yet</div>
                  <!-- While composing, the status prompt (incl. its cursor) hides —
                       the comment line IS the terminal's active line. CSS-hide, not
                       v-if: the teleported console must keep its target mounted. -->
                  <div v-if="typeStatuses.length" v-show="!composing" id="modal-console-dock" class="modal-console-dock"></div>
                  <!-- Comment prompt: appears only on demand — via "comment" in the
                       status prompt. The dim trigger line shows ONLY for items
                       without a status console (no prompt to click there). "/ref"
                       opens an item picker; picks are sent as [[id]] tokens. -->
                  <button v-if="canComment && !composing && !typeStatuses.length" type="button" class="log-line log-add" @click="focusComment">
                    <span class="log-prompt cmt">{{ myName }}:comment</span><span class="log-gt cmt">&gt;</span><span class="log-add-t">write a comment…</span>
                  </button>
                  <div v-if="canComment && composing" class="log-input-row">
                    <span class="log-prompt cmt">{{ myName }}:comment</span><span class="log-gt cmt">&gt;</span>
                    <!-- Terminal-style caret: the native (thin) caret is hidden and a
                         block cursor — same as the status prompt's — sits at the end
                         of the typed text. The input auto-sizes to its content. -->
                    <div class="log-input-wrap" @click="commentInput?.focus()">
                      <input
                        ref="commentInput" v-model="commentText" class="log-input" autocomplete="off" spellcheck="false"
                        :style="{ width: Math.max(commentText.length, 1) + 'ch' }"
                        @input="updateRefSuggest"
                        @keydown.enter.prevent="onCommentEnter"
                        @keydown.down.prevent="refIdx = Math.min(refIdx + 1, refResults.length - 1)"
                        @keydown.up.prevent="refIdx = Math.max(refIdx - 1, 0)"
                        @keydown.esc.stop="onCommentEsc"
                        @blur="onCommentBlur"
                      />
                      <span class="log-cursor"></span>
                      <span v-if="!commentText" class="log-hint">write a comment — type /ref to link an item</span>
                      <div v-if="refResults.length" class="log-refpick">
                        <button v-for="(m, i) in refResults" :key="m.id" type="button" class="log-refopt" :class="{ act: i === refIdx }" @mousedown.prevent="pickRefItem(m)" @mousemove="refIdx = i">
                          <span class="picker-dot" :style="{ background: usesDot(m) }"></span>
                          <span class="log-refopt-t">{{ m.title }}</span>
                          <span class="log-refopt-ty">{{ usesTypeLabel(m) }}</span>
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div v-show="tab === 'flow'" class="ms-panel ms-flow">
                <StatusFlow v-if="typeStatuses.length" inline :statuses="typeStatuses" :current="form.status" :version="milestone?.version || 1" :read-only="readOnly || viewVersion != null" :viewing-version="viewVersion || 0" :arrangeable="false" :layout="currentType?.layout" :commentable="canComment && mode === 'edit'" :user-name="myName" @advance="onFlowAdvance" @back-to-latest="backToLatest" @comment="focusComment" />
              </div>

              <div v-show="tab === 'deps' && !editable" class="ms-panel">
                <ul v-if="readDeps.length" class="read-deps">
                  <li v-for="g in readDeps" :key="g.rel">
                    <span class="read-dep-rel">{{ g.rel }}</span>
                    <span class="read-refs">
                      <span v-for="d in g.items" :key="d.id" class="read-pill" :class="{ 'pill-conflict': pillConflict(d.id) }" :style="{ color: d.color, background: d.color + '22' }" @click="openRef({ id: d.id, version: d.version, exists: true })"><MarkerIcon :shape="d.icon" :color="d.dot" :size="12" :fill="d.fill" />{{ d.title }}<span v-if="d.version" class="read-pill-ver">v{{ d.version }}</span><AlertTriangle v-if="pillConflict(d.id)" :size="12" :stroke-width="2.4" color="#FF3B30" :title="pillConflict(d.id)" /></span>
                    </span>
                  </li>
                </ul>
                <p v-else class="read-none">No dependencies.</p>
              </div>

              <div v-show="tab === 'deps' && editable" class="ms-panel">
              <div v-if="availableRelTypes.length > 1" class="field">
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

              <!-- Uses (STRUCTURE / composition — the BOM edge). One-directional:
                   you edit only what THIS item uses; the reverse ("Used by") is
                   read-only under the Referenced-by tab. Version-pinnable = config. -->
              <!-- Both directions always render, an empty one says "none" — so the
                   structure situation is visible at a glance without guessing. -->
              <div v-show="tab === 'uses' && !editable" class="ms-panel">
                <ul class="read-deps">
                  <li>
                    <span class="read-dep-rel">Uses</span>
                    <span class="read-refs">
                      <span v-for="d in usesOut" :key="d.id" class="read-pill" :class="{ 'pill-conflict': pillConflict(d.id) }" :style="{ color: d.color, background: d.color + '22' }" @click="openRef({ id: d.id, version: d.version, exists: true })"><MarkerIcon :shape="d.icon" :color="d.dot" :size="12" :fill="d.fill" /><span class="read-pill-qty">{{ d.qty || 1 }}×</span>{{ d.title }}<span v-if="d.designators" class="read-pill-ver">{{ d.designators }}</span><span v-if="d.version" class="read-pill-ver">v{{ d.version }}</span><AlertTriangle v-if="pillConflict(d.id)" :size="12" :stroke-width="2.4" color="#FF3B30" :title="pillConflict(d.id)" /></span>
                      <span v-if="!usesOut.length" class="read-none-i">none</span>
                    </span>
                  </li>
                  <li>
                    <span class="read-dep-rel">Used by</span>
                    <span class="read-refs">
                      <span v-for="d in usesIn" :key="d.id" class="read-pill" :style="{ color: d.color, background: d.color + '22' }" @click="openRef({ id: d.id, exists: true })"><MarkerIcon :shape="d.icon" :color="d.dot" :size="12" :fill="d.fill" /><span class="read-pill-qty">{{ d.qty || 1 }}×</span>{{ d.title }}<span v-if="d.designators" class="read-pill-ver">{{ d.designators }}</span></span>
                      <span v-if="!usesIn.length" class="read-none-i">none</span>
                    </span>
                  </li>
                </ul>
              </div>

              <div v-show="tab === 'uses' && editable" class="ms-panel">
                <p class="uses-hint">Backlog items this one <strong>uses / is composed of</strong> (the structure / BOM edge). Timeline items aren't selectable here.</p>
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
                            <div class="picker-info"><span class="picker-title">{{ m.title }}</span><span class="picker-meta">{{ usesTypeLabel(m) }}<template v-if="m.swimlaneId || m.sourceSystem"> · timeline</template></span></div>
                            <svg v-if="localUsesIds.has(m.id)" class="picker-check" width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M2.5 7L5.5 10L11.5 4" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/></svg>
                          </button>
                          <input v-if="localUsesIds.has(m.id)" v-model="usesQty[m.id]" class="picker-qty" type="number" min="1" step="1" placeholder="1" title="Quantity — how many of this the item uses (BOM multiplicity)" @click.stop @mousedown.stop />
                          <span v-if="localUsesIds.has(m.id)" class="picker-qty-x">×</span>
                          <input v-if="localUsesIds.has(m.id)" v-model="usesRefDes[m.id]" class="picker-refdes" type="text" spellcheck="false" autocomplete="off" placeholder="RefDes (C1, C4-C7)" title="Reference designators — names of the usage positions, e.g. C1, C2, C10-C17" @click.stop @mousedown.stop />
                          <AlertTriangle v-if="localUsesIds.has(m.id) && refDesMismatch(m.id)" :size="13" :stroke-width="2.4" color="#FF9F0A" class="picker-refdes-warn" :title="refDesMismatch(m.id)" />
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
              </div>

              <!-- Graph + BOM: this item's exploded structure (node graph / BOM
                   table), read-only display — composition is edited under Uses.
                   Clicking a node/row navigates to that item (like a ref pill). -->
              <div v-show="tab === 'structure'" class="ms-panel ms-struct">
                <StructureGraph v-if="milestone && tab === 'structure'" :key="milestone.id" :root-id="milestone.id" view="graph" @edit="m => openRef({ id: m.id, exists: true })" />
              </div>
              <div v-show="tab === 'bom'" class="ms-panel ms-struct">
                <StructureGraph v-if="milestone && tab === 'bom'" :key="milestone.id" :root-id="milestone.id" view="table" @edit="m => openRef({ id: m.id, exists: true })" />
              </div>

              <!-- Referenced by: all INCOMING links, read-only. Two kinds: items
                   that USE this one (uses reverse) and items whose reference FIELD
                   points here. Edit the link from the other item, never here. -->
              <div v-show="tab === 'refby'" class="ms-panel">
                <ul v-if="usedByGroup.items.length || referencedByGroups.length || mentionHosts.length" class="read-deps">
                  <li v-if="usedByGroup.items.length" :key="'usedby'">
                    <span class="read-dep-rel">{{ usedByGroup.label }}</span>
                    <span class="read-refs">
                      <span v-for="d in usedByGroup.items" :key="d.id" class="read-pill" :style="{ color: d.color, background: d.color + '22' }" @click="openRef({ id: d.id, exists: true })"><MarkerIcon :shape="d.icon" :color="d.dot" :size="12" :fill="d.fill" />{{ d.title }}<span class="read-pill-ver">{{ d.qty || 1 }}×</span></span>
                    </span>
                  </li>
                  <li v-for="g in referencedByGroups" :key="g.key">
                    <span class="read-dep-rel">{{ g.label }}</span>
                    <span class="read-refs">
                      <span v-for="d in g.items" :key="d.id" class="read-pill" :style="{ color: d.color, background: d.color + '22' }" @click="openRef({ id: d.id, exists: true })"><MarkerIcon :shape="d.icon" :color="d.dot" :size="12" :fill="d.fill" />{{ d.title }}</span>
                    </span>
                  </li>
                  <li v-if="mentionHosts.length">
                    <span class="read-dep-rel">Mentioned in comments</span>
                    <span class="read-refs">
                      <span v-for="d in mentionHosts" :key="d.id" class="read-pill" :style="{ color: d.color, background: d.color + '22' }" @click="openRef({ id: d.id, exists: true })"><MarkerIcon :shape="d.icon" :color="d.dot" :size="12" :fill="d.fill" />{{ d.title }}</span>
                    </span>
                  </li>
                </ul>
                <p v-else class="read-none">Nothing references this yet.</p>
                <p class="uses-hint">Read-only — add or change these links from the other item.</p>
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

          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { reactive, ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useAppStore, MONTHS, MATURITY_STAGES, store, groups, swatchColors, stripMarkdown, itemTypes, itemTypeByKey, RELATIONSHIP_TYPES, workspace, session, baselines, canEditWorkspace, canProposeChanges, proposeChange, proposeCreate, memberName, memberInitials, memberById, openProfile, STATUS_TONES, toneColor, statusColor, parseRef, parseDesignators, itemLink, itemStatus, isSchedulableItem, ui, pushNav, checkResourceConflicts, resourceConflicts } from '../stores/useAppStore.js'

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
import StructureGraph from './StructureGraph.vue'
import { Lock, History, Workflow, Link2, Braces, FileText, Pencil, Check, AlignLeft, AlertTriangle, CalendarClock, Boxes, Network, Table2, Layers, Copy as CopyIcon, Terminal, MoreHorizontal, MessageSquarePlus } from 'lucide-vue-next'

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

const TABS = ['flow', 'deps', 'uses', 'structure', 'bom', 'refby', 'groups', 'history', 'log']
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
function decodeRefPins() {
  for (const k of Object.keys(refPins)) delete refPins[k]
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
}
decodeRefPins()

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
// Structured reference for the code views: stable id + human title + the version
// state as separate fields — no "Radar · v2 (latest v4)" display strings that a
// consumer would have to regex apart.
function refExport(entry) {
  const { id, version } = parseRef(entry)
  const target = store.milestones.find(m => m.id === id)
  const o = { item: target?.title || id, id }
  if (version) {
    o.version = `v${version} (pinned)`
    const head = target?.version
    if (head && head > version) o.latest = `v${head}`
  } else {
    o.version = 'latest'
  }
  return o
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
// Read mode hides empty fields (a "—" row says nothing); a dim toggle reveals them.
const showEmptyFields = ref(false)
const rowFilled = (r) => (r.refs ? r.refs.length > 0 : !!r.v)
const visibleReadFieldRows = computed(() => showEmptyFields.value ? readFieldRows.value : readFieldRows.value.filter(rowFilled))
const emptyFieldCount = computed(() => readFieldRows.value.filter(r => !rowFilled(r)).length)

// View-format overflow menu (Normal / JSON / YAML).
const fmtOpen = ref(false)

// Quick actions: assignee / progress / maturity are editable straight from the
// read card (small popover, auto-saved) — same rule as the status quick-action.
const qaOpen = ref(null) // 'maturity' | 'progress' | 'assignee' | null
const quickEditable = computed(() => props.mode === 'edit' && !!props.milestone && !readOnly.value && viewVersion.value == null && !proposing.value)
function toggleQa(which) {
  if (!quickEditable.value) return
  qaOpen.value = qaOpen.value === which ? null : which
}
function setQuick(field, value) {
  qaOpen.value = null
  form[field] = value
  updateMilestone(props.milestone.id, { [field]: value })
}

// All links touching this item, grouped by relationship (read-mode Dependencies).
const readDependencyGroups = computed(() => {
  const id = props.milestone?.id
  if (!id) return []
  const byId = new Map(store.milestones.map(m => [m.id, m]))
  const relLabel = (rel, fwd) => { const r = RELATIONSHIP_TYPES.find(x => x.key === rel); return fwd ? (r?.label || rel) : (r?.inverse || rel) }
  const out = []
  for (const l of store.links) {
    const relKey = l.rel || 'depends-on'
    if (l.a === id && byId.has(l.b)) out.push({ relKey, rel: relLabel(l.rel, true), id: l.b, version: l.version ?? null, qty: l.qty ?? null, designators: l.designators || '', ...itemPill(l.b) })
    else if (l.b === id && byId.has(l.a)) out.push({ relKey, rel: relLabel(l.rel, false), id: l.a, version: l.version ?? null, qty: l.qty ?? null, ...itemPill(l.a) })
  }
  const m = new Map()
  for (const d of out) { if (!m.has(d.rel)) m.set(d.rel, { rel: d.rel, relKey: d.relKey, items: [] }); m.get(d.rel).items.push(d) }
  return [...m.values()]
})
// Scheduling tab shows only the temporal relation; the Uses tab shows composition.
const readDeps = computed(() => readDependencyGroups.value.filter(g => g.relKey === 'depends-on'))
const readUses = computed(() => readDependencyGroups.value.filter(g => g.relKey === 'uses'))
// The two directions split out — the read tab always renders both rows.
const usesOut = computed(() => readUses.value.find(g => g.rel === 'Uses')?.items || [])
const usesIn = computed(() => readUses.value.find(g => g.rel === 'Used by')?.items || [])
// Used by (uses reverse) — read-only, shown under the Referenced-by tab.
const usedByGroup = computed(() => {
  const id = props.milestone?.id
  const items = []
  if (id) {
    for (const l of store.links) {
      if ((l.rel || 'depends-on') === 'uses' && l.b === id && store.milestones.some(m => m.id === l.a)) {
        items.push({ id: l.a, qty: l.qty ?? null, ...itemPill(l.a) })
      }
    }
  }
  return { label: 'Used by', items }
})

// Reference-field back-links: which items point at THIS one through a reference
// field (e.g. every Maschine whose "Projekt" field targets this project). Read-
// only and one-directional — links are always edited from the referencing item.
// Reference values live in item.data (single id or array, optionally id@vN), so
// this is a pure client-side reverse scan; grouped by referencing type + field.
const referencedByGroups = computed(() => {
  const id = props.milestone?.id
  if (!id) return []
  const groups = new Map()
  for (const m of store.milestones) {
    if (m.id === id) continue
    const t = itemTypeByKey(m.typeKey || m.kind || 'milestone')
    if (!t) continue
    for (const f of (t.fields || [])) {
      if (f.type !== 'reference') continue
      const v = m.data?.[f.key]
      const vals = Array.isArray(v) ? v : (v ? [v] : [])
      if (!vals.some(entry => parseRef(entry).id === id)) continue
      const key = (m.typeKey || m.kind || 'milestone') + '::' + f.key
      if (!groups.has(key)) groups.set(key, { key, label: `${t.label} · ${f.label || f.key}`, items: [] })
      groups.get(key).items.push({ id: m.id, ...itemPill(m.id) })
    }
  }
  return [...groups.values()]
})
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
    const entries = Array.isArray(v) ? v : (v ? [v] : [])
    const refs = entries.map(refExport)
    return f.refMulti ? refs : (refs[0] || '')
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
  // The tabs are hidden in the code views, so their content rides along in the
  // export instead: structure (both directions), reference back-links, history.
  o.uses = usesOut.value.map(d => {
    const u = { item: d.title, id: d.id, qty: d.qty || 1 }
    if (d.designators) u.designators = d.designators
    u.version = d.version ? `v${d.version} (pinned)` : 'latest'
    return u
  })
  o.usedBy = usesIn.value.map(d => {
    const u = { item: d.title, id: d.id, qty: d.qty || 1 }
    if (d.designators) u.designators = d.designators
    return u
  })
  o.referencedBy = referencedByGroups.value.flatMap(g => g.items.map(d => ({ item: d.title, id: d.id, via: g.label })))
  if (comments.value.length) {
    o.comments = comments.value.map(c => ({ by: who(c.authorId), at: c.createdAt, text: commentParts(c.body).map(p => p.id ? p.title : p.text).join('') }))
  }
  const mIn = mentions.value.filter(c => c.itemId !== it.id).map(c => ({ item: hostTitle(c.itemId), id: c.itemId, by: who(c.authorId), at: c.createdAt, text: commentParts(c.body).map(p => p.id ? p.title : p.text).join('') }))
  if (mIn.length) o.mentionedIn = mIn
  if (exportRevisions.value.length) {
    o.history = exportRevisions.value.map(r => ({ version: `v${r.version}`, by: who(r.editedBy), at: r.editedAt }))
  }
  return o
})
// Revisions + comments for the code views AND the Log tab — fetched lazily the
// first time either opens (small metadata, no snapshots).
const exportRevisions = ref([])
const comments = ref([])
const mentions = ref([]) // comments elsewhere that reference THIS item ([[id]])
const logHistEl = ref(null)
function scrollLogSoon() { nextTick(() => { const el = logHistEl.value; if (el) el.scrollTop = el.scrollHeight }) }
// Terminal behaviour: anything appended (status move, comment, the composer,
// the typing prompt) keeps the view pinned to the bottom — unless the user has
// deliberately scrolled up to read older lines.
let logObs = null
onMounted(() => {
  if (!logHistEl.value || typeof MutationObserver === 'undefined') return
  logObs = new MutationObserver(() => {
    const el = logHistEl.value
    if (!el) return
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 160) el.scrollTop = el.scrollHeight
  })
  logObs.observe(logHistEl.value, { childList: true, subtree: true, characterData: true })
})
onBeforeUnmount(() => { if (logObs) logObs.disconnect() })
watch([viewFormat, tab, () => props.milestone?.id], async ([f, tb, id]) => {
  if (!id) return
  if (f === 'form' && tb !== 'log' && tb !== 'refby') return
  const [revs, cs, ms] = await Promise.all([
    api.listRevisions(id).catch(() => []),
    api.listComments(id).catch(() => []),
    api.listMentions(id).catch(() => []),
  ])
  exportRevisions.value = revs || []
  comments.value = cs || []
  mentions.value = ms || []
  // terminal convention: newest at the bottom, scrolled into view
  if (tb === 'log') scrollLogSoon()
}, { immediate: true })
// One merged terminal stream: history + comments + incoming mentions, oldest
// first. Self-mentions are already in the stream as the comment itself.
const logEntries = computed(() => {
  const revs = exportRevisions.value.map(r => ({ kind: 'rev', key: 'r' + r.version, time: r.editedAt, version: r.version, by: r.editedBy }))
  const cs = comments.value.map(c => ({ kind: 'comment', key: 'c' + c.id, time: c.createdAt, id: c.id, by: c.authorId, body: c.body }))
  const ms = mentions.value.filter(c => c.itemId !== props.milestone?.id).map(c => ({ kind: 'mention', key: 'm' + c.id, time: c.createdAt, by: c.authorId, body: c.body, host: c.itemId }))
  return [...revs, ...cs, ...ms].sort((a, b) => String(a.time || '').localeCompare(String(b.time || '')))
})
// The DISPLAYED stream compresses runs of consecutive saves into one line
// ("saved v107 → v122 · name (16 saves)") — the full list lives in the History
// tab; the log only keeps saves as time anchors between comments/mentions.
const logDisplay = computed(() => {
  const out = []
  let run = null
  const flush = () => {
    if (!run) return
    const items = run
    const first = items[0], last = items[items.length - 1]
    const authors = [...new Set(items.map(i => i.by || ''))]
    out.push({ kind: 'revgroup', key: 'g' + first.version + '-' + last.version, items, first, last, sameBy: authors.length === 1 ? first.by : null })
    run = null
  }
  for (const e of logEntries.value) {
    if (e.kind === 'rev') { (run = run || []).push(e); continue }
    flush()
    out.push(e)
  }
  flush()
  return out
})
function shortDate(iso) { const d = new Date(iso); return Number.isNaN(d.getTime()) ? '' : `${d.getDate()} ${MONTHS[d.getMonth()]}` }
function revRange(e) {
  const a = shortDate(e.first.time), b = shortDate(e.last.time)
  return a === b ? fmtStamp(e.last.time) : `${a} – ${b}`
}

// Items whose comments mention this one — the Referenced-by tab's third group.
const mentionHosts = computed(() => {
  const ids = [...new Set(mentions.value.map(c => c.itemId))]
  return ids.filter(id => id !== props.milestone?.id && store.milestones.some(m => m.id === id)).map(id => ({ id, ...itemPill(id) }))
})

// ── Comments: terminal input line with /ref item autocompletion ─────────────
// Picked items are shown as [[Title]] in the input and sent as [[id]] tokens;
// display resolves ids back to (current) titles, clickable like ref pills.
const canComment = computed(() => canEditWorkspace() || canProposeChanges())
const commentText = ref('')
const commentInput = ref(null)
const pickedRefs = ref([]) // { title, id } in pick order (consumed on submit)
const refQuery = ref(null) // null = picker closed; '' / text = open with filter
const refIdx = ref(0)
function updateRefSuggest() {
  const m = commentText.value.match(/\/ref\s*([^[\]]*)$/i)
  refQuery.value = m ? m[1].trim().toLowerCase() : null
  refIdx.value = 0
}
const refResults = computed(() => {
  if (refQuery.value == null) return []
  const q = refQuery.value
  return store.milestones.filter(m => m.id !== props.milestone?.id && (!q || (m.title || '').toLowerCase().includes(q))).slice(0, 8)
})
function pickRefItem(m) {
  commentText.value = commentText.value.replace(/\/ref\s*([^[\]]*)$/i, `[[${m.title}]] `)
  pickedRefs.value.push({ title: m.title, id: m.id })
  refQuery.value = null
  commentInput.value?.focus()
}
function onCommentEnter() {
  if (refResults.value.length) { pickRefItem(refResults.value[refIdx.value] || refResults.value[0]); return }
  postComment()
}
function encodeCommentBody(text) {
  const pool = [...pickedRefs.value]
  return text.replace(/\[\[([^\]]+)\]\]/g, (full, title) => {
    const i = pool.findIndex(p => p.title === title)
    if (i === -1) return full
    const [p] = pool.splice(i, 1)
    return `[[${p.id}]]`
  })
}
async function postComment() {
  const text = commentText.value.trim()
  if (!text || !props.milestone) return
  const body = encodeCommentBody(text)
  commentText.value = ''
  pickedRefs.value = []
  refQuery.value = null
  composing.value = false // done — the status prompt returns as the active line
  try {
    const c = await api.addComment(props.milestone.id, body)
    comments.value.push(c)
  } catch (e) { alert(e?.message || 'Could not post the comment') }
  scrollLogSoon()
}
function hostTitle(id) { return store.milestones.find(m => m.id === id)?.title || '(unknown item)' }
function commentParts(body) {
  return String(body || '').split(/(\[\[[^\]]+\]\])/).filter(Boolean).map(part => {
    const m = part.match(/^\[\[([^\]]+)\]\]$/)
    if (!m) return { text: part }
    const it = store.milestones.find(x => x.id === m[1])
    return it ? { id: it.id, title: it.title } : { text: m[1] }
  })
}
const myMemberId = computed(() => workspace.members?.find(w => w.username === session.username)?.id || null)
const myName = computed(() => memberName(myMemberId.value) || session.username || 'you')
function canDeleteComment(e) { return !!e.by && (e.by === myMemberId.value || e.by === 'you') }
async function removeComment(id) {
  try {
    await api.deleteComment(id)
    comments.value = comments.value.filter(c => c.id !== id)
  } catch (e) { alert(e?.message || 'Could not delete the comment') }
}
// The input line exists only while composing — opened via the status prompt's
// "comment" command or the dim trigger line; Esc / clicking away closes it.
const composing = ref(false)
function focusComment() {
  tab.value = 'log'
  composing.value = true
  nextTick(() => { commentInput.value?.focus(); scrollLogSoon() })
}
function onCommentEsc() {
  if (refQuery.value != null) { refQuery.value = null; return }
  composing.value = false
}
function onCommentBlur() {
  if (!commentText.value.trim() && refQuery.value == null) composing.value = false
}
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
      lines.push(`${pad}${k}:`)
      for (const el of v) {
        if (el && typeof el === 'object' && !Array.isArray(el)) {
          // object list entry: "- key: v" on the marker line, rest indented under it
          const sub = toYaml(el, indent + 2).split('\n')
          lines.push(`${pad}  - ${sub[0].trim()}`)
          for (let i = 1; i < sub.length; i++) lines.push(sub[i])
        } else {
          lines.push(`${pad}  - ${yamlScalar(el)}`)
        }
      }
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
// scheduling links live on the timeline. Uses (structure) + Referenced-by apply
// to every item.
watch(currentSchedulable, (ok) => { if (!ok && (tab.value === 'deps' || tab.value === 'groups')) tab.value = 'flow' }, { immediate: true })
const SELF = props.milestone?.id || '__NEW__'
const originalEdges = (props.milestone ? store.links.filter(l => l.a === SELF || l.b === SELF) : [])
  .map(l => ({ a: l.a, b: l.b, rel: l.rel || 'depends-on', version: l.version ?? null, qty: l.qty ?? null, designators: l.designators || '' }))
const edges = ref(originalEdges.map(e => ({ ...e })))

// Pinned versions + quantities for outgoing "uses" links (this item uses N× X at
// version V) — the configuration + multiplicity dimensions of the BOM edge. Kept
// out of edge membership so the version <select> / qty <input> can v-model them.
const usesPins = reactive({})   // { [usedItemId]: version }
const usesQty = reactive({})    // { [usedItemId]: quantity } — empty/1 = default
const usesRefDes = reactive({}) // { [usedItemId]: "C1, C2, C10-C17" } — free text
for (const l of originalEdges) {
  if (l.rel !== 'uses' || l.a !== SELF) continue
  if (l.version != null) usesPins[l.b] = l.version
  if (l.qty != null) usesQty[l.b] = l.qty
  if (l.designators) usesRefDes[l.b] = l.designators
}
// Soft plausibility hint: when the designators parse to a countable list and
// that count differs from qty, warn — non-blocking, qty stays authoritative.
function refDesMismatch(id) {
  const parsed = parseDesignators(usesRefDes[id])
  if (!parsed) return ''
  const q = Number(usesQty[id]) >= 2 ? Math.floor(Number(usesQty[id])) : 1
  return parsed.length === q ? '' : `Designators count ${parsed.length}, but quantity is ${q}`
}

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

function buildPickerGroups(query, onlyIds, backlogOnly = false, includeIds = null) {
  const q = (query || '').toLowerCase()
  const groups = []
  // Uses tab: off-timeline (backlog) items only, in one flat "Backlog" group.
  // includeIds keeps ALREADY-LINKED items visible even when they fail the backlog
  // rule (e.g. a timeline item linked before it was scheduled) — otherwise the
  // link would be stuck: shown in read mode but impossible to deselect here. The
  // rule only restricts NEW links, never editing existing ones.
  if (backlogOnly) {
    const backlog = store.milestones.filter(m => {
      if (m.id === props.milestone?.id) return false
      const isBacklog = !m.swimlaneId && !m.sourceSystem
      if (!isBacklog && !(includeIds && includeIds.has(m.id))) return false
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

// ── Uses tab: the composition / BOM edge (backlog items this one uses). Stored
// as rel='uses' links, version-pinnable. Edited ONE-directionally — only the
// outgoing "Uses" side; the reverse ("Used by") is read-only under Referenced-by.
const localUsesIds = computed(() => new Set(edges.value.filter(e => e.a === SELF && e.rel === 'uses').map(e => e.b)))
function toggleUses(id) {
  const had = edges.value.some(e => e.a === SELF && e.b === id && e.rel === 'uses')
  edges.value = edges.value.filter(e => !(e.rel === 'uses' && ((e.a === SELF && e.b === id) || (e.a === id && e.b === SELF))))
  if (!had) edges.value.push({ a: SELF, b: id, rel: 'uses' })
  else { delete usesPins[id]; delete usesQty[id]; delete usesRefDes[id] }
}
function usesDot(m) { return itemTypeByKey(m.typeKey || m.kind)?.color || '#8a8a8e' }
function usesTypeLabel(m) { return itemTypeByKey(m.typeKey || m.kind)?.label || (m.typeKey || m.kind || '') }
const usesSearch = ref('')
const showOnlyU1 = ref(false)
const usesGroups = computed(() => buildPickerGroups(usesSearch.value, showOnlyU1.value && localUsesIds.value.size ? localUsesIds.value : null, true, localUsesIds.value))

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
    qty: (e.rel === 'uses' && e.a === SELF && Number(usesQty[e.b]) >= 2) ? Math.floor(Number(usesQty[e.b])) : null,
    designators: (e.rel === 'uses' && e.a === SELF) ? String(usesRefDes[e.b] || '').trim() : '',
  }))
}

// Reset the link-edit baseline to the CURRENT store state — called after a save,
// so a later cancel restores the saved state, not the state from when the modal
// first opened.
function snapshotEdgeState() {
  originalEdges.length = 0
  if (!props.milestone) return
  for (const l of store.links.filter(l => l.a === SELF || l.b === SELF)) {
    originalEdges.push({ a: l.a, b: l.b, rel: l.rel || 'depends-on', version: l.version ?? null, qty: l.qty ?? null, designators: l.designators || '' })
  }
}

// Cancel (embedded edit): throw away everything typed since the last saved state
// and flip back to read mode — form fields, links, pins, quantities, groups.
function cancelEdit() {
  viewVersion.value = null
  applyToForm(props.milestone)
  decodeRefPins()
  edges.value = originalEdges.map(e => ({ ...e }))
  for (const k of Object.keys(usesPins)) delete usesPins[k]
  for (const k of Object.keys(usesQty)) delete usesQty[k]
  for (const k of Object.keys(usesRefDes)) delete usesRefDes[k]
  for (const l of originalEdges) {
    if (l.rel !== 'uses' || l.a !== SELF) continue
    if (l.version != null) usesPins[l.b] = l.version
    if (l.qty != null) usesQty[l.b] = l.qty
    if (l.designators) usesRefDes[l.b] = l.designators
  }
  localGroupIds.value = new Set(props.milestone ? itemGroupIds(props.milestone.id) : [])
  invalidFields.value = []
  proposing.value = false
  proposeNote.value = ''
  editing.value = false
}

function syncLinks(msId) {
  // Diff the resolved working edges against the originals (keyed a|b|rel).
  const key = (e) => `${e.a}|${e.b}|${e.rel}`
  const want = new Map(resolveLinks(msId).map(e => [key(e), e]))
  const orig = new Map(originalEdges.map(e => [key(e), e]))
  for (const [k, e] of want) {
    const o = orig.get(k)
    if (!o || (o.version ?? null) !== (e.version ?? null) || (o.qty ?? null) !== (e.qty ?? null) || (o.designators || '') !== (e.designators || '')) addLink(e.a, e.b, e.rel, e.version, e.qty, e.designators) // new link or version/qty/refdes change → upsert
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

// Header Save (also Enter-to-save): the pop-up saves and closes; the embedded
// view saves in place and flips back to read.
function onSave() {
  if (props.embedded) {
    if (submit(true)) { editing.value = false; proposing.value = false; snapshotEdgeState() }
  } else {
    submit()
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

/* JSON / YAML code view — replaces the FIELDS block AND the tabs. Long documents
   scroll INSIDE the box. Popup: bounded by the modal (60vh). Embedded: exactly
   viewport minus the chrome above it (app header ~96 + panel header ~60 +
   bottom padding 20), so it ends flush with the window edge. */
.ms-code-view { flex: 1; min-height: 260px; max-height: 60vh; overflow: auto; background: var(--clr-bg); border: 1px solid var(--clr-border-light); border-radius: var(--r-md); padding: 12px 14px; }
.ms-code-view.full { flex: 0 0 calc(100vh - 176px); height: calc(100vh - 176px); max-height: none; }
.ms-code { margin: 0; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12.5px; line-height: 1.6; color: var(--clr-text); white-space: pre-wrap; word-break: break-word; }
/* Copy-the-document button — sticky top-right so it stays put while the code scrolls. */
.code-copy { position: sticky; top: 0; float: right; z-index: 2; display: inline-flex; align-items: center; justify-content: center;
  width: 30px; height: 30px; border: 1px solid var(--clr-border); border-radius: 8px; background: var(--clr-surface);
  color: var(--clr-text-2); cursor: pointer; transition: all 0.12s; }
.code-copy:hover { color: var(--clr-text); border-color: var(--clr-accent); }
.code-copy.done { background: #30D158; color: #06310f; border-color: transparent; }

/* Log tab: real version history as terminal lines + the status console below
   (teleport target — the console lives in the Flow diagram component). */
.ms-log { gap: 10px; overflow: hidden; }
.log-hist { flex: 1; min-height: 120px; overflow-y: auto; background: var(--clr-bg); border: 1px solid var(--clr-border-light);
  border-radius: var(--r-md); padding: 10px 14px; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px; line-height: 1.7; }
.log-line { white-space: pre-wrap; word-break: break-word; color: var(--clr-text-3); }
.log-line strong { color: var(--clr-text); font-weight: 700; }
.log-prompt { color: #2e9e4f; user-select: none; }
.log-time { color: var(--clr-text-3); margin: 0 4px; user-select: none; }
.log-gt { color: #2e9e4f; margin-right: 7px; user-select: none; }
.log-none { font-style: italic; }
.log-dim { opacity: 0.6; }
/* Comments in the stream: blue prompt, inline item refs, delete on hover. */
.log-prompt.cmt, .log-gt.cmt { color: #4c8dff; }
.log-prompt.mnt, .log-gt.mnt { color: #b478f0; }
.log-ref { display: inline; padding: 0; border: 0; background: none; font: inherit; color: var(--clr-accent); text-decoration: underline; text-underline-offset: 2px; cursor: pointer; }
.log-ref:hover { filter: brightness(1.25); }
.log-del { display: inline-flex; margin-left: 6px; border: 0; background: none; color: var(--clr-text-3); cursor: pointer; font-size: 13px; padding: 0 3px; border-radius: 4px; opacity: 0; }
.log-line:hover .log-del { opacity: 1; }
.log-del:hover { color: #FF3B30; }
/* The comment prompt (input line) + the /ref item picker above it. */
.log-add { display: flex; align-items: center; width: 100%; text-align: left; border: 0; background: none; padding: 0; margin-top: 2px;
  font: inherit; cursor: text; opacity: 0.55; transition: opacity 0.12s; }
.log-add:hover { opacity: 1; }
.log-add-t { color: var(--clr-text-3); font-style: italic; }
.log-input-row { display: flex; align-items: center; margin-top: 2px; }
.log-input-wrap { position: relative; flex: 1; min-width: 0; display: flex; align-items: center; cursor: text; }
.log-input { flex: 0 0 auto; max-width: 100%; border: 0; background: none; outline: none; color: var(--clr-text); font: inherit; padding: 0; caret-color: transparent; }
/* Same block cursor as the status prompt (1 character cell, VS-Code-terminal style). */
.log-cursor { flex-shrink: 0; display: inline-block; width: 1ch; height: 1.2em; margin-left: 1px; background: var(--clr-text-2); animation: log-blink 1s step-end infinite; }
.log-hint { margin-left: 8px; color: var(--clr-text-3); font-style: italic; opacity: 0.7; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
@keyframes log-blink { 50% { opacity: 0; } }
.log-refpick { position: absolute; bottom: calc(100% + 8px); left: 0; z-index: 20; min-width: 280px; max-height: 240px; overflow-y: auto;
  background: var(--clr-surface); border: 1px solid var(--clr-border); border-radius: 10px; box-shadow: 0 -8px 28px rgba(0, 0, 0, 0.3); padding: 5px; }
.log-refopt { display: flex; align-items: center; gap: 8px; width: 100%; text-align: left; border: 0; background: none; color: var(--clr-text);
  padding: 6px 9px; border-radius: 7px; cursor: pointer; font-size: 12.5px; }
.log-refopt.act { background: color-mix(in srgb, var(--clr-accent) 14%, transparent); }
.log-refopt-t { font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.log-refopt-ty { margin-left: auto; padding-left: 12px; font-size: 11px; color: var(--clr-text-3); flex-shrink: 0; }
/* The docked console loses its own chrome + inner scroll inside the terminal —
   the surrounding .log-hist is the single scroll area. */
.modal-console-dock:empty { display: none; }
.log-hist :deep(.sf-foot),
.log-hist :deep(.sf-foot-docked) { border: 0; padding: 0; margin: 0; background: none; }
.log-hist :deep(.sf-foot-docked .sf-console),
.log-hist :deep(.sf-foot-docked .sf-console.tall) { height: auto; max-height: none; overflow: visible; line-height: 1.7; font-size: 12px; }

.two-col { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }

.ms-tabs { display: flex; gap: 2px; border-bottom: 1px solid var(--clr-border-light); margin: 2px 0 0; }
.ms-tab {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 8px 12px; font-size: 13.5px; font-weight: 600;
  color: var(--clr-text-3); background: none;
  border-bottom: 2px solid transparent; margin-bottom: -1px;
  cursor: pointer; transition: color 0.12s, border-color 0.12s;
}
.ms-tab svg { flex-shrink: 0; opacity: 0.75; }
.ms-tab.active svg { opacity: 1; }
.ms-tab:hover { color: var(--clr-text-2); }
.ms-tab.active { color: var(--clr-accent); border-bottom-color: var(--clr-accent); }
.ms-tab-body { height: 360px; display: flex; flex-direction: column; } /* fixed so the modal is the same height on every tab */
.ms-flow { padding: 0; }
/* History list scrolls inside the panel — keep the rows clear of the scrollbar. */
.ms-history { padding-right: 10px; }

/* Empty-fields toggle in the read card. */
.rf-toggle { align-self: flex-start; font-size: 11.5px; color: var(--clr-text-3); font-style: italic; cursor: pointer; padding: 2px 0; }
.rf-toggle:hover { color: var(--clr-text-2); }

/* View-format overflow menu (header). */
.fmt-menu { position: relative; }
.fmt-bg { position: fixed; inset: 0; z-index: 40; }
.fmt-pop { position: absolute; top: calc(100% + 6px); right: 0; z-index: 41; min-width: 160px; background: var(--clr-surface);
  border: 1px solid var(--clr-border); border-radius: 10px; box-shadow: 0 12px 34px rgba(0, 0, 0, 0.35); padding: 5px; }
.fmt-opt { display: flex; align-items: center; gap: 8px; width: 100%; text-align: left; border: 0; background: none;
  color: var(--clr-text-2); padding: 7px 10px; border-radius: 7px; cursor: pointer; font-size: 12.5px; font-weight: 600; }
.fmt-opt:hover { background: var(--clr-surface-2); color: var(--clr-text); }
.fmt-opt.on { color: var(--clr-accent); }

/* Quick-action popovers (read mode: assignee / progress / maturity). Spans, not
   buttons — real form controls would be disabled by the read-mode fieldset. */
/* No permanent decoration — the value looks normal; hovering reveals a small
   pencil and tints the value, which is the whole "you can click this" cue. */
.read-val.qa { cursor: pointer; }
.read-val.qa:hover { color: var(--clr-accent); }
.qa-pen { display: inline-block; vertical-align: baseline; margin-left: 7px; opacity: 0; color: var(--clr-text-3); transition: opacity 0.12s; }
.read-val.qa:hover .qa-pen { opacity: 1; color: var(--clr-accent); }
.qa-wrap { position: relative; }
.qa-bg { position: fixed; inset: 0; z-index: 40; }
.qa-pop { position: absolute; top: 4px; left: 0; z-index: 41; display: flex; flex-wrap: wrap; gap: 4px; max-width: 340px;
  background: var(--clr-surface); border: 1px solid var(--clr-border); border-radius: 10px;
  box-shadow: 0 12px 34px rgba(0, 0, 0, 0.35); padding: 8px; }
.qa-opt { font-size: 12px; font-weight: 600; color: var(--clr-text-2); background: var(--clr-bg-2); border: 1px solid var(--clr-border);
  border-radius: 7px; padding: 4px 10px; cursor: pointer; transition: all 0.12s; }
.qa-opt:hover { color: var(--clr-text); border-color: var(--clr-accent); }
.qa-opt.on { color: #fff; background: var(--clr-accent); border-color: var(--clr-accent); }
.qa-opt.num { font-variant-numeric: tabular-nums; }
/* Structure tab: the graph/table fills the fixed tab body, no outer scroll. */
.ms-struct { padding: 0; overflow: hidden; }
.ms-struct :deep(.sg) { height: 100%; }
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
.read-pill-ver { font-weight: 500; color: var(--clr-text-3); }
.read-pill-qty { font-weight: 750; font-variant-numeric: tabular-nums; margin-right: 1px; }
.read-none-i { color: var(--clr-text-3); font-size: 13px; font-style: italic; }
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
/* BOM quantity: "6×" before the version dropdown. Empty = 1. */
.picker-qty { flex-shrink: 0; width: 44px; padding: 3px 6px; font-size: 11px; font-variant-numeric: tabular-nums; text-align: right;
  color: var(--clr-text-2); background: var(--clr-bg); border: 1px solid var(--clr-border); border-radius: var(--r-sm); -moz-appearance: textfield; }
.picker-qty::-webkit-outer-spin-button, .picker-qty::-webkit-inner-spin-button { -webkit-appearance: none; margin: 0; }
.picker-qty:focus { outline: none; border-color: var(--clr-accent); }
.picker-qty-x { flex-shrink: 0; margin: 0 6px 0 3px; font-size: 11px; color: var(--clr-text-3); }
/* Reference designators (electronics BOM): free text naming usage positions. */
.picker-refdes { flex-shrink: 0; width: 128px; padding: 3px 6px; font-size: 11px; margin-right: 6px;
  color: var(--clr-text-2); background: var(--clr-bg); border: 1px solid var(--clr-border); border-radius: var(--r-sm); }
.picker-refdes:focus { outline: none; border-color: var(--clr-accent); }
.picker-refdes-warn { flex-shrink: 0; margin-right: 6px; }
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
