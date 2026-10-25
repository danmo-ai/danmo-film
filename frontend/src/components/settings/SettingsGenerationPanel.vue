<template>
  <div class="settings-panel">
    <header class="settings-panel__head">
      <h2>{{ title }}</h2>
    </header>

    <DqPrefPane class="settings-grouped-form settings-pref-pane-form film-llm-default">
      <DqPrefRow :label="defaultModelLabel">
        <DqSelect
          size="sm"
          :model-value="defaultModel"
          @update:model-value="(v: unknown) => emit('update:defaultModel', String(v ?? ''))"
        >
          <DqOption
            v-for="opt in modelOptions"
            :key="opt.value"
            :value="opt.value"
            :label="opt.label"
          />
        </DqSelect>
      </DqPrefRow>
    </DqPrefPane>

    <p v-if="!profiles.length" class="settings-panel__hint">
      {{ emptyHint }}
    </p>

    <div
      v-for="(profile, pIdx) in profiles"
      :key="profile.id"
      class="film-llm-card"
    >
      <div class="film-llm-card__head">
        <DqInput
          :model-value="profile.label"
          size="sm"
          class="film-llm-card__title"
          :placeholder="$tt('settings.llmProviderNamePh')"
          @update:model-value="(v: string) => (profile.label = v)"
        />
        <div class="film-llm-card__toggle">
          <span>{{ $tt('settings.llmEnabled') }}</span>
          <DqSwitch v-model="profile.enabled" size="sm" />
        </div>
        <DqButton
          size="sm"
          type="danger"
          @click="emit('remove-profile', pIdx)"
        >
          {{ $tt('settings.llmRemoveProvider') }}
        </DqButton>
      </div>

      <DqPrefPane class="settings-grouped-form settings-pref-pane-form">
        <DqPrefRow :label="$tt('settings.genPreset')">
          <code class="film-page__path">{{ vendorLabel(profile.vendor) }}</code>
        </DqPrefRow>
        <DqPrefRow :label="$tt('settings.baseUrl')">
          <DqInput
            v-model="profile.base_url"
            size="sm"
            :placeholder="urlPlaceholder(profile.vendor)"
          />
        </DqPrefRow>
        <DqPrefRow :label="$tt('settings.apiKey')">
          <DqInput
            v-model="profile.api_key"
            size="sm"
            type="password"
            :placeholder="$tt('settings.apiKeyPh')"
          />
        </DqPrefRow>
      </DqPrefPane>

      <div class="film-llm-models">
        <div class="film-llm-models__head">
          <span>{{ $tt('settings.llmModels') }}</span>
          <DqButton
            size="sm"
            type="text"
            :disabled="isRefreshing(profile.id) || saving"
            @click="emit('refresh-models', profile)"
          >
            {{ isRefreshing(profile.id) ? $tt('settings.llmRefreshingModels') : $tt('settings.llmRefreshModels') }}
          </DqButton>
        </div>

        <p v-if="refreshError(profile.id)" class="settings-panel__error">{{ refreshError(profile.id) }}</p>
        <p class="settings-panel__hint">{{ $tt('settings.genSelectModelsHint') }}</p>

        <div v-if="modelChoices(profile).length" class="film-llm-catalog">
          <div
            v-for="item in modelChoices(profile)"
            :key="item.id"
            class="film-llm-catalog__row"
          >
            <div class="film-llm-catalog__meta">
              <span class="film-llm-catalog__label">{{ item.label || item.id }}</span>
              <code class="film-llm-catalog__id">{{ item.id }}</code>
            </div>
            <div class="film-llm-catalog__controls">
              <DqSwitch
                size="sm"
                :model-value="isModelSelected(profile, item.id)"
                @update:model-value="(v: boolean) => emit('toggle-model', profile, item, Boolean(v))"
              />
            </div>
          </div>
        </div>
        <p v-else class="settings-panel__hint">{{ $tt('settings.llmModelsEmpty') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { $tt } from '@/utils/i18n';
import type { FilmGenProfile } from '@/utils/api';

type GenModelRow = { id: string; label?: string; enabled?: boolean };

const props = defineProps<{
  title: string;
  defaultModelLabel: string;
  emptyHint: string;
  profiles: FilmGenProfile[];
  presets: Array<{ value: string; label: string }>;
  defaultModel: string;
  modelOptions: Array<{ value: string; label: string }>;
  saving: boolean;
  refreshState: Record<string, { loading?: boolean; error?: string; items?: GenModelRow[] }>;
}>();

const emit = defineEmits<{
  (e: 'update:defaultModel', v: string): void;
  (e: 'refresh-models', profile: FilmGenProfile): void;
  (e: 'remove-profile', idx: number): void;
  (e: 'toggle-model', profile: FilmGenProfile, model: GenModelRow, on: boolean): void;
}>();

const VENDOR_URL: Record<string, string> = {
  'danmo-make': 'http://127.0.0.1:7800',
  volcengine: 'https://ark.cn-beijing.volces.com',
  dashscope: 'https://dashscope.aliyuncs.com',
  minimax: 'https://api.minimax.chat',
  openai: 'https://api.openai.com/v1',
};

function vendorLabel(vendor: string) {
  return props.presets.find((p) => p.value === vendor)?.label || vendor;
}

function urlPlaceholder(vendor: string) {
  return VENDOR_URL[vendor] || VENDOR_URL.volcengine;
}

function isRefreshing(id: string) {
  return Boolean(props.refreshState[id]?.loading);
}

function refreshError(id: string) {
  return props.refreshState[id]?.error || '';
}

function modelChoices(profile: FilmGenProfile): GenModelRow[] {
  const byId = new Map<string, GenModelRow>();
  const add = (id: string, label?: string) => {
    const key = id.trim();
    if (!key || byId.has(key)) return;
    byId.set(key, { id: key, label: (label || key).trim() || key });
  };
  for (const m of props.refreshState[profile.id]?.items || []) add(m.id, m.label);
  for (const m of profile.models || []) {
    if (m.enabled) add(m.id, m.label);
  }
  return [...byId.values()];
}

function isModelSelected(profile: FilmGenProfile, id: string) {
  return (profile.models || []).some((m) => m.id === id && m.enabled !== false);
}
</script>

<style scoped>
.film-llm-catalog {
  display: flex;
  flex-direction: column;
  margin-top: 2px;
  max-height: 280px;
  overflow-y: auto;
}

.film-llm-catalog__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 7px 0;
  font-size: 13px;
  color: var(--dq-label-primary);
}

.film-llm-catalog__row + .film-llm-catalog__row {
  border-top: 0.5px solid color-mix(in srgb, var(--dq-border-subtle) 50%, transparent);
}

.film-llm-catalog__meta {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.film-llm-catalog__label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.film-llm-catalog__id {
  font-size: 11px;
  color: var(--dq-label-tertiary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.film-llm-catalog__controls {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
</style>
