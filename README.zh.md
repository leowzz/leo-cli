# leo-cli

中文 | [English](https://github.com/leowzz/leo-cli/blob/main/README.en.md)

`leo-cli` 是一组个人命令行工具，构建产物是 `leo`。当前主要做四件事：

- 扫描本地 Git 仓库，并用交互式终端列表快速选择仓库。
- 生成 shell 函数，让 `repo` 选择仓库后直接 `cd` 进去。
- 从剪贴板、txt 或 csv 构造 SQL `IN` 列表，并复制结果。
- 用 registry alias 简化 `skopeo copy` 的镜像搬运命令。

## 安装

从 [GitHub Releases](https://github.com/leowzz/leo-cli/releases) 下载对应平台的二进制文件。文件名形如：

```text
leo-v0.0.9-darwin-arm64
leo-v0.0.9-linux-amd64
leo-v0.0.9-windows-amd64.exe
```

macOS / Linux:

```bash
chmod +x leo-v0.0.9-darwin-arm64
mv leo-v0.0.9-darwin-arm64 ~/bin/leo
```

Windows:

```powershell
ren leo-v0.0.9-windows-amd64.exe leo.exe
```

把 `leo` 或 `leo.exe` 所在目录加入 `PATH` 后即可使用。

也可以本地构建：

```bash
make build
```

构建产物在：

```text
bin/leo
```

## 快速开始

先建立仓库索引：

```bash
leo repo reindex
```

打开仓库选择器：

```bash
leo repo
```

输入关键字过滤，按 Enter 输出选中仓库路径，按 Esc 或 Ctrl-C 取消。

启用 shell 跳转函数：

```bash
eval "$(leo shell init zsh)"
```

bash:

```bash
eval "$(leo shell init bash)"
```

把对应 `eval` 加到 `~/.zshrc` 或 `~/.bashrc` 后，就可以运行：

```bash
repo
```

`repo` 会打开仓库选择器，并在选中后执行 `cd`。

## SQL IN 助手

从剪贴板读取：

```bash
leo join
```

从文件读取：

```bash
leo join ids.txt
leo join ids.csv
```

交互界面里：

- 左/右：切换 csv 字段。
- 上/下：切换输出格式。
- u：切换唯一/原始值。
- Enter：复制当前结果。
- Esc：取消。

默认按首次出现顺序去重。支持的输出包括逗号列表、括号列表、`field in (...)` 和带引号列表。

## 时间转换

支持 Unix 秒、Unix 毫秒和常见日期时间字符串：

```bash
leo time
leo time 1783512043
leo time 1783512043000
leo time "(2026-07-08 20:00:43)"
```

不传值时，`leo time` 使用当前时间。

没有显式时区的日期时间字符串默认按 UTC+8 解析。用 `--to` 指定输出时区：

```bash
leo time 1783512043 --to +9
leo time "2026-07-08 20:00:43" --to +9
leo time 1783512043 --to Asia/Tokyo
```

配置里的 `time.zones` 会作为额外常用时区行打印，支持 UTC offset 和 IANA 时区名。

## Docker 镜像复制

先在配置里写 registry alias：

```yaml
docker:
  registries:
    it: source-registry.example.com
    t: mirror-registry.example.com
```

查看 alias：

```bash
leo docker list
```

复制镜像：

```bash
leo docker copy it/apps/example-service:v1.2.4 t
leo docker copy it/apps/example-service:v1.2.4 t/library/example-service:latest
leo docker copy registry.example.com/app:v1 mirror.example.com/app:v1
```

只打印将要执行的命令：

```bash
leo docker copy python:3.12 t --dry
```

指定平台：

```bash
leo docker copy python:3.12-slim t --platform linux/arm64
leo docker copy python:3.12-slim t --platform linux/arm64/v8
```

`docker copy` 调用的是 [skopeo](https://github.com/containers/skopeo)，不依赖本机 Docker daemon 或 `docker context`。默认平台是 `linux/amd64`。

## 配置

默认配置文件：

```text
~/.config/leo-cli/config.yaml
```

如果文件不存在，`leo` 会创建：

```yaml
repo:
  roots:
    - ~/work
time:
  zones:
    - +9
    - +0
    - America/Los_Angeles
```

示例：

```yaml
repo:
  roots:
    - ~/work
    - ~/repo
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
```

`time.zones` 控制 `leo time` 额外打印的常用时区。支持 `+9` 这类 UTC offset，也支持 `America/Los_Angeles` 这类 IANA 时区名。

路径支持：

- `~`，表示当前用户 home。
- 环境变量，例如 `$HOME`。
- 相对路径，最终会转成绝对路径。

缺失或无法读取的仓库根目录会以 warning 打印。只有所有根目录都不可用时，`repo reindex` 才会失败。

默认数据文件：

```text
~/.local/share/leo-cli/leo-cli.sqlite3
```

如果设置了 `XDG_CONFIG_HOME` 或 `XDG_DATA_HOME`，会优先使用对应目录。

仓库索引使用 SQLite + WAL。记录按仓库绝对路径 upsert，目前不会自动删除磁盘上已经不存在的旧仓库记录。

## 命令速查

| 命令 | 作用 |
| --- | --- |
| `leo --version` / `leo version` | 打印版本和 commit 信息 |
| `leo repo reindex` | 扫描配置的仓库根目录并更新索引 |
| `leo repo` | 打开交互式仓库选择器，选中后输出绝对路径 |
| `leo shell init zsh` | 打印 zsh 集成脚本 |
| `leo shell init bash` | 打印 bash 集成脚本 |
| `leo join [FILE]` | 从剪贴板、txt 或 csv 构造 SQL `IN` 值 |
| `leo time [VALUE]` | 转换当前时间、时间戳和常见时间字符串 |
| `leo docker list` | 打印 Docker registry alias |
| `leo docker copy SOURCE DESTINATION` | 用 `skopeo copy` 复制镜像 |

## 开发

从源码运行：

```bash
make dev
```

测试：

```bash
make test
```

构建：

```bash
make build
```

版本号来自 `.env`：

```text
version=v0.0.9
```

打 tag 并递增 patch 版本：

```bash
make release
```

指定版本：

```bash
make release V=v0.1.0
```

构建多平台二进制、推送 tag，并用 GitHub CLI 发布 release：

```bash
make release-github
make release-github V=v0.1.0
```

需要已安装并登录 [GitHub CLI](https://cli.github.com/)。

## 项目结构

```text
cmd/                 Cobra 命令入口
internal/config/     YAML 配置、默认路径、路径展开
internal/dockercopy/ Docker 镜像引用和 registry alias 解析
internal/refresh/    初始索引和后台元数据刷新
internal/repoindex/  Git 仓库扫描和元数据提取
internal/repoui/     Bubble Tea 仓库选择器
internal/shellinit/  shell 集成脚本生成
internal/store/      SQLite 存储和迁移
internal/termio/     终端输入输出兼容层
internal/version/    构建期版本信息
scripts/             release 辅助脚本
```
