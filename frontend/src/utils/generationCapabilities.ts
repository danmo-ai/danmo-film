import type { LongVideoChainMode } from '@/types';
import { isBerniniRenderer, supportsBerniniReferenceImages } from '@/utils/videoEditSource';

export type GenerationCapabilityMode =
  | 'image_create'
  | 'image_edit'
  | 'video_animate'
  | 'video_first_last'
  | 'video_reference_r2v';

export interface ModelGenerationCapabilities {
  max_reference_images: number;
  supports_r2v: boolean;
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
  let maxRef = typeof rawMax === 'number' && rawMax >= 0 ? rawMax : bernini ? 5 : 0;
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
    supports_first_last: supportsFlf,
    supports_flf: supportsFlf,
    supports_img2img: p.supports_img2img === true || Boolean(actions?.edit),
  };
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

/** Returns an error message when unsupported; never silent fallback. */
export function assertProviderSupports(
  mode: GenerationCapabilityMode,
  modelConfig: Record<string, unknown> | null | undefined,
): string | null {
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
  if (!max) {
    return `This model accepts no reference images but ${requiredRefCount} reference(s) are required.`;
  }
  return null;
}

export function capReferenceAssetIds(
  ids: string[],
  modelConfig: Record<string, unknown> | null | undefined,
): { ids: string[]; capped: boolean } {
  const max = modelGenerationCapabilities(modelConfig).max_reference_images;
  if (max <= 0) return { ids: [], capped: ids.length > 0 };
  if (ids.length <= max) return { ids: [...ids], capped: false };
  return { ids: ids.slice(0, max), capped: true };
}
