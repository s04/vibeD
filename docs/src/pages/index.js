import React from 'react';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header style={{
      padding: '4rem 0',
      textAlign: 'center',
      position: 'relative',
      overflow: 'hidden',
    }}>
      <div className="container">
        <h1 style={{fontSize: '3rem'}}>{siteConfig.title}</h1>
        <p style={{fontSize: '1.5rem', opacity: 0.8}}>{siteConfig.tagline}</p>
        <div style={{display: 'flex', gap: '1rem', justifyContent: 'center', marginTop: '2rem'}}>
          <Link
            className="button button--primary button--lg"
            to="/docs">
            Get Started
          </Link>
          <Link
            className="button button--secondary button--lg"
            to="/docs/getting-started/installation">
            Installation
          </Link>
        </div>
      </div>
    </header>
  );
}

function Feature({title, description}) {
  return (
    <div style={{flex: 1, padding: '1rem'}}>
      <h3>{title}</h3>
      <p>{description}</p>
    </div>
  );
}

export default function Home() {
  return (
    <Layout
      title="Home"
      description="Workload Orchestrator for GenAI-generated Artifacts">
      <HomepageHeader />
      <main>
        <section style={{padding: '2rem 0'}}>
          <div className="container">
            <div style={{display: 'flex', gap: '2rem', flexWrap: 'wrap'}}>
              <Feature
                title="MCP Native"
                description="Expose deployment tools via the Model Context Protocol. Any AI coding tool (Claude, Gemini, ChatGPT) can deploy directly to your infrastructure."
              />
              <Feature
                title="Multi-Target"
                description="Deploy to Knative for serverless, plain Kubernetes for traditional workloads, or wasmCloud for WebAssembly artifacts."
              />
              <Feature
                title="Zero Config Builds"
                description="Cloud Native Buildpacks automatically detect your language and build optimized container images. No Dockerfiles needed."
              />
            </div>
          </div>
        </section>
      </main>
    </Layout>
  );
}
