<template>
  <aside class="lv-ws-nav">
    <div class="lv-ws-nav__head">
      <h2 class="lv-ws-nav__title">{{ $tt('video.longVideoLibraryTitle') }}</h2>
      <DqButton type="primary" size="sm" block @click="onNewProject">
        + {{ $tt('video.longVideoNewSeries') }}
      </DqButton>
    </div>

    <p v-if="loading" class="lv-ws-nav__hint">{{ $tt('common.loading') }}</p>
    <p v-else-if="!projects.length" class="lv-ws-nav__hint">
      {{ $tt('video.longVideoLibraryEmptyBody') }}
    </p>

    <ul v-else class="lv-ws-nav__tree" role="tree">
      <li
        v-for="p in projects"
        :key="p.id"
        class="lv-ws-nav__project"
        :class="{ 'is-current': p.id === activeSeriesId && !activeEpisodeId }"
        role="treeitem"
        :aria-expanded="expandedIds.has(p.id)"
      >
        <div class="lv-ws-nav__project-row">
          <button
            type="button"
            class="lv-ws-nav__twist"
            :aria-label="expandedIds.has(p.id) ? 'collapse' : 'expand'"
            @click="toggleExpand(p.id)"
          >
            {{ expandedIds.has(p.id) ? '▾' : '▸' }}
          </button>
          <button type="button" class="lv-ws-nav__project-btn" @click="openProject(p.id)">
            <span class="lv-ws-nav__name">{{ p.title }}</span>
            <span class="lv-ws-nav__meta">
              {{ $tt('video.longVideoSeriesEpisodeCount', { n: p.episode_count }) }}
            </span>
          </button>
          <button
            type="button"
            class="lv-ws-nav__icon-btn"
            :title="$tt('common.delete')"
            @click.stop="onDeleteProject(p.id)"
          >
            ×
          </button>
        </div>

        <ul v-if="expandedIds.has(p.id)" class="lv-ws-nav__eps" role="group">
          <li v-if="episodesLoading[p.id]" class="lv-ws-nav__ep-hint">
            {{ $tt('common.loading') }}
          </li>
          <template v-else>
            <li
              v-for="ep in episodesByProject[p.id] || []"
              :key="ep.episode_project_id"
              class="lv-ws-nav__ep"
              :class="{ 'is-active': ep.episode_project_id === activeEpisodeId }"
            >
              <button
                type="button"
                class="lv-ws-nav__ep-btn"
                @click="openEpisode(ep.episode_project_id)"
              >
                {{ formatEpisodeNavLabel(ep.episode_number, ep.title) }}
              </button>
            </li>
            <li class="lv-ws-nav__ep lv-ws-nav__ep--action">
              <button type="button" class="lv-ws-nav__ep-new" @click="onNewEpisode(p.id)">
                + {{ $tt('video.longVideoNewEpisode') }}
              </button>
            </li>
          </template>
        </ul>
      </li>
    </ul>
  </aside>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { toast, confirm } from '@/utils/feedback';
import { api } from '@/utils/api';
import { $tt } from '@/utils/i18n';
import type { LongVideoSeriesEpisodeSummary, LongVideoSeriesSummary } from '@/types';
import { defaultSeriesState } from '@/utils/longVideoSeries';
import {
  defaultLongVideoProject,
  projectStateForServer,
} from '@/utils/longVideoProject';
import { useLongVideoRegistry } from '@/composables/useLongVideoRegistry';

const props = defineProps<{
  refreshToken?: number;
}>();

const emit = defineEmits<{
  (e: 'changed'): void;
}>();

const route = useRoute();
const router = useRouter();
const { profile, loadRegistry } = useLongVideoRegistry();

const loading = ref(true);
const projects = ref<LongVideoSeriesSummary[]>([]);
const expandedIds = ref<Set<string>>(new Set());
const episodesByProject = reactive<Record<string, LongVideoSeriesEpisodeSummary[]>>({});
const episodesLoading = reactive<Record<string, boolean>>({});

const defaultKeyframeModel = computed(
  () => profile.value?.keyframe_models[0] ?? 'z-image-turbo',
);
const defaultSegmentModel = computed(
  () => profile.value?.segment_models[0] ?? 'wan-2.2-i2v-14b',
);

const activeSeriesId = computed(() => {
  if (route.name === 'long_video_series') return String(route.params.seriesId || '');
  // Episode route: resolve from loaded episode lists when possible
  const epId = String(route.params.projectId || '');
  if (!epId) return '';
  for (const [sid, eps] of Object.entries(episodesByProject)) {
    if (eps.some((e) => e.episode_project_id === epId)) return sid;
  }
  return String(route.query.seriesId || '');
});

const activeEpisodeId = computed(() =>
  route.name === 'long_video_create' ? String(route.params.projectId || '') : '',
);

