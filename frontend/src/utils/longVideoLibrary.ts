/**
 * Series roster ↔ global character/scene library (versioned definitions).
 */

import type {
  LibraryEntityVersion,
  LongVideoAssetStatus,
  LongVideoCharacter,
  LongVideoCharacterLook,
  LongVideoScene,
  LongVideoSceneLook,
  LongVideoSeriesDetail,
  LongVideoSeriesState,
} from '@/types';
import { newStableId } from './longVideoSeries';

/**
 * Status lives on the **library version** (`definition.version_status`).
 * Roster `asset_status` is only the working mode: viewing a final pin vs editing a draft.
 */
export function libraryVersionStatus(
  row: Pick<LibraryEntityVersion, 'definition'> | Record<string, unknown>,
): LongVideoAssetStatus {
  const def =
    'definition' in row && row.definition && typeof row.definition === 'object'
      ? (row.definition as Record<string, unknown>)
      : (row as Record<string, unknown>);
  const s = def.version_status;
  if (s === 'draft' || s === 'final') return s;
  // Legacy library rows (pre-status) are treated as finalized.
  return 'final';
}

export function withLibraryVersionStatus(
  definition: Record<string, unknown>,
  status: LongVideoAssetStatus,
): Record<string, unknown> {
  return { ...definition, version_status: status };
}

/** Working mode on the project roster row (not the version itself). */
export function characterAssetStatus(ch: Pick<LongVideoCharacter, 'asset_status' | 'library_entity_id' | 'library_version'>): LongVideoAssetStatus {
  if (ch.asset_status === 'draft' || ch.asset_status === 'final') return ch.asset_status;
  return ch.library_entity_id && ch.library_version ? 'final' : 'draft';
}

export function sceneAssetStatus(sc: Pick<LongVideoScene, 'asset_status' | 'library_entity_id' | 'library_version'>): LongVideoAssetStatus {
  if (sc.asset_status === 'draft' || sc.asset_status === 'final') return sc.asset_status;
  return sc.library_entity_id && sc.library_version ? 'final' : 'draft';
}

/** Open a locked final pin as an editable draft (定稿 writes a new final version). */
export function forkCharacterDraft(ch: LongVideoCharacter): LongVideoCharacter {
  return { ...ch, asset_status: 'draft' };
}

export function forkSceneDraft(sc: LongVideoScene): LongVideoScene {
  return { ...sc, asset_status: 'draft' };
}

export type LibraryApi = {
  createCharacter(definition: Record<string, unknown>): Promise<LibraryEntityVersion>;
  getCharacter(entityId: string, version?: number): Promise<LibraryEntityVersion>;
  newCharacterVersion(
    entityId: string,
    definition: Record<string, unknown>,
  ): Promise<LibraryEntityVersion>;
  listCharacters(limit?: number): Promise<LibraryEntityVersion[]>;
  listCharacterVersions(entityId: string): Promise<LibraryEntityVersion[]>;
  createScene(definition: Record<string, unknown>): Promise<LibraryEntityVersion>;
  getScene(entityId: string, version?: number): Promise<LibraryEntityVersion>;
  newSceneVersion(entityId: string, definition: Record<string, unknown>): Promise<LibraryEntityVersion>;
  listScenes(limit?: number): Promise<LibraryEntityVersion[]>;
  listSceneVersions(entityId: string): Promise<LibraryEntityVersion[]>;
};

export type LibraryUpdateRow = {
  kind: 'character' | 'scene';
  rosterId: string;
  name: string;
  libraryEntityId: string;
  lockedVersion: number;
  latestVersion: number;
};

function lookStatusFromDef(raw: unknown): LongVideoAssetStatus {
  return raw === 'draft' ? 'draft' : 'final';
}

function looksFromDef(raw: unknown): LongVideoCharacterLook[] {
  if (!Array.isArray(raw)) return [];
  return raw
    .filter((x) => x && typeof x === 'object')
    .map((lk) => {
      const o = lk as Record<string, unknown>;
      return {
        id: String(o.id ?? newStableId('clk_')),
        label: String(o.label ?? ''),
        body: String(o.body ?? ''),
        vision_description:
          o.vision_description != null ? String(o.vision_description) : undefined,
        reference_asset_id:
          o.reference_asset_id != null ? String(o.reference_asset_id) : undefined,
        lora_id: o.lora_id != null ? String(o.lora_id) : undefined,
        portrait_prompt: o.portrait_prompt != null ? String(o.portrait_prompt) : undefined,
        // Library snapshots: missing status ⇒ finalized look.
        look_status: lookStatusFromDef(o.look_status),
      };
    });
}

