<template>
  <!-- VS-Code-style "Source Control": all Git lives here, not on the timeline.
       Top: repository sources + colour config. Below: the mirrored repo content
       and any milestones that carry a manual Git link. -->
  <div class="scm">
    <div class="scm-head">
      <GitPullRequest :size="18" />
      <h1 class="scm-title">Source Control</h1>
    </div>

    <div class="scm-body">
      <!-- Repository sources + synced-item colours — owner-only configuration. -->
      <section v-if="canAdminWorkspace()" class="scm-sec">
        <GitHubSourceManager />
      </section>

      <!-- Mirrored repository content (was cluttering the timeline as read-only lanes). -->
      <section class="scm-sec">
        <h2 class="scm-h2">Repository content</h2>
        <p v-if="!sourceGroups.length" class="scm-empty">
          No repository connected yet. Add one above to pull its releases, tags, issues and pull requests.
        </p>
        <div v-for="g in sourceGroups" :key="g.id" class="scm-repo">
          <button class="scm-repo-head" @click="toggle(g.id)">
            <svg class="scm-chev" :class="{ open: isOpen(g.id) }" width="10" height="10" viewBox="0 0 10 10" fill="none"><path d="M3 1.5L6.5 5L3 8.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
            <span class="scm-dot" :style="{ background: g.color || '#8a8a8e' }"></span>
            <span class="scm-repo-name">{{ g.name }}</span>
            <span class="scm-repo-count">{{ g.items.length }}</span>
          </button>
          <ul v-if="isOpen(g.id)" class="scm-list">
            <li v-for="m in g.items" :key="m.id" class="scm-row">
              <span class="scm-status" :style="{ background: m.color || '#8a8a8e' }" :title="m.progress != null ? m.progress + '%' : ''"></span>
              <a v-if="m.scmUrl" :href="m.scmUrl" target="_blank" rel="noopener noreferrer" class="scm-row-link"><ScmBadge :url="m.scmUrl" /></a>
              <span class="scm-row-title">{{ m.title }}</span>
            </li>
            <li v-if="!g.items.length" class="scm-row scm-row-empty">— nothing synced yet —</li>
          </ul>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { GitPullRequest } from 'lucide-vue-next'
import { store, canAdminWorkspace } from '../stores/useAppStore.js'
import GitHubSourceManager from './GitHubSourceManager.vue'
import ScmBadge from './ScmBadge.vue'

defineProps({ readOnly: { type: Boolean, default: false } })

// One group per connected repository (source swimlane), with its mirrored items.
const sourceGroups = computed(() =>
  store.swimlanes
    .filter(sw => sw.sourceSystem)
    .map(sw => ({
      id: sw.id,
      name: sw.name,
      color: sw.color,
      items: store.milestones.filter(m => m.swimlaneId === sw.id),
    })))

// Collapse/expand repositories (open by default; the set tracks closed ones).
const collapsed = ref(new Set())
function isOpen(id) { return !collapsed.value.has(id) }
function toggle(id) {
  const s = new Set(collapsed.value)
  s.has(id) ? s.delete(id) : s.add(id)
  collapsed.value = s
}
</script>

<style scoped>
.scm { flex: 1; min-height: 0; display: flex; flex-direction: column; }
.scm-head { display: flex; align-items: center; gap: 9px; padding: 16px 24px 12px; border-bottom: 1px solid var(--clr-border-light); color: var(--clr-text); flex-shrink: 0; }
.scm-title { font-size: 17px; font-weight: 700; }
.scm-body { flex: 1; min-height: 0; overflow-y: auto; padding: 20px 24px 40px; }

.scm-sec { max-width: 760px; margin: 0 auto 28px; }
.scm-h2 { font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-text-3); margin-bottom: 10px; }
.scm-empty { font-size: 13px; color: var(--clr-text-3); line-height: 1.5; }

.scm-repo { border: 1px solid var(--clr-border-light); border-radius: var(--r-md); margin-bottom: 8px; overflow: hidden; }
.scm-repo-head { display: flex; align-items: center; gap: 8px; width: 100%; text-align: left; padding: 9px 12px; background: var(--clr-surface-2); }
.scm-repo-head:hover { background: var(--clr-surface); }
.scm-chev { color: var(--clr-text-3); flex-shrink: 0; transition: transform 0.12s; }
.scm-chev.open { transform: rotate(90deg); }
.scm-dot { width: 9px; height: 9px; border-radius: 50%; flex-shrink: 0; }
.scm-repo-name { font-weight: 600; font-size: 13px; color: var(--clr-text); }
.scm-repo-count { margin-left: auto; font-size: 10px; color: var(--clr-text-3); background: var(--clr-bg); border-radius: 100px; padding: 1px 8px; }

.scm-list { list-style: none; }
.scm-row { display: flex; align-items: center; gap: 10px; padding: 7px 14px; border-top: 1px solid var(--clr-border-light); }
.scm-repo .scm-row:first-child { border-top: none; }
.scm-row-clickable { cursor: pointer; border: 1px solid var(--clr-border-light); border-radius: var(--r-md); margin-bottom: 6px; }
.scm-row-clickable:hover { background: var(--clr-surface-2); }
.scm-status { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.scm-row-link { flex-shrink: 0; text-decoration: none; }
.scm-row-title { font-size: 13px; color: var(--clr-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.scm-row-where { margin-left: auto; font-size: 11px; color: var(--clr-text-3); flex-shrink: 0; }
.scm-row-empty { font-size: 12px; color: var(--clr-text-3); }
</style>
