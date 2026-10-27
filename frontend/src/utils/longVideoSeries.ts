/**
 * Series ↔ episode asset inheritance.
 * Effective roster = series base + episode variants (undefined fields inherit).
 * Never persist the materialized roster back as shared copies.
 */

import type {
  LongVideoCharacter,
  LongVideoCharacterLook,
  LongVideoCharacterLookVariant,
  LongVideoCharacterVariant,
  LongVideoProjectState,
  LongVideoScene,
  LongVideoSceneLook,
  LongVideoSceneLookVariant,
  LongVideoSceneVariant,
  LongVideoSeriesState,
} from '@/types';

export function newStableId(prefix: string): string {
  const rand =
    typeof crypto !== 'undefined' && 'randomUUID' in crypto
      ? crypto.randomUUID().replace(/-/g, '').slice(0, 12)
      : `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 8)}`;
  return `${prefix}${rand}`;
}

export function defaultSeriesState(
  partial: Partial<LongVideoSeriesState> = {},
): LongVideoSeriesState {
  return {
    version: 1,
    style_anchor: '',
    ...partial,
    characters: partial.characters ?? [],
    scenes: partial.scenes ?? [],
  };
}

function applyLookVariant(
  base: LongVideoCharacterLook,
  patch?: LongVideoCharacterLookVariant,
): LongVideoCharacterLook {
  if (!patch) {
    const wardrobe = base.wardrobe_ref_asset_id ?? base.reference_asset_id;
    return { ...base, wardrobe_ref_asset_id: wardrobe };
  }
  const wardrobePatch =
    patch.wardrobe_ref_asset_id !== undefined
      ? patch.wardrobe_ref_asset_id
      : patch.reference_asset_id !== undefined
        ? patch.reference_asset_id
        : undefined;
  const wardrobeBase = base.wardrobe_ref_asset_id ?? base.reference_asset_id;
  const wardrobe =
    wardrobePatch === null
      ? undefined
      : wardrobePatch !== undefined
        ? wardrobePatch
        : wardrobeBase;
  return {
    id: base.id,
    label: patch.label !== undefined ? patch.label : base.label,
    body: patch.body !== undefined ? patch.body : base.body,
    vision_description:
      patch.vision_description === null
        ? undefined
        : patch.vision_description !== undefined
          ? patch.vision_description
          : base.vision_description,
    wardrobe_ref_asset_id: wardrobe,
    reference_asset_id:
      patch.reference_asset_id === null
        ? undefined
        : patch.reference_asset_id !== undefined
          ? patch.reference_asset_id
          : base.reference_asset_id ?? wardrobe,
    lora_id:
      patch.lora_id === null
        ? undefined
        : patch.lora_id !== undefined
          ? patch.lora_id
          : base.lora_id,
    portrait_prompt:
      patch.portrait_prompt === null
        ? undefined
        : patch.portrait_prompt !== undefined
          ? patch.portrait_prompt
          : base.portrait_prompt,
  };
}

function applySceneLookVariant(
  base: LongVideoSceneLook,
  patch?: LongVideoSceneLookVariant,
): LongVideoSceneLook {
  if (!patch) return { ...base };
  return {
    id: base.id,
    label: patch.label !== undefined ? patch.label : base.label,
    body: patch.body !== undefined ? patch.body : base.body,
    vision_description:
      patch.vision_description === null
        ? undefined
        : patch.vision_description !== undefined
          ? patch.vision_description
          : base.vision_description,
    reference_asset_id:
      patch.reference_asset_id === null
        ? undefined
        : patch.reference_asset_id !== undefined
          ? patch.reference_asset_id
          : base.reference_asset_id,
    environment_prompt:
      patch.environment_prompt === null
        ? undefined
        : patch.environment_prompt !== undefined
          ? patch.environment_prompt
          : base.environment_prompt,
  };
}

function lookFromEpisodeOnlyPatch(
  patch: LongVideoCharacterLookVariant,
  fallbackBody = '',
): LongVideoCharacterLook {
  const wardrobe =
    patch.wardrobe_ref_asset_id === null
      ? undefined
      : patch.wardrobe_ref_asset_id !== undefined
        ? patch.wardrobe_ref_asset_id ?? undefined
        : patch.reference_asset_id === null
          ? undefined
          : (patch.reference_asset_id ?? undefined);
  return {
    id: patch.look_id,
    label: patch.label ?? '本集服装',
    body: patch.body ?? fallbackBody,
    vision_description:
      patch.vision_description === null ? undefined : (patch.vision_description ?? undefined),
    wardrobe_ref_asset_id: wardrobe,
    reference_asset_id:
      patch.reference_asset_id === null ? undefined : (patch.reference_asset_id ?? wardrobe),
    lora_id: patch.lora_id === null ? undefined : (patch.lora_id ?? undefined),
    portrait_prompt:
      patch.portrait_prompt === null ? undefined : (patch.portrait_prompt ?? undefined),
  };
}

