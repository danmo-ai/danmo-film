<template>
  <div class="copilot-page long-video-page long-video-page--episode lv-series-detail">
    <div class="copilot-page__main">
    <p v-if="loading" class="lv-series-detail__status">{{ $tt('common.loading') }}</p>
    <template v-else-if="series">
      <div class="lv-editor-workspace-shell copilot-page__workspace-card">
        <div class="lv-editor-workspace">
          <div class="lv-editor-tabs-wrapper">
            <div class="lv-editor-tabs-shell lv-editor-tabs-shell--rail">
              <div class="lv-editor-tabs lv-editor-tabs--rail" role="tablist">
                <button
                  v-for="tab in workspaceTabs"
                  :key="tab.id"
                  type="button"
                  role="tab"
                  class="lv-editor-tabs__tab"
                  :class="{ 'lv-editor-tabs__tab--active': workspaceTab === tab.id }"
                  :aria-selected="workspaceTab === tab.id"
                  @click="workspaceTab = tab.id"
                >
                  <span class="lv-editor-tabs__text">
                    <span class="lv-editor-tabs__label">{{ tab.label }}</span>
                  </span>
                </button>
              </div>
            </div>
          </div>

          <div class="lv-editor-pane-host">
      <div v-show="workspaceTab === 'settings'" class="lv-editor-pane lv-editor-pane--settings">
        <div class="lv-ep-sheet">
        <LongVideoSettingsPanel
          mode="project"
          :title="seriesTitle"
          :keyframe-model="seriesState.keyframe_model ?? defaultKeyframeModel"
          :segment-model="seriesState.segment_video_model ?? defaultSegmentModel"
          :output-size="seriesState.output_size || '1280x704'"
          :output-size-options="outputSizeOptions"
          :keyframe-model-options="keyframeModelOptions"
          :segment-model-options="segmentModelOptions"
          :model-label="modelLabel"
          :overlap-frames="seriesState.overlap_frames ?? 4"
          :target-duration-sec="seriesState.target_duration_sec ?? 60"
          :chain-mode="seriesState.chain_mode ?? 'keyframe_only'"
          :style-anchor="seriesState.style_anchor ?? ''"
          :project-id="seriesId"
          @update:title="seriesTitle = $event"
          @update:keyframe-model="patchSeriesField('keyframe_model', $event)"
          @update:segment-model="patchSeriesField('segment_video_model', $event)"
          @update:output-size="patchSeriesField('output_size', $event)"
          @update:overlap-frames="patchSeriesField('overlap_frames', Number($event))"
          @update:target-duration-sec="patchSeriesField('target_duration_sec', Number($event) || 60)"
          @update:chain-mode="patchSeriesField('chain_mode', $event)"
          @update:style-anchor="patchSeriesField('style_anchor', $event)"
        />
        </div>
        <LongVideoStepSaveBar
          :save-label="$tt('video.longVideoSeriesSaveSettings')"
          :hint="$tt('video.longVideoSeriesSaveSettingsHint')"
          status=""
          status-label=""
          :loading="saving"
          @save="saveSeries"
        />
      </div>

      <div v-show="workspaceTab === 'cast'" class="lv-editor-pane lv-editor-pane--cast">
        <div class="lv-series-detail__actions">
          <DqButton size="sm" @click="openLibraryImport('character')">
            {{ $tt('video.longVideoSeriesImportFromLibrary') }}
          </DqButton>
        </div>
        <LongVideoCastPanel
          :characters="seriesState.characters ?? []"
          :project-id="seriesId"
          @update:characters="onSeriesCharacters"
        />
        <div v-if="seriesState.characters?.length" class="lv-series-workspace__library-list">
          <div
            v-for="ch in seriesState.characters"
            :key="ch.id"
            class="lv-series-workspace__library-row"
          >
            <span>{{ ch.name }}</span>
            <span v-if="ch.library_version" class="lv-asset-scope-bar__badge">
              {{ $tt('video.longVideoLibraryVersionBadge', { v: ch.library_version }) }}
            </span>
            <DqButton
              v-if="!ch.library_entity_id"
              size="sm"
              type="text"
              @click="linkCharacterLibrary(ch.id)"
            >
              {{ $tt('video.longVideoSeriesLinkLibrary') }}
            </DqButton>
            <DqButton
              v-else
              size="sm"
              type="text"
              @click="openVersionHistory('character', ch.library_entity_id!, ch.name)"
            >
              {{ $tt('video.longVideoSeriesOpenVersions') }}
            </DqButton>
            <DqButton size="sm" type="text" @click="publishCharacter(ch.id)">
              {{ $tt('video.longVideoPublishLibraryVersion') }}
            </DqButton>
          </div>
        </div>
      </div>

      <div v-show="workspaceTab === 'scenes'" class="lv-editor-pane lv-editor-pane--scenes">
        <div class="lv-series-detail__actions">
          <DqButton size="sm" @click="openLibraryImport('scene')">
            {{ $tt('video.longVideoSeriesImportFromLibrary') }}
          </DqButton>
        </div>
        <LongVideoScenePanel
          :scenes="seriesState.scenes ?? []"
          :project-id="seriesId"
          @update:scenes="onSeriesScenes"
        />
        <div v-if="seriesState.scenes?.length" class="lv-series-workspace__library-list">
          <div
            v-for="sc in seriesState.scenes"
            :key="sc.id"
            class="lv-series-workspace__library-row"
          >
            <span>
              {{ sc.name }}
              <template v-if="sc.time_of_day || sc.weather">
                · {{ sc.time_of_day || '—' }} / {{ sc.weather || '—' }}
              </template>
            </span>
            <span v-if="sc.library_version" class="lv-asset-scope-bar__badge">
              {{ $tt('video.longVideoLibraryVersionBadge', { v: sc.library_version }) }}
            </span>
            <DqButton
              v-if="!sc.library_entity_id"
              size="sm"
              type="text"
              @click="linkSceneLibrary(sc.id)"
            >
              {{ $tt('video.longVideoSeriesLinkLibrary') }}
            </DqButton>
            <DqButton
              v-else
              size="sm"
              type="text"
              @click="openVersionHistory('scene', sc.library_entity_id!, sc.name)"
            >
              {{ $tt('video.longVideoSeriesOpenVersions') }}
            </DqButton>
            <DqButton size="sm" type="text" @click="publishScene(sc.id)">
              {{ $tt('video.longVideoPublishLibraryVersion') }}
            </DqButton>
          </div>
        </div>
      </div>

      <div v-show="workspaceTab === 'episodes'" class="lv-editor-pane lv-series-pane">
        <div class="lv-series-detail__actions">
          <DqButton type="primary" @click="onNewEpisode">
            + {{ $tt('video.longVideoNewEpisode') }}
          </DqButton>
          <DqButton :disabled="!episodes.length" @click="onNewEpisodeFromPrevious">
            {{ $tt('video.longVideoSeriesNewEpisodeFromPrev') }}
          </DqButton>
          <DqButton :disabled="!firstEpisodeId" @click="openEpisode(firstEpisodeId!)">
            {{ $tt('video.longVideoOpenLatestEpisode') }}
          </DqButton>
        </div>
        <p v-if="!episodes.length" class="lv-series-detail__hint">
          {{ $tt('video.longVideoEpisodesEmpty') }}
        </p>
        <ul v-else class="lv-series-detail__ep-list" role="list">
          <li v-for="ep in episodes" :key="ep.episode_project_id" class="lv-series-detail__ep">
            <button
              type="button"
              class="lv-series-detail__ep-open"
              @click="openEpisode(ep.episode_project_id)"
            >
              <span class="lv-series-detail__ep-title">
                {{ $tt('video.longVideoEpisodeLabel', { n: ep.episode_number, title: ep.title }) }}
              </span>
              <span class="lv-series-detail__ep-meta">
                {{ $tt('video.longVideoProjectListMeta', { shots: ep.shot_count }) }}
                · {{ formatUpdated(ep.updated_at) }}
              </span>
            </button>
            <button
              type="button"
              class="lv-series-detail__ep-delete"
              @click="onDeleteEpisode(ep.episode_project_id)"
            >
              {{ $tt('common.delete') }}
            </button>
          </li>
        </ul>
      </div>

      <div v-show="workspaceTab === 'updates'" class="lv-editor-pane lv-series-pane">
        <p v-if="!libraryUpdates.length" class="lv-series-detail__hint">
          {{ $tt('video.longVideoSeriesUpdatesEmpty') }}
        </p>
        <ul v-else class="lv-series-detail__ep-list" role="list">
          <li v-for="row in libraryUpdates" :key="`${row.kind}-${row.rosterId}`" class="lv-series-detail__ep">
            <div class="lv-series-detail__ep-open">
              <span class="lv-series-detail__ep-title">
                {{ $tt('video.longVideoSeriesUpdateRow', {
                  name: row.name,
                  locked: row.lockedVersion,
                  latest: row.latestVersion,
                }) }}
              </span>
            </div>
            <DqButton size="sm" type="text" @click="dismissLibraryUpdate(row)">
              {{ $tt('video.longVideoSeriesKeepLockedVersion') }}
            </DqButton>
            <DqButton size="sm" type="primary" @click="upgradeLibraryUpdate(row)">
              {{ $tt('video.longVideoSeriesUpgradeLockedVersion') }}
            </DqButton>
          </li>
        </ul>
      </div>
          </div>
        </div>
      </div>
    </template>
    <p v-else class="lv-series-detail__status">{{ $tt('video.longVideoSeriesLoadFailed') }}</p>
    </div>

    <DqDialog v-model:open="libraryPickerOpen" :title="$tt('video.longVideoSeriesImportFromLibrary')" width="480px">
      <ul v-if="libraryPickerItems.length" class="lv-series-detail__ep-list" role="list">
        <li v-for="item in libraryPickerItems" :key="item.entity_id" class="lv-series-detail__ep">
          <button type="button" class="lv-series-detail__ep-open" @click="importLibraryRow(item)">
            <span class="lv-series-detail__ep-title">
              {{ String(item.definition?.name ?? item.entity_id) }}
            </span>
            <span class="lv-series-detail__ep-meta">v{{ item.version }}</span>
          </button>
        </li>
      </ul>
      <p v-else class="lv-series-detail__hint">{{ $tt('common.loading') }}</p>
    </DqDialog>

    <DqDialog v-model:open="versionDialogOpen" :title="versionDialogTitle" width="480px">
      <ul v-if="versionRows.length" class="lv-series-detail__ep-list" role="list">
        <li v-for="v in versionRows" :key="v.version" class="lv-series-detail__ep">
          <span class="lv-series-detail__ep-meta">v{{ v.version }} · {{ formatUpdated(v.created_at) }}</span>
        </li>
      </ul>
    </DqDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import { toast, confirm } from '@/utils/feedback';
