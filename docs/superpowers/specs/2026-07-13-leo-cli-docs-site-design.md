# leo-cli Documentation Site Design

**Date:** 2026-07-13

## Summary

Build a bilingual documentation site for `leo-cli` with Astro Starlight. Serve
Simplified Chinese at the site root and English under `/en/`. Deploy the static
site to GitHub Pages and use small, reproducible VHS recordings only where
motion explains terminal interaction better than text.

The documentation site becomes the detailed user guide. The repository README
files remain short entry points containing the project summary, installation,
and links to the documentation site.

## Goals

- Publish complete Simplified Chinese and English documentation.
- Keep the same page structure and slugs in both languages.
- Provide fast, language-aware, local full-text search.
- Generate CLI reference pages from the Cobra command tree.
- Demonstrate the interactive repository picker and SQL join UI with VHS.
- Deploy automatically to GitHub Pages without affecting release builds.
- Fail the documentation build when locale coverage or generated reference
  content has drifted.

## Non-goals

- Versioned documentation for every `leo-cli` release.
- A blog, marketing site, analytics pipeline, CMS, or external search service.
- Automatic translation or semantic comparison of translations.
- VHS recordings for every command.
- Publishing internal `docs/superpowers` plans and specifications.
- Redesigning the existing log viewer UI.

## Technology

- Astro Starlight for the documentation site.
- Markdown and MDX for content.
- Starlight's built-in Pagefind integration for static search.
- pnpm for site dependencies and scripts.
- GitHub Actions and GitHub Pages for deployment.
- VHS for scripted terminal recordings.
- Cobra's `doc` package for generated command reference pages.

Keep all Node dependencies inside `site/`. The Go build and the existing
release workflow remain independent of the documentation toolchain.

## URLs And Locales

The GitHub Pages project site uses `/leo-cli/` as its base path:

```text
https://leowzz.github.io/leo-cli/       Simplified Chinese
https://leowzz.github.io/leo-cli/en/    English
```

Configure Starlight with a `root` locale whose language is `zh-CN` and an `en`
locale. Chinese pages live directly in the content root. English pages live in
the matching `en/` subtree.

Equivalent pages use the same relative path and file name. Starlight's locale
fallback remains available during local authoring, but the production check
requires every page to exist in both languages.

## Repository Layout

```text
site/
  .node-version
  astro.config.mjs
  package.json
  pnpm-lock.yaml
  scripts/
    check-locales.mjs
  src/
    components/
      VhsDemo.astro
    content/
      docs/
        index.md
        getting-started/
        guides/
        reference/
        concepts/
        development/
        en/
          index.md
          getting-started/
          guides/
          reference/
          concepts/
          development/
  public/
    demos/
      repo-picker.webm
      repo-picker.png
      join.webm
      join.png
  vhs/
    fixtures/
    repo-picker.tape
    join.tape
tools/
  docsgen/
    main.go
.github/workflows/
  docs.yml
```

Do not move or publish `docs/superpowers`. The separate `site/` directory keeps
public documentation and internal design history from being confused.

## Information Architecture

### Home

- A short project description.
- A copyable installation and first-run path.
- The repository picker VHS demo.
- Links to the main task guides and command reference.

The home page is a compact documentation entry point, not a marketing landing
page.

### Getting Started

- Installation.
- Shell integration.
- Configuration file basics.
- Creating the first repository index.

### Guides

- Switching repositories.
- Building SQL `IN` values.
- Converting timestamps and timezones.
- Copying Docker images.
- Searching and following project logs.

### Reference

- Generated command reference.
- Configuration fields.
- Config and data file paths.
- Environment variables.

### Concepts

- Repository indexing.
- Log viewer security boundaries.
- Local storage and privacy.

### Development

- Local development.
- Tests.
- Release process.

## Content Ownership

Task guides and explanations are maintained as bilingual Markdown. The Chinese
and English trees have identical page paths but independently written prose.

Commands, flags, usage strings, and errors remain in the same English form the
CLI prints. Chinese pages translate the surrounding explanation but do not
translate terminal output. This keeps examples faithful to the executable.

After the site is published, reduce `README.md` and `README.zh.md` to the
project overview, installation, quick start, and links to the matching site
locale. Do not maintain full copies of the guides in README files.

## Locale Parity Check

`site/scripts/check-locales.mjs` uses Node standard-library filesystem APIs to
collect every Chinese Markdown and MDX path outside `en/` and compare it with
the `en/` subtree. Generated command pages participate in the same check;
non-content assets do not. The check fails when either locale is missing a
matching page.

