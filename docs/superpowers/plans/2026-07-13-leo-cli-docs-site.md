# leo-cli Documentation Site Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish a bilingual Starlight documentation site at GitHub Pages with Chinese at `/leo-cli/`, English at `/leo-cli/en/`, generated Cobra reference pages, and two reproducible VHS demos.

**Architecture:** Keep the Astro/Starlight project isolated under `site/`, with Chinese content at the Starlight root locale and mirrored English content under `en/`. Generate command reference Markdown from the existing Cobra tree, validate locale parity with a Node standard-library script, commit VHS tapes and rendered media, and deploy the static `site/dist` directory through a dedicated Pages workflow.

**Tech Stack:** Go 1.25.6, Cobra 1.10.2, Node 24, pnpm 10.30.3, Astro 7.0.7, Starlight 0.41.3, Pagefind, VHS, GitHub Actions, GitHub Pages.

## Global Constraints

- Simplified Chinese is the root locale; English is served under `/en/`.
- GitHub Pages base path is exactly `/leo-cli/`.
- Every authored or generated Markdown/MDX page must exist in both locales.
- CLI commands, flags, usage strings, and errors remain exactly as the English CLI prints them.
- Node dependencies stay inside `site/`; the existing Go release workflow remains independent.
- `docs/superpowers` is not copied into the public site or search index.
- Initial VHS scope is exactly `repo-picker` and `join`.
- VHS media does not autoplay; WebM is primary and PNG is the poster.
- Do not add versioned docs, analytics, a CMS, external search, or automatic translation.
- Preserve the current Makefile default target and existing release workflow.

---

## File Structure

### New site files

- `site/.node-version`: pins Node 24.
- `site/package.json`: pins pnpm, Astro, Starlight, and site scripts.
- `site/pnpm-lock.yaml`: deterministic site dependency lockfile.
- `site/astro.config.mjs`: locales, Pages base path, navigation, theme, and repository link.
- `site/tsconfig.json`: Astro strict TypeScript configuration.
- `site/src/content.config.ts`: Starlight content collection.
- `site/src/styles/custom.css`: small Starlight token overrides only.
- `site/scripts/check-locales.mjs`: compares Chinese and English content paths.
- `site/scripts/check-locales.test.mjs`: Node test for missing-page detection.
- `site/scripts/check-demos.mjs`: validates committed VHS outputs.
- `site/scripts/check-demos.test.mjs`: Node test for missing media detection.
- `site/src/components/VhsDemo.astro`: accessible, base-path-aware video component.
- `site/src/content/docs/**`: Chinese authored and generated documentation.
- `site/src/content/docs/en/**`: mirrored English documentation.
- `site/vhs/setup-repo-demo.sh`: deterministic repository-picker fixture setup.
- `site/vhs/fixtures/users.csv`: deterministic join input.
- `site/vhs/repo-picker.tape`: repository picker recording.
- `site/vhs/join.tape`: SQL join recording.
- `site/public/demos/*.{webm,png}`: committed VHS outputs.
- `tools/docsgen/main.go`: generates localized Cobra Markdown trees.
- `tools/docsgen/main_test.go`: generator regression test.
- `.github/workflows/docs.yml`: build and deploy GitHub Pages.

### Existing files to modify

- `.gitignore`: ignores only site build/cache/dependency output.
- `cmd/root.go`: exposes the initialized Cobra root to repository tooling.
- `go.mod`, `go.sum`: records Cobra doc transitive dependencies if required.
- `Makefile`: adds `docs-dev`, `docs-build`, and `docs-demos` targets.
- `README.md`, `README.zh.md`: become compact entry points to the site.

---

### Task 1: Starlight Foundation And Locale Parity

**Files:**
- Modify: `.gitignore`
- Create: `site/.node-version`
- Create: `site/package.json`
- Create: `site/astro.config.mjs`
- Create: `site/tsconfig.json`
- Create: `site/src/content.config.ts`
- Create: `site/src/styles/custom.css`
- Create: `site/scripts/check-locales.mjs`
- Create: `site/scripts/check-locales.test.mjs`
- Create: `site/src/content/docs/index.mdx`
- Create: `site/src/content/docs/en/index.mdx`
- Create: `site/pnpm-lock.yaml` via `pnpm install`

**Interfaces:**
- Consumes: no earlier task output.
- Produces: `pnpm --dir site test`, `pnpm --dir site check:locales`, `pnpm --dir site build`, Chinese root routing, and English `/en/` routing.

- [ ] **Step 1: Add the site package metadata and a failing locale test**