function sceneLooksFromDef(raw: unknown): LongVideoSceneLook[] {
  if (!Array.isArray(raw)) return [];
  return raw
    .filter((x) => x && typeof x === 'object')
    .map((lk) => {
      const o = lk as Record<string, unknown>;
      return {
        id: String(o.id ?? newStableId('slk_')),
        label: String(o.label ?? ''),
        body: String(o.body ?? ''),
        vision_description:
          o.vision_description != null ? String(o.vision_description) : undefined,
        reference_asset_id:
          o.reference_asset_id != null ? String(o.reference_asset_id) : undefined,
        environment_prompt:
          o.environment_prompt != null ? String(o.environment_prompt) : undefined,
        look_status: lookStatusFromDef(o.look_status),
      };
    });
}

/** Resolve look lock. Explicit final is locked; draft or missing stays editable. */
export function characterLookStatus(
  look: Pick<LongVideoCharacterLook, 'look_status'>,
): LongVideoAssetStatus {
  return look.look_status === 'final' ? 'final' : 'draft';
}

export function sceneLookStatus(
  look: Pick<LongVideoSceneLook, 'look_status'>,
): LongVideoAssetStatus {
  return look.look_status === 'final' ? 'final' : 'draft';
}

export function finalizeCharacterLook(
  ch: LongVideoCharacter,
  lookId: string,
): LongVideoCharacter {
  return {
    ...ch,
    looks: ch.looks.map((lk) =>
      lk.id === lookId ? { ...lk, look_status: 'final' as const } : lk,
    ),
  };
}

export function finalizeSceneLook(sc: LongVideoScene, lookId: string): LongVideoScene {
  return {
    ...sc,
    looks: sc.looks.map((lk) =>
      lk.id === lookId ? { ...lk, look_status: 'final' as const } : lk,
    ),
  };
}

/** Character/scene finalize: lock every look included in this version snapshot. */
export function freezeAllCharacterLooks(ch: LongVideoCharacter): LongVideoCharacter {
  return {
    ...ch,
    looks: ch.looks.map((lk) => ({ ...lk, look_status: 'final' as const })),
  };
}

export function freezeAllSceneLooks(sc: LongVideoScene): LongVideoScene {
  return {
    ...sc,
    looks: sc.looks.map((lk) => ({ ...lk, look_status: 'final' as const })),
  };
}

function persistLookStatus(
  look: Pick<LongVideoCharacterLook | LongVideoSceneLook, 'look_status'>,
): LongVideoAssetStatus {
  return look.look_status === 'draft' ? 'draft' : 'final';
}

export function characterDefinitionFromRoster(ch: LongVideoCharacter): Record<string, unknown> {
  return {
    roster_id: ch.id,
    name: ch.name,
    role: ch.role ?? 'supporting',
    aliases: ch.aliases ?? [],
    age_range: ch.age_range ?? '',
    gender_presentation: ch.gender_presentation ?? '',
    identity_baseline: ch.identity_baseline ?? '',
    visual_identity: ch.visual_identity ?? '',
    voice_manner: ch.voice_manner ?? '',
    performance_notes: ch.performance_notes ?? '',
    identity_notes: ch.identity_notes ?? '',
    // Preserve per-look status so draft looks on a final character stay editable.
    looks: ch.looks.map((lk) => ({ ...lk, look_status: persistLookStatus(lk) })),
    default_look_id: ch.default_look_id,
  };
}

export function sceneDefinitionFromRoster(sc: LongVideoScene): Record<string, unknown> {
  return {
    roster_id: sc.id,
    name: sc.name,
    looks: sc.looks.map((lk) => ({ ...lk, look_status: persistLookStatus(lk) })),
    default_look_id: sc.default_look_id,
    spatial_layout_json: sc.spatial_layout_json ?? null,
    grounding_panorama_asset_id: sc.grounding_panorama_asset_id ?? null,
    grounding_depth_asset_id: sc.grounding_depth_asset_id ?? null,
    time_of_day: sc.time_of_day ?? '',
    weather: sc.weather ?? '',
  };
}

