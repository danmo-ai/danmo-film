<template>
  <div class="settings-page settings-page--rail">
    <aside class="settings-sidebar" :aria-label="$tt('settings.title')">
      <nav class="settings-nav" role="navigation" :aria-label="$tt('settings.title')">
        <button
          v-for="item in navItems"
          :key="item.id"
          type="button"
          class="settings-nav__item"
          :class="{ 'settings-nav__item--active': activeSection === item.id }"
          @click="activeSection = item.id"
        >
          <span class="settings-nav__icon" aria-hidden="true">
            <DqIcon :size="14"><component :is="item.icon" /></DqIcon>
          </span>
          <span class="settings-nav__label">{{ $tt(item.labelKey) }}</span>
        </button>
      </nav>
    </aside>

    <div class="settings-content">
      <div class="settings-content__body">
        <p v-if="loadError" class="settings-panel__error">{{ loadError }}</p>

        <SettingsGeneralPanel
          v-if="activeSection === 'general'"
          :path-rows="pathRows"
        />

        <SettingsLlmPanel
          v-else-if="activeSection === 'llm'"
          :profiles="llmProfiles"
          :default-model="llmDefaultModel"
          :model-options="enabledModelOptions"
          :protocols="llmProtocols"
          :presets="llmPresets"
          :dialects="llmDialects"
          :url-placeholder="llmUrlPlaceholder"
          :refresh-state="llmRefreshState"
          @update:default-model="setDefaultModel"
          @remove-profile="removeProfile"
          @set-protocol="setProfileProtocol"
          @set-default-dialect="setDefaultDialect"
          @set-model-dialect="setModelDialect"
          @apply-preset="applyPresetToProfile"
          @refresh-models="refreshProviderModels"
          @toggle-remote-model="toggleRemoteModel"
        />

        <SettingsGenerationPanel
          v-else-if="activeSection === 'generation'"
          :title="$tt('settings.genProvider')"
          :default-model-label="$tt('settings.genDefaultModel')"
          :empty-hint="$tt('settings.genProvidersEmpty')"
          :profiles="genProfiles"
          :presets="genPresets"
          :default-model="genDefaultModel"
          :model-options="enabledGenModelRefs()"
          :saving="saving"
          :refresh-state="genRefreshState"
          @update:default-model="setGenDefaultModel"
          @refresh-models="refreshGenModels"
          @remove-profile="removeGenProfile"
          @toggle-model="toggleGenModel"
        />

        <SettingsGenerationPanel
          v-else
          :title="$tt('settings.imgProvider')"
          :default-model-label="$tt('settings.imgDefaultModel')"
          :empty-hint="$tt('settings.imgProvidersEmpty')"
          :profiles="imgProfiles"
          :presets="imgPresets"
          :default-model="imgDefaultModel"
          :model-options="enabledImgModelRefs()"
          :saving="saving"
          :refresh-state="imgRefreshState"
          @update:default-model="setImgDefaultModel"
          @refresh-models="refreshImgModels"
          @remove-profile="removeImgProfile"
          @toggle-model="toggleImgModel"
        />
      </div>

      <LongVideoStepSaveBar
        v-if="activeSection === 'llm' || activeSection === 'generation' || activeSection === 'image'"
        :save-label="$tt('settings.saveProviders')"
        :hint="
          activeSection === 'llm'
            ? $tt('settings.saveLlmProvidersHint')
            : activeSection === 'image'
              ? $tt('settings.saveImgProvidersHint')
              : $tt('settings.saveGenProvidersHint')
        "
        :status="providerSaveStatus"
        :status-label="providerSaveStatusLabel"
        :loading="saving"
        @save="save"
      >
        <template v-if="activeSection === 'llm'" #actions-before>
          <DqButton size="sm" @click="openAddLlmProvider">
            {{ $tt('settings.llmAddProvider') }}
          </DqButton>
        </template>
        <template v-else-if="activeSection === 'generation'" #actions-before>
          <DqButton size="sm" @click="openAddGenProvider">
            {{ $tt('settings.llmAddProvider') }}
          </DqButton>
        </template>
        <template v-else-if="activeSection === 'image'" #actions-before>
          <DqButton size="sm" @click="openAddImgProvider">
            {{ $tt('settings.llmAddProvider') }}
          </DqButton>
        </template>
      </LongVideoStepSaveBar>
    </div>

    <DqDrawer
      v-model:open="llmAddDrawerOpen"
      :title="$tt('settings.llmAddProvider')"
      direction="rtl"
      size="400px"
      class="settings-provider-drawer"
    >
      <div class="settings-provider-drawer__body">
        <p class="settings-provider-drawer__hint">{{ $tt('settings.llmAddProviderHint') }}</p>
        <label class="settings-provider-drawer__field">
          <span class="settings-provider-drawer__label">{{ $tt('settings.llmPreset') }}</span>
          <DqSelect
            size="sm"
            class="settings-provider-drawer__select"
            :model-value="llmAddPresetId"
            @update:model-value="(v: unknown) => (llmAddPresetId = String(v ?? 'openai-chat'))"
          >
            <DqOption
              v-for="p in llmPresets"
              :key="p.id"
              :value="p.id"
              :label="p.label"
            />
          </DqSelect>
        </label>
        <p v-if="llmAddPresetMeta" class="settings-provider-drawer__meta">
          {{ llmAddPresetMeta }}
        </p>
        <div class="settings-provider-drawer__actions">
          <DqButton size="sm" @click="llmAddDrawerOpen = false">
            {{ $tt('common.cancel') }}
          </DqButton>
          <DqButton type="primary" size="sm" @click="confirmAddLlmProvider">
            {{ $tt('settings.llmAddProviderConfirm') }}
          </DqButton>
        </div>
      </div>
    </DqDrawer>

    <DqDrawer
      v-model:open="genAddDrawerOpen"
      :title="$tt('settings.llmAddProvider')"
      direction="rtl"
      size="400px"
      class="settings-provider-drawer"
    >
      <div class="settings-provider-drawer__body">
        <p class="settings-provider-drawer__hint">{{ $tt('settings.genAddProviderHint') }}</p>
        <label class="settings-provider-drawer__field">
          <span class="settings-provider-drawer__label">{{ $tt('settings.genPreset') }}</span>
          <DqSelect
            size="sm"
            class="settings-provider-drawer__select"
            :model-value="genAddPresetId"
            @update:model-value="(v: unknown) => (genAddPresetId = String(v ?? 'volcengine'))"
          >
            <DqOption
              v-for="opt in genPresets"
              :key="opt.value"
              :value="opt.value"
              :label="opt.label"
            />
          </DqSelect>
        </label>
        <p v-if="genAddPresetMeta" class="settings-provider-drawer__meta">
          {{ genAddPresetMeta }}
        </p>
        <div class="settings-provider-drawer__actions">
          <DqButton size="sm" @click="genAddDrawerOpen = false">
            {{ $tt('common.cancel') }}
          </DqButton>
          <DqButton type="primary" size="sm" @click="confirmAddGenProvider">
            {{ $tt('settings.llmAddProviderConfirm') }}
          </DqButton>
        </div>
      </div>
    </DqDrawer>

    <DqDrawer
      v-model:open="imgAddDrawerOpen"
      :title="$tt('settings.llmAddProvider')"
      direction="rtl"
      size="400px"
      class="settings-provider-drawer"
    >
      <div class="settings-provider-drawer__body">
        <p class="settings-provider-drawer__hint">{{ $tt('settings.imgAddProviderHint') }}</p>
        <label class="settings-provider-drawer__field">
          <span class="settings-provider-drawer__label">{{ $tt('settings.genPreset') }}</span>
          <DqSelect
            size="sm"
            class="settings-provider-drawer__select"
            :model-value="imgAddPresetId"
            @update:model-value="(v: unknown) => (imgAddPresetId = String(v ?? 'danmo-make'))"
          >
            <DqOption
              v-for="opt in imgPresets"
              :key="opt.value"
              :value="opt.value"
              :label="opt.label"
            />
          </DqSelect>
        </label>
        <p v-if="imgAddPresetMeta" class="settings-provider-drawer__meta">
          {{ imgAddPresetMeta }}
        </p>
        <div class="settings-provider-drawer__actions">
          <DqButton size="sm" @click="imgAddDrawerOpen = false">
            {{ $tt('common.cancel') }}
          </DqButton>
          <DqButton type="primary" size="sm" @click="confirmAddImgProvider">
            {{ $tt('settings.llmAddProviderConfirm') }}
          </DqButton>
        </div>
      </div>
    </DqDrawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import {
  MagicStick,
  Monitor,
  VideoCamera,
} from '@danqing/dq-shell';
import { toast } from '@/utils/feedback';
import { $tt } from '@/utils/i18n';
import {
  api,
  type FilmGenProfile,
  type FilmLlmPreset,
  type FilmLlmProfile,
  type FilmProviderBlock,
  type FilmSettingsPayload,
} from '@/utils/api';
import { DQ_STORAGE, getItem, setItem } from '@/utils/storage';
import SettingsGeneralPanel from '@/components/settings/SettingsGeneralPanel.vue';
import SettingsLlmPanel from '@/components/settings/SettingsLlmPanel.vue';
import SettingsGenerationPanel from '@/components/settings/SettingsGenerationPanel.vue';
import LongVideoStepSaveBar from '@/components/long-video/LongVideoStepSaveBar.vue';

