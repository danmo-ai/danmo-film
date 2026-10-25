<template>
  <div class="copilot-page long-video-page long-video-page--episode">
    <div class="copilot-page__main">
      <div class="lv-editor-workspace-shell copilot-page__workspace-card">
        <div class="lv-editor-workspace">
          <div class="lv-editor-tabs-wrapper">
            <LongVideoEditorTabs
              variant="rail"
              :model-value="editorTab"
              :cast-count="effectiveCharacters.length"
              :scene-entity-count="effectiveScenes.length"
              :shot-count="shots.length"
              :settings-done="settingsStepDone"
              :script-done="scriptStepDone"
              :cast-done="castStepDone"
              :scenes-done="scenesStepDone"
              :storyboard-done="storyboardStepDone"
              :delivery-done="deliveryStepDone"
              :storyboard-disabled="storyboardTabBlocked"
              :storyboard-disabled-reason="storyboardTabBlockedReason"
              @update:model-value="onEditorTabChange"
              @storyboard-blocked="onStoryboardTabBlocked"
            />
          </div>

          <div
            v-if="productionBlockerLabels.length && editorTab !== 'delivery'"
            class="lv-production-blockers"
            role="status"
          >
            <strong>{{
              $tt('video.longVideoProductionBlockersTitle', { n: productionBlockerLabels.length })
            }}</strong>
            <ul>
              <li v-for="(label, i) in productionBlockerLabels" :key="i">{{ label }}</li>
            </ul>
          </div>

          <div class="lv-editor-pane-host">
        <div v-show="editorTab === 'settings'" class="lv-editor-pane lv-editor-pane--settings">
          <div class="lv-ep-sheet">
          <LongVideoSettingsPanel
            mode="episode"
            :title="projectTitle"
            :keyframe-model="project?.keyframe_model ?? defaultKeyframeModel"
            :segment-model="project?.segment_video_model ?? defaultSegmentModel"
            :output-size="projectOutputSize"
            :output-size-options="segmentResolutionOptions"
            :keyframe-model-options="keyframeModelOptions"
            :segment-model-options="segmentModelOptions"
            :model-label="modelLabel"
            :overlap-frames="effectiveOverlapFrames"
            :target-duration-sec="project?.target_duration_sec ?? 60"
            :locked-output-size="projectOutputSize"
            :locked-overlap-frames="effectiveOverlapFrames"
            :can-open-project-settings="Boolean(project?.series_id)"
            @update:title="patchProjectField('title', $event)"
            @update:keyframe-model="onKeyframeModelChange"
            @update:segment-model="onSegmentModelChange"
            @update:target-duration-sec="onTargetDurationChange"
            @open-project-settings="openProjectSettings"
          />
          </div>
          <LongVideoStepSaveBar
            :save-label="$tt('video.longVideoSaveStepSettings')"
            :hint="$tt('video.longVideoSaveStepSettingsHint')"
            :status="autoSaveStatus"
            :status-label="autoSaveLabel"
            :loading="savingProject"
            :next-label="$tt('video.longVideoStepNextScript')"
            @save="saveProject"
            @next="onEditorTabChange('script')"
          />
        </div>

        <div v-show="editorTab === 'script'" class="lv-editor-pane lv-editor-pane--script">
          <LongVideoBriefPanel
            ref="briefPanelRef"
            :script-text="scriptText"
            :chapter-title="project?.chapter_title ?? ''"
            :chapter-analysis="project?.chapter_analysis ?? null"
            :characters="effectiveCharacters"
            :script-parsed="scriptStepDone"
            :style-anchor="effectiveStyleAnchor"
            :target-duration-sec="project?.target_duration_sec ?? 60"
            :segment-duration-sec="project?.segment_duration_sec ?? 5"
            :script-parse-llm-model="project?.script_parse_llm_model ?? ''"
            :parsing="isChapterAnalyzing"
            :parse-progress-phase="scriptParseProgressPhase"
            :parse-progress-message="scriptParseProgressMessage"
            :beat-regenerating-index="beatRegeneratingIndex"
            :expanding="isScriptExpanding"
            :generating-storyboard="isStoryboardGenerating"
            :parse-error="scriptParseError"
            :parse-blocked="scriptParseHardBlocked"
            :parse-blocked-reason="scriptParseHardBlockedReason"
            :project-id="project?.project_id ?? ''"
            :parsed-shot-count="shots.length"
            :script-pipeline="activeScriptPipeline"
            :pipeline-approving="pipelineApproving"
            :pipeline-repairing="pipelineRepairing"
            :pipeline-applying-repair="pipelineApplyingRepair"
            @update:script-text="onScriptTextChange"
            @update:chapter-title="patchProjectField('chapter_title', $event)"
            @update:chapter-analysis="onChapterAnalysisChange"
            @update:target-duration-sec="onTargetDurationChange"
            @update:segment-duration-sec="onSegmentDurationChange"
            @update:script-parse-llm-model="patchProjectField('script_parse_llm_model', $event)"
            @update:style-anchor="onUpdateStyleAnchor"
            @expand="onScriptExpand"
            @parse="onScriptParse"
            @generate-storyboard="onGenerateStoryboard"
            @regenerate-beat="onRegenerateBeat"
            @clear-parse-blocked="clearParseBlocked"
            @approve-facts="onApprovePipelineFacts"
            @approve-shots="onApprovePipelineShots"
            @generate-shots="onGeneratePipelineShots"
            @resume-pipeline="onResumePipeline"
            @rerun-stage="onRerunPipelineStage"
            @repair-preview="onPipelineRepairPreview"
            @apply-repair="onPipelineApplyRepair"
            @discard-repair="onPipelineDiscardRepair"
          />
          <LongVideoStepSaveBar
            :save-label="$tt('video.longVideoSaveStepScript')"
            :hint="$tt('video.longVideoSaveStepScriptHint')"
            :status="autoSaveStatus"
            :status-label="autoSaveLabel"
            :loading="savingProject"
            :next-label="$tt('video.longVideoStepNextCast')"
            :next-disabled="!scriptStepDone"
            @save="saveProject"
            @next="onGoToCastFromScript"
          />
        </div>

        <div v-show="editorTab === 'cast'" class="lv-editor-pane lv-editor-pane--cast">
          <LongVideoCastPanel
            :characters="effectiveCharacters"
            :character-anchor="project?.character_anchor ?? ''"
            :script-synopsis="projectSynopsis"
            :script-parsed="scriptStepDone"
            :project-id="project?.project_id ?? ''"
            :series-linked="Boolean(project?.series_id)"
            :can-restore-from-series="castHasEpisodeOverride"
            :promotable-character-ids="promotableCharacterIds"
            :promotable-look-ids="promotableCharacterLookIds"
            :episode-only-character-ids="episodeOnlyCharacterIds"
            :compatible-loras="keyframeCompatibleLoras"
            :portrait-generating-key="portraitGeneratingKey"
            :vision-backfill-key="visionBackfillKey"
            :batch-generating="batchPortraitGenerating"
            @restore-series-master="onRestoreCharacterFromSeries"
            @promote-to-series="onPromoteCharacterToSeries"
            @promote-look-to-series="onPromoteCharacterLookToSeries"
            @update:characters="onUpdateCharacters"
            @generate-portrait="onGeneratePortrait"
            @pick-portrait-gallery="onPickPortraitGallery"
            @clear-portrait="onClearPortrait"
            @vision-backfill="onVisionBackfillLook"
            @batch-generate-portraits="onBatchGeneratePortraits"
            @import-style-anchor="onUpdateStyleAnchor"
            @go-to-script="onEditorTabChange('script')"
          />
          <LongVideoStepSaveBar
            :save-label="$tt('video.longVideoSaveStepCast')"
            :hint="$tt('video.longVideoSaveStepCastHint')"
            :status="autoSaveStatus"
            :status-label="autoSaveLabel"
            :loading="savingProject"
            :next-label="$tt('video.longVideoStepNextScenes')"
            @save="saveProject"
            @next="onEditorTabChange('scenes')"
          />
        </div>

        <div v-show="editorTab === 'scenes'" class="lv-editor-pane lv-editor-pane--scenes">
          <LongVideoScenePanel
            :scenes="effectiveScenes"
            :script-parsed="scriptStepDone"
            :project-id="project?.project_id ?? ''"
            :series-linked="Boolean(project?.series_id)"
            :can-restore-from-series="sceneHasEpisodeOverrideFlag"
            :promotable-scene-ids="promotableSceneIds"
            :promotable-look-ids="promotableSceneLookIds"
            :episode-only-scene-ids="episodeOnlySceneIds"
            :ref-generating-key="sceneRefGeneratingKey"
            :vision-backfill-key="sceneVisionBackfillKey"
            :batch-generating="batchSceneRefGenerating"
            @restore-series-master="onRestoreSceneFromSeries"
            @promote-to-series="onPromoteSceneToSeries"
            @promote-look-to-series="onPromoteSceneLookToSeries"
            @update:scenes="onUpdateScenes"
            @generate-ref="onGenerateSceneRef"
            @pick-ref-gallery="onPickSceneRefGallery"
            @clear-ref="onClearSceneRef"
            @vision-backfill="onVisionBackfillSceneLook"
            @batch-generate-refs="onBatchGenerateSceneRefs"
            @go-to-script="onEditorTabChange('script')"
          />
          <LongVideoStepSaveBar
            :save-label="$tt('video.longVideoSaveStepScenes')"
            :hint="$tt('video.longVideoSaveStepScenesHint')"
            :status="autoSaveStatus"
            :status-label="autoSaveLabel"
            :loading="savingProject"
            :next-label="$tt('video.longVideoStepNextStoryboard')"
            @save="saveProject"
            @next="onCastGoToStoryboard"
          />
        </div>

        <div v-show="editorTab === 'storyboard'" class="lv-editor-pane lv-editor-pane--storyboard">
          <LongVideoGroupToolbar
            :shots="shots"
            :selection="selection"
            :chain-mode="project?.chain_mode ?? 'keyframe_only'"
            :generating="batchGroupGenerating"
            :batch-blocked="batchGenerateBlocked"
            @generate-group="onBatchGenerateCurrentGroup"
            @generate-all-anchors="onBatchGenerateAllAnchors"
            @generate-all-segments="onBatchGenerateAllSegments"
          />
          <LongVideoGroupProgressBar
            :shots="shots"
            @select-group="onSelectBeatGroup"
          />
          <LongVideoBeatGroupRail
            :shots="shots"
            :selection="selection"
            :keyframe-generating-index="keyframeGeneratingIndex"
            :segment-generating-indices="segmentGeneratingIndicesList"
            :output-width="outputSizePixels.width"
            :output-height="outputSizePixels.height"
            :beat-regenerating-index="beatRegeneratingIndex"
            :parse-loading="isChapterAnalyzing || isStoryboardGenerating"
            @select-segment="onSelectSegment"
            @insert-anchor="onInsertFaceAnchor"
            @resplit-beat="onResplitBeatGroup"
            @regenerate-beat="onRegenerateBeatFromGroup"
          />
          <DqAlert
            v-if="storyboardTabBlocked && !shots.length"
            type="error"
            :closable="false"
            class="long-video-page__parse-blocked"
            :title="storyboardTabBlockedReason || $tt('video.longVideoStoryboardTabBlocked')"
          >
            <DqButton size="sm" type="primary" @click="onEditorTabChange('script')">
              {{ $tt('video.longVideoParseBlockedGoScript') }}
            </DqButton>
          </DqAlert>
          <LongVideoStepSaveBar
            :save-label="$tt('video.longVideoSaveStepStoryboard')"
            :hint="$tt('video.longVideoSaveStepStoryboardHint')"
            :status="autoSaveStatus"
            :status-label="autoSaveLabel"
            :loading="savingProject"
            :next-label="$tt('video.longVideoStepNextDelivery')"
            @save="saveProject"
            @next="onEditorTabChange('delivery')"
          />
        </div>

        <div v-show="editorTab === 'delivery'" class="lv-editor-pane lv-delivery-pane">
          <div class="lv-delivery-pane__sheet">
            <header class="lv-delivery-pane__head">
              <p class="lv-delivery-pane__lead">{{ $tt('video.longVideoDeliveryLead') }}</p>
            </header>

            <ul class="lv-delivery-checklist" aria-label="delivery checklist">
              <li
                v-for="item in deliveryChecklistItems"
                :key="item.id"
                class="lv-delivery-checklist__item"
                :class="{ 'is-ok': item.ok, 'is-pending': !item.ok }"
              >
                <span class="lv-delivery-checklist__mark" aria-hidden="true">{{ item.ok ? '✓' : '·' }}</span>
                <span class="lv-delivery-checklist__label">{{ $tt(item.labelKey) }}</span>
              </li>
            </ul>

            <p v-if="deliveryChecklistReady" class="lv-delivery-pane__ok">
              {{ $tt('video.longVideoDeliveryReady') }}
            </p>
          </div>

          <LongVideoStepSaveBar
            :save-label="$tt('video.longVideoSaveStepDelivery')"
            :hint="$tt('video.longVideoSaveStepDeliveryHint')"
            :status="autoSaveStatus"
            :status-label="autoSaveLabel"
            :loading="savingProject"
            :next-label="$tt('video.longVideoMergeFilm')"
            :next-loading="generating"
            :next-disabled="generating || mergeDisabled || assembleBlocked"
            @save="saveProject"
            @next="mergeLongVideo"
          />
        </div>
          </div>
        </div>
      </div>
    </div>

    <LongVideoInspectorDrawer
      :open="Boolean(selection)"
      :shots="shots"
      :selection="selection"
      :segment-duration-options="segmentDurationOptions"
      :keyframe-generating="keyframeGenerating"
      :segment-generating="segmentGeneratingForSelection"
      :segment-model-supports-r2v="segmentModelSupportsR2v"
      :visual-polishing="visualPolishing"
      :motion-polishing="motionPolishing"
      :output-size-label="outputSizeLabel"
      :segment-model-label="segmentModelLabel"
      :segment-compose-params="segmentComposeParams"
      :segment-param-schema="segmentParamSchema"
      :segment-compatible-loras="segmentCompatibleLoras"
      :segment-show-negative-prompt="segmentShowNegativePrompt"
      :segment-show-seed-field="segmentShowSeedField"
      :segment-show-lora="segmentShowLora"
      :default-chain-mode="project?.chain_mode ?? 'keyframe_only'"
      :compose-model="keyframeModelId"
      :compose-params="keyframeComposeParams"
      :compose-styles="composeStyles"
      :compose-show-negative-prompt="composeShowNegativePrompt"
      :compose-mode="composeMode"
      :compose-model-config="keyframeModelConfig"
      :compatible-loras="keyframeCompatibleLoras"
      :compatible-control-nets="keyframeCompatibleControlNets"
      :control-net-runtime-available="controlNetRuntimeAvailable"
      :reference-image="referenceImage"
      :control-image="keyframeControlImage"
      :inpaint-source-image="keyframeInpaintSourceImage"
      :inpaint-mask-image="keyframeInpaintMaskImage"
      :can-generate-keyframe="canGenerateKeyframe"
      :characters="effectiveCharacters"
      :scenes="effectiveScenes"
      :character-anchor="project?.character_anchor ?? ''"
      :style-anchor="effectiveStyleAnchor"
      :project-id="project?.project_id ?? ''"
      :parse-run-id="project?.chapter_analysis?.parse_run_id ?? ''"
      @update:open="onInspectorOpenChange"
      @update-visual="onUpdateVisual"
      @update-cast-looks="onUpdateCastLooks"
      @update-scene-look="onUpdateSceneLook"
      @update-motion="onUpdateMotion"
      @update-duration="onUpdateDuration"
      @update-chain-mode="onUpdateChainMode"
      @update-compose-mode="composeMode = $event"
      @reset-compose-defaults="resetKeyframeComposeDefaults()"
      @reset-segment-defaults="resetSegmentComposeDefaults()"
      @generate-keyframe="onGenerateKeyframe"
      @generate-segment="onGenerateSegment"
      @pick-keyframe-gallery="onPickKeyframeGallery"
      @clear-keyframe="onClearKeyframe"
      @clear-segment="onClearSegment"
      @select-segment="onSelectSegment"
      @polish-visual="onPolishVisual"
      @polish-motion="onPolishMotion"
      @pick-reference="openAssetPicker('reference')"
      @remove-reference="onRemoveReference"
      @pick-control="openAssetPicker('control')"
      @remove-control="onRemoveControl"
      @pick-inpaint-source="openAssetPicker('inpaint_source')"
      @remove-inpaint-source="onRemoveInpaintSource"
      @pick-inpaint-mask="openAssetPicker('inpaint_mask')"
      @remove-inpaint-mask="onRemoveInpaintMask"
    />

    <LongVideoParseStrategyDialog
      v-model:open="parseStrategyOpen"
      @choose="onParseStrategyChoose"
      @cancel="onParseStrategyCancel"
    />

    <DqDialog v-model:open="showAssetPicker" :title="$tt('assetPicker.dialogTitle')" width="70%">
      <AssetPicker media="image" @pick="onAssetPickerPick" />
    </DqDialog>

    <LongVideoOutputStrip
      :asset-id="project?.final_asset_id"
      @open-gallery="openFinalInGallery"
    />

    <LongVideoGalleryDrawer
      v-model:open="showKeyframePicker"
      :recent-gallery="recentImages"
      @pick="onGalleryDrawerPick"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch, nextTick, inject, type Ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import { toast, confirm } from '@/utils/feedback';
