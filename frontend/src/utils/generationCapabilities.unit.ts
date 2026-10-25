/**
 * Lightweight unit checks for generation capability helpers.
 * Run: npx tsx src/utils/generationCapabilities.unit.ts
 */

import {
  assertProviderSupports,
  capReferenceAssetIds,
  validateReferenceRequirement,
} from './generationCapabilities';

function assert(cond: unknown, msg: string): asserts cond {
  if (!cond) throw new Error(msg);
}

const modelWithRefs = {
  id: 'm1',
  parameters: { max_reference_images: 3, supports_r2v: true, supports_first_last: true },
} as Record<string, unknown>;

const modelNoRefs = {
  id: 'm2',
  parameters: { max_reference_images: 0 },
} as Record<string, unknown>;

assert(capReferenceAssetIds(['a', 'b', 'c', 'd'], modelWithRefs).ids.length === 3, 'cap to max');
assert(capReferenceAssetIds(['a', 'b', 'c', 'd'], modelWithRefs).capped, 'capped flag');

assert(
  Boolean(validateReferenceRequirement(1, modelNoRefs)),
  'fail when refs required but max is 0',
);
assert(!validateReferenceRequirement(2, modelWithRefs), 'ok when model supports refs');

assert(
  Boolean(
    assertProviderSupports('video_first_last', {
      id: 'm3',
      parameters: { supports_first_last: false, supports_flf: false },
    } as Record<string, unknown>),
  ),
  'fail loud when first_last unsupported',
);

assert(!assertProviderSupports('video_reference_r2v', modelWithRefs), 'r2v allowed');

console.log('generationCapabilities.unit: ok');