type SectionId = 'general' | 'llm' | 'generation' | 'image';

interface NavItem {
  id: SectionId;
  labelKey: string;
  icon: object;
}

const SECTION_IDS: SectionId[] = ['general', 'llm', 'generation', 'image'];

const navItems: NavItem[] = [
  { id: 'general', labelKey: 'settings.general', icon: Monitor },
  { id: 'llm', labelKey: 'settings.llmProvider', icon: MagicStick },
  { id: 'image', labelKey: 'settings.imgProvider', icon: MagicStick },
  { id: 'generation', labelKey: 'settings.genProvider', icon: VideoCamera },
];

function readStoredSection(): SectionId {
  const raw = String(getItem(DQ_STORAGE.SETTINGS_TAB) || '').trim();
  if (raw === 'workspace') return 'general';
  return (SECTION_IDS.includes(raw as SectionId) ? raw : 'general') as SectionId;
}

const activeSection = ref<SectionId>(readStoredSection());

watch(activeSection, (id) => {
  setItem(DQ_STORAGE.SETTINGS_TAB, id);
});

const llmUrlPlaceholder = 'https://api.openai.com/v1';

const llmProtocols = [
  { value: 'openai', label: 'OpenAI Chat' },
  { value: 'openai_responses', label: 'OpenAI Responses' },
  { value: 'anthropic', label: 'Anthropic Messages' },
];

const llmPresets = ref<FilmLlmPreset[]>([]);

