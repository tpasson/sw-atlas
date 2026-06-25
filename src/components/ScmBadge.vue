<script setup>
import { computed } from 'vue'
import { parseScmUrl } from '../stores/useAppStore.js'

const props = defineProps({
  url: { type: String, default: '' },
})

const info = computed(() => parseScmUrl(props.url))

const TYPE_LABEL = {
  release: 'Release', tag: 'Tag', pr: 'PR', issue: 'Issue',
  commit: 'Commit', branch: 'Branch', file: 'File', repo: 'Repo', link: 'Link',
}
const PROVIDER_LABEL = {
  gitlab: 'GitLab', azure: 'Azure DevOps', gitea: 'Gitea', bitbucket: 'Bitbucket',
}
// GitHub has its own recognizable mark; the rest share a generic git glyph, so we
// name them explicitly. The unknown 'git' fallback stays unlabelled.
const providerLabel = computed(() => PROVIDER_LABEL[info.value?.provider] || '')
</script>

<template>
  <a
    v-if="info"
    class="scm-badge"
    :href="info.url"
    target="_blank"
    rel="noopener noreferrer"
    :title="info.url"
    @click.stop
  >
    <!-- GitHub mark -->
    <svg v-if="info.provider === 'github'" class="scm-ico" viewBox="0 0 16 16" width="13" height="13" fill="currentColor" aria-hidden="true">
      <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82a7.6 7.6 0 0 1 2-.27c.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8z"/>
    </svg>
    <!-- GitLab / generic git mark -->
    <svg v-else class="scm-ico" viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <line x1="6" y1="3" x2="6" y2="15"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M18 9a9 9 0 0 1-9 9"/>
    </svg>
    <span v-if="providerLabel" class="scm-prov">{{ providerLabel }}</span>
    <span class="scm-repo">{{ info.repo }}</span>
    <span class="scm-ref">{{ (TYPE_LABEL[info.type] || 'Link') + (info.ref ? ' ' + info.ref : '') }}</span>
  </a>
</template>

<style scoped>
.scm-badge {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  max-width: 100%;
  padding: 4px 9px;
  border: 1px solid var(--clr-border);
  border-radius: 100px;
  background: var(--clr-bg);
  font-size: 12px;
  color: var(--clr-text-2);
  text-decoration: none;
  transition: border-color 0.12s, background 0.12s;
}
.scm-badge:hover { border-color: var(--clr-accent); background: var(--clr-surface); }
.scm-ico { flex-shrink: 0; color: var(--clr-text); }
.scm-prov { flex-shrink: 0; font-weight: 600; color: var(--clr-text-2); }
.scm-prov::after { content: '·'; margin-left: 7px; color: var(--clr-text-3); font-weight: 400; }
.scm-repo {
  font-weight: 600;
  color: var(--clr-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.scm-ref {
  flex-shrink: 0;
  font-weight: 600;
  font-size: 11px;
  padding: 1px 7px;
  border-radius: 100px;
  background: var(--clr-surface-2);
  color: var(--clr-text-2);
  white-space: nowrap;
}
</style>