export function materializeCharacters(
  seriesChars: LongVideoCharacter[],
  variants?: Record<string, LongVideoCharacterVariant>,
): LongVideoCharacter[] {
  const vars = variants ?? {};
  const out: LongVideoCharacter[] = seriesChars.map((ch) => {
    const v = vars[ch.id];
    const lookPatches = new Map((v?.looks ?? []).map((lk) => [lk.look_id, lk]));
    const looks = ch.looks.map((lk) => applyLookVariant(lk, lookPatches.get(lk.id)));
    for (const patch of v?.looks ?? []) {
      if (!patch.episode_only) continue;
      if (looks.some((lk) => lk.id === patch.look_id)) continue;
      const base =
        (patch.base_look_id && ch.looks.find((lk) => lk.id === patch.base_look_id)) ||
        ch.looks[0];
      looks.push(lookFromEpisodeOnlyPatch(patch, base?.body ?? ''));
    }
    const defaultLookId =
      (v?.default_look_id && looks.find((lk) => lk.id === v.default_look_id)?.id) ||
      looks.find((lk) => lk.id === ch.default_look_id)?.id ||
      looks[0]?.id ||
      ch.default_look_id;
    const pickStr = (
      patch: string | null | undefined,
      base: string | undefined,
    ): string | undefined => {
      if (patch === null) return undefined;
      if (patch !== undefined) return patch || undefined;
      return base;
    };
    return {
      id: ch.id,
      name: v?.name !== undefined ? v.name : ch.name,
      role:
        v?.role === null
          ? undefined
          : v?.role !== undefined
            ? v.role
            : ch.role,
      aliases:
        v?.aliases === null
          ? undefined
          : v?.aliases !== undefined
            ? v.aliases
            : ch.aliases,
      age_range: pickStr(v?.age_range, ch.age_range),
      gender_presentation: pickStr(v?.gender_presentation, ch.gender_presentation),
      identity_baseline: pickStr(v?.identity_baseline, ch.identity_baseline),
      visual_identity: pickStr(v?.visual_identity, ch.visual_identity),
      voice_manner: pickStr(v?.voice_manner, ch.voice_manner),
      performance_notes: pickStr(v?.performance_notes, ch.performance_notes),
      identity_notes: ch.identity_notes,
      identity_sheet:
        v?.identity_sheet === null
          ? undefined
          : v?.identity_sheet !== undefined
            ? v.identity_sheet
            : ch.identity_sheet,
      looks,
      default_look_id: defaultLookId,
      library_entity_id: ch.library_entity_id,
      library_version: ch.library_version,
      asset_status: ch.asset_status,
    };
  });
  const seriesIds = new Set(seriesChars.map((c) => c.id));
  for (const [id, v] of Object.entries(vars)) {
    if (!v.episode_only || seriesIds.has(id)) continue;
    const looks = (v.looks ?? []).map((patch) => lookFromEpisodeOnlyPatch(patch));
    out.push({
      id,
      name: v.name ?? id,
      role: v.role === null ? undefined : (v.role ?? undefined),
      aliases: v.aliases === null ? undefined : (v.aliases ?? undefined),
      age_range: v.age_range === null ? undefined : (v.age_range ?? undefined),
      gender_presentation:
        v.gender_presentation === null ? undefined : (v.gender_presentation ?? undefined),
      identity_baseline:
        v.identity_baseline === null ? undefined : (v.identity_baseline ?? undefined),
      visual_identity: v.visual_identity === null ? undefined : (v.visual_identity ?? undefined),
      voice_manner: v.voice_manner === null ? undefined : (v.voice_manner ?? undefined),
      performance_notes:
        v.performance_notes === null ? undefined : (v.performance_notes ?? undefined),
      identity_sheet: v.identity_sheet === null ? undefined : (v.identity_sheet ?? undefined),
      looks,
      default_look_id: v.default_look_id ?? looks[0]?.id ?? '',
    });
  }
  return out;
}

function sceneLookFromEpisodeOnlyPatch(
  patch: LongVideoSceneLookVariant,
  fallbackBody = '',
): LongVideoSceneLook {
  return {
    id: patch.look_id,
    label: patch.label ?? '本集变体',
    body: patch.body ?? fallbackBody,
    vision_description:
      patch.vision_description === null ? undefined : (patch.vision_description ?? undefined),
    reference_asset_id:
      patch.reference_asset_id === null ? undefined : (patch.reference_asset_id ?? undefined),
    environment_prompt:
      patch.environment_prompt === null ? undefined : (patch.environment_prompt ?? undefined),
  };
}