Create `site/.node-version`:

```text
24
```

Create `site/package.json`:

```json
{
  "name": "leo-cli-docs",
  "private": true,
  "type": "module",
  "packageManager": "pnpm@10.30.3",
  "engines": {
    "node": ">=24 <25"
  },
  "scripts": {
    "dev": "astro dev",
    "build": "pnpm check:locales && astro build",
    "preview": "astro preview",
    "test": "node --test scripts/*.test.mjs",
    "check:locales": "node scripts/check-locales.mjs"
  },
  "dependencies": {
    "@astrojs/starlight": "0.41.3",
    "astro": "7.0.7"
  }
}
```

Create `site/scripts/check-locales.test.mjs`:

```js
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import { findLocaleMismatches } from './check-locales.mjs';

test('reports pages missing from either locale', async () => {
  const root = await mkdtemp(join(tmpdir(), 'leo-docs-locales-'));
  try {
    await mkdir(join(root, 'guides'), { recursive: true });
    await mkdir(join(root, 'en'), { recursive: true });
    await writeFile(join(root, 'index.md'), '# Chinese');
    await writeFile(join(root, 'guides', 'join.mdx'), '# Join');
    await writeFile(join(root, 'en', 'index.md'), '# English');
    await writeFile(join(root, 'en', 'only-english.md'), '# English only');

    assert.deepEqual(await findLocaleMismatches(root), {
      missingEnglish: ['guides/join.mdx'],
      missingChinese: ['only-english.md'],
    });
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});
```

- [ ] **Step 2: Run the test and verify the missing module failure**

Run:

```bash
rtk node --test site/scripts/check-locales.test.mjs
```

Expected: FAIL with `ERR_MODULE_NOT_FOUND` for `check-locales.mjs`.

- [ ] **Step 3: Implement the locale parity checker**

Create `site/scripts/check-locales.mjs`:

```js
import { readdir } from 'node:fs/promises';
import { dirname, join, relative, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const contentRoot = join(dirname(fileURLToPath(import.meta.url)), '..', 'src', 'content', 'docs');

async function collectPages(root, skipEnglishRoot = false, current = root) {
  const pages = [];
  for (const entry of await readdir(current, { withFileTypes: true })) {
    if (skipEnglishRoot && current === root && entry.name === 'en') continue;
    const path = join(current, entry.name);
    if (entry.isDirectory()) {
      pages.push(...await collectPages(root, false, path));
    } else if (/\.mdx?$/.test(entry.name)) {
      pages.push(relative(root, path).split(sep).join('/'));
    }
  }
  return pages.sort();
}

export async function findLocaleMismatches(root = contentRoot) {
  const chinese = new Set(await collectPages(root, true));
  const english = new Set(await collectPages(join(root, 'en')));
  return {
    missingEnglish: [...chinese].filter((page) => !english.has(page)),
    missingChinese: [...english].filter((page) => !chinese.has(page)),
  };
}

export async function checkLocales(root = contentRoot) {
  const missing = await findLocaleMismatches(root);
  if (missing.missingEnglish.length || missing.missingChinese.length) {
    for (const page of missing.missingEnglish) console.error(`missing English page: en/${page}`);
    for (const page of missing.missingChinese) console.error(`missing Chinese page: ${page}`);
    return false;
  }
  return true;
}

if (process.argv[1] === fileURLToPath(import.meta.url) && !await checkLocales()) {
  process.exitCode = 1;
}
```

- [ ] **Step 4: Run the locale test and verify it passes**

Run:

```bash
rtk node --test site/scripts/check-locales.test.mjs
```

Expected: one test passes.

- [ ] **Step 5: Add the minimal Starlight application**

Create `site/astro.config.mjs`:

```js
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
```

Create `site/tsconfig.json`:

```json
{
  "extends": "astro/tsconfigs/strict"
}
```

Create `site/src/content.config.ts`:

```ts
import { defineCollection } from 'astro:content';
import { docsLoader } from '@astrojs/starlight/loaders';
import { docsSchema } from '@astrojs/starlight/schema';

export const collections = {
  docs: defineCollection({ loader: docsLoader(), schema: docsSchema() }),
};
```

Create `site/src/styles/custom.css`:

```css
:root {
  --sl-color-accent-low: #d9f5ef;
  --sl-color-accent: #087f70;
  --sl-color-accent-high: #064e45;
  --sl-content-width: 52rem;
}

:root[data-theme='dark'] {
  --sl-color-accent-low: #123f3a;
  --sl-color-accent: #5eead4;
  --sl-color-accent-high: #ccfbf1;
}
```

