<template>
  <section
    class="lv-settings"
    :class="{
      'lv-settings--inline': inline,
      'lv-settings--episode': mode === 'episode' && !inline,
      'lv-settings--project': mode === 'project' && !inline,
    }"
  >
    <!-- Project page: compact inline toolbar (series workspace) -->
    <template v-if="inline">
      <div class="lv-settings-bar" role="group" :aria-label="panelAriaLabel">
        <div class="lv-settings-bar__field">
          <label class="lv-settings-bar__label" :for="ids.title">{{ titleLabel }}</label>
          <DqInput
            :id="ids.title"
            :model-value="title"
            size="sm"
            class="lv-settings-bar__control"
            :placeholder="titlePlaceholder"
            @update:model-value="$emit('update:title', $event)"
          />
        </div>

        <div class="lv-settings-bar__field">
          <label class="lv-settings-bar__label" :for="ids.keyframe">{{ $tt('video.longVideoKeyframeModel') }}</label>
          <DqSelect
            :id="ids.keyframe"
            :model-value="keyframeModel"
            size="sm"
            variant="ghost"
            class="lv-settings-bar__control"
            @update:model-value="$emit('update:keyframeModel', $event)"
          >
            <DqOption v-for="optId in keyframeModelOptions" :key="optId" :label="modelLabel(optId)" :value="optId" />
          </DqSelect>
        </div>

        <div class="lv-settings-bar__field">
          <label class="lv-settings-bar__label" :for="ids.segment">{{ $tt('video.longVideoSegmentModel') }}</label>
          <DqSelect
            :id="ids.segment"
            :model-value="segmentModel"
            size="sm"
            variant="ghost"
            class="lv-settings-bar__control"
            @update:model-value="$emit('update:segmentModel', $event)"
          >
            <DqOption v-for="optId in segmentModelOptions" :key="optId" :label="modelLabel(optId)" :value="optId" />
          </DqSelect>
        </div>

        <div class="lv-settings-bar__field">
          <label class="lv-settings-bar__label" :for="ids.size">{{ $tt('video.longVideoOutputSize') }}</label>
          <DqSelect
            :id="ids.size"
            :model-value="outputSize"
            size="sm"
            variant="ghost"
            class="lv-settings-bar__control"
            @update:model-value="$emit('update:outputSize', $event)"
          >
            <DqOption
              v-for="opt in outputSizeOptions"
              :key="opt.value"
              :label="formatResolutionOptionLabel(opt)"
              :value="opt.value"
            />
          </DqSelect>
        </div>

        <div class="lv-settings-bar__field">
          <label class="lv-settings-bar__label" :for="ids.overlap">{{ $tt('video.longVideoOverlapFrames') }}</label>
          <DqSelect
            :id="ids.overlap"
            :model-value="overlapFrames"
            size="sm"
            variant="ghost"
            class="lv-settings-bar__control"
            @update:model-value="$emit('update:overlapFrames', Number($event))"
          >
            <DqOption
              v-for="n in overlapFrameOptions"
              :key="n"
              :label="overlapFrameLabel(n)"
              :value="n"
            />
          </DqSelect>
        </div>
      </div>
    </template>

    <!-- Episode: flat fields only (workspace card is already the surface) -->
    <template v-else-if="mode === 'episode'">
      <div class="lv-ep-settings">
        <div class="lv-ep-settings__row lv-ep-settings__row--title">
          <div class="lv-ep-settings__field">
            <label class="lv-ep-settings__label" :for="ids.title">{{ titleLabel }}</label>
            <DqInput
              :id="ids.title"
              :model-value="title"
              class="lv-ep-settings__control"
              :placeholder="titlePlaceholder"
              @update:model-value="$emit('update:title', $event)"
            />
          </div>
          <div class="lv-ep-settings__field lv-ep-settings__field--duration">
            <label class="lv-ep-settings__label" :for="ids.duration">{{ $tt('video.longVideoTargetDuration') }}</label>
            <div class="lv-ep-settings__duration">
              <DqInput
                :id="ids.duration"
                type="number"
                class="lv-ep-settings__control"
                :model-value="String(targetDurationSec ?? 60)"
                @update:model-value="$emit('update:targetDurationSec', Number($event) || 60)"
              />
              <span class="lv-ep-settings__unit">{{ $tt('video.longVideoSecondsUnit') }}</span>
            </div>
          </div>
        </div>

        <div class="lv-ep-settings__row lv-ep-settings__row--models">
          <div class="lv-ep-settings__field">
            <label class="lv-ep-settings__label" :for="ids.keyframe">{{ $tt('video.longVideoKeyframeModel') }}</label>
            <DqSelect
              :id="ids.keyframe"
              :model-value="keyframeModel"
              class="lv-ep-settings__control"
              @update:model-value="$emit('update:keyframeModel', $event)"
            >
              <DqOption
                v-for="optId in keyframeSelectOptions"
                :key="optId"
                :label="modelLabel(optId) || optId"
                :value="optId"
              />
            </DqSelect>
          </div>
          <div class="lv-ep-settings__field">
            <label class="lv-ep-settings__label" :for="ids.segment">{{ $tt('video.longVideoSegmentModel') }}</label>
            <DqSelect
              :id="ids.segment"
              :model-value="segmentModel"
              class="lv-ep-settings__control"
              @update:model-value="$emit('update:segmentModel', $event)"
            >
              <DqOption
                v-for="optId in segmentSelectOptions"
                :key="optId"
                :label="modelLabel(optId) || optId"
                :value="optId"
              />
            </DqSelect>
          </div>
        </div>

        <p class="lv-ep-settings__meta">
          <span>
            {{ $tt('video.longVideoOutputSize') }}
            {{ lockedOutputSizeLabel }}
            ·
            {{ $tt('video.longVideoOverlapFrames') }}
            {{ overlapFrameLabel(lockedOverlapFrames ?? 4) }}
          </span>
          <button
            v-if="canOpenProjectSettings"
            type="button"
            class="lv-ep-settings__meta-link"
            @click="$emit('open-project-settings')"
          >
            {{ $tt('video.longVideoOpenProjectSettings') }}
          </button>
        </p>
      </div>
    </template>

    <!-- Project: same row form as the episode sheet -->
    <template v-else>
      <div class="lv-ep-settings">
        <div class="lv-ep-settings__field">
          <label class="lv-ep-settings__label" :for="ids.title">{{ titleLabel }}</label>
          <DqInput
            :id="ids.title"
            :model-value="title"
            class="lv-ep-settings__control"
            :placeholder="titlePlaceholder"
            @update:model-value="$emit('update:title', $event)"
          />
        </div>
        <div class="lv-ep-settings__field">
          <label class="lv-ep-settings__label" :for="ids.keyframe">{{ $tt('video.longVideoKeyframeModel') }}</label>
          <DqSelect
            :id="ids.keyframe"
            :model-value="keyframeModel"
            class="lv-ep-settings__control"
            @update:model-value="$emit('update:keyframeModel', $event)"
          >
            <DqOption v-for="optId in keyframeSelectOptions" :key="optId" :label="modelLabel(optId) || optId" :value="optId" />
          </DqSelect>
        </div>
        <div class="lv-ep-settings__field">
          <label class="lv-ep-settings__label" :for="ids.segment">{{ $tt('video.longVideoSegmentModel') }}</label>
          <DqSelect
            :id="ids.segment"
            :model-value="segmentModel"
            class="lv-ep-settings__control"
            @update:model-value="$emit('update:segmentModel', $event)"
          >
            <DqOption v-for="optId in segmentSelectOptions" :key="optId" :label="modelLabel(optId) || optId" :value="optId" />
          </DqSelect>
        </div>
        <div class="lv-ep-settings__field">
          <label class="lv-ep-settings__label" :for="ids.size">{{ $tt('video.longVideoOutputSize') }}</label>
          <DqSelect
            :id="ids.size"
            :model-value="outputSize"
            class="lv-ep-settings__control"
            @update:model-value="$emit('update:outputSize', $event)"
          >
            <DqOption
              v-for="opt in sizeSelectOptions"
              :key="opt.value"
              :label="formatResolutionOptionLabel(opt)"
              :value="opt.value"
            />
          </DqSelect>
        </div>
        <div class="lv-ep-settings__field">
          <label class="lv-ep-settings__label" :for="ids.overlap">{{ $tt('video.longVideoOverlapFrames') }}</label>
          <DqSelect
            :id="ids.overlap"
            :model-value="overlapFrames"
            class="lv-ep-settings__control"
            @update:model-value="$emit('update:overlapFrames', Number($event))"
          >
            <DqOption v-for="n in overlapFrameOptions" :key="n" :label="overlapFrameLabel(n)" :value="n" />
          </DqSelect>
        </div>
        <div class="lv-ep-settings__field lv-ep-settings__field--top">
          <label class="lv-ep-settings__label" :for="ids.style">{{ $tt('video.longVideoProjectStyleLabel') }}</label>
          <textarea
            :id="ids.style"
            class="dq-input dq-input--textarea lv-ep-settings__control"
            :value="styleAnchor"
            rows="3"
            @input="$emit('update:styleAnchor', ($event.target as HTMLTextAreaElement).value)"
          />
        </div>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { formatResolutionOptionLabel, type ResolutionSizeOption } from '@/utils/registryParamSchema';

