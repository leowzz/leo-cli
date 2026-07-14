# leo-cli

[中文](https://github.com/leowzz/leo-cli/blob/main/README.zh.md) | English

`leo-cli` builds a `leo` binary for personal command-line workflows:

- Build SQL `IN` values from clipboard, txt, or csv input and copy the result.
- Index local Git repositories and pick one from an interactive terminal list.
- Generate a shell helper so `repo` can jump into the selected repository.
- Convert Unix timestamps and common date-time strings across timezones.
- Copy container images with registry aliases through `skopeo copy`.
- Search and follow project logs in a temporary browser workspace.

## Install

Download the binary for your platform from [GitHub Releases](https://github.com/leowzz/leo-cli/releases), rename it to `leo` (`leo.exe` on Windows), and put its directory on `PATH`.

You can also build from source:

```bash
make build
```

The output is `bin/leo`.

## Quick Start

```bash
leo join
```

## Documentation

Read the full English documentation at <https://leowzz.github.io/leo-cli/en/>.

## Development

```bash
make dev
make test
make build
```
