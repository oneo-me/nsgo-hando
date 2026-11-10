// Every word the site shows, in English and Simplified Chinese. Add both
// languages together; Chinese headings, titles and eyebrows omit the
// trailing 。

import type { Lang } from './site';

export interface Copy {
  nav: {
    how: string;
    features: string;
    compatibility: string;
    download: string;
    menu: string;
    primary: string;
    close: string;
    skip: string;
    themeLight: string;
    themeDark: string;
    language: string;
  };
  common: {
    download: string;
    downloadFor: Record<'macos' | 'windows' | 'linux' | 'android' | 'ios', string>;
    free: string;
    latest: string;
    allReleases: string;
    source: string;
    github: string;
    external: string;
    home: string;
    backHome: string;
    faq: string;
  };
  mock: {
    nearby: string;
    online: string;
    you: string;
    mac: string;
    iphone: string;
    pc: string;
    server: string;
    quickSave: string;
    sending: string;
    verified: string;
    received: string;
    sent: string;
    folder: string;
    message: string;
    light: string;
    dark: string;
    history: string;
    pin: string;
    noCloud: string;
    localNetwork: string;
  };
  home: {
    title: string;
    description: string;
    hero: {
      badge: string;
      title: [string, string];
      lead: string;
      how: string;
      note: string;
      macAlt: string;
      phoneAlt: string;
      nearby: string;
      noInternet: string;
      sending: string;
      file: string;
      size: string;
      progress: string;
      verified: string;
    };
    factsTitle: string;
    facts: { title: string; body: string }[];
    how: {
      eyebrow: string;
      title: string;
      lead: string;
      steps: { title: string; body: string }[];
    };
    features: {
      eyebrow: string;
      title: string;
      lead: string;
      send: { title: string; body: string };
      favorites: { title: string; body: string };
      pin: { title: string; body: string };
      checksum: { title: string; body: string };
      history: { title: string; body: string };
      calm: { title: string; body: string };
    };
    screens: {
      eyebrow: string;
      title: string;
      lead: string;
      tabs: Record<'macos' | 'ios' | 'android', string>;
      captions: Record<'macos' | 'ios' | 'android', string>;
      alts: Record<'macos' | 'ios' | 'android', string>;
    };
    compatibility: {
      eyebrow: string;
      title: string;
      lead: string;
      official: string;
      protocol: string;
      specs: { label: string; value: string }[];
      tested: string;
      protocolLink: string;
      notAffiliated: string;
    };
    privacy: {
      eyebrow: string;
      title: string[];
      lead: string;
      points: string[];
      link: string;
    };
    download: {
      eyebrow: string;
      title: string;
      lead: string;
      all: string;
    };
    faq: {
      eyebrow: string;
      title: string;
      items: { q: string; a: string }[];
    };
  };
  download: {
    title: string;
    description: string;
    hero: {
      eyebrow: string;
      title: string;
      lead: string;
      latest: string;
      note: string;
    };
    platforms: Record<'macos' | 'windows' | 'linux' | 'android' | 'ios', { note: string; format: string }>;
    open: string;
    you: string;
    after: {
      title: string;
      lead: string;
      steps: Record<'macos' | 'windows' | 'linux' | 'android' | 'ios', string>;
    };
    needs: { title: string; items: string[] };
    verify: { title: string; body: string; workflow: string; source: string };
    build: { title: string; body: string; note: string };
  };
  privacy: {
    title: string;
    description: string;
    hero: { eyebrow: string; title: string; lead: string };
    summary: [string, string, string];
    app: { title: string; paragraphs: string[] };
    website: { title: string; paragraphs: string[] };
    contact: { title: string; body: string };
  };
  notFound: {
    title: string;
    description: string;
    heading: string;
    body: string;
  };
  footer: {
    tagline: string;
    product: string;
    project: string;
    elsewhere: string;
    download: string;
    features: string;
    compatibility: string;
    privacy: string;
    changelog: string;
    license: string;
    protocol: string;
    sponsor: string;
    blog: string;
    builtWith: string;
    rights: string;
    notAffiliated: string;
  };
}

