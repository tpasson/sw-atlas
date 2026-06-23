// The version is injected at build time from package.json via Vite's `define`
// (see vite.config.js), so only the version string is bundled — not the whole
// package.json. package.json remains the single source of truth.
export const APP_VERSION = __APP_VERSION__
export const REPO_URL = 'https://github.com/tpasson/sw-atlas'
