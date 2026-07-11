# leo-cli

[中文](https://github.com/leowzz/leo-cli/blob/main/README.zh.md) | English

`leo-cli` builds a `leo` binary for personal command-line workflows. It currently covers five practical jobs:

- Index local Git repositories and pick one from an interactive terminal list.
- Generate a shell helper so `repo` can jump into the selected repository.
- Build SQL `IN` values from clipboard, txt, or csv input and copy the result.
- Copy container images with registry aliases through `skopeo copy`.
- Search and follow configured project logs in a temporary browser workspace.

## Install

Download a binary from [GitHub Releases](https://github.com/leowzz/leo-cli/releases). Asset names look like:

```text
leo-v0.0.9-darwin-arm64
leo-v0.0.9-linux-amd64
leo-v0.0.9-windows-amd64.exe
```

macOS / Linux:

```bash
chmod +x leo-v0.0.9-darwin-arm64
mv leo-v0.0.9-darwin-arm64 ~/bin/leo
```

Windows:

```powershell
ren leo-v0.0.9-windows-amd64.exe leo.exe
```

Put the directory containing `leo` or `leo.exe` on your `PATH`.

Or build locally:

```bash
make build
```

The binary is written to:

```text
bin/leo
```

## Quick Start

Create the repository index:

```bash
leo repo reindex
```

Open the repository picker:

```bash
leo repo
```

Type to filter. Press Enter to print the selected repository path. Press Esc or Ctrl-C to cancel.

Enable the shell jump helper:

```bash
eval "$(leo shell init zsh)"
```

bash:

```bash
eval "$(leo shell init bash)"
```

Add the matching `eval` line to `~/.zshrc` or `~/.bashrc`, then run:

```bash
repo
```

`repo` opens the picker and runs `cd` after a repository is selected.

## SQL IN Helper

Read from clipboard:

```bash
leo join
```

Read from stdin:

```bash
seq 1 10 | leo join
```

Read from file:

```bash
leo join ids.txt
leo join ids.csv
```

In the interactive picker:

- Left/Right: switch csv columns.
- Up/Down: switch output formats.
- u: toggle unique/original values.
- Enter: copy the current result.
- Esc: cancel.

When no file is provided, piped stdin is used before clipboard. Values are unique by default, preserving first-seen order. Output formats include comma lists, parenthesized lists, `field in (...)`, and quoted lists.

## Time Converter

Convert Unix seconds, Unix milliseconds, and common date-time strings:

```bash
leo time
leo time 1783512043
leo time 1783512043000
leo time "(2026-07-08 20:00:43)"
```

Without a value, `leo time` uses the current time.

Date-time strings without an explicit timezone are treated as UTC+8. Use `--to` to choose the output timezone:

```bash
leo time 1783512043 --to +9
leo time "2026-07-08 20:00:43" --to +9
leo time 1783512043 --to Asia/Tokyo
```

Configured `time.zones` are printed as extra common timezone rows. Values can be UTC offsets or IANA timezone names.

## Docker Image Copy

First configure registry aliases:

```yaml
docker:
  registries:
    it: source-registry.example.com
    t: mirror-registry.example.com
```

List aliases:

```bash
leo docker list
```

Copy images:

```bash
leo docker copy it/apps/example-service:v1.2.4 t
leo docker copy it/apps/example-service:v1.2.4 t/library/example-service:latest
leo docker copy registry.example.com/app:v1 mirror.example.com/app:v1
```

Print the command without running it:

```bash
leo docker copy python:3.12 t --dry
```

Select a platform:

```bash
leo docker copy python:3.12-slim t --platform linux/arm64
leo docker copy python:3.12-slim t --platform linux/arm64/v8
```

`docker copy` calls [skopeo](https://github.com/containers/skopeo). It does not use the local Docker daemon or `docker context`. The default platform is `linux/amd64`.

## Log Viewer

Configure each project and its allowed log directories:

```yaml
proj:
  demo_01:
    logs:
      - runtime/logs
      - /docker-runtime
```

From the project root or any child directory, run:

```bash
leo log
```

`leo` walks upward from the current directory, finds the nearest ancestor whose name contains the project key, starts a temporary HTTP server, and prints a one-time URL. Relative log directories are resolved from that detected project root. An alias can use a different match string:

```yaml
proj:
  mc:
    match: demo_01
    logs:
      - runtime/logs
```

Useful overrides:

```bash
leo log --project mc
leo log --host 0.0.0.0 --port 9031
```

The default listener is `127.0.0.1` on an automatically selected port. On a remote server, open the printed URL manually through SSH port forwarding, or explicitly bind to an internal-network address. Plain HTTP on a non-loopback address is only appropriate on a trusted development network.

The workspace discovers supported text logs recursively, streams bounded on-demand searches, and follows active files with rotation/truncation notices. It reads files in place: it does not copy log contents, build a persistent index, or store queries, tokens, sessions, or UI state. Stop the server with Ctrl-C.

## Configuration

Default config path:

```text
~/.config/leo-cli/config.yaml
```

If the file does not exist, `leo` creates:

```yaml
repo:
  roots:
    - ~/work
time:
  zones:
    - +9
    - +0
    - America/Los_Angeles
```

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
time:
  zones:
    - +9
    - +0
    - America/Los_Angeles
proj:
  demo_01:
    logs:
      - runtime/logs
```

`time.zones` controls the extra common timezone rows printed by `leo time`. It accepts UTC offsets like `+9` and IANA names like `America/Los_Angeles`.

Paths support:

- `~` for the current user's home directory.
- Environment variables such as `$HOME`.
- Relative paths, which are resolved to absolute paths.

Missing or unreadable repository roots are printed as warnings. `repo reindex` only fails when all configured roots are unusable.

Default data path:

```text
~/.local/share/leo-cli/leo-cli.sqlite3
```

`XDG_CONFIG_HOME` and `XDG_DATA_HOME` override the default config and data locations when set.

The repository index uses SQLite with WAL. Rows are upserted by absolute repository path. Stale repositories that no longer exist on disk are not removed automatically yet.

## Commands

| Command | Description |
| --- | --- |
| `leo --version` / `leo version` | Print version and commit metadata |
| `leo repo reindex` | Scan configured repository roots and update the index |
| `leo repo` | Open the interactive repository picker and print the selected path |
| `leo shell init zsh` | Print zsh integration |
| `leo shell init bash` | Print bash integration |
| `leo join [FILE]` | Build SQL `IN` values from clipboard, txt, or csv |
| `leo time [VALUE]` | Convert current time, timestamps, and common time strings |
| `leo docker list` | Print Docker registry aliases |
| `leo docker copy SOURCE DESTINATION` | Copy an image with `skopeo copy` |
| `leo log` | Open the current project's temporary log workspace |

## Development

Run from source:

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

Version metadata comes from `.env`:

```text
version=v0.0.9
```

Create a tag and bump the patch version:

```bash
make release
```

Use an explicit version:

```bash
make release V=v0.1.0
```

Build cross-platform binaries, push the tag, and publish a GitHub release with GitHub CLI:

```bash
make release-github
make release-github V=v0.1.0
```

This requires an installed and authenticated [GitHub CLI](https://cli.github.com/).

## Project Layout

```text
cmd/                 Cobra command wiring
internal/config/     YAML config, default paths, path expansion
internal/dockercopy/ Docker image reference and registry alias resolution
internal/logview/    Safe log discovery, parsing, search, and follow
internal/logweb/     Authenticated HTTP APIs and embedded browser workspace
internal/project/    Current-project matching and root resolution
internal/refresh/    Initial index and background metadata refresh
internal/repoindex/  Git repository scanning and metadata extraction
internal/repoui/     Bubble Tea repository picker
internal/shellinit/  Shell integration script generation
internal/store/      SQLite storage and migrations
internal/termio/     Terminal input/output compatibility layer
internal/version/    Build-time version metadata
scripts/             Release helper scripts
```
