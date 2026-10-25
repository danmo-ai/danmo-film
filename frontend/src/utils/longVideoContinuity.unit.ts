/**
 * Run: npx tsx frontend/src/utils/longVideoContinuity.unit.ts
 */
import { continuityIssues, crossEpisodeConsistencyReport } from './longVideoContinuity';
import { defaultSeriesState } from './longVideoSeries';
import type { LongVideoProjectState } from '@/types';

function assert(cond: unknown, msg: string) {
  if (!cond) throw new Error(msg);
}

const series = defaultSeriesState({
  characters: [{ id: 'c1', name: 'A', default_look_id: 'l1', looks: [{ id: 'l1', label: 'd', body: '' }] }],
  relations: [{ from_id: 'c1', to_id: 'missing', label: 'friend' }],
  props: [{ id: 'prop_sword', name: 'Sword' }],
});

const ep: Pick<
  LongVideoProjectState,
  'project_id' | 'episode_number' | 'characters' | 'scenes' | 'character_variants' | 'scene_variants' | 'series_id' | 'shots'
> = {
  project_id: 'prj_1',
  episode_number: 2,
  series_id: 'lvs_1',
  characters: [],
  scenes: [],
  character_variants: {},
  scene_variants: {},
  shots: [
    {
      id: 's1',
      order: 0,
      visual_prompt: 'prop_sword gleams',
      motion_prompt: 'pan',
      segment_role: 'keyframe',
      start_frame_mode: 'keyframe',
    },
  ],
};

const issues = continuityIssues(series, {}, ep);
assert(issues.some((i) => i.code === 'relation_missing_endpoint'), 'relation endpoint');
assert(issues.some((i) => i.code === 'prop_missing_reference'), 'prop ref');
assert(issues.some((i) => i.code === 'episode_missing_prev_summary'), 'prev summary');

const report = crossEpisodeConsistencyReport(series, [
  { project_id: 'e1', style_anchor: 'noir', series_id: 'lvs_1' },
  { project_id: 'e2', style_anchor: 'bright', series_id: 'lvs_1' },
]);
assert(report.styleAnchorMismatch, 'style mismatch');

console.log('longVideoContinuity.unit: ok');