/** Nav formats "第 N 集 · {title}"; avoid "第 1 集 · 第 1 集" when title is the default. */
function formatEpisodeNavLabel(n: number, title: string): string {
  let cleaned = (title || '').trim();
  const zhPref = new RegExp(`^第\\s*${n}\\s*集(?:\\s*[·•.\\-—–]\\s*|\\s+)`, 'u');
  const enPref = new RegExp(`^Episode\\s*${n}(?:\\s*[·•.\\-—–]\\s*|\\s+)`, 'i');
  for (let i = 0; i < 3; i += 1) {
    const next = cleaned.replace(zhPref, '').replace(enPref, '').trim();
    if (next === cleaned) break;
    cleaned = next;
  }
  const isDefaultOnly =
    !cleaned ||
    new RegExp(`^第\\s*${n}\\s*集$`, 'u').test(cleaned) ||
    new RegExp(`^Episode\\s*${n}$`, 'i').test(cleaned);
  if (isDefaultOnly) {
    return $tt('video.longVideoNewEpisodeDefaultTitle', { n });
  }
  return $tt('video.longVideoEpisodeLabel', { n, title: cleaned });
}

async function refreshProjects() {
  loading.value = true;
  try {
    projects.value = await api.longVideo.listSeries(100);
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
    projects.value = [];
  } finally {
    loading.value = false;
  }
}

async function loadEpisodes(seriesId: string) {
  episodesLoading[seriesId] = true;
  try {
    episodesByProject[seriesId] = await api.longVideo.listSeriesEpisodes(seriesId);
  } catch {
    episodesByProject[seriesId] = [];
  } finally {
    episodesLoading[seriesId] = false;
  }
}

function toggleExpand(seriesId: string) {
  const next = new Set(expandedIds.value);
  if (next.has(seriesId)) next.delete(seriesId);
  else {
    next.add(seriesId);
    void loadEpisodes(seriesId);
  }
  expandedIds.value = next;
}

function ensureExpanded(seriesId: string) {
  if (!seriesId) return;
  if (!expandedIds.value.has(seriesId)) {
    const next = new Set(expandedIds.value);
    next.add(seriesId);
    expandedIds.value = next;
  }
  void loadEpisodes(seriesId);
}

function openProject(seriesId: string) {
  ensureExpanded(seriesId);
  void router.push({ name: 'long_video_series', params: { seriesId } });
}

function openEpisode(projectId: string) {
  void router.push({ name: 'long_video_create', params: { projectId } });
}

async function onNewProject() {
  const titleRaw = window.prompt($tt('video.longVideoNewSeriesTitlePrompt'), '');
  if (titleRaw === null) return;
  const title = titleRaw.trim() || $tt('video.longVideoNewSeries');
  try {
    const created = await api.longVideo.createSeries({
      title,
      state: defaultSeriesState({
        keyframe_model: defaultKeyframeModel.value,
        segment_video_model: defaultSegmentModel.value,
      }),
    });
    // Store a plain episode title — nav already formats with longVideoEpisodeLabel.
    const ep1Title = $tt('video.longVideoNewEpisodeDefaultTitle', { n: 1 });
    await api.longVideo.createSeriesEpisode(created.id, {
      title: ep1Title,
      episode_number: 1,
      state: projectStateForServer(
        defaultLongVideoProject({
          keyframe_model: defaultKeyframeModel.value,
          segment_video_model: defaultSegmentModel.value,
          series_id: created.id,
          episode_number: 1,
          title: ep1Title,
        }),
      ),
    });
    toast.success($tt('video.longVideoNewSeriesDone'));
    await refreshProjects();
    ensureExpanded(created.id);
    emit('changed');
    await router.push({ name: 'long_video_series', params: { seriesId: created.id } });
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  }
}

async function onNewEpisode(seriesId: string) {
  try {
    const ep = await api.longVideo.createSeriesEpisode(seriesId);
    toast.success($tt('video.longVideoNewEpisodeDone', { n: ep.episode.episode_number }));
    await loadEpisodes(seriesId);
    emit('changed');
    await router.push({
      name: 'long_video_create',
      params: { projectId: ep.project.id },
    });
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  }
}

async function onDeleteProject(seriesId: string) {
  try {
    await confirm(
      $tt('video.longVideoSeriesDeleteConfirm'),
      $tt('common.delete'),
      { type: 'warning' },
    );
  } catch {
    return;
  }
  try {
    await api.longVideo.deleteSeries(seriesId);
    toast.success($tt('video.longVideoSeriesDeleted'));
    await refreshProjects();
    emit('changed');
    if (activeSeriesId.value === seriesId) {
      await router.push({ name: 'long_video_projects' });
    }
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    toast.error($tt('studio.error', { msg }));
  }
}

