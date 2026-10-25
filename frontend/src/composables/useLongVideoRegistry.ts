import { computed, ref } from 'vue';
import { useRegistryStore } from '@/stores/registry';
import { api } from '@/utils/api';
import { $mn, $tt } from '@/utils/i18n';
import {
  buildResolutionSizeOptions,
  pickResolutionForModel,
  type ResolutionSizeOption,
} from '@/utils/registryParamSchema';
import { isBerniniRenderer } from '@/utils/videoEditSource';

export interface LongVideoProfile {
  keyframe_models: string[];
  segment_models: string[];
  max_target_duration_sec: number;
  segment_model_id: string;
}

type RegistryModelRow = {
  media?: string;
  actions?: Record<string, unknown>;
  name?: string | { zh?: string; en?: string };
  parameters?: Record<string, unknown>;
  commercial_use_allowed?: boolean;
};

type CloudVideoModel = {
  ref: string;
  label: string;
  provider_id?: string;
};

const cloudVideoModels = ref<CloudVideoModel[]>([]);
const cloudImageModels = ref<CloudVideoModel[]>([]);
const cloudDefaultModel = ref('');
const cloudDefaultImageModel = ref('');
let cloudLoadedAt = 0;

const VENDOR_LABEL: Record<string, string> = {
  'danmo-make': 'settings.genVendorMake',
  volcengine: 'settings.genVendorVolcengine',
  dashscope: 'settings.genVendorDashscope',
  minimax: 'settings.genVendorMinimax',
  openai: 'settings.imgVendorOpenAI',
};

function modelRow(reg: ReturnType<typeof useRegistryStore>['registry'], modelId: string): RegistryModelRow | undefined {
  const bare = bareModelId(modelId);
  return (reg?.models?.[modelId] || reg?.models?.[bare]) as RegistryModelRow | undefined;
}

function bareModelId(ref: string): string {
  const s = String(ref || '').trim();
  const i = s.indexOf('/');
  if (i > 0 && i < s.length - 1) return s.slice(i + 1);
  return s;
}

function providerIdOf(ref: string): string {
  const s = String(ref || '').trim();
  const i = s.indexOf('/');
  if (i > 0) return s.slice(0, i);
  return '';
}

function profileFromRegistry(reg: NonNullable<ReturnType<typeof useRegistryStore>['registry']>): LongVideoProfile | null {
  for (const m of Object.values(reg.models ?? {})) {
    const cfg = m as RegistryModelRow & { id?: string };
    const p = cfg.parameters?.long_video_profile as
      | {
          keyframe_models?: string[];
          segment_models?: string[];
          max_target_duration_sec?: number;
        }
      | undefined;
    if (p && typeof p === 'object') {
      return {
        keyframe_models: p.keyframe_models ?? [],
        segment_models: p.segment_models ?? [],
        max_target_duration_sec: p.max_target_duration_sec ?? 120,
        segment_model_id: cfg.id || p.segment_models?.[0] || 'wan-2.2-i2v-14b',
      };
    }
  }
  return null;
}

function sortWithSuggestions(ids: string[], suggestions: string[]): string[] {
  const seen = new Set<string>();
  const ordered: string[] = [];
  for (const id of suggestions) {
    if (ids.includes(id) && !seen.has(id)) {
      ordered.push(id);
      seen.add(id);
    }
  }
  for (const id of ids.sort()) {
    if (!seen.has(id)) ordered.push(id);
  }
  return ordered;
}