const en: Copy = {
  nav: {
    how: 'How it works',
    features: 'Features',
    compatibility: 'Compatibility',
    download: 'Download',
    menu: 'Menu',
    primary: 'Primary',
    close: 'Close',
    skip: 'Skip to content',
    themeLight: 'Switch to dark appearance',
    themeDark: 'Switch to light appearance',
    language: 'Language',
  },
  common: {
    download: 'Download Hando',
    downloadFor: {
      macos: 'Download for macOS',
      windows: 'Download for Windows',
      linux: 'Download for Linux',
      android: 'Download for Android',
      ios: 'Download for iOS',
    },
    free: 'Free and open source',
    latest: 'Latest release',
    allReleases: 'All releases',
    source: 'Source',
    github: 'GitHub',
    external: 'Opens in a new tab',
    home: 'Home',
    backHome: 'Back to home',
    faq: 'Questions',
  },
  mock: {
    nearby: 'Nearby devices',
    online: '3 devices online',
    you: 'You',
    mac: 'My Mac',
    iphone: "Ada's iPhone",
    pc: 'Studio PC',
    server: 'Home Server',
    quickSave: 'Quick Save',
    sending: 'Sending',
    verified: 'Verified',
    received: 'Received',
    sent: 'Sent',
    folder: 'Summer photos',
    message: 'On my way',
    light: 'Light',
    dark: 'Dark',
    history: 'History',
    pin: 'PIN',
    noCloud: 'No cloud',
    localNetwork: 'Local network',
  },
  home: {
    title: 'Hando — Private file sharing on your local network',
    description:
      'Send files, folders and messages between Mac, Windows, Linux, iOS and Android over your local network. Free, open source and compatible with LocalSend.',
    hero: {
      badge: 'LocalSend compatible · Free and open source',
      title: ['Send files.', 'Keep them close.'],
      lead: 'Hando hands files, folders and messages straight from one device to another over your Wi-Fi. Mac to PC. iPhone to Android. No cloud, no account, no upload.',
      how: 'See how it works',
      note: 'MIT licensed · macOS, Windows, Linux, iOS and Android',
      macAlt: 'Hando on macOS, with nearby devices and favorites',
      phoneAlt: 'Hando on iPhone, showing nearby devices',
      nearby: '3 devices nearby',
      noInternet: 'No internet needed',
      sending: "Sending to Ada's iPhone",
      file: 'Summer photos.zip',
      size: '1.2 GB',
      progress: '68%',
      verified: 'SHA-256 on',
    },
    factsTitle: 'Why Hando',
    facts: [
      {
        title: 'On your network',
        body: 'Direct device-to-device transfers. Nothing is uploaded, and no server sits in the middle.',
      },
      {
        title: 'Encrypted by default',
        body: 'HTTPS with mutual TLS, pinned to the peer’s fingerprint. Plain HTTP stays for older setups.',
      },
      {
        title: 'Speaks LocalSend',
        body: 'The official LocalSend apps use the same protocol. Hando is not required on both sides.',
      },
    ],
    how: {
      eyebrow: 'How it works',
      title: 'Three steps. One Wi-Fi.',
      lead: 'No pairing codes, no accounts, no setup. Open Hando on both devices and it finds the rest.',
      steps: [
        {
          title: 'Find each other',
          body: 'Both devices announce themselves on the local network. Nearby devices appear in a moment, and favorites stay in the list even while offline.',
        },
        {
          title: 'Pick what to send',
          body: 'Choose files or a whole folder, or write a short message. Then tap the device beside you.',
        },
        {
          title: 'They accept. It’s done.',
          body: 'They review the request and accept. Progress is visible on both ends, and checksums are verified as the files land.',
        },
      ],
    },
    features: {
      eyebrow: 'Features',
      title: 'A small app that thinks of everything.',
      lead: 'The details that make a transfer feel effortless, on the desk and in your hand.',
      send: {
        title: 'Everything you’d carry.',
        body: 'Files, whole folders or a quick message. Queue several, watch each one, and cancel a transfer you change your mind about.',
      },
      favorites: {
        title: 'The people you keep close.',
        body: 'Star a device to keep it in reach while it’s offline. Quick Save accepts its transfers without a prompt — for the phone on your desk, not the café downstairs.',
      },
      pin: {
        title: 'A PIN for everyone else.',
        body: 'Set a receive PIN so only the people you tell can send to you.',
      },
      checksum: {
        title: 'Checksums, not hope.',
        body: 'When the sender offers a SHA-256, Hando verifies it before a received file gets its final name.',
      },
      history: {
        title: 'A history you can reopen.',
        body: 'Every finished transfer stays in the app, ready to open its folder.',
      },
      calm: {
        title: 'Native at a glance.',
        body: 'A sidebar on desktops, a tab bar on phones, light and dark, English and Simplified Chinese.',
      },
    },
    screens: {
      eyebrow: 'Screens',
      title: 'At home on every screen.',
      lead: 'The same app, shaped for the device in your hand.',
      tabs: { macos: 'macOS', ios: 'iPhone', android: 'Android' },
      captions: {
        macos: 'A sidebar for the desk. Favorites, nearby devices and transfers in one window.',
        ios: 'A tab bar for the thumb. Nearby, transfers and settings, one tap apart.',
        android: 'The same layout on Android, in light or dark.',
      },
      alts: {
        macos: 'Hando on macOS with a sidebar, nearby devices and favorites',
        ios: 'Hando on iPhone with nearby devices and a tab bar',
        android: 'Hando on Android with nearby devices and a tab bar',
      },
    },
    compatibility: {
      eyebrow: 'Compatibility',
      title: 'It already speaks LocalSend.',
      lead: 'Hando implements LocalSend protocol v2.2, so it sends to and receives from the official apps. The other person doesn’t need Hando.',
      official: 'Official app or CLI',
      protocol: 'Protocol v2.2',
      specs: [
        {
          label: 'Discovery',
          value: 'IPv4 multicast on 224.0.0.167:53317, with a local network scan as fallback.',
        },
        {
          label: 'Transport',
          value: 'HTTPS with mutual TLS by default; plain HTTP for older setups.',
        },
        {
          label: 'Integrity',
          value: 'SHA-256 checksums verified before a received file is saved.',
        },
        {
          label: 'Transfers',
          value: 'Multiple files, folder structure, text messages and receive PINs.',
        },
      ],
      tested: 'Checked by hand against LocalSend CLI 1.18.2, sending and receiving.',
      protocolLink: 'Read the protocol',
      notAffiliated: 'Hando is not affiliated with the LocalSend project.',
    },
    privacy: {
      eyebrow: 'Privacy',
      title: ['No cloud.', 'No account.', 'No detour.'],
      lead: 'The shortest path between two devices is the one already in the room. Hando uses it and nothing else.',
      points: [
        'Files travel directly between the two devices.',
        'TLS with mutual certificates, pinned per transfer.',
        'Received files stay in the folder you choose.',
      ],
      link: 'How Hando handles your data',
    },
    download: {
      eyebrow: 'Download',
      title: 'Take it with you.',
      lead: 'Free and open source under the MIT license. Get the package for your platform from the latest release.',
      all: 'All releases',
    },
    faq: {
      eyebrow: 'Questions',
      title: 'Good to know.',
      items: [
        {
          q: 'Do both devices need Hando?',
          a: 'No. Hando speaks the LocalSend protocol, so the other device can run the official LocalSend app. Hando on both sides adds favorites and Quick Save.',
        },
        {
          q: 'Does it need the internet?',
          a: 'No. Transfers use the local network only. If devices don’t find each other, check that the Wi-Fi isn’t isolating clients; Hando also scans the subnet as a fallback.',
        },
        {
          q: 'Is anything uploaded?',
          a: 'No. There is no Hando server, account or cloud. Files travel directly from one device to the other.',
        },
        {
          q: 'What can I send?',
          a: 'Files, folders and text messages, with per-file progress and cancellation. Received files go to Downloads/Hando on desktops or the app’s folder on phones, and you can choose another folder in Settings.',
        },
        {
          q: 'How does it stay compatible?',
          a: 'It implements LocalSend v2.2: multicast discovery, mutual TLS with pinned fingerprints and SHA-256 checksums. It has been tested against LocalSend CLI 1.18.2 in both directions.',
        },
        {
          q: 'Is it really free?',
          a: 'Yes. Hando is MIT-licensed, and its source and release workflow are on GitHub. If it saves you time, you can sponsor its development.',
        },
        {
          q: 'Can I install it on iPhone?',
          a: 'The iOS package is an unsigned .ipa unless a signing profile is configured. You’ll need to sign it with your own Apple ID or a developer profile before installing.',
        },
      ],
    },
  },
  download: {
    title: 'Download Hando — macOS, Windows, Linux, iOS and Android',
    description:
      'Download Hando for macOS, Windows, Linux, Android or iOS. Free, open source, and built from source by GitHub Actions.',
    hero: {
      eyebrow: 'Download',
      title: 'Get Hando.',
      lead: 'The latest release, built for your platform. Free and MIT-licensed; every package comes from the release workflow.',
      latest: 'Open the latest release',
      note: 'Version notes and all files are on GitHub.',
    },
    platforms: {
      macos: { note: 'For Apple silicon and Intel Macs.', format: 'DMG' },
      windows: { note: 'Installer for x64 and arm64 PCs.', format: 'Setup .exe' },
      linux: { note: 'For x64 and arm64: a .deb package or an Arch package.', format: 'DEB · pkg.tar.gz' },
      android: { note: 'APK for arm64 phones and tablets.', format: 'APK' },
      ios: {
        note: 'Unsigned IPA; sign it with your own Apple ID or developer profile.',
        format: 'IPA',
      },
    },
    open: 'Open release',
    you: 'Your platform',
    after: {
      title: 'After the download',
      lead: 'Each package is a normal app install. A few notes are worth knowing.',
      steps: {
        macos:
          'Open the .dmg and drag Hando to Applications. If macOS says the app can’t be verified, allow it in System Settings → Privacy & Security.',
        windows:
          'Run the setup .exe. The installer is not distributed through a store, so Windows may ask for confirmation.',
        linux: 'Install with sudo apt install ./Hando-*.deb, or sudo pacman -U Hando-*.pkg.tar.gz.',
        android: 'Open the APK. Android will ask you to allow installs from your browser or file manager.',
        ios: 'The IPA is unsigned. Sign it with AltStore, Sideloadly or your own developer profile, then trust it in Settings.',
      },
    },
    needs: {
      title: 'What you need',
      items: [
        'Both devices on the same Wi-Fi or local network.',
        'macOS, Windows, Linux, Android or iOS — Hando runs natively on each.',
        'No account, no subscription and no internet connection.',
      ],
    },
    verify: {
      title: 'Built in the open',
      body: 'Releases are built by GitHub Actions from the repository and published with the version’s changelog. The source is MIT-licensed.',
      workflow: 'Release workflow',
      source: 'Source code',
    },
    build: {
      title: 'Build it yourself',
      body: 'Hando needs Go 1.26 or later. These commands build every target and run the test suite.',
      note: 'Apple targets need Xcode, and Android needs the Android SDK’s platform-tools.',
    },
  },
  privacy: {
    title: 'Privacy — Hando',
    description: 'How Hando and this website handle your data: local transfers, no accounts and no analytics.',
    hero: {
      eyebrow: 'Privacy',
      title: 'Privacy, by architecture.',
      lead: 'Hando has nothing to collect: no account, no server and no upload. This page explains what that means for the app and for this website.',
    },
    summary: ['No account', 'No analytics', 'No upload'],
    app: {
      title: 'The app',
      paragraphs: [
        'Hando creates no account and connects to no Hando service. Transfers go directly between the two devices on your local network.',
        'Transfers use HTTPS with mutual TLS by default. The client pins the peer’s certificate fingerprint, so a transfer cannot be redirected to another device answering on the same address. Older setups can use plain HTTP, which you can choose in Settings.',
        'Device identity, favorites, settings and transfer history are stored on your device. Received files stay in the receive folder you choose.',
        'When you send to or receive from the official LocalSend app, that app handles data on its side under its own policy.',
      ],
    },
    website: {
      title: 'This website',
      paragraphs: [
        'The site is a set of static pages served by GitHub Pages. It sets no cookies and runs no analytics or advertising.',
        'The appearance and language you choose are kept in your browser’s local storage. Nothing is sent to us.',
        'GitHub, as the host, may keep standard access logs under GitHub’s privacy statement.',
        'Links to GitHub, X and other sites leave this site; those sites have their own policies.',
      ],
    },
    contact: {
      title: 'Questions',
      body: 'Open an issue on GitHub. Please don’t include personal data in a public issue.',
    },
  },
  notFound: {
    title: 'Page not found — Hando',
    description: 'This page is not on the local network. Return to Hando’s home page.',
    heading: 'This page went off the grid.',
    body: 'The link may be wrong, or the page moved. Head back home and try again.',
  },
  footer: {
    tagline: 'Private file sharing on your local network.',
    product: 'Product',
    project: 'Project',
    elsewhere: 'Elsewhere',
    download: 'Download',
    features: 'Features',
    compatibility: 'Compatibility',
    privacy: 'Privacy',
    changelog: 'Changelog',
    license: 'License',
    protocol: 'LocalSend protocol',
    sponsor: 'Sponsor',
    blog: 'Blog',
    builtWith: 'Built with NSGo',
    rights: '© {year} Hando',
    notAffiliated: 'Not affiliated with the LocalSend project.',
  },
};

