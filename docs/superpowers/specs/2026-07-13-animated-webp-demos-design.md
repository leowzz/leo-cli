# Animated WebP Demo Design

## Goal

Replace the two embedded VHS WebM videos with animated WebP images that start automatically and loop indefinitely.

## Scope

- Keep the existing `repo-picker` and `join` VHS demonstrations.
- Keep their dimensions, captions, and links to the tape sources.
- Remove WebM playback controls, PNG posters, and media download fallback text.
- Do not add JavaScript playback controls or retain duplicate video assets.

## Generation

VHS renders each tape to a temporary WebM because it does not output animated WebP directly. `make docs-demos` converts each temporary WebM with an ffmpeg build that provides the `libwebp_anim` encoder, configures infinite looping, and removes the temporary file. The target uses `FFMPEG ?= ffmpeg` so a keg-only Homebrew `ffmpeg-full` binary can be supplied without hard-coding a platform path.

The committed outputs are:

- `site/public/demos/repo-picker.webp`
- `site/public/demos/join.webp`

## Rendering

`VhsDemo.astro` renders the animated WebP with a responsive `<img>` inside the existing figure. The caption remains the image alternative text, preserving the current accessible description and page layout.

## Validation

- The demo asset checker requires exactly the two WebP outputs used by the component.
- Its existing test is updated first to fail against the old asset contract.
- The VHS tapes, MDX call sites, generated assets, and checker are then updated.
- Verification covers Go tests, documentation tests/build, asset-format inspection, locale parity, and desktop/mobile browser rendering.