import { api, isRevisionConflictError } from '@/utils/api';
import { $tt } from '@/utils/i18n';
import type { LibraryEntityVersion, LongVideoCharacter, LongVideoScene, LongVideoSeriesDetail, LongVideoSeriesEpisodeSummary, LongVideoSeriesState } from '@/types';
import { defaultSeriesState, newStableId } from '@/utils/longVideoSeries';
import {
  characterFromLibraryDefinition,
  listLibraryUpdates,
  loadSeriesWithLibraryMigration,
  publishCharacterToLibrary,
  publishSceneToLibrary,
  sceneFromLibraryDefinition,
  upgradeSeriesEntityLibraryVersion,
  type LibraryUpdateRow,
} from '@/utils/longVideoLibrary';
import LongVideoCastPanel from '@/components/long-video/LongVideoCastPanel.vue';
import LongVideoScenePanel from '@/components/long-video/LongVideoScenePanel.vue';
import LongVideoSettingsPanel from '@/components/long-video/LongVideoSettingsPanel.vue';
import LongVideoStepSaveBar from '@/components/long-video/LongVideoStepSaveBar.vue';
import { useLongVideoRegistry } from '@/composables/useLongVideoRegistry';
import { resolveScriptText } from '@/utils/longVideoProject';

type WorkspaceTab = 'settings' | 'cast' | 'scenes' | 'episodes' | 'updates';

