<template>
  <section class="lv-pipe-workbench">
    <nav class="lv-pipe-rail" aria-label="pipeline stages">
      <div
        v-for="st in STAGE_RAIL"
        :key="st.id"
        class="lv-pipe-rail__item"
        :class="{
          'is-current': st.id === currentStage,
          'is-done': stageRank(st.id) < stageRank(currentStage) || (st.id === 'gate1' && factsApproved) || (st.id === 'gate2' && shotsApproved),
          'is-awaiting': (st.id === 'gate1' && awaiting === 'facts') || (st.id === 'gate2' && awaiting === 'shots'),
        }"
      >
        <button
          type="button"
          class="lv-pipe-rail__label-btn"
          :disabled="busy"
          @click="emit('select-stage', st.id)"
        >
          <span class="lv-pipe-rail__dot" />
          <span class="lv-pipe-rail__label">{{ $tt(st.labelKey) }}</span>
        </button>
        <button
          v-if="canRerunStage(st.id)"
          type="button"
          class="lv-pipe-rail__rerun"
          :disabled="busy"
          :title="$tt('video.longVideoPipeRerunStage')"
          @click.stop="onRerunClick(st.id)"
        >
          ↻
        </button>
      </div>
    </nav>

    <p v-if="failedStage" class="lv-pipe-hint lv-pipe-hint--fail">
      {{ $tt('video.longVideoPipeFailedAt', { stage: failedStage }) }}
      <button type="button" class="lv-pipe-hint__link" :disabled="busy" @click="emit('resume')">
        {{ $tt('video.longVideoPipeResumeFrom', { stage: failedStage }) }}
      </button>
    </p>
    <p v-else-if="awaiting === 'facts' || (currentStage === 'gate1' && !factsApproved)" class="lv-pipe-hint">
      {{ $tt('video.longVideoPipeNeedFacts') }}
    </p>

    <div class="lv-pipe-workbench__main">
      <div class="lv-pipe-scenes">
        <div
          v-for="sc in pipeline?.scene_index ?? []"
          :key="sc.scene_id"
          class="lv-pipe-scene-card"
          :class="{ 'is-selected': sc.scene_id === selectedSceneId }"
          @click="selectedSceneId = sc.scene_id"
        >
          <div class="lv-pipe-scene-card__title">
            <strong>{{ sc.scene_id }}</strong>
            <span class="lv-pipe-scene-card__heading">{{ sc.heading }}</span>
            <em class="lv-pipe-scene-card__status">{{ artifactStatus(sc.scene_id) }}</em>
          </div>
          <div class="lv-pipe-scene-card__meta" v-if="breakdown(sc.scene_id)">
            <span>{{ $tt('video.longVideoPipeCastPresent') }}: {{ (breakdown(sc.scene_id)?.cast_present ?? []).join('、') || '—' }}</span>
            <span>{{ $tt('video.longVideoPipeMentioned') }}: {{ (breakdown(sc.scene_id)?.mentioned_not_present ?? []).join('、') || '—' }}</span>
          </div>
          <div class="lv-pipe-scene-card__beats" v-if="analysis(sc.scene_id)?.beats?.length">
            <span
              v-for="b in analysis(sc.scene_id)!.beats"
              :key="b.beat_id"
              class="lv-pipe-beat-chip"
            >{{ b.beat_type || b.source_event }}</span>
          </div>
        </div>
        <p v-if="!(pipeline?.scene_index?.length)" class="lv-pipe-empty">
          {{ $tt('video.longVideoPipeEmpty') }}
        </p>
      </div>

      <aside class="lv-pipe-detail" v-if="selectedArtifact">
        <header class="lv-pipe-detail__head">
          <h3>{{ selectedSceneId }}</h3>
          <button
            v-if="pipeline?.source_document"
            type="button"
            class="lv-pipe-detail__toggle"
            @click="showSource = !showSource"
          >
            {{ showSource ? $tt('video.longVideoPipeHideSource') : $tt('video.longVideoPipeShowSource') }}
          </button>
        </header>
        <pre v-if="showSource && pipeline?.source_document" class="lv-pipe-source__pre">{{ highlightedSource }}</pre>
        <template v-if="selectedArtifact.breakdown">
          <h4>{{ $tt('video.longVideoPipeBreakdown') }}</h4>
          <ul class="lv-pipe-el-list">
            <li v-for="el in selectedArtifact.breakdown.elements ?? []" :key="el.element_id">
              <div class="lv-pipe-el-row">
                <span class="lv-pipe-el-cat">{{ el.category }}</span>
                <strong>{{ el.name }}</strong>
                <span class="lv-pipe-basis">{{ el.basis }}</span>
              </div>
              <blockquote v-if="el.evidence?.quote">L{{ el.evidence.source_start_line }}: {{ el.evidence.quote }}</blockquote>
            </li>
          </ul>
        </template>
        <template v-if="selectedArtifact.analysis">
          <h4>{{ $tt('video.longVideoPipeAnalysis') }}</h4>
          <p class="lv-pipe-detail__fn">{{ selectedArtifact.analysis.dramatic_function }}</p>
          <ol class="lv-pipe-beat-list">
            <li v-for="b in selectedArtifact.analysis.beats ?? []" :key="b.beat_id">
              {{ b.source_event }}
              <small v-if="b.beat_type">（{{ b.beat_type }}）</small>
            </li>
          </ol>
        </template>
      </aside>
    </div>

    <footer class="lv-pipe-review" v-if="pipeline">
      <div class="lv-pipe-review__cols" v-if="ruleItems.length || reviewItems.length">
        <section class="lv-pipe-review__col" v-if="ruleItems.length">
          <header class="lv-pipe-review__col-head">
            <h4>{{ $tt('video.longVideoPipeRuleFindings') }}</h4>
            <button type="button" class="lv-pipe-review__link" @click="toggleAll('rule')">
              {{ $tt('video.longVideoPipeSelectAll') }}
            </button>
          </header>
          <div
            v-for="iss in ruleItems"
            :key="iss.id"
            class="lv-pipe-issue"
            :class="[`is-${iss.severity}`, { 'is-checked': isSelected(iss.id) }]"
          >
            <label class="lv-pipe-issue__row">
              <input
                type="checkbox"
                :checked="isSelected(iss.id)"
                :disabled="!iss.autoRepairable || busy"
                @change="toggleId(iss.id)"
              />
              <span class="lv-pipe-issue__body">
                <span class="lv-pipe-issue__tags">
                  <strong>{{ iss.severity }}</strong>
                  <em class="lv-pipe-issue__action">{{ actionLabel(iss.action) }}</em>
                  <span v-if="!iss.needsLlm" class="lv-pipe-issue__local">{{ $tt('video.longVideoPipeLocalFix') }}</span>
                  <span v-if="!iss.autoRepairable" class="lv-pipe-issue__manual">{{ $tt('video.longVideoPipeNotAutoRepairable') }}</span>
                </span>
                <span>{{ iss.problem }}</span>
                <span v-if="iss.fieldPath" class="lv-pipe-issue__path">{{ iss.fieldPath }}</span>
                <span v-if="iss.hint" class="lv-pipe-issue__hint">{{ iss.hint }}</span>
              </span>
            </label>
          </div>
        </section>
        <section class="lv-pipe-review__col" v-if="reviewItems.length">
          <header class="lv-pipe-review__col-head">
            <h4>{{ $tt('video.longVideoPipeReviewFindings') }}</h4>
            <button type="button" class="lv-pipe-review__link" :disabled="busy" @click="toggleAll('review')">
              {{ $tt('video.longVideoPipeSelectAll') }}
            </button>
          </header>
          <div
            v-for="iss in reviewItems"
            :key="iss.id"
            class="lv-pipe-issue"
            :class="[`is-${iss.severity}`, { 'is-checked': isSelected(iss.id) }]"
          >
            <label class="lv-pipe-issue__row">
              <input
                type="checkbox"
                :checked="isSelected(iss.id)"
                :disabled="!iss.autoRepairable || busy"
                @change="toggleId(iss.id)"
              />
              <span class="lv-pipe-issue__body">
                <span class="lv-pipe-issue__tags">
                  <strong>{{ iss.severity }}</strong>
                  <em class="lv-pipe-issue__badge">{{ iss.sourceLabel }}</em>
                  <em v-if="iss.action" class="lv-pipe-issue__action">{{ actionLabel(iss.action) }}</em>
                  <span v-if="!iss.autoRepairable" class="lv-pipe-issue__manual">{{ $tt('video.longVideoPipeNotAutoRepairable') }}</span>
                </span>
                <span>{{ iss.problem }}</span>
                <span v-if="iss.fieldPath" class="lv-pipe-issue__path">{{ iss.fieldPath }}</span>
                <span v-if="iss.hint" class="lv-pipe-issue__hint">{{ iss.hint }}</span>
              </span>
            </label>
          </div>
        </section>
      </div>

      <div v-if="ruleItems.length || reviewItems.length" class="lv-pipe-repair">
        <DqInput
          :model-value="userNote"
          type="textarea"
          :rows="2"
          size="sm"
          class="lv-pipe-repair__note"
          :placeholder="$tt('video.longVideoPipeRepairNotePh')"
          :disabled="busy"
          @update:model-value="userNote = String($event ?? '')"
        />
        <div class="lv-pipe-repair__actions">
          <DqButton
            type="default"
            size="sm"
            :disabled="selectedCount === 0 || busy"
            :loading="repairing"
            @click="onRepairPreview"
          >
            {{ $tt('video.longVideoPipeRepairPreview') }}
          </DqButton>
          <span class="lv-pipe-repair__count">
            {{ $tt('video.longVideoPipeSelectedCount', { n: selectedCount }) }}
          </span>
        </div>
      </div>

      <div v-if="previewPatches.length" class="lv-pipe-diff">
        <header class="lv-pipe-diff__head">
          <h4>{{ $tt('video.longVideoPipeRepairDiffTitle') }}</h4>
          <div class="lv-pipe-diff__btns">
            <DqButton type="primary" size="sm" :loading="applying" @click="onApplyRepair">
              {{ $tt('video.longVideoPipeApplyRepair') }}
            </DqButton>
            <DqButton type="text" size="sm" :disabled="applying" @click="discardPreview">
              {{ $tt('video.longVideoPipeDiscardRepair') }}
            </DqButton>
          </div>
        </header>
        <ul class="lv-pipe-diff__list">
          <li v-for="(p, i) in previewPatches" :key="`${p.finding_id}-${i}`">
            <strong>{{ p.field_path || p.finding_id }}</strong>
            <span v-if="p.applied_locally" class="lv-pipe-issue__local">{{ $tt('video.longVideoPipeLocalFix') }}</span>
            <span v-if="p.note" class="lv-pipe-issue__hint">{{ p.note }}</span>
            <pre v-if="p.before != null || p.after != null" class="lv-pipe-diff__pre">{{ formatPatch(p) }}</pre>
          </li>
        </ul>
      </div>

      <div class="lv-pipe-review__actions">
        <DqButton
          v-if="awaiting === 'facts' || (pipeline.workflow.stage === 'gate1' && !factsApproved)"
          type="primary"
          size="sm"
          :loading="approving"
          :disabled="busy"
          @click="emit('approve-facts')"
        >
          {{ $tt('video.longVideoPipeApproveFacts') }}
        </DqButton>
        <DqButton
          v-if="factsApproved && !shotsApproved && (pipeline.shot_list?.length || awaiting === 'shots')"
          type="primary"
          size="sm"
          :loading="approving"
          :disabled="busy"
          @click="emit('approve-shots')"
        >
          {{ $tt('video.longVideoPipeApproveShots') }}
        </DqButton>
        <DqButton
          v-if="factsApproved && !pipeline.shot_list?.length"
          type="default"
          size="sm"
          :loading="generatingShots"
          :disabled="busy"
          @click="emit('generate-shots')"
        >
          {{ $tt('video.longVideoPipeGenerateShots') }}
        </DqButton>
        <span v-if="factsApproved" class="lv-pipe-badge is-ok">{{ $tt('video.longVideoPipeFactsOk') }}</span>
        <span v-if="shotsApproved" class="lv-pipe-badge is-ok">{{ $tt('video.longVideoPipeShotsOk') }}</span>
      </div>
    </footer>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import type { ScriptPipeline, ScriptRepairPatch, ScriptSceneArtifact } from '@/types';
