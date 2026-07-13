---
title: Navigate Repositories
description: Index local Git repositories and change directory through an interactive picker.
---

## Refresh The Index

List the directories to scan under `repo.roots` in the configuration file, then run:

```bash
leo repo reindex
```

The scan updates the SQLite index. Unreadable roots produce warnings; the command fails only when every root is unusable.

## Use The Picker

```bash
leo repo
```

Press `/` to enter filter mode and type a query. Press Up/Down to apply the filter, then use Up/Down or j/k to move through results. Press Enter to accept the current item. Esc clears an active filter first and quits when no filter is active; Ctrl-C always cancels. When accepted, `leo repo` only prints the selected repository's absolute path to standard output; it cannot change its parent shell's working directory.

## Change Directory In The Shell

```bash
eval "$(leo shell init zsh)"
repo
```

`leo shell init zsh` and `leo shell init bash` print a small `repo` function. The function captures `leo repo` output and runs `cd` only when the path is non-empty. Add the `eval` line to the matching shell startup file to enable it permanently.
