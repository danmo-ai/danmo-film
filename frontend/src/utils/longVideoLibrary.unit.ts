/**
 * Run: npx tsx frontend/src/utils/longVideoLibrary.unit.ts
 */
import {
  listLibraryUpdates,
  migrateSeriesAssetsToLibrary,
} from './longVideoLibrary';
import { defaultSeriesState } from './longVideoSeries';
import type { LibraryEntityVersion } from '@/types';

function assert(cond: unknown, msg: string) {
  if (!cond) throw new Error(msg);
}

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
