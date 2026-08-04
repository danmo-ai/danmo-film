<template>
  <div class="film-settings">
    <h1>{{ $tt('nav.settings') }}</h1>
    <p class="hint">
      Generation defaults to Danmo Make (<code>http://127.0.0.1:7800</code>).
      Override with env <code>FILM_MAKE_BASE_URL</code> / <code>FILM_GENERATION_PROVIDER</code>
      or <code>~/.danmo-film/config/settings.json</code>.
    </p>
    <pre v-if="status" class="status">{{ status }}</pre>
    <p v-if="error" class="error">{{ error }}</p>
    <button type="button" @click="refresh">Refresh provider status</button>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import axios from 'axios';

const status = ref('');
const error = ref('');

async function refresh() {
  error.value = '';
  try {
    const { data } = await axios.get('/api/providers/status');
    status.value = JSON.stringify(data, null, 2);
  } catch (e: any) {
    error.value = e?.message || String(e);
  }
}

onMounted(refresh);
</script>

<style scoped>
.film-settings { padding: 24px; max-width: 720px; }
.hint { opacity: 0.8; line-height: 1.5; }
.status { background: #111; color: #ddd; padding: 12px; overflow: auto; }
.error { color: #c44; }
button { margin-top: 12px; }
</style>
