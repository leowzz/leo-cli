.DEFAULT_GOAL := build

ENV_FILE ?= .env
BIN ?= leo
VERSION := $(shell grep -E '^version=' $(ENV_FILE) 2>/dev/null | head -1 | cut -d= -f2-)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)
ifeq ($(VERSION),)
VERSION := dev
endif
ifeq ($(COMMIT),)
COMMIT := unknown
endif
VERSION_LDFLAGS := -X github.com/leo/leo-cli/internal/version.Value=$(VERSION) -X github.com/leo/leo-cli/internal/version.CommandNameValue=$(BIN) -X github.com/leo/leo-cli/internal/version.CommitValue=$(COMMIT)

.PHONY: dev test build release release-github

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
