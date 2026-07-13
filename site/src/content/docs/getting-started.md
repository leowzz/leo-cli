---
title: 开始使用
description: 安装 leo、建立仓库索引并启用 shell 仓库跳转。
---

## 安装

从 [GitHub Releases](https://github.com/leowzz/leo-cli/releases) 下载对应平台的二进制文件，把它重命名为 `leo`（Windows 为 `leo.exe`），并将所在目录加入 `PATH`。

macOS / Linux：

```bash
chmod +x leo-TAG-darwin-arm64
mv leo-TAG-darwin-arm64 ~/bin/leo
```

也可以从源码构建：

```bash
make build
```

产物位于 `bin/leo`。

## 第一次建立索引

默认配置会扫描 `~/work`。先建立本地仓库索引，再打开选择器：

```bash
leo repo reindex
leo repo
```

输入文字过滤仓库，按 Enter 输出所选仓库的绝对路径；按 Esc 或 Ctrl-C 取消。

## 启用 shell 跳转

`leo shell init` 会把集成脚本打印到标准输出。用 `eval` 在当前 shell 中定义 `repo` 函数：

```bash
eval "$(leo shell init zsh)"
```

bash 用户运行：

```bash
eval "$(leo shell init bash)"
```

把对应命令加入 `~/.zshrc` 或 `~/.bashrc`，之后运行 `repo` 即可选择仓库并切换目录。继续阅读[快速切换仓库](./guides/repositories/)或查看[命令参考](./reference/commands/leo/)。