import {
  api,
  isRevisionConflictError,
  taskIdFromSubmitResponse,
  type LongVideoChapterAnalyzeShot,
} from '@/utils/api';
import {
  analyzeReferenceViaChat,
  checkKeyframeConsistencyViaChat,
  expandScriptViaChat,
} from '@/utils/llmMessages';
import { $tt } from '@/utils/i18n';
import { useTasksStore } from '@/stores/tasks';
import { useRegistryStore } from '@/stores/registry';
import type {
  LongVideoChainMode,
  LongVideoCharacter,
  LongVideoChapterAnalysis,
  LongVideoEditorTab,
  LongVideoProjectState,
  LongVideoScene,
  LongVideoSeriesDetail,
  LongVideoSeriesState,
  LongVideoShotCastLook,
  LongVideoShotSceneLook,
  LongVideoShotState,
  SystemInfo,
} from '@/types';
import LongVideoParseStrategyDialog, {
  type LongVideoParseStrategy,
} from '@/components/long-video/LongVideoParseStrategyDialog.vue';
import LongVideoBriefPanel from '@/components/long-video/LongVideoBriefPanel.vue';
import LongVideoCastPanel from '@/components/long-video/LongVideoCastPanel.vue';
import LongVideoScenePanel from '@/components/long-video/LongVideoScenePanel.vue';
import LongVideoEditorTabs from '@/components/long-video/LongVideoEditorTabs.vue';
import LongVideoSettingsPanel from '@/components/long-video/LongVideoSettingsPanel.vue';
import LongVideoStepSaveBar from '@/components/long-video/LongVideoStepSaveBar.vue';
import LongVideoGroupToolbar from '@/components/long-video/LongVideoGroupToolbar.vue';
import LongVideoBeatGroupRail from '@/components/long-video/LongVideoBeatGroupRail.vue';
import LongVideoGroupProgressBar from '@/components/long-video/LongVideoGroupProgressBar.vue';
import LongVideoInspectorDrawer from '@/components/long-video/LongVideoInspectorDrawer.vue';
import LongVideoGalleryDrawer from '@/components/long-video/LongVideoGalleryDrawer.vue';
import LongVideoOutputStrip from '@/components/long-video/LongVideoOutputStrip.vue';
import AssetPicker from '@/components/asset/AssetPicker.vue';
import { useLongVideoProject } from '@/composables/useLongVideoProject';
import { useLongVideoAutoSave } from '@/composables/useLongVideoAutoSave';
import { useLongVideoRegistry } from '@/composables/useLongVideoRegistry';
import {
  useLongVideoKeyframeCompose,
  type KeyframeComposeParams,
} from '@/composables/useLongVideoKeyframeCompose';
import { useLongVideoSegmentCompose } from '@/composables/useLongVideoSegmentCompose';
import { useComposerLlm } from '@/composables/useComposerLlm';
import { useStudioGallery } from '@/composables/useStudioGallery';
import {
  appendImageInferenceFields,
  appendZImageEnhancementFields,
  isControlNetHostRuntimeAvailable,
  resolveControlAssetId,
} from '@/composables/useStructuralGuide';
import { assetIdFromGalleryPath } from '@/utils/copilotHandoff';
import { openGlobalTaskQueue } from '@/utils/appEvents';
import {
  appendActiveEnumFields,
  formatResolutionOptionLabel,
  normalizeParamsDef,
  parseSizeValue,
  VIDEO_INFERENCE_ENUM_KEYS,
} from '@/utils/registryParamSchema';
import {
  allocateShotDurations,
  buildPortraitPrompt,
  buildCastVisionBackfillQuestion,
  buildSceneVisionBackfillQuestion,
  buildParseProvenanceByShot,
  buildSceneEnvironmentPrompt,
  resolveLongVideoLocale,
  buildWorkspaceSnapshot,
  buildKeyframeReferenceBindings,
  buildSegmentReferenceBindings,
  PRODUCTION_BLOCKER_LABEL_KEYS,
  episodeDeliveryChecklist,
  productionBlockers,
  serializeReferenceBindings,
  castReferenceScope,
  segmentI2vEndFrameAssetId,
  createEmptyShot,
  defaultLongVideoProject,
  effectiveShotChainMode,
  enrichShotsWithSceneLooks,
  mergeParsedShotsWithPrevious,
  insertKeyframeAfter,
  insertKeyframeBefore,
  invalidateSegmentAsset,
  resolveScriptText,
  longVideoHasPersistableContent,
  PORTRAIT_REFERENCE_SIZE,
  PORTRAIT_NEGATIVE_PROMPT_ZH,
  PORTRAIT_NEGATIVE_PROMPT_EN,
  SCENE_REFERENCE_SIZE,
  SCENE_ENV_NEGATIVE_ZH,
  SCENE_ENV_NEGATIVE_EN,
  keyframeGenerationPrompt,
  keyframePromptContextForShot,
  KEYFRAME_CAST_NEGATIVE_ZH,
  KEYFRAME_CAST_NEGATIVE_EN,
  looksMissingPortrait,
  looksMissingSceneReference,
  normalizeCharacterLookLabels,
  mergeCharacterRosters,
  mergeSceneRosters,
  mergeKeyframeLoraAdapters,
  resolvePrimaryCastPortraitForShot,
  shotCastMatchText,
  shotSceneText,
  shotKeyframeText,
  shotNeedsKeyframe,
  shotVideoPrompt,
  canGenerateSegmentShot,
  segmentI2vSourceAssetId,
  findFaceAnchorShot,
  groupShotsByBeat,
  planGroupGeneration,
  allPendingAnchorKeyframeIndices,
  allPendingSegmentIndices,
  selectedBeatGroupId,
  insertFaceAnchorIntoGroup,
  resplitBeatGroupRule,
  groupHasFaceAnchor,
  syncChapterAnalysisFields,
  syncRosterToCharacterAnchor,
  hydrateCharacterRoster,
  MIN_LONG_VIDEO_KEYFRAMES,
  MAX_SHOT_DURATION_SEC,
  SEGMENT_DURATION_CHOICES,
  nextShotId,
  numFramesForDurationSec,
  projectStateForServer,
  removeKeyframeAt,
  removeKeyframeNeedsConfirm,
  shotDurationSec,
  buildKeyframeGroundingMetadata,
  resolveSceneForShot,
} from '@/utils/longVideoProject';
import {
  assertProviderSupports,
  capReferenceAssetIds,
  chainModeToCapabilityMode,
  validateReferenceRequirement,
} from '@/utils/generationCapabilities';
import {
  isScriptParseError,
  mergeQualityIssues,
  type ScriptParseQualityIssue,
} from '@/utils/scriptParseError';
import {
  chapterAnalysisFromPipeline,
  resumeStageFromPipeline,
  rerunClearsFactsApproval,
  rerunClearsShotsApproval,
} from '@/utils/scriptPipeline';
import {
  clearCharacterVariant,
  clearSceneVariant,
  defaultSeriesState,
  diffCharacterAgainstSeries,
  diffSceneAgainstSeries,
  effectiveEpisodeRoster,
  characterHasEpisodeOverride,
  episodeOnlyCharacterVariant,
  episodeOnlySceneVariant,
  materializeCharacters,
  materializeScenes,
  promoteEpisodeCharacterToSeries,
  promoteEpisodeLookToSeriesAndLibrary,
  promoteEpisodeSceneLookToSeriesAndLibrary,
  promoteEpisodeSceneToSeries,
  sceneHasEpisodeOverride,
  restoreEpisodeCharacterFromSeries,
  restoreEpisodeSceneFromSeries,
  upsertCharacterVariant,
  upsertSceneVariant,
} from '@/utils/longVideoSeries';
import {
  attachParseRosterToEpisode,
  remapShotsThroughCharacterLinks,
} from '@/utils/longVideoEpisodeAttach';
import {
  loadSeriesWithLibraryMigration,
} from '@/utils/longVideoLibrary';
import { DQ_STORAGE, setItem } from '@/utils/storage';
import '@/styles/long-video.css';

const route = useRoute();
const router = useRouter();
const tasksStore = useTasksStore();
const registryStore = useRegistryStore();
const systemInfo = inject<Ref<SystemInfo>>('systemInfo', ref({} as SystemInfo));
const refreshWorkspaceNav = inject<() => void>('lvRefreshWorkspaceNav', () => undefined);
const longVideoProject = useLongVideoProject();
const {
  profile,
  modelLabel,
  modelParameters,
  modelHasBerniniRenderer,
  resolutionOptionsForModel,
  pickOutputSizeForModel,
  loadRegistry,
} = useLongVideoRegistry();
const {
  enhance,
  isEnhancing,
  scriptParseDecompose,
  scriptParseExpand,
  scriptParseExpandBeat,
  scriptParseProcess,
  isChapterAnalyzing,
  isStoryboardGenerating,
} = useComposerLlm();
const pipelineApproving = ref(false);
const pipelineRepairing = ref(false);
const pipelineApplyingRepair = ref(false);
const briefPanelRef = ref<{
  setRepairPreview?: (patches: import('@/types').ScriptRepairPatch[], pipe: import('@/types').ScriptPipeline) => void;
  discardRepairPreview?: () => void;
} | null>(null);
const pendingRepairPipeline = ref<import('@/types').ScriptPipeline | null>(null);
const { locale: uiLocale } = useI18n();
const { galleryItems, loadGallery } = useStudioGallery('image', { defaultGroupMode: false });

const generating = ref(false);
const savingProject = ref(false);
const keyframeGeneratingIndex = ref<number | null>(null);
const segmentGeneratingIndices = ref<Set<number>>(new Set());
const segmentGeneratingIndicesList = computed(() => [...segmentGeneratingIndices.value]);
const segmentGeneratingForSelection = computed(() => {
  const sel = selection.value;
  if (sel?.kind !== 'segment' && sel?.kind !== 'clip') return false;
  return segmentGeneratingIndices.value.has(sel.index);
});

const segmentModelSupportsR2v = computed(() =>
  modelHasBerniniRenderer(project.value?.segment_video_model ?? defaultSegmentModel.value),
);
const visualPolishing = ref(false);
const motionPolishing = ref(false);
const showKeyframePicker = ref(false);
const keyframePickIndex = ref<number | null>(null);
const showAssetPicker = ref(false);
const assetPickerMode = ref<'reference' | 'control' | 'inpaint_source' | 'inpaint_mask'>('reference');
const composeMode = ref('text2img');
const referenceImage = ref<{ previewUrl: string; path: string; assetId?: string } | null>(null);
const activeSeriesDetail = ref<LongVideoSeriesDetail | null>(null);
let seriesPersistTimer: ReturnType<typeof setTimeout> | null = null;
const portraitGeneratingKey = ref<string | null>(null);
const visionBackfillKey = ref<string | null>(null);
const sceneVisionBackfillKey = ref<string | null>(null);
const batchPortraitGenerating = ref(false);
const sceneRefGeneratingKey = ref<string | null>(null);
const batchSceneRefGenerating = ref(false);
const batchGroupGenerating = ref(false);
const galleryPickMode = ref<'keyframe' | 'portrait' | 'scene_ref'>('keyframe');
const galleryPickPortrait = ref<{ ci: number; li: number } | null>(null);
const galleryPickSceneRef = ref<{ si: number; li: number } | null>(null);
const consistencyWarnings = ref<Record<number, string>>({});
const consistencyChecking = ref(false);
const suppressAutoSave = ref(0);
const parseStrategyOpen = ref(false);
const scriptParseError = ref('');
const scriptParseProgressPhase = ref('');
const scriptParseProgressMessage = ref('');
const beatRegeneratingIndex = ref<number | null>(null);
const isScriptExpanding = ref(false);
/** Episode cast/scene page always edits episode variants; series masters change via promote. */

type ChapterAnalyzeApiResult = import('@/utils/api').ScriptParseExpandResult;
type DecomposeApiResult = import('@/utils/api').ScriptParseDecomposeResult;

const project = computed(() => longVideoProject.project.value);
const shots = computed(() => project.value?.shots ?? []);
const selection = computed(() => project.value?.selection ?? null);
const editorTab = computed((): LongVideoEditorTab => project.value?.editor_tab ?? 'script');

const matchedSeriesState = computed((): LongVideoSeriesState | null => {
  const lv = project.value;
  const detail = activeSeriesDetail.value;
  if (!lv?.series_id || !detail || detail.id !== lv.series_id) return null;
  return detail.state ?? null;
});

const effectiveRoster = computed(() => {
  const lv = project.value;
  if (!lv) return { characters: [] as LongVideoCharacter[], scenes: [] as LongVideoScene[] };
  return effectiveEpisodeRoster(lv, matchedSeriesState.value);
});

const effectiveCharacters = computed(() => effectiveRoster.value.characters);
const effectiveScenes = computed(() => effectiveRoster.value.scenes);

const effectiveStyleAnchor = computed(() => {
  const lv = project.value;
  if (!lv) return '';
  if (matchedSeriesState.value) {
    return matchedSeriesState.value.style_anchor ?? lv.style_anchor ?? '';
  }
  return lv.style_anchor ?? '';
});

const workspaceSnapshot = computed(() => {
  const lv = project.value;
  if (!lv) return null;
  return buildWorkspaceSnapshot(lv, matchedSeriesState.value);
});

const productionBlockerCodes = computed(() => {
  const lv = project.value;
  if (!lv) return [];
  const roster = effectiveEpisodeRoster(lv, matchedSeriesState.value);
  return productionBlockers(projectWithEffectiveRoster(lv), roster);
});

const productionBlockerLabels = computed(() =>
  productionBlockerCodes.value.map((c) => {
    const key = PRODUCTION_BLOCKER_LABEL_KEYS[c];
    const t = $tt(key);
    return t !== key ? t : c;
  }),
);

const batchGenerateBlocked = computed(() =>
  productionBlockerCodes.value.some((c) =>
    [
      'missing_cast_reference',
      'missing_scene_reference',
      'missing_keyframe',
      'reference_r2v_missing_refs',
      'last_frame_missing_tail',
      'first_last_missing_end_frame',
    ].includes(c),
  ),
);

const assembleBlocked = computed(() => productionBlockerCodes.value.length > 0);

const deliveryChecklist = computed(() => {
  const lv = project.value;
  if (!lv) return { ready: false, items: [] as ReturnType<typeof episodeDeliveryChecklist>['items'] };
  const roster = effectiveEpisodeRoster(lv, matchedSeriesState.value);
  return episodeDeliveryChecklist(projectWithEffectiveRoster(lv), roster);
});

const deliveryChecklistItems = computed(() => deliveryChecklist.value.items);
const deliveryChecklistReady = computed(() => deliveryChecklist.value.ready);

const deliveryStepDone = computed(() => deliveryChecklistReady.value);

const castHasEpisodeOverride = computed(() => {
  const lv = project.value;
  return Boolean(lv?.series_id && Object.keys(lv.character_variants ?? {}).length);
});

const sceneHasEpisodeOverrideFlag = computed(() => {
  const lv = project.value;
  return Boolean(lv?.series_id && Object.keys(lv.scene_variants ?? {}).length);
});

/** Character / scene promote: episode_only entities only. */
const promotableCharacterIds = computed(() => {
  const lv = project.value;
  if (!lv?.series_id) return [] as string[];
  return Object.entries(lv.character_variants ?? {})
    .filter(([, v]) => v.episode_only)
    .map(([id]) => id);
});

const promotableSceneIds = computed(() => {
  const lv = project.value;
  if (!lv?.series_id) return [] as string[];
  return Object.entries(lv.scene_variants ?? {})
    .filter(([, v]) => v.episode_only)
    .map(([id]) => id);
});

/** Look ids per character that can be promoted (episode_only look stubs / overrides). */
const promotableCharacterLookIds = computed(() => {
  const lv = project.value;
  const out: Record<string, string[]> = {};
  if (!lv?.series_id) return out;
  for (const [cid, v] of Object.entries(lv.character_variants ?? {})) {
    const ids = (v.looks ?? []).filter((lk) => lk.episode_only).map((lk) => lk.look_id);
    if (ids.length) out[cid] = ids;
  }
  return out;
});

const promotableSceneLookIds = computed(() => {
  const lv = project.value;
  const out: Record<string, string[]> = {};
  if (!lv?.series_id) return out;
  for (const [sid, v] of Object.entries(lv.scene_variants ?? {})) {
    const ids = (v.looks ?? []).filter((lk) => lk.episode_only).map((lk) => lk.look_id);
    if (ids.length) out[sid] = ids;
  }
  return out;
});

const episodeOnlyCharacterIds = computed(() => promotableCharacterIds.value);
const episodeOnlySceneIds = computed(() => promotableSceneIds.value);

function onRestoreCharacterFromSeries(characterId?: string) {
  const lv = project.value;
  if (!lv?.series_id || !characterId) return;
  longVideoProject.patchProject({
    character_variants: restoreEpisodeCharacterFromSeries(lv.character_variants, characterId),
  });
}

function onRestoreSceneFromSeries(sceneId?: string) {
  const lv = project.value;
  if (!lv?.series_id || !sceneId) return;
  longVideoProject.patchProject({
    scene_variants: restoreEpisodeSceneFromSeries(lv.scene_variants, sceneId),
  });
}

async function onPromoteCharacterToSeries(characterId: string) {
  const lv = project.value;
  const detail = activeSeriesDetail.value;
  if (!lv?.series_id || !detail || detail.id !== lv.series_id || !characterId) return;
  const effective = effectiveCharacters.value.find((c) => c.id === characterId);
  if (!effective) return;
  try {
    const result = promoteEpisodeCharacterToSeries(
      detail.state,
      effective,
      lv.character_variants,
    );
    activeSeriesDetail.value = { ...detail, state: result.series };
    longVideoProject.patchProject({ character_variants: result.characterVariants });
    await persistActiveSeries();
    toast.success($tt('video.longVideoPromoteEntityToSeriesOk'));
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  }
}

async function onPromoteCharacterLookToSeries(characterId: string, lookId: string) {
  const lv = project.value;
  const detail = activeSeriesDetail.value;
  if (!lv?.series_id || !detail || detail.id !== lv.series_id || !characterId || !lookId) return;
  const onSeries = (detail.state.characters ?? []).some((c) => c.id === characterId);
  if (!onSeries) {
    toast.warning($tt('video.longVideoPromoteLookNeedCharacter'));
    return;
  }
  try {
    const result = await promoteEpisodeLookToSeriesAndLibrary(
      detail.state,
      lv.character_variants,
      characterId,
      lookId,
      api.library,
    );
    activeSeriesDetail.value = { ...detail, state: result.series };
    longVideoProject.patchProject({ character_variants: result.characterVariants });
    await persistActiveSeries();
    toast.success($tt('video.longVideoPromoteLookToSeriesOk'));
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  }
}

async function onPromoteSceneToSeries(sceneId: string) {
  const lv = project.value;
  const detail = activeSeriesDetail.value;
  if (!lv?.series_id || !detail || detail.id !== lv.series_id || !sceneId) return;
  const effective = effectiveScenes.value.find((s) => s.id === sceneId);
  if (!effective) return;
  try {
    const result = promoteEpisodeSceneToSeries(detail.state, effective, lv.scene_variants);
    activeSeriesDetail.value = { ...detail, state: result.series };
    longVideoProject.patchProject({ scene_variants: result.sceneVariants });
    await persistActiveSeries();
    toast.success($tt('video.longVideoPromoteEntityToSeriesOk'));
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  }
}

async function onPromoteSceneLookToSeries(sceneId: string, lookId: string) {
  const lv = project.value;
  const detail = activeSeriesDetail.value;
  if (!lv?.series_id || !detail || detail.id !== lv.series_id || !sceneId || !lookId) return;
  const onSeries = (detail.state.scenes ?? []).some((s) => s.id === sceneId);
  if (!onSeries) {
    toast.warning($tt('video.longVideoPromoteLookNeedScene'));
    return;
  }
  try {
    const result = await promoteEpisodeSceneLookToSeriesAndLibrary(
      detail.state,
      lv.scene_variants,
      sceneId,
      lookId,
      api.library,
    );
    activeSeriesDetail.value = { ...detail, state: result.series };
    longVideoProject.patchProject({ scene_variants: result.sceneVariants });
    await persistActiveSeries();
    toast.success($tt('video.longVideoPromoteLookToSeriesOk'));
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  }
}

/** Project snapshot with series-effective cast/scenes for production helpers. */
function toastProductionBlockers(): boolean {
  const lv = project.value;
  if (!lv) return true;
  const roster = effectiveEpisodeRoster(lv, matchedSeriesState.value);
  const lvProd = projectWithEffectiveRoster(lv);
  const codes = productionBlockers(lvProd, roster);
  if (!codes.length) return false;
  const labels = codes.slice(0, 4).map((c) => {
    const key = PRODUCTION_BLOCKER_LABEL_KEYS[c];
    const t = $tt(key);
    return t !== key ? t : c;
  });
  toast.error($tt('video.longVideoProductionBlocked', { items: labels.join(' · ') }));
  return true;
}

function projectWithEffectiveRoster(lv: LongVideoProjectState): LongVideoProjectState {
  const snap = workspaceSnapshot.value;
  if (snap) {
    return {
      ...lv,
      characters: snap.characters,
      scenes: snap.scenes,
      style_anchor: snap.style_anchor,
    };
  }
  const roster = effectiveEpisodeRoster(lv, matchedSeriesState.value);
  return {
    ...lv,
    characters: roster.characters,
    scenes: roster.scenes,
    style_anchor: matchedSeriesState.value?.style_anchor ?? lv.style_anchor,
  };
}

function keyframeContextForShot(shot: LongVideoShotState) {
  const lv = project.value;
  const snap = workspaceSnapshot.value;
  const base = {
    character_anchor: lv?.character_anchor ?? '',
    characters: snap?.characters ?? lv?.characters ?? [],
    scenes: snap?.scenes ?? lv?.scenes ?? [],
    style_anchor: snap?.style_anchor ?? effectiveStyleAnchor.value,
  };
  return keyframePromptContextForShot(shot, base);
}

function isStillActiveProject(projectId: string | undefined): boolean {
  return Boolean(projectId) && project.value?.project_id === projectId;
}

