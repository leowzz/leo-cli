---
title: 搜索剪贴板历史
description: 从 Maccy server 搜索剪贴板历史并复制选中的文本。
---

## 配置

在 `~/.config/leo-cli/config.yaml` 中加入 Maccy server 地址和 bearer token：

```yaml
clipboard:
  base_url: http://127.0.0.1:8080
  token: replace-with-a-long-random-token
  search_interval: 100ms
```

`base_url` 不要包含 `/v1/entries`；命令会自动补上接口路径。token 应与 Maccy server 配置中的 `auth` secret 一致。`search_interval` 是 TUI 输入停止后自动检索的防抖间隔，省略时默认为 `100ms`。

## 搜索和复制

不传查询词时显示最近的剪贴板记录；TUI 中后续搜索默认使用混合检索：

```bash
leo clip
```

传入查询词时默认使用基于 Zvec 的语义与全文混合搜索：

```bash
leo clip kubernetes
leo clip kubernets --fuzzy
leo clip "Go 本地向量数据库" --mix
leo cb -m "Go 本地向量数据库"
leo cb kubernetes --mix=false
```

`-m` / `--mix` 用于显式开启混合检索，默认已开启；`--mix=false` 从普通包含检索开始。混合检索需要 Maccy server 启用 Zvec，并且必须提供非空查询。`--mix` 与 `--fuzzy` 不能同时使用。

TUI 中按 `/` 输入新的远端查询；文本非空且停止输入达到 `search_interval` 后会自动检索，按 Enter 可立即检索并返回结果浏览。浏览结果时按 `m` 使用当前查询切换普通/混合检索。当前模式和查询显示在列表上方。选择记录后按 Enter 复制，Esc 或 Ctrl-C 取消。

`--limit`（或 `-n`）控制每次加载的最大条目数，范围是 1 到 200，默认 50。
