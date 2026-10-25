<template>
  <div class="settings-page settings-page--v2">
    <aside class="settings-sidebar" :aria-label="$tt('settings.title')">
      <div class="settings-sidebar__title">
        <DqIcon><Setting /></DqIcon>
        {{ $tt('settings.title') }}
      </div>
      <nav class="dq-download-menu" role="navigation" :aria-label="$tt('settings.title')">
        <button
          v-for="item in navItems"
          :key="item.id"
          type="button"
          class="dq-download-menu__item"
          :class="{ 'is-active': activeSection === item.id }"
          @click="activeSection = item.id"
        >
          <DqIcon class="dq-download-menu__icon"><component :is="item.icon" /></DqIcon>
          <span class="dq-download-menu__label">{{ $tt(item.labelKey) }}</span>
        </button>
      </nav>
    </aside>

    <div class="settings-content-area">
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
        :saving="saving"
        :save-msg="saveMsg"
        :save-error="saveError"
        :refresh-state="llmRefreshState"
        @update:default-model="setDefaultModel"
        @add-profile="addProfileFromPreset"
        @remove-profile="removeProfile"
        @set-protocol="setProfileProtocol"
        @set-default-dialect="setDefaultDialect"
        @set-model-dialect="setModelDialect"
        @apply-preset="applyPresetToProfile"
        @refresh-models="refreshProviderModels"
        @toggle-remote-model="toggleRemoteModel"
        @save="save"
      />

      <SettingsGenerationPanel
        v-else
        :generation="generation"
        :volcengine="volcengine"
        :dashscope="dashscope"
        :minimax="minimax"
        :gen-providers="genProviders"
        :saving="saving"
        :checking="checking"
        :save-msg="saveMsg"
        :save-error="saveError"
        :provider-status="providerStatus"
        @set-provider="setGenProvider"
        @save="save"
        @test="refreshStatus"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import {
  MagicStick,
  Monitor,
  Setting,
  VideoCamera,
} from '@danqing/dq-shell';
import { toast } from '@/utils/feedback';
import { $tt } from '@/utils/i18n';
import {
  api,
  type FilmLlmPreset,
  type FilmLlmProfile,
  type FilmProviderBlock,
  type FilmSettingsPayload,
} from '@/utils/api';
import { DQ_STORAGE, getItem, setItem } from '@/utils/storage';
import SettingsGeneralPanel from '@/components/settings/SettingsGeneralPanel.vue';
import SettingsLlmPanel from '@/components/settings/SettingsLlmPanel.vue';
import SettingsGenerationPanel from '@/components/settings/SettingsGenerationPanel.vue';

type SectionId = 'general' | 'llm' | 'generation';

interface NavItem {
  id: SectionId;
  labelKey: string;
  icon: object;
}

const SECTION_IDS: SectionId[] = ['general', 'llm', 'generation'];

