---
title: How It Works
description: Understand repository indexing, log security boundaries, and the local data privacy model.
---

## Repository Index

`leo repo reindex` scans configured roots, identifies Git repositories, and writes absolute paths and Git metadata to local SQLite. The picker reads from this index, so it does not rescan every directory each time it opens. Rows are updated by absolute path; stale rows for repositories removed from disk are not currently deleted automatically.

## Log Security Boundary

The log viewer builds a catalog only from resolved log roots and revalidates the allowed boundary on every read. It skips hidden files, symbolic links, compressed files, binary content, and unsupported suffixes. Search has file-size, result-count, concurrency, and duration limits; follow mode recognizes truncation, replacement, deletion, and rotation.

The HTTP service binds only to loopback by default. The browser first uses a one-time bootstrap token, then an in-memory session cookie; state disappears when the service exits. Binding to a non-loopback address exposes the temporary workspace to the network and is only suitable for a trusted development network.

## Local Storage And Privacy

Repository metadata stays in local SQLite. Original files remain the only source of log content: the viewer does not copy logs, build a persistent full-text index, or save queries, tokens, sessions, or UI state. `join` writes to the system clipboard only after the user confirms a result.
