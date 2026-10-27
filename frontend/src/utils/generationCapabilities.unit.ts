/**
 * Lightweight unit checks for generation capability helpers.
 * Run: npx tsx src/utils/generationCapabilities.unit.ts
 */

import {
  assertProviderSupports,
  capReferenceAssetIds,
  chainModeToVideoMode,
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

const modelUnknown = {
  id: 'dashscope-x/wan2.7-image',
} as Record<string, unknown>;

assert(capReferenceAssetIds(['a', 'b', 'c', 'd'], modelWithRefs).ids.length === 3, 'cap to max');
assert(capReferenceAssetIds(['a', 'b', 'c', 'd'], modelWithRefs).capped, 'capped flag');
assert(
  capReferenceAssetIds(['a', 'b'], modelUnknown).ids.length === 2 &&
    !capReferenceAssetIds(['a', 'b'], modelUnknown).capped,
  'unknown max: pass refs through',
);

assert(
  Boolean(validateReferenceRequirement(1, modelNoRefs)),
  'fail when refs required but max is explicitly 0',
);
assert(!validateReferenceRequirement(2, modelWithRefs), 'ok when model supports refs');
assert(
  !validateReferenceRequirement(2, modelUnknown),
  'unknown max: do not invent "unsupported"; allow provider to fail loud',
);

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

assert(chainModeToVideoMode('keyframe_only') === 'i2v', 'chain→i2v');
assert(chainModeToVideoMode('first_last') === 'flf', 'chain→flf');
assert(chainModeToVideoMode('reference_r2v') === 'r2v', 'chain→r2v');
assert(chainModeToVideoMode('last_frame') === 'i2v', 'last_frame→i2v');

console.log('generationCapabilities.unit: ok');
