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
```

`base_url` 不要包含 `/v1/entries`；命令会自动补上接口路径。token 应与 Maccy server 配置中的 `auth` secret 一致。

## 搜索和复制

不传查询词时显示最近的剪贴板记录：

```bash
leo clip
```

也可以传入包含搜索或使用模糊搜索：

```bash
leo clip kubernetes
leo clip kubernets --fuzzy
leo cb kubernetes
```

选择记录后按 Enter 复制，Esc 或 Ctrl-C 取消。`--limit`（或 `-n`）控制加载的最大条目数，范围是 1 到 200，默认 50。