export function materializeScenes(
  seriesScenes: LongVideoScene[],
  variants?: Record<string, LongVideoSceneVariant>,
): LongVideoScene[] {
  const vars = variants ?? {};
  const out: LongVideoScene[] = seriesScenes.map((sc) => {
    const v = vars[sc.id];
    const lookPatches = new Map((v?.looks ?? []).map((lk) => [lk.look_id, lk]));
    const looks = sc.looks.map((lk) => applySceneLookVariant(lk, lookPatches.get(lk.id)));
    for (const patch of v?.looks ?? []) {
      if (!patch.episode_only) continue;
      if (looks.some((lk) => lk.id === patch.look_id)) continue;
      const base =
        (patch.base_look_id && sc.looks.find((lk) => lk.id === patch.base_look_id)) ||
        sc.looks[0];
      looks.push(sceneLookFromEpisodeOnlyPatch(patch, base?.body ?? ''));
    }
    const defaultLookId =
      (v?.default_look_id && looks.find((lk) => lk.id === v.default_look_id)?.id) ||
      looks.find((lk) => lk.id === sc.default_look_id)?.id ||
      looks[0]?.id ||
      sc.default_look_id;
    return {
      id: sc.id,
      name: v?.name !== undefined ? v.name : sc.name,
      looks,
      default_look_id: defaultLookId,
      spatial_layout_json:
        v?.spatial_layout_json === null
          ? undefined
          : v?.spatial_layout_json !== undefined
            ? v.spatial_layout_json
            : sc.spatial_layout_json,
      grounding_panorama_asset_id:
        v?.grounding_panorama_asset_id === null
          ? undefined
          : v?.grounding_panorama_asset_id !== undefined
            ? v.grounding_panorama_asset_id
            : sc.grounding_panorama_asset_id,
      grounding_depth_asset_id:
        v?.grounding_depth_asset_id === null
          ? undefined
          : v?.grounding_depth_asset_id !== undefined
            ? v.grounding_depth_asset_id
            : sc.grounding_depth_asset_id,
      library_entity_id: sc.library_entity_id,
      library_version: sc.library_version,
      asset_status: sc.asset_status,
    };
  });
  const seriesIds = new Set(seriesScenes.map((s) => s.id));
  for (const [id, v] of Object.entries(vars)) {
    if (!v.episode_only || seriesIds.has(id)) continue;
    const looks = (v.looks ?? []).map((patch) => sceneLookFromEpisodeOnlyPatch(patch));
    out.push({
      id,
      name: v.name ?? id,
      looks,
      default_look_id: v.default_look_id ?? looks[0]?.id ?? '',
      spatial_layout_json:
        v.spatial_layout_json === null ? undefined : (v.spatial_layout_json ?? undefined),
      grounding_panorama_asset_id:
        v.grounding_panorama_asset_id === null
          ? undefined
          : (v.grounding_panorama_asset_id ?? undefined),
      grounding_depth_asset_id:
        v.grounding_depth_asset_id === null
          ? undefined
          : (v.grounding_depth_asset_id ?? undefined),
    });
  }
  return out;
}

/** Snapshot a full roster character as an episode-only variant (not yet in series). */
export function episodeOnlyCharacterVariant(ch: LongVideoCharacter): LongVideoCharacterVariant {
  return {
    character_id: ch.id,
    episode_only: true,
    scope: 'episode',
    name: ch.name,
    role: ch.role ?? null,
    aliases: ch.aliases ?? null,
    age_range: ch.age_range ?? null,
    gender_presentation: ch.gender_presentation ?? null,
    identity_baseline: ch.identity_baseline ?? null,
    visual_identity: ch.visual_identity ?? null,
    voice_manner: ch.voice_manner ?? null,
    performance_notes: ch.performance_notes ?? null,
    default_look_id: ch.default_look_id,
    identity_sheet: ch.identity_sheet ?? null,
    looks: ch.looks.map((lk) => ({
      look_id: lk.id,
      episode_only: true,
      label: lk.label,
      body: lk.body,
      vision_description: lk.vision_description ?? null,
      wardrobe_ref_asset_id: lk.wardrobe_ref_asset_id ?? lk.reference_asset_id ?? null,
      reference_asset_id: lk.reference_asset_id ?? lk.wardrobe_ref_asset_id ?? null,
      lora_id: lk.lora_id ?? null,
      portrait_prompt: lk.portrait_prompt ?? null,
    })),
  };
}

export function episodeOnlySceneVariant(sc: LongVideoScene): LongVideoSceneVariant {
  return {
    scene_id: sc.id,
    episode_only: true,
    scope: 'episode',
    name: sc.name,
    default_look_id: sc.default_look_id,
    spatial_layout_json: sc.spatial_layout_json ?? null,
    grounding_panorama_asset_id: sc.grounding_panorama_asset_id ?? null,
    grounding_depth_asset_id: sc.grounding_depth_asset_id ?? null,
    looks: sc.looks.map((lk) => ({
      look_id: lk.id,
      episode_only: true,
      label: lk.label,
      body: lk.body,
      vision_description: lk.vision_description ?? null,
      reference_asset_id: lk.reference_asset_id ?? null,
      environment_prompt: lk.environment_prompt ?? null,
    })),
  };
}

/** Effective cast/scene for the active episode (or standalone project roster). */
export function effectiveEpisodeRoster(
  episode: Pick<LongVideoProjectState, 'characters' | 'scenes' | 'character_variants' | 'scene_variants' | 'series_id'>,
  series: LongVideoSeriesState | null | undefined,
): { characters: LongVideoCharacter[]; scenes: LongVideoScene[] } {
  if (episode.series_id && series) {
    return {
      characters: materializeCharacters(series.characters ?? [], episode.character_variants),
      scenes: materializeScenes(series.scenes ?? [], episode.scene_variants),
    };
  }
  return {
    characters: episode.characters ?? [],
    scenes: episode.scenes ?? [],
  };
}

