import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    vueDevTools(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    // #339. The API is a different port, so a browser treats every `/v1`
    // call from the dev server as cross-origin — and the Go server sends no
    // CORS headers. Without this proxy the app loads and reaches nothing.
    //
    // The compose stack's nginx carries the same rule (`web/nginx.conf`),
    // which is the point: dev and the shipped front door are both
    // same-origin, so `src/api/client.ts` can default to a relative base URL
    // and neither needs `VITE_API_BASE_URL` set.
    proxy: {
      '/v1': {
        target: process.env.VITE_DEV_API_TARGET ?? 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
