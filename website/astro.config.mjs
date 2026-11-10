import { defineConfig } from 'astro/config';
import sitemap from '@astrojs/sitemap';

// The site is deployed under the repository path on GitHub Pages. English
// lives at the base, Chinese under zh/. Change `base` if the project moves
// to another repository or a custom domain.
const base = '/nsgo-hando/';

export default defineConfig({
  site: 'https://oneo-me.github.io',
  base,
  trailingSlash: 'always',
  devToolbar: { enabled: false },
  integrations: [sitemap()],
});
