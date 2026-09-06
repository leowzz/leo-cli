---
title: Parse and Repair JSON
description: Read escaped JSON, Python dicts, and malformed payloads from the clipboard, arguments, or stdin and output standard JSON.
---

Copy a payload, then run:

```bash
leo json
```

By default, the command reads the clipboard and opens an interactive terminal view of JSON with two-space indentation. You can also pass a payload directly or use piped or redirected input:

```bash
leo json "{'name': 'Leo', 'active': True, 'value': None}"
cat payload.txt | leo json
leo json < payload.txt
```

Input precedence is: command argument, stdin, then clipboard. An explicitly empty argument or pipe returns an error instead of falling back to the clipboard.

The Python dict above produces:

```json
{
  "active": true,
  "name": "Leo",
  "value": null
}
```

Common repairs include single quotes, unquoted keys, `True` / `False` / `None`, trailing commas, comments, missing commas or colons, incomplete brackets or quotes, Markdown code fences, and newline-delimited JSON. Multiple JSON documents become an array.

Outer JSON strings and Python string representations are automatically unwrapped, including multiple encoding layers. Each input below produces `{"ok": true}`:

```text
{'ok': True}
"{\"ok\": true}"
'{"ok": true}'
{\"ok\": true}
```

Only the outer payload is unwrapped. Strings inside object fields remain strings. Large integers and decimals never pass through floating-point conversion, and Unicode escapes are displayed as readable characters.

## Log Extraction and Deeper Parsing

Paste a log with its prefix intact:

```text
im_service.send_message send message to im: 123, {"Content":"{\"text\":\"system message\"}","Ext":{"mc:ext_json":"{\"message_type\":1}"}}
```

The command extracts the object. Initially, both `Content` and `Ext["mc:ext_json"]` remain strings. Expandable string paths appear below the JSON view. The footer shows clickable key bindings with Chinese action labels and wraps on narrow terminals:

- Up / Down or Tab: select a string field.
- Enter: parse only the selected string into an object or array. Deeper strings retain their types.
- `u`: undo the most recent parse and restore the original string.
- PageUp / PageDown or the mouse wheel: scroll JSON. Left / Right: scroll long lines horizontally.
- `c` / `C`: copy the current JSON, including manually parsed fields. A confirmation appears on success.
- `q`: finish viewing and print the current JSON to stdout.
- Esc / Ctrl+C: exit without printing or automatically copying the result.

New JSON strings become selectable after each expansion. Ordinary text and numeric strings are not expansion targets.

JSON-like strings that cannot be repaired remain selectable with an `[invalid]` marker. Enter displays the parse error and preserves the original string. Repair can remove an extra `]` or `}` with no matching opener. Brackets inside strings remain unchanged, and objects are not automatically converted to arrays.

## Output Options

```bash
leo json --compact
leo json --plain
leo json --copy
leo json --compact --copy
leo json --interactive > expanded.json
```

`--compact` (`-c`) prints one-line JSON directly. `--plain` prints indented JSON directly. `--copy` also copies the result after the viewer finishes; clipboard writes require an explicit copy action.

Piped or redirected output skips the viewer by default. `--interactive` (`-i`) forces the viewer and writes the current JSON to the redirected destination when done. The viewer uses a separate terminal, so stdout contains only JSON:

```bash
leo json > repaired.json
```

Repair uses the Go `jsonrepair` library, requires no Python installation, and never executes input as code. It infers intent from syntax, cannot guarantee recovery of the original meaning, and is not a complete Python expression parser. Empty or unrecoverable input exits with a nonzero status. A parse failure never overwrites the clipboard.
