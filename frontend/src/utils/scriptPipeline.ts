import type { LongVideoChapterAnalysis, ScriptPipeline } from '@/types';

/** Sole runtime SoR for multi-stage parse. */
export function resolveScriptPipeline(
  project: { script_pipeline?: ScriptPipeline },
): ScriptPipeline | null {
  return project.script_pipeline ?? null;
}

export function pipelineFactsApproved(pipe: ScriptPipeline | null | undefined): boolean {
  return Boolean(pipe?.workflow?.facts_approved);
}

export function pipelineShotsApproved(pipe: ScriptPipeline | null | undefined): boolean {
  return Boolean(pipe?.workflow?.shots_approved);
}

export function pipelineHasBlockers(pipe: ScriptPipeline | null | undefined): boolean {
  const issues = [
    ...(pipe?.continuity_report?.issues ?? []),
    ...(pipe?.audit_report?.issues ?? []),
  ];
  return issues.some((i) => i.severity === 'blocker' && i.status !== 'resolved');
}

/** Summary-only chapter_analysis fields derived from pipeline (no nested pipeline copy). */
export function chapterAnalysisFromPipeline(
  pipe: ScriptPipeline,
  prev?: LongVideoChapterAnalysis | null,
): LongVideoChapterAnalysis {
  const scene_beats = (pipe.scene_index ?? []).flatMap((sc, order) => {
    const art = pipe.scene_artifacts?.[sc.scene_id];
    const beats = art?.analysis?.beats;
    if (!beats?.length) {
      return [{
        order: order + 1,
        title: sc.heading,
        beat: sc.synopsis || sc.heading,
      }];
    }
    return beats.map((b, i) => ({
      order: order * 10 + i + 1,
      title: b.beat_type || sc.heading,
      beat: `${sc.location_text || sc.heading} — ${b.source_event}`,
    }));
  });
  return {
    synopsis: pipe.story_map?.logline || prev?.synopsis || '',
    mood: pipe.story_map?.mood ?? prev?.mood,
    style_anchor: pipe.story_map?.style_anchor ?? prev?.style_anchor,
    scene_beats,
    parse_run_id: pipe.parse_run_id ?? prev?.parse_run_id,
    last_parse_at: new Date().toISOString(),
    parse_blocked: false,
  };
}

/** Next stage to resume after a failed/partial run. */
export function resumeStageFromPipeline(pipe: ScriptPipeline | null | undefined): string | null {
  if (!pipe?.workflow) return null;
  if (pipe.workflow.status !== 'failed' && !pipe.workflow.error) return null;
  const stage = pipe.workflow.stage || 'r1';
  return stage;
}

const STAGE_ORDER: Record<string, number> = {
  r0: 0, r1: 1, r2: 2, r3: 3, r4: 4, gate1: 5, r5: 6, r6: 7, r7: 8, gate2: 9, completed: 10,
};

export function pipelineStageRank(id: string): number {
  return STAGE_ORDER[id] ?? -1;
}

/** Stages that can be re-run via /rerun/:stage. */
export function isRerunnableStage(stage: string): boolean {
  return ['r1', 'r2', 'r3', 'r4', 'r5', 'r6', 'r7'].includes(stage);
}

/** Rerun of this stage clears facts_approved (backend invalidateDownstreamFrom). */
export function rerunClearsFactsApproval(stage: string): boolean {
  return stage === 'r2' || stage === 'r3' || stage === 'r4';
}

/** Rerun of this stage clears shots_approved. */
export function rerunClearsShotsApproval(stage: string): boolean {
  return stage === 'r2' || stage === 'r3' || stage === 'r4' || stage === 'r5';
}

export function findingAutoRepairable(fieldPath?: string | null): boolean {
  return Boolean(fieldPath?.trim());
}