const zh: Copy = {
  nav: {
    how: '工作原理',
    features: '功能',
    compatibility: '兼容性',
    download: '下载',
    menu: '菜单',
    primary: '主导航',
    close: '关闭',
    skip: '跳到正文',
    themeLight: '切换到深色外观',
    themeDark: '切换到浅色外观',
    language: '语言',
  },
  common: {
    download: '下载 Hando',
    downloadFor: {
      macos: '下载 macOS 版',
      windows: '下载 Windows 版',
      linux: '下载 Linux 版',
      android: '下载 Android 版',
      ios: '下载 iOS 版',
    },
    free: '免费开源',
    latest: '最新版本',
    allReleases: '所有版本',
    source: '源代码',
    github: 'GitHub',
    external: '在新标签页中打开',
    home: '首页',
    backHome: '返回首页',
    faq: '常见问题',
  },
  mock: {
    nearby: '附近设备',
    online: '3 台设备在线',
    you: '你',
    mac: '我的 Mac',
    iphone: 'Ada 的 iPhone',
    pc: '工作室 PC',
    server: '家用服务器',
    quickSave: '快速保存',
    sending: '正在发送',
    verified: '已校验',
    received: '已接收',
    sent: '已发送',
    folder: '夏日照片',
    message: '我快到了',
    light: '浅色',
    dark: '深色',
    history: '传输记录',
    pin: '接收 PIN',
    noCloud: '不经过云端',
    localNetwork: '本地网络',
  },
  home: {
    title: 'Hando — 局域网里的私密文件互传',
    description:
      '在 Mac、Windows、Linux、iOS 和 Android 之间通过局域网发送文件、文件夹和消息。免费开源，兼容 LocalSend。',
    hero: {
      badge: '兼容 LocalSend · 免费开源',
      title: ['传文件，', '就在身边'],
      lead: 'Hando 让文件、文件夹和消息在同一个 Wi-Fi 下直接从一台设备传到另一台。Mac 到 PC，iPhone 到 Android。没有云端，没有账号，也不上传。',
      how: '看看怎么用',
      note: 'MIT 许可证 · macOS、Windows、Linux、iOS 与 Android',
      macAlt: 'macOS 上的 Hando，显示附近设备和收藏',
      phoneAlt: 'iPhone 上的 Hando，显示附近设备',
      nearby: '附近有 3 台设备',
      noInternet: '无需互联网',
      sending: '正在发送至 Ada 的 iPhone',
      file: '夏日照片.zip',
      size: '1.2 GB',
      progress: '68%',
      verified: 'SHA-256 校验',
    },
    factsTitle: '为什么选择 Hando',
    facts: [
      {
        title: '只在局域网',
        body: '设备之间直接传输。没有上传，也没有服务器挡在中间。',
      },
      {
        title: '默认加密',
        body: 'HTTPS 双向 TLS，并固定对端的证书指纹。兼容旧设备时可改用普通 HTTP。',
      },
      {
        title: '会说 LocalSend',
        body: '官方 LocalSend 应用使用同一协议，不必两台设备都装 Hando。',
      },
    ],
    how: {
      eyebrow: '工作原理',
      title: '三步，一个 Wi-Fi',
      lead: '没有配对码，没有账号，也不用设置。在两台设备上打开 Hando，剩下的交给它。',
      steps: [
        {
          title: '互相发现',
          body: '两台设备在局域网中广播自己，附近的设备很快出现。收藏的设备即使离线，也会留在列表里。',
        },
        {
          title: '选择要传的内容',
          body: '选择文件或整个文件夹，也可以写一条短消息，然后点一下身边的设备。',
        },
        {
          title: '对方接收，完成',
          body: '对方确认请求后开始传输。两端都能看到进度，文件保存前还会校验哈希。',
        },
      ],
    },
    features: {
      eyebrow: '功能',
      title: '小巧，但什么都想到了',
      lead: '让一次传输变得轻松的细节，无论在手边还是桌面上。',
      send: {
        title: '想传的，一次带走',
        body: '文件、整个文件夹或一条消息。可以一次选多个，逐个查看进度，改变主意时可以取消。',
      },
      favorites: {
        title: '把常用设备留在身边',
        body: '给设备加星，离线时也能看到。开启快速保存后，收藏设备发来的内容无需确认——适合桌上那台手机，而不是楼下的咖啡店。',
      },
      pin: {
        title: '其余的人，需要 PIN',
        body: '设置接收 PIN 后，只有知道 PIN 的人才能发给你。',
      },
      checksum: {
        title: '校验，而不是祈祷',
        body: '发送方提供 SHA-256 时，Hando 会在文件写入最终名称前完成校验。',
      },
      history: {
        title: '记录都在，随时再打开',
        body: '完成的传输会留在应用里，随时打开所在文件夹。',
      },
      calm: {
        title: '一眼就是原生应用',
        body: '桌面使用侧边栏，手机使用标签栏；浅色、深色、英文和简体中文都已备好。',
      },
    },
    screens: {
      eyebrow: '界面',
      title: '每一块屏幕，都恰到好处',
      lead: '同一个应用，随手里的设备变换形态。',
      tabs: { macos: 'macOS', ios: 'iPhone', android: 'Android' },
      captions: {
        macos: '桌面上是侧边栏。收藏、附近设备和传输都在一个窗口里。',
        ios: '手机上是拇指就能切换的标签栏。附近、传输和设置，一步切换。',
        android: 'Android 上也是同样的布局，浅色或深色。',
      },
      alts: {
        macos: 'macOS 上的 Hando，包含侧边栏、附近设备和收藏',
        ios: 'iPhone 上的 Hando，包含附近设备和标签栏',
        android: 'Android 上的 Hando，包含附近设备和标签栏',
      },
    },
    compatibility: {
      eyebrow: '兼容性',
      title: '它本来就会说 LocalSend',
      lead: 'Hando 实现 LocalSend v2.2 协议，可以和官方应用互相收发。对方不必安装 Hando。',
      official: '官方应用或 CLI',
      protocol: '协议 v2.2',
      specs: [
        {
          label: '发现',
          value: 'IPv4 组播 224.0.0.167:53317；没有回应时扫描本地网段。',
        },
        {
          label: '传输',
          value: '默认使用 HTTPS 双向 TLS；兼容旧设备时可使用普通 HTTP。',
        },
        {
          label: '校验',
          value: '接收文件保存前验证 SHA-256 校验和。',
        },
        {
          label: '内容',
          value: '多个文件、文件夹结构、文字消息和接收 PIN。',
        },
      ],
      tested: '已与 LocalSend CLI 1.18.2 双向实测。',
      protocolLink: '阅读协议',
      notAffiliated: 'Hando 与 LocalSend 项目没有隶属关系。',
    },
    privacy: {
      eyebrow: '隐私',
      title: ['没有云', '没有账号', '没有绕路'],
      lead: '两台设备之间最短的路径，就是它们已经在的那个房间。Hando 只走这条路。',
      points: [
        '文件直接在两台设备之间传输。',
        '双向 TLS，并针对每次传输固定对端证书。',
        '收到的文件只保存在你选择的文件夹。',
      ],
      link: '了解 Hando 如何处理数据',
    },
    download: {
      eyebrow: '下载',
      title: '带上它',
      lead: 'MIT 许可证，免费开源。从最新版本中选择适合你平台的安装包。',
      all: '所有版本',
    },
    faq: {
      eyebrow: '常见问题',
      title: '你可能想知道',
      items: [
        {
          q: '两台设备都要装 Hando 吗？',
          a: '不用。Hando 使用 LocalSend 协议，另一台设备可以运行官方 LocalSend 应用。两边都是 Hando 时，还能使用收藏和快速保存。',
        },
        {
          q: '需要联网吗？',
          a: '不需要，传输只使用局域网。如果互相发现不了，请检查路由器是否启用了客户端隔离；Hando 也会扫描本地网段作为备选。',
        },
        {
          q: '文件会上传到云端吗？',
          a: '不会。Hando 没有服务器、账号或云端，文件只在两台设备之间直接传输。',
        },
        {
          q: '可以传什么？',
          a: '文件、文件夹和文字消息，支持逐文件进度和取消。桌面端的接收文件保存在 Downloads/Hando，手机端保存在应用文件夹，也可以在设置里更换。',
        },
        {
          q: '如何保持兼容？',
          a: '它实现 LocalSend v2.2：组播发现、固定指纹的双向 TLS 和 SHA-256 校验。已与 LocalSend CLI 1.18.2 双向测试。',
        },
        {
          q: '真的免费吗？',
          a: '是。Hando 使用 MIT 许可证，源码和发布流程都在 GitHub 上。如果它为你节省了时间，可以赞助开发。',
        },
        {
          q: 'iPhone 上怎么安装？',
          a: '除非配置了签名描述文件，iOS 安装包是未签名的 .ipa。安装前需要用你的 Apple ID 或开发者描述文件签名。',
        },
      ],
    },
  },
  download: {
    title: '下载 Hando — macOS、Windows、Linux、iOS 与 Android',
    description: '下载适用于 macOS、Windows、Linux、Android 或 iOS 的 Hando。免费开源，安装包由 GitHub Actions 从源码构建。',
    hero: {
      eyebrow: '下载',
      title: '获取 Hando',
      lead: '最新版本，为你的平台而构建。免费且使用 MIT 许可证；每个安装包都来自发布流程。',
      latest: '打开最新版本',
      note: '版本说明和全部文件都在 GitHub 上。',
    },
    platforms: {
      macos: { note: '适用于 Apple 芯片和 Intel Mac。', format: 'DMG' },
      windows: { note: '适用于 x64 和 arm64 PC 的安装程序。', format: '安装包 .exe' },
      linux: { note: '适用于 x64 和 arm64：.deb 包或 Arch 包。', format: 'DEB · pkg.tar.gz' },
      android: { note: '适用于 arm64 手机和平板的 APK。', format: 'APK' },
      ios: {
        note: '未签名的 IPA；需要用你的 Apple ID 或开发者描述文件签名。',
        format: 'IPA',
      },
    },
    open: '打开版本页',
    you: '你的平台',
    after: {
      title: '下载之后',
      lead: '每个安装包都按常规方式安装。有几点值得留意。',
      steps: {
        macos:
          '打开 .dmg，把 Hando 拖进“应用程序”。如果 macOS 提示无法验证，可在“系统设置 → 隐私与安全性”中允许打开。',
        windows: '运行安装程序。安装包不通过应用商店分发，Windows 可能要求确认。',
        linux: '使用 sudo apt install ./Hando-*.deb 安装，或使用 sudo pacman -U Hando-*.pkg.tar.gz。',
        android: '打开 APK。Android 会要求允许浏览器或文件管理器安装应用。',
        ios: 'IPA 未签名。请使用 AltStore、Sideloadly 或自己的开发者描述文件签名，然后在“设置”中信任它。',
      },
    },
    needs: {
      title: '需要什么',
      items: [
        '两台设备连接同一个 Wi-Fi 或局域网。',
        'macOS、Windows、Linux、Android 或 iOS——Hando 在每个平台上原生运行。',
        '不需要账号、订阅或互联网连接。',
      ],
    },
    verify: {
      title: '公开构建',
      body: '每个版本都由 GitHub Actions 从仓库构建，并附上对应版本的更新日志。源码使用 MIT 许可证。',
      workflow: '发布流程',
      source: '源代码',
    },
    build: {
      title: '自己构建',
      body: 'Hando 需要 Go 1.26 或更高版本。以下命令构建所有目标并运行测试。',
      note: 'Apple 平台需要 Xcode，Android 需要 Android SDK 的 platform-tools。',
    },
  },
  privacy: {
    title: '隐私 — Hando',
    description: 'Hando 和本网站如何处理你的数据：本地传输、没有账号、没有分析。',
    hero: {
      eyebrow: '隐私',
      title: '隐私，由架构决定',
      lead: 'Hando 没有什么可收集的：没有账号、没有服务器，也不上传。这一页说明这对应用和本网站分别意味着什么。',
    },
    summary: ['没有账号', '没有分析', '没有上传'],
    app: {
      title: '应用',
      paragraphs: [
        'Hando 不创建账号，也不连接任何 Hando 服务。传输只在你局域网内的两台设备之间直接进行。',
        '传输默认使用 HTTPS 双向 TLS。客户端会固定对端证书的指纹，因此传输不会被重定向到同一地址上的另一台设备。兼容旧设备时，可以在设置中改用普通 HTTP。',
        '设备身份、收藏、设置和传输记录都保存在你的设备上。收到的文件只留在你选择的接收文件夹。',
        '与官方 LocalSend 应用互传时，对方应用会按其自身的隐私政策处理数据。',
      ],
    },
    website: {
      title: '本网站',
      paragraphs: [
        '本网站是由 GitHub Pages 提供的静态页面，不设置 Cookie，也不运行分析或广告。',
        '你选择的外观和语言保存在浏览器的本地存储中，不会发送给我们。',
        '作为托管方，GitHub 可能按照其隐私声明保留标准访问日志。',
        '指向 GitHub、X 等站点的链接会离开本网站，这些站点有自己的隐私政策。',
      ],
    },
    contact: {
      title: '问题',
      body: '请在 GitHub 上提交 issue。公开 issue 中请勿包含个人数据。',
    },
  },
  notFound: {
    title: '页面未找到 — Hando',
    description: '这个页面不在局域网里。返回 Hando 首页。',
    heading: '这个页面掉线了',
    body: '可能是链接写错了，或者页面已经搬走。回到首页，重新找找。',
  },
  footer: {
    tagline: '局域网里的私密文件互传。',
    product: '产品',
    project: '项目',
    elsewhere: '其他',
    download: '下载',
    features: '功能',
    compatibility: '兼容性',
    privacy: '隐私',
    changelog: '更新日志',
    license: '许可证',
    protocol: 'LocalSend 协议',
    sponsor: '赞助',
    blog: '博客',
    builtWith: '使用 NSGo 构建',
    rights: '© {year} Hando',
    notAffiliated: '与 LocalSend 项目没有隶属关系。',
  },
};

export const copies: Record<Lang, Copy> = { en, zh };

export const languageNames: Record<Lang, string> = { en: 'English', zh: '简体中文' };
