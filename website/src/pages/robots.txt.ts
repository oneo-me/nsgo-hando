import type { APIRoute } from 'astro';

// robots.txt points crawlers at the sitemap under the site's base path.
export const GET: APIRoute = ({ site }) => {
  const base = import.meta.env.BASE_URL;
  const sitemap = new URL(`${base}sitemap-index.xml`, site ?? 'https://oneo-me.github.io');
  return new Response(`User-agent: *\nAllow: /\n\nSitemap: ${sitemap}\n`, {
    headers: { 'Content-Type': 'text/plain; charset=utf-8' },
  });
};
