---
title: Search And Follow Logs
description: Discover logs from a project directory and start a temporary browser search workspace.
---

## Start Without Configuration

Run from a project root or any child directory:

```bash
leo log
```

`leo` first uses a matching `proj` configuration. Without a match, it uses the nearest Git root (or the current directory outside Git), checks `runtime/logs`, `logs`, `log`, `var/log`, and `storage/logs`, then performs a bounded search up to four levels deep for directories named `log` or `logs`.

## Select Log Directories Explicitly

`--logs` is repeatable, and relative paths resolve from the directory where the command runs:

```bash
leo log --logs ./custom/logs
leo log --logs ./api/logs --logs ./worker/logs
```

`--project NAME` strictly selects a configured project. An unknown project, directory mismatch, or lack of eligible logs is an error and never falls back to automatic discovery. `--project` and `--logs` cannot be combined.

## Network Boundary

The default listener is `127.0.0.1` on an available system-selected port. For remote use, open the printed URL through SSH port forwarding, or explicitly set `--host` and `--port`:

```bash
leo log --host 0.0.0.0 --port 9031
```

Plain HTTP on a non-loopback address is only appropriate on a trusted development network. The workspace exchanges a one-time bootstrap token for an in-memory session; Ctrl-C stops the service.

## Search And Follow

The workspace recursively discovers supported text logs, streams on-demand searches, and follows active files. Search supports time ranges, text or regular expressions, include/exclude terms, and structured fields; file rotation, truncation, and deletion produce visible notices. Logs are always read in place: the viewer does not copy content, build a persistent index, or save queries, tokens, sessions, or UI state.
