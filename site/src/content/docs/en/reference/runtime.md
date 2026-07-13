---
title: Files, Data, And Environment
description: Locate leo-cli configuration, SQLite data, and path-expansion behavior.
---

## Configuration File

Default path:

```text
~/.config/leo-cli/config.yaml
```

With `XDG_CONFIG_HOME` set, the path becomes `$XDG_CONFIG_HOME/leo-cli/config.yaml`. When the file is absent, the command creates a minimal configuration containing `repo.roots` and `time.zones`.

## Data File

The repository index defaults to:

```text
~/.local/share/leo-cli/leo-cli.sqlite3
```

With `XDG_DATA_HOME` set, the path becomes `$XDG_DATA_HOME/leo-cli/leo-cli.sqlite3`. SQLite uses WAL, `synchronous = NORMAL`, and a five-second busy timeout. Repositories are upserted by absolute path; stale repositories removed from disk are not currently removed from the index automatically.

## Path Expansion

Configuration paths support `$VARIABLE` environment variables and a leading `~`. Relative paths are converted to absolute paths from the command's working directory and cleaned. Relative log paths from configuration are then resolved from the selected project root; relative explicit `--logs` paths resolve from the invocation directory.