const route = useRoute();
const router = useRouter();
const { locale } = useI18n();
const refreshNav = inject<() => void>('lvRefreshWorkspaceNav', () => undefined);
const {
  profile,
  modelLabel,
  resolutionOptionsForModel,
  loadRegistry,
} = useLongVideoRegistry();

const loading = ref(true);
const saving = ref(false);
const series = ref<LongVideoSeriesDetail | null>(null);
const episodes = ref<LongVideoSeriesEpisodeSummary[]>([]);
const workspaceTab = ref<WorkspaceTab>('settings');
const seriesTitle = ref('');
const libraryUpdates = ref<LibraryUpdateRow[]>([]);
const libraryPickerOpen = ref(false);
const libraryPickerKind = ref<'character' | 'scene'>('character');
const libraryPickerItems = ref<LibraryEntityVersion[]>([]);
const versionDialogOpen = ref(false);
const versionDialogTitle = ref('');
const versionRows = ref<LibraryEntityVersion[]>([]);

const seriesId = computed(() => String(route.params.seriesId || ''));
const seriesState = computed((): LongVideoSeriesState =>
  series.value?.state ?? defaultSeriesState(),
);
const firstEpisodeId = computed(() => episodes.value[0]?.episode_project_id);
const defaultKeyframeModel = computed(() => profile.value?.keyframe_models[0] ?? 'z-image-turbo');
const defaultSegmentModel = computed(() => profile.value?.segment_models[0] ?? 'wan-2.2-i2v-14b');
const keyframeModelOptions = computed(() => profile.value?.keyframe_models ?? []);
const segmentModelOptions = computed(() => profile.value?.segment_models ?? []);
const outputSizeOptions = computed(() =>
  resolutionOptionsForModel(seriesState.value.segment_video_model ?? defaultSegmentModel.value),
);