function withProjectMetadata(meta: Record<string, unknown> = {}): Record<string, unknown> {
  const pid = project.value?.project_id;
  if (!pid) return meta;
  const sid = project.value?.series_id;
  return {
    ...meta,
    group_id: pid,
    long_video_project_id: pid,
    long_video_project_title: project.value?.title ?? '',
    ...(sid ? { long_video_series_id: sid } : {}),
  };
}

/** Best-effort save so generated assets get a gallery group via project_id. */
async function ensureProjectSavedForGeneration(): Promise<void> {
  const lv = project.value;
  if (!lv || lv.project_id) return;
  if (!longVideoHasPersistableContent(lv)) return;
  await persistProjectToServer({ silent: true });
}

const scriptText = computed(() => {
  const lv = project.value;
  return lv ? resolveScriptText(lv) : '';
});

const projectSynopsis = computed(() => {
  const lv = project.value;
  if (!lv) return '';
  return lv.chapter_analysis?.synopsis?.trim() || '';
});

/** Episode production setup ready: models chosen (resolution locked to project). */
const settingsStepDone = computed(() => {
  const lv = project.value;
  if (!lv) return false;
  return Boolean(
    String(lv.keyframe_model || '').trim() && String(lv.segment_video_model || '').trim(),
  );
});

const activeScriptPipeline = computed(() => {
  const lv = project.value;
  return lv?.script_pipeline ?? null;
});

const factsApproved = computed(() => Boolean(activeScriptPipeline.value?.workflow?.facts_approved));
const shotsApproved = computed(() => Boolean(activeScriptPipeline.value?.workflow?.shots_approved));

const scriptStepDone = computed(() => {
  const lv = project.value;
  if (!lv) return false;
  if (activeScriptPipeline.value) {
    return factsApproved.value || (lv.chapter_analysis?.scene_beats?.length ?? 0) >= 1;
  }
  return (lv.chapter_analysis?.scene_beats?.length ?? 0) >= 2;
});

const castStepDone = computed(() => {
  const chars = effectiveCharacters.value;
  if (!chars.length) return false;
  return looksMissingPortrait(chars).length === 0;
});

const scenesStepDone = computed(() => {
  const scenes = effectiveScenes.value;
  if (!scenes.length) return false;
  return looksMissingSceneReference(scenes).length === 0;
});

const storyboardStepDone = computed(() => shots.value.length >= MIN_LONG_VIDEO_KEYFRAMES);

/** Soft gate: facts not approved — blocks cast/scenes/storyboard tabs, not a parse failure. */
const awaitingFactsGate = computed(
  () => Boolean(activeScriptPipeline.value && !factsApproved.value),
);

/** Hard block shown as error alert (failed parse / critical quality / audit blockers). */
const scriptParseHardBlocked = computed(() => {
  const analysis = project.value?.chapter_analysis;
  if (analysis?.parse_blocked) return true;
  if (activeScriptPipeline.value?.audit_report?.issues?.some((i) => i.severity === 'blocker' && i.status !== 'resolved')) {
    return true;
  }
  return (analysis?.quality_issues ?? []).some((i) => i.severity === 'critical');
});

const scriptParseHardBlockedReason = computed(() => {
  const analysis = project.value?.chapter_analysis;
  return analysis?.parse_blocked_reason?.trim()
    || (analysis?.quality_issues ?? []).find((i) => i.severity === 'critical')?.message
    || '';
});

const storyboardTabBlocked = computed(() => awaitingFactsGate.value || scriptParseHardBlocked.value);

const storyboardTabBlockedReason = computed(() => {
  if (awaitingFactsGate.value) return $tt('video.longVideoPipeNeedFacts');
  return scriptParseHardBlockedReason.value;
});

const defaultKeyframeModel = computed(() => profile.value?.keyframe_models[0] ?? 'z-image-turbo');
const defaultSegmentModel = computed(() => profile.value?.segment_models[0] ?? 'wan-2.2-i2v-14b');
const keyframeModelOptions = computed(() => profile.value?.keyframe_models ?? []);
const segmentModelOptions = computed(() => profile.value?.segment_models ?? []);
const segmentDurationOptions = [...SEGMENT_DURATION_CHOICES];

const keyframeGenerating = computed(() => keyframeGeneratingIndex.value != null);

const composeModelId = ref('z-image-turbo');
const composeOutputSize = ref('1280x720');
const keyframeCompose = useLongVideoKeyframeCompose(composeModelId, composeOutputSize);
const {
  params: keyframeComposeParams,
  compatibleLoras: keyframeCompatibleLoras,
  compatibleControlNets: keyframeCompatibleControlNets,
  controlImage: keyframeControlImage,
  inpaintSourceImage: keyframeInpaintSourceImage,
  inpaintMaskImage: keyframeInpaintMaskImage,
  currentModelConfig: keyframeModelConfig,
  filteredPresets: composeStyles,
  showNegativePrompt: composeShowNegativePrompt,
  resetToDefaults: resetKeyframeComposeDefaults,
} = keyframeCompose;

const keyframeModelId = computed(() => project.value?.keyframe_model ?? defaultKeyframeModel.value);

const controlNetRuntimeAvailable = computed(() =>
  isControlNetHostRuntimeAvailable(
    systemInfo.value?.controlnet_runtime_available,
    [],
    systemInfo.value,
  ),
);

const selectedNodeIndex = computed(() =>
  selection.value?.kind === 'segment' ? selection.value.index : null,
);

const canGenerateKeyframe = computed(() => {
  const idx = selectedNodeIndex.value;
  if (idx == null) return false;
  const shot = shots.value[idx];
  if (!shotNeedsKeyframe(shot)) return false;
  if (!shotKeyframeText(shot)) return false;
  if (composeMode.value === 'img2img' && !referenceImage.value) return false;
  return true;
});

const consistencyWarningForSelection = computed(() => {
  const idx = selectedNodeIndex.value;
  if (idx == null) return null;
  return consistencyWarnings.value[idx] ?? null;
});

function onParseProgress(phase: string, message?: string) {
  scriptParseProgressPhase.value = phase;
  scriptParseProgressMessage.value = message?.trim() || '';
}

function markParseBlocked(reason: string, incomingIssues: ScriptParseQualityIssue[] = []) {
  const lv = project.value;
  if (!lv) return;
  const prev = lv.chapter_analysis ?? {
    synopsis: '',
    scene_beats: [],
  };
  const structured = incomingIssues.length
    ? incomingIssues
    : [{ code: 'parse_blocked', message: reason, severity: 'critical' as const }];
  const mergedIssues = mergeQualityIssues(prev.quality_issues, structured);
  setProjectWithChapterSync({
    ...lv,
    chapter_analysis: {
      ...prev,
      parse_blocked: true,
      parse_blocked_reason: reason,
      quality_issues: mergedIssues,
    },
  });
}

function clearParseBlocked() {
  const lv = project.value;
  if (!lv?.chapter_analysis?.parse_blocked) return;
  const prev = lv.chapter_analysis;
  setProjectWithChapterSync({
    ...lv,
    chapter_analysis: {
      ...prev,
      parse_blocked: false,
      parse_blocked_reason: undefined,
      quality_issues: (prev.quality_issues ?? []).filter(
        (i) => i.severity !== 'critical' && i.code !== 'parse_blocked',
      ),
    },
  });
  scriptParseError.value = '';
}

function onStoryboardTabBlocked() {
  const reason = storyboardTabBlockedReason.value;
  toast.warning(reason || $tt('video.longVideoStoryboardTabBlocked'));
}

function onEditorTabChange(tab: LongVideoEditorTab) {
  if (
    activeScriptPipeline.value
    && !factsApproved.value
    && (tab === 'cast' || tab === 'scenes' || tab === 'storyboard')
  ) {
    toast.warning($tt('video.longVideoPipeNeedFacts'));
    return;
  }
  if (tab === 'storyboard' && storyboardTabBlocked.value) {
    onStoryboardTabBlocked();
    return;
  }
  patchProjectField('editor_tab', tab);
}

function portraitKey(ci: number, li: number): string {
  return `${ci}-${li}`;
}

function patchLookPortrait(
  ci: number,
  li: number,
  patch: { reference_asset_id?: string; portrait_prompt?: string; body?: string; vision_description?: string },
) {
  const base = effectiveCharacters.value;
  if (!base[ci]?.looks[li]) return;
  const characters = base.map((ch, i) => {
    if (i !== ci) return ch;
    return {
      ...ch,
      looks: ch.looks.map((lk, j) => {
        if (j !== li) return lk;
        const next = { ...lk, ...patch };
        if ('portrait_prompt' in patch && patch.portrait_prompt === undefined) {
          delete next.portrait_prompt;
        }
        if ('vision_description' in patch && patch.vision_description === undefined) {
          delete next.vision_description;
        }
        return next;
      }),
    };
  });
  onUpdateCharacters(characters);
}

function sceneRefKey(si: number, li: number): string {
  return `${si}-${li}`;
}

function patchSceneLookRef(
  si: number,
  li: number,
  patch: { reference_asset_id?: string; environment_prompt?: string; body?: string; vision_description?: string },
) {
  const base = effectiveScenes.value;
  if (!base[si]?.looks[li]) return;
  const scenes = base.map((sc, i) => {
    if (i !== si) return sc;
    return {
      ...sc,
      looks: sc.looks.map((lk, j) => {
        if (j !== li) return lk;
        const next = { ...lk, ...patch };
        if ('environment_prompt' in patch && patch.environment_prompt === undefined) {
          delete next.environment_prompt;
        }
        if ('vision_description' in patch && patch.vision_description === undefined) {
          delete next.vision_description;
        }
        return next;
      }),
    };
  });
  onUpdateScenes(scenes);
}

function onUpdateScenes(scenes: LongVideoScene[]) {
  const lv = project.value;
  if (!lv) return;
  if (lv.series_id && activeSeriesDetail.value?.id === lv.series_id) {
    const seriesScenes = activeSeriesDetail.value.state.scenes ?? [];
    let nextVariants = { ...(lv.scene_variants ?? {}) };
    const keepIds = new Set(scenes.map((s) => s.id));
    for (const id of Object.keys(nextVariants)) {
      if (!keepIds.has(id)) nextVariants = clearSceneVariant(nextVariants, id);
    }
    for (const sc of scenes) {
      const base = seriesScenes.find((s) => s.id === sc.id);
      if (!base) {
        nextVariants = {
          ...nextVariants,
          [sc.id]: episodeOnlySceneVariant(sc),
        };
        continue;
      }
      const diff = diffSceneAgainstSeries(base, sc, { looksAndDefaultOnly: true });
      if (diff) {
        nextVariants = upsertSceneVariant(nextVariants, sc.id, diff);
      } else {
        nextVariants = clearSceneVariant(nextVariants, sc.id);
      }
    }
    longVideoProject.patchProject({ scenes: [], scene_variants: nextVariants });
    return;
  }
  longVideoProject.patchProject({ scenes });
}

function patchSceneGrounding(
  si: number,
  patch: { grounding_panorama_asset_id?: string; grounding_depth_asset_id?: string },
) {
  const base = effectiveScenes.value;
  if (!base[si]) return;
  const scenes = base.map((sc, i) => (i === si ? { ...sc, ...patch } : sc));
  onUpdateScenes(scenes);
}

async function onGenerateSceneRef(si: number, li: number) {
  const lv = project.value;
  if (!lv) return;
  const boundProjectId = lv.project_id;
  const sc = effectiveScenes.value[si];
  const look = sc?.looks[li];
  if (!sc || !look) return;

  const locale = uiLocale.value.startsWith('zh') ? 'zh' : 'en';
  const prompt = buildSceneEnvironmentPrompt(sc, look, effectiveStyleAnchor.value, locale, {
    useCache: false,
  });
  const model = lv.keyframe_model ?? defaultKeyframeModel.value;
  const { width, height } = SCENE_REFERENCE_SIZE;
  const key = sceneRefKey(si, li);
  const modelParameters = (registryStore.registry?.models?.[model] as Record<string, unknown> | undefined)
    ?.parameters as Record<string, unknown> | undefined;

  sceneRefGeneratingKey.value = key;
  try {
    await ensureProjectSavedForGeneration();
    const genBody: Record<string, unknown> = {
      model,
      prompt,
      negative_prompt: locale === 'zh' ? SCENE_ENV_NEGATIVE_ZH : SCENE_ENV_NEGATIVE_EN,
      size: `${width}x${height}`,
      n: 1,
      steps: keyframeComposeParams.steps,
      guidance: keyframeComposeParams.guidance,
      metadata: withProjectMetadata({ long_video_phase: 'scene_ref', scene_id: sc.id, scene_look_id: look.id }),
      priority: 'normal',
    };
    appendImageInferenceFields(
      genBody,
      keyframeComposeParams as Record<string, unknown>,
      modelParameters,
    );
    const submitRes = await api.gen.createImageGeneration(genBody);
    const tid = taskIdFromSubmitResponse(submitRes);
    if (!tid) throw new Error('no task id');
    tasksStore.registerPageOwnedStream(tid);
    const { assetId } = await waitForImageTask(tid);
    if (!assetId) throw new Error('no asset');
    if (!isStillActiveProject(boundProjectId)) return;
    patchSceneLookRef(si, li, { reference_asset_id: assetId, environment_prompt: prompt });
    patchSceneGrounding(si, { grounding_panorama_asset_id: assetId });
    try {
      const depth = await api.longVideo.sceneGroundingDepthFromAsset({
        source_asset_id: assetId,
        width,
        height,
      });
      if (depth.depth_asset_id && isStillActiveProject(boundProjectId)) {
        patchSceneGrounding(si, {
          grounding_panorama_asset_id: assetId,
          grounding_depth_asset_id: depth.depth_asset_id,
        });
      }
    } catch {
      /* depth-pro optional; panorama still usable */
    }
    try {
      await loadGallery(true);
    } catch {
      /* gallery refresh must not fail the generation flow */
    }
    toast.success($tt('video.longVideoSceneRefDone'));
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.genFailed', { msg }));
  } finally {
    if (sceneRefGeneratingKey.value === key) sceneRefGeneratingKey.value = null;
  }
}

function onPickSceneRefGallery(si: number, li: number) {
  galleryPickMode.value = 'scene_ref';
  galleryPickSceneRef.value = { si, li };
  showKeyframePicker.value = true;
}

function onClearSceneRef(si: number, li: number) {
  patchSceneLookRef(si, li, {
    reference_asset_id: undefined,
    environment_prompt: undefined,
    vision_description: undefined,
  });
}

async function onVisionBackfillSceneLook(si: number, li: number) {
  const sc = effectiveScenes.value[si];
  const look = sc?.looks[li];
  const assetId = look?.reference_asset_id;
  if (!sc || !look || !assetId) return;
  const key = sceneRefKey(si, li);
  sceneVisionBackfillKey.value = key;
  const locale = resolveLongVideoLocale(uiLocale.value);
  try {
    const visionInfo = await api.gen.getVisionModelInfo();
    if (!visionInfo.available) {
      toast.warning($tt('video.longVideoSceneVisionUnavailable'));
      return;
    }
    const question = buildSceneVisionBackfillQuestion(sc, look, locale);
    const text = await analyzeReferenceViaChat(assetId, question, { locale });
    if (text) {
      patchSceneLookRef(si, li, { vision_description: text });
      toast.success($tt('video.longVideoSceneVisionBackfillDone'));
    }
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  } finally {
    if (sceneVisionBackfillKey.value === key) sceneVisionBackfillKey.value = null;
  }
}

async function onBatchGenerateSceneRefs() {
  if (!project.value || batchSceneRefGenerating.value) return;
  const missing = looksMissingSceneReference(effectiveScenes.value);
  if (!missing.length) return;
  batchSceneRefGenerating.value = true;
  try {
    for (const { sceneIndex, lookIndex } of missing) {
      await onGenerateSceneRef(sceneIndex, lookIndex);
    }
  } finally {
    batchSceneRefGenerating.value = false;
  }
}

async function onGeneratePortrait(ci: number, li: number) {
  const lv = project.value;
  if (!lv) return;
  const boundProjectId = lv.project_id;
  const chars = effectiveCharacters.value;
  const ch = chars[ci];
  const look = ch?.looks[li];
  if (!ch || !look) return;

  const locale = resolveLongVideoLocale(uiLocale.value);
  const otherNames = chars
    .filter((_, i) => i !== ci)
    .map((c) => c.name)
    .filter(Boolean);
  const prompt = buildPortraitPrompt(ch, look, effectiveStyleAnchor.value, locale, {
    otherCharacterNames: otherNames,
    useCache: false,
  });
  const model = lv.portrait_model ?? lv.keyframe_model ?? defaultKeyframeModel.value;
  const { width, height } = PORTRAIT_REFERENCE_SIZE;
  const key = portraitKey(ci, li);
  const modelParameters = (registryStore.registry?.models?.[model] as Record<string, unknown> | undefined)
    ?.parameters as Record<string, unknown> | undefined;

  portraitGeneratingKey.value = key;
  try {
    await ensureProjectSavedForGeneration();
    const genBody: Record<string, unknown> = {
      model,
      prompt,
      negative_prompt: locale === 'zh' ? PORTRAIT_NEGATIVE_PROMPT_ZH : PORTRAIT_NEGATIVE_PROMPT_EN,
      size: `${width}x${height}`,
      n: 1,
      steps: keyframeComposeParams.steps,
      guidance: keyframeComposeParams.guidance,
      metadata: withProjectMetadata({ long_video_phase: 'cast_portrait', cast_character_id: ch.id, cast_look_id: look.id }),
      priority: 'normal',
    };
    appendImageInferenceFields(
      genBody,
      keyframeComposeParams as Record<string, unknown>,
      modelParameters,
    );
    const submitRes = await api.gen.createImageGeneration(genBody);
    const tid = taskIdFromSubmitResponse(submitRes);
    if (!tid) throw new Error('no task id');
    tasksStore.registerPageOwnedStream(tid);
    const { assetId } = await waitForImageTask(tid);
    if (!assetId) throw new Error('no asset');
    if (!isStillActiveProject(boundProjectId)) return;
    patchLookPortrait(ci, li, { reference_asset_id: assetId, portrait_prompt: prompt });
    try {
      await loadGallery(true);
    } catch {
      /* gallery refresh must not fail the generation flow */
    }
    toast.success($tt('video.longVideoPortraitDone'));
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.genFailed', { msg }));
  } finally {
    if (portraitGeneratingKey.value === key) portraitGeneratingKey.value = null;
  }
}

function onPickPortraitGallery(ci: number, li: number) {
  galleryPickMode.value = 'portrait';
  galleryPickPortrait.value = { ci, li };
  showKeyframePicker.value = true;
}

function onClearPortrait(ci: number, li: number) {
  patchLookPortrait(ci, li, {
    reference_asset_id: undefined,
    portrait_prompt: undefined,
    vision_description: undefined,
  });
}

async function onVisionBackfillLook(ci: number, li: number) {
  const chars = effectiveCharacters.value;
  const ch = chars[ci];
  const look = ch?.looks[li];
  const assetId = look?.reference_asset_id;
  if (!ch || !look || !assetId) return;
  const key = portraitKey(ci, li);
  visionBackfillKey.value = key;
  const locale = resolveLongVideoLocale(uiLocale.value);
  const otherNames = chars
    .filter((_, i) => i !== ci)
    .map((c) => c.name.trim())
    .filter(Boolean);
  try {
    const visionInfo = await api.gen.getVisionModelInfo();
    if (!visionInfo.available) {
      toast.warning($tt('video.longVideoCastVisionUnavailable'));
      return;
    }
    const question = buildCastVisionBackfillQuestion(ch, otherNames, locale);
    const text = await analyzeReferenceViaChat(assetId, question, { locale });
    if (text) {
      patchLookPortrait(ci, li, { vision_description: text });
      toast.success($tt('video.longVideoCastVisionBackfillDone'));
    }
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  } finally {
    if (visionBackfillKey.value === key) visionBackfillKey.value = null;
  }
}

