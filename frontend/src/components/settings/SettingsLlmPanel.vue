<template>
  <div class="settings-panel">
    <header class="settings-panel__head">
      <h2>{{ $tt('settings.llmProvider') }}</h2>
    </header>

    <DqPrefPane class="settings-grouped-form settings-pref-pane-form film-llm-default">
      <DqPrefRow :label="$tt('settings.llmDefaultModel')">
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
          :disabled="profiles.length <= 1"
          @click="emit('remove-profile', pIdx)"
        >
          {{ $tt('settings.llmRemoveProvider') }}
        </DqButton>
      </div>

      <DqPrefPane class="settings-grouped-form settings-pref-pane-form">
        <DqPrefRow :label="$tt('settings.llmPreset')">
          <DqSelect
            size="sm"
            :model-value="profile.preset_id || matchedPresetId(profile)"
            @update:model-value="(v: unknown) => emit('apply-preset', profile, String(v ?? ''))"
          >
            <DqOption
              v-for="p in presets"
              :key="p.id"
              :value="p.id"
              :label="p.label"
            />
          </DqSelect>
        </DqPrefRow>
        <DqPrefRow :label="$tt('settings.llmProtocol')">
          <DqSelect
            v-if="protocolEditable(profile)"
            size="sm"
            :model-value="profile.protocol"
            @update:model-value="(v: unknown) => emit('set-protocol', profile, v)"
          >
            <DqOption
              v-for="opt in protocols"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </DqSelect>
          <code v-else class="film-page__path">{{ protocolLabel(profile.protocol) }}</code>
        </DqPrefRow>
        <DqPrefRow v-if="isChatProtocol(profile)" :label="$tt('settings.llmDefaultDialect')">
          <DqSelect
            size="sm"
            :model-value="profile.default_dialect || DIALECT_INFER"
            @update:model-value="(v: unknown) => onDefaultDialect(profile, v)"
          >
            <DqOption :value="DIALECT_INFER" :label="$tt('settings.llmDialectInfer')" />
            <DqOption
              v-for="d in dialectOptions"
              :key="d.id"
              :value="d.id"
              :label="d.label"
            />
          </DqSelect>
        </DqPrefRow>
        <DqPrefRow :label="$tt('settings.baseUrl')">
          <DqInput v-model="profile.base_url" size="sm" :placeholder="urlPlaceholder" />
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
            :disabled="isRefreshing(profile.id)"
            @click="emit('refresh-models', profile)"
          >
            {{ isRefreshing(profile.id) ? $tt('settings.llmRefreshingModels') : $tt('settings.llmRefreshModels') }}
          </DqButton>
        </div>

        <p v-if="refreshError(profile.id)" class="settings-panel__error">{{ refreshError(profile.id) }}</p>
        <p class="settings-panel__hint">{{ $tt('settings.llmSelectModelsHint') }}</p>

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
              <DqSelect
                v-if="isChatProtocol(profile) && isModelSelected(profile, item.id)"
                size="sm"
                class="film-llm-catalog__dialect"
                :model-value="modelDialect(profile, item.id) || DIALECT_INHERIT"
                @update:model-value="(v: unknown) => onModelDialect(profile, item.id, v)"
              >
                <DqOption :value="DIALECT_INHERIT" :label="$tt('settings.llmDialectInherit')" />
                <DqOption
                  v-for="d in dialectOptions"
                  :key="d.id"
                  :value="d.id"
                  :label="d.label"
                />
              </DqSelect>
              <DqSwitch
                size="sm"
                :model-value="isModelSelected(profile, item.id)"
                @update:model-value="(v: boolean) => onCatalogToggle(profile, item, Boolean(v))"
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
import { computed } from 'vue';
import { $tt } from '@/utils/i18n';
import type { FilmLlmPreset, FilmLlmProfile } from '@/utils/api';

export type RemoteLlmModel = { id: string; label: string };

/** DqSelect rejects empty-string values — use sentinels. */
const DIALECT_INFER = '__infer__';
const DIALECT_INHERIT = '__inherit__';

const FALLBACK_DIALECTS: Array<{ id: string; label: string }> = [
  { id: 'openai', label: 'OpenAI (reasoning_effort)' },
  { id: 'deepseek', label: 'DeepSeek (thinking.type)' },
  { id: 'qwen', label: 'Qwen (enable_thinking)' },
  { id: 'kimi', label: 'Kimi K2.5/K2.6' },
  { id: 'kimi_code', label: 'Kimi Code K2.7' },
  { id: 'kimi_k3', label: 'Kimi K3' },
  { id: 'glm', label: 'GLM / 智谱' },
  { id: 'minimax', label: 'MiniMax' },
  { id: 'gemini', label: 'Gemini' },
  { id: 'grok', label: 'Grok' },
];