/** Fallback if /api/settings/llm/presets omits reasoning_dialects. */
const DEFAULT_LLM_DIALECTS: Array<{ id: string; label: string }> = [
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
const llmDialects = ref<Array<{ id: string; label: string }>>([...DEFAULT_LLM_DIALECTS]);

const genPresets = computed(() => [
  { value: 'danmo-make', label: $tt('settings.genVendorMake') },
  { value: 'volcengine', label: $tt('settings.genVendorVolcengine') },
  { value: 'dashscope', label: $tt('settings.genVendorDashscope') },
  { value: 'minimax', label: $tt('settings.genVendorMinimax') },
]);

const imgPresets = computed(() => [
  { value: 'danmo-make', label: $tt('settings.genVendorMake') },
  { value: 'volcengine', label: $tt('settings.genVendorVolcengine') },
  { value: 'dashscope', label: $tt('settings.genVendorDashscope') },
  { value: 'minimax', label: $tt('settings.genVendorMinimax') },
  { value: 'openai', label: $tt('settings.imgVendorOpenAI') },
]);

const PATH_LABEL_KEYS: Record<string, string> = {
  assets_root: 'settings.pathAssetsRoot',
  film_db: 'settings.pathFilmDb',
  settings_file: 'settings.pathSettingsFile',
  user_data_dir: 'settings.pathUserDataDir',
  workspace_dir: 'settings.pathWorkspaceDir',
};

function emptyBlock(): FilmProviderBlock {
  return {
    protocol: '',
    provider: '',
    base_url: '',
    api_key: '',
    model: '',
    enabled: true,
    models: [],
  };
}

function newProfile(partial?: Partial<FilmLlmProfile>): FilmLlmProfile {
  const id = partial?.id || `llm-${Date.now().toString(36)}`;
  return {
    id,
    label: partial?.label || 'OpenAI · Chat',
    protocol: partial?.protocol || 'openai',
    base_url: partial?.base_url || llmUrlPlaceholder,
    api_key: partial?.api_key || '',
    auth_style: partial?.auth_style || 'bearer',
    enabled: partial?.enabled ?? true,
    default_dialect: partial?.default_dialect || '',
    preset_id: partial?.preset_id || '',
    models: (partial?.models || []).map((m) => ({
      id: m.id || '',
      label: m.label || m.id || '',
      enabled: m.enabled ?? true,
      reasoning_dialect: m.reasoning_dialect || '',
    })),
  };
}

function normalizeLegacyProviderId(id: string): string {
  const raw = id.trim().toLowerCase();
  if (raw === 'danmo-make' || raw === 'make' || raw === 'danqing') return 'openai';
  return id.trim() || 'openai';
}

const generation = reactive<FilmProviderBlock>(emptyBlock());
const genProfiles = ref<FilmGenProfile[]>([]);
const imgProfiles = ref<FilmGenProfile[]>([]);
const paths = ref<Record<string, string>>({});
const llmProfiles = ref<FilmLlmProfile[]>([]);
const llmDefaultModel = ref('');
/** Per-model capacity + sampling + thinking; round-tripped (not edited on provider page). */
const llmModelParams = ref<Record<string, {
  context_window?: number;
  max_output_tokens?: number;
  temperature?: number;
  top_p?: number;
  thinking_mode?: string;
  thinking_budget?: number;
}>>({});
/** Video/image capability catalogs; round-tripped like llm_model_params. */
const videoModelParams = ref<Record<string, {
  modes?: string[];
  protocol?: string;
  v2v_slot?: string;
  max_reference_images?: number;
  max_reference_videos?: number;
  max_duration_sec?: number;
}>>({});
const imageModelParams = ref<Record<string, {
  modes?: string[];
  protocol?: string;
  max_reference_images?: number;
}>>({});
const llmDefaultContextWindow = ref(256000);
const llmDefaultMaxOutput = ref(128000);
const llmDefaultTemperature = ref(0.35);
const llmDefaultTopP = ref(0);
const llmDefaultThinkingMode = ref('off');
const llmRefreshState = reactive<
  Record<string, { loading?: boolean; error?: string; items?: Array<{ id: string; label: string }> }>
>({});

const loadError = ref('');
const saveError = ref('');
const saveMsg = ref('');
const saving = ref(false);
const genRefreshState = reactive<
  Record<string, { loading?: boolean; error?: string; items?: Array<{ id: string; label: string }> }>
>({});
const imgRefreshState = reactive<
  Record<string, { loading?: boolean; error?: string; items?: Array<{ id: string; label: string }> }>
>({});
const llmAddPresetId = ref('openai-chat');
const llmAddDrawerOpen = ref(false);
const genAddPresetId = ref('volcengine');
const genAddDrawerOpen = ref(false);
const imgAddPresetId = ref('danmo-make');
const imgAddDrawerOpen = ref(false);
const genDefaultModel = ref('');
const imgDefaultModel = ref('');

const GEN_DEFAULTS: Record<string, { base_url: string }> = {
  'danmo-make': { base_url: 'http://127.0.0.1:7800' },
  volcengine: { base_url: 'https://ark.cn-beijing.volces.com' },
  dashscope: { base_url: 'https://dashscope.aliyuncs.com' },
  minimax: { base_url: 'https://api.minimax.chat' },
  openai: { base_url: 'https://api.openai.com/v1' },
};

const llmAddPresetMeta = computed(() => {
  const preset = llmPresets.value.find((p) => p.id === llmAddPresetId.value);
  if (!preset) return '';
  const parts: string[] = [];
  if (preset.protocol) parts.push(preset.protocol);
  if (preset.base_url) parts.push(preset.base_url);
  return parts.join(' · ');
});

const genAddPresetMeta = computed(() => {
  const id = normalizeGenVendor(genAddPresetId.value);
  return GEN_DEFAULTS[id]?.base_url || '';
});

const imgAddPresetMeta = computed(() => {
  const id = normalizeImgVendor(imgAddPresetId.value);
  return GEN_DEFAULTS[id]?.base_url || '';
});

const providerSaveStatus = computed(() => {
  if (saving.value) return 'saving';
  if (saveError.value) return 'error';
  if (saveMsg.value) return 'saved';
  return 'idle';
});

const providerSaveStatusLabel = computed(() => {
  if (saving.value) return $tt('video.longVideoAutoSaveSaving');
  if (saveError.value) return saveError.value;
  if (saveMsg.value) return saveMsg.value;
  return '';
});

const pathRows = computed(() =>
  Object.entries(paths.value).map(([key, value]) => ({
    key,
    value,
    label: PATH_LABEL_KEYS[key] ? $tt(PATH_LABEL_KEYS[key]) : key,
  })),
);

const enabledModelOptions = computed(() => {
  const opts: Array<{ value: string; label: string }> = [];
  for (const p of llmProfiles.value) {
    if (!p.enabled) continue;
    const pid = (p.id || '').trim();
    if (!pid) continue;
    for (const m of p.models) {
      if (!m.enabled || !m.id.trim()) continue;
      const mid = m.id.trim();
      const name = (m.label || mid).trim();
      opts.push({
        value: `${pid}/${mid}`,
        label: `${(p.label || pid).trim()} / ${name}`,
      });
    }
  }
  return opts;
});

function normalizeGenVendor(v: unknown): string {
  const raw = String(v ?? '').trim();
  const map: Record<string, string> = {
    'danmo-make': 'danmo-make',
    make: 'danmo-make',
    danqing: 'danmo-make',
    'Danmo Make': 'danmo-make',
    volcengine: 'volcengine',
    'Volcengine Ark Seedance': 'volcengine',
    火山引擎: 'volcengine',
    dashscope: 'dashscope',
    'DashScope 万相': 'dashscope',
    阿里云: 'dashscope',
    minimax: 'minimax',
    'MiniMax Hailuo': 'minimax',
    MiniMax: 'minimax',
  };
  return map[raw] || raw || 'volcengine';
}

function normalizeImgVendor(v: unknown): string {
  const raw = String(v ?? '').trim();
  if (raw === 'openai' || raw === 'OpenAI' || raw === 'dall-e' || raw === 'dalle') return 'openai';
  return normalizeGenVendor(raw) || 'danmo-make';
}

function newGenProfile(partial?: Partial<FilmGenProfile>): FilmGenProfile {
  const vendor = normalizeGenVendor(partial?.vendor || 'volcengine');
  const preset = genPresets.value.find((p) => p.value === vendor);
  const id =
    partial?.id?.trim() ||
    `${vendor}-${Date.now().toString(36)}`;
  return {
    id,
    label: (partial?.label || preset?.label || vendor).trim(),
    vendor,
    base_url: (partial?.base_url || GEN_DEFAULTS[vendor]?.base_url || '').trim(),
    api_key: partial?.api_key || '',
    enabled: partial?.enabled ?? true,
    models: (partial?.models || []).map((m) => ({
      id: String(m.id || '').trim(),
      label: String(m.label || m.id || '').trim(),
      enabled: m.enabled !== false,
    })).filter((m) => m.id),
  };
}

function enabledGenModelRefs(): Array<{ value: string; label: string }> {
  const opts: Array<{ value: string; label: string }> = [];
  for (const p of genProfiles.value) {
    if (!p.enabled) continue;
    const pid = (p.id || '').trim();
    if (!pid) continue;
    const plabel = (p.label || pid).trim();
    for (const m of p.models || []) {
      if (!m.enabled || !m.id?.trim()) continue;
      const mid = m.id.trim();
      opts.push({
        value: `${pid}/${mid}`,
        label: `${plabel} / ${(m.label || mid).trim()}`,
      });
    }
  }
  return opts;
}

function resolveGenModelRef(raw: string): string {
  const s = String(raw || '').trim();
  const opts = enabledGenModelRefs();
  if (s && opts.some((o) => o.value === s)) return s;
  return opts[0]?.value || '';
}

function syncGenDefaultModel() {
  genDefaultModel.value = resolveGenModelRef(genDefaultModel.value);
  const ref = genDefaultModel.value;
  if (ref.includes('/')) {
    generation.provider = ref.split('/')[0];
  }
}

function setGenDefaultModel(v: unknown) {
  genDefaultModel.value = String(v ?? '').trim();
  syncGenDefaultModel();
  void save({ quiet: true });
}

function openAddGenProvider() {
  if (!genPresets.value.some((p) => p.value === genAddPresetId.value)) {
    genAddPresetId.value = genPresets.value[0]?.value || 'volcengine';
  }
  genAddDrawerOpen.value = true;
}

function confirmAddGenProvider() {
  const vendor = normalizeGenVendor(genAddPresetId.value);
  if (!genPresets.value.some((o) => o.value === vendor)) return;
  const profile = newGenProfile({ vendor });
  genProfiles.value = [...genProfiles.value, profile];
  syncGenDefaultModel();
  genAddDrawerOpen.value = false;
  void save({ quiet: true });
}

function removeGenProfile(idx: number) {
  if (idx < 0 || idx >= genProfiles.value.length) return;
  genProfiles.value = genProfiles.value.filter((_, i) => i !== idx);
  syncGenDefaultModel();
  void save({ quiet: true });
}

function toggleGenModel(
  profile: FilmGenProfile,
  model: { id: string; label?: string },
  on: boolean,
) {
  if (!profile.models) profile.models = [];
  const id = model.id.trim();
  const idx = profile.models.findIndex((m) => m.id === id);
  if (on) {
    if (idx >= 0) {
      profile.models[idx].enabled = true;
      if (model.label) profile.models[idx].label = model.label;
    } else {
      profile.models.push({ id, label: model.label || id, enabled: true });
    }
  } else if (idx >= 0) {
    profile.models[idx].enabled = false;
  }
  syncGenDefaultModel();
  void save({ quiet: true });
}

function newImgProfile(partial?: Partial<FilmGenProfile>): FilmGenProfile {
  const vendor = normalizeImgVendor(partial?.vendor || 'danmo-make');
  const preset = imgPresets.value.find((p) => p.value === vendor);
  const id = partial?.id?.trim() || `${vendor}-${Date.now().toString(36)}`;
  return {
    id,
    label: (partial?.label || preset?.label || vendor).trim(),
    vendor,
    base_url: (partial?.base_url || GEN_DEFAULTS[vendor]?.base_url || '').trim(),
    api_key: partial?.api_key || '',
    enabled: partial?.enabled ?? true,
    models: (partial?.models || [])
      .map((m) => ({
        id: String(m.id || '').trim(),
        label: String(m.label || m.id || '').trim(),
        enabled: m.enabled !== false,
      }))
      .filter((m) => m.id),
  };
}

function enabledImgModelRefs(): Array<{ value: string; label: string }> {
  const opts: Array<{ value: string; label: string }> = [];
  for (const p of imgProfiles.value) {
    if (!p.enabled) continue;
    const pid = (p.id || '').trim();
    if (!pid) continue;
    const plabel = (p.label || pid).trim();
    for (const m of p.models || []) {
      if (!m.enabled || !m.id?.trim()) continue;
      const mid = m.id.trim();
      opts.push({
        value: `${pid}/${mid}`,
        label: `${plabel} / ${(m.label || mid).trim()}`,
      });
    }
  }
  return opts;
}

function resolveImgModelRef(raw: string): string {
  const s = String(raw || '').trim();
  const opts = enabledImgModelRefs();
  if (s && opts.some((o) => o.value === s)) return s;
  return opts[0]?.value || '';
}

function syncImgDefaultModel() {
  imgDefaultModel.value = resolveImgModelRef(imgDefaultModel.value);
}

function setImgDefaultModel(v: unknown) {
  imgDefaultModel.value = String(v ?? '').trim();
  syncImgDefaultModel();
  void save({ quiet: true });
}

function openAddImgProvider() {
  if (!imgPresets.value.some((p) => p.value === imgAddPresetId.value)) {
    imgAddPresetId.value = imgPresets.value[0]?.value || 'danmo-make';
  }
  imgAddDrawerOpen.value = true;
}

function confirmAddImgProvider() {
  const vendor = normalizeImgVendor(imgAddPresetId.value);
  if (!imgPresets.value.some((o) => o.value === vendor)) return;
  imgProfiles.value = [...imgProfiles.value, newImgProfile({ vendor })];
  syncImgDefaultModel();
  imgAddDrawerOpen.value = false;
  void save({ quiet: true });
}

function removeImgProfile(idx: number) {
  if (idx < 0 || idx >= imgProfiles.value.length) return;
  imgProfiles.value = imgProfiles.value.filter((_, i) => i !== idx);
  syncImgDefaultModel();
  void save({ quiet: true });
}

function toggleImgModel(
  profile: FilmGenProfile,
  model: { id: string; label?: string },
  on: boolean,
) {
  if (!profile.models) profile.models = [];
  const id = model.id.trim();
  const idx = profile.models.findIndex((m) => m.id === id);
  if (on) {
    if (idx >= 0) {
      profile.models[idx].enabled = true;
      if (model.label) profile.models[idx].label = model.label;
    } else {
      profile.models.push({ id, label: model.label || id, enabled: true });
    }
  } else if (idx >= 0) {
    profile.models[idx].enabled = false;
  }
  syncImgDefaultModel();
  void save({ quiet: true });
}

function parseImgProfilesFromSettings(data: FilmSettingsPayload): FilmGenProfile[] {
  const raw = data.providers?.image_profiles as unknown;
  const known = new Set(imgPresets.value.map((o) => o.value));
  if (!Array.isArray(raw) || !raw.length) return [];
  if (typeof raw[0] === 'object' && raw[0] !== null) {
    return (raw as FilmGenProfile[])
      .map((p) =>
        newImgProfile({
          id: p.id,
          label: p.label,
          vendor: normalizeImgVendor(p.vendor || p.id),
          base_url: p.base_url,
          api_key: p.api_key,
          enabled: p.enabled,
          models: p.models,
        }),
      )
      .filter((p) => known.has(p.vendor) && p.id);
  }
  return [];
}

function parseGenProfilesFromSettings(data: FilmSettingsPayload): FilmGenProfile[] {
  const raw = data.providers?.generation_profiles as unknown;
  const known = new Set(genPresets.value.map((o) => o.value));
  if (Array.isArray(raw) && raw.length && typeof raw[0] === 'object' && raw[0] !== null) {
    return (raw as FilmGenProfile[])
      .map((p) =>
        newGenProfile({
          id: p.id,
          label: p.label,
          vendor: normalizeGenVendor(p.vendor || p.id),
          base_url: p.base_url,
          api_key: p.api_key,
          enabled: p.enabled,
          models: p.models,
        }),
      )
      .filter((p) => known.has(p.vendor) && p.id);
  }
  // Legacy: string[] of vendor ids + fixed providers.<vendor> blocks.
  const ids: string[] = [];
  const seen = new Set<string>();
  if (Array.isArray(raw)) {
    for (const x of raw as string[]) {
      const id = normalizeGenVendor(x);
      if (!known.has(id) || seen.has(id)) continue;
      seen.add(id);
      ids.push(id);
    }
  }
  const active = normalizeGenVendor(data.providers?.generation?.provider || '');
  if (active && known.has(active) && !seen.has(active)) {
    ids.unshift(active);
    seen.add(active);
  }
  const blocks: Record<string, FilmProviderBlock | undefined> = {
    'danmo-make': data.providers?.['danmo-make'],
    volcengine: data.providers?.volcengine,
    dashscope: data.providers?.dashscope,
    minimax: data.providers?.minimax,
  };
  if (!ids.length) {
    for (const id of known) {
      const b = blocks[id];
      if (String(b?.api_key || '').trim() || (b?.models || []).some((m) => m.enabled && m.id)) {
        ids.push(id);
      }
    }
  }
  return ids.map((vendor) => {
    const b = blocks[vendor] || {};
    let base = String(b.base_url || '').trim();
    let key = String(b.api_key || '').trim();
    let models = Array.isArray(b.models) ? b.models : [];
    if (vendor === 'danmo-make' && !base) {
      base = String(data.providers?.generation?.base_url || '').trim();
      if (!key) key = String(data.providers?.generation?.api_key || '').trim();
    }
    if (!models.length && b.model) {
      models = [{ id: b.model, label: b.model, enabled: true }];
    }
    return newGenProfile({
      id: vendor,
      label: genPresets.value.find((p) => p.value === vendor)?.label || vendor,
      vendor,
      base_url: base || GEN_DEFAULTS[vendor]?.base_url || '',
      api_key: key,
      enabled: b.enabled !== false,
      models,
    });
  });
}

function setDefaultModel(v: unknown) {
  llmDefaultModel.value = String(v ?? '').trim();
  void save({ quiet: true });
}

function setProfileProtocol(profile: FilmLlmProfile, v: unknown) {
  const raw = String(v ?? '').trim();
  const map: Record<string, string> = {
    openai: 'openai',
    'OpenAI-compatible': 'openai',
    openai_responses: 'openai_responses',
    'OpenAI Responses': 'openai_responses',
    anthropic: 'anthropic',
    Anthropic: 'anthropic',
  };
  profile.protocol = map[raw] || raw || 'openai';
}

function openAddLlmProvider() {
  if (!llmPresets.value.some((p) => p.id === llmAddPresetId.value)) {
    llmAddPresetId.value = llmPresets.value[0]?.id || 'openai-chat';
  }
  llmAddDrawerOpen.value = true;
}

function confirmAddLlmProvider() {
  addProfileFromPreset(llmAddPresetId.value);
  llmAddDrawerOpen.value = false;
}

function addProfileFromPreset(presetId?: string) {
  const preset = llmPresets.value.find((p) => p.id === (presetId || 'openai-chat'))
    || llmPresets.value.find((p) => p.id === 'custom')
    || {
      id: 'openai-chat',
      label: 'OpenAI · Chat',
      protocol: 'openai',
      base_url: llmUrlPlaceholder,
      auth_style: 'bearer',
      default_dialect: 'openai',
      models: [{ id: 'gpt-4.1', label: 'GPT-4.1', enabled: true, reasoning_dialect: 'openai' }],
    };
  const suffix = Date.now().toString(36);
  const id = preset.id === 'custom' ? `custom-${suffix}` : `${preset.id}-${suffix}`;
  llmProfiles.value.push(
    newProfile({
      id,
      label: preset.label,
      protocol: preset.protocol || 'openai',
      base_url: preset.base_url || '',
      auth_style: preset.auth_style || 'bearer',
      default_dialect: preset.default_dialect || '',
      preset_id: preset.id,
      models: (preset.models || []).map((m) => mapPresetModel(m)),
    }),
  );
  void save({ quiet: true });
}

function mapPresetModel(m: {
  id: string;
  label?: string;
  enabled?: boolean;
  reasoning_dialect?: string;
}) {
  return {
    id: m.id,
    label: m.label || m.id,
    enabled: m.enabled ?? true,
    reasoning_dialect: m.reasoning_dialect || '',
  };
}

function applyPresetToProfile(profile: FilmLlmProfile, presetId: string) {
  const preset = llmPresets.value.find((p) => p.id === presetId);
  if (!preset) return;
  profile.label = preset.label;
  profile.protocol = preset.protocol || 'openai';
  profile.base_url = preset.base_url || '';
  profile.auth_style = preset.auth_style || 'bearer';
  profile.default_dialect = preset.default_dialect || '';
  profile.preset_id = preset.id;
  if (preset.models?.length) {
    profile.models = preset.models.map((m) => mapPresetModel(m));
  }
  syncDefaultModel();
  void save({ quiet: true });
}

function normalizeDialectId(raw: string): string {
  const s = String(raw || '').trim();
  if (!s || s === '__infer__' || s === '__inherit__') return '';
  const byId = llmDialects.value.find((d) => d.id === s);
  if (byId) return byId.id;
  const byLabel = llmDialects.value.find((d) => d.label === s);
  if (byLabel) return byLabel.id;
  // i18n sentinel labels
  if (s === $tt('settings.llmDialectInfer') || s === $tt('settings.llmDialectInherit')) return '';
  return s;
}

function setDefaultDialect(profile: FilmLlmProfile, dialect: string) {
  profile.default_dialect = normalizeDialectId(dialect);
  void save({ quiet: true });
}

function setModelDialect(profile: FilmLlmProfile, modelId: string, dialect: string) {
  const m = profile.models.find((x) => x.id.trim() === modelId.trim());
  if (m) m.reasoning_dialect = normalizeDialectId(dialect);
  void save({ quiet: true });
}

function removeProfile(idx: number) {
  if (llmProfiles.value.length <= 1) return;
  llmProfiles.value.splice(idx, 1);
  syncDefaultModel();
  void save({ quiet: true });
}

async function refreshProviderModels(profile: FilmLlmProfile) {
  const key = profile.id || profile.base_url || 'default';
  llmRefreshState[key] = {
    ...(llmRefreshState[key] || {}),
    loading: true,
    error: '',
  };
  try {
    const data = await api.settings.fetchLlmProviderModels({
      profile_id: profile.id,
      protocol: profile.protocol,
      base_url: profile.base_url,
      api_key: profile.api_key,
    });
    llmRefreshState[key] = {
      loading: false,
      error: '',
      items: (data.items || []).map((m) => ({
        id: String(m.id || '').trim(),
        label: String(m.label || m.id || '').trim(),
      })).filter((m) => m.id),
    };
    if (!llmRefreshState[key].items?.length) {
      llmRefreshState[key].error = $tt('settings.llmRefreshModelsEmpty');
      return;
    }
    // Credentials + endpoint validated — persist so refresh replaces explicit save.
    await save({ quiet: true });
  } catch (e: any) {
    llmRefreshState[key] = {
      loading: false,
      error: e?.response?.data?.detail || e?.message || String(e),
      items: llmRefreshState[key]?.items || [],
    };
  }
}

function toggleRemoteModel(
  profile: FilmLlmProfile,
  model: { id: string; label: string },
  selected: boolean,
) {
  const id = String(model.id || '').trim();
  if (!id) return;
  const idx = profile.models.findIndex((m) => m.id.trim() === id);
  if (selected) {
    if (idx >= 0) {
      profile.models[idx].enabled = true;
      if (!profile.models[idx].label) {
        profile.models[idx].label = model.label || id;
      }
    } else {
      profile.models.push({
        id,
        label: model.label || id,
        enabled: true,
        reasoning_dialect: '',
      });
    }
  } else if (idx >= 0) {
    profile.models.splice(idx, 1);
  }
  syncDefaultModel();
  void save({ quiet: true });
}

function resolveModelRef(raw: string): string {
  const s = String(raw || '').trim();
  const opts = enabledModelOptions.value;
  if (s && opts.some((o) => o.value === s)) return s;
  return opts[0]?.value || '';
}

function syncDefaultModel() {
  llmDefaultModel.value = resolveModelRef(llmDefaultModel.value);
}

function assign(target: FilmProviderBlock, src?: FilmProviderBlock) {
  target.protocol = src?.protocol ?? '';
  target.provider = src?.provider ?? '';
  target.base_url = src?.base_url ?? '';
  target.api_key = src?.api_key ?? '';
  target.model = src?.model ?? '';
  target.enabled = src?.enabled !== false;
  // Only persist models the user enabled after refresh — do not invent from legacy model field.
  const models = Array.isArray(src?.models) ? src!.models! : [];
  target.models = models
    .map((m) => ({
      id: String(m.id || '').trim(),
      label: String(m.label || m.id || '').trim(),
      enabled: m.enabled !== false,
    }))
    .filter((m) => m.id);
}

async function load() {
  loadError.value = '';
  try {
    const data = await api.settings.getFilmSettings();
    assign(generation, data.providers?.generation);
    genProfiles.value = parseGenProfilesFromSettings(data);
    imgProfiles.value = parseImgProfilesFromSettings(data);
    if (!generation.provider && genProfiles.value.length) {
      generation.provider = genProfiles.value[0].id;
    }
    genDefaultModel.value = data.providers?.generation_default_model || '';
    imgDefaultModel.value = data.providers?.image_default_model || '';
    syncGenDefaultModel();
    syncImgDefaultModel();
    paths.value = data.paths ?? {};

    try {
      const presets = await api.settings.listLlmPresets();
      llmPresets.value = presets.items || [];
      llmDialects.value =
        presets.reasoning_dialects?.length ? presets.reasoning_dialects : [...DEFAULT_LLM_DIALECTS];
    } catch {
      llmDialects.value = [...DEFAULT_LLM_DIALECTS];
      const catalog = (data.catalog || []).filter((c) => c.kind === 'llm');
      llmPresets.value = catalog.map((c: any) => ({
        id: String(c.id || ''),
        label: String(c.label || c.id || ''),
        protocol: String(c.protocol || 'openai'),
        base_url: String(c.base_url || ''),
        auth_style: String(c.auth_style || 'bearer'),
        default_dialect: String(c.default_dialect || ''),
        models: Array.isArray(c.models) ? c.models : [],
      }));
    }

    const profiles = (data.providers?.llm_profiles || []).map((p) => {
      const np = newProfile(p);
      const wasMake = /^(danmo-make|make|danqing)$/i.test(np.id) || /danmo make/i.test(np.label || '');
      if (wasMake) {
        np.id = normalizeLegacyProviderId(np.id);
        np.label = 'OpenAI · Chat';
        np.preset_id = 'openai-chat';
        if (!np.base_url || /127\.0\.0\.1:7800/.test(np.base_url)) {
          np.base_url = llmUrlPlaceholder;
        }
      }
      return np;
    });
    llmProfiles.value = profiles.length
      ? profiles
      : [
          newProfile({
            id: 'openai-chat',
            label: 'OpenAI · Chat',
            protocol: data.providers?.llm?.protocol || 'openai',
            base_url: data.providers?.llm?.base_url || llmUrlPlaceholder,
            api_key: data.providers?.llm?.api_key || '',
            preset_id: 'openai-chat',
            default_dialect: 'openai',
            models: [
              {
                id: data.providers?.llm?.model || 'gpt-4.1',
                label: data.providers?.llm?.model || 'GPT-4.1',
                enabled: true,
                reasoning_dialect: 'openai',
              },
            ],
          }),
        ];
    llmDefaultModel.value =
      data.providers?.llm_default_model ||
      data.providers?.llm?.model ||
      enabledModelOptions.value[0]?.value ||
      '';
    llmModelParams.value = { ...(data.providers?.llm_model_params || {}) };
    videoModelParams.value = { ...(data.providers?.video_model_params || {}) };
    imageModelParams.value = { ...(data.providers?.image_model_params || {}) };
    llmDefaultContextWindow.value = Number(data.providers?.llm_default_context_window) || 256000;
    llmDefaultMaxOutput.value = Number(data.providers?.llm_default_max_output_tokens) || 128000;
    llmDefaultTemperature.value =
      data.providers?.llm_default_temperature != null
        ? Number(data.providers.llm_default_temperature)
        : 0.35;
    llmDefaultTopP.value =
      data.providers?.llm_default_top_p != null ? Number(data.providers.llm_default_top_p) : 0;
    llmDefaultThinkingMode.value =
      String(data.providers?.llm_default_thinking_mode || 'off').trim() || 'off';
    syncDefaultModel();
  } catch (e: any) {
    loadError.value = e?.response?.data?.detail || e?.message || String(e);
  }
}

function payload(): FilmSettingsPayload {
  const firstModel = enabledModelOptions.value[0]?.value || '';
  const profiles = llmProfiles.value.map((p) => ({
    ...p,
    id: p.id.trim() || `llm-${Date.now().toString(36)}`,
    label: (p.label || '').trim() || p.id,
    protocol: p.protocol || 'openai',
    base_url: (p.base_url || '').trim(),
    api_key: p.api_key ?? '',
    auth_style: p.auth_style || 'bearer',
    default_dialect: p.default_dialect || '',
    preset_id: p.preset_id || '',
    enabled: Boolean(p.enabled),
    models: p.models
      .filter((m) => m.id.trim())
      .map((m) => ({
        id: m.id.trim(),
        label: (m.label || m.id).trim(),
        enabled: Boolean(m.enabled),
        reasoning_dialect: m.reasoning_dialect || '',
      })),
  }));
  return {
    providers: {
      llm: {
        protocol: profiles[0]?.protocol || 'openai',
        provider: profiles[0]?.id || 'openai-chat',
        base_url: profiles[0]?.base_url || llmUrlPlaceholder,
        api_key: profiles[0]?.api_key || '',
        model: llmDefaultModel.value || firstModel,
      },
      llm_profiles: profiles,
      llm_default_model: llmDefaultModel.value || firstModel,
      llm_default_context_window: llmDefaultContextWindow.value,
      llm_default_max_output_tokens: llmDefaultMaxOutput.value,
      llm_default_temperature: llmDefaultTemperature.value,
      llm_default_top_p: llmDefaultTopP.value,
      llm_default_thinking_mode: llmDefaultThinkingMode.value,
      llm_model_params: { ...llmModelParams.value },
      video_model_params: { ...videoModelParams.value },
      image_model_params: { ...imageModelParams.value },
      generation: {
        ...generation,
        provider: genDefaultModel.value.includes('/')
          ? genDefaultModel.value.split('/')[0]
          : generation.provider || genProfiles.value[0]?.id || '',
      },
      generation_profiles: genProfiles.value.map((p) => ({
        id: p.id.trim(),
        label: (p.label || '').trim() || p.id,
        vendor: normalizeGenVendor(p.vendor),
        base_url: (p.base_url || '').trim(),
        api_key: p.api_key ?? '',
        enabled: Boolean(p.enabled),
        models: (p.models || [])
          .filter((m) => m.id?.trim())
          .map((m) => ({
            id: m.id.trim(),
            label: (m.label || m.id).trim(),
            enabled: Boolean(m.enabled),
          })),
      })),
      generation_default_model: genDefaultModel.value || enabledGenModelRefs()[0]?.value || '',
      image_profiles: imgProfiles.value.map((p) => ({
        id: p.id.trim(),
        label: (p.label || '').trim() || p.id,
        vendor: normalizeImgVendor(p.vendor),
        base_url: (p.base_url || '').trim(),
        api_key: p.api_key ?? '',
        enabled: Boolean(p.enabled),
        models: (p.models || [])
          .filter((m) => m.id?.trim())
          .map((m) => ({
            id: m.id.trim(),
            label: (m.label || m.id).trim(),
            enabled: Boolean(m.enabled),
          })),
      })),
      image_default_model: imgDefaultModel.value || enabledImgModelRefs()[0]?.value || '',
    },
  };
}

async function save(opts?: { quiet?: boolean }) {
  saving.value = true;
  saveError.value = '';
  if (!opts?.quiet) saveMsg.value = '';
  try {
    await api.settings.updateFilmSettings(payload());
    if (!opts?.quiet) {
      saveMsg.value = $tt('settings.saveProvidersOk');
      toast.success($tt('settings.saveProvidersOk'));
    }
    await load();
  } catch (e: any) {
    saveError.value = e?.response?.data?.detail || e?.message || String(e);
  } finally {
    saving.value = false;
  }
}

async function refreshProviderCatalog(
  profile: FilmGenProfile,
  media: 'video' | 'image',
  state: Record<string, { loading?: boolean; error?: string; items?: Array<{ id: string; label: string }> }>,
  onSync: () => void,
  okKey: string,
) {
  const id = String(profile.id || '').trim();
  if (!id) return;
  state[id] = { ...(state[id] || {}), loading: true, error: '' };
  try {
    await save({ quiet: true });
    // Make has a real media tag — pass it. Cloud catalogs / untyped vendor lists:
    // omit media so everything is shown; user enables what they want.
    const params: Record<string, unknown> = { provider: id };
    if (normalizeGenVendor(profile.vendor) === 'danmo-make' || normalizeImgVendor(profile.vendor) === 'danmo-make') {
      params.media = media;
    }
    const data = (await api.models.list(params)) as {
      items?: Array<Record<string, unknown>>;
    };
    const items = (data.items || [])
      .map((row) => {
        let mid = String(row.id || '').trim();
        if (mid.includes('/')) mid = mid.slice(mid.indexOf('/') + 1);
        if (!mid) return null;
        const label = String(row.label || mid).trim() || mid;
        return { id: mid, label };
      })
      .filter((x): x is { id: string; label: string } => Boolean(x));
    state[id] = { loading: false, error: '', items };
    if (!items.length) {
      state[id].error = $tt('settings.llmRefreshModelsEmpty');
      toast.error($tt('settings.llmRefreshModelsEmpty'));
      return;
    }
    const prevOn = new Set((profile.models || []).filter((m) => m.enabled).map((m) => m.id));
    profile.models = items.map((m) => ({
      id: m.id,
      label: m.label,
      enabled: prevOn.has(m.id),
    }));
    onSync();
    await save({ quiet: true });
    toast.success($tt(okKey, { n: items.length }));
  } catch (e: any) {
    const msg = e?.response?.data?.detail || e?.message || String(e);
    state[id] = { ...(state[id] || {}), loading: false, error: msg };
    toast.error(msg);
  }
}

async function refreshGenModels(profile: FilmGenProfile) {
  await refreshProviderCatalog(
    profile,
    'video',
    genRefreshState,
    syncGenDefaultModel,
    'settings.genRefreshModelsOk',
  );
}

async function refreshImgModels(profile: FilmGenProfile) {
  await refreshProviderCatalog(
    profile,
    'image',
    imgRefreshState,
    syncImgDefaultModel,
    'settings.imgRefreshModelsOk',
  );
}

onMounted(async () => {
  await load();
});
</script>

<style scoped>
/* Full-height rail column — same 贯穿 divider chrome as create-page tab strip */
.settings-page--rail {
  display: flex;
  width: 100%;
  height: 100%;
  overflow: hidden;
  box-sizing: border-box;
}

.settings-sidebar {
  width: 168px;
  flex-shrink: 0;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  padding: 12px 8px 20px 12px;
  border-right: 0.5px solid var(--dq-border-subtle);
  border-radius: 0;
  background: color-mix(in srgb, var(--dq-surface-elevated) 55%, transparent);
  box-shadow: inset 0 1px 0 color-mix(in srgb, white 4%, transparent);
  box-sizing: border-box;
}

.settings-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
}