export function deriveCharacterLookVariant(
  characterId: string,
  baseLook: LongVideoCharacterLook,
  overrides: Partial<LongVideoCharacterLook> = {},
): { lookId: string; variant: LongVideoCharacterLookVariant } {
  const lookId = newStableId('elk_');
  return {
    lookId,
    variant: {
      look_id: lookId,
      episode_only: true,
      base_look_id: baseLook.id,
      label: overrides.label ?? `${baseLook.label}·本集`,
      body: overrides.body ?? baseLook.body,
      vision_description: null,
      wardrobe_ref_asset_id:
        overrides.wardrobe_ref_asset_id
        ?? overrides.reference_asset_id
        ?? null,
      reference_asset_id: overrides.reference_asset_id ?? overrides.wardrobe_ref_asset_id ?? null,
      portrait_prompt: null,
    },
  };
}

export function upsertCharacterVariant(
  variants: Record<string, LongVideoCharacterVariant> | undefined,
  characterId: string,
  patch: Partial<LongVideoCharacterVariant> & { looks?: LongVideoCharacterLookVariant[] },
): Record<string, LongVideoCharacterVariant> {
  const next = { ...(variants ?? {}) };
  const prev = next[characterId] ?? { character_id: characterId };
  const lookMap = new Map((prev.looks ?? []).map((lk) => [lk.look_id, lk]));
  for (const lk of patch.looks ?? []) {
    lookMap.set(lk.look_id, { ...(lookMap.get(lk.look_id) ?? { look_id: lk.look_id }), ...lk });
  }
  next[characterId] = {
    ...prev,
    ...patch,
    character_id: characterId,
    looks: Array.from(lookMap.values()),
  };
  return next;
}

export function upsertSceneVariant(
  variants: Record<string, LongVideoSceneVariant> | undefined,
  sceneId: string,
  patch: Partial<LongVideoSceneVariant> & { looks?: LongVideoSceneLookVariant[] },
): Record<string, LongVideoSceneVariant> {
  const next = { ...(variants ?? {}) };
  const prev = next[sceneId] ?? { scene_id: sceneId };
  const lookMap = new Map((prev.looks ?? []).map((lk) => [lk.look_id, lk]));
  for (const lk of patch.looks ?? []) {
    lookMap.set(lk.look_id, { ...(lookMap.get(lk.look_id) ?? { look_id: lk.look_id }), ...lk });
  }
  next[sceneId] = {
    ...prev,
    ...patch,
    scene_id: sceneId,
    looks: Array.from(lookMap.values()),
  };
  return next;
}

/** Promote an episode-only look into the series character master. */
export function promoteCharacterLookToSeries(
  seriesChars: LongVideoCharacter[],
  characterId: string,
  look: LongVideoCharacterLook,
): LongVideoCharacter[] {
  return seriesChars.map((ch) => {
    if (ch.id !== characterId) return ch;
    if (ch.looks.some((lk) => lk.id === look.id)) {
      return {
        ...ch,
        looks: ch.looks.map((lk) => (lk.id === look.id ? { ...look } : lk)),
      };
    }
    return { ...ch, looks: [...ch.looks, { ...look }] };
  });
}

/**
 * Write identity (+ already-shared looks) into the project shared roster.
 * Episode-only looks stay on the episode until look-promote.
 */
export function promoteEpisodeCharacterToSeries(
  series: LongVideoSeriesState,
  effective: LongVideoCharacter,
  characterVariants: Record<string, LongVideoCharacterVariant> | undefined,
): {
  series: LongVideoSeriesState;
  characterVariants: Record<string, LongVideoCharacterVariant>;
} {
  const prev = characterVariants?.[effective.id];
  const epOnlyLookIds = new Set(
    (prev?.looks ?? []).filter((lk) => lk.episode_only).map((lk) => lk.look_id),
  );
  let seriesLooks = effective.looks.filter((lk) => !epOnlyLookIds.has(lk.id));
  let remainingEpLooks = (prev?.looks ?? []).filter((lk) => lk.episode_only);

  // Brand-new episode_only: seed series with the default look so the master is usable.
  if (seriesLooks.length === 0 && effective.looks.length > 0) {
    const seedId = effective.default_look_id || effective.looks[0].id;
    const seed = effective.looks.find((lk) => lk.id === seedId) || effective.looks[0];
    seriesLooks = [{ ...seed }];
    remainingEpLooks = remainingEpLooks.filter((lk) => lk.look_id !== seed.id);
  }

  const defaultLookId =
    seriesLooks.find((lk) => lk.id === effective.default_look_id)?.id
    ?? seriesLooks[0]?.id
    ?? effective.default_look_id;

  const chars = [...(series.characters ?? [])];
  const idx = chars.findIndex((c) => c.id === effective.id);
  const nextChar: LongVideoCharacter = {
    ...effective,
    default_look_id: defaultLookId,
    looks: seriesLooks.map((lk) => ({ ...lk })),
  };
  if (idx >= 0) chars[idx] = nextChar;
  else chars.push(nextChar);

  const nextVariants = { ...(characterVariants ?? {}) };
  if (remainingEpLooks.length) {
    nextVariants[effective.id] = {
      character_id: effective.id,
      scope: 'episode',
      looks: remainingEpLooks,
      ...(remainingEpLooks.some((lk) => lk.look_id === effective.default_look_id)
        ? { default_look_id: effective.default_look_id }
        : {}),
    };
  } else {
    delete nextVariants[effective.id];
  }

  return {
    series: { ...series, characters: chars },
    characterVariants: nextVariants,
  };
}

