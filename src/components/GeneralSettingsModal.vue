<template>
  <Teleport to="body">
    <Transition name="modal">
      <div class="backdrop">
        <div class="gs-panel">
          <div class="gs-header">
            <h2 class="gs-title">Settings</h2>
            <button class="gs-close" title="Close" @click="$emit('close')"><X :size="18" /></button>
          </div>
          <div class="gs-body">
            <div class="card">
              <p class="section-label">Appearance</p>
              <p class="card-hint">Your personal preference — it only affects your browser.</p>
              <div class="gs-row">
                <span class="gs-row-label">Dark mode</span>
                <button class="gs-toggle" :class="{ on: isDark }" role="switch" :aria-checked="isDark" @click="toggleTheme">
                  <span class="gs-knob"></span>
                </button>
              </div>
            </div>

            <AccountManager v-if="session.authenticated" />
            <div v-else class="card">
              <p class="section-label">Account</p>
              <p class="card-hint">Log in to manage your account, profile and password.</p>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed } from 'vue'
import { X } from 'lucide-vue-next'
import AccountManager from './AccountManager.vue'
import { session, settings, toggleTheme } from '../stores/useAppStore.js'

defineEmits(['close'])
const isDark = computed(() => settings.theme === 'dark')
</script>

<style scoped>
.backdrop {
  position: fixed; inset: 0; background: rgba(0,0,0,0.45);
  backdrop-filter: blur(4px); -webkit-backdrop-filter: blur(4px);
  z-index: 1000; display: flex; align-items: flex-start; justify-content: center; padding: 60px 24px 24px; overflow-y: auto;
}
.gs-panel {
  background: var(--clr-surface); border-radius: var(--r-xl);
  width: 100%; max-width: 460px; box-shadow: var(--sh-modal);
  display: flex; flex-direction: column; overflow: hidden;
}
.gs-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 18px 20px 14px; border-bottom: 1px solid var(--clr-border-light);
}
.gs-title { font-size: 18px; font-weight: 700; letter-spacing: -0.3px; color: var(--clr-text); }
.gs-close {
  width: 30px; height: 30px; display: flex; align-items: center; justify-content: center;
  background: var(--clr-surface-2); border-radius: 50%; color: var(--clr-text-2); transition: background 0.15s;
}
.gs-close:hover { background: var(--clr-border-light); }
.gs-body { padding: 16px 20px 20px; display: flex; flex-direction: column; gap: 14px; }
.gs-row { display: flex; align-items: center; justify-content: space-between; margin-top: 10px; }
.gs-row-label { font-size: 14px; font-weight: 500; color: var(--clr-text); }
.gs-toggle {
  width: 44px; height: 26px; border-radius: 100px; background: var(--clr-border);
  position: relative; transition: background 0.18s; flex-shrink: 0;
}
.gs-toggle.on { background: var(--clr-accent); }
.gs-knob {
  position: absolute; top: 3px; left: 3px; width: 20px; height: 20px; border-radius: 50%;
  background: #fff; box-shadow: 0 1px 3px rgba(0,0,0,0.3); transition: transform 0.18s;
}
.gs-toggle.on .gs-knob { transform: translateX(18px); }
</style>
