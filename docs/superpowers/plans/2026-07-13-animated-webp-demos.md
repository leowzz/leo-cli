# Animated WebP Demos Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace both embedded VHS WebM videos and PNG posters with infinitely looping animated WebP images.

**Architecture:** VHS keeps producing a temporary WebM for each tape. `make docs-demos` converts it with ffmpeg's `libwebp_anim` encoder, deletes the temporary input, and commits only the WebP; the Astro component becomes a responsive accessible image.

**Tech Stack:** VHS, ffmpeg-full/libwebp_anim, Astro/Starlight, Node test runner, animated WebP.

## Global Constraints

- Keep exactly the existing `repo-picker` and `join` demonstrations.
- Keep their 1000x600 dimensions, captions, and tape-source links.
- Animated WebP starts automatically and loops indefinitely.
- Remove WebM playback controls, PNG posters, download fallback text, and duplicate media assets.
- Use `FFMPEG ?= ffmpeg`; do not hard-code a Homebrew path in repository files.
- Do not add a Node or Go dependency.

---

### Task 1: Replace Video Assets With Animated WebP

**Files:**
- Modify: `site/scripts/check-demos.test.mjs`
- Modify: `site/scripts/check-demos.mjs`
- Modify: `site/vhs/repo-picker.tape`
- Modify: `site/vhs/join.tape`
- Modify: `Makefile`
- Modify: `site/src/components/VhsDemo.astro`
- Modify: `site/src/content/docs/index.mdx`
- Modify: `site/src/content/docs/en/index.mdx`
- Modify: `site/src/content/docs/guides/join.mdx`
- Modify: `site/src/content/docs/en/guides/join.mdx`
- Delete: `site/public/demos/repo-picker.webm`
- Delete: `site/public/demos/repo-picker.png`
- Delete: `site/public/demos/join.webm`
- Delete: `site/public/demos/join.png`
- Create: `site/public/demos/repo-picker.webp`
- Create: `site/public/demos/join.webp`

**Interfaces:**
- Consumes: VHS tapes and `FFMPEG`, defaulting to the `ffmpeg` executable on `PATH`.
- Produces: `VhsDemo` props `{ src, caption, tapeHref, sourceLabel }` and two 1000x600 looping animated WebP files.

- [ ] **Step 1: Write the failing asset-contract test**

Replace the test fixture and expected result in `site/scripts/check-demos.test.mjs` so only one new-format asset exists:

```js
await writeFile(join(root, 'demos', 'repo-picker.webp'), 'image');
assert.deepEqual(await findMissingDemos(root), ['demos/join.webp']);
```

- [ ] **Step 2: Run the test and verify RED**

Run:

```bash
rtk pnpm --dir site test
```

Expected: the demo test fails because `findMissingDemos()` still expects PNG and WebM files.

- [ ] **Step 3: Implement the minimal asset contract**

Change `required` in `site/scripts/check-demos.mjs` to:

```js
const required = [
  'demos/join.webp',
  'demos/repo-picker.webp',
];
```

Run `rtk pnpm --dir site test`; expected: both Node tests pass.

- [ ] **Step 4: Make VHS outputs temporary**

Change the first line of each tape to:

```text
Output site/public/demos/repo-picker.tmp.webm
```

```text
Output site/public/demos/join.tmp.webm
```

Remove both `Screenshot site/public/demos/*.png` commands. Keep all terminal actions and timings unchanged.

- [ ] **Step 5: Convert each temporary WebM in `make docs-demos`**

Add near the existing Make variables:

```makefile
FFMPEG ?= ffmpeg
```

Make the target require ffmpeg and convert each recording immediately after VHS renders it:

```makefile
docs-demos: build
	command -v vhs >/dev/null
	command -v $(FFMPEG) >/dev/null
	vhs site/vhs/repo-picker.tape
	$(FFMPEG) -y -i site/public/demos/repo-picker.tmp.webm -an -c:v libwebp_anim -preset text -quality 85 -loop 0 site/public/demos/repo-picker.webp
	rm -f site/public/demos/repo-picker.tmp.webm
	vhs site/vhs/join.tape
	$(FFMPEG) -y -i site/public/demos/join.tmp.webm -an -c:v libwebp_anim -preset text -quality 85 -loop 0 site/public/demos/join.webp
	rm -f site/public/demos/join.tmp.webm
```

- [ ] **Step 6: Render the component as an image**

Replace `VhsDemo.astro` with the same figure and caption structure, but use these props and media element:

```astro
interface Props {
  src: string;
  caption: string;
  tapeHref: string;
  sourceLabel: string;
}

const { src, caption, tapeHref, sourceLabel } = Astro.props;
const assetUrl = (path: string) => `${import.meta.env.BASE_URL}${path.replace(/^\/+/, '')}`;
```

```astro
<figure class="vhs-demo">
  <img src={assetUrl(src)} alt={caption} width="1000" height="600" loading="lazy" decoding="async" />
  <figcaption>
    <span>{caption}</span>
    <a href={tapeHref}>{sourceLabel}</a>
  </figcaption>
</figure>
```

Keep the existing figure/caption styles and move the current responsive media styles from `video` to `img`.

- [ ] **Step 7: Update all four bilingual call sites**

For `repo-picker`, use:

```astro
<VhsDemo
  src="demos/repo-picker.webp"
  caption="输入关键词并选择仓库。"
  tapeHref="https://github.com/leowzz/leo-cli/blob/main/site/vhs/repo-picker.tape"
  sourceLabel="查看 VHS 源码"
/>
```

Preserve the existing English caption/source label. Apply the same change to both `join` pages with `src="demos/join.webp"`. Remove only the obsolete `poster` and `downloadLabel` props.

- [ ] **Step 8: Generate and inspect the media**

Run:

```bash
rtk make docs-demos FFMPEG=/opt/homebrew/opt/ffmpeg-full/bin/ffmpeg
rtk file site/public/demos/repo-picker.webp site/public/demos/join.webp
rtk webpmux -info site/public/demos/repo-picker.webp
rtk webpmux -info site/public/demos/join.webp
```

Expected: both files are 1000x600 animated WebP images, contain multiple frames, and report loop count `0`.

Delete the four obsolete committed PNG/WebM files with `rtk git rm` if generation did not already replace them.

- [ ] **Step 9: Run complete verification**

Run:

```bash
rtk go test ./...
rtk pnpm --dir site test
rtk pnpm --dir site build
rtk git diff --check
```

Expected: 161 Go tests and 2 Node tests pass; Starlight builds 57 pages and Pagefind completes.

Start the production preview:

```bash
rtk pnpm --dir site preview --host 127.0.0.1 --port 4321
```

At 1440x900 and 390x844, inspect `/leo-cli/`, `/leo-cli/en/`, `/leo-cli/guides/join/`, and `/leo-cli/en/guides/join/`. Confirm the `<img>` uses `.webp`, no `<video>` remains, the image has nonblank pixels, frames change over time, captions/source links remain readable, and there is no horizontal overflow.

- [ ] **Step 10: Commit**

```bash
rtk git add Makefile site
rtk git commit -m "docs: replace VHS videos with animated WebP"
```
