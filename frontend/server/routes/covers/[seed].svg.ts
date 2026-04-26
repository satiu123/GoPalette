const PALETTES = [
  ['#0f766e', '#84cc16', '#f8fafc'],
  ['#1d4ed8', '#06b6d4', '#f8fafc'],
  ['#7c2d12', '#f59e0b', '#fff7ed'],
  ['#334155', '#14b8a6', '#f1f5f9'],
  ['#4c1d95', '#db2777', '#fdf2f8'],
  ['#14532d', '#22c55e', '#f0fdf4']
]

function hashSeed(seed: string) {
  let hash = 0
  for (const char of seed) {
    hash = ((hash << 5) - hash) + char.charCodeAt(0)
    hash |= 0
  }
  return Math.abs(hash)
}

function escapeXml(value: string) {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;')
}

export default defineEventHandler((event) => {
  const params = getRouterParams(event)
  const seed = decodeURIComponent(String(params.seed || 'gopalette')).replace(/\.svg$/i, '')
  const hash = hashSeed(seed)
  const palette = PALETTES[hash % PALETTES.length] || PALETTES[0]!
  const title = escapeXml(seed.replace(/[-_]+/g, ' ').slice(0, 42) || 'GoPalette')
  const patternOffset = 80 + (hash % 140)

  setHeader(event, 'content-type', 'image/svg+xml; charset=utf-8')
  setHeader(event, 'cache-control', 'public, max-age=31536000, immutable')

  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1200 640" role="img" aria-label="${title}">
  <defs>
    <linearGradient id="bg" x1="0" x2="1" y1="0" y2="1">
      <stop offset="0%" stop-color="${palette[0]}"/>
      <stop offset="58%" stop-color="${palette[1]}"/>
      <stop offset="100%" stop-color="${palette[2]}"/>
    </linearGradient>
    <pattern id="grid" width="56" height="56" patternUnits="userSpaceOnUse" patternTransform="rotate(18)">
      <path d="M 56 0 L 0 0 0 56" fill="none" stroke="rgba(255,255,255,.18)" stroke-width="1"/>
    </pattern>
  </defs>
  <rect width="1200" height="640" fill="url(#bg)"/>
  <rect width="1200" height="640" fill="url(#grid)" opacity=".7"/>
  <circle cx="${patternOffset}" cy="140" r="220" fill="rgba(255,255,255,.16)"/>
  <circle cx="${1000 - (hash % 120)}" cy="${460 + (hash % 80)}" r="260" fill="rgba(15,23,42,.16)"/>
  <path d="M0 470 C 260 390, 390 560, 640 472 S 1010 330, 1200 420 L1200 640 L0 640 Z" fill="rgba(255,255,255,.2)"/>
  <text x="72" y="500" fill="rgba(255,255,255,.92)" font-family="Georgia, 'Times New Roman', serif" font-size="54" font-weight="700">${title}</text>
  <text x="74" y="548" fill="rgba(255,255,255,.72)" font-family="Arial, sans-serif" font-size="20" letter-spacing="4">GOPALETTE BLOG</text>
</svg>`
})
