# leo-cli

Personal command-line tools, starting with a local repository browser.

`leo-cli` builds a `leo` binary. The current CLI can index Git repositories
under configured local roots, open an interactive terminal picker, and print the
selected repository path. With the shell integration enabled, the selected path
is used to `cd` directly into the repository.

## Features

- Recursively scans configured repository roots.
- Detects normal Git repositories and worktree-style `.git` files.
- Stores repository metadata in a local SQLite database.
- Sorts repositories by recent Git activity.
- Shows an interactive fuzzy-filtered terminal list.
- Displays branch, last commit time, and full path in the picker.
- Provides a shell helper so `repo` can jump to the selected repository.
- Copies container images between registries with short aliases via `leo docker copy`.
- Builds SQL `IN` value lists from txt or csv files and copies the result.

## Requirements

- Go 1.25.6 or newer, matching `go.mod`.
- Git, for fallback commit metadata lookup during indexing.
- Skopeo, for `leo docker copy`.
- A terminal environment for the interactive picker.

## Install

Download a prebuilt binary from [GitHub Releases](https://github.com/leowzz/leo-cli/releases).
Assets include the release tag and platform, for example `leo-v0.1.0-darwin-arm64`
or `leo-v0.1.0-windows-amd64.exe`.
Make the file executable and place it on your `PATH`:

```bash
chmod +x leo-v0.1.0-darwin-arm64
mv leo-v0.1.0-darwin-arm64 ~/bin/leo
```

Or build locally:

```bash
make build
```

The binary is written to:

```text
bin/leo
```

To install it somewhere on your `PATH`, copy or symlink `bin/leo` into a
directory such as `~/bin` or `/usr/local/bin`.

For development, you can also run the CLI directly:

```bash
go run .
```

## Quick Start

Create or update the repository index:

```bash
leo repo reindex
```

Open the interactive repository picker:

```bash
leo repo
```

Type to filter. Press Enter to print the selected repository path. Press Escape
or Ctrl-C to exit without selecting anything.

Enable the shell integration so `repo` changes into the selected directory:

```bash
eval "$(leo shell init zsh)"
```

For bash:

```bash
eval "$(leo shell init bash)"
```

To make this permanent, add the matching `eval` line to your shell startup file,
for example `~/.zshrc` or `~/.bashrc`.

After that, run:

```bash
repo
```

The shell function opens the picker and runs `cd` with the selected path.

## Configuration

Config is stored at:

```text
~/.config/leo-cli/config.yaml
```

If the file does not exist, `leo` creates this default:

```yaml
repo:
  roots:
    - ~/work
```

`repo.roots` is the list of directories scanned by `leo repo reindex`.
`docker.registries` maps short aliases to Docker registry domains for
`leo docker copy` and `leo docker list`.

Paths support:

- `~` for the current user's home directory.
- Environment variables supported by Go's `os.ExpandEnv`, such as `$HOME`.
- Relative paths, which are converted to absolute paths.

Example:

```yaml
repo:
  roots:
    - ~/work
    - ~/repo
    - $HOME/src
docker:
  registries:
    it: source-registry.example.com
    t: mirror-registry.example.com
```

Missing or unreadable roots are reported as warnings. Reindexing only fails when
all configured roots are unusable.

## Data

The repository index is stored at:

```text
~/.local/share/leo-cli/leo-cli.sqlite3
```

If `XDG_CONFIG_HOME` or `XDG_DATA_HOME` is set, `leo` uses those locations
instead of `~/.config` and `~/.local/share`.

The database uses SQLite with WAL mode. Repository rows are keyed by absolute
path and include:

- repository path
- display name
- current branch
- last commit time
- last Git activity time
- last indexed time

Reindexing upserts discovered repositories. It does not currently remove stale
repositories that no longer exist on disk.

## Commands

```bash
leo --version
leo version
```

Print version and commit metadata.

```bash
leo repo reindex
```

Scan configured roots and update the local repository index.

```bash
leo repo
```

Open the interactive repository picker. If a repository is selected, print its
absolute path to stdout.

```bash
leo shell init zsh
leo shell init bash
```

Print shell integration code for the requested shell.

```bash
leo join ids.txt
leo join ids.csv
```

Open an interactive picker for building SQL `IN` values. Use Left/Right to
switch CSV columns, Up/Down to switch output format, Enter to copy the preview
to the clipboard, and Escape to cancel.

```bash
leo docker list
```

Print configured Docker registry aliases.

`docker copy` resolves registry aliases from config, then invokes `skopeo copy`.
Skopeo talks to registries directly; it does not use the local Docker daemon or
`docker context`.

#### Source and destination formats

Source is resolved in two steps:

1. If the first path segment matches a configured alias, expand it as
   `ALIAS/REPOSITORY[:TAG]`.
2. Otherwise treat the value as a full image reference (no alias error).

When the destination is only a registry alias or domain, the command reuses the
source repository path and tag. When the destination includes `/REPOSITORY[:TAG]`,
that explicit target is used instead.

Examples:

```bash
# Alias source, reuse repository and tag on destination alias
leo docker copy it/apps/example-service:v1.2.4 t

# Alias source, explicit destination repository
leo docker copy it/apps/example-service:v1.2.4 t/library/example-service:latest

# Full registry source and destination (no aliases required)
leo docker copy registry.example.com/app:v1 mirror.example.com/app:v1

# Single-segment Docker Hub image; source becomes docker.io/library/python:3.12
leo docker copy python:3.12 t

# Full source reference preserved; destination reuses example-user/python:3.12
leo docker copy registry.example.com/example-user/python:3.12 t
```

#### Flags

```bash
leo docker copy --dry it/apps/example-service:v1.2.4 t
```

Print the rendered `skopeo copy` command without running it.

```bash
leo docker copy python:3.12-slim tx/example-user/python:3.12-slim
```

Copy a single platform from multi-arch images. Default is `linux/amd64`. On
macOS, skopeo otherwise selects `darwin/<arch>` from manifest lists and often
fails for official Linux-only images.

```bash
leo docker copy python:3.12-slim tx/example-user/python:3.12-slim --platform linux/arm64
leo docker copy python:3.12-slim tx/example-user/python:3.12-slim --platform linux/arm64/v8
```

`--platform` accepts `OS/ARCH` or `OS/ARCH/VARIANT` and maps to skopeo
`--override-os`, `--override-arch`, and optionally `--override-variant`.

Dry-run example:

```bash
leo docker copy python:3.12 t --dry
# skopeo copy --override-os linux --override-arch amd64 docker://docker.io/library/python:3.12 docker://...
```

## Development

Run the CLI from source:

```bash
make dev
```

Run tests:

```bash
make test
```

Build:

```bash
make build
```

The build embeds version metadata from `.env`:

```text
version=v0.0.0
```

The command name defaults to `leo` and is embedded during `make build`.

Create a release tag and bump the patch version from `.env`:

```bash
make release
```

Set an explicit release version:

```bash
make release V=v0.1.0
```

Build cross-platform binaries, push the tag, and publish a GitHub release with
`gh` (requires the GitHub CLI and authenticated `gh auth`):

```bash
make release-github
make release-github V=v0.1.0
```

This uploads raw binaries named `leo-<tag>-<os>-<arch>` for darwin/linux/windows
on amd64 and arm64. Windows assets use the `.exe` suffix. See
`scripts/release-github.sh` for details.

## Project Layout

```text
cmd/                 Cobra command wiring
internal/config/     YAML config, default paths, path expansion
internal/dockercopy/ Docker image reference and registry alias resolution
internal/refresh/    Initial and background metadata refresh behavior
internal/repoindex/  Git repository scanning and metadata extraction
internal/repoui/     Bubble Tea repository picker
internal/shellinit/  Shell integration script generation
internal/store/      SQLite storage and migrations
internal/version/    Build-time version metadata
scripts/             Release helpers
```