function mapCloudRows(
  items: Array<Record<string, unknown>>,
  profileLabels: Map<string, string>,
): CloudVideoModel[] {
  return items
    .map((row) => {
      const ref = String(row.id || '').trim();
      if (!ref) return null;
      const pid = String(row.provider_id || providerIdOf(ref) || '').trim();
      const bare = bareModelId(ref);
      const labelRaw = String(row.label || bare).trim() || bare;
      const vendor = String(row.vendor || '').trim();
      const plabel =
        String(row.provider_label || '').trim()
        || profileLabels.get(pid)
        || (VENDOR_LABEL[pid] ? $tt(VENDOR_LABEL[pid]) : '')
        || (vendor && VENDOR_LABEL[vendor] ? $tt(VENDOR_LABEL[vendor]) : '')
        || pid;
      return {
        ref: ref.includes('/') ? ref : pid ? `${pid}/${ref}` : ref,
        label: plabel ? `${plabel} / ${labelRaw}` : labelRaw,
        provider_id: pid,
      };
    })
    .filter((x): x is CloudVideoModel => Boolean(x));
}

function profileLabelMap(
  profiles: Array<{ id?: string; label?: string; vendor?: string } | string> | undefined,
): Map<string, string> {
  const profileLabels = new Map<string, string>();
  for (const p of profiles || []) {
    if (typeof p === 'string') continue;
    const id = String(p.id || '').trim();
    if (!id) continue;
    const label = String(p.label || id).trim() || id;
    profileLabels.set(id, label);
    if (p.vendor) {
      const vk = VENDOR_LABEL[String(p.vendor)];
      if (vk && !p.label) profileLabels.set(id, $tt(vk));
    }
  }
  return profileLabels;
}

async function loadCloudVideoModels(force = false) {
  const now = Date.now();
  if (
    !force
    && cloudVideoModels.value.length
    && cloudImageModels.value.length
    && now - cloudLoadedAt < 60_000
  ) {
    return;
  }
  try {
    const [videoRes, imageRes, settings] = await Promise.all([
      api.models.list({ media: 'video' }) as Promise<{ items?: Array<Record<string, unknown>> }>,
      api.models.list({ media: 'image' }) as Promise<{ items?: Array<Record<string, unknown>> }>,
      api.settings.getFilmSettings().catch(() => null),
    ]);
    const videoLabels = profileLabelMap(settings?.providers?.generation_profiles as any);
    const imageLabels = profileLabelMap(settings?.providers?.image_profiles as any);
    cloudVideoModels.value = mapCloudRows(videoRes.items || [], videoLabels);
    cloudImageModels.value = mapCloudRows(imageRes.items || [], imageLabels);
    const def = String(settings?.providers?.generation_default_model || '').trim();
    cloudDefaultModel.value =
      (def && cloudVideoModels.value.some((m) => m.ref === def) ? def : '')
      || cloudVideoModels.value[0]?.ref
      || '';
    const idef = String(settings?.providers?.image_default_model || '').trim();
    cloudDefaultImageModel.value =
      (idef && cloudImageModels.value.some((m) => m.ref === idef) ? idef : '')
      || cloudImageModels.value[0]?.ref
      || '';
    cloudLoadedAt = now;
  } catch {
    // Keep prior cloud list; create page may still use registry Make models.
  }
}

