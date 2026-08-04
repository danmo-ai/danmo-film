import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import { resolve } from 'path';

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
    dedupe: ['vue'],
  },
  optimizeDeps: {
    include: ['reka-ui'],
    exclude: ['@danqing/dq-ui', '@danqing/dq-shell'],
  },
  server: {
    host: '127.0.0.1',
    port: Number(process.env.DQ_FRONTEND_PORT || 5803),
    strictPort: true,
    proxy: {
      '/api': {
        target: `http://127.0.0.1:${process.env.DQ_BACKEND_PORT || 7803}`,
        changeOrigin: true,
      },
      '/v1': {
        // LLM often points at Make; Film FE may call Make chat via env-configured base.
        target: `http://127.0.0.1:${process.env.FILM_MAKE_BASE_URL_PORT || process.env.DQ_MAKE_PORT || 7800}`,
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: resolve(__dirname, '../out/frontend/dist'),
    emptyOutDir: true,
    sourcemap: true,
  },
});
