# Zero-Configuration Log Command Design

**Date:** 2026-07-13

## Summary

Make `leo log` useful on first run without requiring a `proj:` configuration.
Configured projects remain the preferred and strict path, while unmatched
directories fall back to a bounded, safe log-directory discovery mode.

Add `--logs <path>` as an explicit zero-configuration override. The command
continues to launch the same temporary authenticated browser workspace after a
project root and log roots have been selected.

## Resolution Order

The command resolves its runtime in this order:

1. **Explicit configured project:** `leo log --project <name>` uses the
   existing configured-project resolution. Unknown names, a current directory
   outside the configured project, invalid configured roots, and ambiguous
   matches remain errors. Automatic discovery never hides an explicit project
   mistake.
2. **Explicit log roots:** one or more `--logs <path>` flags create an
   ad-hoc project. `--logs` and `--project` are mutually exclusive.
3. **Implicit configured project:** plain `leo log` first tries the existing
   configured-project match.
4. **Automatic discovery:** when no configured project matches, including when
   `proj:` is absent or empty, the command derives an ad-hoc project and
   discovers log roots.

Only a genuine no-match result enters automatic discovery. Ambiguous configured
matches and other resolution errors remain visible.

## Ad-Hoc Project Root

For explicit log roots and automatic discovery, determine the project root by
walking from the invocation directory toward the filesystem root:

- the nearest directory containing a `.git` directory or `.git` file is the
  project root;
- if no Git marker exists, the invocation directory itself is the project
  root.

The ad-hoc project name is the project-root basename. Startup output appends
`(auto)` to the project label so users can tell that no persisted project
configuration was used.

Automatic discovery is ephemeral. It does not edit the user's configuration
file.

## Explicit `--logs`

`--logs` is a repeatable flag:

```bash
leo log --logs ./runtime/logs
leo log --logs ./api/logs --logs /var/log/my-service
```

Relative paths are resolved from the invocation directory, matching normal
shell expectations. Home and environment-variable expansion use the existing
configuration path-expansion behavior. Absolute paths remain absolute.

Each supplied path must resolve to a usable directory under the existing log
catalog safety rules. Duplicate paths are removed after cleaning and canonical
resolution. If no supplied directory is usable or no eligible log files are
found, the command returns an actionable error rather than opening an empty
workspace.

## Automatic Directory Discovery

Automatic discovery has two bounded phases.

### Conventional Paths

First check these project-root-relative directories:

1. `runtime/logs`
2. `logs`
3. `log`
4. `var/log`
5. `storage/logs`

All existing conventional directories are selected and deduplicated. If at
least one exists, bounded traversal is not needed.

### Bounded Traversal

If no conventional directory exists, walk at most four directory levels below
the project root and select directories whose basename is exactly `log` or
`logs`, case-insensitively.

Skip hidden directories and these common dependency/build trees:

- `.git`
- `node_modules`
- `vendor`
- `dist`
- `build`
- `target`
- `.venv`
- `venv`
- `__pycache__`

When a log directory is selected, do not descend into it looking for nested log
roots. Directory results are cleaned, deduplicated, and sorted for deterministic
startup output.

Discovery never scans the entire project as a log root. This prevents source
files, dependencies, and large build trees from becoming catalog candidates.

## Catalog Validation

The discovered or explicit roots are passed through the existing
`logview.BuildCatalog` safety checks, including symlink, file-type, path, and
binary-prefix validation.

A zero-file catalog is treated as a startup failure because the current catalog
is static and an empty workspace cannot begin following files that appear
later. The error reports the project root and attempted log roots.

Warnings for individual unusable roots are retained and included in the final
error or normal startup output as appropriate.

## Friendly Errors

When automatic discovery finds no usable logs, return a concise error with:

- the root that was inspected;
- the conventional paths and bounded `log` / `logs` search that were tried;
- an immediately executable override:
  `leo log --logs ./path/to/logs`;
- a minimal persistent configuration example using the detected directory
  basename.

Example shape:

```text
no log files found for /work/my-service
tried common log directories and log/logs folders up to depth 4
run: leo log --logs ./path/to/logs
or add:
  proj:
    my-service:
      logs:
        - runtime/logs
```

Explicit `--logs` errors name the rejected paths and suggest correcting the
flag. They do not suggest `--project` unless the user supplied it.

## Component Boundaries

### `internal/project`

Expose a typed or sentinel no-match error from configured-project resolution so
the command can distinguish safe fallback from ambiguity and invalid explicit
selection. Preserve the current human-readable error text for existing callers
and tests.

Add a small project-root helper for nearest-Git-root-or-current-directory
selection. It performs filesystem inspection only and does not invoke the
`git` executable.

### `cmd/log.go`

Own command policy:

- register and validate the repeatable `--logs` flag;
- enforce `--project` / `--logs` mutual exclusion;
- choose configured, explicit, or automatic runtime mode;
- call bounded directory discovery;
- reject empty catalogs;
- format startup and actionable failure messages.

### `internal/logview`

Keep catalog discovery and file safety unchanged. The command supplies selected
root directories; the catalog remains responsible for eligible log files and
safe opening.

## Startup Output

Configured mode keeps the current output:

```text
Project: configured-name
Root: /work/project
```

Ad-hoc mode prints:

```text
Project: project-directory (auto)
Root: /work/project-directory
```

The `Logs:`, warnings, and bootstrap URL sections remain unchanged.

## Verification

Automated tests cover:

- no `proj:` configuration with a conventional `runtime/logs` directory;
- configured-project match still winning over automatic discovery;
- no configured match falling back to discovery;
- explicit `--project` remaining strict;
- `--project` and `--logs` mutual exclusion;
- repeated relative and absolute `--logs` roots;
- nearest Git root and non-Git current-directory fallback;
- all existing conventional directories being selected;
- bounded case-insensitive `log` / `logs` discovery;
- depth limit and skipped dependency/build trees;
- deterministic deduplication and sorting;
- empty catalogs producing actionable errors; and
- configured versus `(auto)` startup output.

Integration verification runs `leo log` in an isolated unconfigured Git
project, opens the bootstrap URL, lists discovered files, performs a search,
starts Follow, appends a record, and shuts the server down cleanly.
