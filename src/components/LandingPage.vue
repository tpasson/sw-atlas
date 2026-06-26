<template>
  <div class="lp">
    <!-- top bar — matches the app header (TheHeader.vue) -->
    <header class="header">
      <div class="header-inner">
        <div class="brand">
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
          <span class="brand-tag">explore plans</span>
        </div>

        <div class="header-right">
          <button class="hdr-icon-btn" :title="theme === 'dark' ? 'Light mode' : 'Dark mode'" @click="toggleTheme">
            <Sun v-if="theme === 'dark'" :size="16" />
            <Moon v-else :size="16" />
          </button>
          <template v-if="session.authenticated">
            <button v-if="workspace.ownSlug" class="btn-manage" @click="goTo(workspace.ownSlug)">My plan</button>
            <button class="btn-manage" @click="$emit('logout')">Log out</button>
          </template>
          <button v-else class="btn-manage" @click="$emit('login')">Log in</button>
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
import { Sun, Moon } from 'lucide-vue-next'
import { api } from '../api.js'
import { session, workspace, settings, toggleTheme } from '../stores/useAppStore.js'
import { APP_VERSION } from '../version.js'

defineEmits(['login', 'logout'])

const version = APP_VERSION

const plans = ref([])
const ready = ref(false)
const error = ref(null)
const theme = computed(() => settings.theme)
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
    return () => h('div', { class: 'card', onClick: () => emit('open', props.w.slug) }, [
      h('div', { class: 'card-head' }, [
        h('span', { class: 'card-name' }, props.w.ownerName || props.w.slug),
        props.admin
          ? h('button', {
              class: ['star', { on: props.w.featured }],
              title: props.w.featured ? 'Unfeature' : 'Feature on the landing page',
              onClick: (e) => { e.stopPropagation(); emit('feature', props.w) },
            }, '★')
          : null,
      ]),
      h('span', { class: 'card-slug' }, '/' + props.w.slug),
      h('div', { class: 'card-meta' }, [
        h('span', `${props.w.itemCount} milestone${props.w.itemCount === 1 ? '' : 's'}`),
        props.w.nextDate ? h('span', { class: 'card-next' }, `next · ${fmt(props.w.nextDate)}`) : null,
      ]),
      props.w.nextTitle ? h('span', { class: 'card-title' }, props.w.nextTitle) : null,
    ])
  },
}
</script>

<style scoped>
.lp { min-height: 100vh; display: flex; flex-direction: column; background: var(--clr-bg); }

/* Header — identical to TheHeader.vue so the landing matches the app. */
.header { background: var(--clr-header); position: sticky; top: 0; z-index: 100;
  box-shadow: 0 1px 0 rgba(255,255,255,0.06), var(--sh-md); }
.header-inner { display: flex; align-items: center; justify-content: space-between; padding: 0 24px; height: 64px; gap: 16px; }
.brand { display: flex; align-items: center; gap: 12px; min-width: 0; }
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

/* card (rendered by the inline PlanCard) */
:deep(.card) { cursor: pointer; border: 1px solid var(--clr-border-light); border-radius: var(--r-lg, 12px);
  padding: 14px 16px; background: var(--clr-surface); display: flex; flex-direction: column; gap: 4px;
  transition: border-color 0.15s, transform 0.06s, box-shadow 0.15s; }
:deep(.card:hover) { border-color: var(--clr-accent); box-shadow: 0 2px 12px rgba(0,0,0,0.06); transform: translateY(-1px); }
:deep(.card-head) { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
:deep(.card-name) { font-size: 15px; font-weight: 700; color: var(--clr-text); }
:deep(.card-slug) { font-size: 12px; color: var(--clr-text-3); }
:deep(.card-meta) { display: flex; gap: 8px; flex-wrap: wrap; font-size: 12.5px; color: var(--clr-text-2); margin-top: 4px; }
:deep(.card-next) { color: var(--clr-accent); font-weight: 600; }
:deep(.card-title) { font-size: 12.5px; color: var(--clr-text-3); margin-top: 2px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
:deep(.star) { background: none; font-size: 15px; line-height: 1; color: var(--clr-border); padding: 2px 4px; border-radius: 4px; }
:deep(.star.on) { color: #FFCC00; }
:deep(.star:hover) { color: #FFCC00; }
</style>
