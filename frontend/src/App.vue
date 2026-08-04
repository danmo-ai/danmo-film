<template>
  <div class="main-container dq-app-shell dq-app-shell--sidebar">
    <aside class="app-sidebar dq-app-sidebar dq-glass--sidebar">
      <nav class="dq-top-nav-menu film-nav">
        <button
          type="button"
          class="film-nav__brand"
          @click="router.push({ name: 'long_video_create' })"
        >
          Danmo Film
        </button>
        <button
          type="button"
          :class="['film-nav__item', { active: activePage === 'long_video_create' }]"
          @click="router.push({ name: 'long_video_create' })"
        >
          {{ $tt('nav.long_video_create') }}
        </button>
        <button
          type="button"
          :class="['film-nav__item', { active: activePage === 'settings' }]"
          @click="router.push({ name: 'settings' })"
        >
          {{ $tt('nav.settings') }}
        </button>
      </nav>
    </aside>
    <main class="app-main dq-app-main">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';

const route = useRoute();
const router = useRouter();
const activePage = ref(String(route.name || 'long_video_create'));
watch(
  () => route.name,
  (name) => {
    activePage.value = String(name || 'long_video_create');
  },
);
</script>

<style scoped>
.film-nav {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px 12px;
}
.film-nav__brand {
  font-family: "Iowan Old Style", "Palatino Linotype", Palatino, serif;
  font-size: 1.25rem;
  font-weight: 600;
  text-align: left;
  background: transparent;
  border: 0;
  color: var(--dq-text, #e8e8e8);
  cursor: pointer;
  margin-bottom: 12px;
}
.film-nav__item {
  text-align: left;
  background: transparent;
  border: 0;
  color: var(--dq-text-muted, #aaa);
  padding: 8px 10px;
  border-radius: 6px;
  cursor: pointer;
}
.film-nav__item.active,
.film-nav__item:hover {
  color: var(--dq-text, #fff);
  background: color-mix(in srgb, var(--dq-accent, #6a8) 18%, transparent);
}
</style>
