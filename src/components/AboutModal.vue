<template>
  <Teleport to="body">
    <Transition name="modal">
      <div class="backdrop" @click.self="$emit('close')">
        <Transition name="modal-panel" appear>
          <div class="panel">
            <button class="btn-close" @click="$emit('close')">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <path d="M1 1l12 12M13 1L1 13" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/>
              </svg>
            </button>

            <div class="about-head">
              <div class="about-icon">
                <svg width="26" height="26" viewBox="0 0 22 22" fill="none">
                  <rect x="2" y="2" width="8" height="8" rx="2" fill="white" opacity="0.95"/>
                  <rect x="12" y="2" width="8" height="8" rx="2" fill="white" opacity="0.65"/>
                  <rect x="2" y="12" width="8" height="8" rx="2" fill="white" opacity="0.65"/>
                  <rect x="12" y="12" width="8" height="8" rx="2" fill="white" opacity="0.4"/>
                </svg>
              </div>
              <div>
                <div class="about-name">ATLAS</div>
                <div class="about-ver">Version {{ version }}</div>
              </div>
            </div>

            <p class="about-desc">
              ATLAS is a free, open-source milestone &amp; roadmap planner with date-anchored
              timelines, events, baselines and groups.
            </p>
            <p class="about-desc">
              Everyone works from the same view of what's happening when: C-level management
              get the big picture at a glance, while program leads, project leads and
              engineering teams keep the details on track.
            </p>

            <div class="about-rows">
              <div class="about-row">
                <span class="ar-k">License</span>
                <span class="ar-v">Open source · Apache-2.0</span>
              </div>
              <div class="about-row">
                <span class="ar-k">Developer</span>
                <span class="ar-v">Thomas Passon</span>
              </div>
              <div class="about-row">
                <span class="ar-k">Built with</span>
                <span class="ar-v">Go · Vue · PostgreSQL · Docker</span>
              </div>
              <div class="about-row">
                <span class="ar-k">Source</span>
                <a class="ar-link" :href="repoUrl" target="_blank" rel="noopener">{{ repoUrl.replace('https://', '') }}</a>
              </div>
              <div class="about-row">
                <span class="ar-k">Feedback</span>
                <a class="ar-link" :href="issuesUrl" target="_blank" rel="noopener">Request a feature or report an issue</a>
              </div>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { APP_VERSION, REPO_URL } from '../version.js'

defineEmits(['close'])
const version = APP_VERSION
const repoUrl = REPO_URL
const issuesUrl = REPO_URL + '/issues'
</script>

<style scoped>
.backdrop {
  position: fixed; inset: 0;
  background: rgba(0,0,0,0.45);
  backdrop-filter: blur(4px); -webkit-backdrop-filter: blur(4px);
  z-index: 1000;
  display: flex; align-items: center; justify-content: center;
  padding: 24px;
}
.panel {
  position: relative;
  background: var(--clr-surface);
  border-radius: var(--r-xl);
  width: 100%; max-width: 430px;
  box-shadow: var(--sh-modal);
  padding: 28px;
}
.btn-close {
  position: absolute; top: 16px; right: 16px;
  width: 30px; height: 30px;
  display: flex; align-items: center; justify-content: center;
  background: var(--clr-surface-2);
  border-radius: 50%;
  color: var(--clr-text-2);
  transition: background 0.15s;
}
.btn-close:hover { background: var(--clr-border-light); }

.about-head { display: flex; align-items: center; gap: 14px; margin-bottom: 16px; }
.about-icon {
  width: 48px; height: 48px;
  border-radius: 12px;
  background: var(--clr-accent);
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.about-name { font-size: 20px; font-weight: 700; letter-spacing: -0.3px; color: var(--clr-text); }
.about-ver { font-size: 13px; color: var(--clr-text-3); margin-top: 1px; }

.about-desc { font-size: 13.5px; color: var(--clr-text-2); line-height: 1.5; margin-bottom: 18px; text-align: justify; }

.about-rows { display: flex; flex-direction: column; gap: 10px; }
.about-row { display: grid; grid-template-columns: 86px 1fr; gap: 10px; align-items: baseline; }
.ar-k { font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--clr-text-3); }
.ar-v { font-size: 13px; color: var(--clr-text); }
.ar-link { font-size: 13px; color: var(--clr-accent); text-decoration: none; word-break: break-all; }
.ar-link:hover { text-decoration: underline; }
</style>