const navItems: NavItem[] = [
  { id: 'general', labelKey: 'settings.general', icon: Monitor },
  { id: 'llm', labelKey: 'settings.llmProvider', icon: MagicStick },
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

const genProviders = [
  { value: 'volcengine', label: 'Volcengine Ark Seedance' },
  { value: 'dashscope', label: 'DashScope 万相' },
  { value: 'minimax', label: 'MiniMax Hailuo' },
];

const PATH_LABEL_KEYS: Record<string, string> = {
  assets_root: 'settings.pathAssetsRoot',
  film_db: 'settings.pathFilmDb',
  settings_file: 'settings.pathSettingsFile',
  user_data_dir: 'settings.pathUserDataDir',
  workspace_dir: 'settings.pathWorkspaceDir',
};

function emptyBlock(): FilmProviderBlock {
  return { protocol: '', provider: '', base_url: '', api_key: '', model: '' };
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
const volcengine = reactive<FilmProviderBlock>(emptyBlock());
const dashscope = reactive<FilmProviderBlock>(emptyBlock());
const minimax = reactive<FilmProviderBlock>(emptyBlock());
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
const llmDefaultContextWindow = ref(256000);
const llmDefaultMaxOutput = ref(128000);
const llmDefaultTemperature = ref(0.35);
const llmDefaultTopP = ref(0);
const llmDefaultThinkingMode = ref('off');
const llmRefreshState = reactive<
  Record<string, { loading?: boolean; error?: string; items?: Array<{ id: string; label: string }> }>
>({});

type ProviderStatus = {
  llm?: { provider_id?: string; available?: boolean };
  generation?: {
    provider_id?: string;
    health?: { ok?: boolean; error?: string };
  };
};

const loadError = ref('');
const saveError = ref('');
const saveMsg = ref('');
const providerStatus = ref<ProviderStatus | null>(null);
const saving = ref(false);
const checking = ref(false);

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

function setGenProvider(v: unknown) {
  const raw = String(v ?? '').trim();
  const map: Record<string, string> = {
    volcengine: 'volcengine',
    'Volcengine Ark Seedance': 'volcengine',
    dashscope: 'dashscope',
    'DashScope 万相': 'dashscope',
    minimax: 'minimax',
    'MiniMax Hailuo': 'minimax',
    // Migrate legacy Make selection → Volcengine
    'danmo-make': 'volcengine',
    make: 'volcengine',
    'Danmo Make': 'volcengine',
  };
  const s = map[raw] || raw;
  if (s) generation.provider = s;
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
}

async function load() {
  loadError.value = '';
  try {
    const data = await api.settings.getFilmSettings();
    assign(generation, data.providers?.generation);
    if (!generation.provider || generation.provider === 'danmo-make' || generation.provider === 'make') {
      generation.provider = 'volcengine';
    }
    assign(volcengine, data.providers?.volcengine);
    assign(dashscope, data.providers?.dashscope);
    assign(minimax, data.providers?.minimax);
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
      generation: { ...generation },
      volcengine: { ...volcengine },
      dashscope: { ...dashscope },
      minimax: { ...minimax },
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
    if (!opts?.quiet) await refreshStatus();
  } catch (e: any) {
    saveError.value = e?.response?.data?.detail || e?.message || String(e);
  } finally {
    saving.value = false;
  }
}

async function refreshStatus() {
  checking.value = true;
  try {
    const data = (await api.settings.providerStatus()) as ProviderStatus;
    providerStatus.value = data;
  } catch (e: any) {
    providerStatus.value = null;
    saveError.value = e?.response?.data?.detail || e?.message || String(e);
  } finally {
    checking.value = false;
  }
}

onMounted(async () => {
  await load();
  await refreshStatus();
});
</script>

<style scoped>
.settings-page--v2 {
  display: flex;
  width: 100%;
  height: 100%;
  overflow: hidden;
  box-sizing: border-box;
}

.settings-sidebar {
  width: 220px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 20px 12px 24px 20px;
  border-right: 0.5px solid var(--dq-border-subtle);
  box-sizing: border-box;
  overflow-y: auto;
}

.settings-sidebar__title {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--dq-label-secondary);
}

.settings-content-area {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  padding: 20px 28px 32px;
  box-sizing: border-box;
}

.settings-panel__error {
  margin: 0 0 12px;
  color: var(--dq-danger, #ff453a);
  font-size: 13px;
}
</style>

<style>
/* Shared panel chrome (child components) — open surface, no nested boxes */
.settings-page--v2 .settings-grouped-form,
.settings-page--v2 .dq-pref-pane.settings-pref-pane-form {
  border: none;
  border-radius: 0;
  background: transparent;
  overflow: visible;
}

.settings-page--v2 .dq-pref-pane.settings-pref-pane-form .dq-pref-row {
  padding-left: 0;
  padding-right: 0;
  min-height: 40px;
}

.settings-page--v2 .dq-pref-pane.settings-pref-pane-form .dq-pref-row + .dq-pref-row {
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
