---
title: 搜索和跟随日志
description: 从项目目录发现日志并启动临时浏览器搜索工作台。
---

## 零配置启动

在项目根目录或任意子目录运行：

```bash
leo log
```

`leo` 优先使用匹配的 `proj` 配置。没有匹配项时，它使用最近的 Git 根目录（非 Git 目录使用当前目录），先检查 `runtime/logs`、`logs`、`log`、`var/log` 和 `storage/logs`，再有界搜索四层以内名为 `log` 或 `logs` 的目录。

## 显式选择日志目录

`--logs` 可重复使用，相对路径从运行命令的目录解析：

```bash
leo log --logs ./custom/logs
leo log --logs ./api/logs --logs ./worker/logs
```

`--project NAME` 严格选择已配置项目。未知项目、目录不匹配或无有效日志都会报错，不会退回自动发现。`--project` 与 `--logs` 不能同时使用。

## 网络边界

默认监听 `127.0.0.1` 和系统选择的可用端口。远程使用时可通过 SSH 端口转发打开打印的 URL，或显式设置 `--host` 与 `--port`：

```bash
leo log --host 0.0.0.0 --port 9031
```

非 loopback 地址上的明文 HTTP 只适合可信开发网络。工作台使用一次性启动 token 换取内存 session；按 Ctrl-C 会停止服务。

## 搜索和跟随

工作台递归发现受支持的文本日志，按需流式搜索，并可跟随活动文件。搜索支持时间范围、文本或正则、包含/排除词和结构化字段；文件轮转、截断和删除会显示提示。日志始终原地读取，不会复制内容、建立持久索引或保存查询、token、session 和 UI 状态。
