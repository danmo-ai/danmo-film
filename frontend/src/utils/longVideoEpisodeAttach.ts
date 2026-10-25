/**
 * Episode parse attach: link parse characters/scenes to project masters,
 * rules-based look selection, no auto-promote to series.
 */

import type {
  LongVideoCharacter,
  LongVideoCharacterLook,
  LongVideoCharacterLookVariant,
  LongVideoCharacterVariant,
  LongVideoScene,
  LongVideoSceneLook,
  LongVideoSceneLookVariant,
  LongVideoSceneVariant,
  LongVideoShotCastLook,
  LongVideoShotState,
} from '@/types';
import { newStableId } from './longVideoSeries';

export type CharacterLinkMap = Record<string, string>;
export type SceneLinkMap = Record<string, string>;

export type CharacterLinkResult = {
  characterId: string;
  /** Parse key used for the link map (name or incoming id). */
  linkKey: string;
  matched: boolean;
  ambiguous: boolean;
  seriesCharacter?: LongVideoCharacter;
};

export type SceneLinkResult = {
  sceneId: string;
  linkKey: string;
  matched: boolean;
  ambiguous: boolean;
  seriesScene?: LongVideoScene;
};

export type LookSelectResult = {
  lookId: string;
  /** Episode-only look stub to merge into variants (unmatched wardrobe cue). */
  episodeLookStub?: LongVideoCharacterLook;
  createdStub: boolean;
};

export type SceneLookSelectResult = {
  lookId: string;
  episodeLookStub?: LongVideoSceneLook;
  createdStub: boolean;
};

export type AttachParseRosterResult = {
  characterVariants: Record<string, LongVideoCharacterVariant>;
  sceneVariants: Record<string, LongVideoSceneVariant>;
  characterLinks: CharacterLinkMap;
  sceneLinks: SceneLinkMap;
  /** Effective default look id per character after rules select. */
  characterDefaultLooks: Record<string, string>;
  sceneDefaultLooks: Record<string, string>;
  reusedNames: string[];
  newEpisodeNames: string[];
};

function normKey(s: string): string {
  return s.trim().toLowerCase();
}

function minSoftLen(s: string): boolean {
  const t = s.replace(/\s/g, '');
  return t.length >= 2;
}

/** All names a character can be referred to by (display + aliases). */
export function characterNameKeys(ch: LongVideoCharacter): string[] {
  const keys = [ch.name, ...(ch.aliases ?? [])]
    .map((n) => normKey(n || ''))
    .filter(Boolean);
  return [...new Set(keys)];
}

function softContainMatch(a: string, b: string): boolean {
  if (!minSoftLen(a) || !minSoftLen(b)) return false;
  const na = normKey(a);
  const nb = normKey(b);
  if (na === nb) return true;
  const short = na.length <= nb.length ? na : nb;
  const long = na.length <= nb.length ? nb : na;
  return short.length >= 2 && long.includes(short);
}

/**
 * Resolve parse character to project master or mark as new.
 * Match order: id → exact name → alias → soft containment (ambiguous if >1).
 */
