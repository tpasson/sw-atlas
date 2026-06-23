import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// The app is served from the Go backend at the site root in production, so the
// base is '/'. In dev, Vite proxies /api to the backend (override the target
// with VITE_API_TARGET, e.g. http://localhost:8099).
export default defineConfig({
  plugins: [vue()],
  // Served at the site root by the Go backend; the GitHub Pages demo build
  // (VITE_DEMO=1) is served from the repo subpath instead.
  base: process.env.VITE_DEMO ? '/sw-atlas/' : '/',
  server: {
    proxy: {
      '/api': {
        target: process.env.VITE_API_TARGET || 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