const workspaceTabs = computed(() => [
  { id: 'settings' as const, label: $tt('video.longVideoSeriesTabSettings') },
  { id: 'cast' as const, label: $tt('video.longVideoSeriesTabCast') },
  { id: 'scenes' as const, label: $tt('video.longVideoSeriesTabScenes') },
  { id: 'episodes' as const, label: $tt('video.longVideoSeriesTabEpisodes') },
  { id: 'updates' as const, label: $tt('video.longVideoSeriesTabUpdates') },
]);

function formatUpdated(iso: string): string {
  if (!iso) return '';
  try {
    return new Date(iso).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return iso;
  }
}

function patchSeriesField<K extends keyof LongVideoSeriesState>(key: K, value: LongVideoSeriesState[K]) {
  if (!series.value) return;
  series.value = {
    ...series.value,
    state: { ...series.value.state, [key]: value },
  };
}

function onSeriesCharacters(characters: LongVideoCharacter[]) {
  patchSeriesField('characters', characters);
}

function onSeriesScenes(scenes: LongVideoScene[]) {
  patchSeriesField('scenes', scenes);
}

function openEpisode(projectId: string) {
  void router.push({ name: 'long_video_create', params: { projectId } });
}

async function refreshLibraryUpdates() {
  if (!series.value) {
    libraryUpdates.value = [];
    return;
  }
  try {
    const [characters, scenes] = await Promise.all([
      api.library.listCharacters(),
      api.library.listScenes(),
    ]);
    libraryUpdates.value = listLibraryUpdates(series.value.state, { characters, scenes });
  } catch {
    libraryUpdates.value = [];
  }
}

async function load() {
  const id = seriesId.value;
  if (!id) return;
  loading.value = true;
  try {
    const detail = await loadSeriesWithLibraryMigration(
      (sid) => api.longVideo.getSeries(sid),
      (sid, body) => api.longVideo.updateSeries(sid, body),
      id,
      api.library,
    );
    series.value = { ...detail, state: defaultSeriesState(detail.state ?? {}) };
    seriesTitle.value = detail.title;
    episodes.value = await api.longVideo.listSeriesEpisodes(id);
    await refreshLibraryUpdates();
  } catch (e: unknown) {
    series.value = null;
    episodes.value = [];
    if (isRevisionConflictError(e)) {
      toast.error($tt('video.longVideoRevisionConflict'));
    } else {
      toast.warning($tt('video.longVideoSeriesLoadFailed'));
    }
  } finally {
    loading.value = false;
  }
}

