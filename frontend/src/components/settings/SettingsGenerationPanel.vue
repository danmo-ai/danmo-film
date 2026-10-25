<template>
  <div class="settings-panel">
    <header class="settings-panel__head">
      <h2>{{ $tt('settings.genProvider') }}</h2>
      <p>{{ $tt('settings.genProviderDesc') }}</p>
    </header>

    <DqPrefPane class="settings-grouped-form settings-pref-pane-form">
      <DqPrefRow :label="$tt('settings.activeProvider')">
        <DqSelect
          size="sm"
          :model-value="generation.provider"
          @update:model-value="(v: unknown) => emit('set-provider', v)"
        >
          <DqOption
            v-for="opt in genProviders"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </DqSelect>
      </DqPrefRow>

      <template v-if="generation.provider === 'volcengine' || generation.provider === 'ark'">
        <DqPrefRow :label="$tt('settings.baseUrl')">
          <DqInput v-model="volcengine.base_url" size="sm" placeholder="https://ark.cn-beijing.volces.com" />
        </DqPrefRow>
        <DqPrefRow :label="$tt('settings.apiKey')">
          <DqInput v-model="volcengine.api_key" size="sm" type="password" :placeholder="$tt('settings.apiKeyPh')" />
        </DqPrefRow>
        <DqPrefRow :label="$tt('settings.cloudModel')">
          <DqInput v-model="volcengine.model" size="sm" placeholder="doubao-seedance-1-0-pro-250528" />
        </DqPrefRow>
      </template>

      <template v-else-if="generation.provider === 'dashscope'">
        <DqPrefRow :label="$tt('settings.baseUrl')">
          <DqInput v-model="dashscope.base_url" size="sm" placeholder="https://dashscope.aliyuncs.com" />
        </DqPrefRow>
        <DqPrefRow :label="$tt('settings.apiKey')">
          <DqInput v-model="dashscope.api_key" size="sm" type="password" :placeholder="$tt('settings.apiKeyPh')" />
        </DqPrefRow>
        <DqPrefRow :label="$tt('settings.cloudModel')">
          <DqInput v-model="dashscope.model" size="sm" placeholder="wan2.2-t2v-plus" />
        </DqPrefRow>
      </template>

      <template v-else-if="generation.provider === 'minimax'">
        <DqPrefRow :label="$tt('settings.baseUrl')">
          <DqInput v-model="minimax.base_url" size="sm" placeholder="https://api.minimax.chat" />
        </DqPrefRow>
        <DqPrefRow :label="$tt('settings.apiKey')">
          <DqInput v-model="minimax.api_key" size="sm" type="password" :placeholder="$tt('settings.apiKeyPh')" />
        </DqPrefRow>
        <DqPrefRow :label="$tt('settings.cloudModel')">
          <DqInput v-model="minimax.model" size="sm" placeholder="MiniMax-Hailuo-02" />
        </DqPrefRow>
      </template>
    </DqPrefPane>

    <div class="settings-panel__actions">
      <DqButton type="primary" size="sm" :disabled="saving" @click="emit('save')">
        {{ $tt('settings.saveProviders') }}
      </DqButton>
      <DqButton size="sm" :disabled="checking" @click="emit('test')">
        {{ $tt('settings.testProviders') }}
      </DqButton>
    </div>
    <p v-if="saveMsg" class="settings-panel__ok">{{ saveMsg }}</p>
    <p v-if="saveError" class="settings-panel__error">{{ saveError }}</p>

    <div v-if="providerStatus" class="film-page__status-list">
      <div class="film-page__status-item">
        <span class="film-page__status-name">{{ $tt('settings.statusLlm') }}</span>
        <span class="film-page__status-id">{{ providerStatus.llm?.provider_id || '—' }}</span>
        <span
          class="film-page__status-badge"
          :class="providerStatus.llm?.available ? 'is-ok' : 'is-bad'"
        >
          {{ providerStatus.llm?.available ? $tt('settings.statusAvailable') : $tt('settings.statusUnavailable') }}
        </span>
      </div>
      <div class="film-page__status-item">
        <span class="film-page__status-name">{{ $tt('settings.statusGeneration') }}</span>
        <span class="film-page__status-id">{{ providerStatus.generation?.provider_id || '—' }}</span>
        <span
          class="film-page__status-badge"
          :class="genHealthOk ? 'is-ok' : 'is-bad'"
        >
          {{ genHealthOk ? $tt('settings.statusOk') : $tt('settings.statusFail') }}
        </span>
      </div>
      <p v-if="genHealthError" class="film-page__status-detail">{{ genHealthError }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { $tt } from '@/utils/i18n';
import type { FilmProviderBlock } from '@/utils/api';

const props = defineProps<{
  generation: FilmProviderBlock;
  volcengine: FilmProviderBlock;
  dashscope: FilmProviderBlock;
  minimax: FilmProviderBlock;
  genProviders: Array<{ value: string; label: string }>;
  saving: boolean;
  checking: boolean;
  saveMsg: string;
  saveError: string;
  providerStatus: {
    llm?: { provider_id?: string; available?: boolean };
    generation?: { provider_id?: string; health?: { ok?: boolean; error?: string } };
  } | null;
}>();

const emit = defineEmits<{
  (e: 'set-provider', v: unknown): void;
  (e: 'save'): void;
  (e: 'test'): void;
}>();

const genHealthOk = computed(() => Boolean(props.providerStatus?.generation?.health?.ok));
const genHealthError = computed(() => {
  if (genHealthOk.value) return '';
  return String(props.providerStatus?.generation?.health?.error || '').trim();
});
</script>
