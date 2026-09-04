---
title: Search Clipboard History
description: Search Maccy server clipboard history and copy a selected entry.
---

## Configure

Add the Maccy server address and bearer token to `~/.config/leo-cli/config.yaml`:

```yaml
clipboard:
  base_url: http://127.0.0.1:8080
  token: replace-with-a-long-random-token
```

Do not include `/v1/entries` in `base_url`; the command appends the endpoint path. The token must match a secret in the Maccy server `auth` map.

## Search And Copy

Without a query, the command shows the most recent clipboard entries. Subsequent searches in the TUI use hybrid search by default:

```bash
leo clip
```

With a query, Zvec-backed semantic and full-text hybrid search is the default:

```bash
leo clip kubernetes
leo clip kubernets --fuzzy
leo clip "local Go vector database" --mix
leo cb -m "local Go vector database"
leo cb kubernetes --mix=false
```

`-m` / `--mix` explicitly enables hybrid search, which is already the default. Use `--mix=false` to start with regular contains search. Hybrid search requires Zvec to be enabled on the Maccy server and a non-empty query. `--mix` cannot be combined with `--fuzzy`.

In the TUI, press `/`, enter a new remote query, and press Enter to search. While browsing results, press `m` to rerun the current query in regular or hybrid mode. The current mode and query appear above the list. Press Enter on a result to copy it; Esc or Ctrl-C cancels.

`--limit` (or `-n`) controls the maximum number of entries loaded per search, from 1 to 200, with a default of 50.
