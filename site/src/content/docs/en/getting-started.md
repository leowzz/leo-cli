---
title: Getting Started
description: Install leo, create a repository index, and enable shell repository jumps.
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

## Create The First Index

The default configuration scans `~/work`. Create the local repository index, then open the picker:

```bash
leo repo reindex
leo repo
```

Type to filter repositories. Press Enter to print the selected repository's absolute path, or Esc or Ctrl-C to cancel.

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
