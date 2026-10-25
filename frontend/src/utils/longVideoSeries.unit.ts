/**
 * Run: npx tsx frontend/src/utils/longVideoSeries.unit.ts
 */
import {
  clearCharacterVariant,
  characterHasEpisodeOverride,
  defaultSeriesState,
  deriveCharacterLookVariant,
  diffCharacterAgainstSeries,
  effectiveEpisodeRoster,
  episodeOnlyCharacterVariant,
  materializeCharacters,
  promoteCharacterLookToSeries,
  promoteEpisodeCharacterToSeries,
  restoreCharacterFromSeries,
  upsertCharacterVariant,
  workspaceRosterRevisions,
} from './longVideoSeries';
import type { LongVideoCharacter, LongVideoProjectState } from '@/types';

function assert(cond: unknown, msg: string) {
  if (!cond) throw new Error(msg);
}

const baseChars: LongVideoCharacter[] = [
  {
    id: 'c1',
    name: '林婉',
    default_look_id: 'l1',
    looks: [
      {
        id: 'l1',
        label: '日常',
        body: '黑发白裙',
        reference_asset_id: 'fas_base',
      },
    ],
  },
];

const series = defaultSeriesState({ characters: baseChars, scenes: [] });

const inherited = materializeCharacters(baseChars, {});
assert(inherited[0].looks[0].reference_asset_id === 'fas_base', 'inherit ref');
assert(inherited[0].name === '林婉', 'inherit name');

const withOverride = materializeCharacters(baseChars, {
  c1: {
    character_id: 'c1',
    name: '林总',
    looks: [{ look_id: 'l1', body: '黑发红裙', reference_asset_id: null }],
  },
});
assert(withOverride[0].name === '林总', 'override name');
assert(withOverride[0].looks[0].body === '黑发红裙', 'override body');
assert(withOverride[0].looks[0].reference_asset_id === undefined, 'explicit null clears ref');

const { lookId, variant } = deriveCharacterLookVariant('c1', baseChars[0].looks[0], {
  label: '受伤',
  body: '绷带',
});
const vars = upsertCharacterVariant({}, 'c1', { looks: [variant] });
const derived = materializeCharacters(baseChars, vars);
assert(derived[0].looks.length === 2, 'derived look added');
assert(derived[0].looks.some((lk) => lk.id === lookId && lk.label === '受伤'), 'derived label');

const episode: Pick<
  LongVideoProjectState,
  'characters' | 'scenes' | 'character_variants' | 'scene_variants' | 'series_id'
> = {
  series_id: 'lvs_1',
  characters: [],
  scenes: [],
  character_variants: vars,
  scene_variants: {},
};
const roster = effectiveEpisodeRoster(episode, series);
assert(roster.characters[0].looks.length === 2, 'effective uses series+variants');

const standalone = effectiveEpisodeRoster(
  { characters: baseChars, scenes: [] },
  null,
);
assert(standalone.characters[0].name === '林婉', 'standalone uses local roster');

const diff = diffCharacterAgainstSeries(baseChars[0], {
  ...baseChars[0],
  name: '林总',
  looks: [{ ...baseChars[0].looks[0], body: '黑发红裙' }],
});
assert(diff?.name === '林总', 'diff character name');
assert(diff?.looks?.[0]?.body === '黑发红裙', 'diff look body');

const identityDiff = diffCharacterAgainstSeries(baseChars[0], {
  ...baseChars[0],
  visual_identity: '瓜子脸',
  role: 'protagonist',
});
assert(identityDiff?.visual_identity === '瓜子脸', 'diff visual_identity');
assert(identityDiff?.role === 'protagonist', 'diff role');
const identityMat = materializeCharacters(baseChars, {
  c1: { character_id: 'c1', visual_identity: '瓜子脸', role: 'protagonist' },
});
assert(identityMat[0]?.visual_identity === '瓜子脸', 'materialize visual_identity');
assert(identityMat[0]?.role === 'protagonist', 'materialize role');
assert(characterHasEpisodeOverride({ c1: identityDiff! }, 'c1'), 'identity-only override counts');

