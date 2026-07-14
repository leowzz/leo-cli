# Repo Picker Demo Fixture Design

## Goal

Make the `leo repo` VHS demonstration slow enough to read and populate its existing metadata row with realistic branch and commit data.

## Fixture Data

`site/vhs/setup-repo-demo.sh` creates three real local Git repositories so `leo repo reindex` exercises the production metadata reader:

| Repository | Branch | Last commit |
| --- | --- | --- |
| `atlas` | `main` | `2026-07-10 09:30` |
| `beacon` | `feat/search` | `2026-07-13 16:20` |
| `cascade` | `release/v1.8.0` | `2026-07-08 11:00` |

Each repository gets one deterministic empty commit using author and committer timestamps with the `+0800` offset, so the UI displays the exact table values in Asia/Shanghai. The demo does not seed SQLite or fake terminal output.

## Timing

Both existing `Sleep 1s` commands in `site/vhs/repo-picker.tape` become `Sleep 2s`: one after filtering for `bea`, and one after selecting `beacon`.

## Verification

- A shell-level fixture check runs the setup script with the demo environment and confirms all three branch names and commit timestamps through Git.
- Regenerate only `repo-picker.webp`; preserve its 1000x600 canvas, infinite loop, and `#161616` animation background.
- Inspect the browser preview to confirm the branch, commit time, and path are readable and that the slower timing is visible.