Create `site/src/content/docs/index.mdx`:

```mdx
---
title: leo-cli
description: 面向个人开发工作流的命令行工具。
---

`leo-cli` 提供仓库切换、SQL IN 构造、时间转换、镜像搬运和本地日志查看能力。

```bash
leo repo reindex
leo repo
```
```

Create `site/src/content/docs/en/index.mdx`:

```mdx
---
title: leo-cli
description: Command-line tools for personal development workflows.
---

`leo-cli` provides repository navigation, SQL IN construction, time conversion,
image copying, and local log inspection.

```bash
leo repo reindex
leo repo
```
```

Append to `.gitignore`:

```gitignore
site/.astro/
site/dist/
site/node_modules/
```

- [ ] **Step 6: Install and build the site**

Run:

```bash
rtk pnpm --dir site install
rtk pnpm --dir site test
rtk pnpm --dir site build
```

Expected: lockfile created, one Node test passes, and Astro builds Chinese and English index routes under `site/dist`.

- [ ] **Step 7: Commit the foundation**

```bash
rtk git add .gitignore site
rtk git commit -m "docs: scaffold bilingual Starlight site"
```

---

### Task 2: Generated Cobra Command Reference

**Files:**
- Modify: `cmd/root.go`
- Modify: `go.mod`
- Modify: `go.sum`
- Create: `tools/docsgen/main.go`
- Create: `tools/docsgen/main_test.go`
- Create: `site/src/content/docs/reference/commands/*.md`
- Create: `site/src/content/docs/en/reference/commands/*.md`

**Interfaces:**
- Consumes: `site/src/content/docs` from Task 1 and the existing initialized Cobra tree.
- Produces: `cmd.RootCommand() *cobra.Command`, `generate(root *cobra.Command, contentRoot string) error`, and `go run ./tools/docsgen`.

- [ ] **Step 1: Write the failing generator test**

Create `tools/docsgen/main_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestGenerateWritesBothLocalesWithoutDates(t *testing.T) {
	root := &cobra.Command{Use: "leo", Short: "test root"}
	root.AddCommand(&cobra.Command{Use: "child", Short: "test child"})
	docsRoot := t.TempDir()

	if err := generate(root, docsRoot); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		filepath.Join(docsRoot, "reference", "commands", "leo.md"),
		filepath.Join(docsRoot, "reference", "commands", "leo_child.md"),
		filepath.Join(docsRoot, "en", "reference", "commands", "leo.md"),
		filepath.Join(docsRoot, "en", "reference", "commands", "leo_child.md"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if !strings.HasPrefix(text, "---\ntitle:") {
			t.Fatalf("%s has no frontmatter", path)
		}
		if strings.Contains(text, "Auto generated by") {
			t.Fatalf("%s contains a generated date", path)
		}
	}
}
```

- [ ] **Step 2: Run the Go test and verify it fails**

Run:

```bash
rtk go test ./tools/docsgen
```

Expected: FAIL because `generate` does not exist.

- [ ] **Step 3: Expose the existing root command to repository tooling**

Add to `cmd/root.go` immediately before `Execute`:

```go
func RootCommand() *cobra.Command {
	return rootCmd
}
```

Do not construct a second command tree.

- [ ] **Step 4: Implement the deterministic generator**

Create `tools/docsgen/main.go`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leo/leo-cli/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

type localeOutput struct {
	dir         string
	description string
	note        string
}