async function onBatchGeneratePortraits() {
  if (!project.value || batchPortraitGenerating.value) return;
  const missing = looksMissingPortrait(effectiveCharacters.value);
  if (!missing.length) return;
  batchPortraitGenerating.value = true;
  try {
    for (const { characterIndex, lookIndex } of missing) {
      await onGeneratePortrait(characterIndex, lookIndex);
    }
  } finally {
    batchPortraitGenerating.value = false;
  }
}

async function onCheckConsistency(index: number) {
  const lv = project.value;
  const shot = lv?.shots[index];
  if (!lv || !shot?.keyframe_asset_id) return;
  const scene = shotCastMatchText(shot);
  const portrait = resolvePrimaryCastPortraitForShot(
    effectiveCharacters.value,
    shot.cast_looks ?? [],
    scene,
  );
  if (!portrait?.reference_asset_id) return;

  consistencyChecking.value = true;
  try {
    const visionInfo = await api.gen.getVisionModelInfo();
    if (!visionInfo.available) {
      toast.warning($tt('video.longVideoCastVisionUnavailable'));
      return;
    }
    const locale = resolveLongVideoLocale(uiLocale.value);
    const answer = await checkKeyframeConsistencyViaChat(
      portrait.reference_asset_id,
      shot.keyframe_asset_id,
      locale,
    );
    const ok = /一致|consistent/i.test(answer);
    if (ok) {
      const next = { ...consistencyWarnings.value };
      delete next[index];
      consistencyWarnings.value = next;
    } else {
      consistencyWarnings.value = {
        ...consistencyWarnings.value,
        [index]: answer || $tt('video.longVideoConsistencyWarning'),
      };
    }
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  } finally {
    consistencyChecking.value = false;
  }
}

function onGalleryDrawerPick(payload: { path: string }) {
  if (galleryPickMode.value === 'portrait' && galleryPickPortrait.value) {
    const assetId = assetIdFromGalleryPath(payload.path);
    if (!assetId) {
      toast.warning($tt('canvas.controlImageAssetRequired'));
      return;
    }
    const { ci, li } = galleryPickPortrait.value;
    patchLookPortrait(ci, li, { reference_asset_id: assetId });
    galleryPickPortrait.value = null;
    galleryPickMode.value = 'keyframe';
    showKeyframePicker.value = false;
    return;
  }
  if (galleryPickMode.value === 'scene_ref' && galleryPickSceneRef.value) {
    const assetId = assetIdFromGalleryPath(payload.path);
    if (!assetId) {
      toast.warning($tt('canvas.controlImageAssetRequired'));
      return;
    }
    const { si, li } = galleryPickSceneRef.value;
    patchSceneLookRef(si, li, { reference_asset_id: assetId });
    galleryPickSceneRef.value = null;
    galleryPickMode.value = 'keyframe';
    showKeyframePicker.value = false;
    return;
  }
  onKeyframeAssetPick(payload);
}

function bindReferenceFromShot(shot: LongVideoShotState | undefined) {
  const refId = shot?.reference_asset_id;
  if (refId) {
    referenceImage.value = {
      path: `asset:${refId}`,
      previewUrl: api.gallery.getImageUrl(`asset:${refId}`),
      assetId: refId,
    };
    composeMode.value = 'img2img';
    return;
  }
  referenceImage.value = null;
  if (composeMode.value === 'img2img') composeMode.value = 'text2img';
}

watch(
  () => (selection.value?.kind === 'segment' ? shots.value[selection.value.index] : null),
  (shot) => bindReferenceFromShot(shot ?? undefined),
  { immediate: true },
);

function openAssetPicker(mode: 'reference' | 'control' | 'inpaint_source' | 'inpaint_mask') {
  assetPickerMode.value = mode;
  showAssetPicker.value = true;
}

function assetIdFromPickPath(path: string): string | undefined {
  return path.startsWith('asset:') ? path.slice('asset:'.length) : undefined;
}

function resolveInpaintAssetId(ref: { path: string; assetId?: string } | null): string {
  if (!ref) return '';
  if (ref.assetId) return ref.assetId;
  if (ref.path.startsWith('asset:')) return ref.path.slice('asset:'.length);
  return '';
}

function patchShotReference(index: number, assetId: string | undefined) {
  const lv = project.value;
  if (!lv) return;
  const shotsNext = lv.shots.map((s, i) =>
    i === index ? { ...s, reference_asset_id: assetId } : s,
  );
  longVideoProject.setProject({ ...lv, shots: shotsNext });
}

function onRemoveReference() {
  referenceImage.value = null;
  composeMode.value = 'text2img';
  const idx = selectedNodeIndex.value;
  if (idx != null) patchShotReference(idx, undefined);
}

function onAssetPickerPick(payload: { path: string; previewUrl: string }) {
  const { path, previewUrl } = payload;
  const assetId = assetIdFromPickPath(path);
  showAssetPicker.value = false;

  if (assetPickerMode.value === 'control') {
    keyframeControlImage.value = { path, previewUrl };
    return;
  }
  if (assetPickerMode.value === 'inpaint_source') {
    if (!path.startsWith('asset:')) {
      toast.warning($tt('canvas.controlImageAssetRequired'));
      return;
    }
    keyframeInpaintSourceImage.value = { path, previewUrl };
    return;
  }
  if (assetPickerMode.value === 'inpaint_mask') {
    if (!path.startsWith('asset:')) {
      toast.warning($tt('canvas.controlImageAssetRequired'));
      return;
    }
    keyframeInpaintMaskImage.value = { path, previewUrl };
    return;
  }

  referenceImage.value = { path, previewUrl, assetId };
  composeMode.value = 'img2img';
  const idx = selectedNodeIndex.value;
  if (idx != null && assetId) patchShotReference(idx, assetId);
}

const projectTitle = computed(() => project.value?.title ?? '');

function effectiveTargetDurationSec(lv: LongVideoProjectState): number {
  if (!lv.shots.length) return lv.target_duration_sec ?? 60;
  const total = lv.shots.reduce((sum, s) => sum + shotDurationSec(s), 0);
  return Math.max(lv.target_duration_sec ?? 60, total);
}

const segmentModelId = computed(() => project.value?.segment_video_model ?? defaultSegmentModel.value);

const segmentModelLabel = computed(() => modelLabel(segmentModelId.value));

const segmentCompose = useLongVideoSegmentCompose(segmentModelId);
const {
  params: segmentComposeParams,
  compatibleLoras: segmentCompatibleLoras,
  paramSchema: segmentParamSchema,
  showNegativePrompt: segmentShowNegativePrompt,
  showSeedField: segmentShowSeedField,
  showLora: segmentShowLora,
  resetToDefaults: resetSegmentComposeDefaults,
} = segmentCompose;

const segmentResolutionOptions = computed(() => resolutionOptionsForModel(segmentModelId.value));

/** Resolution is project-scoped; episode inherits series output_size. */
const projectOutputSize = computed(() => {
  const seriesSize = matchedSeriesState.value?.output_size;
  const saved =
    project.value?.series_id && seriesSize
      ? seriesSize
      : project.value?.output_size || seriesSize;
  const picked = pickOutputSizeForModel(segmentModelId.value, saved);
  return picked ?? segmentResolutionOptions.value[0]?.value ?? '1280x720';
});

const effectiveOverlapFrames = computed(() => {
  if (project.value?.series_id && matchedSeriesState.value) {
    return matchedSeriesState.value.overlap_frames ?? 4;
  }
  return project.value?.overlap_frames ?? 4;
});

const outputSizePixels = computed(() => {
  const parsed = parseSizeValue(projectOutputSize.value);
  return parsed ?? { width: 1280, height: 720 };
});

const outputSizeLabel = computed(() => {
  const opt = segmentResolutionOptions.value.find((o) => o.value === projectOutputSize.value);
  if (opt) return formatResolutionOptionLabel(opt);
  return projectOutputSize.value.replace('x', '×');
});

function scalarDefault(modelId: string, key: string, fallback: number): number {
  const spec = normalizeParamsDef(modelParameters(modelId))[key];
  const def = spec?.default;
  return typeof def === 'number' ? def : fallback;
}

function syncOutputSizeForSegmentModel(modelId: string) {
  // Under a series, resolution is project-locked — do not write episode.output_size.
  if (project.value?.series_id) return;
  const next = pickOutputSizeForModel(modelId, project.value?.output_size);
  if (next && next !== project.value?.output_size) {
    patchProjectField('output_size', next);
  }
}

function onSegmentModelChange(modelId: string) {
  patchProjectField('segment_video_model', modelId);
  syncOutputSizeForSegmentModel(modelId);
}

function onKeyframeModelChange(modelId: string) {
  patchProjectField('keyframe_model', modelId);
}

function openProjectSettings() {
  const sid = project.value?.series_id;
  if (!sid) return;
  void router.push({ name: 'long_video_series', params: { seriesId: sid } });
}

watch(
  keyframeModelId,
  (id) => {
    composeModelId.value = id;
  },
  { immediate: true },
);

watch(
  projectOutputSize,
  (size) => {
    composeOutputSize.value = size;
  },
  { immediate: true },
);

function onRemoveControl() {
  keyframeControlImage.value = null;
}

function onRemoveInpaintSource() {
  keyframeInpaintSourceImage.value = null;
}

function onRemoveInpaintMask() {
  keyframeInpaintMaskImage.value = null;
}

function onInspectorOpenChange(open: boolean) {
  if (!open) clearSelection();
}

const mergeDisabled = computed(() => {
  if (!shots.value.length || generating.value || segmentGeneratingIndices.value.size > 0) return true;
  return !shots.value.every((s) => s.status === 'segment_ready' && s.segment_asset_id);
});

const recentImages = computed(() => galleryItems.value.slice(0, 40) as Array<Record<string, unknown>>);

function patchProjectField<K extends keyof LongVideoProjectState>(key: K, value: LongVideoProjectState[K]) {
  longVideoProject.patchProject({ [key]: value } as Partial<LongVideoProjectState>);
}

function hasCastPortraitWork(): boolean {
  return effectiveCharacters.value.some((ch) =>
    ch.looks.some((lk) => Boolean(lk.reference_asset_id)),
  );
}

function hasStoryboardGeneratedWork(): boolean {
  const lv = project.value;
  if (!lv) return false;
  return lv.shots.some(
    (s) => s.visual_prompt?.trim() || s.motion_prompt?.trim() || s.keyframe_asset_id || s.segment_asset_id,
  );
}

function hasExistingParseWork(): boolean {
  const lv = project.value;
  if (!lv) return false;
  if (scriptStepDone.value) return true;
  if (effectiveCharacters.value.length > 0) return true;
  if ((lv.shots?.length ?? 0) > 0) return true;
  if (hasCastPortraitWork()) return true;
  return hasStoryboardGeneratedWork();
}

function deriveNewProjectTitle(): string {
  const lv = project.value;
  const fromChapter = lv?.chapter_title?.trim();
  if (fromChapter) return fromChapter;
  const text = scriptText.value.trim();
  if (!text) return $tt('video.longVideoPageTitle');
  return text.length > 28 ? `${text.slice(0, 28)}…` : text;
}

async function bootstrapNewProjectForParse(): Promise<boolean> {
  const lv = project.value;
  if (!lv) return false;
  const seriesId = lv.series_id?.trim();
  if (!seriesId) {
    toast.warning($tt('video.longVideoStandaloneRemoved'));
    return false;
  }
  const draft = defaultLongVideoProject({
    keyframe_model: lv.keyframe_model,
    segment_video_model: lv.segment_video_model,
    segment_duration_sec: lv.segment_duration_sec ?? 5,
    target_duration_sec: lv.target_duration_sec ?? 60,
    output_size: lv.output_size,
    overlap_frames: lv.overlap_frames,
    chain_mode: lv.chain_mode,
    title: deriveNewProjectTitle(),
    script_text: scriptText.value,
    chapter_title: lv.chapter_title,
    series_id: seriesId,
  });
  try {
    const { project: created } = await api.longVideo.createSeriesEpisode(seriesId, {
      title: draft.title?.trim() || $tt('video.longVideoPageTitle'),
      state: projectStateForServer(draft),
    });
    applyServerProject(created, { silent: true });
    if (created.id) {
      await router.replace({
        name: 'long_video_create',
        params: { projectId: created.id },
      });
    }
    await loadSeriesDetail(seriesId);
    toast.info($tt('video.longVideoParseStrategyNewProjectStarted'));
    return true;
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
    return false;
  }
}

function setProjectWithChapterSync(next: LongVideoProjectState) {
  const locale: 'zh' | 'en' = uiLocale.value.startsWith('zh') ? 'zh' : 'en';
  const beatTexts = (next.chapter_analysis?.scene_beats ?? []).map((b) => b.beat);
  const characters = normalizeCharacterLookLabels(next.characters ?? [], locale, beatTexts);
  const style = (next.style_anchor ?? '').trim();
  const characterAnchor =
    characters.length > 0
      ? syncRosterToCharacterAnchor(characters, style)
      : next.character_anchor;
  let patched: LongVideoProjectState = {
    ...next,
    characters,
    character_anchor: characterAnchor,
  };
  const analysis = patched.chapter_analysis;
  if (analysis) {
    patched = {
      ...patched,
      chapter_analysis: syncChapterAnalysisFields(analysis, {
        characters,
        character_anchor: characterAnchor,
        style_anchor: patched.style_anchor ?? analysis.style_anchor,
      }),
    };
  }
  longVideoProject.setProject(patched);
}

function onScriptTextChange(text: string) {
  const lv = project.value;
  if (!lv) return;
  longVideoProject.patchProject({ script_text: text });
}

async function onScriptExpand() {
  const lv = project.value;
  if (!lv) return;
  const text = scriptText.value.trim();
  if (!text) {
    toast.warning($tt('video.longVideoScriptNeedText'));
    return;
  }
  isScriptExpanding.value = true;
  scriptParseError.value = '';
  try {
    const seg = Math.max(1, lv.segment_duration_sec ?? 5);
    const target = Math.max(seg, lv.target_duration_sec ?? 60);
    const expanded = await expandScriptViaChat(text, {
      locale: uiLocale.value,
      targetShotCount: Math.max(2, Math.round(target / seg)),
      narrativeBudget: 'standard',
    });
    const next = expanded.trim();
    if (!next) {
      toast.error($tt('video.longVideoScriptExpandFailed', { msg: $tt('video.longVideoChapterAnalyzeEmpty') }));
      return;
    }
    longVideoProject.patchProject({ script_text: next });
    toast.success($tt('video.longVideoScriptExpandDone'));
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('video.longVideoScriptExpandFailed', { msg }));
  } finally {
    isScriptExpanding.value = false;
  }
}

async function onScriptParse() {
  if (hasExistingParseWork()) {
    parseStrategyOpen.value = true;
    return;
  }
  await runScriptParsePipeline('replace');
}

async function onParseStrategyChoose(strategy: LongVideoParseStrategy) {
  await runScriptParsePipeline(strategy);
}

function onParseStrategyCancel() {
  parseStrategyOpen.value = false;
}

/**
 * Standalone projects: write cast/scenes onto the project.
 * Episode mode: style only onto series — cast/scenes attach via {@link attachParseRosterToEpisode}.
 */
function commitSharedRoster(opts: {
  characters: LongVideoCharacter[];
  scenes: LongVideoScene[];
  style_anchor?: string;
  character_anchor?: string;
}): Pick<LongVideoProjectState, 'characters' | 'scenes' | 'style_anchor' | 'character_anchor'> {
  const lv = project.value;
  if (lv?.series_id && activeSeriesDetail.value?.id === lv.series_id) {
    if (opts.style_anchor !== undefined) {
      const st: LongVideoSeriesState = {
        ...activeSeriesDetail.value.state,
        style_anchor: opts.style_anchor,
      };
      activeSeriesDetail.value = { ...activeSeriesDetail.value, state: st };
      scheduleSeriesPersist();
    }
    return {
      characters: [],
      scenes: [],
      style_anchor: opts.style_anchor,
      character_anchor: opts.character_anchor,
    };
  }
  return {
    characters: opts.characters,
    scenes: opts.scenes,
    style_anchor: opts.style_anchor,
    character_anchor: opts.character_anchor,
  };
}

/** Attach parse characters/scenes to episode (link or episode_only); never mutates series cast/scenes. */
async function attachIncomingRosterToEpisode(
  incomingChars: LongVideoCharacter[],
  incomingScenes: LongVideoScene[],
): Promise<{
  characterVariants: NonNullable<LongVideoProjectState['character_variants']>;
  sceneVariants: NonNullable<LongVideoProjectState['scene_variants']>;
  characterLinks: Record<string, string>;
  sceneLinks: Record<string, string>;
  characters: LongVideoCharacter[];
  scenes: LongVideoScene[];
  reusedNames: string[];
  newEpisodeNames: string[];
}> {
  const lv = project.value!;
  const series = matchedSeriesState.value;
  // Episode parse uses each project character/scene's **currently selected** version pin.
  const seriesChars = series?.characters ?? [];
  const seriesScenes = series?.scenes ?? [];
  const attach = attachParseRosterToEpisode({
    incomingCharacters: incomingChars,
    incomingScenes,
    seriesCharacters: seriesChars,
    seriesScenes,
    priorCharacterLinks: lv.character_links,
    priorSceneLinks: lv.scene_links,
    priorCharacterVariants: lv.character_variants,
    priorSceneVariants: lv.scene_variants,
  });
  const characters = materializeCharacters(seriesChars, attach.characterVariants);
  const scenes = materializeScenes(seriesScenes, attach.sceneVariants);
  return {
    characterVariants: attach.characterVariants,
    sceneVariants: attach.sceneVariants,
    characterLinks: attach.characterLinks,
    sceneLinks: attach.sceneLinks,
    characters,
    scenes,
    reusedNames: attach.reusedNames,
    newEpisodeNames: attach.newEpisodeNames,
  };
}

function toastParseAttachSummary(reusedNames: string[], newEpisodeNames: string[]) {
  if (reusedNames.length) {
    toast.success(
      $tt('video.longVideoParseReuseCast', {
        names: reusedNames.slice(0, 4).join('、'),
        n: reusedNames.length,
      }),
    );
  }
  if (newEpisodeNames.length) {
    toast.info(
      $tt('video.longVideoParseNewEpisodeCast', {
        names: newEpisodeNames.slice(0, 4).join('、'),
        n: newEpisodeNames.length,
      }),
    );
  }
}

function onChapterAnalysisChange(value: LongVideoChapterAnalysis | undefined) {
  const lv = project.value;
  if (!lv) return;
  if (!value) {
    setProjectWithChapterSync({ ...lv, chapter_analysis: value });
    return;
  }
  const nextChars = value.characters?.length ? value.characters : effectiveCharacters.value;
  const nextScenes = value.scenes?.length ? value.scenes : effectiveScenes.value;
  const roster = commitSharedRoster({
    characters: nextChars,
    scenes: nextScenes,
    style_anchor: value.style_anchor ?? effectiveStyleAnchor.value,
    character_anchor: value.character_anchor ?? lv.character_anchor,
  });
  setProjectWithChapterSync({
    ...lv,
    chapter_analysis: value,
    ...roster,
  });
}

function onTargetDurationChange(value: number) {
  patchProjectField('target_duration_sec', value);
}

function onSegmentDurationChange(value: number) {
  const sec = Math.max(1, Number(value) || 5);
  longVideoProject.patchProject({ segment_duration_sec: sec });
}

