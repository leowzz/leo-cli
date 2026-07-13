---
title: Development And Releases
description: Run, test, build, and release leo-cli from source.
---

## Common Commands

Run from source:

```bash
make dev
```

Test and build:

```bash
make test
make build
```

`make build` writes a binary with version metadata to `bin/leo`. The version defaults to the `version=` value in `.env`.

## Release

Create a tag and bump the patch version, or provide an explicit version:

```bash
make release
make release V=v0.1.0
```

Build cross-platform assets, push the tag, and publish a GitHub Release:

```bash
make release-github
make release-github V=v0.1.0
```

`release-github` requires an installed and authenticated [GitHub CLI](https://cli.github.com/).

## Code Layout

```text
cmd/                 Cobra command wiring
internal/config/     YAML config, default paths, path expansion
internal/dockercopy/ Docker image reference and registry alias resolution
internal/logview/    Safe log discovery, parsing, search, and follow
internal/logweb/     Authenticated HTTP APIs and embedded browser workspace
internal/project/    Current-project matching and root resolution
internal/repoindex/  Git repository scanning and metadata extraction
internal/repoui/     Bubble Tea repository picker
internal/store/      SQLite storage and migrations
scripts/             Release helper scripts
```
