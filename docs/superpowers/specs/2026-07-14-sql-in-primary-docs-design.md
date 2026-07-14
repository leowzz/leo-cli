# SQL IN Primary Documentation Design

**Date:** 2026-07-14

## Goal

Make SQL `IN` construction the first feature users see and try, while keeping
the rest of `leo-cli` documented as supporting tools.

## Changes

- Put SQL `IN` construction first in the Chinese and English README feature
  lists and quick-start examples.
- Lead both documentation home pages with `leo join`, using the existing join
  demo instead of the repository picker demo.
- Make `leo join` the first runnable workflow in both Getting Started pages;
  keep repository indexing and shell integration as later sections.
- Put the SQL `IN` guide first in the Guides sidebar and make homepage guide
  links point to it.
- Keep Chinese and English content structurally aligned.

## Non-goals

- No CLI behavior changes.
- No new demo assets or components.
- No removal of repository, time, Docker, or log documentation.

## Verification

- Search the source documentation for stale repository-first quick starts.
- Run the existing documentation build to check locale parity, links, and MDX.
- Review the final diff for matching Chinese and English hierarchy.