/** Same as {@link promoteEpisodeCharacterToSeries} for scene entities. */
export function promoteEpisodeSceneToSeries(
  series: LongVideoSeriesState,
  effective: LongVideoScene,
  sceneVariants: Record<string, LongVideoSceneVariant> | undefined,
): {
  series: LongVideoSeriesState;
  sceneVariants: Record<string, LongVideoSceneVariant>;
} {
  const prev = sceneVariants?.[effective.id];
  const epOnlyLookIds = new Set(
    (prev?.looks ?? []).filter((lk) => lk.episode_only).map((lk) => lk.look_id),
  );
  let seriesLooks = effective.looks.filter((lk) => !epOnlyLookIds.has(lk.id));
  let remainingEpLooks = (prev?.looks ?? []).filter((lk) => lk.episode_only);

  if (seriesLooks.length === 0 && effective.looks.length > 0) {
    const seedId = effective.default_look_id || effective.looks[0].id;
    const seed = effective.looks.find((lk) => lk.id === seedId) || effective.looks[0];
    seriesLooks = [{ ...seed }];
    remainingEpLooks = remainingEpLooks.filter((lk) => lk.look_id !== seed.id);
  }

  const defaultLookId =
    seriesLooks.find((lk) => lk.id === effective.default_look_id)?.id
    ?? seriesLooks[0]?.id
    ?? effective.default_look_id;

  const scenes = [...(series.scenes ?? [])];
  const idx = scenes.findIndex((s) => s.id === effective.id);
  const nextScene: LongVideoScene = {
    ...effective,
    default_look_id: defaultLookId,
    looks: seriesLooks.map((lk) => ({ ...lk })),
  };
  if (idx >= 0) scenes[idx] = nextScene;
  else scenes.push(nextScene);

  const nextVariants = { ...(sceneVariants ?? {}) };
  if (remainingEpLooks.length) {
    nextVariants[effective.id] = {
      scene_id: effective.id,
      scope: 'episode',
      looks: remainingEpLooks,
      ...(remainingEpLooks.some((lk) => lk.look_id === effective.default_look_id)
        ? { default_look_id: effective.default_look_id }
        : {}),
    };
  } else {
    delete nextVariants[effective.id];
  }

  return {
    series: { ...series, scenes },
    sceneVariants: nextVariants,
  };
}

/** True when the episode holds overrides (or episode-only entity) for this id. */
export function characterHasEpisodeOverride(
  variants: Record<string, LongVideoCharacterVariant> | undefined,
  characterId: string,
): boolean {
  const v = variants?.[characterId];
  if (!v) return false;
  if (v.episode_only) return true;
  return Boolean(
    v.name !== undefined
      || v.role !== undefined
      || v.aliases !== undefined
      || v.age_range !== undefined
      || v.gender_presentation !== undefined
      || v.identity_baseline !== undefined
      || v.visual_identity !== undefined
      || v.voice_manner !== undefined
      || v.performance_notes !== undefined
      || v.default_look_id !== undefined
      || (v.looks && v.looks.length > 0),
  );
}

export function sceneHasEpisodeOverride(
  variants: Record<string, LongVideoSceneVariant> | undefined,
  sceneId: string,
): boolean {
  const v = variants?.[sceneId];
  if (!v) return false;
  if (v.episode_only) return true;
  return Boolean(
    v.name !== undefined
      || v.default_look_id !== undefined
      || (v.looks && v.looks.length > 0)
      || v.spatial_layout_json !== undefined
      || v.grounding_panorama_asset_id !== undefined
      || v.grounding_depth_asset_id !== undefined,
  );
}

