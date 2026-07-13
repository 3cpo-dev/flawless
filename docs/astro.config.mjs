import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://3cpo-dev.github.io',
  base: '/flawless',
  integrations: [
    starlight({
      title: 'flawless',
      description:
        'A pre-push quality gate in one binary: agent review, tests, lint, push, PR. No daemon, no setup, no ceremony.',
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/3cpo-dev/flawless' },
      ],
      sidebar: [
        {
          label: 'Start Here',
          items: [
            { label: 'Introduction', slug: 'start-here/introduction' },
            { label: 'Quick Start', slug: 'start-here/quick-start' },
            { label: 'Installation', slug: 'start-here/installation' },
          ],
        },
        {
          label: 'Concepts',
          items: [
            { label: 'The Gate Model', slug: 'concepts/gate-model' },
            { label: 'Pipeline', slug: 'concepts/pipeline' },
            { label: 'Auto-Fix Loop', slug: 'concepts/auto-fix' },
            { label: 'Worktrees & State', slug: 'concepts/worktrees-and-state' },
          ],
        },
        {
          label: 'Guides',
          items: [
            { label: 'Configuration', slug: 'guides/configuration' },
            { label: 'Choosing an Agent', slug: 'guides/agents' },
            { label: 'Using flawless from an Agent', slug: 'guides/agent-mode' },
            { label: 'Troubleshooting', slug: 'guides/troubleshooting' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { label: 'CLI Commands', slug: 'reference/cli' },
            { label: 'Pipeline Steps', slug: 'reference/pipeline-steps' },
            { label: 'Configuration', slug: 'reference/configuration' },
            { label: 'Environment Variables', slug: 'reference/environment' },
          ],
        },
      ],
    }),
  ],
});
