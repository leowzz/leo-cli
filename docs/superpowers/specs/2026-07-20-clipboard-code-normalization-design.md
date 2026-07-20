# Clipboard Code Normalization Design

## Goal

Add `leo norm`, a no-argument command that reads the entire clipboard, converts
Chinese and full-width punctuation to ASCII equivalents, and writes the result
back to the clipboard.

## Behavior

- Read the clipboard with the existing `github.com/atotto/clipboard` dependency.
- Convert every occurrence, including punctuation inside strings and comments.
- Convert full-width ASCII `U+FF01` through `U+FF5E` to `U+0021` through
  `U+007E`, and convert the ideographic space `U+3000` to an ASCII space.
- Convert common CJK punctuation not covered by that range:
  - `U+3001` to `,` and `U+3002` to `.`
  - `U+3008`, `U+300A` to `<`; `U+3009`, `U+300B` to `>`
  - `U+3010`, `U+3014` to `[`; `U+3011`, `U+3015` to `]`
  - `U+2018`, `U+2019` to `'`; `U+201C`, `U+201D` to `"`
- Preserve all other characters byte-for-byte.
- Write the normalized text to the clipboard and print a short success message.
- Return clipboard read or write errors unchanged.

## Implementation

Add one Cobra command file under `cmd/`. Keep normalization as a pure function
and inject clipboard read/write functions into the command runner so behavior is
testable without touching the user's clipboard. Use a rune loop and no new
dependencies.

## Verification

One focused test file will prove the mapping, preservation of ordinary text, and
the read-normalize-write flow. The full Go test suite and `git diff --check` must
also pass.

## Deliberate Limits

`leo norm` does not parse programming languages, format whitespace, or invoke
language formatters. Those are separate behaviors and are not required for
clipboard punctuation normalization.