const props = defineProps<{
  profiles: FilmLlmProfile[];
  defaultModel: string;
  modelOptions: Array<{ value: string; label: string }>;
  protocols: Array<{ value: string; label: string }>;
  presets: FilmLlmPreset[];
  dialects: Array<{ id: string; label: string }>;
  urlPlaceholder: string;
  refreshState?: Record<string, { loading?: boolean; error?: string; items?: RemoteLlmModel[] }>;
}>();

const dialectOptions = computed(() =>
  props.dialects?.length ? props.dialects : FALLBACK_DIALECTS,
);

const emit = defineEmits<{
  (e: 'update:defaultModel', v: string): void;
  (e: 'remove-profile', idx: number): void;
  (e: 'set-protocol', profile: FilmLlmProfile, v: unknown): void;
  (e: 'set-default-dialect', profile: FilmLlmProfile, dialect: string): void;
  (e: 'set-model-dialect', profile: FilmLlmProfile, modelId: string, dialect: string): void;
  (e: 'apply-preset', profile: FilmLlmProfile, presetId: string): void;
  (e: 'refresh-models', profile: FilmLlmProfile): void;
  (e: 'toggle-remote-model', profile: FilmLlmProfile, model: RemoteLlmModel, selected: boolean): void;
}>();

function matchedPresetId(profile: FilmLlmProfile): string {
  if (profile.preset_id) return profile.preset_id;
  const base = (profile.base_url || '').replace(/\/+$/, '');
  const hit = props.presets.find((p) => {
    if (!p.base_url) return false;
    return p.base_url.replace(/\/+$/, '') === base && p.protocol === profile.protocol;
  });
  return hit?.id || 'custom';
}

function protocolEditable(profile: FilmLlmProfile) {
  const pid = profile.preset_id || matchedPresetId(profile);
  return pid === 'custom' || pid === '';
}

function protocolLabel(protocol: string) {
  const hit = props.protocols.find((p) => p.value === protocol);
  return hit?.label || protocol;
}

function isChatProtocol(profile: FilmLlmProfile) {
  const p = (profile.protocol || '').replace(/-/g, '_');
  return p === 'openai' || p === 'openai_compatible' || p === '';
}

function isRefreshing(profileId: string) {
  return Boolean(props.refreshState?.[profileId]?.loading);
}

function refreshError(profileId: string) {
  const raw = String(props.refreshState?.[profileId]?.error || '').trim();
  if (!raw) return '';
  if (/401|incorrect.?api.?key|invalid.?api.?key/i.test(raw)) {
    return $tt('settings.llmRefreshAuthFailed');
  }
  return raw;
}

function modelChoices(profile: FilmLlmProfile): RemoteLlmModel[] {
  const byId = new Map<string, RemoteLlmModel>();
  const add = (id: string, label?: string) => {
    const key = id.trim();
    if (!key) return;
    if (!byId.has(key)) {
      byId.set(key, { id: key, label: (label || key).trim() || key });
    }
  };
  const preset = props.presets.find((p) => p.id === (profile.preset_id || matchedPresetId(profile)));
  for (const m of preset?.models || []) add(m.id, m.label);
  for (const m of props.refreshState?.[profile.id]?.items || []) add(m.id, m.label);
  for (const m of profile.models || []) add(m.id, m.label);
  return Array.from(byId.values()).sort((a, b) =>
    a.label.localeCompare(b.label, undefined, { sensitivity: 'base' }),
  );
}

function isModelSelected(profile: FilmLlmProfile, modelId: string) {
  const id = modelId.trim();
  return profile.models.some((m) => m.id.trim() === id && m.enabled !== false);
}

function modelDialect(profile: FilmLlmProfile, modelId: string) {
  const m = profile.models.find((x) => x.id.trim() === modelId.trim());
  return m?.reasoning_dialect || '';
}

function onCatalogToggle(profile: FilmLlmProfile, item: RemoteLlmModel, selected: boolean) {
  emit('toggle-remote-model', profile, item, selected);
}

function onDefaultDialect(profile: FilmLlmProfile, v: unknown) {
  const raw = String(v ?? '');
  emit('set-default-dialect', profile, raw === DIALECT_INFER ? '' : raw);
}

function onModelDialect(profile: FilmLlmProfile, modelId: string, v: unknown) {
  const raw = String(v ?? '');
  emit('set-model-dialect', profile, modelId, raw === DIALECT_INHERIT ? '' : raw);
}
</script>

<style scoped>
.film-llm-catalog {
  display: flex;
  flex-direction: column;
  margin-top: 2px;
  padding: 0;
  border: none;
  border-radius: 0;
  background: transparent;
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

.film-llm-catalog__dialect {
  min-width: 140px;
}
</style>