function storyboardShotsFromResponse(
  apiShots: LongVideoChapterAnalyzeShot[],
  opts?: {
    defaultSegmentSec?: number;
    targetDurationSec?: number;
    beatTexts?: string[];
  },
): LongVideoShotState[] {
  const fallbackDurations = allocateShotDurations({
    sceneCount: apiShots.length,
    targetDurationSec: opts?.targetDurationSec ?? 60,
    defaultSegmentSec: opts?.defaultSegmentSec ?? 5,
    beatTexts: opts?.beatTexts,
  });
  return apiShots.map((s, i) => {
    const videoPrompt = (s.video_prompt || s.motion_prompt || '').trim();
    const startVisual = (s.start_visual_prompt || s.visual_prompt || '').trim();
    const chainMode =
      s.chain_mode ??
      (s.flf_mode === 'first_last'
        ? ('first_last' as const)
        : s.start_frame_mode === 'prev_segment_tail'
          ? ('last_frame' as const)
          : ('keyframe_only' as const));
    return {
      id: s.id || `shot_${String(i).padStart(2, '0')}`,
      order: i,
      visual_prompt: startVisual,
      motion_prompt: videoPrompt,
      video_prompt: videoPrompt,
      start_visual_prompt: startVisual || undefined,
      end_visual_prompt: s.end_visual_prompt?.trim() || undefined,
      anchor_visual_prompt: s.anchor_visual_prompt?.trim() || undefined,
      segment_role: s.segment_role ?? 'keyframe',
      start_frame_mode: s.start_frame_mode ?? 'keyframe',
      segment_group_id: s.segment_group_id,
      segment_group_index: s.segment_group_index,
      face_anchor_shot_id: s.face_anchor_shot_id,
      flf_mode: s.flf_mode ?? 'none',
      end_frame_sync_anchor: s.end_frame_sync_anchor,
      chain_mode: chainMode,
      scene_prompt: s.scene_prompt || '',
      cast_looks: s.cast_looks ?? [],
      scene_look: s.scene_look,
      first_frame_visibility: s.first_frame_visibility,
      end_visibility: s.end_visibility,
      characters_on_screen: s.characters_on_screen ?? [],
      clip_start_state: s.clip_start_state,
      clip_end_state: s.clip_end_state,
      first_frame_requirement: s.first_frame_requirement,
      camera_zone_id: s.camera_zone_id,
      first_frame_strategy: s.first_frame_strategy,
      location: s.location?.trim() || undefined,
      narrative_beat_index:
        typeof s.narrative_beat_index === 'number' ? s.narrative_beat_index : undefined,
      shot_size: s.shot_size?.trim() || undefined,
      duration_sec:
        typeof s.duration_sec === 'number' && s.duration_sec > 0
          ? s.duration_sec
          : fallbackDurations[i] ?? opts?.defaultSegmentSec ?? 5,
      status: 'draft' as const,
    };
  });
}

async function applyScriptDecomposeResult(
  result: DecomposeApiResult,
  opts: { mergeCharacters?: boolean } = {},
): Promise<number> {
  const lv = project.value;
  if (!lv || !result?.scene_beats?.length) return 0;

  const scene_beats = result.scene_beats.map((s) => ({
    order: s.order,
    title: s.title || '',
    beat: s.beat || '',
  }));
  const incomingRaw = (result.characters ?? []) as LongVideoCharacter[];
  const rosterPatch = hydrateCharacterRoster(
    {
      characters: incomingRaw,
      character_anchor: result.character_anchor || lv.character_anchor || '',
      style_anchor: result.style_anchor || effectiveStyleAnchor.value || '',
    },
    uiLocale.value.startsWith('zh') ? 'zh' : 'en',
  );
  const incomingScenes = (result.scenes ?? []) as LongVideoScene[];
  const parseRunId = result.parse_run_id || lv.chapter_analysis?.parse_run_id || '';
  const parseAt = parseRunId ? new Date().toISOString() : lv.chapter_analysis?.last_parse_at;

  const inEpisode = Boolean(lv.series_id && matchedSeriesState.value);
  let displayChars = rosterPatch.characters ?? [];
  let displayScenes = incomingScenes;
  let episodePatch: Partial<LongVideoProjectState> = {};

  if (inEpisode) {
    const attached = await attachIncomingRosterToEpisode(displayChars, incomingScenes);
    displayChars = attached.characters;
    displayScenes = attached.scenes;
    episodePatch = {
      characters: [],
      scenes: [],
      character_variants: attached.characterVariants,
      scene_variants: attached.sceneVariants,
      character_links: attached.characterLinks,
      scene_links: attached.sceneLinks,
      style_anchor: rosterPatch.style_anchor,
      character_anchor: rosterPatch.character_anchor,
    };
    if (rosterPatch.style_anchor !== undefined) {
      commitSharedRoster({
        characters: [],
        scenes: [],
        style_anchor: rosterPatch.style_anchor,
        character_anchor: rosterPatch.character_anchor,
      });
    }
    toastParseAttachSummary(attached.reusedNames, attached.newEpisodeNames);
  } else {
    const mergedChars =
      opts.mergeCharacters !== false
        ? mergeCharacterRosters(effectiveCharacters.value, displayChars)
        : displayChars;
    const mergedScenes =
      opts.mergeCharacters !== false
        ? mergeSceneRosters(effectiveScenes.value, incomingScenes)
        : incomingScenes;
    const hydrated = hydrateCharacterRoster(
      {
        characters: mergedChars,
        character_anchor: rosterPatch.character_anchor || '',
        style_anchor: rosterPatch.style_anchor || '',
      },
      uiLocale.value.startsWith('zh') ? 'zh' : 'en',
    );
    displayChars = hydrated.characters ?? mergedChars;
    displayScenes = mergedScenes;
    episodePatch = commitSharedRoster({
      characters: displayChars,
      scenes: displayScenes,
      style_anchor: hydrated.style_anchor,
      character_anchor: hydrated.character_anchor,
    });
  }

  const analysis: LongVideoChapterAnalysis = {
    synopsis: result.synopsis || '',
    mood: result.mood || '',
    scene_beats,
    character_anchor: rosterPatch.character_anchor,
    style_anchor: rosterPatch.style_anchor,
    characters: displayChars,
    scenes: displayScenes,
    quality_warnings: [],
    quality_issues: [],
    parse_blocked: false,
    parse_blocked_reason: undefined,
    parse_run_id: parseRunId,
    last_parse_at: parseAt,
    parse_phases: result.parse_phases ?? lv.chapter_analysis?.parse_phases,
    script_artifact: result.script_artifact ?? lv.chapter_analysis?.script_artifact,
    shot_t2i_provenance: lv.chapter_analysis?.shot_t2i_provenance,
    parse_history: lv.chapter_analysis?.parse_history,
  };
  setProjectWithChapterSync({
    ...lv,
    chapter_analysis: analysis,
    ...episodePatch,
    chapter_title: result.chapter_title?.trim() || lv.chapter_title,
    shots: [],
  });
  return scene_beats.length;
}

function buildScriptArtifactForExpand(lv: LongVideoProjectState): Record<string, unknown> | null {
  const base = lv.chapter_analysis?.script_artifact;
  if (!base || base.version !== '2.0') return null;
  const eff = projectWithEffectiveRoster(lv);
  const merged = { ...base } as Record<string, unknown>;
  merged.title = lv.chapter_title || base.title || '';
  merged.style_anchor = eff.style_anchor || base.style_anchor || '';
  if (eff.characters?.length) {
    merged.characters = eff.characters.map((ch) => ({
      id: ch.id,
      name: ch.name,
      role: (base.characters as Array<{ id: string; role?: string }> | undefined)?.find((c) => c.id === ch.id)?.role ?? 'supporting',
      looks: ch.looks.map((lk) => ({
        id: lk.id,
        label: lk.label,
        body: lk.body,
        portrait_prompt_hint: '',
      })),
      default_look_id: ch.default_look_id,
    }));
  }
  if (eff.scenes?.length) {
    merged.scenes = eff.scenes.map((sc) => ({
      id: sc.id,
      name: sc.name,
      looks: sc.looks.map((lk) => ({
        id: lk.id,
        label: lk.label,
        body: lk.body,
        environment_prompt_hint: '',
      })),
      default_look_id: sc.default_look_id,
      spatial_layout_json: sc.spatial_layout_json ?? {},
      grounding_panorama_asset_id: sc.grounding_panorama_asset_id ?? '',
      grounding_depth_asset_id: sc.grounding_depth_asset_id ?? '',
    }));
  }
  return merged;
}

async function applyScriptAnalyzeResult(
  result: ChapterAnalyzeApiResult,
  opts: { mergeCharacters?: boolean } = {},
): Promise<number> {
  const lv = project.value;
  if (!lv || !result?.scene_beats?.length) return 0;

  const scene_beats = result.scene_beats.map((s) => ({
    order: s.order,
    title: s.title || '',
    beat: s.beat || '',
  }));
  const incomingRaw = (result.characters ?? []) as LongVideoCharacter[];
  const rosterPatch = hydrateCharacterRoster(
    {
      characters: incomingRaw,
      character_anchor: result.character_anchor || lv.character_anchor || '',
      style_anchor: result.style_anchor || effectiveStyleAnchor.value || '',
    },
    uiLocale.value.startsWith('zh') ? 'zh' : 'en',
  );
  const incomingScenes = (result.scenes ?? []) as LongVideoScene[];
  const inEpisode = Boolean(lv.series_id && matchedSeriesState.value);

  let displayChars = rosterPatch.characters ?? [];
  let displayScenes = incomingScenes;
  let characterLinks = lv.character_links ?? {};
  let episodePatch: Partial<LongVideoProjectState> = {};

  if (inEpisode) {
    const attached = await attachIncomingRosterToEpisode(displayChars, incomingScenes);
    displayChars = attached.characters;
    displayScenes = attached.scenes;
    characterLinks = attached.characterLinks;
    episodePatch = {
      characters: [],
      scenes: [],
      character_variants: attached.characterVariants,
      scene_variants: attached.sceneVariants,
      character_links: attached.characterLinks,
      scene_links: attached.sceneLinks,
      style_anchor: rosterPatch.style_anchor,
      character_anchor: rosterPatch.character_anchor,
    };
    if (rosterPatch.style_anchor !== undefined) {
      commitSharedRoster({
        characters: [],
        scenes: [],
        style_anchor: rosterPatch.style_anchor,
        character_anchor: rosterPatch.character_anchor,
      });
    }
    toastParseAttachSummary(attached.reusedNames, attached.newEpisodeNames);
  } else {
    const mergedChars =
      opts.mergeCharacters !== false
        ? mergeCharacterRosters(effectiveCharacters.value, displayChars)
        : displayChars;
    const mergedScenes =
      opts.mergeCharacters !== false
        ? mergeSceneRosters(effectiveScenes.value, incomingScenes)
        : incomingScenes;
    const hydrated = hydrateCharacterRoster(
      {
        characters: mergedChars,
        character_anchor: rosterPatch.character_anchor || '',
        style_anchor: rosterPatch.style_anchor || '',
      },
      uiLocale.value.startsWith('zh') ? 'zh' : 'en',
    );
    displayChars = hydrated.characters ?? mergedChars;
    displayScenes = mergedScenes;
    episodePatch = commitSharedRoster({
      characters: displayChars,
      scenes: displayScenes,
      style_anchor: hydrated.style_anchor,
      character_anchor: hydrated.character_anchor,
    });
  }

  const snap = buildWorkspaceSnapshot(
    {
      ...lv,
      character_variants: (episodePatch.character_variants as LongVideoProjectState['character_variants'])
        ?? lv.character_variants,
      scene_variants: (episodePatch.scene_variants as LongVideoProjectState['scene_variants'])
        ?? lv.scene_variants,
    },
    matchedSeriesState.value,
  );
  const projectForProvenance = {
    character_anchor: rosterPatch.character_anchor,
    characters: displayChars.length ? displayChars : snap.characters,
    scenes: displayScenes.length ? displayScenes : snap.scenes,
    style_anchor: rosterPatch.style_anchor ?? snap.style_anchor,
  };
  let parsedShots = result.shots?.length
    ? enrichShotsWithSceneLooks(
        storyboardShotsFromResponse(result.shots, {
          defaultSegmentSec: lv.segment_duration_sec ?? 5,
          targetDurationSec: lv.target_duration_sec ?? 60,
          beatTexts: scene_beats.map((s) => s.beat),
        }),
        displayScenes,
      )
    : [];
  if (parsedShots.length && inEpisode) {
    parsedShots = remapShotsThroughCharacterLinks(
      parsedShots,
      characterLinks,
      displayChars,
    );
  }
  const provenanceByShot = parsedShots.length
    ? buildParseProvenanceByShot(parsedShots, projectForProvenance)
    : {};
  const parseRunId = result.parse_run_id || lv.chapter_analysis?.parse_run_id || '';
  const parseAt = parseRunId ? new Date().toISOString() : lv.chapter_analysis?.last_parse_at;
  const parseHistory = [...(lv.chapter_analysis?.parse_history ?? [])];
  if (parseRunId && parsedShots.length) {
    parseHistory.push({
      parse_run_id: parseRunId,
      at: parseAt || new Date().toISOString(),
      shot_count: parsedShots.length,
      provenance_by_shot_id: provenanceByShot,
    });
    while (parseHistory.length > 5) parseHistory.shift();
  }
  const analysis: LongVideoChapterAnalysis = {
    synopsis: result.synopsis || '',
    mood: result.mood || '',
    scene_beats,
    character_anchor: rosterPatch.character_anchor,
    style_anchor: rosterPatch.style_anchor,
    characters: displayChars,
    scenes: displayScenes,
    quality_warnings: result.quality_warnings ?? [],
    quality_issues: result.quality_issues ?? [],
    parse_blocked: false,
    parse_blocked_reason: undefined,
    parse_run_id: parseRunId,
    last_parse_at: parseAt,
    parse_phases: result.parse_phases ?? lv.chapter_analysis?.parse_phases,
    script_artifact: result.script_artifact ?? lv.chapter_analysis?.script_artifact,
    shot_t2i_provenance: Object.keys(provenanceByShot).length ? provenanceByShot : lv.chapter_analysis?.shot_t2i_provenance,
    parse_history: parseHistory.length ? parseHistory : lv.chapter_analysis?.parse_history,
  };
  setProjectWithChapterSync({
    ...lv,
    chapter_analysis: analysis,
    ...episodePatch,
    chapter_title: result.chapter_title?.trim() || lv.chapter_title,
    shots:
      parsedShots.length
        ? mergeParsedShotsWithPrevious(lv.shots ?? [], parsedShots)
        : [],
  });
  return scene_beats.length;
}

function persistPipelineCheckpoint(result: import('@/utils/api').ScriptParsePipelineResult) {
  const pipe = result.script_pipeline;
  if (!pipe) return;
  const lv = project.value;
  if (!lv) return;
  setProjectWithChapterSync({
    ...lv,
    script_pipeline: pipe,
    chapter_analysis: chapterAnalysisFromPipeline(pipe, lv.chapter_analysis),
  });
}

function handlePipelineStreamError(msg: string, err?: unknown) {
  scriptParseError.value = msg;
  if (isScriptParseError(err) && err.scriptPipeline) {
    persistPipelineCheckpoint({ script_pipeline: err.scriptPipeline });
  }
}

function pipelineBusy(): boolean {
  return isChapterAnalyzing.value || isStoryboardGenerating.value || pipelineApproving.value || pipelineRepairing.value || pipelineApplyingRepair.value;
}

async function runScriptParsePipeline(strategy: LongVideoParseStrategy) {
  if (pipelineBusy()) return;
  parseStrategyOpen.value = false;
  scriptParseError.value = '';
  scriptParseProgressPhase.value = 'r0';

  if (strategy === 'new_project') {
    if (!(await bootstrapNewProjectForParse())) return;
  }

  const lv = project.value;
  if (!lv) return;
  const text = scriptText.value.trim();
  if (!text) {
    toast.warning($tt('video.longVideoScriptNeedText'));
    return;
  }

  const mergeCharacters = strategy === 'replace';

  await ensureProjectSavedForGeneration();
  const projectId = project.value?.project_id ?? lv.project_id ?? '';
  const parseModel = lv.script_parse_llm_model?.trim() || '';

  const pipelineResult = await scriptParseProcess(
    {
      script_text: text,
      title: (lv.chapter_title || '').trim(),
      locale: uiLocale.value,
      long_video_project_id: projectId,
      model: parseModel,
      existing_pipeline: activeScriptPipeline.value ?? undefined,
      stop_at: 'gate1',
    },
    {
      quietSuccess: true,
      onProgress: onParseProgress,
      onCheckpoint: persistPipelineCheckpoint,
      onError: handlePipelineStreamError,
    },
  );
  scriptParseProgressPhase.value = '';
  scriptParseProgressMessage.value = '';
  if (!pipelineResult?.script_pipeline) {
    if (!scriptParseError.value) {
      scriptParseError.value = $tt('video.longVideoChapterAnalyzeFailedGeneric');
    }
    return;
  }

  await applyPipelineResult(pipelineResult, { mergeCharacters, clearShots: true });
  toast.success($tt('video.longVideoPipeParseOk'));
  // Stay on script tab for Gate1 review — do not auto-jump to cast.
}

async function onResumePipeline() {
  if (pipelineBusy()) return;
  const pipe = activeScriptPipeline.value;
  const stage = resumeStageFromPipeline(pipe);
  if (!pipe || !stage) return;
  scriptParseError.value = '';
  if (pipe.workflow.facts_approved || ['r5', 'r6', 'r7', 'gate2'].includes(stage)) {
    await runExpandPipeline({ mode: 'full' });
    return;
  }
  await runScriptParsePipeline('replace');
}

async function onRerunPipelineStage(payload: { stage: string; sceneId?: string }) {
  if (pipelineBusy()) return;
  const pipe = activeScriptPipeline.value;
  if (!pipe) return;
  const warnings: string[] = [];
  if (rerunClearsFactsApproval(payload.stage) && pipe.workflow.facts_approved) {
    warnings.push($tt('video.longVideoPipeRerunClearsFacts'));
  }
  if (rerunClearsShotsApproval(payload.stage) && pipe.workflow.shots_approved) {
    warnings.push($tt('video.longVideoPipeRerunClearsShots'));
  }
  if (warnings.length) {
    const ok = window.confirm(`${warnings.join('\n')}\n\n${$tt('video.longVideoPipeRerunConfirm', { stage: payload.stage })}`);
    if (!ok) return;
  }
  pipelineApproving.value = true;
  scriptParseError.value = '';
  scriptParseProgressPhase.value = payload.stage;
  try {
    const lv = project.value;
    const result = await api.gen.scriptParseRerun(payload.stage, {
      existing_pipeline: pipe,
      model: lv?.script_parse_llm_model || undefined,
      locale: uiLocale.value,
      long_video_project_id: lv?.project_id,
      only_scene_ids: payload.sceneId ? [payload.sceneId] : undefined,
      target_duration_sec: lv?.target_duration_sec ?? 60,
      segment_duration_sec: lv?.segment_duration_sec ?? 5,
      max_clip_sec: MAX_SHOT_DURATION_SEC,
    });
    await applyPipelineResult(result, {
      mergeCharacters: true,
      clearShots: false,
      applyShots: Boolean(result.shots?.length),
    });
    toast.success($tt('video.longVideoPipeRerunOk', { stage: payload.stage }));
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    scriptParseError.value = msg;
    toast.error($tt('studio.error', { msg }));
  } finally {
    pipelineApproving.value = false;
    scriptParseProgressPhase.value = '';
    scriptParseProgressMessage.value = '';
  }
}