async function saveSeries() {
  const s = series.value;
  if (!s) return;
  saving.value = true;
  try {
    const saved = await api.longVideo.updateSeries(s.id, {
      title: seriesTitle.value.trim() || s.title,
      state: s.state,
    });
    series.value = { ...saved, state: defaultSeriesState(saved.state ?? {}) };
    seriesTitle.value = saved.title;
    toast.success($tt('video.longVideoSeriesSaveSettingsOk'));
    refreshNav();
    await refreshLibraryUpdates();
  } catch (e: unknown) {
    if (isRevisionConflictError(e)) {
      toast.error($tt('video.longVideoRevisionConflict'));
      await load();
    } else {
      const msg = e instanceof Error ? e.message : String(e);
      toast.error($tt('studio.error', { msg }));
    }
  } finally {
    saving.value = false;
  }
}

async function onNewEpisode() {
  const id = seriesId.value;
  if (!id) return;
  try {
    const ep = await api.longVideo.createSeriesEpisode(id);
    toast.success($tt('video.longVideoNewEpisodeDone', { n: ep.episode.episode_number }));
    refreshNav();
    await router.push({
      name: 'long_video_create',
      params: { projectId: ep.project.id },
    });
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  }
}

async function onNewEpisodeFromPrevious() {
  const id = seriesId.value;
  const prev = episodes.value[0];
  if (!id || !prev) return;
  try {
    const prevProject = await api.longVideo.getProject(prev.episode_project_id);
    const st = prevProject.state ?? {};
    const ep = await api.longVideo.createSeriesEpisode(id, {
      state: {
        script_text: resolveScriptText(st),
        chapter_title: st.chapter_title,
        character_variants: {},
        scene_variants: {},
      },
    });
    toast.success($tt('video.longVideoNewEpisodeDone', { n: ep.episode.episode_number }));
    refreshNav();
    await load();
    await router.push({ name: 'long_video_create', params: { projectId: ep.project.id } });
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  }
}

async function onDeleteEpisode(projectId: string) {
  try {
    await confirm(
      $tt('video.longVideoProjectDeleteConfirm'),
      $tt('common.delete'),
      { type: 'warning' },
    );
  } catch {
    return;
  }
  try {
    await api.longVideo.deleteProject(projectId);
    toast.success($tt('video.longVideoProjectDeleted'));
    refreshNav();
    await load();
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  }
}

async function openLibraryImport(kind: 'character' | 'scene') {
  libraryPickerKind.value = kind;
  libraryPickerOpen.value = true;
  libraryPickerItems.value = [];
  try {
    libraryPickerItems.value =
      kind === 'character' ? await api.library.listCharacters() : await api.library.listScenes();
  } catch {
    libraryPickerItems.value = [];
  }
}

function importLibraryRow(row: LibraryEntityVersion) {
  if (!series.value) return;
  if (libraryPickerKind.value === 'character') {
    const ch = characterFromLibraryDefinition(
      row.definition,
      row.entity_id,
      row.version,
      newStableId('lc_'),
    );
    onSeriesCharacters([...(seriesState.value.characters ?? []), ch]);
  } else {
    const sc = sceneFromLibraryDefinition(
      row.definition,
      row.entity_id,
      row.version,
      newStableId('ls_'),
    );
    onSeriesScenes([...(seriesState.value.scenes ?? []), sc]);
  }
  libraryPickerOpen.value = false;
  void saveSeries();
}

async function linkCharacterLibrary(rosterId: string) {
  const ch = seriesState.value.characters?.find((c) => c.id === rosterId);
  if (!ch || !series.value) return;
  try {
    const row = await publishCharacterToLibrary(ch, api.library);
    onSeriesCharacters(
      (seriesState.value.characters ?? []).map((c) => (c.id === rosterId ? row : c)),
    );
    await saveSeries();
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  }
}

