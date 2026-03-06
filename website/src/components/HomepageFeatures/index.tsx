import type {ReactNode} from 'react';
import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  emoji: string;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'One Command to Start',
    emoji: '⚡',
    description: (
      <>
        Run <code>lingti-bot relay --provider deepseek --api-key sk-xxx</code> and get your
        personal bot page at <code>https://bot.lingti.com/bots/xxx</code> instantly.
        No platform account, no server, no domain needed.
      </>
    ),
  },
  {
    title: '19 Chat Platforms',
    emoji: '🌐',
    description: (
      <>
        Connect to WeCom, Feishu, DingTalk, WeChat, Slack, Telegram, Discord, WhatsApp,
        LINE, Teams, and more via cloud relay or self-hosted gateway.
      </>
    ),
  },
  {
    title: '16 AI Backends',
    emoji: '🧠',
    description: (
      <>
        Switch between Claude, DeepSeek, Kimi, MiniMax, Gemini, OpenAI, and 10 more
        providers per conversation. Supports Claude Extended Thinking out of the box.
      </>
    ),
  },
  {
    title: '75+ MCP Tools',
    emoji: '🛠️',
    description: (
      <>
        Files, Shell, Git, GitHub, calendar, network, browser automation, social platform
        automation, scheduled tasks — all available to your AI via MCP protocol.
      </>
    ),
  },
  {
    title: 'Zero Dependency Binary',
    emoji: '🚀',
    description: (
      <>
        Single 30MB binary. No Node.js, no Python, no Docker required to run.
        Works on macOS, Linux, Windows, ARM, MIPS — scp and run anywhere.
      </>
    ),
  },
  {
    title: 'Built-in Web Chat UI',
    emoji: '💬',
    description: (
      <>
        No client app needed. Open the browser, get multiple parallel sessions each with
        isolated AI memory. All sessions persist across page refreshes.
      </>
    ),
  },
];

function Feature({title, emoji, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center padding-horiz--md padding-vert--md">
        <div className={styles.featureEmoji}>{emoji}</div>
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