async function applyPipelineResult(
  result: import('@/utils/api').ScriptParsePipelineResult,
  opts: { mergeCharacters?: boolean; clearShots?: boolean; applyShots?: boolean } = {},
) {
  const lv = project.value;
  if (!lv) return;
  const pipe = result.script_pipeline;
  const scene_beats = result.scene_beats ?? [];
  const analysis = {
    ...(lv.chapter_analysis ?? { synopsis: '', scene_beats: [] }),
    synopsis: result.synopsis || pipe.story_map?.logline || lv.chapter_analysis?.synopsis || '',
    mood: result.mood || pipe.story_map?.mood,
    style_anchor: result.style_anchor || pipe.story_map?.style_anchor,
    scene_beats: scene_beats.length ? scene_beats : lv.chapter_analysis?.scene_beats ?? [],
    parse_run_id: result.parse_run_id || pipe.parse_run_id,
    last_parse_at: new Date().toISOString(),
    parse_blocked: false,
    parse_blocked_reason: '',
  };
  if (pipe.workflow.facts_approved && result.characters?.length) {
    const fake = {
      chapter_title: lv.chapter_title || '',
      synopsis: analysis.synopsis,
      mood: analysis.mood,
      character_anchor: '',
      style_anchor: analysis.style_anchor,
      characters: result.characters,
      scenes: result.scenes,
      scene_beats,
      scene_count: scene_beats.length,
      script_artifact: { version: '2.0', beats: [], characters: result.characters, scenes: result.scenes },
      llm_calls: result.llm_calls ?? 0,
      parse_run_id: result.parse_run_id,
    };
    await applyScriptDecomposeResult(fake as import('@/utils/api').ScriptParseDecomposeResult, {
      mergeCharacters: opts.mergeCharacters ?? true,
    });
    const after = project.value;
    if (after) {
      setProjectWithChapterSync({
        ...after,
        script_pipeline: pipe,
        chapter_analysis: analysis,
        shots: opts.clearShots ? [] : after.shots,
      });
    }
    return;
  }
  let nextShots = lv.shots;
  if (opts.clearShots) {
    nextShots = [];
  } else if (opts.applyShots && result.shots?.length) {
    let parsed = storyboardShotsFromResponse(result.shots as never, {
      defaultSegmentSec: lv.segment_duration_sec ?? 5,
      targetDurationSec: lv.target_duration_sec ?? 60,
      beatTexts: (result.scene_beats ?? []).map((b) => b.beat),
    });
    if (lv.series_id && matchedSeriesState.value) {
      parsed = remapShotsThroughCharacterLinks(
        parsed,
        lv.character_links ?? {},
        effectiveCharacters.value,
      );
    }
    nextShots = mergeParsedShotsWithPrevious(lv.shots ?? [], parsed);
  }
  setProjectWithChapterSync({
    ...lv,
    script_pipeline: pipe,
    chapter_analysis: analysis,
    shots: nextShots,
  });
}

async function onApprovePipelineFacts() {
  const pipe = activeScriptPipeline.value;
  if (!pipe) return;
  pipelineApproving.value = true;
  try {
    const result = await api.gen.scriptParseApproveFacts(pipe);
    await applyPipelineResult(result, { mergeCharacters: true, clearShots: false });
    toast.success($tt('video.longVideoPipeFactsOk'));
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  } finally {
    pipelineApproving.value = false;
  }
}

async function onApprovePipelineShots() {
  const pipe = activeScriptPipeline.value;
  if (!pipe) return;
  pipelineApproving.value = true;
  try {
    const result = await api.gen.scriptParseApproveShots(pipe);
    await applyPipelineResult(result, { applyShots: true });
    toast.success($tt('video.longVideoPipeShotsOk'));
    onEditorTabChange('storyboard');
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  } finally {
    pipelineApproving.value = false;
  }
}

async function onGeneratePipelineShots() {
  if (!factsApproved.value) {
    toast.warning($tt('video.longVideoPipeNeedFacts'));
    return;
  }
  await runExpandPipeline({ mode: 'full' });
}

async function onPipelineRepairPreview(payload: { findingIds: string[]; userNote: string }) {
  const pipe = activeScriptPipeline.value;
  if (!pipe || !payload.findingIds.length) return;
  pipelineRepairing.value = true;
  try {
    const result = await api.gen.scriptParseRepairPreview({
      pipeline: pipe,
      finding_ids: payload.findingIds,
      user_note: payload.userNote,
      model: project.value?.script_parse_llm_model || undefined,
    });
    pendingRepairPipeline.value = result.script_pipeline;
    briefPanelRef.value?.setRepairPreview?.(result.patches ?? [], result.script_pipeline);
    toast.success($tt('video.longVideoPipeRepairPreviewReady', { n: result.patches?.length ?? 0 }));
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  } finally {
    pipelineRepairing.value = false;
  }
}

async function onPipelineApplyRepair(previewPipe: import('@/types').ScriptPipeline) {
  const pipe = previewPipe || pendingRepairPipeline.value;
  if (!pipe) return;
  pipelineApplyingRepair.value = true;
  try {
    const result = await api.gen.scriptParseApplyPatches(pipe);
    await applyPipelineResult(result, { mergeCharacters: true, clearShots: false });
    pendingRepairPipeline.value = null;
    briefPanelRef.value?.discardRepairPreview?.();
    toast.success($tt('video.longVideoPipeRepairApplied'));
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  } finally {
    pipelineApplyingRepair.value = false;
  }
}

function onPipelineDiscardRepair() {
  pendingRepairPipeline.value = null;
}

function onGoToCastFromScript() {
  if (activeScriptPipeline.value && !factsApproved.value) {
    toast.warning($tt('video.longVideoPipeNeedFacts'));
    return;
  }
  onEditorTabChange('cast');
}

async function runStoryboardExpandPipeline() {
  await runExpandPipeline({ mode: 'full' });
}

function shotsToExpandPayload(shotList: typeof shots.value): Array<Record<string, unknown>> {
  return shotList.map((s, i) => ({
    id: s.id,
    order: s.order ?? i,
    narrative_beat_index: s.narrative_beat_index ?? i,
    visual_prompt: s.visual_prompt,
    start_visual_prompt: s.start_visual_prompt,
    video_prompt: s.video_prompt ?? s.motion_prompt,
    motion_prompt: s.motion_prompt,
    scene_prompt: s.scene_prompt,
    segment_role: s.segment_role,
    start_frame_mode: s.start_frame_mode,
    segment_group_id: s.segment_group_id,
    segment_group_index: s.segment_group_index,
    duration_sec: s.duration_sec,
    cast_looks: s.cast_looks,
    scene_look: s.scene_look,
  }));
}

async function runExpandPipeline(opts: { mode: 'full' } | { mode: 'beat'; beatIndex: number }) {
  scriptParseError.value = '';
  scriptParseProgressPhase.value = 'beat_plan';
  scriptParseProgressMessage.value = '';
  if (opts.mode === 'beat') {
    beatRegeneratingIndex.value = opts.beatIndex;
  }

  const lv = project.value;
  if (!lv) {
    beatRegeneratingIndex.value = null;
    return;
  }

  // Pipeline path (facts approved) does not need legacy script_artifact v2.
  const usePipelineShots =
    opts.mode === 'full' && Boolean(activeScriptPipeline.value?.workflow?.facts_approved);

  const artifact = usePipelineShots ? null : buildScriptArtifactForExpand(lv);
  if (!usePipelineShots && !artifact) {
    scriptParseError.value = $tt('video.longVideoStoryboardSynopsisEmpty');
    beatRegeneratingIndex.value = null;
    return;
  }

  await ensureProjectSavedForGeneration();
  const projectId = project.value?.project_id ?? lv.project_id ?? '';
  const parseModel = lv.script_parse_llm_model?.trim() || '';
  const segmentDurationSec = lv.segment_duration_sec ?? 5;
  const parseOpts = {
    quietSuccess: true as const,
    onProgress: onParseProgress,
    onError: (msg: string, qualityIssues?: ScriptParseQualityIssue[]) => {
      scriptParseError.value = msg;
      markParseBlocked(msg, qualityIssues ?? []);
    },
  };

  if (usePipelineShots && activeScriptPipeline.value) {
    if (pipelineBusy()) {
      beatRegeneratingIndex.value = null;
      return;
    }
    const pipeResult = await scriptParseProcess(
      {
        script_text: scriptText.value.trim() || (lv as { script_text?: string }).script_text || '',
        title: (lv.chapter_title || '').trim(),
        locale: uiLocale.value,
        model: parseModel,
        long_video_project_id: projectId,
        existing_pipeline: {
          ...activeScriptPipeline.value,
          workflow: { ...activeScriptPipeline.value.workflow, facts_approved: true },
        },
        until_shots: true,
        stop_at: 'gate2',
        target_duration_sec: lv.target_duration_sec ?? 60,
        segment_duration_sec: segmentDurationSec,
        max_clip_sec: MAX_SHOT_DURATION_SEC,
      },
      {
        quietSuccess: true,
        forShots: true,
        onProgress: onParseProgress,
        onCheckpoint: persistPipelineCheckpoint,
        onError: (msg, err) => {
          handlePipelineStreamError(msg, err);
          markParseBlocked(msg, []);
        },
      },
    );
    beatRegeneratingIndex.value = null;
    scriptParseProgressPhase.value = '';
    scriptParseProgressMessage.value = '';
    if (!pipeResult?.script_pipeline) {
      if (!scriptParseError.value) {
        scriptParseError.value = $tt('video.longVideoScriptGenerateStoryboardFailedGeneric');
      }
      return;
    }
    await applyPipelineResult(pipeResult, { applyShots: true });
    toast.success($tt('video.longVideoStoryboardReady'));
    onEditorTabChange('storyboard');
    return;
  }

  if (!artifact) {
    scriptParseError.value = $tt('video.longVideoStoryboardSynopsisEmpty');
    beatRegeneratingIndex.value = null;
    return;
  }

  let expandResult: ChapterAnalyzeApiResult | null = null;
  if (opts.mode === 'full') {
    expandResult = await scriptParseExpand(
      {
        script_artifact: artifact,
        locale: uiLocale.value,
        target_duration_sec: lv.target_duration_sec ?? 60,
        segment_duration_sec: segmentDurationSec,
        max_clip_sec: MAX_SHOT_DURATION_SEC,
        long_video_project_id: projectId,
        model: parseModel,
      },
      parseOpts,
    );
  } else {
    expandResult = await scriptParseExpandBeat(
      {
        script_artifact: artifact,
        beat_index: opts.beatIndex,
        existing_shots: shotsToExpandPayload(shots.value),
        locale: uiLocale.value,
        target_duration_sec: lv.target_duration_sec ?? 60,
        segment_duration_sec: segmentDurationSec,
        max_clip_sec: MAX_SHOT_DURATION_SEC,
        long_video_project_id: projectId,
        model: parseModel,
      },
      parseOpts,
    );
  }

  beatRegeneratingIndex.value = null;
  scriptParseProgressPhase.value = '';
  scriptParseProgressMessage.value = '';

  if (!expandResult?.shots?.length) {
    if (!expandResult) {
      if (!scriptParseError.value) {
        scriptParseError.value =
          opts.mode === 'beat'
            ? $tt('video.longVideoBeatExpandFailedGeneric')
            : $tt('video.longVideoScriptGenerateStoryboardFailedGeneric');
      }
    } else {
      scriptParseError.value = $tt('video.longVideoChapterAnalyzeNoShots');
    }
    return;
  }

  await applyScriptAnalyzeResult(expandResult, { mergeCharacters: false });

  if (opts.mode === 'full') {
    toast.success($tt('video.longVideoStoryboardReady'));
    const qWarn = expandResult.quality_warnings?.length ?? 0;
    if (qWarn > 0) {
      toast.warning(
        $tt('video.longVideoParseQualityWarnings', {
          count: qWarn,
          sample: expandResult.quality_warnings?.[0] ?? '',
        }),
      );
    }
    onEditorTabChange('storyboard');
  } else {
    toast.success($tt('video.longVideoBeatExpandDone', { index: opts.beatIndex + 1 }));
  }
}

async function runBeatExpandPipeline(beatIndex: number) {
  await runExpandPipeline({ mode: 'beat', beatIndex });
}

async function onRegenerateBeat(beatIndex: number) {
  if (!scriptStepDone.value) {
    toast.warning($tt('video.longVideoStoryboardSynopsisEmpty'));
    return;
  }
  const ok = window.confirm($tt('video.longVideoBeatExpandConfirm', { index: beatIndex + 1 }));
  if (!ok) return;
  await runBeatExpandPipeline(beatIndex);
}

async function onRegenerateBeatFromGroup(beatIndex: number) {
  if (isStoryboardGenerating.value || isChapterAnalyzing.value) return;
  await onRegenerateBeat(beatIndex);
}

async function onGenerateStoryboard() {
  if (!scriptStepDone.value) {
    toast.warning($tt('video.longVideoStoryboardSynopsisEmpty'));
    return;
  }
  if (shots.value.length > 0) {
    const ok = window.confirm($tt('video.longVideoStoryboardOverwriteConfirm'));
    if (!ok) return;
  }
  await runStoryboardExpandPipeline();
}

async function onCastGoToStoryboard() {
  if (storyboardTabBlocked.value) {
    onStoryboardTabBlocked();
    return;
  }
  if (!shots.value.length && scriptStepDone.value) {
    await runStoryboardExpandPipeline();
    return;
  }
  onEditorTabChange('storyboard');
}

function clearSelection() {
  longVideoProject.setSelection(null);
}

function onSelectSegment(index: number) {
  longVideoProject.setSelection({ kind: 'segment', index });
}

function onSelectBeatGroup(groupId: string) {
  longVideoProject.setSelection({ kind: 'beat_group', groupId });
  const idx = shots.value.findIndex((s) => s.segment_group_id === groupId);
  if (idx >= 0) onSelectSegment(idx);
  requestAnimationFrame(() => {
    const el = document.querySelector(`[data-group-id="${CSS.escape(groupId)}"]`);
    el?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
  });
}

function onUpdateCharacters(characters: LongVideoCharacter[]) {
  const lv = project.value;
  if (!lv) return;
  const style = effectiveStyleAnchor.value;
  const character_anchor = syncRosterToCharacterAnchor(characters, style);
  if (lv.series_id && activeSeriesDetail.value?.id === lv.series_id) {
    const seriesChars = activeSeriesDetail.value.state.characters ?? [];
    let nextVariants = { ...(lv.character_variants ?? {}) };
    const keepIds = new Set(characters.map((c) => c.id));
    for (const id of Object.keys(nextVariants)) {
      if (!keepIds.has(id)) nextVariants = clearCharacterVariant(nextVariants, id);
    }
    for (const ch of characters) {
      const base = seriesChars.find((c) => c.id === ch.id);
      if (!base) {
        nextVariants = {
          ...nextVariants,
          [ch.id]: episodeOnlyCharacterVariant(ch),
        };
        continue;
      }
      const diff = diffCharacterAgainstSeries(base, ch, { looksAndDefaultOnly: true });
      if (diff) {
        nextVariants = upsertCharacterVariant(nextVariants, ch.id, diff);
      } else {
        nextVariants = clearCharacterVariant(nextVariants, ch.id);
      }
    }
    setProjectWithChapterSync({
      ...lv,
      characters: [],
      character_anchor,
      character_variants: nextVariants,
    });
    return;
  }
  setProjectWithChapterSync({
    ...lv,
    characters,
    character_anchor,
  });
}

function onUpdateStyleAnchor(styleAnchor: string) {
  const lv = project.value;
  if (!lv) return;
  const character_anchor = syncRosterToCharacterAnchor(effectiveCharacters.value, styleAnchor);
  if (lv.series_id && activeSeriesDetail.value?.id === lv.series_id) {
    const st = { ...activeSeriesDetail.value.state, style_anchor: styleAnchor };
    activeSeriesDetail.value = { ...activeSeriesDetail.value, state: st };
    setProjectWithChapterSync({
      ...lv,
      style_anchor: styleAnchor,
      character_anchor,
    });
    scheduleSeriesPersist();
    return;
  }
  setProjectWithChapterSync({
    ...lv,
    style_anchor: styleAnchor,
    character_anchor,
  });
}

function onUpdateVisual(index: number, value: string) {
  const lv = project.value;
  if (!lv) return;
  const scene = value.trim();
  const shotsNext = lv.shots.map((s, i) => {
    if (i !== index) return s;
    const next = { ...s, visual_prompt: scene };
    if (s.segment_role === 'face_anchor') {
      next.anchor_visual_prompt = scene;
    } else {
      next.start_visual_prompt = scene;
    }
    return next;
  });
  longVideoProject.setProject({ ...lv, shots: shotsNext });
}

function onUpdateMotion(index: number, value: string) {
  const lv = project.value;
  if (!lv) return;
  const motion = value;
  const shotsNext = lv.shots.map((s, i) =>
    i === index ? { ...s, motion_prompt: motion, video_prompt: motion } : s,
  );
  longVideoProject.setProject({ ...lv, shots: shotsNext });
}

function onUpdateCastLooks(index: number, castLooks: LongVideoShotCastLook[]) {
  const lv = project.value;
  if (!lv) return;
  const shotsNext = lv.shots.map((s, i) => (i === index ? { ...s, cast_looks: castLooks } : s));
  longVideoProject.setProject({ ...lv, shots: shotsNext });
}

function onUpdateSceneLook(index: number, sceneLook: LongVideoShotSceneLook | undefined) {
  const lv = project.value;
  if (!lv) return;
  const shotsNext = lv.shots.map((s, i) => {
    if (i !== index) return s;
    const next = { ...s };
    if (sceneLook) next.scene_look = sceneLook;
    else delete next.scene_look;
    return next;
  });
  longVideoProject.setProject({ ...lv, shots: shotsNext });
}

function onUpdateDuration(index: number, value: number) {
  const lv = project.value;
  if (!lv) return;
  const sec = Math.max(1, Number(value) || 5);
  const shotsNext = lv.shots.map((s, i) => (i === index ? { ...s, duration_sec: sec } : s));
  longVideoProject.setProject({ ...lv, shots: shotsNext });
}

function onUpdateChainMode(index: number, value: LongVideoChainMode) {
  const lv = project.value;
  if (!lv) return;
  const shotsNext = lv.shots.map((s, i) => {
    if (i !== index) return s;
    const stored = value === lv.chain_mode ? undefined : value;
    if (stored === s.chain_mode) return s;
    return invalidateSegmentAsset({ ...s, chain_mode: stored });
  });
  longVideoProject.setProject({ ...lv, shots: shotsNext });
}

async function onPolishMotion(index: number) {
  const lv = project.value;
  if (!lv) return;
  const shot = lv.shots[index];
  const nextShot = lv.shots[index + 1];
  if (!shot) return;

  const left = shotKeyframeText(shot);
  const right = nextShot ? shotKeyframeText(nextShot) : '';
  const motion = shotVideoPrompt(shot);
  const seed = motion || [left, right].filter(Boolean).join('\n\n');
  if (!seed) {
    toast.warning($tt('video.longVideoPolishNeedContext'));
    return;
  }

  const contextParts = [
    left ? `${$tt('video.longVideoPolishFrom')}: ${left}` : '',
    right ? `${$tt('video.longVideoPolishTo')}: ${right}` : '',
    motion ? `${$tt('video.longVideoMotionPrompt')}: ${motion}` : '',
  ].filter(Boolean);

  const prompt = contextParts.length > 1 ? contextParts.join('\n') : seed;
  motionPolishing.value = true;
  try {
    const enhanced = await enhance(prompt, undefined, 'video_create', lv.segment_video_model);
    if (enhanced) onUpdateMotion(index, enhanced);
  } finally {
    motionPolishing.value = false;
  }
}

async function onPolishVisual(index: number) {
  const lv = project.value;
  if (!lv) return;
  const shot = lv.shots[index];
  const prompt = shot ? shotKeyframeText(shot) : '';
  if (!prompt.trim()) return;

  visualPolishing.value = true;
  try {
    const enhanced = await enhance(prompt, undefined, 'image_create', lv.keyframe_model);
    if (enhanced) onUpdateVisual(index, enhanced);
  } finally {
    visualPolishing.value = false;
  }
}

function openFinalInGallery() {
  const id = project.value?.final_asset_id;
  if (!id) return;
  window.open(`/api/assets/${encodeURIComponent(id)}/file`, '_blank', 'noopener,noreferrer');
}

