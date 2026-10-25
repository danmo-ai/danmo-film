<template>
  <div
    class="lv-step-save"
    :class="{ 'lv-step-save--flush': flush }"
    role="group"
    :aria-label="saveLabel"
  >
    <span class="lv-step-save__hint">{{ hint }}</span>
    <div class="lv-step-save__actions">
      <span class="lv-step-save__status" :data-status="status">{{ statusLabel }}</span>
      <DqButton
        :type="nextLabel ? 'default' : 'primary'"
        size="sm"
        :loading="loading"
        @click="$emit('save')"
      >
        {{ saveLabel }}
      </DqButton>
      <DqButton
        v-if="nextLabel"
        type="primary"
        size="sm"
        :loading="nextLoading"
        :disabled="nextDisabled"
        @click="$emit('next')"
      >
        {{ nextLabel }}
      </DqButton>
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    saveLabel: string;
    hint?: string;
    status: string;
    statusLabel: string;
    loading?: boolean;
    /** Forward / primary step action shown after Save (e.g. next tab or merge). */
    nextLabel?: string;
    nextDisabled?: boolean;
    nextLoading?: boolean;
    /** When true, sit under content instead of pinning to the pane bottom. */
    flush?: boolean;
  }>(),
  { flush: false, nextDisabled: false, nextLoading: false },
);

defineEmits<{
  (e: 'save'): void;
  (e: 'next'): void;
}>();
</script>

<style scoped>
.lv-step-save {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-shrink: 0;
  margin-top: auto;
  padding: 12px 0;
  border-top: 0.5px solid var(--dq-border-subtle);
  box-sizing: border-box;
}

.lv-step-save--flush {
  margin-top: 12px;
}

.lv-step-save__hint {
  font-size: 12px;
  color: var(--dq-label-tertiary);
  line-height: 1.4;
  min-width: 0;
}

.lv-step-save__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.lv-step-save__status {
  font-size: 11px;
  color: var(--dq-label-tertiary);
  max-width: 7em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.lv-step-save__status[data-status='error'] {
  color: var(--dq-danger);
}

.lv-step-save__status[data-status='saved'] {
  color: var(--dq-label-secondary);
}
</style>
