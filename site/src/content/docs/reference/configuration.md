---
title: 配置字段
description: 查看 leo-cli YAML 配置中的全部字段及其含义。
---

默认配置文件是 `~/.config/leo-cli/config.yaml`。完整示例：

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
clipboard:
  base_url: https://clipboard.example.com
  token: replace-with-a-long-random-token
  search_interval: 100ms
proj:
  mc:
    match: demo_01
    logs:
      - runtime/logs
      - /docker-runtime
```

## `repo`

`repo.roots` 是 `leo repo reindex` 扫描的目录列表。每个值都会展开 home、环境变量和相对路径。

## `docker`

`docker.registries` 将短 alias 映射到 registry hostname。`leo docker copy` 可以在源或目标引用中使用这些 alias。

## `time`

`time.zones` 是 `leo time` 主要结果之后额外打印的时区列表。每项可以是 `+9` 形式的 UTC offset 或 IANA 时区名。

## `clipboard`

`clipboard.base_url` 是 Maccy server 的地址，`leo clip` 会请求其 `/v1/entries` 接口。`clipboard.token` 是服务端 `auth` 中对应的 bearer secret。`clipboard.search_interval` 控制 TUI 输入停止后自动搜索的防抖间隔，默认是 `100ms`。建议使用足够长的随机 token，并限制配置文件权限。

## `proj`

`proj` 的每个 key 是项目名和默认目录匹配字符串。可选的 `match` 可覆盖目录匹配字符串；`logs` 列出从项目根解析的日志目录。`leo log --project NAME` 使用项目 key 严格选择配置。