const props = withDefaults(
  defineProps<{
    mode?: 'episode' | 'project';
    title: string;
    keyframeModel: string;
    segmentModel: string;
    outputSize: string;
    overlapFrames: number;
    outputSizeOptions: ResolutionSizeOption[];
    keyframeModelOptions: string[];
    segmentModelOptions: string[];
    modelLabel: (id: string) => string;
    inline?: boolean;
    projectId?: string;
    targetDurationSec?: number;
    lockedOutputSize?: string;
    lockedOverlapFrames?: number;
    canOpenProjectSettings?: boolean;
    styleAnchor?: string;
  }>(),
  {
    inline: false,
    mode: 'project',
    targetDurationSec: 60,
    lockedOverlapFrames: 4,
    canOpenProjectSettings: false,
    styleAnchor: '',
  },
);

defineEmits<{
  (e: 'update:title', value: string): void;
  (e: 'update:keyframeModel', value: string): void;
  (e: 'update:segmentModel', value: string): void;
  (e: 'update:outputSize', value: string): void;
  (e: 'update:overlapFrames', value: number): void;
  (e: 'update:targetDurationSec', value: number): void;
  (e: 'update:styleAnchor', value: string): void;
  (e: 'open-project-settings'): void;
}>();

const { t: $tt } = useI18n();

