<template>
  <div class="lp">
    <!-- top bar -->
    <header class="lp-top">
      <div class="lp-brand">
        <span class="lp-logo">▦</span>
        <span class="lp-title">ATLAS</span>
        <span class="lp-tag">explore plans</span>
      </div>
      <div class="lp-actions">
        <button class="lp-btn ghost" :title="theme === 'dark' ? 'Light mode' : 'Dark mode'" @click="toggleTheme">{{ theme === 'dark' ? '☀' : '☾' }}</button>
        <template v-if="session.authenticated">
          <button v-if="workspace.ownSlug" class="lp-btn" @click="goTo(workspace.ownSlug)">My plan</button>
          <button class="lp-btn ghost" @click="$emit('logout')">Log out</button>
        </template>
        <button v-else class="lp-btn" @click="$emit('login')">Log in</button>
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
import { api } from '../api.js'
import { session, workspace, settings, toggleTheme } from '../stores/useAppStore.js'

defineEmits(['login', 'logout'])

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
.lp-top { display: flex; align-items: center; justify-content: space-between;
  padding: 14px 24px; border-bottom: 1px solid var(--clr-border-light); }
.lp-brand { display: flex; align-items: baseline; gap: 8px; }
.lp-logo { font-size: 18px; color: var(--clr-accent); }
.lp-title { font-size: 18px; font-weight: 800; letter-spacing: 0.5px; color: var(--clr-text); }
.lp-tag { font-size: 12.5px; color: var(--clr-text-3); }
.lp-actions { display: flex; gap: 8px; align-items: center; }
.lp-btn { padding: 7px 14px; font-size: 13px; font-weight: 600; color: #fff;
  background: var(--clr-accent); border-radius: var(--r-md); transition: background 0.15s; }
.lp-btn:hover { background: var(--clr-accent-hover); }
.lp-btn.ghost { color: var(--clr-text-2); background: transparent; border: 1px solid var(--clr-border-light); }
.lp-btn.ghost:hover { background: var(--clr-surface); }
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
