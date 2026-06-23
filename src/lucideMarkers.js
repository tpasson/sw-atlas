// The full Lucide icon set is available as item markers. Shapes referencing a
// Lucide icon use the "l:" prefix, e.g. "l:Save". (Importing the whole namespace
// bundles all icons — that's the trade-off for offering the complete library.)
import * as Lucide from 'lucide-vue-next'

// Icons are functional components. Lucide also exports alias names
// ("SaveIcon", "LucideSave") for each — keep only the canonical PascalCase name.
function isIconName(k) {
  if (!/^[A-Z][A-Za-z0-9]*$/.test(k)) return false
  if (k === 'Icon' || k.endsWith('Icon') || k.startsWith('Lucide')) return false
  return typeof Lucide[k] === 'function'
}

export const LUCIDE_MARKERS = Lucide
export const LUCIDE_MARKER_SHAPES = Object.keys(Lucide).filter(isIconName).sort().map(n => 'l:' + n)