function onAddKeyframe() {
  const lv =
    project.value ||
    defaultLongVideoProject({
      keyframe_model: defaultKeyframeModel.value,
      segment_video_model: defaultSegmentModel.value,
      title: projectTitle.value,
    });
  const id = nextShotId(lv.shots);
  const shotsNext = [...lv.shots, createEmptyShot(lv.shots.length, id)];
  longVideoProject.setProject({
    ...lv,
    shots: shotsNext,
    selection: { kind: 'segment', index: shotsNext.length - 1 },
  });
}

function onInsertKeyframeBefore(index: number) {
  const lv = project.value;
  if (!lv) return;
  const id = nextShotId(lv.shots);
  const { shots: shotsNext, newIndex } = insertKeyframeBefore(lv.shots, index, id);
  longVideoProject.setProject({
    ...lv,
    shots: shotsNext,
    selection: { kind: 'segment', index: newIndex },
  });
}

function onInsertKeyframeAfter(index: number) {
  const lv = project.value;
  if (!lv) return;
  const id = nextShotId(lv.shots);
  const { shots: shotsNext, newIndex } = insertKeyframeAfter(lv.shots, index, id);
  longVideoProject.setProject({
    ...lv,
    shots: shotsNext,
    selection: { kind: 'segment', index: newIndex },
  });
}

async function onRemoveKeyframe(index: number) {
  const lv = project.value;
  if (!lv) return;
  if (lv.shots.length <= MIN_LONG_VIDEO_KEYFRAMES) {
    toast.warning($tt('video.longVideoRemoveKeyframeMin'));
    return;
  }

  const n = index + 1;
  const isFirst = index === 0;
  const isLast = index === lv.shots.length - 1;

  if (removeKeyframeNeedsConfirm(lv.shots, index)) {
    let body: string;
    if (isFirst) {
      body = $tt('video.longVideoRemoveKeyframeConfirmFirst', { n });
    } else if (isLast) {
      body = $tt('video.longVideoRemoveKeyframeConfirmLast', { n });
    } else {
      body = $tt('video.longVideoRemoveKeyframeConfirmMiddle', {
        n,
        from: index,
        to: index + 2,
      });
    }
    try {
      await confirm(body, $tt('video.longVideoRemoveKeyframeConfirmTitle'), { type: 'warning' });
    } catch {
      return;
    }
  }

  const result = removeKeyframeAt(lv.shots, index);
  if (!result) return;
  longVideoProject.setProject({
    ...lv,
    shots: result.shots,
    selection: result.selection,
  });
}

function onClearKeyframe(index: number) {
  const lv = project.value;
  if (!lv) return;
  const shotsNext = lv.shots.map((s, i) =>
    i === index ? { ...s, keyframe_asset_id: undefined, status: 'draft' as const } : s,
  );
  longVideoProject.setProject({ ...lv, shots: shotsNext });
}

function onClearSegment(index: number) {
  const lv = project.value;
  if (!lv) return;
  const shotsNext = lv.shots.map((s, i) => {
    if (i !== index) return s;
    const next = { ...s, segment_asset_id: undefined };
    if (next.status === 'segment_ready') {
      next.status = next.keyframe_asset_id ? ('keyframe_ready' as const) : ('draft' as const);
    }
    return next;
  });
  longVideoProject.setProject({ ...lv, shots: shotsNext });
}

function onPickKeyframeGallery(index: number) {
  keyframePickIndex.value = index;
  showKeyframePicker.value = true;
}

function onKeyframeAssetPick(payload: { path: string }) {
  const index = keyframePickIndex.value;
  showKeyframePicker.value = false;
  keyframePickIndex.value = null;
  if (index == null) return;
  const assetId = assetIdFromGalleryPath(payload.path);
  if (!assetId) return;
  const lv = project.value;
  if (!lv) return;
  const shotsNext = lv.shots.map((s, i) =>
    i === index ? { ...s, keyframe_asset_id: assetId, status: 'keyframe_ready' as const } : s,
  );
  longVideoProject.setProject({ ...lv, shots: shotsNext });
}

function waitForImageTask(tid: string): Promise<{ assetId: string | null }> {
  return new Promise((resolve, reject) => {
    let assetId: string | null = null;
    api.gen.streamMediaTask(tid, {
      onResult: (resultData: unknown) => {
        const data = resultData as { asset_ids?: string[] };
        const ids = data?.asset_ids || [];
        if (ids.length > 0) assetId = ids[0];
      },
      onDone: (doneData: unknown) => {
        tasksStore.unregisterPageOwnedStream(tid);
        const data = doneData as { status: string };
        if (data.status === 'completed') resolve({ assetId });
        else reject(new Error('keyframe generation failed'));
      },
      onError: () => {
        tasksStore.unregisterPageOwnedStream(tid);
        reject(new Error('connection lost'));
      },
    });
  });
}

async function onGenerateKeyframe(index: number) {
  const lv = project.value;
  if (!lv) return;
  const shot = lv.shots[index];
  const keyframeText = shotKeyframeText(shot);
  if (!shotNeedsKeyframe(shot)) {
    toast.warning($tt('video.longVideoSegmentTailFrameNoKeyframe'));
    return;
  }
  if (!keyframeText.trim()) {
    toast.warning($tt('studio.enterPrompt'));
    return;
  }
  if (composeMode.value === 'img2img') {
    toast.error($tt('video.longVideoKeyframeImg2imgUnsupported'));
    return;
  }

  const scene = shotCastMatchText(shot);
  const snap = workspaceSnapshot.value;
  const chars = snap?.characters ?? effectiveCharacters.value;
  const scenes = snap?.scenes ?? effectiveScenes.value;
  const castScope = castReferenceScope(
    shot.first_frame_visibility,
    shot.segment_role,
    shot.is_establishing_empty,
  );
  const refBindings = buildKeyframeReferenceBindings(
    chars,
    shot.cast_looks ?? [],
    scene,
    castScope,
    shot.scene_look,
    scenes,
  );
  const keyframeModelRow = registryStore.registry?.models?.[keyframeModelId.value] as
    | Record<string, unknown>
    | undefined;
  const refErr = validateReferenceRequirement(refBindings.length, keyframeModelRow);
  if (refErr) {
    toast.error(refErr);
    return;
  }
  const { ids: castRefIds, capped: refsCapped } = capReferenceAssetIds(
    refBindings.map((b) => b.asset_id),
    keyframeModelRow,
  );
  if (refsCapped) {
    toast.warning($tt('video.longVideoReferenceCapApplied'));
  }

  const p = keyframeComposeParams;
  const modelStr = keyframeModelId.value;
  const adapters = mergeKeyframeLoraAdapters(
    p.lora ? [{ id: String(p.lora), weight: Number(p.lora_scale) || 0.8 }] : [],
    chars,
    shot.cast_looks ?? [],
    scene,
    lv.character_lora_id,
    Number(p.lora_scale) || 0.8,
  );

  let control_asset_id: string | null = null;
  if (p.controlnet) {
    try {
      control_asset_id = resolveControlAssetId(keyframeControlImage.value, {
        assetRequired: $tt('canvas.controlImageAssetRequired'),
        required: $tt('canvas.controlImageRequired'),
      });
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      toast.warning(msg);
      return;
    }
  }

  const locale = uiLocale.value.startsWith('zh') ? 'zh' : 'en';
  const castNegative = locale === 'zh' ? KEYFRAME_CAST_NEGATIVE_ZH : KEYFRAME_CAST_NEGATIVE_EN;
  const negativePrompt = [p.negative_prompt, castRefIds.length ? castNegative : '']
    .filter(Boolean)
    .join(', ');

  const meta = withProjectMetadata({
    long_video_shot_id: shot.id,
    long_video_phase: 'keyframe',
  });
  if (snap?.revisions && Object.keys(snap.revisions).length) {
    meta.long_video_asset_revisions = JSON.stringify(snap.revisions);
  }
  if (shot.cast_looks?.length) {
    meta.long_video_cast_looks = JSON.stringify(shot.cast_looks);
  }
  if (shot.scene_look) {
    meta.long_video_scene_look = JSON.stringify(shot.scene_look);
  }
  if (refBindings.length) {
    meta.long_video_reference_bindings = serializeReferenceBindings(
      refBindings.slice(0, castRefIds.length),
    );
  }
  if (shot.first_frame_strategy === 't2i_from_grounding' || shot.first_frame_strategy === 'scene_composite') {
    const grounded = resolveSceneForShot(lv, shot, scenes);
    Object.assign(meta, buildKeyframeGroundingMetadata(shot, grounded));
  }
  if (p.scheduler) meta.scheduler = p.scheduler;

  const inpSrc = resolveInpaintAssetId(keyframeInpaintSourceImage.value);
  const inpMsk = resolveInpaintAssetId(keyframeInpaintMaskImage.value);
  const enhCommon = {
    parameters: keyframeModelConfig.value?.parameters as Record<string, unknown> | undefined,
    params: p,
    controlnet: String(p.controlnet || ''),
    controlAssetId: control_asset_id,
    controlnetStrength: Number(p.controlnet_strength) || 0.8,
    inpaintSourceId: inpSrc,
    inpaintMaskId: inpMsk,
  };

  const seedNum = p.seed ? parseInt(String(p.seed), 10) : null;
  const { width, height } = outputSizePixels.value;
  const t2iPrompt = keyframeGenerationPrompt(keyframeText, keyframeContextForShot(shot));

  keyframeGeneratingIndex.value = index;
  try {
    await ensureProjectSavedForGeneration();
    const genBody: Record<string, unknown> = {
      model: modelStr,
      prompt: t2iPrompt,
      negative_prompt: negativePrompt || '',
      size: `${width}x${height}`,
      n: 1,
      steps: p.steps,
      guidance: p.guidance,
      seed: seedNum,
      adapters,
      metadata: meta,
      priority: 'normal',
    };
    if (castRefIds.length) {
      genBody.reference_asset_ids = castRefIds;
    }
    const genEnh = appendZImageEnhancementFields(genBody, enhCommon);
    if (!genEnh.ok) throw new Error($tt('create.inpaintPairRequired'));
    const submitRes = await api.gen.createImageGeneration(genBody);

    const tid = taskIdFromSubmitResponse(submitRes);
    if (!tid) throw new Error('no task id');
    tasksStore.registerPageOwnedStream(tid);
    const { assetId } = await waitForImageTask(tid);
    if (!assetId) throw new Error('no asset');
    const latest = project.value ?? lv;
    const shotsNext = latest.shots.map((s, i) =>
      i === index ? { ...s, keyframe_asset_id: assetId, status: 'keyframe_ready' as const } : s,
    );
    longVideoProject.setProject({ ...latest, shots: shotsNext });
    const nextWarnings = { ...consistencyWarnings.value };
    delete nextWarnings[index];
    consistencyWarnings.value = nextWarnings;
    try {
      await loadGallery(true);
    } catch {
      /* gallery refresh must not fail the generation flow */
    }
    toast.success($tt('video.longVideoKeyframeDone'));
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.genFailed', { msg }));
  } finally {
    if (keyframeGeneratingIndex.value === index) {
      keyframeGeneratingIndex.value = null;
    }
  }
}

function markSegmentPending(index: number) {
  const next = new Set(segmentGeneratingIndices.value);
  next.add(index);
  segmentGeneratingIndices.value = next;
}

function unmarkSegmentPending(index: number) {
  const next = new Set(segmentGeneratingIndices.value);
  next.delete(index);
  segmentGeneratingIndices.value = next;
}

function applySegmentGenerationResult(index: number, shotId: string, assetId: string) {
  const lv = project.value;
  if (!lv?.shots[index] || lv.shots[index].id !== shotId) return;
  const shotsNext = lv.shots.map((s, i) =>
    i === index
      ? { ...s, segment_asset_id: assetId, status: 'segment_ready' as const, error: undefined }
      : s,
  );
  longVideoProject.setProject({ ...lv, shots: shotsNext });
  void extractAndBindPrevTailFrame(index, shotId, assetId);
}

/** Extract real last frame from completed segment for last_frame chain continuity. */
async function extractAndBindPrevTailFrame(
  index: number,
  shotId: string,
  segmentAssetId: string,
) {
  try {
    const frame = await api.gen.extractLastFrame(segmentAssetId);
    const frameId = String((frame as { id?: string }).id || '').trim();
    if (!frameId) return;
    const lv = project.value;
    if (!lv?.shots[index] || lv.shots[index].id !== shotId) return;
    const shotsNext = lv.shots.map((s, i) => {
      if (i === index) {
        return { ...s, end_frame_asset_id: frameId };
      }
      if (i === index + 1) {
        const nextMode = effectiveShotChainMode(s, lv.chain_mode);
        if (nextMode === 'last_frame' || s.start_frame_mode === 'prev_segment_tail') {
          return {
            ...s,
            start_frame_asset_id: frameId,
            prev_tail_frame_asset_id: frameId,
          };
        }
      }
      return s;
    });
    longVideoProject.setProject({ ...lv, shots: shotsNext });
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.warning($tt('video.longVideoTailFrameExtractFailed', { msg }));
  }
}

function watchSegmentTaskCompletion(taskId: string, index: number, shotId: string) {
  let assetId: string | null = null;
  tasksStore.openTaskLogStream(taskId, {
    onResult: (resultData: unknown) => {
      const ids = (resultData as { asset_ids?: string[] })?.asset_ids || [];
      if (ids.length > 0) {
        assetId = ids[0];
        applySegmentGenerationResult(index, shotId, assetId);
      }
    },
    onDone: async (doneData: unknown) => {
      unmarkSegmentPending(index);
      const data = doneData as { status: string };
      if (data.status === 'completed') {
        if (!assetId) {
          try {
            const task = (await api.gen.getMediaTask(taskId)) as { asset_ids?: string[] };
            const ids = task?.asset_ids || [];
            if (ids[0]) applySegmentGenerationResult(index, shotId, ids[0]);
          } catch {
            /* optional fallback */
          }
        }
        try {
          await loadGallery(true);
        } catch {
          /* gallery refresh must not fail the generation flow */
        }
        toast.success($tt('video.longVideoSegmentDone'));
      } else {
        toast.error($tt('studio.genFailed', { msg: '' }));
      }
    },
    onError: () => {
      unmarkSegmentPending(index);
      toast.error($tt('studio.genFailed', { msg: '' }));
    },
  });
}

function buildSegmentSubmitBody(
  lv: LongVideoProjectState,
  index: number,
): Record<string, unknown> | null {
  const lvProd = projectWithEffectiveRoster(lv);
  const shot = lvProd.shots[index];
  if (!shot || !canGenerateSegmentShot(lvProd, index)) return null;
  const sourceAssetId = segmentI2vSourceAssetId(lvProd, index);
  if (!sourceAssetId) return null;
  const motion = shotVideoPrompt(shot);
  if (!motion) return null;

  const chainMode = effectiveShotChainMode(shot, lvProd.chain_mode);
  const prevSegId = index > 0 ? lvProd.shots[index - 1]?.segment_asset_id : undefined;
  const endFrameId = segmentI2vEndFrameAssetId(lvProd, index);
  const segModel = lv.segment_video_model;
  const segRefBindings = buildSegmentReferenceBindings(lvProd, index, lvProd.characters ?? []);
  const identityRefCount = segRefBindings.filter((b) => b.purpose === 'identity').length;
  const segModelRow = registryStore.registry?.models?.[segModel] as Record<string, unknown> | undefined;
  if (chainMode === 'reference_r2v') {
    const refReqErr = validateReferenceRequirement(identityRefCount || 1, segModelRow);
    if (refReqErr) return null;
  }
  const p = segmentComposeParams;
  const fps = p.fps || scalarDefault(segModel, 'fps', 16);
  const nfSchema = normalizeParamsDef(modelParameters(segModel)).num_frames as
    | { min?: number; max?: number; step?: number }
    | undefined;
  const cappedDurationSec = Math.min(shotDurationSec(shot), MAX_SHOT_DURATION_SEC);
  const numFrames = numFramesForDurationSec(cappedDurationSec, fps, nfSchema);
  const { width, height } = outputSizePixels.value;
  const body: Record<string, unknown> = {
    model: segModel,
    operation: 'animate',
    source_asset_id: sourceAssetId,
    prompt: motion,
    negative_prompt: p.negative_prompt || '',
    size: `${width}x${height}`,
    num_frames: numFrames,
    fps,
    steps: p.steps || scalarDefault(segModel, 'steps', 40),
    guidance: p.guide_scale ?? scalarDefault(segModel, 'guide_scale', 3.0),
    shift: p.shift || scalarDefault(segModel, 'shift', 12.0) || undefined,
    seed: shot.seed ?? (p.seed ? parseInt(p.seed, 10) : null),
    priority: 'normal',
    metadata: withProjectMetadata({
      long_video_shot_id: shot.id,
      long_video_phase: 'segment',
      long_video_chain_mode: chainMode,
      ...(segRefBindings.length
        ? { long_video_reference_bindings: serializeReferenceBindings(segRefBindings) }
        : {}),
      ...(chainMode === 'last_frame' && prevSegId
        ? { long_video_prev_segment_asset_id: prevSegId }
        : {}),
    }),
  };
  const adapters = buildSegmentAdapters();
  if (adapters.length > 0) {
    body.adapters = adapters;
  }
  if (chainMode === 'first_last' && endFrameId) {
    body.tail_asset_id = endFrameId;
    body.end_asset_id = endFrameId;
  }
  if (modelHasBerniniRenderer(segModel) || chainMode === 'reference_r2v') {
    const refIds = segRefBindings
      .filter((b) => b.purpose === 'identity')
      .map((b) => b.asset_id);
    const { ids: cappedRefs } = capReferenceAssetIds(refIds, segModelRow);
    if (cappedRefs.length) {
      body.reference_asset_ids = cappedRefs;
    } else if (chainMode === 'reference_r2v') {
      return null;
    }
  }
  appendActiveEnumFields(
    body,
    p as Record<string, unknown>,
    VIDEO_INFERENCE_ENUM_KEYS,
    normalizeParamsDef(modelParameters(segModel)) as Record<string, unknown> | undefined,
  );
  return body;
}

async function onGenerateSegment(index: number) {
  const lv = project.value;
  if (!lv) return;
  const shot = lv.shots[index];
  if (!shot) return;

  const lvProd = projectWithEffectiveRoster(lv);
  const chainMode = effectiveShotChainMode(shot, lvProd.chain_mode);
  const prevSegId = index > 0 ? lvProd.shots[index - 1]?.segment_asset_id : undefined;
  if (chainMode === 'last_frame' && index > 0 && !prevSegId) {
    toast.warning($tt('video.longVideoSegmentChainNeedPrevSegment'));
    return;
  }
  if (chainMode === 'last_frame' && index > 0 && prevSegId && !segmentI2vSourceAssetId(lvProd, index)) {
    toast.warning($tt('video.longVideoSegmentChainNeedTailFrame'));
    return;
  }
  if (!canGenerateSegmentShot(lvProd, index)) {
    if (!shotVideoPrompt(shot)) {
      toast.warning($tt('video.longVideoNeedMotionForSegment'));
    } else if (shot.start_frame_mode === 'anchor_link' || shot.segment_role === 'post_anchor') {
      toast.warning($tt('video.longVideoNeedAnchorForSegment'));
    } else {
      toast.warning($tt('video.longVideoNeedKeyframeForSegment'));
    }
    return;
  }
  if (segmentGeneratingIndices.value.has(index)) {
    toast.warning($tt('video.longVideoSegmentAlreadyQueued'));
    return;
  }

  const segModelRow = registryStore.registry?.models?.[lvProd.segment_video_model] as
    | Record<string, unknown>
    | undefined;
  const segSupportErr = assertProviderSupports(
    chainModeToCapabilityMode(chainMode, 'segment'),
    segModelRow,
  );
  if (segSupportErr) {
    toast.error(segSupportErr);
    return;
  }

  await ensureProjectSavedForGeneration();
  const lvAfterSave = project.value ?? lv;
  const body = buildSegmentSubmitBody(lvAfterSave, index);
  if (!body) {
    if (chainMode === 'reference_r2v') {
      const refErr = validateReferenceRequirement(1, segModelRow);
      if (refErr) toast.error(refErr);
    }
    return;
  }

  const shotId = shot.id;
  markSegmentPending(index);
  try {
    const submitRes = await api.gen.createVideoEdit(body);
    const tid = taskIdFromSubmitResponse(submitRes);
    if (!tid) throw new Error('no task id');
    watchSegmentTaskCompletion(tid, index, shotId);
    tasksStore.pollQueueOnce();
    openGlobalTaskQueue();
    toast.success($tt('video.longVideoSegmentQueued'));
  } catch (e: unknown) {
    unmarkSegmentPending(index);
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.genFailed', { msg }));
  }
}