import {
  findingAutoRepairable,
  isRerunnableStage,
  pipelineStageRank,
  resumeStageFromPipeline,
} from '@/utils/scriptPipeline';

const props = defineProps<{
  pipeline: ScriptPipeline | null;
  approving?: boolean;
  generatingShots?: boolean;
  repairing?: boolean;
  applyingRepair?: boolean;
  /** Parse / rerun / expand in flight — blocks dual submit. */
  busy?: boolean;
}>();

const emit = defineEmits<{
  (e: 'approve-facts'): void;
  (e: 'approve-shots'): void;
  (e: 'generate-shots'): void;
  (e: 'select-stage', stage: string): void;
  (e: 'rerun-stage', payload: { stage: string; sceneId?: string }): void;
  (e: 'resume'): void;
  (e: 'repair-preview', payload: { findingIds: string[]; userNote: string }): void;
  (e: 'apply-repair', pipeline: ScriptPipeline): void;
  (e: 'discard-repair'): void;
}>();

const { t: $tt } = useI18n();

const STAGE_RAIL = [
  { id: 'r0', labelKey: 'video.longVideoPipeStageR0' },
  { id: 'r1', labelKey: 'video.longVideoPipeStageR1' },
  { id: 'r2', labelKey: 'video.longVideoPipeStageR2' },
  { id: 'r3', labelKey: 'video.longVideoPipeStageR3' },
  { id: 'r4', labelKey: 'video.longVideoPipeStageR4' },
  { id: 'gate1', labelKey: 'video.longVideoPipeStageGate1' },
  { id: 'r5', labelKey: 'video.longVideoPipeStageR5' },
  { id: 'r6', labelKey: 'video.longVideoPipeStageR6' },
  { id: 'r7', labelKey: 'video.longVideoPipeStageR7' },
  { id: 'gate2', labelKey: 'video.longVideoPipeStageGate2' },
] as const;