.settings-nav__item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 40px;
  padding: 9px 10px 9px 12px;
  border: 0.5px solid transparent;
  border-radius: 11px;
  background: transparent;
  color: var(--dq-label-secondary);
  cursor: pointer;
  text-align: left;
  font-family: inherit;
  transition: background 0.15s, border-color 0.15s, box-shadow 0.15s, color 0.15s;
}

.settings-nav__item::before {
  content: '';
  position: absolute;
  left: 4px;
  top: 10px;
  bottom: 10px;
  width: 3px;
  border-radius: 999px;
  background: transparent;
  transition: background 0.15s;
}

.settings-nav__item:hover:not(.settings-nav__item--active) {
  background: color-mix(in srgb, var(--dq-fill-control) 65%, transparent);
  color: var(--dq-label-primary);
}

.settings-nav__item:hover:not(.settings-nav__item--active) .settings-nav__label {
  color: var(--dq-label-primary);
}

.settings-nav__item--active {
  background: color-mix(in srgb, var(--dq-accent) 11%, transparent);
  border-color: color-mix(in srgb, var(--dq-accent) 32%, transparent);
  box-shadow: 0 2px 10px color-mix(in srgb, var(--dq-accent) 8%, transparent);
  color: var(--dq-label-primary);
}

.settings-nav__item--active::before {
  background: var(--dq-accent);
}

