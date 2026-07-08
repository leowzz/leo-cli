# leo-cli

Chinese is the default documentation: [README.md](README.md).

`leo-cli` builds a `leo` binary for personal command-line workflows:

- Index local Git repositories and pick one from an interactive terminal list.
- Generate a shell helper so `repo` can jump into the selected repository.
- Build SQL `IN` values from clipboard, txt, or csv input and copy the result.
- Copy container images with registry aliases through `skopeo copy`.

## Install

Download a binary from [GitHub Releases](https://github.com/leowzz/leo-cli/releases).

Example asset names:

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

Or build locally:

```bash
make build
```

The binary is written to `bin/leo`.

## Quick Start

Index repositories:

```bash
leo repo reindex
```

Open the picker:

```bash
leo repo
```

Enable shell jumping:

```bash
eval "$(leo shell init zsh)"
eval "$(leo shell init bash)"
```

Then run:

```bash
repo
```

Build SQL `IN` values:

```bash
leo join
leo join ids.txt
leo join ids.csv
```

Copy images:

```bash
leo docker list
leo docker copy it/apps/example-service:v1.2.4 t
leo docker copy python:3.12 t --dry
```

`leo docker copy` requires [skopeo](https://github.com/containers/skopeo). It does not use the local Docker daemon.

## Configuration

Default config path:

```text
~/.config/leo-cli/config.yaml
```

Default config:

```yaml
repo:
  roots:
    - ~/work
```

Example:

```yaml
repo:
  roots:
    - ~/work
    - ~/repo
docker:
  registries:
    it: source-registry.example.com
    t: mirror-registry.example.com
```

Default data path:

```text
~/.local/share/leo-cli/leo-cli.sqlite3
```

`XDG_CONFIG_HOME` and `XDG_DATA_HOME` override those defaults when set.

## Commands

| Command | Description |
| --- | --- |
| `leo --version` / `leo version` | Print version and commit metadata |
| `leo repo reindex` | Scan configured repository roots |
| `leo repo` | Open the repository picker |
| `leo shell init zsh` | Print zsh integration |
| `leo shell init bash` | Print bash integration |
| `leo join [FILE]` | Build SQL `IN` values |
| `leo docker list` | List registry aliases |
| `leo docker copy SOURCE DESTINATION` | Copy an image with `skopeo` |

## Development

```bash
make dev
make test
make build
make release
make release-github
```