function waitForVideoTask(tid: string): Promise<{ assetId: string | null }> {
  return new Promise((resolve, reject) => {
    let assetId: string | null = null;
    api.gen.streamMediaTask(tid, {
      onResult: (resultData: unknown) => {
        const ids = (resultData as { asset_ids?: string[] })?.asset_ids || [];
        if (ids.length > 0) assetId = ids[0];
      },
      onDone: async (doneData: unknown) => {
        tasksStore.unregisterPageOwnedStream(tid);
        const data = doneData as { status: string };
        if (data.status === 'completed') {
          if (!assetId) {
            try {
              const task = (await api.gen.getMediaTask(tid)) as { asset_ids?: string[] };
              if (task?.asset_ids?.[0]) assetId = task.asset_ids[0];
            } catch {
              /* optional fallback */
            }
          }
          resolve({ assetId });
        } else {
          reject(new Error('segment generation failed'));
        }
      },
      onError: () => {
        tasksStore.unregisterPageOwnedStream(tid);
        reject(new Error('connection lost'));
      },
    });
  });
}

async function generateSegmentAndWait(
  index: number,
  opts: { openQueue?: boolean; toastOnDone?: boolean } = {},
): Promise<boolean> {
  const lv = project.value;
  if (!lv) return false;
  const shot = lv.shots[index];
  if (!shot) return false;

  const lvProd = projectWithEffectiveRoster(lv);
  const chainMode = effectiveShotChainMode(shot, lvProd.chain_mode);
  const prevSegId = index > 0 ? lvProd.shots[index - 1]?.segment_asset_id : undefined;
  if (chainMode === 'last_frame' && index > 0 && !prevSegId) {
    toast.warning($tt('video.longVideoSegmentChainNeedPrevSegment'));
    return false;
  }
  if (chainMode === 'last_frame' && index > 0 && prevSegId && !segmentI2vSourceAssetId(lvProd, index)) {
    toast.warning($tt('video.longVideoSegmentChainNeedTailFrame'));
    return false;
  }
  if (!canGenerateSegmentShot(lvProd, index)) {
    if (!shotVideoPrompt(shot)) {
      toast.warning($tt('video.longVideoNeedMotionForSegment'));
    } else if (
      shot.start_frame_mode === 'anchor_link' ||
      shot.segment_role === 'post_anchor'
    ) {
      toast.warning($tt('video.longVideoNeedAnchorForSegment'));
    } else {
      toast.warning($tt('video.longVideoNeedKeyframeForSegment'));
    }
    return false;
  }
  if (segmentGeneratingIndices.value.has(index)) {
    toast.warning($tt('video.longVideoSegmentAlreadyQueued'));
    return false;
  }

  const segModelRowBatch = registryStore.registry?.models?.[lvProd.segment_video_model] as
    | Record<string, unknown>
    | undefined;
  const segSupportErrBatch = assertProviderSupports(
    chainModeToCapabilityMode(chainMode, 'segment'),
    segModelRowBatch,
  );
  if (segSupportErrBatch) {
    if (opts.toastOnDone !== false) toast.error(segSupportErrBatch);
    return false;
  }

  await ensureProjectSavedForGeneration();
  const lvAfterSave = project.value ?? lv;
  const body = buildSegmentSubmitBody(lvAfterSave, index);
  if (!body) return false;

  const shotId = shot.id;
  markSegmentPending(index);
  try {
    const submitRes = await api.gen.createVideoEdit(body);
    const tid = taskIdFromSubmitResponse(submitRes);
    if (!tid) throw new Error('no task id');
    tasksStore.registerPageOwnedStream(tid);
    if (opts.openQueue) {
      tasksStore.pollQueueOnce();
      openGlobalTaskQueue();
    }
    const { assetId } = await waitForVideoTask(tid);
    unmarkSegmentPending(index);
    if (assetId) {
      applySegmentGenerationResult(index, shotId, assetId);
      try {
        await loadGallery(true);
      } catch {
        /* gallery refresh must not fail the generation flow */
      }
      if (opts.toastOnDone) toast.success($tt('video.longVideoSegmentDone'));
      return true;
    }
    toast.error($tt('studio.genFailed', { msg: '' }));
    return false;
  } catch (e: unknown) {
    unmarkSegmentPending(index);
    const msg = e instanceof Error ? e.message : String(e);
    if (opts.toastOnDone || opts.openQueue) toast.error($tt('studio.genFailed', { msg }));
    return false;
  }
}

async function runBatchPlan(keyframeIndices: number[], segmentIndices: number[]) {
  if (batchGroupGenerating.value) return;
  if (toastProductionBlockers()) return;
  if (!keyframeIndices.length && !segmentIndices.length) {
    toast.info($tt('video.longVideoBatchNothingToDo'));
    return;
  }
  batchGroupGenerating.value = true;
  openGlobalTaskQueue();
  let ok = 0;
  let fail = 0;
  try {
    for (const idx of keyframeIndices) {
      await onGenerateKeyframe(idx);
      if (project.value?.shots[idx]?.keyframe_asset_id) ok += 1;
      else fail += 1;
    }
    for (const idx of segmentIndices) {
      const success = await generateSegmentAndWait(idx, { openQueue: false, toastOnDone: false });
      if (success) ok += 1;
      else fail += 1;
    }
    if (fail === 0) {
      toast.success($tt('video.batchSubmitted', { count: ok }));
    } else {
      toast.warning($tt('video.longVideoBatchPartial', { ok, fail }));
    }
  } finally {
    batchGroupGenerating.value = false;
    await loadGallery(true);
  }
}

async function onBatchGenerateCurrentGroup() {
  const lv = project.value;
  if (!lv) return;
  const groupId = selectedBeatGroupId(lv.shots, selection.value);
  if (!groupId) {
    toast.warning($tt('video.longVideoBatchSelectGroup'));
    return;
  }
  const group = groupShotsByBeat(lv.shots).find((g) => g.groupId === groupId);
  if (!group) return;
  const plan = planGroupGeneration(lv, group);
  await runBatchPlan(plan.keyframeIndices, plan.segmentIndices);
}

async function onBatchGenerateAllAnchors() {
  const lv = project.value;
  if (!lv) return;
  const indices = allPendingAnchorKeyframeIndices(lv.shots);
  await runBatchPlan(indices, []);
}

async function onBatchGenerateAllSegments() {
  const lv = project.value;
  if (!lv) return;
  const indices = allPendingSegmentIndices(lv);
  await runBatchPlan([], indices);
}

function onInsertFaceAnchor(groupId: string) {
  const lv = project.value;
  if (!lv) return;
  if (groupHasFaceAnchor(lv.shots, groupId)) {
    toast.warning($tt('video.longVideoAnchorAlreadyExists'));
    return;
  }
  const shotsNext = insertFaceAnchorIntoGroup(lv.shots, groupId);
  if (shotsNext === lv.shots) {
    toast.warning($tt('video.longVideoInsertAnchorFailed'));
    return;
  }
  const firstIdx = shotsNext.findIndex((s) => s.segment_group_id === groupId);
  longVideoProject.setProject({
    ...lv,
    shots: shotsNext,
    selection: firstIdx >= 0 ? { kind: 'segment', index: firstIdx } : lv.selection,
  });
  toast.success($tt('video.longVideoInsertAnchorDone'));
}

async function onResplitBeatGroup(groupId: string) {
  const lv = project.value;
  if (!lv) return;
  try {
    await confirm($tt('video.longVideoResplitBeatConfirm'), $tt('video.longVideoResplitBeat'), {
      type: 'warning',
    });
  } catch {
    return;
  }
  const shotsNext = resplitBeatGroupRule(lv.shots, groupId);
  const firstIdx = shotsNext.findIndex((s) => s.segment_group_id === groupId);
  longVideoProject.setProject({
    ...lv,
    shots: shotsNext,
    selection: firstIdx >= 0 ? { kind: 'segment', index: firstIdx } : null,
  });
  toast.success($tt('video.longVideoResplitBeatDone'));
}

async function loadSeriesDetail(seriesId: string) {
  if (!seriesId) {
    activeSeriesDetail.value = null;
    return;
  }
  try {
    const detail = await loadSeriesWithLibraryMigration(
      (id) => api.longVideo.getSeries(id),
      (id, body) => api.longVideo.updateSeries(id, body),
      seriesId,
      api.library,
    );
    activeSeriesDetail.value = {
      ...detail,
      state: defaultSeriesState(detail.state ?? {}),
    };
    setItem(DQ_STORAGE.LONG_VIDEO_SERIES_ID, seriesId);
  } catch (e: unknown) {
    activeSeriesDetail.value = null;
    if (isRevisionConflictError(e)) {
      toast.error($tt('video.longVideoRevisionConflict'));
    } else {
      toast.warning($tt('video.longVideoSeriesLoadFailed'));
    }
  }
}

function scheduleSeriesPersist() {
  if (seriesPersistTimer) clearTimeout(seriesPersistTimer);
  seriesPersistTimer = setTimeout(() => {
    seriesPersistTimer = null;
    void persistActiveSeries();
  }, 800);
}

async function persistActiveSeries(): Promise<boolean> {
  const detail = activeSeriesDetail.value;
  if (!detail) return false;
  try {
    const saved = await api.longVideo.updateSeries(detail.id, {
      title: detail.title,
      state: detail.state,
    });
    if (activeSeriesDetail.value?.id === saved.id) {
      activeSeriesDetail.value = {
        ...saved,
        state: defaultSeriesState(saved.state ?? {}),
      };
    }
    return true;
  } catch (e: unknown) {
    if (isRevisionConflictError(e)) {
      toast.error($tt('video.longVideoRevisionConflict'));
      return false;
    }
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
    return false;
  }
}

async function flushBeforeSwitch(): Promise<void> {
  if (seriesPersistTimer) {
    clearTimeout(seriesPersistTimer);
    seriesPersistTimer = null;
    await persistActiveSeries();
  }
  await flushAutoSave();
}

function applyServerProject(
  detail: { id: string; title: string; state: Partial<LongVideoProjectState> },
  opts?: { silent?: boolean },
) {
  runWithAutoSaveSuppressed(() => {
    const base = hydrateCharacterRoster(
      defaultLongVideoProject(detail.state),
      uiLocale.value.startsWith('zh') ? 'zh' : 'en',
    );
    setProjectWithChapterSync({
      ...base,
      project_id: detail.id,
      title: detail.title || detail.state.title,
      series_id: detail.state.series_id,
      episode_number: detail.state.episode_number,
      revision: detail.state.revision,
      character_variants: detail.state.character_variants ?? {},
      scene_variants: detail.state.scene_variants ?? {},
      selection: null,
    });
  });
  syncOutputSizeForSegmentModel(detail.state.segment_video_model ?? defaultSegmentModel.value);
  resetKeyframeComposeDefaults();
  resetSegmentComposeDefaults();
  referenceImage.value = null;
  keyframeControlImage.value = null;
  keyframeInpaintSourceImage.value = null;
  keyframeInpaintMaskImage.value = null;
  consistencyWarnings.value = {};
  if (!opts?.silent) {
    toast.success($tt('video.longVideoProjectLoaded'));
  }
}

function runWithAutoSaveSuppressed(fn: () => void) {
  suppressAutoSave.value += 1;
  fn();
  void nextTick(() => {
    suppressAutoSave.value = Math.max(0, suppressAutoSave.value - 1);
  });
}

async function persistProjectToServer(opts: { silent?: boolean } = {}): Promise<boolean> {
  const lv = project.value;
  if (!lv) return false;
  if (!lv.project_id && !longVideoHasPersistableContent(lv)) {
    if (!opts.silent) {
      toast.warning($tt('video.longVideoSaveNothing'));
    }
    return false;
  }
  if (savingProject.value) return false;
  savingProject.value = true;
  try {
    const title = lv.title?.trim() || $tt('video.longVideoPageTitle');
    const state = projectStateForServer({ ...lv, title });
    if (lv.project_id) {
      const saved = await api.longVideo.updateProject(lv.project_id, { title, state });
      runWithAutoSaveSuppressed(() => {
        patchProjectField('title', saved.title);
        if (typeof saved.state?.revision === 'number') {
          patchProjectField('revision', saved.state.revision);
        }
      });
    } else if (lv.series_id) {
      const { project: created } = await api.longVideo.createSeriesEpisode(lv.series_id, {
        title,
        state,
      });
      runWithAutoSaveSuppressed(() => {
        longVideoProject.patchProject({ project_id: created.id, title: created.title });
      });
      if (!opts.silent) {
        toast.success($tt('video.longVideoProjectCreated'));
      }
    } else {
      toast.warning($tt('video.longVideoStandaloneRemoved'));
      return false;
    }
    longVideoProject.persistNow();
    refreshWorkspaceNav();
    if (!opts.silent && lv.project_id) {
      toast.success($tt('video.longVideoSaveProjectOk'));
    }
    return true;
  } catch (e: unknown) {
    if (isRevisionConflictError(e)) {
      if (!opts.silent) toast.error($tt('video.longVideoRevisionConflict'));
      return false;
    }
    if (!opts.silent) {
      const msg = e instanceof Error ? e.message : String(e);
      toast.error($tt('studio.error', { msg }));
    }
    return false;
  } finally {
    savingProject.value = false;
  }
}

const {
  status: autoSaveStatus,
  cancelPending: cancelAutoSavePending,
  flush: flushAutoSave,
} = useLongVideoAutoSave({
  project,
  suppress: suppressAutoSave,
  hasPersistableContent: longVideoHasPersistableContent,
  save: persistProjectToServer,
});

const autoSaveLabel = computed(() => {
  switch (autoSaveStatus.value) {
    case 'pending':
      return $tt('video.longVideoAutoSavePending');
    case 'saving':
      return $tt('video.longVideoAutoSaveSaving');
    case 'saved':
      return $tt('video.longVideoAutoSaveSaved');
    case 'error':
      return $tt('video.longVideoAutoSaveError');
    default:
      return '';
  }
});

async function loadSavedProject(projectId: string, opts?: { silent?: boolean }) {
  if (project.value?.project_id && project.value.project_id !== projectId) {
    await flushBeforeSwitch();
  }
  try {
    const detail = await api.longVideo.getProject(projectId);
    const sid = detail.state?.series_id;
    if (!sid) {
      // Standalone episodes are no longer supported — only project → episode.
      toast.warning($tt('video.longVideoStandaloneRemoved'));
      await router.replace({ name: 'long_video_projects' });
      return;
    }
    applyServerProject(detail, opts);
    await loadSeriesDetail(sid);
  } catch (e: unknown) {
    if (!opts?.silent) {
      toast.warning($tt('video.longVideoProjectLoadFailed'));
    }
    const msg = e instanceof Error ? e.message : String(e);
    if (!opts?.silent) toast.error($tt('studio.error', { msg }));
    throw e;
  }
}

async function saveProject() {
  cancelAutoSavePending();
  if (seriesPersistTimer) {
    clearTimeout(seriesPersistTimer);
    seriesPersistTimer = null;
  }
  await persistActiveSeries();
  await persistProjectToServer({ silent: false });
}

function buildSegmentAdapters() {
  const adapters: { id: string; weight: number }[] = [];
  if (segmentComposeParams.lora) {
    adapters.push({
      id: String(segmentComposeParams.lora),
      weight: Number(segmentComposeParams.lora_scale) || 1.0,
    });
  }
  return adapters;
}

async function runGenerationTask(submitRes: unknown) {
  const tid = taskIdFromSubmitResponse(submitRes);
  if (!tid) throw new Error('missing task id');
  tasksStore.registerPageOwnedStream(tid);
  generating.value = true;
  return new Promise<void>((resolve, reject) => {
    api.gen.streamMediaTask(tid, {
      onDone: async (doneData: unknown) => {
        generating.value = false;
        tasksStore.unregisterPageOwnedStream(tid);
        const data = doneData as { status: string };
        if (data.status === 'completed') {
          const updated = (await api.gen.getMediaTask(tid)) as {
            result?: { metadata?: { shots?: LongVideoProjectState['shots']; primary_asset_id?: string } };
            metadata?: { long_video?: { shots?: LongVideoProjectState['shots'] } };
          };
          const shotsMeta =
            updated?.result?.metadata?.shots ?? updated?.metadata?.long_video?.shots;
          const primaryId = updated?.result?.metadata?.primary_asset_id;
          if (longVideoProject.project.value) {
            const patch: Partial<LongVideoProjectState> = {};
            if (Array.isArray(shotsMeta)) patch.shots = shotsMeta;
            if (primaryId) patch.final_asset_id = primaryId;
            longVideoProject.patchProject(patch);
          }
          toast.success($tt('studio.genComplete'));
          resolve();
        } else {
          toast.error($tt('studio.genFailed', { msg: '' }));
          reject(new Error('generation failed'));
        }
      },
      onError: () => {
        generating.value = false;
        tasksStore.unregisterPageOwnedStream(tid);
        reject(new Error('connection lost'));
      },
    });
  });
}

async function mergeLongVideo() {
  const lv =
    project.value ||
    defaultLongVideoProject({
      keyframe_model: defaultKeyframeModel.value,
      segment_video_model: defaultSegmentModel.value,
      target_duration_sec: 60,
      title: projectTitle.value,
    });

  if (!lv.shots.length) {
    toast.warning($tt('video.longVideoNeedKeyframes'));
    return;
  }
  if (toastProductionBlockers()) return;

  generating.value = true;
  try {
    const missing = lv.shots.filter((s) => !String(s.segment_asset_id || '').trim());
    if (missing.length) {
      throw new Error($tt('video.longVideoNeedSegments') || 'Every shot needs a segment video before assemble');
    }
    const p = segmentComposeParams;
    const res = await api.longVideo.assembleSegments({
      segment_asset_ids: lv.shots.map((s) => String(s.segment_asset_id)),
      overlap_frames: effectiveOverlapFrames.value,
      fps: p.fps || scalarDefault(lv.segment_video_model, 'fps', 16),
      title: projectTitle.value.trim(),
    });
    const assetId = String(res.asset_id || '');
    if (!assetId) throw new Error('assemble returned no asset_id');
    longVideoProject.patchProject({ final_asset_id: assetId });
    toast.success($tt('video.longVideoAssembleDone') || 'Assembled final video');
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  } finally {
    generating.value = false;
  }
}

async function bootstrapEpisodeFromRoute() {
  const projectId = String(route.params.projectId || '');
  if (!projectId) {
    await router.replace({ name: 'long_video_projects' });
    return;
  }
  try {
    await loadSavedProject(projectId, { silent: true });
  } catch {
    toast.warning($tt('video.longVideoProjectLoadFailed'));
    await router.replace({ name: 'long_video_projects' });
  }
}

onMounted(async () => {
  await loadRegistry();
  await bootstrapEpisodeFromRoute();
  syncOutputSizeForSegmentModel(segmentModelId.value);
  await loadGallery(true);
});

watch(
  () => route.params.projectId,
  (pid, prev) => {
    if (pid && pid !== prev) void bootstrapEpisodeFromRoute();
  },
);

watch(segmentModelId, (modelId) => {
  syncOutputSizeForSegmentModel(modelId);
});

</script>
