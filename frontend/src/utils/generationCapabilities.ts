import type { LongVideoChainMode } from '@/types';
import { isBerniniRenderer, supportsBerniniReferenceImages } from '@/utils/videoEditSource';

export type GenerationCapabilityMode =
  | 'image_create'
  | 'image_edit'
  | 'video_animate'
  | 'video_first_last'
  | 'video_reference_r2v'
  | 'video_reference_v2v';

/** Capability catalog tokens derived from filled media slots (not an HTTP body field). */
export type VideoGenerationMode = 't2v' | 'i2v' | 'flf' | 'r2v' | 'v2v';

export interface ModelGenerationCapabilities {
  /** null = vendor/registry did not advertise a limit (do not invent one). */
  max_reference_images: number | null;
  supports_r2v: boolean;
  supports_v2v: boolean;
  supports_first_last: boolean;
  supports_flf: boolean;
  supports_img2img: boolean;
}

function paramRecord(modelConfig: Record<string, unknown> | null | undefined): Record<string, unknown> {
  if (!modelConfig) return {};
  const nested = modelConfig.parameters;
  if (nested && typeof nested === 'object' && !Array.isArray(nested)) {
    return { ...modelConfig, ...(nested as Record<string, unknown>) };
  }
  return modelConfig;
}

export function modelGenerationCapabilities(
  modelConfig: Record<string, unknown> | null | undefined,
): ModelGenerationCapabilities {
  const p = paramRecord(modelConfig);
  const bernini = isBerniniRenderer(p);
  const rawMax = p.max_reference_images;
  // Only trust an explicit number from registry/provider. Missing ≠ 0.
  // Bernini is a local renderer with a known fixed slot count.
  let maxRef: number | null =
    typeof rawMax === 'number' && rawMax >= 0 ? rawMax : bernini ? 5 : null;
  if (bernini && maxRef === 0) maxRef = 5;

  const supportsFlf =
    p.supports_first_last === true ||
    p.supports_flf === true ||
    p.flf === true ||
    (bernini && p.video_edit_source_mode === 'first_frame');

  const actions = modelConfig?.actions as Record<string, unknown> | undefined;
  return {
    max_reference_images: maxRef,
    supports_r2v: p.supports_r2v === true || bernini || supportsBerniniReferenceImages(p),
    supports_v2v:
      p.supports_v2v === true ||
      p.supports_video_reference === true ||
      p.supports_first_clip === true,
    supports_first_last: supportsFlf,
    supports_flf: supportsFlf,
    supports_img2img: p.supports_img2img === true || Boolean(actions?.edit),
  };
}

/** Map long-video chain_mode → capability catalog token (for modes[] checks). */
export function chainModeToVideoMode(chainMode: LongVideoChainMode): VideoGenerationMode {
  switch (chainMode) {
    case 'reference_r2v':
      return 'r2v';
    case 'first_last':
      return 'flf';
    case 'keyframe_only':
    case 'last_frame':
      return 'i2v';
    default:
      return 'i2v';
  }
}

export function chainModeToCapabilityMode(
  chainMode: LongVideoChainMode,
  phase: 'keyframe' | 'segment',
): GenerationCapabilityMode {
  if (phase === 'keyframe') return 'image_create';
  if (chainMode === 'reference_r2v') return 'video_reference_r2v';
  if (chainMode === 'first_last') return 'video_first_last';
  return 'video_animate';
}

function modelModesList(modelConfig: Record<string, unknown> | null | undefined): string[] | null {
  if (!modelConfig) return null;
  const raw = modelConfig.modes ?? (modelConfig.parameters as Record<string, unknown> | undefined)?.modes;
  if (!Array.isArray(raw) || raw.length === 0) return null;
  return raw.map((x) => String(x).trim().toLowerCase()).filter(Boolean);
}

/** Map Film capability check → port mode token for modes[] matching. */
function capabilityToPortMode(mode: GenerationCapabilityMode): string | null {
  switch (mode) {
    case 'video_animate':
      return 'i2v';
    case 'video_first_last':
      return 'flf';
    case 'video_reference_r2v':
      return 'r2v';
    case 'video_reference_v2v':
      return 'v2v';
    case 'image_create':
      return 't2i';
    case 'image_edit':
      return 'i2i';
    default:
      return null;
  }
}

/** Returns an error message when unsupported; never silent fallback. */
export function assertProviderSupports(
  mode: GenerationCapabilityMode,
  modelConfig: Record<string, unknown> | null | undefined,
): string | null {
  const portMode = capabilityToPortMode(mode);
  const modes = modelModesList(modelConfig);
  if (portMode && modes) {
    const ok = modes.includes(portMode) || (portMode === 'i2i' && (modes.includes('img2img') || modes.includes('edit')));
    if (!ok) {
      return `Model does not support mode ${portMode} (supports: ${modes.join(', ')}).`;
    }
    return null;
  }

  const caps = modelGenerationCapabilities(modelConfig);
  switch (mode) {
    case 'image_edit':
      if (!caps.supports_img2img) {
        return 'Model does not support image-to-image (img2img).';
      }
      return null;
    case 'video_reference_r2v':
      if (!caps.supports_r2v) {
        return 'Model does not support reference-to-video (R2V).';
      }
      return null;
    case 'video_reference_v2v':
      if (!caps.supports_v2v) {
        return 'Model does not support video-to-video (V2V).';
      }
      return null;
    case 'video_first_last':
      if (!caps.supports_first_last && !caps.supports_flf) {
        return 'Model does not support first-last-frame (FLF) video.';
      }
      return null;
    case 'image_create':
    case 'video_animate':
      return null;
    default:
      return null;
  }
}

export function validateReferenceRequirement(
  requiredRefCount: number,
  modelConfig: Record<string, unknown> | null | undefined,
): string | null {
  if (requiredRefCount <= 0) return null;
  const max = modelGenerationCapabilities(modelConfig).max_reference_images;
  // Unknown (null): allow; provider fails loud if refs unsupported.
  if (max === null) return null;
  if (max === 0) {
    return `This model accepts no reference images but ${requiredRefCount} reference(s) are required.`;
  }
  return null;
}

export function capReferenceAssetIds(
  ids: string[],
  modelConfig: Record<string, unknown> | null | undefined,
): { ids: string[]; capped: boolean } {
  const max = modelGenerationCapabilities(modelConfig).max_reference_images;
  if (max === null) return { ids: [...ids], capped: false };
  if (max <= 0) return { ids: [], capped: ids.length > 0 };
  if (ids.length <= max) return { ids: [...ids], capped: false };
  return { ids: ids.slice(0, max), capped: true };
}
