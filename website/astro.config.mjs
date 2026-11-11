import { defineConfig } from 'astro/config';
import sitemap from '@astrojs/sitemap';

// The site is deployed under the repository path on GitHub Pages. English
// lives at the base, Chinese under zh/. Change `base` if the project moves
// to another repository or a custom domain.
const base = '/nsgo-hando/';

const releases = 'https://github.com/oneo-me/nsgo-hando/releases/latest';

export default defineConfig({
  site: 'https://oneo-me.github.io',
  base,
  trailingSlash: 'always',
  devToolbar: { enabled: false },
  integrations: [sitemap()],
  // Downloads open the latest GitHub release; keep the former page's links
  // working with a client redirect.
  redirects: {
    '/download': releases,
    '/zh/download': releases,
  },
});
