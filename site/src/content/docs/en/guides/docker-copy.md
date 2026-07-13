---
title: Copy Docker Images
description: Use registry aliases and skopeo to copy images between registries.
---

## Configure Registry Aliases

```yaml
docker:
  registries:
    it: source-registry.example.com
    t: mirror-registry.example.com
```

Run `leo docker list` to inspect configured aliases.

## Copy An Image

The source and destination may use aliases or full image references:

```bash
leo docker copy it/apps/example-service:v1.2.4 t
leo docker copy it/apps/example-service:v1.2.4 t/library/example-service:latest
leo docker copy registry.example.com/app:v1 mirror.example.com/app:v1
```

When the destination is only an alias, it retains the source image path and tag.

## Inspect The Command And Platform

`--dry` only prints the `skopeo` command that would run:

```bash
leo docker copy python:3.12 t --dry
```

The default platform is `linux/amd64`. Pass either `OS/ARCH` or `OS/ARCH/VARIANT`:

```bash
leo docker copy python:3.12-slim t --platform linux/arm64
leo docker copy python:3.12-slim t --platform linux/arm64/v8
```

This command calls `skopeo copy`; it does not use the local Docker daemon or `docker context`. A working `skopeo` executable must already be installed for a real copy, and `skopeo` owns authentication and network access.