export function resolveCharacterLink(
  incoming: Pick<LongVideoCharacter, 'id' | 'name' | 'aliases'>,
  seriesChars: LongVideoCharacter[],
  priorLinks?: CharacterLinkMap,
): CharacterLinkResult {
  const nameKey = normKey(incoming.name || '');
  const linkKey = nameKey || incoming.id || newStableId('chk_');

  if (priorLinks?.[linkKey]) {
    const id = priorLinks[linkKey];
    const seriesCharacter = seriesChars.find((c) => c.id === id);
    if (seriesCharacter) {
      return { characterId: id, linkKey, matched: true, ambiguous: false, seriesCharacter };
    }
  }
  if (priorLinks?.[incoming.id]) {
    const id = priorLinks[incoming.id];
    const seriesCharacter = seriesChars.find((c) => c.id === id);
    if (seriesCharacter) {
      return {
        characterId: id,
        linkKey: incoming.id,
        matched: true,
        ambiguous: false,
        seriesCharacter,
      };
    }
  }

  const byId = seriesChars.find((c) => c.id === incoming.id);
  if (byId) {
    return {
      characterId: byId.id,
      linkKey,
      matched: true,
      ambiguous: false,
      seriesCharacter: byId,
    };
  }

  const incomingKeys = [
    incoming.name,
    ...(incoming.aliases ?? []),
  ]
    .map((n) => normKey(n || ''))
    .filter(Boolean);

  const exactHits = seriesChars.filter((ch) => {
    const keys = characterNameKeys(ch);
    return incomingKeys.some((k) => keys.includes(k));
  });
  if (exactHits.length === 1) {
    return {
      characterId: exactHits[0].id,
      linkKey,
      matched: true,
      ambiguous: false,
      seriesCharacter: exactHits[0],
    };
  }
  if (exactHits.length > 1) {
    return {
      characterId: incoming.id || newStableId('cep_'),
      linkKey,
      matched: false,
      ambiguous: true,
    };
  }

  const softHits = seriesChars.filter((ch) => {
    const keys = characterNameKeys(ch);
    return incomingKeys.some((ik) => keys.some((pk) => softContainMatch(ik, pk)));
  });
  if (softHits.length === 1) {
    return {
      characterId: softHits[0].id,
      linkKey,
      matched: true,
      ambiguous: false,
      seriesCharacter: softHits[0],
    };
  }
  if (softHits.length > 1) {
    return {
      characterId: incoming.id || newStableId('cep_'),
      linkKey,
      matched: false,
      ambiguous: true,
    };
  }

  return {
    characterId: incoming.id || newStableId('cep_'),
    linkKey,
    matched: false,
    ambiguous: false,
  };
}

export function resolveSceneLink(
  incoming: Pick<LongVideoScene, 'id' | 'name'>,
  seriesScenes: LongVideoScene[],
  priorLinks?: SceneLinkMap,
): SceneLinkResult {
  const nameKey = normKey(incoming.name || '');
  const linkKey = nameKey || incoming.id || newStableId('ssk_');

  if (priorLinks?.[linkKey]) {
    const id = priorLinks[linkKey];
    const seriesScene = seriesScenes.find((s) => s.id === id);
    if (seriesScene) {
      return { sceneId: id, linkKey, matched: true, ambiguous: false, seriesScene };
    }
  }

  const byId = seriesScenes.find((s) => s.id === incoming.id);
  if (byId) {
    return {
      sceneId: byId.id,
      linkKey,
      matched: true,
      ambiguous: false,
      seriesScene: byId,
    };
  }

  const exact = seriesScenes.filter((s) => normKey(s.name) === nameKey && nameKey);
  if (exact.length === 1) {
    return {
      sceneId: exact[0].id,
      linkKey,
      matched: true,
      ambiguous: false,
      seriesScene: exact[0],
    };
  }
  if (exact.length > 1) {
    return {
      sceneId: incoming.id || newStableId('sep_'),
      linkKey,
      matched: false,
      ambiguous: true,
    };
  }

  const soft = seriesScenes.filter(
    (s) => nameKey && softContainMatch(incoming.name || '', s.name),
  );
  if (soft.length === 1) {
    return {
      sceneId: soft[0].id,
      linkKey,
      matched: true,
      ambiguous: false,
      seriesScene: soft[0],
    };
  }
  if (soft.length > 1) {
    return {
      sceneId: incoming.id || newStableId('sep_'),
      linkKey,
      matched: false,
      ambiguous: true,
    };
  }

  return {
    sceneId: incoming.id || newStableId('sep_'),
    linkKey,
    matched: false,
    ambiguous: false,
  };
}

function tokenizeCue(text: string): string[] {
  const raw = text.trim().toLowerCase();
  if (!raw) return [];
  const parts = raw.split(/[\s,，、|;；/\\|｜\-—–·.。:：]+/).filter(Boolean);
  const cjk = raw.replace(/[^\u4e00-\u9fff]/g, '');
  const grams: string[] = [];
  for (let i = 0; i < cjk.length - 1; i += 1) {
    grams.push(cjk.slice(i, i + 2));
  }
  return [...new Set([...parts, ...grams])];
}

/** Score wardrobe/environment cue against look label+body. Higher is better. */
export function scoreLookAgainstCue(look: { label: string; body: string }, cue: string): number {
  const cueTokens = tokenizeCue(cue);
  if (!cueTokens.length) return 0;
  const hay = `${look.label} ${look.body}`.toLowerCase();
  if (!hay.trim()) return 0;
  let hits = 0;
  for (const t of cueTokens) {
    if (t.length >= 2 && hay.includes(t)) hits += 1;
  }
  return hits / cueTokens.length;
}