export function characterFromLibraryDefinition(
  def: Record<string, unknown>,
  entityId: string,
  version: number,
  rosterId?: string,
): LongVideoCharacter {
  const id = rosterId ?? String(def.roster_id ?? def.id ?? newStableId('lc_'));
  const looks = looksFromDef(def.looks);
  const defaultLookId = String(def.default_look_id ?? looks[0]?.id ?? '');
  const aliases = Array.isArray(def.aliases)
    ? def.aliases.map((a) => String(a)).filter(Boolean)
    : undefined;
  const roleRaw = def.role != null ? String(def.role) : '';
  const role =
    roleRaw === 'protagonist' || roleRaw === 'supporting' || roleRaw === 'extra'
      ? roleRaw
      : undefined;
  const str = (k: string) => (def[k] != null ? String(def[k]) : undefined);
  return {
    id,
    name: String(def.name ?? ''),
    looks: looks.length ? looks : [{ id: newStableId('clk_'), label: '默认', body: '' }],
    default_look_id: defaultLookId || looks[0]?.id || '',
    role,
    aliases: aliases?.length ? aliases : undefined,
    age_range: str('age_range') || undefined,
    gender_presentation: str('gender_presentation') || undefined,
    identity_baseline: str('identity_baseline') || undefined,
    visual_identity: str('visual_identity') || str('identity_notes') || undefined,
    voice_manner: str('voice_manner') || undefined,
    performance_notes: str('performance_notes') || undefined,
    identity_notes: str('identity_notes') || undefined,
    library_entity_id: entityId,
    library_version: version,
    asset_status: 'final',
  };
}

export function sceneFromLibraryDefinition(
  def: Record<string, unknown>,
  entityId: string,
  version: number,
  rosterId?: string,
): LongVideoScene {
  const id = rosterId ?? String(def.roster_id ?? def.id ?? newStableId('ls_'));
  const looks = sceneLooksFromDef(def.looks);
  const defaultLookId = String(def.default_look_id ?? looks[0]?.id ?? '');
  return {
    id,
    name: String(def.name ?? ''),
    looks: looks.length ? looks : [{ id: newStableId('slk_'), label: '默认', body: '' }],
    default_look_id: defaultLookId || looks[0]?.id || '',
    spatial_layout_json:
      def.spatial_layout_json && typeof def.spatial_layout_json === 'object'
        ? (def.spatial_layout_json as Record<string, unknown>)
        : undefined,
    grounding_panorama_asset_id:
      def.grounding_panorama_asset_id != null
        ? String(def.grounding_panorama_asset_id)
        : undefined,
    grounding_depth_asset_id:
      def.grounding_depth_asset_id != null
        ? String(def.grounding_depth_asset_id)
        : undefined,
    time_of_day: def.time_of_day != null ? String(def.time_of_day) : undefined,
    weather: def.weather != null ? String(def.weather) : undefined,
    library_entity_id: entityId,
    library_version: version,
    asset_status: 'final',
  };
}

export async function migrateSeriesAssetsToLibrary(
  seriesState: LongVideoSeriesState,
  library: LibraryApi,
): Promise<{ state: LongVideoSeriesState; changed: boolean }> {
  let changed = false;
  const characters: LongVideoCharacter[] = [];
  for (const ch of seriesState.characters ?? []) {
    if (ch.library_entity_id && ch.library_version) {
      characters.push(ch);
      continue;
    }
    const row = await library.createCharacter(characterDefinitionFromRoster(ch));
    characters.push(
      characterFromLibraryDefinition(row.definition, row.entity_id, row.version, ch.id),
    );
    changed = true;
  }

  const scenes: LongVideoScene[] = [];
  for (const sc of seriesState.scenes ?? []) {
    if (sc.library_entity_id && sc.library_version) {
      scenes.push(sc);
      continue;
    }
    const row = await library.createScene(sceneDefinitionFromRoster(sc));
    scenes.push(sceneFromLibraryDefinition(row.definition, row.entity_id, row.version, sc.id));
    changed = true;
  }

  if (!changed) return { state: seriesState, changed: false };
  return {
    state: { ...seriesState, characters, scenes },
    changed: true,
  };
}

export async function refreshSeriesFromLibrary(
  seriesState: LongVideoSeriesState,
  library: LibraryApi,
  opts: { upgrade: boolean },
): Promise<LongVideoSeriesState> {
  const characters: LongVideoCharacter[] = [];
  for (const ch of seriesState.characters ?? []) {
    if (!ch.library_entity_id) {
      characters.push(ch);
      continue;
    }
    const version = opts.upgrade ? 0 : ch.library_version ?? 0;
    const row = await library.getCharacter(ch.library_entity_id, version);
    characters.push(
      characterFromLibraryDefinition(row.definition, row.entity_id, row.version, ch.id),
    );
  }

  const scenes: LongVideoScene[] = [];
  for (const sc of seriesState.scenes ?? []) {
    if (!sc.library_entity_id) {
      scenes.push(sc);
      continue;
    }
    const version = opts.upgrade ? 0 : sc.library_version ?? 0;
    const row = await library.getScene(sc.library_entity_id, version);
    scenes.push(sceneFromLibraryDefinition(row.definition, row.entity_id, row.version, sc.id));
  }

  return { ...seriesState, characters, scenes };
}