function lookFieldPatch(
  base: LongVideoCharacterLook,
  effective: LongVideoCharacterLook,
): LongVideoCharacterLookVariant | null {
  const patch: LongVideoCharacterLookVariant = { look_id: base.id };
  let changed = false;
  if (effective.label !== base.label) {
    patch.label = effective.label;
    changed = true;
  }
  if (effective.body !== base.body) {
    patch.body = effective.body;
    changed = true;
  }
  if ((effective.vision_description ?? '') !== (base.vision_description ?? '')) {
    patch.vision_description = effective.vision_description ?? null;
    changed = true;
  }
  const effWardrobe = effective.wardrobe_ref_asset_id ?? effective.reference_asset_id ?? '';
  const baseWardrobe = base.wardrobe_ref_asset_id ?? base.reference_asset_id ?? '';
  if (effWardrobe !== baseWardrobe) {
    patch.wardrobe_ref_asset_id = effWardrobe || null;
    patch.reference_asset_id = (effective.reference_asset_id ?? effWardrobe) || null;
    changed = true;
  } else if ((effective.reference_asset_id ?? '') !== (base.reference_asset_id ?? '')) {
    patch.reference_asset_id = effective.reference_asset_id ?? null;
    changed = true;
  }
  if ((effective.lora_id ?? '') !== (base.lora_id ?? '')) {
    patch.lora_id = effective.lora_id ?? null;
    changed = true;
  }
  if ((effective.portrait_prompt ?? '') !== (base.portrait_prompt ?? '')) {
    patch.portrait_prompt = effective.portrait_prompt ?? null;
    changed = true;
  }
  return changed ? patch : null;
}

function sceneLookFieldPatch(
  base: LongVideoSceneLook,
  effective: LongVideoSceneLook,
): LongVideoSceneLookVariant | null {
  const patch: LongVideoSceneLookVariant = { look_id: base.id };
  let changed = false;
  if (effective.label !== base.label) {
    patch.label = effective.label;
    changed = true;
  }
  if (effective.body !== base.body) {
    patch.body = effective.body;
    changed = true;
  }
  if ((effective.vision_description ?? '') !== (base.vision_description ?? '')) {
    patch.vision_description = effective.vision_description ?? null;
    changed = true;
  }
  if ((effective.reference_asset_id ?? '') !== (base.reference_asset_id ?? '')) {
    patch.reference_asset_id = effective.reference_asset_id ?? null;
    changed = true;
  }
  if ((effective.environment_prompt ?? '') !== (base.environment_prompt ?? '')) {
    patch.environment_prompt = effective.environment_prompt ?? null;
    changed = true;
  }
  return changed ? patch : null;
}

/** Build episode variant patch from an edited effective character vs series master. */
export function diffCharacterAgainstSeries(
  seriesChar: LongVideoCharacter,
  effectiveChar: LongVideoCharacter,
  opts?: { /** Episode scope: identity is RO — only look selection / patches. */
    looksAndDefaultOnly?: boolean },
): LongVideoCharacterVariant | null {
  const variant: LongVideoCharacterVariant = {
    character_id: seriesChar.id,
    scope: 'episode',
    base_version: seriesChar.library_version,
  };
  const lookPatches: LongVideoCharacterLookVariant[] = [];
  let hasCharPatch = false;
  const looksOnly = Boolean(opts?.looksAndDefaultOnly);

  if (!looksOnly) {
    if (effectiveChar.name !== seriesChar.name) {
      variant.name = effectiveChar.name;
      hasCharPatch = true;
    }
    if ((effectiveChar.role ?? '') !== (seriesChar.role ?? '')) {
      variant.role = effectiveChar.role ?? null;
      hasCharPatch = true;
    }
    const aliasesEqual =
      JSON.stringify(effectiveChar.aliases ?? []) === JSON.stringify(seriesChar.aliases ?? []);
    if (!aliasesEqual) {
      variant.aliases = effectiveChar.aliases ?? null;
      hasCharPatch = true;
    }
    const patchStr = (
      key: 'age_range' | 'gender_presentation' | 'identity_baseline' | 'visual_identity' | 'voice_manner' | 'performance_notes',
    ) => {
      if ((effectiveChar[key] ?? '') !== (seriesChar[key] ?? '')) {
        variant[key] = effectiveChar[key] ?? null;
        hasCharPatch = true;
      }
    };
    patchStr('age_range');
    patchStr('gender_presentation');
    patchStr('identity_baseline');
    patchStr('visual_identity');
    patchStr('voice_manner');
    patchStr('performance_notes');
  }
  if (effectiveChar.default_look_id !== seriesChar.default_look_id) {
    variant.default_look_id = effectiveChar.default_look_id;
    hasCharPatch = true;
  }

  const seriesLooksById = new Map(seriesChar.looks.map((lk) => [lk.id, lk]));
  for (const lk of effectiveChar.looks) {
    const base = seriesLooksById.get(lk.id);
    if (!base) {
      lookPatches.push({
        look_id: lk.id,
        episode_only: true,
        base_look_id: seriesChar.default_look_id,
        label: lk.label,
        body: lk.body,
        vision_description: lk.vision_description ?? null,
        wardrobe_ref_asset_id: lk.wardrobe_ref_asset_id ?? lk.reference_asset_id ?? null,
        reference_asset_id: lk.reference_asset_id ?? lk.wardrobe_ref_asset_id ?? null,
        lora_id: lk.lora_id ?? null,
        portrait_prompt: lk.portrait_prompt ?? null,
      });
      continue;
    }
    const patch = lookFieldPatch(base, lk);
    if (patch) lookPatches.push(patch);
  }

  if (!hasCharPatch && !lookPatches.length) return null;
  if (lookPatches.length) variant.looks = lookPatches;
  return variant;
}