function stageRank(id: string) {
  return pipelineStageRank(id);
}

const busy = computed(() => Boolean(
  props.busy || props.approving || props.generatingShots || props.repairing || props.applyingRepair,
));

const selectedSceneId = ref('');
const showSource = ref(false);
watch(
  () => props.pipeline?.scene_index,
  (idx) => {
    if (!selectedSceneId.value && idx?.[0]) selectedSceneId.value = idx[0].scene_id;
  },
  { immediate: true },
);

const currentStage = computed(() => props.pipeline?.workflow?.stage || 'r0');
const factsApproved = computed(() => Boolean(props.pipeline?.workflow?.facts_approved));
const shotsApproved = computed(() => Boolean(props.pipeline?.workflow?.shots_approved));
const awaiting = computed(() => props.pipeline?.workflow?.awaiting_review || '');
const failedStage = computed(() => resumeStageFromPipeline(props.pipeline));

function canRerunStage(id: string) {
  if (!isRerunnableStage(id) || !props.pipeline) return false;
  // Allow rerun for completed stages and the current/failed stage.
  return stageRank(id) <= stageRank(currentStage.value) || id === failedStage.value;
}

function onRerunClick(stage: string) {
  if (busy.value || !canRerunStage(stage)) return;
  // Full-stage rerun (scene-scoped only_scene_ids left for a dedicated control).
  emit('rerun-stage', { stage });
}

