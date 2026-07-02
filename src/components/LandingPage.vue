<template>
  <div class="lp">
    <!-- top bar — matches the app header (TheHeader.vue) -->
    <header class="header">
      <div class="header-inner">
        <div class="brand">
          <button class="brand-btn" title="About ATLAS" @click="$emit('about')">
            <div class="brand-icon">
              <svg width="22" height="22" viewBox="0 0 22 22" fill="none">
                <rect x="2" y="2" width="8" height="8" rx="2" fill="white" opacity="0.9"/>
                <rect x="12" y="2" width="8" height="8" rx="2" fill="white" opacity="0.6"/>
                <rect x="2" y="12" width="8" height="8" rx="2" fill="white" opacity="0.6"/>
                <rect x="12" y="12" width="8" height="8" rx="2" fill="white" opacity="0.35"/>
              </svg>
            </div>
            <div class="brand-text">
              <span class="brand-title">ATLAS</span>
              <span class="brand-ver">v{{ version }}</span>
            </div>
          </button>
          <span class="brand-tag">explore plans</span>
        </div>

        <div class="header-right">
          <!-- Same plan switcher as the in-plan header: jump to your area, projects
               or any public plan straight from the discovery landing. -->
          <PlanSwitcher />
        </div>
      </div>
    </header>

    <main class="lp-body">
      <div v-if="!ready" class="lp-empty">Loading…</div>
      <div v-else-if="error" class="lp-empty">{{ error }}</div>
      <template v-else>
        <section v-if="featured.length" class="lp-section">
          <h2 class="lp-h">★ Featured</h2>
          <div class="lp-grid">
            <PlanCard v-for="w in featured" :key="w.slug" :w="w" :admin="isAdmin" @open="goTo" @feature="toggleFeatured" />
          </div>
        </section>

        <section class="lp-section">
          <h2 class="lp-h">{{ featured.length ? 'All public plans' : 'Public plans' }}</h2>
          <div v-if="others.length" class="lp-grid">
            <PlanCard v-for="w in others" :key="w.slug" :w="w" :admin="isAdmin" @open="goTo" @feature="toggleFeatured" />
          </div>
          <div v-else class="lp-empty">No public plans yet. Log in, build a plan and turn on “Public read access” to list it here.</div>
        </section>
      </template>
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, h } from 'vue'
import { User } from 'lucide-vue-next'
import { api } from '../api.js'
import { session, workspace, openProfile, personName } from '../stores/useAppStore.js'
import { APP_VERSION } from '../version.js'
import PlanSwitcher from './PlanSwitcher.vue'

defineEmits(['about'])

const version = APP_VERSION

const plans = ref([])
const ready = ref(false)
const error = ref(null)
const isAdmin = computed(() => session.role === 'admin')

const featured = computed(() => plans.value.filter(w => w.featured))
const others = computed(() => plans.value.filter(w => !w.featured))

function goTo(slug) {
  window.location.assign('/' + encodeURIComponent(slug))
}

async function load() {
  try { plans.value = (await api.listPublicWorkspaces()).workspaces || [] }
  catch (e) { error.value = e.message || 'Failed to load' }
  ready.value = true
}
onMounted(load)

async function toggleFeatured(w) {
  try { await api.setWorkspaceFeatured(w.slug, !w.featured); w.featured = !w.featured }
  catch (e) { error.value = e.message || 'Could not change featured state' }
}

