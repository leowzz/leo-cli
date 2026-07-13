---
title: Configuration Fields
description: See every field in the leo-cli YAML configuration and what it controls.
---

The default configuration file is `~/.config/leo-cli/config.yaml`. Complete example:

```yaml
repo:
  roots:
    - ~/work
    - $HOME/src
docker:
  registries:
    it: source-registry.example.com
    t: mirror-registry.example.com
time:
  zones:
    - +9
    - +0
    - America/Los_Angeles
proj:
  mc:
    match: demo_01
    logs:
      - runtime/logs
      - /docker-runtime
```

## `repo`

`repo.roots` lists directories scanned by `leo repo reindex`. Each value expands home, environment variables, and relative paths.

## `docker`

`docker.registries` maps short aliases to registry hostnames. `leo docker copy` can use these aliases in source or destination references.

## `time`

`time.zones` lists extra timezone rows printed after the primary `leo time` result. Each entry may be a UTC offset such as `+9` or an IANA timezone name.

## `proj`

Each `proj` key is a project name and the default directory match string. Optional `match` overrides that directory match string; `logs` lists log directories resolved from the project root. `leo log --project NAME` strictly selects by project key.