const LOOK_MATCH_THRESHOLD = 0.28;

export function selectCharacterLookByRules(
  seriesChar: LongVideoCharacter,
  cue: string,
): LookSelectResult {
  const defaultId = seriesChar.default_look_id || seriesChar.looks[0]?.id || '';
  const trimmed = cue.trim();
  if (!trimmed) {
    return { lookId: defaultId, createdStub: false };
  }
  let best: { id: string; score: number } | null = null;
  for (const lk of seriesChar.looks) {
    const score = scoreLookAgainstCue(lk, trimmed);
    if (!best || score > best.score) best = { id: lk.id, score };
  }
  if (best && best.score >= LOOK_MATCH_THRESHOLD) {
    return { lookId: best.id, createdStub: false };
  }
  const stubId = newStableId('elk_');
  const label = trimmed.slice(0, 24) || '本集装扮';
  return {
    lookId: stubId,
    createdStub: true,
    episodeLookStub: {
      id: stubId,
      label,
      body: trimmed,
    },
  };
}

export function selectSceneLookByRules(
  seriesScene: LongVideoScene,
  cue: string,
): SceneLookSelectResult {
  const defaultId = seriesScene.default_look_id || seriesScene.looks[0]?.id || '';
  const trimmed = cue.trim();
  if (!trimmed) {
    return { lookId: defaultId, createdStub: false };
  }
  let best: { id: string; score: number } | null = null;
  for (const lk of seriesScene.looks) {
    const score = scoreLookAgainstCue(lk, trimmed);
    if (!best || score > best.score) best = { id: lk.id, score };
  }
  if (best && best.score >= LOOK_MATCH_THRESHOLD) {
    return { lookId: best.id, createdStub: false };
  }
  const stubId = newStableId('esk_');
  const label = trimmed.slice(0, 24) || '本集变体';
  return {
    lookId: stubId,
    createdStub: true,
    episodeLookStub: {
      id: stubId,
      label,
      body: trimmed,
    },
  };
}

function wardrobeCueFromIncoming(ch: LongVideoCharacter): string {
  const parts = ch.looks.map((lk) => [lk.label, lk.body].filter(Boolean).join(' ')).filter(Boolean);
  return parts.join('；');
}

function sceneCueFromIncoming(sc: LongVideoScene): string {
  const parts = sc.looks.map((lk) => [lk.label, lk.body].filter(Boolean).join(' ')).filter(Boolean);
  return parts.join('；');
}

function lookToEpisodeVariant(
  look: LongVideoCharacterLook,
  baseLookId?: string,
): LongVideoCharacterLookVariant {
  return {
    look_id: look.id,
    episode_only: true,
    base_look_id: baseLookId,
    label: look.label,
    body: look.body,
    vision_description: look.vision_description ?? null,
    reference_asset_id: look.reference_asset_id ?? null,
    lora_id: look.lora_id ?? null,
    portrait_prompt: look.portrait_prompt ?? null,
  };
}

function sceneLookToEpisodeVariant(
  look: LongVideoSceneLook,
  baseLookId?: string,
): LongVideoSceneLookVariant {
  return {
    look_id: look.id,
    episode_only: true,
    base_look_id: baseLookId,
    label: look.label,
    body: look.body,
    vision_description: look.vision_description ?? null,
    reference_asset_id: look.reference_asset_id ?? null,
    environment_prompt: look.environment_prompt ?? null,
  };
}

/**
 * Attach parse roster to episode: link to series masters or create episode_only.
 * Does not mutate series state.
 */
