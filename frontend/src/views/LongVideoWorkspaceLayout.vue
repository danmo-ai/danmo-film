<template>
  <div class="lv-workspace">
    <LongVideoWorkspaceNav
      ref="navRef"
      :refresh-token="navRefreshToken"
      @changed="onNavChanged"
    />
    <div class="lv-workspace__main">
      <router-view />
    </div>
  </div>
</template>

<script setup lang="ts">
import { provide, ref } from 'vue';
import LongVideoWorkspaceNav from '@/components/long-video/LongVideoWorkspaceNav.vue';
import '@/styles/long-video.css';

const navRef = ref<InstanceType<typeof LongVideoWorkspaceNav> | null>(null);
const navRefreshToken = ref(0);

function refreshNav() {
  navRefreshToken.value += 1;
  void (navRef.value?.refreshAll?.() ?? navRef.value?.refreshProjects?.());
}

provide('lvRefreshWorkspaceNav', refreshNav);

function onNavChanged() {
  /* child pages may react if needed */
}
</script>

<style scoped>
.lv-workspace {
  display: flex;
  height: 100%;
  min-height: 0;
  width: 100%;
  overflow: hidden;
}

.lv-workspace > :first-child {
  width: 260px;
  flex-shrink: 0;
  min-height: 0;
}

.lv-workspace__main {
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
</style>