// Small inline card component (keeps the file self-contained).
const PlanCard = {
  props: { w: Object, admin: Boolean },
  emits: ['open', 'feature'],
  setup(props, { emit }) {
    const fmt = (d) => { try { return new Date(d).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }) } catch { return d } }
    const fmtStamp = (d) => { try { return new Date(d).toLocaleString(undefined, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' }) } catch { return d } }
    return () => {
      const w = props.w
      const late = w.lateCount || 0
      const kids = [
        h('div', { class: 'card-head' }, [
          h('div', { class: 'card-headtext' }, [
            h('span', { class: 'card-name' }, w.name || w.slug),
            h('span', { class: 'card-kind' }, w.personal ? 'Private schedule' : 'Project schedule'),
          ]),
          props.admin
            ? h('button', {
                class: ['star', { on: w.featured }],
                title: w.featured ? 'Unfeature' : 'Feature on the landing page',
                onClick: (e) => { e.stopPropagation(); emit('feature', w) },
              }, '★')
            : null,
        ]),
      ]
      if (w.ownerName) {
        const owner = { username: w.ownerName, firstName: w.ownerFirstName, lastName: w.ownerLastName, email: w.ownerEmail }
        kids.push(h('div', {
          class: 'card-owner',
          title: 'View profile',
          onClick: (e) => { e.stopPropagation(); openProfile(owner, e) },
        }, [h(User, { size: 12 }), h('span', personName(owner))]))
      }
      if (w.itemCount > 0) {
        kids.push(h('div', { class: ['card-status', late ? 'late' : 'ok'] }, [
          h('span', { class: 'dot' }),
          late ? `${late} item${late === 1 ? '' : 's'} late` : 'On track',
        ]))
      }
      kids.push(h('div', { class: 'card-meta' }, [
        h('span', `${w.itemCount} item${w.itemCount === 1 ? '' : 's'}`),
        w.nextDate ? h('span', { class: 'card-next' }, `next · ${fmt(w.nextDate)}`) : null,
      ]))
      if (w.nextTitle) kids.push(h('span', { class: 'card-title' }, w.nextTitle))
      if (w.lastChange) kids.push(h('span', { class: 'card-updated' }, `Updated ${fmtStamp(w.lastChange)}`))
      return h('div', { class: 'card', onClick: () => emit('open', w.slug) }, kids)
    }
  },
}
</script>

<style scoped>
.lp { flex: 1; min-height: 0; display: flex; flex-direction: column; background: var(--clr-bg); overflow-y: auto; }

/* Header — identical to TheHeader.vue so the landing matches the app. */
.header { background: var(--clr-header); position: sticky; top: 0; z-index: 100;
  box-shadow: 0 1px 0 rgba(255,255,255,0.06), var(--sh-md); }
.header-inner { display: flex; align-items: center; justify-content: space-between; padding: 0 24px; height: 64px; gap: 16px; }
.brand { display: flex; align-items: center; gap: 12px; min-width: 0; }
.brand-btn { display: flex; align-items: center; gap: 12px; background: none; padding: 4px; margin: -4px; border-radius: 10px; transition: background 0.15s; }
.brand-btn:hover { background: rgba(255,255,255,0.07); }
.brand-icon { width: 38px; height: 38px; background: rgba(255,255,255,0.08); border-radius: 10px;
  display: flex; align-items: center; justify-content: center; border: 1px solid rgba(255,255,255,0.1); flex-shrink: 0; }
.brand-text { display: flex; flex-direction: column; gap: 1px; }
.brand-title { font-size: 15px; font-weight: 600; color: #FFFFFF; letter-spacing: -0.2px; }
.brand-ver { font-size: 10px; font-weight: 500; color: rgba(255,255,255,0.4); letter-spacing: 0.3px; }
.brand-tag { font-size: 12.5px; color: rgba(255,255,255,0.5); padding-left: 12px; border-left: 1px solid rgba(255,255,255,0.12); }
.header-right { display: flex; align-items: center; gap: 8px; }
.hdr-icon-btn { width: 32px; height: 32px; flex-shrink: 0; display: inline-flex; align-items: center; justify-content: center;
  border-radius: 100px; color: rgba(255,255,255,0.8); background: rgba(255,255,255,0.08);
  border: 1px solid rgba(255,255,255,0.12); transition: background 0.15s, color 0.15s; }
.hdr-icon-btn:hover { background: rgba(255,255,255,0.14); color: #FFFFFF; }
.btn-manage { display: inline-flex; align-items: center; gap: 7px; height: 32px; box-sizing: border-box; padding: 0 14px;
  font-size: 13px; font-weight: 500; color: rgba(255,255,255,0.85); background: rgba(255,255,255,0.08);
  border: 1px solid rgba(255,255,255,0.12); border-radius: 100px; transition: background 0.15s, color 0.15s, border-color 0.15s; }
.btn-manage:hover { background: rgba(255,255,255,0.14); color: #FFFFFF; border-color: rgba(255,255,255,0.2); }

.lp-body { flex: 1; max-width: 1080px; width: 100%; margin: 0 auto; padding: 28px 24px 48px; }
.lp-section { margin-bottom: 32px; }
.lp-h { font-size: 13px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.6px;
  color: var(--clr-text-3); margin: 0 0 12px; }
.lp-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 14px; }
.lp-empty { font-size: 13.5px; color: var(--clr-text-3); padding: 12px 0; }

/* card (rendered by the inline PlanCard) — soft, elevated, Apple-like */
:deep(.card) { cursor: pointer; border: 1px solid var(--clr-border-light); border-radius: 18px;
  padding: 18px 18px 16px; background: var(--clr-surface); display: flex; flex-direction: column; gap: 10px;
  box-shadow: 0 1px 2px rgba(0,0,0,0.04);
  transition: transform 0.2s cubic-bezier(0.4,0,0.2,1), box-shadow 0.2s ease, border-color 0.2s ease; }
:deep(.card:hover) { transform: translateY(-3px); border-color: var(--clr-border);
  box-shadow: 0 10px 30px rgba(0,0,0,0.12); }
:deep(.card:active) { transform: translateY(-1px); }
:deep(.card-head) { display: flex; align-items: flex-start; justify-content: space-between; gap: 8px; }
:deep(.card-headtext) { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
:deep(.card-name) { font-size: 16px; font-weight: 650; letter-spacing: -0.25px; color: var(--clr-text);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
:deep(.card-kind) { font-size: 10.5px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-text-3); }
:deep(.card-owner) { display: inline-flex; align-items: center; gap: 5px; align-self: flex-start;
  font-size: 12.5px; color: var(--clr-text-2); cursor: pointer; border-radius: 6px; padding: 1px 4px; margin: -1px -4px; transition: background 0.15s; }
:deep(.card-owner:hover) { background: var(--clr-surface-2); color: var(--clr-text); }
:deep(.card-owner svg) { color: var(--clr-text-3); flex-shrink: 0; }

/* status pill: On track (green) / N late (amber) */
:deep(.card-status) { display: inline-flex; align-items: center; gap: 6px; align-self: flex-start;
  font-size: 12px; font-weight: 600; padding: 3px 11px 3px 8px; border-radius: 999px; }
:deep(.card-status .dot) { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
:deep(.card-status.ok) { color: var(--clr-success); background: rgba(52,199,89,0.13); }
:deep(.card-status.ok .dot) { background: #34C759; }
:deep(.card-status.late) { color: var(--clr-warn); background: rgba(255,149,0,0.15); }
:deep(.card-status.late .dot) { background: #FF9500; }

:deep(.card-meta) { display: flex; gap: 10px; flex-wrap: wrap; font-size: 12.5px; color: var(--clr-text-2); }
:deep(.card-next) { color: var(--clr-accent); font-weight: 600; }
:deep(.card-title) { font-size: 12.5px; color: var(--clr-text-3);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
:deep(.card-updated) { font-size: 11px; color: var(--clr-text-3); letter-spacing: 0.1px; margin-top: 1px; }
:deep(.star) { background: none; font-size: 15px; line-height: 1; color: var(--clr-border); padding: 2px 4px; border-radius: 4px; }
:deep(.star.on) { color: #FFCC00; }
:deep(.star:hover) { color: #FFCC00; }
</style>