async function refreshAll() {
  await refreshProjects();
  for (const sid of expandedIds.value) {
    await loadEpisodes(sid);
  }
}

watch(
  () => props.refreshToken,
  () => {
    void refreshAll();
  },
);

watch(
  () => [route.name, route.params.seriesId, route.params.projectId] as const,
  async () => {
    const sid = String(route.params.seriesId || '');
    if (sid) ensureExpanded(sid);
    if (route.name === 'long_video_create') {
      const epId = String(route.params.projectId || '');
      for (const p of projects.value) {
        if (!episodesByProject[p.id]) await loadEpisodes(p.id);
        if (episodesByProject[p.id]?.some((e) => e.episode_project_id === epId)) {
          ensureExpanded(p.id);
          break;
        }
      }
    }
  },
);

onMounted(async () => {
  await loadRegistry();
  await refreshProjects();
  const sid = String(route.params.seriesId || '');
  if (sid) ensureExpanded(sid);
  else if (route.name === 'long_video_create') {
    for (const p of projects.value) {
      await loadEpisodes(p.id);
      const epId = String(route.params.projectId || '');
      if (episodesByProject[p.id]?.some((e) => e.episode_project_id === epId)) {
        ensureExpanded(p.id);
        break;
      }
    }
  }
});

defineExpose({ refreshProjects, refreshAll, loadEpisodes, ensureExpanded });
</script>

<style scoped>
.lv-ws-nav {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  padding: 14px 12px;
  box-sizing: border-box;
  border-right: 0.5px solid var(--dq-border-subtle);
  background: color-mix(in srgb, var(--dq-surface-elevated) 55%, transparent);
}

.lv-ws-nav__head {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 14px;
  flex-shrink: 0;
}

.lv-ws-nav__title {
  margin: 0;
  font-size: 13px;
  font-weight: 650;
  color: var(--dq-label-primary);
}

.lv-ws-nav__hint {
  margin: 8px 4px;
  font-size: 12px;
  line-height: 1.45;
  color: var(--dq-label-tertiary);
}

.lv-ws-nav__tree {
  list-style: none;
  margin: 0;
  padding: 0;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.lv-ws-nav__project-row {
  display: flex;
  align-items: stretch;
  border-radius: 8px;
}

.lv-ws-nav__project-row:hover {
  background: color-mix(in srgb, var(--dq-fill-control) 70%, transparent);
}

.lv-ws-nav__project.is-current > .lv-ws-nav__project-row {
  background: color-mix(in srgb, var(--dq-fill-control) 100%, transparent);
}

.lv-ws-nav__project.is-current > .lv-ws-nav__project-row .lv-ws-nav__name {
  color: var(--dq-label-primary);
}

.lv-ws-nav__twist {
  width: 28px;
  flex-shrink: 0;
  border: none;
  background: transparent;
  color: var(--dq-label-tertiary);
  cursor: pointer;
  font-size: 11px;
}

.lv-ws-nav__project-btn {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  padding: 8px 6px;
  border: none;
  background: transparent;
  cursor: pointer;
  text-align: left;
  color: inherit;
  font: inherit;
}

.lv-ws-nav__project-btn:hover {
  background: transparent;
}

.lv-ws-nav__name {
  font-size: 13px;
  font-weight: 600;
  color: var(--dq-label-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

.lv-ws-nav__meta {
  font-size: 11px;
  color: var(--dq-label-tertiary);
}

.lv-ws-nav__icon-btn {
  width: 28px;
  border: none;
  background: transparent;
  color: var(--dq-label-tertiary);
  cursor: pointer;
  font-size: 16px;
  line-height: 1;
}

.lv-ws-nav__icon-btn:hover {
  color: var(--dq-danger);
}

.lv-ws-nav__eps {
  list-style: none;
  margin: 0 0 6px;
  padding: 2px 0 2px 28px;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.lv-ws-nav__ep-hint {
  font-size: 11px;
  color: var(--dq-label-tertiary);
  padding: 4px 8px;
}

.lv-ws-nav__ep-btn,
.lv-ws-nav__ep-new {
  width: 100%;
  border: none;
  background: transparent;
  text-align: left;
  padding: 7px 10px;
  border-radius: 7px;
  font: inherit;
  font-size: 12px;
  color: var(--dq-label-secondary);
  cursor: pointer;
}

.lv-ws-nav__ep-btn:hover,
.lv-ws-nav__ep-new:hover {
  background: color-mix(in srgb, var(--dq-fill-control) 70%, transparent);
  color: var(--dq-label-primary);
}

.lv-ws-nav__ep.is-active .lv-ws-nav__ep-btn {
  background: color-mix(in srgb, var(--dq-fill-control) 100%, transparent);
  color: var(--dq-label-primary);
  font-weight: 600;
}

.lv-ws-nav__ep-new {
  color: var(--dq-accent);
}
</style>
