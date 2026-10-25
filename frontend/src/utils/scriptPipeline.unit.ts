/**
 * Run: npx tsx frontend/src/utils/scriptPipeline.unit.ts
 */
import {
  chapterAnalysisFromPipeline,
  findingAutoRepairable,
  pipelineFactsApproved,
  pipelineHasBlockers,
  resolveScriptPipeline,
  resumeStageFromPipeline,
  rerunClearsFactsApproval,
} from './scriptPipeline';
import type { ScriptPipeline } from '@/types';

function assert(cond: unknown, msg: string) {
  if (!cond) throw new Error(msg);
}

const base: ScriptPipeline = {
  schema_version: '1.0',
  script_id: 'script_1',
  version_id: 'ver_1',
  source_hash: 'abc',
  workflow: {
    stage: 'gate1',
    status: 'awaiting_review',
    facts_approved: false,
    shots_approved: false,
    awaiting_review: 'facts',
  },
  scene_index: [
    {
      scene_id: 'S001',
      heading: '内景 林夏公寓 夜',
      source_start_line: 1,
      source_end_line: 10,
      source_hash: 'h1',
      location_text: '林夏公寓',
    },
  ],
  story_map: { logline: '钥匙滑落的警告之夜', mood: '悬疑' },
  scene_artifacts: {
    S001: {
      scene_id: 'S001',
      analysis: {
        artifact_id: 'a',
        scene_id: 'S001',
        beats: [
          { beat_id: 'S001_B01', source_event: '暴雨与绷带', beat_type: '状态建立' },
          { beat_id: 'S001_B02', source_event: '陈屿警告', beat_type: '信息释放' },
        ],
      },
    },
  },
  audit_report: {
    artifact_id: 'aud',
    issues: [{ issue_id: '1', severity: 'blocker', problem: 'x', status: 'open' }],
  },
};

assert(!pipelineFactsApproved(base), 'facts not approved');
assert(pipelineHasBlockers(base), 'has blockers');
assert(resolveScriptPipeline({ script_pipeline: base })?.script_id === 'script_1', 'resolve');

const ca = chapterAnalysisFromPipeline(base);
assert(ca.synopsis.includes('钥匙'), 'synopsis from logline');
assert(ca.scene_beats.length === 2, 'beats flattened');
assert(!(ca as { script_pipeline?: unknown }).script_pipeline, 'no nested pipeline copy');

base.workflow.facts_approved = true;
assert(pipelineFactsApproved(base), 'facts approved');

assert(resumeStageFromPipeline(base) == null, 'no resume when ok');
base.workflow.status = 'failed';
base.workflow.error = 'boom';
base.workflow.stage = 'r3';
assert(resumeStageFromPipeline(base) === 'r3', 'resume from failed stage');
assert(rerunClearsFactsApproval('r4'), 'r4 clears facts');
assert(!rerunClearsFactsApproval('r6'), 'r6 keeps facts');
assert(findingAutoRepairable('breakdown.elements'), 'path ok');
assert(!findingAutoRepairable(''), 'empty path not auto');

console.log('scriptPipeline.unit: ok');