func main() {
	if err := generate(cmd.RootCommand(), filepath.Join("site", "src", "content", "docs")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(root *cobra.Command, contentRoot string) error {
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	root.DisableAutoGenTag = true

	locales := []localeOutput{
		{
			description: "由 leo CLI 命令树生成的命令参考。",
			note:        "> 本页由 Cobra 命令树生成；命令、参数和输出保持 CLI 原文。",
		},
		{
			dir:         "en",
			description: "Command reference generated from the leo CLI command tree.",
			note:        "> This page is generated from the Cobra command tree.",
		},
	}

	for _, locale := range locales {
		outputDir := filepath.Join(contentRoot, locale.dir, "reference", "commands")
		if err := os.RemoveAll(outputDir); err != nil {
			return err
		}
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return err
		}

		prepend := func(filename string) string {
			base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
			title := strings.ReplaceAll(base, "_", " ")
			return fmt.Sprintf("---\ntitle: %q\ndescription: %q\n---\n\n%s\n\n", title, locale.description, locale.note)
		}
		link := func(name string) string { return "./" + name }
		if err := doc.GenMarkdownTreeCustom(root, outputDir, prepend, link); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 5: Resolve dependencies and run the generator test**

Run:

```bash
rtk go mod tidy
rtk go test ./tools/docsgen
rtk go run ./tools/docsgen
```

Expected: generator test passes and both command trees are created. Hidden `repo refresh-metadata` must not produce a page.

- [ ] **Step 6: Verify generated output and all Go tests**

Run:

```bash
rtk find site/src/content/docs/reference/commands
rtk grep -n "Auto generated by|refresh-metadata" site/src/content/docs/reference/commands site/src/content/docs/en/reference/commands
rtk go test ./...
rtk pnpm --dir site build
```

Expected: command pages exist, the grep reports zero matches, Go tests pass, and the site builds.

- [ ] **Step 7: Commit the generator and generated pages**

```bash
rtk git add cmd/root.go go.mod go.sum tools/docsgen site/src/content/docs/reference/commands site/src/content/docs/en/reference/commands
rtk git commit -m "docs: generate Cobra command reference"
```

---

### Task 3: Bilingual User Documentation And Navigation

**Files:**
- Modify: `site/astro.config.mjs`
- Modify: `site/src/content/docs/index.mdx`
- Modify: `site/src/content/docs/en/index.mdx`
- Create in both locale trees: `getting-started.md`
- Create in both locale trees: `guides/repositories.md`
- Create in both locale trees: `guides/join.mdx`
- Create in both locale trees: `guides/time.md`
- Create in both locale trees: `guides/docker-copy.md`
- Create in both locale trees: `guides/log-viewer.md`
- Create in both locale trees: `reference/configuration.md`
- Create in both locale trees: `reference/runtime.md`
- Create in both locale trees: `concepts.md`
- Create in both locale trees: `development.md`

**Interfaces:**
- Consumes: locale checker from Task 1 and generated reference directory from Task 2.
- Produces: the complete public information architecture and stable slugs used by navigation and later VHS embeds.

- [ ] **Step 1: Add Chinese pages first to exercise the parity failure**

Create the Chinese files with this exact title and section map:

| File | Title | Required sections and source of truth |
| --- | --- | --- |
| `getting-started.md` | `开始使用` | Install, local build, first `repo reindex`, `repo`, shell init; current `README.zh.md` plus command help |
| `guides/repositories.md` | `快速切换仓库` | Index refresh, picker keys, printed path, shell helper; `cmd/repo.go`, `cmd/shell.go` |
| `guides/join.mdx` | `构造 SQL IN` | Clipboard/stdin/file priority, CSV keys, formats, cancel/copy behavior; `cmd/sql_in.go` |
| `guides/time.md` | `转换时间与时区` | Seconds, milliseconds, date strings, default UTC+8, `--to`, configured zones; `cmd/time.go` |
| `guides/docker-copy.md` | `搬运 Docker 镜像` | Registry aliases, full references, `--dry`, `--platform`, skopeo boundary; `cmd/docker_copy.go` |
| `guides/log-viewer.md` | `搜索和跟随日志` | Zero-config discovery, `--logs`, strict `--project`, network boundary, search/follow behavior; `cmd/log.go` and current README |
| `reference/configuration.md` | `配置字段` | Complete YAML example and field meanings for `repo`, `docker`, `time`, `proj`; `internal/config/config.go` |
| `reference/runtime.md` | `文件、数据与环境变量` | Config/data paths, `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, path expansion, SQLite WAL |
| `concepts.md` | `工作原理` | Repository index, log security boundary, local-only storage/privacy; current design specs, no implementation-plan history |
| `development.md` | `开发与发布` | `make dev/test/build/release/release-github`, layout, GitHub CLI requirement |

Every file must use this frontmatter shape with a concrete localized description:

```md
---
title: 页面标题
description: 一句说明该页面解决的问题。
---
```

Copy terminal examples from the actual current README or fresh `go run . COMMAND --help` output. Do not translate CLI output.

- [ ] **Step 2: Verify the parity check fails with the exact missing English paths**

Run:

```bash
rtk pnpm --dir site check:locales
```

Expected: FAIL listing each new path under `en/`.

- [ ] **Step 3: Add the matching English pages**

Create the same relative paths under `site/src/content/docs/en/` with these titles:

| Chinese path | English title |
| --- | --- |
| `getting-started.md` | `Getting Started` |
| `guides/repositories.md` | `Navigate Repositories` |
| `guides/join.mdx` | `Build SQL IN Values` |
| `guides/time.md` | `Convert Time And Timezones` |
| `guides/docker-copy.md` | `Copy Docker Images` |
| `guides/log-viewer.md` | `Search And Follow Logs` |
| `reference/configuration.md` | `Configuration Fields` |
| `reference/runtime.md` | `Files, Data, And Environment` |
| `concepts.md` | `How It Works` |
| `development.md` | `Development And Releases` |

Translate the prose directly from the Chinese page while preserving all code,
command names, flags, paths, and error text.

- [ ] **Step 4: Configure explicit bilingual navigation**

Add this `sidebar` value to the Starlight options in `site/astro.config.mjs`:

```js
sidebar: [
  { slug: 'getting-started' },
  {
    label: '使用指南',
    translations: { en: 'Guides' },
    items: [
      { slug: 'guides/repositories' },
      { slug: 'guides/join' },
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
```

Expand both index pages into compact documentation entry points with:

- One-sentence scope.
- A copyable install/build command.
- A three-command quick start.
- Links to getting started, guides, and command reference.
- No marketing hero, feature-card grid, or duplicated full manual.

- [ ] **Step 5: Verify locale parity, command examples, and production build**

Run:

```bash
rtk pnpm --dir site check:locales
rtk go run . --help
rtk go run . log --help
rtk pnpm --dir site build
```

Expected: parity passes; documented root and log flags match command help; Astro and Pagefind build successfully.

- [ ] **Step 6: Commit the authored content**

```bash
rtk git add site/astro.config.mjs site/src/content/docs
rtk git commit -m "docs: add bilingual user guides"
```

---

### Task 4: Accessible VHS Demos

**Files:**
- Modify: `site/package.json`
- Modify: `site/src/content/docs/index.mdx`
- Modify: `site/src/content/docs/en/index.mdx`
- Modify: `site/src/content/docs/guides/join.mdx`
- Modify: `site/src/content/docs/en/guides/join.mdx`
- Create: `site/scripts/check-demos.mjs`
- Create: `site/scripts/check-demos.test.mjs`
- Create: `site/src/components/VhsDemo.astro`
- Create: `site/vhs/setup-repo-demo.sh`
- Create: `site/vhs/fixtures/users.csv`
- Create: `site/vhs/repo-picker.tape`
- Create: `site/vhs/join.tape`
- Create: `site/public/demos/repo-picker.webm`
- Create: `site/public/demos/repo-picker.png`
- Create: `site/public/demos/join.webm`
- Create: `site/public/demos/join.png`

**Interfaces:**
- Consumes: MDX pages from Task 3 and the built `bin/leo` executable.
- Produces: `VhsDemo.astro` props, deterministic tapes, committed media, and a build-time media check.

- [ ] **Step 1: Write the failing demo asset test**

Create `site/scripts/check-demos.test.mjs`:

```js
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import { findMissingDemos } from './check-demos.mjs';

test('reports missing demo outputs', async () => {
  const root = await mkdtemp(join(tmpdir(), 'leo-docs-demos-'));
  try {
    await mkdir(join(root, 'demos'), { recursive: true });
    await writeFile(join(root, 'demos', 'repo-picker.webm'), 'video');
    assert.deepEqual(await findMissingDemos(root), [
      'demos/join.png',
      'demos/join.webm',
      'demos/repo-picker.png',
    ]);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});
```

- [ ] **Step 2: Run the Node test and verify it fails**

Run:

```bash
rtk node --test site/scripts/check-demos.test.mjs
```

Expected: FAIL with `ERR_MODULE_NOT_FOUND` for `check-demos.mjs`.

- [ ] **Step 3: Implement the media checker and wire it into builds**

Create `site/scripts/check-demos.mjs`:

```js
import { access } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const publicRoot = join(dirname(fileURLToPath(import.meta.url)), '..', 'public');
const required = [
  'demos/join.png',
  'demos/join.webm',
  'demos/repo-picker.png',
  'demos/repo-picker.webm',
];

export async function findMissingDemos(root = publicRoot) {
  const missing = [];
  for (const path of required) {
    try {
      await access(join(root, path));
    } catch {
      missing.push(path);
    }
  }
  return missing;
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const missing = await findMissingDemos();
  for (const path of missing) console.error(`missing demo asset: ${path}`);
  if (missing.length) process.exitCode = 1;
}
```

Change the `build` script in `site/package.json` to:

```json
"build": "pnpm check:locales && node scripts/check-demos.mjs && astro build"
```

Run the test and confirm it passes:

```bash
rtk node --test site/scripts/check-demos.test.mjs
```

- [ ] **Step 4: Add the base-path-aware video component**

Create `site/src/components/VhsDemo.astro`:

```astro
---
interface Props {
  src: string;
  poster: string;
  caption: string;
  tapeHref: string;
  downloadLabel: string;
  sourceLabel: string;
}

const { src, poster, caption, tapeHref, downloadLabel, sourceLabel } = Astro.props;
const assetUrl = (path: string) => `${import.meta.env.BASE_URL}${path.replace(/^\/+/, '')}`;
---

<figure class="vhs-demo">
  <video controls muted playsinline preload="metadata" poster={assetUrl(poster)} aria-label={caption}>
    <source src={assetUrl(src)} type="video/webm" />
    <a href={assetUrl(src)}>{downloadLabel}</a>
  </video>
  <figcaption>
    <span>{caption}</span>
    <a href={tapeHref}>{sourceLabel}</a>
  </figcaption>
</figure>

<style>
  .vhs-demo { margin: 1.5rem 0; }
  video { display: block; width: 100%; aspect-ratio: 5 / 3; background: #161616; border: 1px solid var(--sl-color-gray-5); }
  figcaption { display: flex; flex-wrap: wrap; justify-content: space-between; gap: .5rem 1rem; margin-top: .5rem; color: var(--sl-color-gray-2); font-size: var(--sl-text-sm); }
  a { white-space: nowrap; }
</style>
```

- [ ] **Step 5: Add deterministic fixtures and tapes**

Create `site/vhs/setup-repo-demo.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

root=/tmp/leo-cli-vhs
rm -rf "$root"
mkdir -p "$root/config/leo-cli" "$root/data" "$root/repos/atlas/.git" "$root/repos/beacon/.git" "$root/repos/cascade/.git"
printf 'repo:\n  roots:\n    - %s\n' "$root/repos" > "$root/config/leo-cli/config.yaml"
XDG_CONFIG_HOME="$root/config" XDG_DATA_HOME="$root/data" bin/leo repo reindex >/dev/null
echo ready
```

Create `site/vhs/fixtures/users.csv`:

```csv
user_id,character_id
1001,9001
1002,9002
1002,9003
1003,9004
```

Create `site/vhs/repo-picker.tape`:

```tape
Output site/public/demos/repo-picker.webm
Require bin/leo
Require site/vhs/setup-repo-demo.sh
Set Shell "bash"
Set Width 1000
Set Height 600
Set FontSize 20
Set TypingSpeed 45ms
Set Theme { "name": "leo", "black": "#161616", "red": "#ef4444", "green": "#22c55e", "yellow": "#eab308", "blue": "#3b82f6", "magenta": "#d946ef", "cyan": "#14b8a6", "white": "#f5f5f5", "brightBlack": "#737373", "brightRed": "#f87171", "brightGreen": "#4ade80", "brightYellow": "#facc15", "brightBlue": "#60a5fa", "brightMagenta": "#e879f9", "brightCyan": "#2dd4bf", "brightWhite": "#ffffff", "background": "#161616", "foreground": "#f5f5f5", "selection": "#404040", "cursor": "#f5f5f5" }
Env XDG_CONFIG_HOME "/tmp/leo-cli-vhs/config"
Env XDG_DATA_HOME "/tmp/leo-cli-vhs/data"

Hide
Type "site/vhs/setup-repo-demo.sh"
Enter
Wait /ready/
Show
Type "bin/leo repo"
Enter
Wait /Repositories/
Type "bea"
Sleep 1s
Enter
Wait /beacon/
Sleep 1s
Screenshot site/public/demos/repo-picker.png
```

Create `site/vhs/join.tape`:

```tape
Output site/public/demos/join.webm
Require bin/leo
Set Shell "bash"
Set Width 1000
Set Height 600
Set FontSize 20
Set TypingSpeed 45ms
Set Theme { "name": "leo", "black": "#161616", "red": "#ef4444", "green": "#22c55e", "yellow": "#eab308", "blue": "#3b82f6", "magenta": "#d946ef", "cyan": "#14b8a6", "white": "#f5f5f5", "brightBlack": "#737373", "brightRed": "#f87171", "brightGreen": "#4ade80", "brightYellow": "#facc15", "brightBlue": "#60a5fa", "brightMagenta": "#e879f9", "brightCyan": "#2dd4bf", "brightWhite": "#ffffff", "background": "#161616", "foreground": "#f5f5f5", "selection": "#404040", "cursor": "#f5f5f5" }

Type "bin/leo join site/vhs/fixtures/users.csv"
Enter
Wait /SQL IN/
Right
Down 2
Type "u"
Sleep 2s
Screenshot site/public/demos/join.png
Hide
Escape
```

Make the setup script executable:

```bash
rtk chmod +x site/vhs/setup-repo-demo.sh
```

- [ ] **Step 6: Render and inspect both demos**

Run:

```bash
rtk make build
rtk vhs site/vhs/repo-picker.tape
rtk vhs site/vhs/join.tape
rtk ls -lh site/public/demos
rtk pnpm --dir site test
```

Expected: two nonempty WebM files, two nonblank PNG posters, and both Node tests pass. Open both PNG files and inspect the final frame before continuing.

- [ ] **Step 7: Embed shared media in both locales**

Import `VhsDemo` in the Chinese and English home pages and pass the same media paths:

```mdx
<VhsDemo
  src="demos/repo-picker.webm"
  poster="demos/repo-picker.png"
  caption="输入关键词并选择仓库。"
  tapeHref="https://github.com/leowzz/leo-cli/blob/main/site/vhs/repo-picker.tape"
  downloadLabel="下载演示视频"
  sourceLabel="查看 VHS 源码"
/>
```

Use the translated labels in the English home page. Embed `join.webm` and
`join.png` in both `guides/join.mdx` pages with the corresponding localized
caption. Keep an equivalent command block adjacent to each component.

- [ ] **Step 8: Build and commit the demos**

Run:

```bash
rtk pnpm --dir site build
rtk git diff --check
```

Expected: media checks and the full static build pass.

Commit:

```bash
rtk git add site
rtk git commit -m "docs: add reproducible VHS demos"
```

---

### Task 5: Repository Documentation Commands And README Entry Points

**Files:**
- Modify: `Makefile`
- Modify: `README.md`
- Modify: `README.zh.md`

**Interfaces:**
- Consumes: generator, site scripts, and VHS tapes from Tasks 1-4.
- Produces: `make docs-dev`, `make docs-build`, `make docs-demos`, and compact README entry points.

- [ ] **Step 1: Add the three documentation Make targets**

Extend `.PHONY` and append these targets without changing `.DEFAULT_GOAL`:

```make
.PHONY: dev test build release release-github docs-dev docs-build docs-demos

docs-dev:
	go run ./tools/docsgen
	pnpm --dir site dev

docs-build:
	go run ./tools/docsgen
	pnpm --dir site test
	pnpm --dir site build

docs-demos: build
	command -v vhs >/dev/null
	vhs site/vhs/repo-picker.tape
	vhs site/vhs/join.tape
```

- [ ] **Step 2: Verify the Make targets**

Run:

```bash
rtk make docs-build
rtk git diff --exit-code -- site/src/content/docs/reference/commands site/src/content/docs/en/reference/commands
```

Expected: Node tests, locale/media checks, and Astro build pass; regeneration leaves command docs clean.

- [ ] **Step 3: Replace duplicated README manuals with compact entry points**

Keep both README files under roughly 120 lines with exactly these sections:

```text
Project summary and six workflow bullets
Install from GitHub Releases or make build
Quick start: repo reindex, repo, shell init
Documentation link
Development: make dev, make test, make build
```

Use these links:

```text
Chinese: https://leowzz.github.io/leo-cli/
English: https://leowzz.github.io/leo-cli/en/
```

Remove the duplicated SQL, time, Docker, log viewer, full configuration,
command table, and project-layout sections only after their bilingual site pages
exist. Keep command examples identical to the site.

- [ ] **Step 4: Verify README links and existing Go behavior**

Run:

```bash
rtk grep -n "https://leowzz.github.io/leo-cli" README.md README.zh.md
rtk go run . --help
rtk go test ./...
rtk make docs-build
```

Expected: each README links to its locale, Go tests pass, and the documentation build passes.

- [ ] **Step 5: Commit repository integration**

```bash
rtk git add Makefile README.md README.zh.md
rtk git commit -m "docs: make the site the primary manual"
```

---

### Task 6: GitHub Pages Workflow

**Files:**
- Create: `.github/workflows/docs.yml`

**Interfaces:**
- Consumes: `site/pnpm-lock.yaml`, `.node-version`, docs generator, site tests, and `site/dist`.
- Produces: a Pages artifact and deployment from `main` without touching `.github/workflows/release.yml`.

- [ ] **Step 1: Create the Pages workflow**

Create `.github/workflows/docs.yml`:

```yaml
name: Docs

on:
  push:
    branches: [main]
    paths:
      - ".github/workflows/docs.yml"
      - "cmd/**"
      - "site/**"
      - "tools/docsgen/**"
      - "go.mod"
      - "go.sum"
      - "Makefile"
      - "README.md"
      - "README.zh.md"
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: pages
  cancel-in-progress: false

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
      - uses: pnpm/action-setup@v6
        with:
          version: 10.30.3
      - uses: actions/setup-node@v6
        with:
          node-version-file: site/.node-version
          cache: pnpm
          cache-dependency-path: site/pnpm-lock.yaml
      - uses: actions/configure-pages@v6
      - run: pnpm --dir site install --frozen-lockfile
      - run: go run ./tools/docsgen
      - run: git diff --exit-code -- site/src/content/docs
      - run: pnpm --dir site test
      - run: pnpm --dir site build
      - uses: actions/upload-pages-artifact@v5
        with:
          path: site/dist

  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    needs: build
    runs-on: ubuntu-latest
    steps:
      - name: Deploy
        id: deployment
        uses: actions/deploy-pages@v5
```

- [ ] **Step 2: Validate the workflow inputs locally**

Run:

```bash
rtk pnpm --dir site install --frozen-lockfile
rtk go run ./tools/docsgen
rtk git diff --exit-code -- site/src/content/docs
rtk pnpm --dir site test
rtk pnpm --dir site build
rtk git diff --check
```

Expected: all commands pass and command generation leaves no diff.

- [ ] **Step 3: Confirm the release workflow is unchanged**

Run:

```bash
rtk git diff HEAD -- .github/workflows/release.yml
```

Expected: no output.

- [ ] **Step 4: Commit the deployment workflow**

```bash
rtk git add .github/workflows/docs.yml
rtk git commit -m "ci: deploy documentation to GitHub Pages"
```

---

### Task 7: Production Build And Browser Verification

**Files:**
- Modify only if verification exposes a defect in files created by Tasks 1-6.
- Create temporary screenshots outside the repository or under an ignored directory.

**Interfaces:**
- Consumes: the complete site and workflows.
- Produces: verified production output and a locally running preview URL for handoff.

- [ ] **Step 1: Run the complete repository verification**

Run:

```bash
rtk go test ./...
rtk make build
rtk make docs-build
rtk git diff --exit-code -- site/src/content/docs
rtk git diff --check
```

Expected: Go tests, binary build, Node tests, locale/media checks, Astro/Pagefind build, generated-doc cleanliness, and whitespace checks all pass.

- [ ] **Step 2: Start the production preview**

Run:

```bash
rtk pnpm --dir site preview --host 127.0.0.1 --port 4321
```

Keep the process running for browser verification. The local URL is
`http://127.0.0.1:4321/leo-cli/`.

- [ ] **Step 3: Verify desktop and mobile routes with Playwright**

At 375x812, 768x1024, 1024x768, and 1440x900, verify:

```text
/leo-cli/
/leo-cli/en/
/leo-cli/guides/join/
/leo-cli/en/guides/join/
/leo-cli/reference/commands/leo/
/leo-cli/en/reference/commands/leo/
```

For every route confirm:

- No horizontal overflow, overlap, clipped labels, or hidden navigation.
- Chinese pages have `lang="zh-CN"`; English pages have `lang="en"`.
- Locale switching reaches the corresponding page.
- The GitHub link and internal links include the correct base path.
- Code blocks remain readable at mobile width.

- [ ] **Step 4: Verify search and media behavior**

In the production preview:

- Search Chinese for `仓库` and confirm only Chinese results.
- Search English for `repository` and confirm only English results.
- Confirm each poster has nonblank pixels before playback.
- Confirm both WebM files have native controls, do not autoplay, and play.
- Confirm each tape-source link opens the correct repository path.
- Capture one desktop and one mobile screenshot per locale.

- [ ] **Step 5: Fix only observed defects and repeat the full verification**

After any correction, rerun:

```bash
rtk go test ./...
rtk make docs-build
rtk git diff --check
```

Expected: all checks pass with fresh output. Commit only actual fixes:

```bash
rtk git add site .github/workflows/docs.yml Makefile README.md README.zh.md cmd/root.go tools/docsgen go.mod go.sum
rtk git commit -m "fix: polish documentation site verification issues"
```

Skip this commit when verification required no changes.

- [ ] **Step 6: Final handoff**

Report:

- The local preview URL.
- The exact verification commands and results.
- The Pages URL that will be published after the branch reaches `main`.
- Whether GitHub repository settings still need Pages source set to GitHub Actions.

Do not claim the public Pages deployment succeeded unless its GitHub Actions run has completed successfully.
