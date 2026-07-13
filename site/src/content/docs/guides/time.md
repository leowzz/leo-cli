---
title: 转换时间与时区
description: 转换 Unix 时间戳和常见日期字符串并查看多个时区。
---

## 支持的输入

`leo time` 接受 Unix 秒、Unix 毫秒和常见日期时间字符串。不传值时使用当前时间。

```bash
leo time
leo time 1783512043
leo time 1783512043000
leo time "(2026-07-08 20:00:43)"
```

纯数字达到 13 位时按毫秒解析，否则按秒解析。日期支持 `-` 或 `/` 分隔、可选秒数、RFC 3339 和带显式 offset 的形式。

## 输入与输出时区

没有显式时区的日期字符串默认按 UTC+8 解析。`--to` 控制主要输出时区，支持 UTC offset 或 IANA 名称：

```bash
leo time 1783512043 --to +9
leo time "2026-07-08 20:00:43" --to Asia/Tokyo
```

默认值为 `+8`。配置中的 `time.zones` 会追加常用时区行；与主要输出相同的时区不会重复打印。
