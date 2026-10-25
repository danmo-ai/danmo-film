/**
 * Run: npx tsx frontend/src/utils/longVideoEpisodeAttach.unit.ts
 */
import {
  attachParseRosterToEpisode,
  remapShotsThroughCharacterLinks,
  resolveCharacterLink,
  scoreLookAgainstCue,
  selectCharacterLookByRules,
} from './longVideoEpisodeAttach';
import type { LongVideoCharacter, LongVideoShotState } from '@/types';

function assert(cond: unknown, msg: string) {
  if (!cond) throw new Error(msg);
}

const seriesChars: LongVideoCharacter[] = [
  {
    id: 'c_lin',
    name: '林婉',
    aliases: ['婉姐', '林总'],
    default_look_id: 'lk_daily',
    looks: [
      { id: 'lk_daily', label: '日常', body: '黑发白裙' },
      { id: 'lk_red', label: '晚宴', body: '红裙披肩' },
    ],
  },
  {
    id: 'c_chen',
    name: '陈默',
    default_look_id: 'ck1',
    looks: [{ id: 'ck1', label: '默认', body: '深色西装' }],
  },
];

// Exact name
{
  const r = resolveCharacterLink({ id: 'tmp', name: '林婉' }, seriesChars);
  assert(r.matched && r.characterId === 'c_lin', 'exact name match');
}

// Alias match
{
  const r = resolveCharacterLink({ id: 'tmp', name: '婉姐' }, seriesChars);
  assert(r.matched && r.characterId === 'c_lin', 'alias match');
}

// Soft containment
{
  const r = resolveCharacterLink({ id: 'tmp', name: '林婉姐' }, seriesChars);
  assert(r.matched && r.characterId === 'c_lin', 'soft containment match');
}

// Ambiguous soft → no auto pick
{
  const ambiguousSeries: LongVideoCharacter[] = [
    { id: 'a', name: '青衫客', default_look_id: 'l', looks: [{ id: 'l', label: 'a', body: '' }] },
    { id: 'b', name: '青衫剑客', default_look_id: 'l', looks: [{ id: 'l', label: 'b', body: '' }] },
  ];
  const r = resolveCharacterLink({ id: 'tmp', name: '青衫' }, ambiguousSeries);
  assert(!r.matched && r.ambiguous, 'ambiguous soft match stays episode_only');
}

// Look scoring
{
  assert(scoreLookAgainstCue({ label: '晚宴', body: '红裙披肩' }, '红裙') > 0.2, 'cue scores red look');
  const sel = selectCharacterLookByRules(seriesChars[0], '红裙晚宴');
  assert(sel.lookId === 'lk_red' && !sel.createdStub, 'rules select matching look');
  const noCue = selectCharacterLookByRules(seriesChars[0], '');
  assert(noCue.lookId === 'lk_daily' && !noCue.createdStub, 'empty cue keeps default');
  const stub = selectCharacterLookByRules(seriesChars[0], '铠甲铁盔战甲');
  assert(stub.createdStub && stub.episodeLookStub, 'unmatched cue creates episode look stub');
}

// Attach: same name reuses series, does not invent series write
{
  const incoming: LongVideoCharacter[] = [
    {
      id: 'parse_lin',
      name: '林婉',
      default_look_id: 'p1',
      looks: [{ id: 'p1', label: '晚宴', body: '红裙' }],
    },
    {
      id: 'parse_x',
      name: '路人甲',
      default_look_id: 'px',
      looks: [{ id: 'px', label: '默认', body: '灰衣' }],
    },
  ];
  const seriesCopy = JSON.parse(JSON.stringify(seriesChars)) as LongVideoCharacter[];
  const attached = attachParseRosterToEpisode({
    incomingCharacters: incoming,
    incomingScenes: [],
    seriesCharacters: seriesChars,
    seriesScenes: [],
  });
  assert(JSON.stringify(seriesChars) === JSON.stringify(seriesCopy), 'attach does not mutate series');
  assert(attached.characterLinks['林婉'.toLowerCase()] === 'c_lin' || attached.characterLinks['林婉'] === 'c_lin'
    || Object.values(attached.characterLinks).includes('c_lin'), 'link map points to series id');
  assert(attached.reusedNames.includes('林婉'), 'reused name listed');
  assert(attached.newEpisodeNames.some((n) => n.includes('路人')), 'new episode name listed');
  assert(attached.characterVariants.parse_x?.episode_only || attached.characterVariants[
    Object.keys(attached.characterVariants).find((id) => attached.characterVariants[id].episode_only)!
  ]?.episode_only, 'new char is episode_only');
  const epOnlyId = Object.keys(attached.characterVariants).find(
    (id) => attached.characterVariants[id].episode_only,
  );
  assert(epOnlyId && attached.characterVariants[epOnlyId].name === '路人甲', 'episode_only name preserved');
}

// Shot remap through links
{
  const links = { 行者: 'c_lin', xingzhe: 'c_lin' };
  const shots: LongVideoShotState[] = [
    {
      id: 's1',
      order: 0,
      visual_prompt: '行者走过',
      motion_prompt: '',
      duration_sec: 5,
      status: 'draft',
      characters_on_screen: ['行者'],
      cast_looks: [{ character_id: '行者', look_id: '' }],
    },
  ];
  const remapped = remapShotsThroughCharacterLinks(shots, links, seriesChars, {
    c_lin: 'lk_red',
  });
  assert(remapped[0].cast_looks?.[0]?.character_id === 'c_lin', 'remap cast to series id');
  assert(remapped[0].cast_looks?.[0]?.look_id === 'lk_red', 'remap fills default look');
  assert(remapped[0].characters_on_screen?.[0] === '林婉', 'on-screen remapped to series name');
}

console.log('longVideoEpisodeAttach.unit: ok');
