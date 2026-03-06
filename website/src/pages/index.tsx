import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import HomepageFeatures from '@site/src/components/HomepageFeatures';
import Heading from '@theme/Heading';

import styles from './index.module.css';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <Heading as="h1" className="hero__title">
          {siteConfig.title}
        </Heading>
        <p className="hero__subtitle">{siteConfig.tagline}</p>

        <div className={styles.quickStart}>
          <p className={styles.quickStartLabel}>⚡ Two steps — no platform account, no public server needed</p>

          <div className={styles.steps}>
            <div className={styles.step}>
              <span className={styles.stepNum}>1</span>
              <div>
                <p className={styles.stepTitle}>Install</p>
                <pre className={styles.code}><code>curl -fsSL https://files.lingti.com/install-bot.sh | bash</code></pre>
              </div>
            </div>

            <div className={styles.step}>
              <span className={styles.stepNum}>2</span>
              <div>
                <p className={styles.stepTitle}>Run relay with your AI provider and key</p>
                <pre className={styles.code}><code>lingti-bot relay --provider deepseek --api-key sk-xxx</code></pre>
                <p className={styles.output}>→ <strong>Your bot page: https://bot.lingti.com/bots/xxx</strong></p>
              </div>
            </div>
          </div>
        </div>

        <div className={styles.buttons}>
          <Link className="button button--secondary button--lg" to="/docs/intro">
            Read the docs
          </Link>
          <Link className="button button--outline button--secondary button--lg" href="https://github.com/ruilisi/lingti-bot">
            GitHub
          </Link>
        </div>
      </div>
    </header>
  );
}

export default function Home(): ReactNode {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title={siteConfig.title}
      description="One-command AI bot. Run lingti-bot relay --provider deepseek --api-key sk-xxx and get your bot page instantly.">
      <HomepageHeader />
      <main>
        <HomepageFeatures />
      </main>
    </Layout>
  );
}