.settings-nav__icon {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--dq-label-tertiary);
  background: transparent;
  border: none;
}

.settings-nav__item--active .settings-nav__icon {
  color: var(--dq-accent);
}

.settings-nav__label {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 650;
  letter-spacing: var(--dq-tracking-tight, -0.01em);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--dq-label-secondary);
}

.settings-nav__item--active .settings-nav__label {
  color: var(--dq-label-primary);
  font-weight: 700;
}

.settings-content {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 20px 28px 0;
  box-sizing: border-box;
}

.settings-content__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding-bottom: 16px;
}

.settings-content :deep(.lv-step-save) {
  margin-top: 0;
  padding-bottom: 16px;
}

.settings-provider-drawer__body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 4px 0 8px;
}

.settings-provider-drawer__hint {
  margin: 0;
  font-size: 13px;
  line-height: 1.45;
  color: var(--dq-label-secondary);
}

.settings-provider-drawer__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.settings-provider-drawer__label {
  font-size: 12px;
  font-weight: 600;
  color: var(--dq-label-secondary);
}

.settings-provider-drawer__select {
  width: 100%;
}

.settings-provider-drawer__meta {
  margin: -8px 0 0;
  font-size: 12px;
  line-height: 1.4;
  color: var(--dq-label-tertiary);
  word-break: break-all;
}

