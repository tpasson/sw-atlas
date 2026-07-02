<template>
  <Teleport to="body">
    <Transition name="modal">
      <div class="backdrop">
        <Transition name="modal-panel" appear>
          <div class="panel">
            <div class="panel-header">
              <h2 class="panel-title">Login</h2>
              <button class="btn-close" @click="$emit('close')">
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                  <path d="M1 1l12 12M13 1L1 13" stroke="currentColor" stroke-width="1.75" stroke-linecap="round"/>
                </svg>
              </button>
            </div>

            <form class="panel-form" @submit.prevent="submit">
              <div class="field">
                <label class="field-label">Username</label>
                <input v-model="username" class="field-input" placeholder="Username" autocomplete="username" ref="userInput" />
              </div>
              <div class="field">
                <label class="field-label">Password</label>
                <input v-model="password" type="password" class="field-input" autocomplete="current-password" />
              </div>

              <p v-if="error" class="form-error">{{ error }}</p>

              <div class="panel-actions">
                <button type="button" class="btn-ghost" @click="$emit('close')">Cancel</button>
                <button type="submit" class="btn-primary" :disabled="busy">
                  {{ busy ? 'Signing in…' : 'Log in' }}
                </button>
              </div>
            </form>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAppStore } from '../stores/useAppStore.js'

const emit = defineEmits(['close'])
const { login } = useAppStore()

const username = ref('')
const password = ref('')
const error = ref('')
const busy = ref(false)
const userInput = ref(null)

onMounted(() => userInput.value?.focus())

async function submit() {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    await login(username.value, password.value)
    emit('close')
  } catch (e) {
    error.value = e.status === 401 ? 'Invalid credentials' : (e.message || 'Login failed')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.backdrop {
  position: fixed; inset: 0;
  background: rgba(0,0,0,0.45);
  backdrop-filter: blur(4px); -webkit-backdrop-filter: blur(4px);
  z-index: 1000; display: flex; align-items: center; justify-content: center; padding: 24px;
}
.panel {
  background: var(--clr-surface); border-radius: var(--r-xl);
  width: 100%; max-width: 380px; box-shadow: var(--sh-modal);
  display: flex; flex-direction: column; overflow: hidden;
}
.panel-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 20px 20px 16px; border-bottom: 1px solid var(--clr-border-light);
}
.panel-title { font-size: 18px; font-weight: 700; letter-spacing: -0.3px; color: var(--clr-text); }
.btn-close {
  width: 30px; height: 30px; display: flex; align-items: center; justify-content: center;
  background: var(--clr-surface-2); border-radius: 50%; color: var(--clr-text-2); transition: background 0.15s;
}
.btn-close:hover { background: var(--clr-border-light); }
.panel-form { padding: 18px 20px 20px; display: flex; flex-direction: column; gap: 14px; }
.field { display: flex; flex-direction: column; gap: 5px; }
.field-label {
  font-size: 11.5px; font-weight: 600; color: var(--clr-text-2);
  text-transform: uppercase; letter-spacing: 0.4px;
}
.field-input {
  border: 1.5px solid var(--clr-border); border-radius: var(--r-md);
  padding: 9px 12px; font-size: 14px; color: var(--clr-text); background: var(--clr-bg);
  outline: none; width: 100%; transition: border-color 0.15s, box-shadow 0.15s;
}
.field-input:focus {
  border-color: var(--clr-accent); box-shadow: 0 0 0 3px rgba(0,113,227,0.12); background: var(--clr-surface);
}
.form-error { font-size: 13px; color: var(--clr-danger); margin: -4px 0 0; }
.panel-actions {
  display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px;
  padding-top: 16px; border-top: 1px solid var(--clr-border-light);
}
.btn-primary {
  padding: 9px 20px; font-size: 14px; font-weight: 600; color: #fff;
  background: var(--clr-accent); border-radius: var(--r-md); transition: background 0.15s;
}
.btn-primary:hover:not(:disabled) { background: var(--clr-accent-hover); }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-ghost {
  padding: 9px 16px; font-size: 14px; font-weight: 500; color: var(--clr-text-2);
  background: transparent; border-radius: var(--r-md); transition: background 0.15s;
}
.btn-ghost:hover { background: var(--clr-surface-2); }
</style>