export function attachParseRosterToEpisode(opts: {
  incomingCharacters: LongVideoCharacter[];
  incomingScenes: LongVideoScene[];
  seriesCharacters: LongVideoCharacter[];
  seriesScenes: LongVideoScene[];
  priorCharacterLinks?: CharacterLinkMap;
  priorSceneLinks?: SceneLinkMap;
  priorCharacterVariants?: Record<string, LongVideoCharacterVariant>;
  priorSceneVariants?: Record<string, LongVideoSceneVariant>;
}): AttachParseRosterResult {
  const characterLinks: CharacterLinkMap = { ...(opts.priorCharacterLinks ?? {}) };
  const sceneLinks: SceneLinkMap = { ...(opts.priorSceneLinks ?? {}) };
  const characterVariants: Record<string, LongVideoCharacterVariant> = {
    ...(opts.priorCharacterVariants ?? {}),
  };
  const sceneVariants: Record<string, LongVideoSceneVariant> = {
    ...(opts.priorSceneVariants ?? {}),
  };
  const characterDefaultLooks: Record<string, string> = {};
  const sceneDefaultLooks: Record<string, string> = {};
  const reusedNames: string[] = [];
  const newEpisodeNames: string[] = [];

  for (const inc of opts.incomingCharacters) {
    const link = resolveCharacterLink(inc, opts.seriesCharacters, characterLinks);
    characterLinks[link.linkKey] = link.characterId;
    if (inc.id) characterLinks[inc.id] = link.characterId;
    if (normKey(inc.name)) characterLinks[normKey(inc.name)] = link.characterId;

    if (link.matched && link.seriesCharacter) {
      reusedNames.push(link.seriesCharacter.name);
      const cue = wardrobeCueFromIncoming(inc);
      const sel = selectCharacterLookByRules(link.seriesCharacter, cue);
      characterDefaultLooks[link.characterId] = sel.lookId || link.seriesCharacter.default_look_id;

      if (sel.createdStub && sel.episodeLookStub) {
        const prev = characterVariants[link.characterId] ?? {
          character_id: link.characterId,
          scope: 'episode' as const,
        };
        const looks = [...(prev.looks ?? [])];
        looks.push(lookToEpisodeVariant(sel.episodeLookStub, link.seriesCharacter.default_look_id));
        characterVariants[link.characterId] = {
          ...prev,
          character_id: link.characterId,
          default_look_id: sel.lookId,
          looks,
        };
      } else if (sel.lookId && sel.lookId !== link.seriesCharacter.default_look_id) {
        const prev = characterVariants[link.characterId] ?? {
          character_id: link.characterId,
          scope: 'episode' as const,
        };
        characterVariants[link.characterId] = {
          ...prev,
          character_id: link.characterId,
          default_look_id: sel.lookId,
        };
      }
      continue;
    }

    // New / ambiguous → episode_only
    newEpisodeNames.push(inc.name || link.characterId);
    const looks =
      inc.looks.length > 0
        ? inc.looks.map((lk) => ({ ...lk, id: lk.id || newStableId('elk_') }))
        : [
            {
              id: newStableId('elk_'),
              label: '默认',
              body: '',
            },
          ];
    const defaultLookId = looks.find((l) => l.id === inc.default_look_id)?.id ?? looks[0].id;
    characterDefaultLooks[link.characterId] = defaultLookId;
    characterVariants[link.characterId] = {
      character_id: link.characterId,
      episode_only: true,
      scope: 'episode',
      name: inc.name,
      role: inc.role ?? null,
      aliases: inc.aliases ?? null,
      age_range: inc.age_range ?? null,
      gender_presentation: inc.gender_presentation ?? null,
      identity_baseline: inc.identity_baseline ?? null,
      visual_identity: inc.visual_identity ?? null,
      voice_manner: inc.voice_manner ?? null,
      performance_notes: inc.performance_notes ?? null,
      default_look_id: defaultLookId,
      looks: looks.map((lk) => lookToEpisodeVariant(lk)),
    };
  }

  for (const inc of opts.incomingScenes) {
    const link = resolveSceneLink(inc, opts.seriesScenes, sceneLinks);
    sceneLinks[link.linkKey] = link.sceneId;
    if (inc.id) sceneLinks[inc.id] = link.sceneId;
    if (normKey(inc.name)) sceneLinks[normKey(inc.name)] = link.sceneId;

    if (link.matched && link.seriesScene) {
      const cue = sceneCueFromIncoming(inc);
      const sel = selectSceneLookByRules(link.seriesScene, cue);
      sceneDefaultLooks[link.sceneId] = sel.lookId || link.seriesScene.default_look_id;

      if (sel.createdStub && sel.episodeLookStub) {
        const prev = sceneVariants[link.sceneId] ?? {
          scene_id: link.sceneId,
          scope: 'episode' as const,
        };
        const looks = [...(prev.looks ?? [])];
        looks.push(sceneLookToEpisodeVariant(sel.episodeLookStub, link.seriesScene.default_look_id));
        sceneVariants[link.sceneId] = {
          ...prev,
          scene_id: link.sceneId,
          default_look_id: sel.lookId,
          looks,
        };
      } else if (sel.lookId && sel.lookId !== link.seriesScene.default_look_id) {
        const prev = sceneVariants[link.sceneId] ?? {
          scene_id: link.sceneId,
          scope: 'episode' as const,
        };
        sceneVariants[link.sceneId] = {
          ...prev,
          scene_id: link.sceneId,
          default_look_id: sel.lookId,
        };
      }
      continue;
    }

    const looks =
      inc.looks.length > 0
        ? inc.looks.map((lk) => ({ ...lk, id: lk.id || newStableId('esk_') }))
        : [{ id: newStableId('esk_'), label: 'default', body: '' }];
    const defaultLookId = looks.find((l) => l.id === inc.default_look_id)?.id ?? looks[0].id;
    sceneDefaultLooks[link.sceneId] = defaultLookId;
    sceneVariants[link.sceneId] = {
      scene_id: link.sceneId,
      episode_only: true,
      scope: 'episode',
      name: inc.name,
      default_look_id: defaultLookId,
      looks: looks.map((lk) => sceneLookToEpisodeVariant(lk)),
    };
  }

  return {
    characterVariants,
    sceneVariants,
    characterLinks,
    sceneLinks,
    characterDefaultLooks,
    sceneDefaultLooks,
    reusedNames: [...new Set(reusedNames)],
    newEpisodeNames: [...new Set(newEpisodeNames)],
  };
}

