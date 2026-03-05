import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docsSidebar: [
    'intro',
    {
      type: 'category',
      label: '快速上手',
      collapsed: false,
      items: ['relay', 'configuration', 'ai-providers', 'cli-reference'],
    },
    {
      type: 'category',
      label: '平台接入',
      collapsed: false,
      items: [
        'chat-platforms',
        'feishu-integration',
        'wecom-integration',
        'wechat-integration',
        'slack-integration',
        'dingtalk-integration',
      ],
    },
    {
      type: 'category',
      label: '部署方式',
      items: ['gateway', 'cloud-relay', 'gateway-vs-relay'],
    },
    {
      type: 'category',
      label: '功能',
      items: [
        'browser-automation',
        'social-platform-automation',
        'skills',
        'cron-jobs',
        'file-sending',
        'auto-approval',
      ],
    },
    'roadmap',
  ],
};

export default sidebars;