const overlapFrameOptions = [0, 4, 8, 16] as const;

const ids = {
  title: 'lv-settings-title',
  keyframe: 'lv-settings-keyframe',
  segment: 'lv-settings-segment',
  size: 'lv-settings-size',
  overlap: 'lv-settings-overlap',
  duration: 'lv-settings-duration',
  style: 'lv-settings-style',
} as const;

const panelAriaLabel = computed(() =>
  props.mode === 'episode'
    ? $tt('video.longVideoEditorTabSettings')
    : $tt('video.longVideoSeriesTabSettings'),
);

const titleLabel = computed(() =>
  props.mode === 'episode'
    ? $tt('video.longVideoProjectTitle')
    : $tt('video.longVideoSeriesNameLabel'),
);

const titlePlaceholder = computed(() =>
  props.mode === 'episode'
    ? $tt('video.longVideoProjectTitlePh')
    : $tt('video.longVideoSeriesNamePh'),
);

const lockedOutputSizeLabel = computed(() => {
  const raw = (props.lockedOutputSize || props.outputSize || '').trim();
  if (!raw) return '—';
  const opt = props.outputSizeOptions.find((o) => o.value === raw);
  return opt ? formatResolutionOptionLabel(opt) : raw.replace(/x/i, '×');
});

/** Keep current value selectable even if registry list is sparse / stale. */
const keyframeSelectOptions = computed(() => {
  const opts = [...props.keyframeModelOptions];
  const cur = props.keyframeModel?.trim();
  if (cur && !opts.includes(cur)) opts.unshift(cur);
  return opts;
});