function breakdown(id: string) {
  return props.pipeline?.scene_artifacts?.[id]?.breakdown;
}
function analysis(id: string) {
  return props.pipeline?.scene_artifacts?.[id]?.analysis;
}
function artifactStatus(id: string) {
  return props.pipeline?.scene_artifacts?.[id]?.review_status || 'pending';
}

const selectedArtifact = computed<ScriptSceneArtifact | null>(() => {
  const id = selectedSceneId.value;
  if (!id) return null;
  return props.pipeline?.scene_artifacts?.[id] ?? null;
});

const highlightedSource = computed(() => {
  const doc = props.pipeline?.source_document;
  if (!doc) return '';
  const sel = props.pipeline?.scene_index?.find((s) => s.scene_id === selectedSceneId.value);
  if (!sel) return doc.numbered_text;
  return doc.lines
    .map((ln, i) => {
      const n = i + 1;
      const mark = n >= sel.source_start_line && n <= sel.source_end_line ? '>' : ' ';
      return `${mark}${n}|${ln}`;
    })
    .join('\n');
});

type FindingRow = {
  id: string;
  kind: 'rule' | 'review';
  sourceLabel: string;
  severity: string;
  action: string;
  fieldPath?: string;
  problem: string;
  hint?: string;
  needsLlm: boolean;
  autoRepairable: boolean;
  defaultChecked: boolean;
};