The check verifies file coverage only. It does not attempt automatic
translation or semantic comparison.

## Generated Command Reference

`tools/docsgen` reuses the existing Cobra command tree and Cobra's `doc`
package to generate Markdown reference pages. Generate the same command syntax,
flags, descriptions, and links into both locale trees. Add locale-appropriate
frontmatter and a short localized note around the generated CLI output.

Commit generated reference pages. During CI:

1. Run the generator.
2. Run the locale parity check.
3. Fail if `git diff --exit-code` reports generated changes.

This makes command code the source of truth while keeping documentation builds
deterministic.

## VHS Demos

Create two demos initially:

- `repo-picker`: filter repositories, select one, and print its path.
- `join`: open a fixed CSV fixture, change the selected field or format, and
  display the result.

Use fixture repositories, CSV input, and temporary `XDG_CONFIG_HOME` and
`XDG_DATA_HOME` paths so recordings do not depend on the developer's machine.
Hide setup commands from the recording. Fix the terminal dimensions, theme,
typing speed, and delays in each tape.

Generate a WebM and PNG poster for each tape. Commit tapes, fixtures, and
rendered media. `make docs-demos` performs explicit regeneration. Normal site
builds consume committed media and do not install VHS, ttyd, or ffmpeg.

Do not create VHS recordings for commands whose behavior is clearer as static
text. Use code blocks for `time` and `docker copy --dry`. Use screenshots or a
browser recording for the browser-based log viewer if visual documentation is
later required.

## VHS Component

`VhsDemo.astro` renders one shared media component for both locales. It accepts
the media path, poster path, localized caption, and tape source link.

The component:

- Uses WebM as the primary format.
- Does not autoplay.
- Shows native playback controls.
- Uses `playsinline` and muted playback.
- Includes a fallback download link.
- Places an equivalent command description next to the media.
- Respects reduced-motion preferences.

Both locales reference the same rendered media files.

## Visual Direction

Keep Starlight's default documentation layout and accessibility behavior. Apply
only a small theme override:

- Dense, predictable navigation for repeated technical use.
- Search-first header behavior.
- High-contrast neutral surfaces with restrained blue-green accents.
- System UI fonts and a readable monospace stack.
- Visible keyboard focus and stable hover states.
- No oversized marketing hero, decorative cards, gradients, or ornamental
  animation.

## Local Commands

Add these Makefile targets without changing the existing default target:

```text
make docs-dev       Start the Starlight development server.
make docs-build     Generate and validate reference docs, check locales, build.
make docs-demos     Regenerate committed VHS media.
```

`make docs-build` is the single local verification entry point for the site.

## GitHub Pages Deployment

Add `.github/workflows/docs.yml` alongside the existing release workflow. The
workflow runs on pushes to `main` and through `workflow_dispatch`.

The workflow:

1. Checks out the repository.
2. Sets up Go from `go.mod`, Node 24 from `site/.node-version`, and the exact
   pnpm 10 release recorded in `site/package.json`.
3. Installs site dependencies with the frozen lockfile.
4. Generates command reference pages.
5. Checks locale parity.
6. Fails if generated files changed.
7. Builds the Starlight site with base path `/leo-cli/`.
8. Uploads `site/dist` as the Pages artifact.
9. Deploys the artifact with the official GitHub Pages actions.

The existing tag-triggered release workflow is unchanged.

## Error Handling And Validation

The documentation pipeline fails for:

- A page present in only one locale.
- Stale generated command reference pages.
- Invalid Starlight content or build errors.
- Broken imports or missing build-time assets.

Before completion, verify:

- Chinese routes under `/leo-cli/` and English routes under `/leo-cli/en/`.
- Locale switching preserves the corresponding page where it exists.
- Search returns results only for the active language and handles Chinese text.
- Navigation, code blocks, and media fit at 375, 768, 1024, and 1440 pixel
  viewports.
- Both WebM files display a nonblank poster and working controls.
- VHS media does not autoplay.
- Direct links and static assets work under the GitHub Pages base path.
- Regenerating command docs leaves a clean worktree.
- Existing Go tests and release builds still pass.

Use Playwright screenshots for desktop and mobile visual verification after the
site is implemented.

## Success Criteria

- The GitHub Pages site publishes successfully from `main`.
- Chinese is served at the project site root and English under `/en/`.
- Every public content page exists in both languages.
- Pagefind search works in both languages, including Chinese segmentation.
- Command reference content matches the current Cobra command tree.
- The repository picker and join pages include reproducible, accessible VHS
  demos.
- README files point users to the site instead of duplicating the full manual.
