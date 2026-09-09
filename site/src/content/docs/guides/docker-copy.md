---
title: 搬运 Docker 镜像
description: 使用 registry alias 和 skopeo 在镜像仓库之间复制镜像。
---

## 配置 registry alias

```yaml
docker:
  registries:
    it: source-registry.example.com
    t: mirror-registry.example.com
```

运行 `leo docker list` 查看已配置的 alias。

## 复制镜像

源和目标可以使用 alias，也可以使用完整镜像引用：

```bash
leo docker copy it/apps/example-service:v1.2.4 t
leo docker copy it/apps/example-service:v1.2.4 t/library/example-service:latest
leo docker copy registry.example.com/app:v1 mirror.example.com/app:v1
```

目标只写 alias 时，默认保留源镜像的路径和 tag。配置该 alias 的默认命名空间后，会改为使用 `命名空间/源镜像最后一级名称:tag`：

```yaml
docker:
  registries:
    ali: registry.cn-heyuan.aliyuncs.com
  default_namespaces:
    ali: leo03w
```

```bash
leo docker copy cxbdasheng/dnet:v2.4.3 ali
# 等同于 leo docker copy cxbdasheng/dnet:v2.4.3 ali/leo03w/dnet:v2.4.3
```

显式指定目标路径时，不使用默认命名空间。

## 检查命令和平台

`--dry` 只打印将执行的 `skopeo` 命令：

```bash
leo docker copy python:3.12 t --dry
```

默认平台为 `linux/amd64`。可以传入 `OS/ARCH` 或 `OS/ARCH/VARIANT`：

```bash
leo docker copy python:3.12-slim t --platform linux/arm64
leo docker copy python:3.12-slim t --platform linux/arm64/v8
```

该命令调用 `skopeo copy`，不使用本机 Docker daemon 或 `docker context`。实际复制需要系统中已有可执行的 `skopeo`，认证和网络访问也由 `skopeo` 负责。
