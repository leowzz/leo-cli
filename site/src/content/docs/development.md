---
title: 开发与发布
description: 从源码运行、测试、构建并发布 leo-cli。
---

## 常用命令

从源码运行：

```bash
make dev
```

测试和构建：

```bash
make test
make build
```

`make build` 将带版本信息的二进制写入 `bin/leo`。版本默认从 `.env` 的 `version=` 读取。

## 发布

创建 tag 并递增 patch 版本，或显式指定版本：

```bash
make release
make release V=v0.1.0
```

构建多平台产物、推送 tag 并发布 GitHub Release：

```bash
make release-github
make release-github V=v0.1.0
```

`release-github` 需要已安装并登录 [GitHub CLI](https://cli.github.com/)。

## 代码布局

```text
cmd/                 Cobra command wiring
internal/config/     YAML config, default paths, path expansion
internal/dockercopy/ Docker image reference and registry alias resolution
internal/logview/    Safe log discovery, parsing, search, and follow
internal/logweb/     Authenticated HTTP APIs and embedded browser workspace
internal/project/    Current-project matching and root resolution
internal/repoindex/  Git repository scanning and metadata extraction
internal/repoui/     Bubble Tea repository picker
internal/store/      SQLite storage and migrations
scripts/             Release helper scripts
```
