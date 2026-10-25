<template>
  <div class="lv-editor-tabs-shell" :class="{ 'lv-editor-tabs-shell--rail': variant === 'rail' }">
    <div
      class="lv-editor-tabs"
      :class="[`lv-editor-tabs--${variant}`]"
      role="tablist"
      :aria-label="$tt('video.longVideoEditorTabsAria')"
      @keydown="onKeydown"
    >
      <template v-for="(tab, index) in tabs" :key="tab.id">
        <span v-if="index > 0 && variant !== 'rail'" class="lv-editor-tabs__connector" aria-hidden="true" />

        <button
          type="button"
          role="tab"
          class="lv-editor-tabs__tab"
          :class="{
            'lv-editor-tabs__tab--active': modelValue === tab.id,
            'lv-editor-tabs__tab--disabled': tab.disabled,
            'lv-editor-tabs__tab--done': tab.done && modelValue !== tab.id,
          }"
          :aria-selected="modelValue === tab.id"
          :aria-disabled="tab.disabled || undefined"
          :disabled="tab.disabled"
          :title="tab.disabled ? tab.disabledReason || undefined : tab.desc"
          @click="onTabClick(tab)"
        >
          <span
            class="lv-editor-tabs__icon"
            :class="[
              `lv-editor-tabs__icon--${tab.id}`,
              { 'lv-editor-tabs__icon--done': tab.done },
              { 'lv-editor-tabs__icon--active': modelValue === tab.id },
            ]"
            aria-hidden="true"
          >
            <span v-if="tab.done" class="lv-editor-tabs__icon-mark">✓</span>
            <span v-else class="lv-editor-tabs__icon-mark">{{ index + 1 }}</span>
          </span>
          <span class="lv-editor-tabs__text">
            <span class="lv-editor-tabs__label">{{ tab.label }}</span>
            <span v-if="variant === 'cards'" class="lv-editor-tabs__desc">{{ tab.desc }}</span>
          </span>
          <span v-if="tab.badge != null" class="lv-editor-tabs__badge">{{ tab.badge }}</span>
        </button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import type { LongVideoEditorTab } from '@/types';

const props = withDefaults(
  defineProps<{
    modelValue: LongVideoEditorTab;
    variant?: 'cards' | 'rail';
    castCount?: number;
    sceneEntityCount?: number;
    shotCount?: number;
    settingsDone?: boolean;
    scriptDone?: boolean;
    castDone?: boolean;
    scenesDone?: boolean;
    storyboardDone?: boolean;
    deliveryDone?: boolean;
    storyboardDisabled?: boolean;
    storyboardDisabledReason?: string;
  }>(),
  { variant: 'cards' },
);

const emit = defineEmits<{
  (e: 'update:modelValue', value: LongVideoEditorTab): void;
  (e: 'storyboard-blocked'): void;
}>();

const { t: $tt } = useI18n();

const tabs = computed(() => [
  {
    id: 'settings' as const,
    label: $tt('video.longVideoEditorTabSettings'),
    desc: $tt('video.longVideoEditorTabSettingsDesc'),
    done: Boolean(props.settingsDone),
    badge: null as number | null,
    disabled: false,
    disabledReason: '',
  },
  {
    id: 'script' as const,
    label: $tt('video.longVideoEditorTabScript'),
    desc: $tt('video.longVideoEditorTabScriptDesc'),
    done: props.scriptDone,
    badge: null as number | null,
    disabled: false,
    disabledReason: '',
  },
  {
    id: 'cast' as const,
    label: $tt('video.longVideoEditorTabCast'),
    desc: $tt('video.longVideoEditorTabCastDesc'),
    done: props.castDone,
    badge: props.castCount || null,
    disabled: false,
    disabledReason: '',
  },
  {
    id: 'scenes' as const,
    label: $tt('video.longVideoEditorTabScenes'),
    desc: $tt('video.longVideoEditorTabScenesDesc'),
    done: props.scenesDone,
    badge: props.sceneEntityCount || null,
    disabled: false,
    disabledReason: '',
  },
  {
    id: 'storyboard' as const,
    label: $tt('video.longVideoEditorTabStoryboard'),
    desc: $tt('video.longVideoEditorTabStoryboardDesc'),
    done: props.storyboardDone,
    badge: props.shotCount || null,
    disabled: Boolean(props.storyboardDisabled),
    disabledReason: props.storyboardDisabledReason?.trim() || $tt('video.longVideoStoryboardTabBlocked'),
  },
  {
    id: 'delivery' as const,
    label: $tt('video.longVideoEditorTabDelivery'),
    desc: $tt('video.longVideoEditorTabDeliveryDesc'),
    done: props.deliveryDone,
    badge: null as number | null,
    disabled: false,
    disabledReason: '',
  },
]);

function onTabClick(tab: (typeof tabs.value)[number]) {
  if (tab.disabled) {
    if (tab.id === 'storyboard') emit('storyboard-blocked');
    return;
  }
  emit('update:modelValue', tab.id);
}

function onKeydown(e: KeyboardEvent) {
  if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight' && e.key !== 'Home' && e.key !== 'End') {
    return;
  }
  const list = tabs.value;
  const idx = list.findIndex((t) => t.id === props.modelValue);
  if (idx < 0) return;

  e.preventDefault();
  const trySelect = (i: number) => {
    const tab = list[i];
    if (!tab || tab.disabled) return false;
    emit('update:modelValue', tab.id);
    return true;
  };

  if (e.key === 'Home') {
    for (let i = 0; i < list.length; i += 1) {
      if (trySelect(i)) return;
    }
    return;
  }
  if (e.key === 'End') {
    for (let i = list.length - 1; i >= 0; i -= 1) {
      if (trySelect(i)) return;
    }
    return;
  }

  const dir = e.key === 'ArrowRight' ? 1 : -1;
  for (let i = idx + dir; i >= 0 && i < list.length; i += dir) {
    if (trySelect(i)) return;
  }
}
</script>

<style scoped>
.lv-editor-tabs__tab--disabled {
  opacity: 0.42;
  cursor: not-allowed;
}

.lv-editor-tabs__tab:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--dq-accent) 55%, transparent);
  outline-offset: 2px;
}
</style>
