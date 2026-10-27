/**
 * Shared identity-sheet / wardrobe-ref image generation for Cast panel
 * (episode CreateView + series Cast tab).
 */
import { ref, type Ref } from 'vue';
import { useRegistryStore } from '@/stores/registry';
import { useTasksStore } from '@/stores/tasks';
import { api, taskIdFromSubmitResponse } from '@/utils/api';
import { toast } from '@/utils/feedback';
import { $tt } from '@/utils/i18n';
import type { CharacterIdentityAngle, LongVideoCharacter } from '@/types';
import { appendImageInferenceFields } from '@/composables/useStructuralGuide';
import {
  PORTRAIT_NEGATIVE_PROMPT_EN,
  PORTRAIT_NEGATIVE_PROMPT_ZH,
  PORTRAIT_REFERENCE_SIZE,
  buildIdentityAnglePrompt,
  buildIdentityFrontPrompt,
  buildWardrobeRefPrompt,
  identityFrontAssetId,
  normalizeCharacterRoster,
  patchIdentityView,
  resolveLongVideoLocale,
} from '@/utils/longVideoProject';

export function useCastIdentityGeneration(opts: {
  getCharacters: () => LongVideoCharacter[];
  setCharacters: (next: LongVideoCharacter[]) => void;
  getModel: () => string;
  getStyleAnchor: () => string;
  getLocale: () => string;
  /** Optional project/series id for task metadata. */
  getOwnerId?: () => string;
  ensureSaved?: () => Promise<void>;
  /** Called after a successful image apply (e.g. quiet series persist). */
  afterSuccess?: () => Promise<void> | void;
  composeParams?: () => { steps?: unknown; guidance?: unknown };
}) {
  const registryStore = useRegistryStore();
  const tasksStore = useTasksStore();
  const portraitGeneratingKey: Ref<string | null> = ref(null);
  const batchGenerating = ref(false);

  function identityGenKey(ci: number, angle: CharacterIdentityAngle): string {
    return `id-${ci}-${angle}`;
  }

  function portraitKey(ci: number, li: number): string {
    return `${ci}-${li}`;
  }

  function waitForImageTask(tid: string): Promise<{ assetId: string | null }> {
    return new Promise((resolve, reject) => {
      let assetId: string | null = null;
      api.gen.streamMediaTask(tid, {
        onResult: (resultData: unknown) => {
          const data = resultData as { asset_ids?: string[] };
          const ids = data?.asset_ids || [];
          if (ids.length > 0) assetId = ids[0] ?? null;
        },
        onDone: (doneData: unknown) => {
          tasksStore.unregisterPageOwnedStream(tid);
          const data = doneData as { status: string };
          if (data.status === 'completed') resolve({ assetId });
          else reject(new Error('image generation failed'));
        },
        onError: () => {
          tasksStore.unregisterPageOwnedStream(tid);
          reject(new Error('connection lost'));
        },
      });
    });
  }

  async function submitImage(optsSubmit: {
    key: string;
    prompt: string;
    metadata: Record<string, unknown>;
    referenceAssetIds?: string[];
  }): Promise<string> {
    const model = String(opts.getModel() || '').trim();
    if (!model) {
      throw new Error($tt('video.selectModel'));
    }
    const locale = resolveLongVideoLocale(opts.getLocale());
    const modelParameters = (registryStore.registry?.models?.[model] as Record<string, unknown> | undefined)
      ?.parameters as Record<string, unknown> | undefined;
    const compose = opts.composeParams?.() ?? {};
    const { width, height } = PORTRAIT_REFERENCE_SIZE;
    portraitGeneratingKey.value = optsSubmit.key;
    if (opts.ensureSaved) await opts.ensureSaved();
    const genBody: Record<string, unknown> = {
      model,
      mode: optsSubmit.referenceAssetIds?.length ? 'i2i' : 't2i',
      prompt: optsSubmit.prompt,
      negative_prompt: locale === 'zh' ? PORTRAIT_NEGATIVE_PROMPT_ZH : PORTRAIT_NEGATIVE_PROMPT_EN,
      size: `${width}x${height}`,
      n: 1,
      steps: compose.steps,
      guidance: compose.guidance,
      metadata: optsSubmit.metadata,
      priority: 'normal',
    };
    if (optsSubmit.referenceAssetIds?.length) {
      genBody.reference_asset_ids = optsSubmit.referenceAssetIds;
    }
    appendImageInferenceFields(genBody, compose as Record<string, unknown>, modelParameters);
    const submitRes = await api.gen.createImageGeneration(genBody);
    const tid = taskIdFromSubmitResponse(submitRes);
    if (!tid) throw new Error('no task id');
    tasksStore.registerPageOwnedStream(tid);
    const { assetId } = await waitForImageTask(tid);
    if (!assetId) throw new Error('no asset');
    return assetId;
  }

  function applyCharacters(mutator: (chars: LongVideoCharacter[]) => LongVideoCharacter[]) {
    const base = normalizeCharacterRoster(opts.getCharacters());
    opts.setCharacters(mutator(base));
  }

  async function generateIdentityFront(ci: number) {
    const chars = normalizeCharacterRoster(opts.getCharacters());
    const ch = chars[ci];
    if (!ch) return;
    const locale = resolveLongVideoLocale(opts.getLocale());
    const prompt = buildIdentityFrontPrompt(ch, opts.getStyleAnchor(), locale, { useCache: false });
    const key = identityGenKey(ci, 'front');
    try {
      const assetId = await submitImage({
        key,
        prompt,
        metadata: {
          long_video_phase: 'cast_identity',
          cast_character_id: ch.id,
          cast_identity_angle: 'front',
          owner_id: opts.getOwnerId?.() ?? '',
        },
      });
      applyCharacters((list) =>
        list.map((c, i) => (i === ci ? patchIdentityView(c, 'front', { asset_id: assetId, prompt }) : c)),
      );
      await opts.afterSuccess?.();
      toast.success($tt('video.longVideoIdentityFrontDone'));
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      toast.error($tt('studio.genFailed', { msg }));
    } finally {
      if (portraitGeneratingKey.value === key) portraitGeneratingKey.value = null;
    }
  }

  async function generateIdentityAngle(ci: number, angle: CharacterIdentityAngle) {
    if (angle === 'front') {
      await generateIdentityFront(ci);
      return;
    }
    const chars = normalizeCharacterRoster(opts.getCharacters());
    const ch = chars[ci];
    if (!ch) return;
    const frontId = identityFrontAssetId(ch);
    if (!frontId) {
      toast.warning($tt('video.longVideoIdentityFrontRequired'));
      return;
    }
    const locale = resolveLongVideoLocale(opts.getLocale());
    const prompt = buildIdentityAnglePrompt(ch, angle, opts.getStyleAnchor(), locale);
    const key = identityGenKey(ci, angle);
    try {
      const assetId = await submitImage({
        key,
        prompt,
        metadata: {
          long_video_phase: 'cast_identity',
          cast_character_id: ch.id,
          cast_identity_angle: angle,
          owner_id: opts.getOwnerId?.() ?? '',
        },
        referenceAssetIds: [frontId],
      });
      applyCharacters((list) =>
        list.map((c, i) => (i === ci ? patchIdentityView(c, angle, { asset_id: assetId, prompt }) : c)),
      );
      await opts.afterSuccess?.();
      toast.success($tt('video.longVideoIdentityAngleDone'));
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      toast.error($tt('studio.genFailed', { msg }));
    } finally {
      if (portraitGeneratingKey.value === key) portraitGeneratingKey.value = null;
    }
  }

  function clearIdentityView(ci: number, angle: CharacterIdentityAngle) {
    applyCharacters((list) =>
      list.map((c, i) => (i === ci ? patchIdentityView(c, angle, { asset_id: null, prompt: null }) : c)),
    );
  }

  async function generateWardrobeRef(ci: number, li: number) {
    const chars = normalizeCharacterRoster(opts.getCharacters());
    const ch = chars[ci];
    const look = ch?.looks[li];
    if (!ch || !look) return;
    const identityRefs = [
      identityFrontAssetId(ch),
      ch.identity_sheet?.views.find((v) => v.angle === 'three_quarter')?.asset_id?.trim(),
    ].filter(Boolean) as string[];
    if (!identityRefs.length) {
      toast.warning($tt('video.longVideoIdentityFrontRequired'));
      return;
    }
    const locale = resolveLongVideoLocale(opts.getLocale());
    const prompt = buildWardrobeRefPrompt(ch, look, opts.getStyleAnchor(), locale);
    const key = portraitKey(ci, li);
    try {
      const assetId = await submitImage({
        key,
        prompt,
        metadata: {
          long_video_phase: 'cast_wardrobe',
          cast_character_id: ch.id,
          cast_look_id: look.id,
          owner_id: opts.getOwnerId?.() ?? '',
        },
        referenceAssetIds: identityRefs,
      });
      applyCharacters((list) =>
        list.map((c, i) => {
          if (i !== ci) return c;
          return {
            ...c,
            looks: c.looks.map((lk, j) =>
              j === li
                ? {
                    ...lk,
                    wardrobe_ref_asset_id: assetId,
                    reference_asset_id: assetId,
                    portrait_prompt: prompt,
                  }
                : lk,
            ),
          };
        }),
      );
      await opts.afterSuccess?.();
      toast.success($tt('video.longVideoWardrobeRefDone'));
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      toast.error($tt('studio.genFailed', { msg }));
    } finally {
      if (portraitGeneratingKey.value === key) portraitGeneratingKey.value = null;
    }
  }

  function clearWardrobeRef(ci: number, li: number) {
    applyCharacters((list) =>
      list.map((c, i) => {
        if (i !== ci) return c;
        return {
          ...c,
          looks: c.looks.map((lk, j) => {
            if (j !== li) return lk;
            const next = { ...lk };
            delete next.wardrobe_ref_asset_id;
            delete next.reference_asset_id;
            delete next.portrait_prompt;
            delete next.vision_description;
            return next;
          }),
        };
      }),
    );
  }

  return {
    portraitGeneratingKey,
    batchGenerating,
    generateIdentityFront,
    generateIdentityAngle,
    clearIdentityView,
    generateWardrobeRef,
    clearWardrobeRef,
  };
}
