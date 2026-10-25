<template>
  <div class="settings-panel">
    <header class="settings-panel__head">
      <h2>{{ $tt('settings.general') }}</h2>
      <p>{{ $tt('settings.generalDesc') }}</p>
    </header>
    <DqPrefPane class="settings-grouped-form settings-pref-pane-form">
      <DqPrefRow :label="$tt('settings.theme')">
        <DqSelect
          size="sm"
          :model-value="themeStore.currentTheme"
          @update:model-value="(v: string) => themeStore.setTheme(v as ThemeId)"
        >
          <DqOption
            v-for="opt in THEME_OPTIONS"
            :key="opt.id"
            :value="opt.id"
            :label="opt.label"
          />
        </DqSelect>
      </DqPrefRow>
      <DqPrefRow :label="$tt('settings.language')">
        <DqSelect size="sm" :model-value="locale" @update:model-value="onLocale">
          <DqOption value="zh" label="中文" />
          <DqOption value="en" label="English" />
        </DqSelect>
      </DqPrefRow>
    </DqPrefPane>

    <header class="settings-panel__head">
      <h2>{{ $tt('settings.workspace') }}</h2>
      <p>{{ $tt('settings.workspaceDesc') }}</p>
    </header>
    <DqPrefPane class="settings-grouped-form settings-pref-pane-form">
      <DqPrefRow
        v-for="row in pathRows"
        :key="row.key"
        :label="row.label"
      >
        <code class="film-page__path">{{ row.value }}</code>
      </DqPrefRow>
    </DqPrefPane>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { $tt } from '@/utils/i18n';
import { THEME_OPTIONS, useThemeStore, type ThemeId } from '@/stores/theme';
import { setItem, DQ_STORAGE } from '@/utils/storage';

defineProps<{
  pathRows: Array<{ key: string; label: string; value: string }>;
}>();

const { locale } = useI18n();
const themeStore = useThemeStore();

function onLocale(v: unknown) {
  const lang = String(v ?? 'zh');
  locale.value = lang;
  setItem(DQ_STORAGE.LANG, lang);
}
</script>
