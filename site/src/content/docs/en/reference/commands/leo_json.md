---
title: "leo json"
description: "Command reference generated from the leo CLI command tree."
---

> This page is generated from the Cobra command tree.

## leo json

Repair and format JSON, escaped JSON, or Python dicts

### Synopsis

Repair JSON-like payloads and print standard JSON.

Input is taken from PAYLOAD, piped stdin, or the clipboard, in that order.
Handles single quotes, Python True/False/None, comments, missing punctuation,
truncated JSON, Markdown code fences, log prefixes, and nested string encoding.
Only top-level encoded payloads are unwrapped; object field strings stay strings.
On a terminal, select JSON strings to parse deeper in an interactive viewer.
Piped output, --plain, and --compact print JSON directly; --interactive forces the viewer.
Repair is heuristic and does not execute Python or JavaScript code.

```
leo json [PAYLOAD] [flags]
```

### Examples

```
  leo json
  leo json "{'name': 'Leo', 'active': True, 'value': None}"
  cat payload.txt | leo json
  leo json --copy
  leo json --compact
  leo json --plain
  leo json --interactive > expanded.json
```

### Options

```
  -c, --compact       Print JSON on one line
      --copy          Also copy the result to the clipboard
  -h, --help          help for json
  -i, --interactive   Open the viewer even when stdout is redirected
      --plain         Print JSON without the interactive viewer
```

### Options inherited from parent commands

```
  -v, --version   Print version
```

### SEE ALSO

* [leo](../leo/)	 - Personal command-line tools

