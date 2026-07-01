<template>
  <Teleport to="body">
    <div v-if="p.person" class="uprof-backdrop" @click="close" @contextmenu.prevent="close">
      <div class="uprof-pop" :style="popStyle" @click.stop>
        <div class="uprof">
          <div class="uprof-av">{{ initials }}</div>
          <div class="uprof-body">
            <div class="uprof-name">{{ name }}</div>
            <div v-if="showUser" class="uprof-user">@{{ p.person.username }}</div>
            <a v-if="p.person.email" class="uprof-email" :href="'mailto:' + p.person.email">{{ p.person.email }}</a>
            <div v-else-if="!authed" class="uprof-hint">Log in to see contact details</div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed } from 'vue'
import { profilePopover as p, closeProfile, personName, personInitials, session } from '../stores/useAppStore.js'

const name = computed(() => personName(p.person))
const initials = computed(() => personInitials(p.person))
const showUser = computed(() => p.person && name.value !== p.person.username)
const authed = computed(() => session.authenticated)
function close() { closeProfile() }

const popStyle = computed(() => {
  const w = 260, h = 130
  const x = Math.min(Math.max(8, p.x), window.innerWidth - w - 8)
  const y = Math.min(p.y + 10, window.innerHeight - h - 8)
  return { left: x + 'px', top: Math.max(8, y) + 'px' }
})
</script>

<style scoped>
.uprof-backdrop { position: fixed; inset: 0; z-index: 3000; }
.uprof-pop {
  position: fixed; width: 260px; box-sizing: border-box;
  background: var(--clr-surface); border: 1px solid var(--clr-border-light);
  border-radius: 14px; box-shadow: var(--sh-lg); padding: 14px 16px;
}
.uprof { display: flex; gap: 12px; align-items: center; }
.uprof-av {
  width: 42px; height: 42px; flex-shrink: 0; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-size: 15px; font-weight: 700; color: #fff; background: var(--clr-accent);
}
.uprof-body { min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.uprof-name { font-size: 15px; font-weight: 700; color: var(--clr-text); letter-spacing: -0.2px; }
.uprof-user { font-size: 12.5px; color: var(--clr-text-3); }
.uprof-email { font-size: 13px; color: var(--clr-accent); text-decoration: none; word-break: break-all; }
.uprof-email:hover { text-decoration: underline; }
.uprof-hint { font-size: 11.5px; color: var(--clr-text-3); font-style: italic; }
</style>
