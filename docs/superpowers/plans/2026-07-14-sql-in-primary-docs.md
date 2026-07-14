# SQL IN Primary Documentation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `leo join` the first feature users see and try across the bilingual README and documentation site.

**Architecture:** Reorder and rewrite existing documentation entry points; reuse the existing join guide and `join.webp` demo. Keep all command behavior, assets, and secondary feature documentation unchanged.

**Tech Stack:** Markdown, MDX, Astro Starlight

## Global Constraints

- Chinese and English pages must keep the same hierarchy.
- Do not change CLI behavior or add assets.
- Preserve repository, time, Docker, and log documentation.

---

### Task 1: Make SQL IN The Primary Documentation Path

**Files:**
- Modify: `README.md`
- Modify: `README.zh.md`
- Modify: `site/astro.config.mjs`
- Modify: `site/src/content/docs/index.mdx`
- Modify: `site/src/content/docs/en/index.mdx`
- Modify: `site/src/content/docs/getting-started.md`
- Modify: `site/src/content/docs/en/getting-started.md`

**Interfaces:**
- Consumes: existing `leo join` command, `guides/join` pages, and `demos/join.webp`
- Produces: matching Chinese and English documentation paths led by `leo join`

- [x] **Step 1: Update README entry points**

Move SQL `IN` construction to the first feature-list item in both README files. Replace the repository-first quick start with the zero-configuration command:

```bash
leo join
```

- [x] **Step 2: Update documentation home pages**

Put SQL `IN` first in each homepage description, replace the quick-start block with `leo join`, switch `repo-picker.webp` to `join.webp`, and link the task-guide call to `./guides/join/`. Use these captions:

```text
选择 CSV 字段和输出格式，并复制 SQL IN 列表。
Choose a CSV field and output format, then copy the SQL IN list.
```

- [x] **Step 3: Update Getting Started pages**

Change each page description to lead with SQL `IN`. Add the first post-install section before repository indexing with these runnable flows:

```bash
leo join
seq 1 10 | leo join
```

Explain that `leo join` reads copied values from the clipboard, the pipeline takes priority when present, and Enter copies the chosen output. Keep repository indexing and shell integration below it.

- [x] **Step 4: Reorder the Guides sidebar**

In `site/astro.config.mjs`, place `{ slug: 'guides/join' }` before `{ slug: 'guides/repositories' }`. Do not change any other navigation items.

- [x] **Step 5: Verify source consistency**

Run:

```bash
rtk grep -n -C 3 "Quick Start\|快速开始\|Create The First Index\|第一次建立索引\|guides/repositories\|repo-picker.webp" README.md README.zh.md site/src/content/docs/index.mdx site/src/content/docs/en/index.mdx site/src/content/docs/getting-started.md site/src/content/docs/en/getting-started.md site/astro.config.mjs
```

Expected: repository sections remain in Getting Started, while README and homepage quick starts no longer lead with repository setup; homepage demo and guide links no longer reference the repository picker.

- [x] **Step 6: Build the documentation**

Run:

```bash
rtk make docs-build
```

Expected: command reference generation, locale checks, and Astro build all pass.

- [x] **Step 7: Check and commit the diff**

Run:

```bash
rtk git diff --check
rtk git status --short
```

Expected: no whitespace errors; only the plan and seven intended documentation files are modified. Commit with:

```bash
rtk git add README.md README.zh.md site/astro.config.mjs site/src/content/docs/index.mdx site/src/content/docs/en/index.mdx site/src/content/docs/getting-started.md site/src/content/docs/en/getting-started.md docs/superpowers/plans/2026-07-14-sql-in-primary-docs.md
rtk git commit -m "docs: feature SQL IN workflow"
```
