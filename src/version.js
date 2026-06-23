// Single source of truth for the app version, read from package.json at build time.
import pkg from '../package.json'

export const APP_VERSION = pkg.version
export const REPO_URL = 'https://github.com/tpasson/sw-atlas'