export function diffSceneAgainstSeries(
  seriesScene: LongVideoScene,
  effectiveScene: LongVideoScene,
  opts?: { looksAndDefaultOnly?: boolean },
): LongVideoSceneVariant | null {
  const variant: LongVideoSceneVariant = {
    scene_id: seriesScene.id,
    scope: 'episode',
    base_version: seriesScene.library_version,
  };
  const lookPatches: LongVideoSceneLookVariant[] = [];
  let hasScenePatch = false;
  const looksOnly = Boolean(opts?.looksAndDefaultOnly);

  if (!looksOnly && effectiveScene.name !== seriesScene.name) {
    variant.name = effectiveScene.name;
    hasScenePatch = true;
  }
  if (effectiveScene.default_look_id !== seriesScene.default_look_id) {
    variant.default_look_id = effectiveScene.default_look_id;
    hasScenePatch = true;
  }
  if (
    !looksOnly
    && JSON.stringify(effectiveScene.spatial_layout_json ?? null)
      !== JSON.stringify(seriesScene.spatial_layout_json ?? null)
  ) {
    variant.spatial_layout_json = effectiveScene.spatial_layout_json ?? null;
    hasScenePatch = true;
  }
  if (
    !looksOnly
    && (effectiveScene.grounding_panorama_asset_id ?? '')
      !== (seriesScene.grounding_panorama_asset_id ?? '')
  ) {
    variant.grounding_panorama_asset_id = effectiveScene.grounding_panorama_asset_id ?? null;
    hasScenePatch = true;
  }
  if (
    !looksOnly
    && (effectiveScene.grounding_depth_asset_id ?? '')
      !== (seriesScene.grounding_depth_asset_id ?? '')
  ) {
    variant.grounding_depth_asset_id = effectiveScene.grounding_depth_asset_id ?? null;
    hasScenePatch = true;
  }

  const seriesLooksById = new Map(seriesScene.looks.map((lk) => [lk.id, lk]));
  for (const lk of effectiveScene.looks) {
    const base = seriesLooksById.get(lk.id);
    if (!base) {
      lookPatches.push({
        look_id: lk.id,
        episode_only: true,
        base_look_id: seriesScene.default_look_id,
        label: lk.label,
        body: lk.body,
        vision_description: lk.vision_description ?? null,
        reference_asset_id: lk.reference_asset_id ?? null,
        environment_prompt: lk.environment_prompt ?? null,
      });
      continue;
    }
    const patch = sceneLookFieldPatch(base, lk);
    if (patch) lookPatches.push(patch);
  }

  if (!hasScenePatch && !lookPatches.length) return null;
  if (lookPatches.length) variant.looks = lookPatches;
  return variant;
}

export function clearCharacterVariant(
  variants: Record<string, LongVideoCharacterVariant> | undefined,
  characterId: string,
): Record<string, LongVideoCharacterVariant> {
  const next = { ...(variants ?? {}) };
  delete next[characterId];
  return next;
}

export function clearSceneVariant(
  variants: Record<string, LongVideoSceneVariant> | undefined,
  sceneId: string,
): Record<string, LongVideoSceneVariant> {
  const next = { ...(variants ?? {}) };
  delete next[sceneId];
  return next;
}

/** Drop episode override and inherit series master again. */
export function restoreCharacterFromSeries(
  variants: Record<string, LongVideoCharacterVariant> | undefined,
  characterId: string,
): Record<string, LongVideoCharacterVariant> {
  return clearCharacterVariant(variants, characterId);
}

/** Provenance map: entity id → library_version (or series revision). */
export function workspaceRosterRevisions(
  episode: Pick<LongVideoProjectState, 'character_variants' | 'scene_variants' | 'series_id' | 'revision'>,
  series: LongVideoSeriesState | null | undefined,
): Record<string, number> {
  const out: Record<string, number> = {};
  const baseRev = series?.revision ?? episode.revision ?? 0;
  for (const ch of series?.characters ?? []) {
    out[ch.id] = ch.library_version ?? baseRev;
  }
  for (const sc of series?.scenes ?? []) {
    out[sc.id] = sc.library_version ?? baseRev;
  }
  for (const [id, v] of Object.entries(episode.character_variants ?? {})) {
    if (typeof v.base_version === 'number') out[id] = v.base_version;
  }
  for (const [id, v] of Object.entries(episode.scene_variants ?? {})) {
    if (typeof v.base_version === 'number') out[id] = v.base_version;
  }
  return out;
}

export function seriesStateFromProject(
  lv: LongVideoProjectState,
): LongVideoSeriesState {
  return defaultSeriesState({
    characters: lv.characters ?? [],
    scenes: lv.scenes ?? [],
    style_anchor: lv.style_anchor,
    keyframe_model: lv.keyframe_model,
    segment_video_model: lv.segment_video_model,
    segment_duration_sec: lv.segment_duration_sec,
    overlap_frames: lv.overlap_frames,
    chain_mode: lv.chain_mode,
    output_size: lv.output_size,
    target_duration_sec: lv.target_duration_sec,
    portrait_model: lv.portrait_model,
    script_parse_llm_model: lv.script_parse_llm_model,
  });
}

