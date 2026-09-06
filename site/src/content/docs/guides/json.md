---
title: 解析和修复 JSON
description: 从剪贴板、命令参数或管道读取转义 JSON、Python dict 和非标准 Payload，输出标准 JSON。
---

复制 Payload 后运行：

```bash
leo json
```

默认读取剪贴板，在终端中以两空格缩进展示 JSON，并打开交互视图。也可以直接传入 Payload，或从管道和文件重定向读取：

```bash
leo json "{'name': 'Leo', 'active': True, 'value': None}"
cat payload.txt | leo json
leo json < payload.txt
```

输入优先级为：命令参数、标准输入、剪贴板。显式提供的空参数或空管道会报错，不会回退读取剪贴板。

上面的 Python dict 会输出：

```json
{
  "active": true,
  "name": "Leo",
  "value": null
}
```

支持常见的单引号、未加引号的键、`True` / `False` / `None`、尾逗号、注释、缺失逗号或冒号、不完整的括号或引号、Markdown 代码块及换行分隔的 JSON。连续多条 JSON 会合并成数组。

外层 JSON 字符串或 Python 打印出的字符串表示会自动解包，包括多层编码。例如以下内容都会得到 `{"ok": true}`：

```text
{'ok': True}
"{\"ok\": true}"
'{"ok": true}'
{\"ok\": true}
```

只解包整个 Payload 的外层字符串，字段中的字符串不会自动改成对象。大整数和小数不会经过浮点数转换；Unicode 转义会显示为可读字符。

## 日志提取与逐层解析

可以直接粘贴带前缀的日志，无需先手动截取 JSON：

```text
im_service.send_message send message to im: 123, {"Content":"{\"text\":\"系统消息\"}","Ext":{"mc:ext_json":"{\"message_type\":1}"}}
```

命令提取其中的对象，初始视图中 `Content` 和 `Ext["mc:ext_json"]` 都保持字符串。视图下方列出可继续解析的字段路径：

- `↑` / `↓` 或 `Tab`：选择字符串字段。
- `Enter` 或点击 `enter 解析`：只将选中的字符串解析成对象或数组，更深层的字符串仍保持原类型。
- `u` 或点击 `u 撤销`：撤销最近一次解析，恢复原字符串。
- `PageUp` / `PageDown` 或鼠标滚轮：滚动 JSON；`←` / `→`：横向查看长行。
- `c` / `C` 或点击 `c 复制`：复制当前 JSON，包括已经手动解析的字段，成功后显示“已复制到剪贴板”。
- `q` 或点击 `q 完成`：结束查看，将当前 JSON 输出到标准输出。
- `Esc` / `Ctrl+C` 或点击 `esc 取消`：取消退出，不输出或自动复制结果。

底部常驻显示按键和中文动作提示，窄终端会自动换行。

解析一层后，新出现的 JSON 字符串会继续列在选择区。普通文本和数字字符串不列为展开目标。

疑似 JSON 但无法修复的字段也会列出，标记为 `[invalid]`，按 `Enter` 显示失败原因并保留原字符串。对于没有对应开括号的多余 `]` 或 `}`，修复会尝试移除它们；字符串内容中的括号保持不变，也不会因此将对象自动改成数组。

## 输出选项

```bash
leo json --compact
leo json --plain
leo json --copy
leo json --compact --copy
leo json --interactive > expanded.json
```

`--compact`（`-c`）直接输出单行 JSON。`--plain` 直接输出带缩进的 JSON。`--copy` 在完成查看后同时将结果写入剪贴板；只有显式复制才会更改剪贴板。

输出连接到管道或文件时，默认跳过交互视图；`--interactive`（`-i`）可强制打开视图，完成后再将当前 JSON 写入重定向目标。视图使用独立终端，标准输出仅包含 JSON：

```bash
leo json > repaired.json
```

修复使用 Go 的 `jsonrepair` 库，无需安装 Python，也不会执行输入中的代码。它根据语法推测意图，无法保证恢复损坏数据的原始含义，也不等同于完整的 Python 表达式解析器。空输入或无法修复的输入以非零状态退出，解析失败时不会覆盖剪贴板。
