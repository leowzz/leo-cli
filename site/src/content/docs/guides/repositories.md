---
title: 快速切换仓库
description: 建立本地 Git 仓库索引并通过交互选择器切换目录。
---

## 刷新索引

在配置文件的 `repo.roots` 中列出要扫描的目录，然后运行：

```bash
leo repo reindex
```

扫描会更新 SQLite 索引。无法读取的根目录会产生 warning；只有所有根目录都不可用时命令才失败。

## 使用选择器

```bash
leo repo
```

输入文字过滤仓库。按 Enter 接受当前项目，按 Esc 或 Ctrl-C 取消。接受后，`leo repo` 只向标准输出打印所选仓库的绝对路径；它本身不能改变父 shell 的工作目录。

## 在 shell 中切换目录

```bash
eval "$(leo shell init zsh)"
repo
```

`leo shell init zsh` 和 `leo shell init bash` 打印一个小型 `repo` 函数。该函数捕获 `leo repo` 的输出，并只在路径非空时执行 `cd`。把 `eval` 行加入对应 shell 的启动文件可永久启用它。
