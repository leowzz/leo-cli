# leo-cli

中文 | [English](https://github.com/leowzz/leo-cli/blob/main/README.md)

`leo-cli` 是一组个人命令行工具，构建产物是 `leo`：

- 扫描本地 Git 仓库，并用交互式终端列表快速选择仓库。
- 生成 shell 函数，让 `repo` 选择仓库后直接 `cd` 进去。
- 从剪贴板、txt 或 csv 构造 SQL `IN` 列表，并复制结果。
- 转换 Unix 时间戳和常见日期时间字符串，并输出多时区结果。
- 用 registry alias 简化 `skopeo copy` 的镜像搬运命令。
- 在临时浏览器工作台中搜索和跟随项目日志。

## 安装

从 [GitHub Releases](https://github.com/leowzz/leo-cli/releases) 下载对应平台的二进制文件，把它重命名为 `leo`（Windows 为 `leo.exe`），并将所在目录加入 `PATH`。

也可以从源码构建：

```bash
make build
```

产物位于 `bin/leo`。

## 快速开始

```bash
leo repo reindex
leo repo
eval "$(leo shell init zsh)"
```

## 文档

完整中文文档：<https://leowzz.github.io/leo-cli/>。

## 开发

```bash
make dev
make test
make build
```
