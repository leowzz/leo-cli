# Docker Copy Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `leo docker copy SOURCE DESTINATION` to copy Docker images between configured registry aliases.

**Architecture:** Keep parsing and alias resolution in `internal/dockercopy` so it is easy to test without running external commands. Keep Cobra wiring and `skopeo` execution in `cmd/docker_copy.go`, using an injectable runner for command tests.

**Tech Stack:** Go, Cobra, YAML v3, `skopeo` as an external runtime dependency.

---

## File Structure

- `internal/config/config.go`: add `DockerConfig` with `docker.registries`.
- `internal/config/config_test.go`: verify YAML loads registry aliases.
- `internal/dockercopy/dockercopy.go`: parse references and resolve registry aliases.
- `internal/dockercopy/dockercopy_test.go`: test source and destination resolution.
- `cmd/docker_copy.go`: add the `docker` parent command, `copy`, `list`, and `skopeo copy` runner.
- `cmd/docker_copy_test.go`: test `skopeo` command arguments with a fake runner, dry-run output, and registry alias listing.
- `scripts/release-github.sh`: build cross-platform binaries and publish GitHub releases with `gh`.
- `README.md`: document requirements, configuration, command usage, install, and release workflow.

## Task 1: Config Shape

- [x] Write a failing config test that loads:

```yaml
docker:
  registries:
    it: source-registry.example.com
    t: mirror-registry.example.com
```

- [x] Verify it fails because `Config` has no `Docker` field.
- [x] Add `DockerConfig` to `internal/config/config.go`.
- [x] Rerun `go test ./internal/config`.

## Task 2: Reference Resolution

- [x] Write failing tests for:

```text
it/apps/example-service:v1.2.4 + t
it/apps/example-service:v1.2.4 + t/library/example-service:latest
registry.example.com/apps/example-service:v1.2.4 + mirror.example.com
unknown alias "missing"
```

- [x] Verify they fail because `Resolve` is missing.
- [x] Implement `internal/dockercopy.Resolve`.
- [x] Rerun `go test ./internal/dockercopy`.

## Task 3: Cobra Command

- [x] Write a failing command test that injects a fake runner and expects:

```text
skopeo copy docker://source-registry.example.com/apps/example-service:v1.2.4 docker://mirror-registry.example.com/library/example-service:latest
```

- [x] Verify it fails because `runDockerCopy` is missing.
- [x] Add `cmd/docker_copy.go`.
- [x] Rerun `go test ./cmd`.

## Task 4: Documentation and Verification

- [x] Update README with `skopeo`, `docker.registries`, and command examples.
- [x] Run `gofmt -w cmd internal`.
- [x] Run `go test ./...`.
- [x] Run `go run . docker-copy --help`.

## Task 5: Dry Run

- [x] Write a failing command test for `--dry` that expects this stdout and no runner call:

```text
skopeo copy docker://source-registry.example.com/apps/example-service:v1.2.4 docker://mirror-registry.example.com/apps/example-service:v1.2.4
```

- [x] Verify it fails because `runDockerCopy` has no dry-run behavior.
- [x] Add the `--dry` flag and dry-run command rendering.
- [x] Run `gofmt -w cmd internal`.
- [x] Run `go test ./...`.
- [x] Run `go run . docker-copy --help`.

## Task 6: Docker Subcommands

- [x] Write failing command tests for `docker list` sorted output and empty-config output.
- [x] Verify they fail because `runDockerList` is missing.
- [x] Move the user-facing copy command from `leo docker-copy` to `leo docker copy`.
- [x] Add `leo docker list`.
- [x] Update README and design docs to use `leo docker copy` and document `leo docker list`.
- [x] Run `gofmt -w cmd internal`.
- [x] Run `go test ./...`.
- [x] Run `go run . docker copy --help`.
- [x] Run `go run . docker list` with a temporary config.

## Task 7: Full Image Sources and Platform Flag

- [x] Write failing tests for unmatched source aliases treated as full image references.
- [x] Write failing tests for single-segment Docker Hub sources such as `python:3.12`.
- [x] Write failing tests for full registry sources reused on destination-only aliases.
- [x] Implement two-step source parsing and Docker Hub `docker.io/library/` normalization in `internal/dockercopy`.
- [x] Add `--platform` to `leo docker copy` with default `linux/amd64`.
- [x] Map `--platform` to skopeo override flags in `cmd/docker_copy.go`.
- [x] Update README and design docs.
- [x] Run `go test ./...`.

## Task 8: GitHub Release Publishing

- [x] Add `scripts/release-github.sh` to build cross-platform binaries and publish with `gh release create`.
- [x] Add `make release-github`.
- [x] Upload raw binaries named `leo-<os>-<arch>` instead of tarballs.
- [x] Document install and release workflow in README.
