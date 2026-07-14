---
title: Getting Started
description: Install leo, build your first SQL IN list, then optionally enable repository jumps.
---

## Install

Download the binary for your platform from [GitHub Releases](https://github.com/leowzz/leo-cli/releases), rename it to `leo` (`leo.exe` on Windows), and put its directory on `PATH`.

macOS / Linux:

```bash
chmod +x leo-TAG-darwin-arm64
mv leo-TAG-darwin-arm64 ~/bin/leo
```

You can also build from source:

```bash
make build
```

The output is `bin/leo`.

## Build Your First SQL IN List

Copy values separated by commas or whitespace, then run:

```bash
leo join
```

You can also pipe values directly. Piped input takes priority over the clipboard:

```bash
seq 1 10 | leo join
```

Choose an output format and press Enter to copy the result. See [Build SQL IN Values](./guides/join/) for more input methods and interactive keys.

## Create The First Index

The default configuration scans `~/work`. Create the local repository index, then open the picker:

```bash
leo repo reindex
leo repo
```

Press `/` to enter filter mode and type a query. Press Up/Down to apply the filter, then use Up/Down or j/k to move through results. Press Enter to print the selected repository's absolute path. Esc clears an active filter first and quits when no filter is active; Ctrl-C always cancels.

## Enable Shell Jumps

`leo shell init` prints its integration script to standard output. Use `eval` to define the `repo` function in the current shell:

```bash
eval "$(leo shell init zsh)"
```

For bash:

```bash
eval "$(leo shell init bash)"
```

Add the matching command to `~/.zshrc` or `~/.bashrc`. You can then run `repo` to select a repository and change directory. Continue with [Navigate Repositories](./guides/repositories/) or browse the [command reference](./reference/commands/leo/).