const selectedIdList = ref<string[]>([]);
const userNote = ref('');
const previewPatches = ref<ScriptRepairPatch[]>([]);
const previewPipeline = ref<ScriptPipeline | null>(null);
const repairing = computed(() => Boolean(props.repairing));
const applying = computed(() => Boolean(props.applyingRepair));
const selectedCount = computed(() => selectedIdList.value.length);

function isSelected(id: string) {
  return selectedIdList.value.includes(id);
}

const ruleItems = computed<FindingRow[]>(() => {
  const out: FindingRow[] = [];
  for (const f of props.pipeline?.rule_findings ?? []) {
    if (f.status === 'resolved') continue;
    const auto = findingAutoRepairable(f.field_path);
    out.push({
      id: f.finding_id,
      kind: 'rule',
      sourceLabel: 'rule',
      severity: f.severity || 'info',
      action: f.action || 'fill_field',
      fieldPath: f.field_path,
      problem: f.problem,
      hint: f.repair_instruction,
      needsLlm: Boolean(f.needs_llm),
      autoRepairable: auto,
      defaultChecked: auto && (f.severity === 'blocker' || f.needs_llm === false),
    });
  }
  return out.slice(0, 24);
});

const reviewItems = computed<FindingRow[]>(() => {
  const out: FindingRow[] = [];
  for (const iss of props.pipeline?.continuity_report?.issues ?? []) {
    if (iss.status === 'resolved') continue;
    const auto = findingAutoRepairable(iss.field_path);
    out.push({
      id: iss.issue_id,
      kind: 'review',
      sourceLabel: 'R4',
      severity: iss.severity || 'info',
      action: iss.action || 'fill_field',
      fieldPath: iss.field_path,
      problem: iss.problem,
      hint: iss.repair_instruction,
      needsLlm: true,
      autoRepairable: auto,
      defaultChecked: auto && (iss.severity === 'blocker' || iss.severity === 'major'),
    });
  }
  for (const iss of props.pipeline?.audit_report?.issues ?? []) {
    if (iss.status === 'resolved') continue;
    const auto = findingAutoRepairable(iss.field_path);
    out.push({
      id: iss.issue_id,
      kind: 'review',
      sourceLabel: 'R7',
      severity: iss.severity || 'info',
      action: iss.action || 'fill_field',
      fieldPath: iss.field_path,
      problem: iss.problem,
      hint: iss.repair_instruction,
      needsLlm: true,
      autoRepairable: auto,
      defaultChecked: auto && (iss.severity === 'blocker' || iss.severity === 'major'),
    });
  }
  return out.slice(0, 24);
});

