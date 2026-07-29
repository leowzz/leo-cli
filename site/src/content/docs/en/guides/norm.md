---
title: Normalize Clipboard Code
description: Convert Chinese and full-width punctuation in clipboard code to ASCII.
---

Copy code, then run:

```bash
leo norm
```

The command reads the entire clipboard, writes the normalized text back in place, and prints `已规范化剪贴板`. For example:

```text
（a＝１，b＝“value”）  ->  (a=1,b="value")
```

It converts full-width ASCII characters and spaces, Chinese commas, periods and enumeration commas, brackets, book-title marks, and curly quotes. Other characters are preserved.

`leo norm` processes every position in the clipboard, including strings and comments. It does not parse programming languages, change indentation, or invoke formatters such as gofmt or Prettier.
