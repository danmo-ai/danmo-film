import { onUnmounted, ref, watch, type Ref } from 'vue';
import type { LongVideoProjectState } from '@/types';

export type LongVideoAutoSaveStatus = 'idle' | 'pending' | 'saving' | 'saved' | 'error';

const DEBOUNCE_MS = 2000;

export function useLongVideoAutoSave(options: {
  project: Ref<LongVideoProjectState | null>;
  suppress: Ref<number>;
  hasPersistableContent: (lv: LongVideoProjectState) => boolean;
  save: (opts: { silent: boolean }) => Promise<boolean>;
}) {
  const status = ref<LongVideoAutoSaveStatus>('idle');
  let timer: ReturnType<typeof setTimeout> | null = null;
  let chain: Promise<void> = Promise.resolve();

  function canAutoSave(lv: LongVideoProjectState | null): lv is LongVideoProjectState {
    if (!lv || options.suppress.value > 0) return false;
    return Boolean(lv.project_id) || options.hasPersistableContent(lv);
  }

  function cancelPending() {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
    if (status.value === 'pending') {
      status.value = 'idle';
    }
  }

  function scheduleSave() {
    cancelPending();
    if (!canAutoSave(options.project.value)) {
      return;
    }
    status.value = 'pending';
    timer = setTimeout(() => {
      timer = null;
      chain = chain.then(async () => {
        const lv = options.project.value;
        if (!canAutoSave(lv)) return;
        status.value = 'saving';
        const ok = await options.save({ silent: true });
        status.value = ok ? 'saved' : 'error';
      });
    }, DEBOUNCE_MS);
  }

  /** Cancel debounce and await in-flight / immediate save for the current project. */
  async function flush(): Promise<boolean> {
    cancelPending();
    if (!canAutoSave(options.project.value)) {
      await chain;
      return true;
    }
    status.value = 'saving';
    const run = chain.then(async () => {
      const lv = options.project.value;
      if (!canAutoSave(lv)) return true;
      const ok = await options.save({ silent: true });
      status.value = ok ? 'saved' : 'error';
      return ok;
    });
    chain = run.then(() => undefined);
    return run;
  }

  watch(
    () => options.project.value,
    () => scheduleSave(),
    { deep: true },
  );

  onUnmounted(() => cancelPending());

  return { status, cancelPending, flush };
}
