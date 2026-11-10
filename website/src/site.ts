// The site's languages, pages, outside links and platforms. English lives
// at the base and Chinese under zh/; the GitHub Pages base comes from
// astro.config.mjs.

export type Lang = 'en' | 'zh';
export const langs: Lang[] = ['en', 'zh'];

export type PageId = 'home' | 'download' | 'privacy';

export const pages: { id: PageId; path: string }[] = [
  { id: 'home', path: '' },
  { id: 'download', path: 'download/' },
  { id: 'privacy', path: 'privacy/' },
];

/** localePath is a path below a language, below the site's base. */
export function localePath(lang: Lang, path = ''): string {
  return `${import.meta.env.BASE_URL}${lang === 'en' ? '' : `${lang}/`}${path}`;
}

/** pageUrl is one of the site's pages in a language. */
export function pageUrl(lang: Lang, id: PageId): string {
  return localePath(lang, pages.find((page) => page.id === id)!.path);
}

/** homeAnchor is a section of the home page, from any page. */
export function homeAnchor(lang: Lang, id: string): string {
  return `${pageUrl(lang, 'home')}#${id}`;
}

/** langPaths gives a page's route for every language, for getStaticPaths. */
export function langPaths() {
  return langs.map((lang) => ({ params: { lang: lang === 'en' ? undefined : lang } }));
}

export function langOf(param: string | undefined): Lang {
  return param === 'zh' ? 'zh' : 'en';
}

const repo = 'https://github.com/oneo-me/nsgo-hando';

export const links = {
  repo,
  releases: `${repo}/releases/latest`,
  allReleases: `${repo}/releases`,
  issues: `${repo}/issues`,
  sponsor: 'https://github.com/sponsors/oneo-me',
  protocol: 'https://github.com/localsend/protocol',
  license: `${repo}/blob/main/LICENSE`,
  changelog: `${repo}/blob/main/CHANGELOG.md`,
  workflow: `${repo}/blob/main/.github/workflows/release.yml`,
  blog: 'https://oneo.me',
  x: 'https://x.com/oneo666',
  nsgo: 'https://github.com/oneo-me/nsgo',
};

export type PlatformId = 'macos' | 'windows' | 'linux' | 'android' | 'ios';

/** platforms lists the packages the release offers, in download order. */
export const platforms: { id: PlatformId; name: string }[] = [
  { id: 'macos', name: 'macOS' },
  { id: 'windows', name: 'Windows' },
  { id: 'linux', name: 'Linux' },
  { id: 'android', name: 'Android' },
  { id: 'ios', name: 'iOS' },
];

export const year = new Date().getFullYear();