async function linkSceneLibrary(rosterId: string) {
  const sc = seriesState.value.scenes?.find((s) => s.id === rosterId);
  if (!sc || !series.value) return;
  try {
    const row = await publishSceneToLibrary(sc, api.library);
    onSeriesScenes((seriesState.value.scenes ?? []).map((s) => (s.id === rosterId ? row : s)));
    await saveSeries();
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  }
}

async function publishCharacter(rosterId: string) {
  await linkCharacterLibrary(rosterId);
}

async function publishScene(rosterId: string) {
  await linkSceneLibrary(rosterId);
}

async function openVersionHistory(kind: 'character' | 'scene', entityId: string, name: string) {
  versionDialogTitle.value = name;
  versionDialogOpen.value = true;
  versionRows.value = [];
  try {
    versionRows.value =
      kind === 'character'
        ? await api.library.listCharacterVersions(entityId)
        : await api.library.listSceneVersions(entityId);
  } catch {
    versionRows.value = [];
  }
}

function dismissLibraryUpdate(row: LibraryUpdateRow) {
  libraryUpdates.value = libraryUpdates.value.filter(
    (u) => !(u.kind === row.kind && u.rosterId === row.rosterId),
  );
}

async function upgradeLibraryUpdate(row: LibraryUpdateRow) {
  if (!series.value) return;
  try {
    const next = await upgradeSeriesEntityLibraryVersion(
      series.value.state,
      api.library,
      row,
      row.latestVersion,
    );
    series.value = { ...series.value, state: next };
    await saveSeries();
    dismissLibraryUpdate(row);
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  }
}

watch(seriesId, () => {
  void load();
});

onMounted(async () => {
  await loadRegistry();
  await load();
});
</script>

<style scoped>
.lv-series-detail {
  height: 100%;
  min-height: 0;
  overflow: hidden;
  padding: 0;
  box-sizing: border-box;
}

.lv-series-pane,
.lv-series-detail .lv-editor-pane--cast,
.lv-series-detail .lv-editor-pane--scenes {
  overflow: auto;
  padding: 16px 28px 24px;
  align-items: flex-start;
}

.lv-series-detail__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 14px;
}

.lv-series-detail__status {
  color: var(--dq-label-tertiary);
}

.lv-series-detail__hint {
  margin: 0 0 12px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--dq-label-tertiary);
}

.lv-series-detail__field-grid {
  display: grid;
  gap: 12px;
  margin: 16px 0;
  max-width: 640px;
}

.lv-series-detail__field span {
  display: block;
  font-size: 11px;
  color: var(--dq-label-tertiary);
  margin-bottom: 4px;
}

.lv-series-detail__ep-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 640px;
}

.lv-series-detail__ep {
  display: flex;
  align-items: stretch;
  gap: 8px;
}

.lv-series-detail__ep-open {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  padding: 14px 16px;
  border-radius: 12px;
  border: 0.5px solid var(--dq-border-subtle);
  background: color-mix(in srgb, var(--dq-surface-elevated) 70%, transparent);
  color: inherit;
  text-align: left;
  cursor: pointer;
  font: inherit;
}

.lv-series-detail__ep-open:hover {
  border-color: color-mix(in srgb, var(--dq-accent) 40%, transparent);
  background: color-mix(in srgb, var(--dq-accent) 8%, var(--dq-surface-elevated));
}

.lv-series-detail__ep-title {
  font-size: 14px;
  font-weight: 650;
  color: var(--dq-label-primary);
}

.lv-series-detail__ep-meta {
  font-size: 12px;
  color: var(--dq-label-tertiary);
}

.lv-series-detail__ep-delete {
  border: none;
  background: transparent;
  color: var(--dq-label-tertiary);
  font-size: 12px;
  cursor: pointer;
  padding: 0 4px;
}

.lv-series-detail__ep-delete:hover {
  color: var(--dq-danger);
}
</style>