watch(
  () => [ruleItems.value, reviewItems.value] as const,
  ([rules, reviews]) => {
    selectedIdList.value = [...rules, ...reviews]
      .filter((r) => r.defaultChecked && r.autoRepairable)
      .map((r) => r.id);
    previewPatches.value = [];
    previewPipeline.value = null;
  },
  { immediate: true },
);

function toggleId(id: string) {
  const row = [...ruleItems.value, ...reviewItems.value].find((r) => r.id === id);
  if (!row?.autoRepairable) return;
  if (selectedIdList.value.includes(id)) {
    selectedIdList.value = selectedIdList.value.filter((x) => x !== id);
  } else {
    selectedIdList.value = [...selectedIdList.value, id];
  }
}

function toggleAll(kind: 'rule' | 'review') {
  const rows = (kind === 'rule' ? ruleItems.value : reviewItems.value).filter((r) => r.autoRepairable);
  const ids = new Set(selectedIdList.value);
  const allOn = rows.length > 0 && rows.every((r) => ids.has(r.id));
  if (allOn) {
    for (const r of rows) ids.delete(r.id);
  } else {
    for (const r of rows) ids.add(r.id);
  }
  selectedIdList.value = Array.from(ids);
}

function actionLabel(action: string) {
  const key = `video.longVideoPipeAction_${action}`;
  const label = $tt(key);
  return label === key ? action : label;
}

function onRepairPreview() {
  emit('repair-preview', {
    findingIds: [...selectedIdList.value],
    userNote: userNote.value.trim(),
  });
}

function onApplyRepair() {
  if (!previewPipeline.value) return;
  emit('apply-repair', previewPipeline.value);
}

function discardPreview() {
  previewPatches.value = [];
  previewPipeline.value = null;
  emit('discard-repair');
}

/** Parent calls after repair-preview API returns. */
function setRepairPreview(patches: ScriptRepairPatch[], pipe: ScriptPipeline) {
  previewPatches.value = patches ?? [];
  previewPipeline.value = pipe;
}

function formatPatch(p: ScriptRepairPatch) {
  const before = p.before == null ? '—' : JSON.stringify(p.before, null, 0);
  const after = p.after == null ? '—' : JSON.stringify(p.after, null, 0);
  return `${before}\n→ ${after}`;
}

defineExpose({ setRepairPreview, discardPreview });
</script>

<style scoped>
.lv-pipe-workbench {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
  flex: 1;
  color: var(--dq-label-primary);
}

.lv-pipe-rail {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.lv-pipe-rail__item {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  color: var(--dq-label-tertiary);
  font-size: 11px;
  padding: 2px 4px 2px 7px;
  border-radius: 6px;
}

.lv-pipe-rail__label-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
  padding: 1px 0;
}

.lv-pipe-rail__label-btn:disabled {
  cursor: default;
  opacity: 0.7;
}

.lv-pipe-rail__item.is-current {
  color: var(--dq-label-primary);
  background: color-mix(in srgb, var(--dq-accent) 18%, transparent);
}

.lv-pipe-rail__item.is-done {
  color: var(--dq-label-secondary);
}

.lv-pipe-rail__item.is-awaiting {
  color: var(--dq-label-primary);
  box-shadow: inset 0 0 0 1px var(--dq-accent);
}

.lv-pipe-rail__dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
  flex-shrink: 0;
}

.lv-pipe-rail__rerun {
  border: 0;
  background: transparent;
  color: inherit;
  opacity: 0.55;
  cursor: pointer;
  font-size: 12px;
  line-height: 1;
  padding: 2px 4px;
  border-radius: 4px;
}

.lv-pipe-rail__rerun:hover:not(:disabled) {
  opacity: 1;
  background: color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
}

.lv-pipe-rail__rerun:disabled {
  cursor: default;
  opacity: 0.35;
}

