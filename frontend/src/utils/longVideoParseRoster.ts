/**
 * Text-only roster catalog for script-parse Story Map (reuse|create decisions).
 */
import type { LongVideoCharacter, LongVideoScene } from '@/types';

export type ExistingParseLook = {
  id: string;
  label: string;
  body: string;
};

export type ExistingParseCharacter = {
  id: string;
  name: string;
  aliases?: string[];
  visual_identity?: string;
  identity_baseline?: string;
  looks: ExistingParseLook[];
};

export type ExistingParseScene = {
  id: string;
  name: string;
  looks: ExistingParseLook[];
};

export type ExistingParseRoster = {
  characters: ExistingParseCharacter[];
  scenes: ExistingParseScene[];
};

function lookBodyFromSceneLook(lk: {
  body?: string;
  environment_prompt?: string;
  vision_description?: string;
}): string {
  return (
    lk.environment_prompt?.trim()
    || lk.vision_description?.trim()
    || lk.body?.trim()
    || ''
  );
}

/** Build existing_roster payload from effective cast/scenes (no reference images). */
export function buildExistingRosterForParse(
  characters: LongVideoCharacter[],
  scenes: LongVideoScene[],
): ExistingParseRoster {
  return {
    characters: characters
      .filter((ch) => ch.id && ch.name?.trim())
      .map((ch) => ({
        id: ch.id,
        name: ch.name.trim(),
        aliases: (ch.aliases ?? []).map((a) => a.trim()).filter(Boolean),
        visual_identity: (ch.visual_identity || ch.identity_notes || '').trim() || undefined,
        identity_baseline: ch.identity_baseline?.trim() || undefined,
        looks: (ch.looks ?? []).map((lk) => ({
          id: lk.id,
          label: lk.label?.trim() || '默认',
          body: lk.body?.trim() || '',
        })),
      })),
    scenes: scenes
      .filter((sc) => sc.id && sc.name?.trim())
      .map((sc) => ({
        id: sc.id,
        name: sc.name.trim(),
        looks: (sc.looks ?? []).map((lk) => ({
          id: lk.id,
          label: lk.label?.trim() || '默认',
          body: lookBodyFromSceneLook(lk),
        })),
      })),
  };
}
