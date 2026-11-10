// Refresh the site's copies of the app's icon and screenshots, and the
// favicons made from the icon. Run `bun run sync` after the app's
// screenshots or icon change; `bun run og` redraws the social image too.

import sharp from 'sharp';
import { copyFile, mkdir } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

const root = new URL('../../', import.meta.url);
const assets = new URL('assets/', root);
const site = new URL('../', import.meta.url);

await mkdir(new URL('src/assets/screens/', site), { recursive: true });
await copyFile(new URL('Icon.png', assets), new URL('src/assets/icon.png', site));
for (const name of [
  'android-en.png',
  'android-zh-Hans.png',
  'ios-en.png',
  'ios-zh-Hans.png',
  'macos-en.png',
  'macos-zh-Hans.png',
]) {
  await copyFile(new URL(`screenshots/${name}`, assets), new URL(`src/assets/screens/${name}`, site));
}

const icon = fileURLToPath(new URL('src/assets/icon.png', site));
for (const [name, size] of [
  ['icon.png', 256],
  ['icon-192.png', 192],
  ['icon-512.png', 512],
  ['apple-touch-icon.png', 180],
]) {
  await sharp(icon).resize(size, size).toFile(fileURLToPath(new URL(`public/${name}`, site)));
}

console.log('assets synced');
