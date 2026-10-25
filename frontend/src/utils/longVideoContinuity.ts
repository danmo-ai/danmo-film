import type {
  LongVideoCharacterRelation,
  LongVideoProjectState,
  LongVideoProp,
  LongVideoSeriesState,
  LongVideoShotState,
} from '@/types';
import { effectiveEpisodeRoster } from '@/utils/longVideoSeries';

export type ContinuityIssueCode =
  | 'relation_missing_endpoint'
  | 'prop_missing_reference'
  | 'character_not_in_series'
  | 'library_version_drift'
  | 'episode_missing_prev_summary';

export interface ContinuityIssue {
  code: ContinuityIssueCode;
  messageKey: string;
  detail?: string;
}

export interface CrossEpisodeConsistencyReport {
  sharedCharacterIds: string[];
  episodeOnlyCharacterIds: string[];
  variantDivergence: Array<{ character_id: string; episodes: string[] }>;
  styleAnchorMismatch: boolean;
  styleAnchors: Record<string, string>;
}

function seriesCharacterIds(series: LongVideoSeriesState): Set<string> {
  return new Set((series.characters ?? []).map((c) => c.id));
}

function shotMentionsPropId(shot: LongVideoShotState, propId: string): boolean {
  const blob = [
    shot.visual_prompt,
    shot.motion_prompt,
    shot.video_prompt,
    shot.scene_prompt,
    shot.start_visual_prompt,
  ]
    .filter(Boolean)
    .join('\n');
  return blob.includes(propId);
}

export function continuityIssues(
  series: LongVideoSeriesState,
  episodeSummaries: Record<string, { prev_summary?: string; ending_summary?: string }>,
  currentEpisode: Pick<
    LongVideoProjectState,
    | 'project_id'
    | 'episode_number'
    | 'characters'
    | 'scenes'
    | 'character_variants'
    | 'scene_variants'
    | 'series_id'
    | 'opening_from_prev'
    | 'shots'
  >,
): ContinuityIssue[] {
  const issues: ContinuityIssue[] = [];
  const masterIds = seriesCharacterIds(series);

  for (const rel of series.relations ?? []) {
    if (!masterIds.has(rel.from_id) || !masterIds.has(rel.to_id)) {
      issues.push({
        code: 'relation_missing_endpoint',
        messageKey: 'video.continuityRelationMissingEndpoint',
        detail: `${rel.from_id}→${rel.to_id}`,
      });
    }
  }

  const roster = effectiveEpisodeRoster(currentEpisode, series);
  for (const ch of roster.characters) {
    if (!masterIds.has(ch.id)) {
      issues.push({
        code: 'character_not_in_series',
        messageKey: 'video.continuityCharacterNotInSeries',
        detail: ch.name || ch.id,
      });
    }
  }

  for (const prop of series.props ?? []) {
    if (prop.reference_asset_id) continue;
    const usedInShot = currentEpisode.shots?.some((s) => shotMentionsPropId(s, prop.id));
    if (usedInShot) {
      issues.push({
        code: 'prop_missing_reference',
        messageKey: 'video.continuityPropMissingReference',
        detail: prop.name || prop.id,
      });
    }
  }

  for (const ch of series.characters ?? []) {
    if (!ch.library_entity_id || typeof ch.library_version !== 'number') continue;
    for (const [epId, patch] of Object.entries(episodeSummaries)) {
      if (epId === currentEpisode.project_id) continue;
      void patch;
    }
    const variant = currentEpisode.character_variants?.[ch.id];
    if (variant?.base_version != null && variant.base_version < ch.library_version) {
      issues.push({
        code: 'library_version_drift',
        messageKey: 'video.continuityLibraryVersionDrift',
        detail: ch.name || ch.id,
      });
    }
  }

  const epNum = currentEpisode.episode_number ?? 1;
  if (epNum > 1 && currentEpisode.project_id) {
    const prevKey = Object.keys(episodeSummaries).find(() => false);
    void prevKey;
    const cont = series.episode_continuity?.[currentEpisode.project_id];
    if (!cont?.prev_summary?.trim() && !currentEpisode.opening_from_prev?.trim()) {
      issues.push({
        code: 'episode_missing_prev_summary',
        messageKey: 'video.continuityMissingPrevSummary',
      });
    }
  }

  return issues;
}

export function crossEpisodeConsistencyReport(
  series: LongVideoSeriesState,
  episodeStates: Array<
    Pick<
      LongVideoProjectState,
      'project_id' | 'character_variants' | 'scene_variants' | 'style_anchor' | 'series_id'
    >
  >,
): CrossEpisodeConsistencyReport {
  const masterIds = seriesCharacterIds(series);
  const perEpChars: Record<string, Set<string>> = {};
  const styleAnchors: Record<string, string> = {};
  const variantChars: Map<string, Set<string>> = new Map();

  for (const ep of episodeStates) {
    const id = ep.project_id || 'unknown';
    const roster = effectiveEpisodeRoster(
      {
        characters: [],
        scenes: [],
        character_variants: ep.character_variants,
        scene_variants: ep.scene_variants,
        series_id: ep.series_id,
      },
      series,
    );
    perEpChars[id] = new Set(roster.characters.map((c) => c.id));
    styleAnchors[id] = (ep.style_anchor ?? series.style_anchor ?? '').trim();

    for (const [charId, variant] of Object.entries(ep.character_variants ?? {})) {
      if (!variant?.looks?.length) continue;
      const set = variantChars.get(charId) ?? new Set<string>();
      set.add(id);
      variantChars.set(charId, set);
    }
  }

  const allEpIds = Object.keys(perEpChars);
  const shared = [...masterIds].filter((cid) =>
    allEpIds.every((eid) => perEpChars[eid]?.has(cid)),
  );
  const episodeOnly = [...masterIds].filter((cid) => !shared.includes(cid));

  const variantDivergence = [...variantChars.entries()]
    .filter(([, eps]) => eps.size > 1)
    .map(([character_id, eps]) => ({ character_id, episodes: [...eps] }));

  const anchorValues = new Set(Object.values(styleAnchors).filter(Boolean));
  const styleAnchorMismatch = anchorValues.size > 1;

  return {
    sharedCharacterIds: shared,
    episodeOnlyCharacterIds: episodeOnly,
    variantDivergence,
    styleAnchorMismatch,
    styleAnchors,
  };
}

export function mergeSeriesContinuityFields(
  series: LongVideoSeriesState,
  patch: {
    relations?: LongVideoCharacterRelation[];
    props?: LongVideoProp[];
    episode_continuity?: LongVideoSeriesState['episode_continuity'];
  },
): LongVideoSeriesState {
  return {
    ...series,
    relations: patch.relations ?? series.relations,
    props: patch.props ?? series.props,
    episode_continuity: patch.episode_continuity ?? series.episode_continuity,
  };
}
