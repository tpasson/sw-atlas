import { readFileSync } from 'node:fs'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Read the version from package.json at build time and expose ONLY the version
// string to the client (via __APP_VERSION__), so the rest of package.json never
// ends up in the bundle. package.json stays the single source of truth.
const { version } = JSON.parse(readFileSync(new URL('./package.json', import.meta.url)))

// The app is served from the Go backend at the site root in production, so the
// base is '/'. In dev, Vite proxies /api to the backend (override the target
// with VITE_API_TARGET, e.g. http://localhost:8099).
export default defineConfig({
  plugins: [vue()],
  define: {
    __APP_VERSION__: JSON.stringify(version),
  },
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
