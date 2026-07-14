import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://leowzz.github.io',
  base: '/leo-cli/',
  integrations: [
    starlight({
      title: {
        'zh-CN': 'leo-cli 文档',
        en: 'leo-cli Documentation',
      },
      locales: {
        root: { label: '简体中文', lang: 'zh-CN' },
        en: { label: 'English', lang: 'en' },
      },
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/leowzz/leo-cli' },
      ],
      sidebar: [
        { slug: 'getting-started' },
        {
          label: '使用指南',
          translations: { en: 'Guides' },
          items: [
            { slug: 'guides/join' },
            { slug: 'guides/repositories' },
            { slug: 'guides/time' },
            { slug: 'guides/docker-copy' },
            { slug: 'guides/log-viewer' },
          ],
        },
        {
          label: '参考',
          translations: { en: 'Reference' },
          items: [
            { slug: 'reference/configuration' },
            { slug: 'reference/runtime' },
            { autogenerate: { directory: 'reference/commands', collapsed: true } },
          ],
        },
        { slug: 'concepts' },
        { slug: 'development' },
      ],
      customCss: ['./src/styles/custom.css'],
    }),
  ],
});