export function listLibraryUpdates(
  seriesState: LongVideoSeriesState,
  libraryLatest: { characters: LibraryEntityVersion[]; scenes: LibraryEntityVersion[] },
): LibraryUpdateRow[] {
  const out: LibraryUpdateRow[] = [];
  const charLatest = new Map(libraryLatest.characters.map((r) => [r.entity_id, r]));
  const sceneLatest = new Map(libraryLatest.scenes.map((r) => [r.entity_id, r]));

  for (const ch of seriesState.characters ?? []) {
    const eid = ch.library_entity_id;
    const locked = ch.library_version ?? 0;
    if (!eid || !locked) continue;
    const latest = charLatest.get(eid);
    if (latest && latest.version > locked) {
      out.push({
        kind: 'character',
        rosterId: ch.id,
        name: ch.name,
        libraryEntityId: eid,
        lockedVersion: locked,
        latestVersion: latest.version,
      });
    }
  }

  for (const sc of seriesState.scenes ?? []) {
    const eid = sc.library_entity_id;
    const locked = sc.library_version ?? 0;
    if (!eid || !locked) continue;
    const latest = sceneLatest.get(eid);
    if (latest && latest.version > locked) {
      out.push({
        kind: 'scene',
        rosterId: sc.id,
        name: sc.name,
        libraryEntityId: eid,
        lockedVersion: locked,
        latestVersion: latest.version,
      });
    }
  }

  return out;
}

/** Pull one library entity at `targetVersion` (or latest when 0) into the series roster. */
export async function upgradeSeriesEntityLibraryVersion(
  seriesState: LongVideoSeriesState,
  library: LibraryApi,
  row: LibraryUpdateRow,
  targetVersion: number,
): Promise<LongVideoSeriesState> {
  if (row.kind === 'character') {
    const rowData = await library.getCharacter(row.libraryEntityId, targetVersion);
    const nextChars = (seriesState.characters ?? []).map((ch) =>
      ch.id === row.rosterId
        ? characterFromLibraryDefinition(rowData.definition, rowData.entity_id, rowData.version, ch.id)
        : ch,
    );
    return { ...seriesState, characters: nextChars };
  }
  const rowData = await library.getScene(row.libraryEntityId, targetVersion);
  const nextScenes = (seriesState.scenes ?? []).map((sc) =>
    sc.id === row.rosterId
      ? sceneFromLibraryDefinition(rowData.definition, rowData.entity_id, rowData.version, sc.id)
      : sc,
  );
  return { ...seriesState, scenes: nextScenes };
}

export async function publishCharacterToLibrary(
  ch: LongVideoCharacter,
  library: LibraryApi,
): Promise<LongVideoCharacter> {
  const def = withLibraryVersionStatus(characterDefinitionFromRoster(ch), 'final');
  if (!ch.library_entity_id) {
    const row = await library.createCharacter(def);
    return {
      ...characterFromLibraryDefinition(row.definition, row.entity_id, row.version, ch.id),
      asset_status: 'final',
    };
  }
  const row = await library.newCharacterVersion(ch.library_entity_id, def);
  return {
    ...characterFromLibraryDefinition(row.definition, row.entity_id, row.version, ch.id),
    asset_status: 'final',
  };
}

export async function publishSceneToLibrary(
  sc: LongVideoScene,
  library: LibraryApi,
): Promise<LongVideoScene> {
  const def = withLibraryVersionStatus(sceneDefinitionFromRoster(sc), 'final');
  if (!sc.library_entity_id) {
    const row = await library.createScene(def);
    return {
      ...sceneFromLibraryDefinition(row.definition, row.entity_id, row.version, sc.id),
      asset_status: 'final',
    };
  }
  const row = await library.newSceneVersion(sc.library_entity_id, def);
  return {
    ...sceneFromLibraryDefinition(row.definition, row.entity_id, row.version, sc.id),
    asset_status: 'final',
  };
}

/** Load series, migrate unlinked roster rows to library once, persist when changed. */
export async function loadSeriesWithLibraryMigration(
  getSeries: (id: string) => Promise<LongVideoSeriesDetail>,
  updateSeries: (
    id: string,
    body: { title?: string; state?: LongVideoSeriesState },
  ) => Promise<LongVideoSeriesDetail>,
  seriesId: string,
  library: LibraryApi,
): Promise<LongVideoSeriesDetail> {
  const detail = await getSeries(seriesId);
  const base = { ...detail.state, version: 1 as const };
  const { state: migrated, changed } = await migrateSeriesAssetsToLibrary(base, library);
  if (!changed) return { ...detail, state: migrated };
  return updateSeries(seriesId, { title: detail.title, state: migrated });
}