/** Drop episode override and inherit series master again. */
export const restoreEpisodeCharacterFromSeries = restoreCharacterFromSeries;

export function restoreEpisodeSceneFromSeries(
  variants: Record<string, LongVideoSceneVariant> | undefined,
  sceneId: string,
): Record<string, LongVideoSceneVariant> {
  return clearSceneVariant(variants, sceneId);
}

function stripEpisodeOnlyLookVariants<T extends { look_id: string; episode_only?: boolean }>(
  looks: T[] | undefined,
  lookId: string,
): T[] | undefined {
  if (!looks?.length) return looks;
  const next = looks.filter((lk) => !(lk.episode_only && lk.look_id === lookId));
  return next.length ? next : undefined;
}

function pruneCharacterVariantAfterPromote(
  variants: Record<string, LongVideoCharacterVariant> | undefined,
  characterId: string,
  lookId: string,
): Record<string, LongVideoCharacterVariant> {
  const next = { ...(variants ?? {}) };
  const prev = next[characterId];
  if (!prev) return next;
  const looks = stripEpisodeOnlyLookVariants(prev.looks, lookId);
  const rest = { ...prev, looks };
  const empty =
    !rest.name
    && !rest.default_look_id
    && !looks?.length;
  if (empty) {
    delete next[characterId];
  } else {
    next[characterId] = rest;
  }
  return next;
}

function pruneSceneVariantAfterPromote(
  variants: Record<string, LongVideoSceneVariant> | undefined,
  sceneId: string,
  lookId: string,
): Record<string, LongVideoSceneVariant> {
  const next = { ...(variants ?? {}) };
  const prev = next[sceneId];
  if (!prev) return next;
  const looks = stripEpisodeOnlyLookVariants(prev.looks, lookId);
  const rest = { ...prev, looks };
  const empty =
    !rest.name
    && !rest.default_look_id
    && !looks?.length
    && rest.spatial_layout_json === undefined
    && rest.grounding_panorama_asset_id === undefined
    && rest.grounding_depth_asset_id === undefined;
  if (empty) {
    delete next[sceneId];
  } else {
    next[sceneId] = rest;
  }
  return next;
}

/** Promote episode-only look into series roster and bump library when linked. */
export async function promoteEpisodeLookToSeriesAndLibrary(
  series: LongVideoSeriesState,
  characterVariants: Record<string, LongVideoCharacterVariant> | undefined,
  characterId: string,
  lookId: string,
  library: import('./longVideoLibrary').LibraryApi,
): Promise<{
  series: LongVideoSeriesState;
  characterVariants: Record<string, LongVideoCharacterVariant>;
}> {
  const { publishCharacterToLibrary } = await import('./longVideoLibrary');
  const roster = materializeCharacters(series.characters ?? [], characterVariants);
  const eff = roster.find((c) => c.id === characterId);
  const look = eff?.looks.find((lk) => lk.id === lookId);
  if (!eff || !look) {
    throw new Error('episode look not found');
  }
  let chars = promoteCharacterLookToSeries(series.characters ?? [], characterId, look);
  let ch = chars.find((c) => c.id === characterId);
  if (!ch) throw new Error('character not found');
  if (ch.library_entity_id) {
    ch = await publishCharacterToLibrary(ch, library);
    chars = chars.map((c) => (c.id === characterId ? ch! : c));
  }
  return {
    series: { ...series, characters: chars },
    characterVariants: pruneCharacterVariantAfterPromote(characterVariants, characterId, lookId),
  };
}

export async function promoteEpisodeSceneLookToSeriesAndLibrary(
  series: LongVideoSeriesState,
  sceneVariants: Record<string, LongVideoSceneVariant> | undefined,
  sceneId: string,
  lookId: string,
  library: import('./longVideoLibrary').LibraryApi,
): Promise<{
  series: LongVideoSeriesState;
  sceneVariants: Record<string, LongVideoSceneVariant>;
}> {
  const { publishSceneToLibrary } = await import('./longVideoLibrary');
  const roster = materializeScenes(series.scenes ?? [], sceneVariants);
  const eff = roster.find((s) => s.id === sceneId);
  const look = eff?.looks.find((lk) => lk.id === lookId);
  if (!eff || !look) {
    throw new Error('episode scene look not found');
  }
  let scenes = series.scenes ?? [];
  scenes = scenes.map((sc) => {
    if (sc.id !== sceneId) return sc;
    if (sc.looks.some((lk) => lk.id === look.id)) {
      return {
        ...sc,
        looks: sc.looks.map((lk) => (lk.id === look.id ? { ...look } : lk)),
      };
    }
    return { ...sc, looks: [...sc.looks, { ...look }] };
  });
  let sc = scenes.find((s) => s.id === sceneId);
  if (!sc) throw new Error('scene not found');
  if (sc.library_entity_id) {
    sc = await publishSceneToLibrary(sc, library);
    scenes = scenes.map((s) => (s.id === sceneId ? sc! : s));
  }
  return {
    series: { ...series, scenes },
    sceneVariants: pruneSceneVariantAfterPromote(sceneVariants, sceneId, lookId),
  };
}