.lv-pipe-hint {
  margin: 0;
  font-size: 12px;
  line-height: 1.4;
  color: var(--dq-label-secondary);
  padding: 8px 10px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--dq-accent) 12%, var(--dq-surface-elevated, transparent));
  border: 0.5px solid color-mix(in srgb, var(--dq-accent) 35%, transparent);
}

.lv-pipe-hint--fail {
  background: color-mix(in srgb, var(--dq-danger, #c44) 12%, var(--dq-surface-elevated, transparent));
  border-color: color-mix(in srgb, var(--dq-danger, #c44) 35%, transparent);
}

.lv-pipe-hint__link {
  border: 0;
  background: transparent;
  color: var(--dq-accent);
  font: inherit;
  font-weight: 600;
  cursor: pointer;
  margin-left: 8px;
  text-decoration: underline;
}

.lv-pipe-hint__link:disabled {
  opacity: 0.5;
  cursor: default;
}

.lv-pipe-workbench__main {
  display: grid;
  grid-template-columns: minmax(140px, 0.9fr) minmax(180px, 1.1fr);
  gap: 8px;
  min-height: 0;
  flex: 1;
}

.lv-pipe-scenes,
.lv-pipe-detail {
  border: 0.5px solid var(--dq-border-subtle);
  border-radius: 8px;
  padding: 10px;
  overflow: auto;
  max-height: min(520px, 55vh);
  background: var(--dq-surface-elevated, var(--dq-surface));
  color: var(--dq-label-primary);
}

.lv-pipe-scene-card {
  padding: 8px;
  border-radius: 6px;
  cursor: pointer;
  margin-bottom: 4px;
}

.lv-pipe-scene-card:hover,
.lv-pipe-scene-card.is-selected {
  background: color-mix(in srgb, var(--dq-accent) 14%, transparent);
}

.lv-pipe-scene-card__title {
  display: flex;
  gap: 6px;
  align-items: baseline;
  font-size: 12px;
  color: var(--dq-label-primary);
}

.lv-pipe-scene-card__heading {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.lv-pipe-scene-card__status {
  margin-left: auto;
  font-style: normal;
  font-size: 10px;
  color: var(--dq-label-tertiary);
  flex-shrink: 0;
}

.lv-pipe-scene-card__meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 11px;
  color: var(--dq-label-secondary);
  margin-top: 4px;
}

.lv-pipe-beat-chip {
  display: inline-block;
  font-size: 10px;
  padding: 1px 6px;
  margin: 2px 4px 0 0;
  border-radius: 999px;
  color: var(--dq-label-secondary);
  background: color-mix(in srgb, var(--dq-label-primary) 10%, transparent);
}

.lv-pipe-detail__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
}

.lv-pipe-detail h3 {
  margin: 0;
  font-size: 13px;
  color: var(--dq-label-primary);
}

.lv-pipe-detail h4 {
  margin: 12px 0 6px;
  font-size: 11px;
  letter-spacing: 0.04em;
  color: var(--dq-label-tertiary);
  text-transform: uppercase;
}

.lv-pipe-detail__toggle {
  border: 0;
  background: transparent;
  color: var(--dq-accent);
  font-size: 11px;
  cursor: pointer;
  padding: 0;
}

.lv-pipe-detail__fn {
  margin: 0;
  font-size: 12px;
  line-height: 1.45;
  color: var(--dq-label-secondary);
}

.lv-pipe-source__pre {
  margin: 0 0 10px;
  padding: 8px;
  font-size: 11px;
  line-height: 1.45;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  color: var(--dq-label-secondary);
  background: color-mix(in srgb, var(--dq-label-primary) 6%, transparent);
  border-radius: 6px;
  max-height: 160px;
  overflow: auto;
}

.lv-pipe-el-list,
.lv-pipe-beat-list {
  margin: 0;
  padding-left: 16px;
  font-size: 12px;
  color: var(--dq-label-primary);
}

.lv-pipe-el-row {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 6px;
}

.lv-pipe-el-cat {
  font-size: 10px;
  color: var(--dq-label-tertiary);
  text-transform: uppercase;
}

.lv-pipe-el-list blockquote {
  margin: 4px 0 8px;
  padding-left: 8px;
  border-left: 2px solid var(--dq-border-subtle);
  color: var(--dq-label-secondary);
  font-size: 11px;
}

.lv-pipe-basis {
  color: var(--dq-label-tertiary);
  font-size: 10px;
}

.lv-pipe-review {
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-top: 0.5px solid var(--dq-border-subtle);
  padding-top: 10px;
}

.lv-pipe-review__cols {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.lv-pipe-review__col {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.lv-pipe-review__col-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.lv-pipe-review__col-head h4 {
  margin: 0;
  font-size: 12px;
  font-weight: 650;
  color: var(--dq-label-primary);
}

.lv-pipe-review__link {
  border: 0;
  background: transparent;
  color: var(--dq-accent);
  font-size: 11px;
  cursor: pointer;
  padding: 0;
}

.lv-pipe-review__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.lv-pipe-issue {
  font-size: 12px;
  padding: 6px 8px;
  border-radius: 6px;
  color: var(--dq-label-primary);
  background: color-mix(in srgb, #f59e0b 16%, transparent);
  border: 0.5px solid transparent;
}

.lv-pipe-issue.is-checked {
  border-color: color-mix(in srgb, var(--dq-accent) 45%, transparent);
}

.lv-pipe-issue.is-blocker,
.lv-pipe-issue.is-major {
  background: color-mix(in srgb, #ef4444 18%, transparent);
}

.lv-pipe-issue.is-info {
  background: color-mix(in srgb, var(--dq-fill-control) 55%, transparent);
}

.lv-pipe-issue__row {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  cursor: pointer;
}

.lv-pipe-issue__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.lv-pipe-issue__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.lv-pipe-issue__action,
.lv-pipe-issue__badge,
.lv-pipe-issue__local,
.lv-pipe-issue__manual {
  font-style: normal;
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--dq-accent) 16%, transparent);
  color: var(--dq-label-secondary);
}

.lv-pipe-issue__local {
  background: color-mix(in srgb, #22c55e 18%, transparent);
}

.lv-pipe-issue__manual {
  background: color-mix(in srgb, var(--dq-label-tertiary) 22%, transparent);
}

.lv-pipe-issue__path {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 10px;
  color: var(--dq-label-tertiary);
  word-break: break-all;
}

.lv-pipe-issue__hint {
  display: block;
  opacity: 0.8;
  margin-top: 2px;
  color: var(--dq-label-secondary);
}

.lv-pipe-repair {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.lv-pipe-repair__actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.lv-pipe-repair__count {
  font-size: 11px;
  color: var(--dq-label-tertiary);
}

.lv-pipe-diff {
  border: 0.5px solid var(--dq-border-subtle);
  border-radius: 8px;
  padding: 10px;
  background: color-mix(in srgb, var(--dq-surface-elevated) 50%, transparent);
}

.lv-pipe-diff__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.lv-pipe-diff__head h4 {
  margin: 0;
  font-size: 12px;
}

.lv-pipe-diff__btns {
  display: flex;
  gap: 6px;
}

.lv-pipe-diff__list {
  margin: 0;
  padding-left: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 12px;
}

.lv-pipe-diff__pre {
  margin: 4px 0 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 11px;
  color: var(--dq-label-secondary);
}

.lv-pipe-badge.is-ok {
  font-size: 12px;
  color: #4ade80;
}

@media (max-width: 1100px) {
  .lv-pipe-review__cols {
    grid-template-columns: 1fr;
  }
}

.lv-pipe-empty {
  font-size: 13px;
  color: var(--dq-label-tertiary);
  margin: 8px 0;
}

@media (max-width: 1100px) {
  .lv-pipe-workbench__main {
    grid-template-columns: 1fr;
  }
}
</style>
