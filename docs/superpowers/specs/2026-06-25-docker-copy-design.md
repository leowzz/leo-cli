# Docker Copy Command Design

## Goal

Add `leo docker copy` for copying container images between registries with short registry aliases.

The command should support the two workflows:

- `leo docker copy it/apps/example-service:v1.2.4 t`
- `leo docker copy it/apps/example-service:v1.2.4 t/library/example-service:latest`

It also supports listing configured registry aliases:

- `leo docker list`

Additional workflows:

- `leo docker copy python:3.12 t` — single-segment Docker Hub source
- `leo docker copy registry.example.com/example-user/python:3.12 t` — full source reference without aliases
- `leo docker copy python:3.12 registry.example.com/example-user/python:3.12` — full destination without aliases

## Configuration

Registry aliases live in the existing YAML config:

```yaml
docker:
  registries:
    it: source-registry.example.com
    t: mirror-registry.example.com
```

The existing default config remains minimal and does not create registry aliases automatically.

## Command Behavior

`docker copy` accepts exactly two arguments:

```text
leo docker copy SOURCE DESTINATION
```

Skopeo talks to registries directly. It does not use the local Docker daemon or `docker context`.

### Source resolution

Source is resolved in two steps:

1. If the first path segment matches a configured alias, expand it as `ALIAS/REPOSITORY[:TAG]`.
2. Otherwise treat the value as a full image reference. Do not fail with an unknown-alias error.

Alias source format:

```text
ALIAS/REPOSITORY[:TAG]
```

Full source examples:

```text
python:3.12
docker.io/library/python:3.12
registry.example.com/namespace/app:v1
```

Single-segment Docker Hub images such as `python:3.12` are normalized to
`docker.io/library/python:3.12` for the skopeo source. Destination reuse still
uses the short repository name `python:3.12`.

### Destination resolution

Destination format:

```text
REGISTRY_OR_ALIAS[/REPOSITORY[:TAG]]
```

Both the registry segment and optional repository segment may be either a configured alias or a literal registry domain.

When the destination has only a registry alias or domain, the command reuses the source repository path and tag. When the destination includes a repository path, it uses that explicit target path and optional tag.

Examples:

| Source | Destination | Resolved destination repository |
| --- | --- | --- |
| `python:3.12` | `t` | `python:3.12` |
| `registry.example.com/example-user/python:3.12` | `t` | `example-user/python:3.12` |
| `it/apps/example-service:v1.2.4` | `t` | `apps/example-service:v1.2.4` |

### Platform selection

`--platform` selects which manifest to copy from multi-arch images. Default is `linux/amd64`.

Format:

```text
OS/ARCH
OS/ARCH/VARIANT
```

The command maps this to skopeo:

```bash
skopeo copy \
  --override-os <OS> \
  --override-arch <ARCH> \
  [--override-variant <VARIANT>] \
  docker://<resolved-source> \
  docker://<resolved-destination>
```

On macOS, skopeo otherwise selects `darwin/<host-arch>` from manifest lists. Official container images are Linux-only, so explicit platform selection is required for those sources.

Platform override is always explicit via `--platform`; the command does not silently change platform based on host OS.

### Dry run

With `--dry`, the command prints the rendered `skopeo copy ...` command to stdout and does not invoke `skopeo`.

### List aliases

`docker list` reads `docker.registries` and prints aliases sorted by name:

```text
it	source-registry.example.com
t	mirror-registry.example.com
```

If no aliases are configured, it prints `No Docker registries configured.`

## Package Boundaries

- `internal/dockercopy`: parses source/destination references, resolves aliases, normalizes Docker Hub library images, and builds the final source and destination references.
- `cmd/docker_copy.go`: wires the `docker` parent command, `copy`, and `list`.
- `cmd/docker_copy.go`: loads config, parses `--platform`, and invokes `skopeo` for `docker copy`.
- `cmd/docker_copy.go`: also owns the `--dry` and `--platform` flags and command rendering for dry runs.
- `internal/config`: loads `docker.registries` alongside existing repo config.

## Error Handling

- Unknown destination aliases fail with a message naming the missing `docker.registries.<alias>` key.
- Unknown source aliases do not fail when the source can be parsed as a full image reference.
- Malformed source references fail before invoking `skopeo`.
- Invalid `--platform` values fail before invoking `skopeo`.
- `skopeo` errors are returned directly so authentication, network, or registry failures remain visible.
- `--dry` still validates aliases, image reference shape, and platform before printing.

## Tests

Focused tests cover:

- alias destination reusing the source repository and tag
- explicit destination repository and tag
- literal registry domains
- unmatched source alias treated as full image reference
- single-segment Docker Hub source normalization
- full source reference preserved for destination reuse
- unknown destination aliases
- command runner arguments passed to `skopeo`, including platform override flags
- `--dry` output and the guarantee that the external runner is not called
- `docker list` sorted output and empty-config message