const sizeSelectOptions = computed(() => {
  const opts = [...props.outputSizeOptions];
  const cur = props.outputSize?.trim();
  if (cur && !opts.some((o) => o.value === cur)) {
    const [w, h] = cur.toLowerCase().split('x').map(Number);
    if (Number.isFinite(w) && Number.isFinite(h) && w > 0 && h > 0) {
      const g = (a: number, b: number): number => (b ? g(b, a % b) : a);
      const d = g(w, h) || 1;
      opts.unshift({
        label: w === h ? '1:1' : `${w / d}:${h / d}`,
        value: cur,
        pixelLabel: `${w}×${h}`,
      });
    } else {
      opts.unshift({ label: cur.replace(/x/i, '×'), value: cur, pixelLabel: cur.replace(/x/i, '×') });
    }
  }
  return opts;
});

const segmentSelectOptions = computed(() => {
  const opts = [...props.segmentModelOptions];
  const cur = props.segmentModel?.trim();
  if (cur && !opts.includes(cur)) opts.unshift(cur);
  return opts;
});

function overlapFrameLabel(n: number): string {
  return $tt('video.longVideoOverlapFramesOption', { n });
}
</script>

<style scoped>
.lv-settings--episode {
  width: 100%;
  margin: 0;
  padding: 0;
  align-self: stretch;
}

.lv-ep-settings {
  display: flex;
  flex-direction: column;
  width: 100%;
}

.lv-ep-settings__row {
  display: grid;
  grid-template-columns: 7.5em minmax(0, 1fr);
  gap: 8px 16px;
  align-items: center;
  min-width: 0;
  padding: 12px 0;
  border-bottom: 0.5px solid var(--dq-border-subtle);
}

.lv-ep-settings__row--title,
.lv-ep-settings__row--models {
  grid-template-columns: 7.5em minmax(0, 1fr);
}

.lv-ep-settings__row--title {
  display: contents;
}

.lv-ep-settings__row--models {
  display: contents;
}

.lv-ep-settings__row--title > .lv-ep-settings__field,
.lv-ep-settings__row--models > .lv-ep-settings__field,
.lv-ep-settings > .lv-ep-settings__field {
  display: grid;
  grid-template-columns: 7.5em minmax(0, 1fr);
  gap: 8px 16px;
  align-items: center;
  padding: 12px 0;
  border-bottom: 0.5px solid var(--dq-border-subtle);
}

.lv-ep-settings__field--top {
  align-items: start;
}

.lv-ep-settings__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.lv-ep-settings__label {
  margin: 0;
  font-size: 12px;
  font-weight: 550;
  color: var(--dq-label-secondary);
}

.lv-ep-settings__control {
  width: 100%;
  min-width: 0;
}

.lv-ep-settings__control :deep(.dq-select__trigger),
.lv-ep-settings__control.dq-input {
  min-height: 36px;
}

.lv-ep-settings__duration {
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: 160px;
}

.lv-ep-settings__unit {
  font-size: 13px;
  color: var(--dq-label-secondary);
  flex-shrink: 0;
}

.lv-ep-settings__meta {
  margin: 0;
  padding-top: 14px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 14px;
  font-size: 12px;
  line-height: 1.4;
  color: var(--dq-label-tertiary);
}

.lv-ep-settings__meta-link {
  margin: 0;
  padding: 0;
  border: 0;
  background: none;
  color: var(--dq-accent);
  font: inherit;
  cursor: pointer;
}

.lv-ep-settings__meta-link:hover {
  text-decoration: underline;
}

@media (max-width: 640px) {
  .lv-ep-settings__row--title > .lv-ep-settings__field,
  .lv-ep-settings__row--models > .lv-ep-settings__field,
  .lv-ep-settings > .lv-ep-settings__field {
    grid-template-columns: 1fr;
  }
}
</style>