.settings-provider-drawer__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 8px;
}

.settings-panel__error {
  margin: 0 0 12px;
  color: var(--dq-danger, #ff453a);
  font-size: 13px;
}
</style>

<style>
/* Shared panel chrome (child components) — open surface, no nested boxes */
.settings-page--rail .settings-grouped-form,
.settings-page--rail .dq-pref-pane.settings-pref-pane-form {
  border: none;
  border-radius: 0;
  background: transparent;
  overflow: visible;
}

.settings-page--rail .dq-pref-pane.settings-pref-pane-form .dq-pref-row {
  padding-left: 0;
  padding-right: 0;
  min-height: 40px;
}

.settings-page--rail .dq-pref-pane.settings-pref-pane-form .dq-pref-row + .dq-pref-row {
  border-top: 0.5px solid color-mix(in srgb, var(--dq-border-subtle) 55%, transparent);
}

.settings-panel {
  display: flex;
  flex-direction: column;
  gap: 28px;
  max-width: 640px;
}

.settings-panel__head h2 {
  margin: 0 0 4px;
  font-size: var(--dq-font-size-title, 18px);
  font-weight: 650;
  color: var(--dq-label-primary);
}

.settings-panel__head p {
  margin: 0;
  font-size: 13px;
  line-height: 1.45;
  color: var(--dq-label-secondary);
}

.settings-panel__head--row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.settings-panel__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 4px;
}

