# Repo Picker Demo Fixture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Slow the repo-picker demonstration and show realistic branch, last-commit, and path metadata read through the production repository indexer.

**Architecture:** The VHS setup script creates three minimal real Git repositories with deterministic branches and empty commits. The existing tape and CLI stay unchanged except for two longer pauses; only `repo-picker.webp` is regenerated.

**Tech Stack:** Bash, Git, VHS, ffmpeg-full/libwebp_anim, webpmux, Astro preview.

## Global Constraints

- Keep the repositories named `atlas`, `beacon`, and `cascade`.
- Use the production `leo repo reindex` path; do not seed SQLite or fake terminal output.
- Use branches `main`, `feat/search`, and `release/v1.8.0` respectively.
- Display last-commit times `2026-07-10 09:30`, `2026-07-13 16:20`, and `2026-07-08 11:00` in Asia/Shanghai.
- Change both existing repo-picker pauses from 1 second to 2 seconds.
- Regenerate only `site/public/demos/repo-picker.webp`.
- Preserve 1000x600 dimensions, loop count `0`, and animation background `#161616`.
- Do not change CLI behavior or add a project dependency.

---

### Task 1: Add Deterministic Git Metadata And Slower Timing

**Files:**
- Modify: `site/vhs/setup-repo-demo.sh`
- Modify: `site/vhs/repo-picker.tape`
- Modify: `site/public/demos/repo-picker.webp`

**Interfaces:**
- Consumes: `bin/leo`, Git, VHS, ffmpeg with `libwebp_anim`, and `webpmux`.
- Produces: three indexed repositories whose existing UI descriptions show `branch | last commit | path`, plus a slower animated WebP.

- [ ] **Step 1: Reproduce the missing metadata**

Run the current setup and query a branch through Git:

```bash
rtk site/vhs/setup-repo-demo.sh
rtk git -C /tmp/leo-cli-vhs/repos/atlas symbolic-ref --short HEAD
```

Expected: the setup prints `ready`, then Git fails because the fixture only creates an empty `.git` directory and has no branch or commit.

- [ ] **Step 2: Create real deterministic repositories**

Replace the directory setup in `site/vhs/setup-repo-demo.sh` with:

```bash
root=/tmp/leo-cli-vhs
rm -rf "$root"
mkdir -p "$root/config/leo-cli" "$root/data" "$root/repos"

create_repo() {
  name=$1
  branch=$2
  timestamp=$3
  repo="$root/repos/$name"

  git init -q --initial-branch="$branch" "$repo"
  GIT_AUTHOR_DATE="$timestamp" GIT_COMMITTER_DATE="$timestamp" \
    git -C "$repo" \
      -c user.name="Leo CLI VHS" \
      -c user.email="vhs@example.invalid" \
      commit --allow-empty -q -m "Initial demo commit"
}

create_repo atlas main "2026-07-10T09:30:00+08:00"
create_repo beacon feat/search "2026-07-13T16:20:00+08:00"
create_repo cascade release/v1.8.0 "2026-07-08T11:00:00+08:00"
```

Keep the existing config generation, `leo repo reindex`, and `ready` output below this block.

- [ ] **Step 3: Verify fixture metadata**

Run:

```bash
rtk site/vhs/setup-repo-demo.sh
rtk git -C /tmp/leo-cli-vhs/repos/atlas symbolic-ref --short HEAD
rtk proxy env TZ=Asia/Shanghai git -C /tmp/leo-cli-vhs/repos/atlas log -1 --format=%cd --date=format-local:%Y-%m-%dT%H:%M
rtk git -C /tmp/leo-cli-vhs/repos/beacon symbolic-ref --short HEAD
rtk proxy env TZ=Asia/Shanghai git -C /tmp/leo-cli-vhs/repos/beacon log -1 --format=%cd --date=format-local:%Y-%m-%dT%H:%M
rtk git -C /tmp/leo-cli-vhs/repos/cascade symbolic-ref --short HEAD
rtk proxy env TZ=Asia/Shanghai git -C /tmp/leo-cli-vhs/repos/cascade log -1 --format=%cd --date=format-local:%Y-%m-%dT%H:%M
```

Expected in order: `main`, `2026-07-10T09:30`, `feat/search`, `2026-07-13T16:20`, `release/v1.8.0`, and `2026-07-08T11:00`.

- [ ] **Step 4: Slow both readable states**

In `site/vhs/repo-picker.tape`, change exactly both occurrences:

```text
Sleep 1s
```

to:

```text
Sleep 2s
```

- [ ] **Step 5: Regenerate only the repo-picker animation**

Run:

```bash
rtk shasum -a 256 site/public/demos/join.webp
rtk make build
rtk vhs site/vhs/repo-picker.tape
rtk proxy /opt/homebrew/opt/ffmpeg-full/bin/ffmpeg -y -i site/public/demos/repo-picker.tmp.webm -an -c:v libwebp_anim -preset text -quality 85 -loop 0 site/public/demos/repo-picker.tmp.webp
rtk webpmux -set bgcolor 255,22,22,22 site/public/demos/repo-picker.tmp.webp -o site/public/demos/repo-picker.ready.webp
rtk mv site/public/demos/repo-picker.ready.webp site/public/demos/repo-picker.webp
rtk rm site/public/demos/repo-picker.tmp.webm site/public/demos/repo-picker.tmp.webp
```

Expected: `join.webp` remains byte-for-byte unchanged.

- [ ] **Step 6: Verify media and repository tests**

Run:

```bash
rtk shasum -a 256 site/public/demos/join.webp
rtk webpmux -info site/public/demos/repo-picker.webp
rtk go test ./...
rtk proxy env PATH=/opt/homebrew/opt/node@24/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin pnpm --dir site test
rtk proxy env PATH=/opt/homebrew/opt/node@24/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin pnpm --dir site build
rtk git diff --check
```

Expected: the `join.webp` hash matches Step 5; repo-picker is 1000x600 with multiple frames, loop `0`, and background `0xFF161616`; 161 Go tests and 4 Node tests pass; Starlight builds 57 pages.

- [ ] **Step 7: Inspect the real preview**

Start the production preview on an unused port and inspect `/leo-cli/` at desktop and mobile widths. Confirm:

- `beacon` visibly shows `feat/search | 2026-07-13 16:20 | /tmp/leo-cli-vhs/repos/beacon`.
- Both filtered and selected states remain visible long enough to read.
- No white flash, horizontal overflow, clipped metadata, or layout overlap appears.

- [ ] **Step 8: Commit**

```bash
rtk git add site/vhs/setup-repo-demo.sh site/vhs/repo-picker.tape site/public/demos/repo-picker.webp
rtk git commit -m "docs: slow and enrich repo picker demo"
```
