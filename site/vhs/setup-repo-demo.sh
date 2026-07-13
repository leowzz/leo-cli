#!/usr/bin/env bash
set -euo pipefail

root=/tmp/leo-cli-vhs
rm -rf "$root"
mkdir -p "$root/config/leo-cli" "$root/data" "$root/repos/atlas/.git" "$root/repos/beacon/.git" "$root/repos/cascade/.git"
printf 'repo:\n  roots:\n    - %s\n' "$root/repos" > "$root/config/leo-cli/config.yaml"
XDG_CONFIG_HOME="$root/config" XDG_DATA_HOME="$root/data" bin/leo repo reindex >/dev/null
echo ready
