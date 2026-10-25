<template>
  <div class="main-container film-shell dq-app-shell dq-app-shell--sidebar">
    <aside class="film-sidebar app-sidebar dq-app-sidebar dq-glass--sidebar">
      <div class="dq-tauri-titlebar-inset" aria-hidden="true" />
      <FilmSideNav :active-page="activePage" @navigate="onNavigate" />
    </aside>
    <main class="film-main app-main dq-app-main">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import FilmSideNav from '@/components/shell/FilmSideNav.vue';

const route = useRoute();
const router = useRouter();

const CREATE_SECTION = new Set([
  'long_video_projects',
  'long_video_series',
  'long_video_create',
]);

function navActiveForRoute(name: unknown): string {
  const n = String(name || 'long_video_projects');
  if (CREATE_SECTION.has(n)) return 'long_video_create';
  return n;
}

const activePage = ref(navActiveForRoute(route.name));

watch(
  () => route.name,
  (name) => {
    activePage.value = navActiveForRoute(name);
  },
);

function onNavigate(name: string) {
  if (name === 'long_video_create') {
    void router.push({ name: 'long_video_projects' });
    return;
  }
  void router.push({ name });
}
</script>

<style scoped>
.main-container {
  height: 100%;
}
</style>
