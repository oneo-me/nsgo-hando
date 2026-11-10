// Draws public/og.png from the app icon and a real screenshot, in the
// site's forest and mint. Run `bun run og` after changing the hero artwork.

import sharp from 'sharp';
import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

const icon = (await readFile(new URL('../src/assets/icon.png', import.meta.url))).toString('base64');
const shot = (await readFile(new URL('../src/assets/screens/macos-en.png', import.meta.url))).toString(
  'base64',
);

const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630">
  <defs>
    <radialGradient id="glow" cx="82%" cy="8%" r="72%">
      <stop offset="0" stop-color="#2dd4a7" stop-opacity=".22"/>
      <stop offset=".55" stop-color="#2dd4a7" stop-opacity=".05"/>
      <stop offset="1" stop-color="#2dd4a7" stop-opacity="0"/>
    </radialGradient>
    <pattern id="dots" width="24" height="24" patternUnits="userSpaceOnUse">
      <circle cx="1" cy="1" r="1" fill="#eaf3ee" opacity=".5"/>
    </pattern>
    <linearGradient id="fade" x1="0" x2="0" y1="0" y2="1">
      <stop offset="0" stop-color="#fff" stop-opacity=".5"/>
      <stop offset="1" stop-color="#fff" stop-opacity="0"/>
    </linearGradient>
    <mask id="dotmask">
      <rect width="1200" height="630" fill="url(#fade)"/>
    </mask>
    <filter id="shot" x="-20%" y="-20%" width="140%" height="140%">
      <feDropShadow dx="0" dy="26" stdDeviation="26" flood-color="#03110b" flood-opacity=".65"/>
    </filter>
  </defs>

  <rect width="1200" height="630" fill="#0c211a"/>
  <rect width="1200" height="630" fill="url(#glow)"/>
  <rect width="1200" height="630" fill="url(#dots)" opacity=".28" mask="url(#dotmask)"/>
  <circle cx="1140" cy="40" r="260" fill="none" stroke="#eaf3ee" stroke-opacity=".07"/>

  <image href="data:image/png;base64,${icon}" x="80" y="72" width="88" height="88"/>
  <text x="184" y="133" fill="#eaf3ee" font-family="Helvetica Neue, Helvetica, Arial, sans-serif" font-size="36" font-weight="600" letter-spacing="-1">Hando</text>

  <rect x="80" y="218" width="42" height="2" rx="1" fill="#35d8ab"/>
  <text x="80" y="292" fill="#eaf3ee" font-family="Helvetica Neue, Helvetica, Arial, sans-serif" font-size="64" font-weight="600" letter-spacing="-2.4">Send files.</text>
  <text x="80" y="364" fill="#35d8ab" font-family="Helvetica Neue, Helvetica, Arial, sans-serif" font-size="64" font-weight="600" letter-spacing="-2.4">Keep them close.</text>
  <text x="80" y="428" fill="#a4b8ae" font-family="Helvetica Neue, Helvetica, Arial, sans-serif" font-size="23">Private file sharing on your local network.</text>
  <text x="80" y="558" fill="#7f948a" font-family="SF Mono, Menlo, Consolas, monospace" font-size="13" letter-spacing="1.4">LOCALSEND COMPATIBLE · MACOS · WINDOWS · LINUX · IOS · ANDROID</text>

  <g transform="rotate(-1.2 870 350)" filter="url(#shot)">
    <image href="data:image/png;base64,${shot}" x="610" y="150" width="520" height="389"/>
  </g>
</svg>`;

await sharp(Buffer.from(svg)).png({ compressionLevel: 9 }).toFile(fileURLToPath(new URL('../public/og.png', import.meta.url)));
console.log('public/og.png');
