# leo-cli

中文 | [English](https://github.com/leowzz/leo-cli/blob/main/README.md)

`leo-cli` 是一组个人命令行工具，构建产物是 `leo`。当前主要包括：

- 扫描本地 Git 仓库，并用交互式终端列表快速选择仓库。
- 生成 shell 函数，让 `repo` 选择仓库后直接 `cd` 进去。
- 从剪贴板、txt 或 csv 构造 SQL `IN` 列表，并复制结果。
- 转换 Unix 时间戳和常见日期时间字符串，并输出多时区结果。
- 用 registry alias 简化 `skopeo copy` 的镜像搬运命令。
- 在临时浏览器工作台中搜索和跟随项目日志。

## 安装

从 [GitHub Releases](https://github.com/leowzz/leo-cli/releases) 下载对应平台的二进制文件。文件名使用发布 tag：

```text
leo-TAG-darwin-arm64
leo-TAG-linux-amd64
leo-TAG-windows-amd64.exe
```

macOS / Linux:

```bash
chmod +x leo-TAG-darwin-arm64
mv leo-TAG-darwin-arm64 ~/bin/leo
```

Windows:

```powershell
ren leo-TAG-windows-amd64.exe leo.exe
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

从 stdin 管道读取：

```bash
seq 1 10 | leo join
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

不传文件时，stdin 管道优先于剪贴板。默认按首次出现顺序去重。支持的输出包括逗号列表、括号列表、`field in (...)` 和带引号列表。

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

## 日志查看器

在项目根目录或任意子目录运行：

```bash
leo log
```

无需提前配置项目。`leo` 会使用最近的 Git 根目录（非 Git 目录中使用当前目录），先检查 `runtime/logs`、`logs`、`log`、`var/log` 和 `storage/logs` 等常见路径；没有命中时，再有界搜索名为 `log` 或 `logs` 的目录。

日志位于其他位置时，可以显式传入一个或多个目录：

```bash
leo log --logs ./custom/logs
leo log --logs ./api/logs --logs ./worker/logs
```

相对 `--logs` 路径从执行命令时的目录解析。需要持久化项目别名或自定义根目录时，再添加配置：

```yaml
proj:
  demo_01:
    logs:
      - runtime/logs
      - /docker-runtime
```

当当前目录命中已配置项目时，配置优先。`leo` 会向上查找目录名包含项目 key 的最近祖先，并从该根目录解析相对日志目录。别名可以使用不同的匹配字符串：

```yaml
proj:
  mc:
    match: demo_01
    logs:
      - runtime/logs
```

常用覆盖参数：

```bash
leo log --project mc
leo log --host 0.0.0.0 --port 9031
```

显式 `--project` 会严格匹配配置，且不能与 `--logs` 同时使用。

默认只监听 `127.0.0.1`，端口由系统自动选择。在远程服务器上，可以通过 SSH 端口转发手动打开打印出的 URL；也可以显式绑定内网地址。非 loopback 地址上的明文 HTTP 只适合可信开发网络。

工作台会递归发现受支持的文本日志，以有界的按需扫描流式返回搜索结果，并在文件轮转或截断时继续跟随和显示提示。时间范围支持最近 1、5、10 分钟以及更长区间；选择下拉项或点击当前范围按钮都会立即搜索。Clear 会保留全部筛选条件、清空表格，并把点击时刻作为下一次搜索的起点，直到重新应用普通时间范围。新日志显示在顶部；向下查看旧日志时会保持当前阅读位置并累计新日志数量。

日志始终原地读取：不会复制日志内容，不会建立持久索引，也不会保存查询、token、session 或 UI 状态。按 Ctrl-C 停止服务。

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
proj:
  demo_01:
    logs:
      - runtime/logs
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
| `leo completion SHELL` | 打印 bash、fish、PowerShell 或 zsh 补全脚本 |
| `leo join [FILE]` | 从剪贴板、txt 或 csv 构造 SQL `IN` 值 |
| `leo time [VALUE]` | 转换当前时间、时间戳和常见时间字符串 |
| `leo docker list` | 打印 Docker registry alias |
| `leo docker copy SOURCE DESTINATION` | 用 `skopeo copy` 复制镜像 |
| `leo log` | 打开当前项目的临时日志工作台 |

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
version=0.1.0
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
internal/logview/    安全日志发现、解析、搜索和跟随
internal/logweb/     认证 HTTP API 和内嵌浏览器工作台
internal/project/    当前项目匹配和根目录解析
internal/refresh/    初始索引和后台元数据刷新
internal/repoindex/  Git 仓库扫描和元数据提取
internal/repoui/     Bubble Tea 仓库选择器
internal/shellinit/  shell 集成脚本生成
internal/store/      SQLite 存储和迁移
internal/termio/     终端输入输出兼容层
internal/version/    构建期版本信息
scripts/             release 辅助脚本
```
