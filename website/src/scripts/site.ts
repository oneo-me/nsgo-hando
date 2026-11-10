// The site's small behaviors: the appearance switch, the mobile menu, the
// screenshot tabs and a gentle reveal of sections. Content stays visible
// when a browser can't run this.

type Theme = 'light' | 'dark';
type ThemePreference = Theme | 'system';

const themeKey = 'hando-theme';
const languageKey = 'hando-lang';
const media = window.matchMedia('(prefers-color-scheme: dark)');

function storedPreference(): ThemePreference {
  try {
    const stored = localStorage.getItem(themeKey);
    if (stored === 'light' || stored === 'dark' || stored === 'system') return stored;
  } catch {
    // Storage can be unavailable; follow the system.
  }
  return 'system';
}

function applyTheme(preference: ThemePreference): void {
  const theme: Theme = preference === 'system' ? (media.matches ? 'dark' : 'light') : preference;
  const root = document.documentElement;
  root.dataset.theme = theme;
  root.dataset.themePreference = preference;
  root.style.colorScheme = theme;
  document
    .querySelector('meta[name="theme-color"]')
    ?.setAttribute('content', theme === 'dark' ? '#0b1310' : '#f6f8f5');
  updateThemeSwitch();
}

function updateThemeSwitch(): void {
  const button = document.querySelector<HTMLButtonElement>('[data-theme-switch]');
  if (!button) return;
  const theme = document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light';
  const label = theme === 'dark' ? button.dataset.labelDark : button.dataset.labelLight;
  if (!label) return;
  button.setAttribute('aria-label', label);
  button.title = label;
}

function setTheme(preference: ThemePreference): void {
  try {
    localStorage.setItem(themeKey, preference);
  } catch {
    // The appearance still applies for this visit.
  }
  applyTheme(preference);
}

updateThemeSwitch();
document.querySelector<HTMLButtonElement>('[data-theme-switch]')?.addEventListener('click', () => {
  setTheme(document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark');
});
media.addEventListener('change', () => {
  if (storedPreference() === 'system') applyTheme('system');
});

// The inline head script records the device's OS; label each download once
// the markup exists.
const os = document.documentElement.dataset.os ?? 'other';
document.querySelectorAll<HTMLElement>('[data-download-label]').forEach((element) => {
  let labels: Record<string, string> = {};
  try {
    labels = JSON.parse(element.dataset.osLabels ?? '{}') as Record<string, string>;
  } catch {
    return;
  }
  const label = labels[os];
  if (label) element.textContent = label;
});

// The language link remembers the choice, so the first-visit redirect does
// not undo it.
document.querySelectorAll<HTMLAnchorElement>('[data-lang]').forEach((link) => {
  link.addEventListener('click', () => {
    try {
      localStorage.setItem(languageKey, link.dataset.lang ?? (document.documentElement.lang === 'zh-CN' ? 'zh' : 'en'));
    } catch {
      // The link still opens the language.
    }
  });
});

// The header turns solid on scroll, and its menu opens below it on phones.
const header = document.querySelector<HTMLElement>('[data-header]');
const burger = document.querySelector<HTMLButtonElement>('.header__burger');

function setMenu(open: boolean): void {
  if (!header || !burger) return;
  header.classList.toggle('is-open', open);
  burger.setAttribute('aria-expanded', String(open));
  const label = open ? burger.dataset.closeLabel : burger.dataset.openLabel;
  if (label) burger.setAttribute('aria-label', label);
  document.documentElement.classList.toggle('has-open-menu', open);
}

if (header) {
  const update = () => header.classList.toggle('is-scrolled', window.scrollY > 6);
  window.addEventListener('scroll', update, { passive: true });
  update();
}

burger?.addEventListener('click', () => setMenu(!header?.classList.contains('is-open')));
header?.querySelectorAll('a').forEach((link) => link.addEventListener('click', () => setMenu(false)));
document.addEventListener('keydown', (event) => {
  if (event.key === 'Escape') setMenu(false);
});
document.addEventListener('click', (event) => {
  if (!header?.classList.contains('is-open')) return;
  if (!(event.target instanceof Node) || !header.contains(event.target)) setMenu(false);
});
window.matchMedia('(min-width: 901px)').addEventListener('change', () => setMenu(false));

// Screenshot tabs: one visible panel, with the arrow keys moving between
// the tabs.
document.querySelectorAll<HTMLElement>('[data-tabs]').forEach((root) => {
  const tabs = Array.from(root.querySelectorAll<HTMLButtonElement>('[role="tab"]'));
  const panels = Array.from(root.querySelectorAll<HTMLElement>('[role="tabpanel"]'));
  if (tabs.length === 0) return;

  const select = (index: number, focus = false) => {
    tabs.forEach((tab, i) => {
      const active = i === index;
      tab.setAttribute('aria-selected', String(active));
      tab.tabIndex = active ? 0 : -1;
    });
    panels.forEach((panel, i) => {
      panel.hidden = i !== index;
    });
    if (focus) tabs[index]?.focus();
  };

  tabs.forEach((tab, index) => {
    tab.addEventListener('click', () => select(index));
    tab.addEventListener('keydown', (event) => {
      let next = -1;
      if (event.key === 'ArrowRight') next = (index + 1) % tabs.length;
      else if (event.key === 'ArrowLeft') next = (index - 1 + tabs.length) % tabs.length;
      else if (event.key === 'Home') next = 0;
      else if (event.key === 'End') next = tabs.length - 1;
      if (next === -1) return;
      event.preventDefault();
      select(next, true);
    });
  });
});

// Sections rise in once, unless the visitor asked for less motion. The
// class is added only when the observer can hide them first.
if ('IntersectionObserver' in window && !window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
  const targets = Array.from(document.querySelectorAll<HTMLElement>('[data-reveal]'));
  if (targets.length > 0) {
    document.documentElement.classList.add('has-reveal');
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue;
          entry.target.classList.add('is-visible');
          observer.unobserve(entry.target);
        }
      },
      { rootMargin: '0px 0px -8% 0px', threshold: 0.06 },
    );
    targets.forEach((target) => observer.observe(target));
  }
}