.settings-panel__ok {
  margin: 0;
  font-size: 13px;
  color: var(--dq-success, #30d158);
}

.settings-panel__error {
  margin: 0;
  font-size: 13px;
  color: var(--dq-danger, #ff453a);
}

.settings-panel__hint {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--dq-label-tertiary);
}

.film-llm-default {
  margin-bottom: 4px;
}

.film-llm-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 0;
  margin: 0;
  border: none;
  border-radius: 0;
  background: transparent;
}

.film-llm-card + .film-llm-card {
  margin-top: 12px;
  padding-top: 32px;
  border-top: 0.5px solid color-mix(in srgb, var(--dq-label-tertiary) 28%, transparent);
}

.film-llm-card__head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 0;
  padding: 0;
  flex-wrap: wrap;
  background: transparent;
}

.film-llm-card__title.dq-input,
.film-llm-card__title {
  flex: 1 1 200px;
  min-width: 160px;
  font-size: 16px;
  font-weight: 650;
  letter-spacing: -0.01em;
  border-color: transparent !important;
  background: transparent !important;
  box-shadow: none !important;
  padding-left: 0;
}

.film-llm-card__title.dq-input:hover,
.film-llm-card__title.dq-input:focus {
  border-color: var(--dq-border-subtle) !important;
  background: color-mix(in srgb, var(--dq-fill-tertiary) 40%, transparent) !important;
}

.film-llm-card__toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--dq-label-secondary);
  white-space: nowrap;
}

.film-llm-models {
  margin-top: 10px;
}

.film-llm-models__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--dq-label-primary);
}

.film-llm-models__row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.film-llm-models__enable {
  flex-shrink: 0;
}

.film-llm-models__id,
.film-llm-models__label {
  flex: 1 1 140px;
  min-width: 100px;
}
</style>
