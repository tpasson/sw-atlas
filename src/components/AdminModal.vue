<template>
  <Teleport to="body">
    <Transition name="modal">
      <div class="backdrop">
        <div class="am-panel">
          <div class="am-header">
            <h2 class="am-title"><ServerCog :size="18" /> Admin</h2>
            <button class="am-close" title="Close" @click="$emit('close')"><X :size="18" /></button>
          </div>
          <div class="am-tabs">
            <button class="am-tab" :class="{ active: tab === 'display' }" @click="tab = 'display'">Display</button>
            <button class="am-tab" :class="{ active: tab === 'server' }" @click="tab = 'server'">Server</button>
            <button class="am-tab" :class="{ active: tab === 'users' }" @click="tab = 'users'">Users</button>
          </div>
          <div class="am-body">
            <div v-show="tab === 'display'">
              <p class="am-note">These display settings apply to <strong>every dashboard</strong> on this instance. Changes save automatically.</p>
              <DisplaySettings />
            </div>
            <ServerSettings v-if="tab === 'server'" />
            <UsersManager v-if="tab === 'users'" />
          </div>
          <div class="am-footer">
            <button class="btn-primary" @click="$emit('close')">Done</button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref } from 'vue'
import { X, ServerCog } from 'lucide-vue-next'
import DisplaySettings from './DisplaySettings.vue'
import ServerSettings from './ServerSettings.vue'
import UsersManager from './UsersManager.vue'

defineEmits(['close'])
const tab = ref('display')
</script>

<style scoped>
.backdrop {
  position: fixed; inset: 0; background: rgba(0,0,0,0.45);
  backdrop-filter: blur(4px); -webkit-backdrop-filter: blur(4px);
  z-index: 1000; display: flex; align-items: flex-start; justify-content: center; padding: 40px 24px 24px;
}
.am-panel {
  background: var(--clr-surface); border-radius: var(--r-xl); width: 100%; max-width: 640px;
  max-height: calc(100vh - 64px); box-shadow: var(--sh-modal); display: flex; flex-direction: column; overflow: hidden;
}
.am-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 18px 20px 14px; border-bottom: 1px solid var(--clr-border-light);
}
.am-title { display: flex; align-items: center; gap: 9px; font-size: 18px; font-weight: 700; letter-spacing: -0.3px; color: var(--clr-text); }
.am-close { width: 30px; height: 30px; display: flex; align-items: center; justify-content: center;
  background: var(--clr-surface-2); border-radius: 50%; color: var(--clr-text-2); transition: background 0.15s; }
.am-close:hover { background: var(--clr-border-light); }
.am-tabs { display: flex; gap: 2px; padding: 8px 16px 0; border-bottom: 1px solid var(--clr-border-light); }
.am-tab { padding: 8px 14px; font-size: 13.5px; font-weight: 600; color: var(--clr-text-3);
  border-bottom: 2px solid transparent; margin-bottom: -1px; transition: color 0.15s; background: none; }
.am-tab:hover { color: var(--clr-text); }
.am-tab.active { color: var(--clr-accent); border-bottom-color: var(--clr-accent); }
.am-body { flex: 1; min-height: 0; overflow-y: auto; padding: 18px 20px; }
.am-note { font-size: 12px; color: var(--clr-text-3); margin: 0 0 12px; line-height: 1.5; }
.am-footer { display: flex; justify-content: flex-end; padding: 14px 20px; border-top: 1px solid var(--clr-border-light); }
.btn-primary { padding: 9px 20px; font-size: 14px; font-weight: 600; color: #fff; background: var(--clr-accent); border-radius: var(--r-md); transition: background 0.15s; }
.btn-primary:hover { background: var(--clr-accent-hover); }
</style>
