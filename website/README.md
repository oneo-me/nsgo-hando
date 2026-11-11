# Hando website

The site at <https://oneo-me.github.io/nsgo-hando/>: static Astro pages
in English and Chinese, with no UI framework. It presents the app and its
LocalSend compatibility, and sends downloads to this repository's GitHub
releases.

```sh
bun install
bun run dev          # serve at http://localhost:4321/nsgo-hando/
bun run build        # typecheck, then write dist/
bun run preview      # serve the built dist/
bun run sync         # refresh the icon and screenshots from ../assets
bun run og           # redraw public/og.png
```

Astro 7 can leave the dev server running in the background; `bunx astro dev stop`
stops it.

`astro.config.mjs` fixes the base at `/nsgo-hando/` for GitHub Pages,
so the dev server also runs below that path. Change `base` for another
repository path, and `site` if the host changes; the canonical, robots and
social URLs follow them.

## Pages

| Page | Path | Chinese |
| --- | --- | --- |
| Home | `/` | `/zh/` |
| Privacy | `/privacy/` | `/zh/privacy/` |
| Not found | `/404.html` | English only |

The former download pages redirect to the latest GitHub release.

English lives at the base and Chinese under `zh/`. A first visit from a
Chinese browser opens the Chinese page; the language link remembers a
choice in `hando-lang`. Every page's words are in `src/copy.ts`; add both
languages together, and keep Chinese headings and titles free of a trailing
`。`. `src/site.ts` holds the languages, pages, outside links and platforms.

## Look

The app's palette: a hero in mint paper or deep forest with the page's
appearance, warm paper sections and mint cards, with the app's real screens
and small UI echoes in the feature cards. The appearance follows the system
unless a choice is made in the header, and the hero switches with it;
`hando-theme` keeps the choice. `src/styles/global.css` has all tokens and
layout; `src/components/Icon.astro` is the small line-icon set.

The app icon and screenshots in `src/assets/` and the favicons in `public/`
are copies of `../assets/Icon.png` and `../assets/screenshots/`. Run
`bun run sync` after they change, then `bun run og`. The copies are checked
in, so the site builds on its own. `public/og.png` is the social preview
drawn by `scripts/og.mjs`.

## Deploy

The [Website workflow](../.github/workflows/website.yml) builds `website/dist`
and publishes it to GitHub Pages on every push to `main` that changes the
site. The repository's Pages settings must use **GitHub Actions** as their
source once. A custom domain also needs `public/CNAME`, with `base` set to
`/` and `site` set to the domain in `astro.config.mjs`.
