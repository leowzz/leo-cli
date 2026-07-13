.DEFAULT_GOAL := build

ENV_FILE ?= .env
BIN ?= leo
FFMPEG ?= ffmpeg
VERSION := $(shell grep -E '^version=' $(ENV_FILE) 2>/dev/null | head -1 | cut -d= -f2-)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)
ifeq ($(VERSION),)
VERSION := dev
endif
ifeq ($(COMMIT),)
COMMIT := unknown
endif
VERSION_LDFLAGS := -X github.com/leo/leo-cli/internal/version.Value=$(VERSION) -X github.com/leo/leo-cli/internal/version.CommandNameValue=$(BIN) -X github.com/leo/leo-cli/internal/version.CommitValue=$(COMMIT)

.PHONY: dev test build release release-github docs-dev docs-build docs-demos

dev:
	go run .

test:
	go test ./...

build:
	mkdir -p bin
	go build -ldflags "$(VERSION_LDFLAGS)" -o bin/$(BIN) .

release:
	ENV_FILE="$(ENV_FILE)" V="$(V)" scripts/release.sh

release-github:
	ENV_FILE="$(ENV_FILE)" BIN="$(BIN)" V="$(V)" scripts/release-github.sh

docs-dev:
	go run ./tools/docsgen
	pnpm --dir site dev

docs-build:
	go run ./tools/docsgen
	pnpm --dir site test
	pnpm --dir site build

docs-demos: build
	command -v vhs >/dev/null
	command -v $(FFMPEG) >/dev/null
	vhs site/vhs/repo-picker.tape
	$(FFMPEG) -y -i site/public/demos/repo-picker.tmp.webm -an -c:v libwebp_anim -preset text -quality 85 -loop 0 site/public/demos/repo-picker.webp
	rm -f site/public/demos/repo-picker.tmp.webm
	vhs site/vhs/join.tape
	$(FFMPEG) -y -i site/public/demos/join.tmp.webm -an -c:v libwebp_anim -preset text -quality 85 -loop 0 site/public/demos/join.webp
	rm -f site/public/demos/join.tmp.webm
