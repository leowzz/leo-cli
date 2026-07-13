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
      customCss: ['./src/styles/custom.css'],
    }),
  ],
});