export function useLongVideoRegistry() {
  const registryStore = useRegistryStore();

  const profile = computed((): LongVideoProfile | null => {
    const reg = registryStore.registry;
    const fromReg = reg?.models ? profileFromRegistry(reg) : null;
    const registryKeyframeIds = listImageCreateModelIds();
    const registrySegmentIds = listVideoAnimateModelIds();
    const cloudImgIds = cloudImageModels.value.map((m) => m.ref);
    const cloudVidIds = cloudVideoModels.value.map((m) => m.ref);
    // Prefer Settings-enabled Image / Video provider model refs when present.
    const segmentIds = cloudVidIds.length ? cloudVidIds : registrySegmentIds;
    return {
      keyframe_models: cloudImgIds.length
        ? cloudImgIds
        : fromReg?.keyframe_models?.length
          ? sortWithSuggestions(registryKeyframeIds, fromReg.keyframe_models)
          : registryKeyframeIds,
      segment_models: cloudVidIds.length
        ? cloudVidIds
        : fromReg?.segment_models?.length
          ? sortWithSuggestions(registrySegmentIds, fromReg.segment_models)
          : registrySegmentIds,
      max_target_duration_sec: fromReg?.max_target_duration_sec ?? 120,
      segment_model_id:
        cloudDefaultModel.value
        || fromReg?.segment_model_id
        || segmentIds[0]
        || 'wan-2.2-i2v-14b',
    };
  });

  function listImageCreateModelIds(): string[] {
    const reg = registryStore.registry?.models ?? {};
    return Object.keys(reg).filter((id) => {
      const m = reg[id] as RegistryModelRow;
      if (!m) return false;
      if (m.media === 'image') return m.actions?.create !== false;
      return Boolean(m.actions?.create);
    });
  }

  function listVideoAnimateModelIds(): string[] {
    const reg = registryStore.registry?.models ?? {};
    return Object.keys(reg).filter((id) => {
      const m = reg[id] as RegistryModelRow;
      if (!m) return false;
      if (m.media === 'video') return m.actions?.animate !== false;
      return Boolean(m.actions?.animate);
    });
  }

  function modelOptions(ids: string[]) {
    const reg = registryStore.registry?.models ?? {};
    const cloudByRef = new Map([
      ...cloudVideoModels.value.map((m) => [m.ref, m] as const),
      ...cloudImageModels.value.map((m) => [m.ref, m] as const),
    ]);
    return ids.map((id) => {
      const cloud = cloudByRef.get(id);
      if (cloud) {
        return {
          label: cloud.label,
          value: id,
          commercialUseAllowed: true,
        };
      }
      const m = reg[id] as RegistryModelRow | undefined;
      return {
        label: m ? $mn(m, id) : id,
        value: id,
        commercialUseAllowed: Boolean(m?.commercial_use_allowed),
      };
    });
  }

  function modelLabel(modelId: string): string {
    const cloud =
      cloudVideoModels.value.find((m) => m.ref === modelId)
      || cloudImageModels.value.find((m) => m.ref === modelId);
    if (cloud) return cloud.label;
    const m = modelRow(registryStore.registry, modelId);
    if (!m?.name) return modelId;
    if (typeof m.name === 'string') return m.name;
    return m.name.zh || m.name.en || modelId;
  }

  function durationOptions(maxSec?: number): { label: string; value: number }[] {
    const max = maxSec ?? profile.value?.max_target_duration_sec ?? 120;
    const opts = [45, 60, 90, 120].filter((s) => s <= max);
    if (!opts.length) opts.push(30);
    return opts.map((value) => ({ value, label: `${value}s` }));
  }

  function modelParameters(modelId: string): Record<string, unknown> | undefined {
    return modelRow(registryStore.registry, modelId)?.parameters;
  }

  function modelHasBerniniRenderer(modelId: string): boolean {
    return isBerniniRenderer(modelParameters(modelId));
  }

  function defaultShiftForModel(modelId: string, fallback = 12): number {
    const params = modelParameters(modelId);
    const shift = params?.shift;
    if (typeof shift === 'number') return shift;
    const schema = params?.parameters as Record<string, { default?: number }> | undefined;
    const nested = schema?.shift?.default;
    return typeof nested === 'number' ? nested : fallback;
  }

  function resolutionOptionsForModel(modelId: string): ResolutionSizeOption[] {
    return buildResolutionSizeOptions(modelParameters(modelId));
  }

  function pickOutputSizeForModel(modelId: string, saved?: string | null): string | null {
    return pickResolutionForModel(modelParameters(modelId), saved);
  }

  async function loadRegistry(force = false) {
    await Promise.all([registryStore.load(force), loadCloudVideoModels(force)]);
  }

  return {
    profile,
    modelLabel,
    durationOptions,
    modelParameters,
    modelHasBerniniRenderer,
    defaultShiftForModel,
    listImageCreateModelIds,
    listVideoAnimateModelIds,
    modelOptions,
    resolutionOptionsForModel,
    pickOutputSizeForModel,
    loadRegistry,
  };
}
