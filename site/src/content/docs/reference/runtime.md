---
title: 文件、数据与环境变量
description: 定位 leo-cli 的配置、SQLite 数据和路径展开规则。
---

## 配置文件

默认路径：

```text
~/.config/leo-cli/config.yaml
```

设置 `XDG_CONFIG_HOME` 后，路径变为 `$XDG_CONFIG_HOME/leo-cli/config.yaml`。文件不存在时，命令会创建包含 `repo.roots` 和 `time.zones` 的最小配置。

## 数据文件

仓库索引默认位于：

```text
~/.local/share/leo-cli/leo-cli.sqlite3
```

设置 `XDG_DATA_HOME` 后，路径变为 `$XDG_DATA_HOME/leo-cli/leo-cli.sqlite3`。SQLite 使用 WAL、`synchronous = NORMAL` 和 5 秒 busy timeout。仓库按绝对路径 upsert；磁盘上已删除的旧仓库目前不会自动从索引移除。

## 路径展开

配置路径支持 `$VARIABLE` 环境变量和开头的 `~`。相对路径会根据命令运行时的工作目录转换为绝对路径，并经过清理。日志配置中的相对路径随后按所选项目根目录解析；显式 `--logs` 相对路径按命令运行目录解析。
