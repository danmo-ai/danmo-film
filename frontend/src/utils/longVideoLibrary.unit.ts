/**
 * Run: npx tsx frontend/src/utils/longVideoLibrary.unit.ts
 */
import {
  characterDefinitionFromRoster,
  characterLookStatus,
  finalizeCharacterLook,
  freezeAllCharacterLooks,
  listLibraryUpdates,
  migrateSeriesAssetsToLibrary,
} from './longVideoLibrary';
import { defaultSeriesState } from './longVideoSeries';
import type { LibraryEntityVersion, LongVideoCharacter } from '@/types';

function assert(cond: unknown, msg: string) {
  if (!cond) throw new Error(msg);
}

const lookFixture: LongVideoCharacter = {
  id: 'c_look',
  name: 'Hero',
  default_look_id: 'l1',
  asset_status: 'final',
  looks: [
    { id: 'l1', label: 'a', body: 'x', look_status: 'final' },
    { id: 'l2', label: 'b', body: 'y', look_status: 'draft' },
  ],
};
assert(characterLookStatus(lookFixture.looks[0]) === 'final', 'final look locked');
assert(characterLookStatus(lookFixture.looks[1]) === 'draft', 'draft look editable');
assert(characterLookStatus({ look_status: undefined }) === 'draft', 'missing look status editable');

const afterLook = finalizeCharacterLook(lookFixture, 'l2');
assert(afterLook.looks[1].look_status === 'final', 'finalize one look');
assert(afterLook.looks[0].look_status === 'final', 'sibling look unchanged');

const def = characterDefinitionFromRoster({
  ...lookFixture,
  looks: [
    { id: 'l1', label: 'a', body: 'x', look_status: 'final' },
    { id: 'l2', label: 'b', body: 'y', look_status: 'draft' },
  ],
});
const defLooks = def.looks as Array<{ look_status: string }>;
assert(defLooks[0].look_status === 'final' && defLooks[1].look_status === 'draft', 'publish preserves draft looks');

const frozen = freezeAllCharacterLooks(lookFixture);
assert(
  frozen.looks.every((lk) => lk.look_status === 'final'),
  'character finalize freezes all looks',
);

const series = defaultSeriesState({
  characters: [
    {
      id: 'c1',
      name: 'Hero',
      default_look_id: 'l1',
      looks: [{ id: 'l1', label: 'a', body: 'x' }],
      library_entity_id: 'flc_abc',
      library_version: 1,
    },
  ],
  scenes: [],
});

const updates = listLibraryUpdates(series, {
  characters: [
    {
      entity_id: 'flc_abc',
      version: 3,
      kind: 'character',
      definition: {},
      created_at: '',
    },
  ],
  scenes: [],
});
assert(updates.length === 1 && updates[0].latestVersion === 3, 'detect library update');

let createCalls = 0;
const library = {
  async createCharacter(definition: Record<string, unknown>) {
    createCalls += 1;
    return {
      entity_id: 'flc_new',
      version: 1,
      kind: 'character',
      definition,
      created_at: '',
    } satisfies LibraryEntityVersion;
  },
  async getCharacter() {
    throw new Error('unused');
  },
  async newCharacterVersion() {
    throw new Error('unused');
  },
  async listCharacters() {
    return [];
  },
  async createScene(definition: Record<string, unknown>) {
    return {
      entity_id: 'fls_new',
      version: 1,
      kind: 'scene',
      definition,
      created_at: '',
    } satisfies LibraryEntityVersion;
  },
  async getScene() {
    throw new Error('unused');
  },
  async newSceneVersion() {
    throw new Error('unused');
  },
  async listCharacterVersions() {
    return [];
  },
  async listSceneVersions() {
    return [];
  },
  async listScenes() {
    return [];
  },
};

async function runAsyncTests() {
  const unlinked = defaultSeriesState({
    characters: [
      {
        id: 'c2',
        name: 'New',
        default_look_id: 'l1',
        looks: [{ id: 'l1', label: 'a', body: '' }],
      },
    ],
    scenes: [],
  });
  const first = await migrateSeriesAssetsToLibrary(unlinked, library);
  assert(first.changed && createCalls === 1, 'migrate creates library row');
  const second = await migrateSeriesAssetsToLibrary(first.state, library);
  assert(!second.changed && createCalls === 1, 'migrate idempotent');
}

await runAsyncTests();

console.log('longVideoLibrary.unit: ok');
