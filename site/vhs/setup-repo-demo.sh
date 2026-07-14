#!/usr/bin/env bash
set -euo pipefail

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

printf 'repo:\n  roots:\n    - %s\n' "$root/repos" > "$root/config/leo-cli/config.yaml"
XDG_CONFIG_HOME="$root/config" XDG_DATA_HOME="$root/data" bin/leo repo reindex >/dev/null
echo ready
