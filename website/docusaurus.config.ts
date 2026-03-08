import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'lingti-bot',
  tagline: '极简至上 · 效率为王 · 秒级接入 · 一链即用',
  favicon: 'img/favicon.ico',

  future: {
    v4: true,
  },

  url: 'https://ruilisi.github.io',
  baseUrl: '/lingti-bot/',
  organizationName: 'ruilisi',
  projectName: 'lingti-bot',
  deploymentBranch: 'gh-pages',
  trailingSlash: false,

  onBrokenLinks: 'warn',
  onBrokenMarkdownLinks: 'warn',

  markdown: {
    format: 'detect',
  },

  i18n: {
    defaultLocale: 'zh-Hans',
    locales: ['zh-Hans'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/ruilisi/lingti-bot/edit/master/',
          routeBasePath: '/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    colorMode: {
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'lingti-bot',
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: '文档',
        },
        {
          href: 'https://github.com/ruilisi/lingti-bot',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: '快速上手',
          items: [
            {label: '云中继', to: '/relay'},
            {label: '配置参考', to: '/configuration'},
            {label: 'AI 提供商', to: '/ai-providers'},
          ],
        },
        {
          title: '平台接入',
          items: [
            {label: '飞书', to: '/feishu-integration'},
            {label: '企业微信', to: '/wecom-integration'},
            {label: '微信公众号', to: '/wechat-integration'},
            {label: 'Slack', to: '/slack-integration'},
          ],
        },
        {
          title: '更多',
          items: [
            {label: 'GitHub', href: 'https://github.com/ruilisi/lingti-bot'},
            {label: '官网', href: 'https://bot.lingti.com'},
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} ruilisi. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'yaml', 'go'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
