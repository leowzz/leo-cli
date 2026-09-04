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

Without a query, the command shows the most recent clipboard entries:

```bash
leo clip
```

You can use contains, fuzzy, or Zvec-backed semantic and full-text hybrid search:

```bash
leo clip kubernetes
leo clip kubernets --fuzzy
leo clip "local Go vector database" --mix
leo cb -m "local Go vector database"
leo cb kubernetes
```

`-m` / `--mix` requires Zvec to be enabled on the Maccy server and a non-empty query. It performs semantic and full-text hybrid search and cannot be combined with `--fuzzy`.

Press Enter to copy the selected entry. Esc or Ctrl-C cancels. `--limit` (or `-n`) controls the maximum number of entries to load, from 1 to 200, with a default of 50.