const cleared = clearCharacterVariant(vars, 'c1');
assert(!cleared.c1, 'clear character variant');
const restored = restoreCharacterFromSeries(vars, 'c1');
assert(!restored.c1, 'restore clears variant');

const rev = workspaceRosterRevisions(
  { series_id: 'lvs_1', character_variants: { c1: { character_id: 'c1', base_version: 3 } } },
  series,
);
assert(rev.c1 === 3, 'workspace revisions from variant base_version');

const seriesCopy = JSON.parse(JSON.stringify(series)) as typeof series;
materializeCharacters(series.characters, vars);
assert(JSON.stringify(series) === JSON.stringify(seriesCopy), 'materialize does not mutate series');

const promoted = promoteCharacterLookToSeries(series.characters, 'c1', {
  id: 'l_ep',
  label: 'Ep look',
  body: 'red coat',
});
assert(promoted[0].looks.some((lk) => lk.id === 'l_ep'), 'promote adds look to series master');
assert(series.characters[0].looks.length === 1, 'promote does not mutate input array in place');

const epOnly: LongVideoCharacter = {
  id: 'c_ep',
  name: '本集路人',
  default_look_id: 'l_ep_only',
  looks: [{ id: 'l_ep_only', label: '默认', body: '灰衣' }],
};
const epVars = { c_ep: episodeOnlyCharacterVariant(epOnly) };
assert(characterHasEpisodeOverride(epVars, 'c_ep'), 'episode-only entity is promotable');
const withEpOnly = materializeCharacters(baseChars, epVars);
assert(withEpOnly.some((c) => c.id === 'c_ep' && c.name === '本集路人'), 'materialize episode-only char');
const toSeries = promoteEpisodeCharacterToSeries(series, withEpOnly.find((c) => c.id === 'c_ep')!, epVars);
assert(toSeries.series.characters.some((c) => c.id === 'c_ep'), 'promote writes entity into series');
assert(!toSeries.characterVariants.c_ep, 'promote clears episode variant');

const overridePromote = promoteEpisodeCharacterToSeries(
  series,
  { ...baseChars[0], name: '林总', looks: [{ ...baseChars[0].looks[0], body: '红裙' }] },
  {
    c1: {
      character_id: 'c1',
      name: '林总',
      looks: [{ look_id: 'l1', body: '红裙' }],
    },
  },
);
assert(overridePromote.series.characters[0].name === '林总', 'promote updates series master fields');
assert(overridePromote.series.characters[0].looks[0].body === '红裙', 'promote updates look body');
assert(!overridePromote.characterVariants.c1, 'promote clears shared-look overrides');

// Episode scope: identity patches dropped when looksAndDefaultOnly
const identityLocked = diffCharacterAgainstSeries(
  baseChars[0],
  { ...baseChars[0], name: '改名', role: 'protagonist', default_look_id: 'l1' },
  { looksAndDefaultOnly: true },
);
assert(!identityLocked || identityLocked.name === undefined, 'RO identity: name not patched');
assert(!identityLocked || identityLocked.role === undefined, 'RO identity: role not patched');

// Character promote leaves episode_only looks on episode
const withEpLook = promoteEpisodeCharacterToSeries(
  series,
  {
    ...baseChars[0],
    looks: [
      ...baseChars[0].looks,
      { id: 'l_ep_new', label: '本集铠甲', body: '铁甲' },
    ],
  },
  {
    c1: {
      character_id: 'c1',
      looks: [
        {
          look_id: 'l_ep_new',
          episode_only: true,
          label: '本集铠甲',
          body: '铁甲',
        },
      ],
    },
  },
);
assert(
  !withEpLook.series.characters[0].looks.some((lk) => lk.id === 'l_ep_new'),
  'character promote does not auto-promote episode_only look',
);
assert(
  withEpLook.characterVariants.c1?.looks?.some((lk) => lk.look_id === 'l_ep_new' && lk.episode_only),
  'episode_only look stays on episode after character promote',
);

console.log('longVideoSeries.unit: ok');