/** Remap shot cast bindings through character link map (parse name → stable id). */
export function remapShotsThroughCharacterLinks(
  shots: LongVideoShotState[],
  characterLinks: CharacterLinkMap,
  characters: LongVideoCharacter[],
  defaultLooks?: Record<string, string>,
): LongVideoShotState[] {
  const byId = new Map(characters.map((c) => [c.id, c]));
  const resolveId = (raw: string): string | undefined => {
    const t = raw.trim();
    if (!t) return undefined;
    if (byId.has(t)) return t;
    const linked = characterLinks[t] || characterLinks[normKey(t)];
    if (linked && byId.has(linked)) return linked;
    for (const ch of characters) {
      if (characterNameKeys(ch).includes(normKey(t))) return ch.id;
    }
    return undefined;
  };

  return shots.map((shot) => {
    const onScreen = (shot.characters_on_screen ?? [])
      .map((n) => {
        const id = resolveId(n);
        if (!id) return n;
        return byId.get(id)?.name ?? n;
      })
      .filter(Boolean);

    let castLooks: LongVideoShotCastLook[] = (shot.cast_looks ?? [])
      .map((cl) => {
        const id = resolveId(cl.character_id) ?? cl.character_id;
        const ch = byId.get(id);
        const lookId =
          cl.look_id
          || defaultLooks?.[id]
          || ch?.default_look_id
          || ch?.looks[0]?.id
          || '';
        return { character_id: id, look_id: lookId };
      })
      .filter((c) => c.look_id);

    if (!castLooks.length && onScreen.length) {
      castLooks = onScreen
        .map((name) => {
          const id = resolveId(name);
          if (!id) return null;
          const ch = byId.get(id);
          const lookId =
            defaultLooks?.[id] || ch?.default_look_id || ch?.looks[0]?.id || '';
          return lookId ? { character_id: id, look_id: lookId } : null;
        })
        .filter((c): c is LongVideoShotCastLook => Boolean(c));
    }

    return {
      ...shot,
      characters_on_screen: onScreen.length ? onScreen : shot.characters_on_screen,
      cast_looks: castLooks.length ? castLooks : shot.cast_looks,
    };
  });
}

/**
 * Remap all shot cast_looks from old character id to new (manual relink).
 */
export function remapCharacterIdInShots(
  shots: LongVideoShotState[],
  fromId: string,
  toId: string,
  toDefaultLookId: string,
): LongVideoShotState[] {
  return shots.map((shot) => {
    const cast = (shot.cast_looks ?? []).map((cl) =>
      cl.character_id === fromId
        ? { character_id: toId, look_id: cl.look_id || toDefaultLookId }
        : cl,
    );
    return { ...shot, cast_looks: cast.length ? cast : shot.cast_looks };
  });
}
