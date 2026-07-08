#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

ENV_FILE="${ENV_FILE:-.env}"
BIN="${BIN:-leo}"

require_env_file() {
  if [[ -f "$ENV_FILE" ]]; then
    return
  fi
  echo "release-github: missing $ENV_FILE"
  exit 1
}

if [[ -n "${V:-}" ]]; then
  if ! git rev-parse "$V" >/dev/null 2>&1; then
    require_env_file
    ENV_FILE="$ENV_FILE" V="$V" scripts/release.sh
  fi
  VERSION="${V//$'\r'/}"
else
  require_env_file
  ENV_FILE="$ENV_FILE" scripts/release.sh
  VERSION="$(grep -E '^version=' "$ENV_FILE" | head -1 | cut -d= -f2-)"
  VERSION="${VERSION//$'\r'/}"
fi

TAG="$VERSION"
COMMIT="$(git rev-parse --short HEAD)"
LDFLAGS="-X github.com/leo/leo-cli/internal/version.Value=${VERSION} -X github.com/leo/leo-cli/internal/version.CommandNameValue=${BIN} -X github.com/leo/leo-cli/internal/version.CommitValue=${COMMIT}"

rm -rf dist
mkdir -p dist

platforms=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
  "windows amd64"
  "windows arm64"
)

for entry in "${platforms[@]}"; do
  read -r goos goarch <<<"$entry"
  binary="${BIN}-${VERSION}-${goos}-${goarch}"
  if [[ "$goos" == "windows" ]]; then
    binary="${binary}.exe"
  fi
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o "dist/${binary}" .
done

if git ls-remote --exit-code --tags origin "refs/tags/${TAG}" >/dev/null 2>&1; then
  echo "release-github: remote tag $TAG already exists"
else
  git push origin "$TAG"
fi

if gh release view "$TAG" >/dev/null 2>&1; then
  gh release upload "$TAG" dist/"${BIN}"-* --clobber
  echo "release-github: uploaded assets to existing release $TAG"
else
  gh release create "$TAG" \
    --title "$TAG" \
    --generate-notes \
    dist/"${BIN}"-*
  echo "release-github: published $TAG"
fi

echo "release-github: https://github.com/$(gh repo view --json nameWithOwner -q .nameWithOwner)/releases/tag/${TAG}"
