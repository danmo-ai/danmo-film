<template>
  <div class="lv-ws-home">
    <h1>{{ $tt('video.longVideoLibraryTitle') }}</h1>
    <p>{{ $tt('video.longVideoWorkspaceHomeHint') }}</p>
  </div>
</template>

<script setup lang="ts">
import { $tt } from '@/utils/i18n';
</script>

<style scoped>
.lv-ws-home {
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: flex-start;
  padding: 40px 48px;
  box-sizing: border-box;
  color: var(--dq-label-secondary);
}

.lv-ws-home h1 {
  margin: 0 0 10px;
  font-size: 22px;
  font-weight: 650;
  color: var(--dq-label-primary);
}

.lv-ws-home p {
  margin: 0;
  max-width: 420px;
  font-size: 14px;
  line-height: 1.55;
}
</style>
