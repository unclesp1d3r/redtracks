# RedTracks justfile
# Run `just` (or `just --list`) to see available recipes.
#
# Wails v3 desktop app: Go backend, React + TypeScript frontend on bun.
# The Wails build pipeline (bindings, frontend bundle, per-platform packaging,
# server and docker builds) lives in Taskfile.yml and build/*/Taskfile.yml.
# Those recipes delegate to `wails3 task` instead of duplicating it. Lint,
# format, and tests run directly here rather than through Task; the golangci
# hooks in .pre-commit-config.yaml call the same mise-pinned binary and
# config, and `just check` runs the full hook set.
#
# All dev tools are pinned in mise.toml (including just itself, so run
# `mise exec -- just ...` until mise is activated) and invoked through
# `mise exec` so recipes use the same versions everywhere. Recipes are bash:
# on Windows use Git Bash or WSL. See AGENTS.md.

set shell := ["bash", "-euo", "pipefail", "-c"]
set quiet
set ignore-comments

mise_exec := "mise exec --"

# golangci-lint is named on `mise exec` because a `go install`ed copy in GOBIN
# (the VS Code Go extension drops a v1 there) otherwise shadows the pin on PATH.
golangci := "mise exec golangci-lint -- golangci-lint"
wails := mise_exec + " wails3 task"

# Default & Help

[private]
default:
    {{ just_executable() }} --list --unsorted --justfile {{ justfile() }}

# main.go embeds all:frontend/dist, so go vet/test/lint need the directory to
# exist. A placeholder keeps a fresh clone (or `just clean`) working until the
# first real frontend build, which replaces it.
[private]
_dist:
    [ -n "$(ls -A frontend/dist 2>/dev/null)" ] || { mkdir -p frontend/dist && touch frontend/dist/.keep; }

# Setup

alias i := install
alias setup := install

# Install pinned tools (mise), git hooks, Go modules, and frontend deps (bun)
[group('setup')]
install:
    mise install
    {{ mise_exec }} pre-commit install
    {{ mise_exec }} go mod tidy
    cd frontend && {{ mise_exec }} bun install

# Install the pre-commit git hooks
[group('setup')]
install-hooks:
    {{ mise_exec }} pre-commit install

# Bump pinned tools, Go modules, frontend deps (bunfig release-age gate applies), and hook revs; Wails stays pinned
[group('setup')]
[no-quiet]
update-deps:
    #!/usr/bin/env bash
    # Wails is bumped only in a dedicated commit (AGENTS.md), so the module and
    # the wails3 CLI are held at their current versions here.
    set -euo pipefail
    wails_mod=github.com/wailsapp/wails/v3
    wails_ver="$(mise exec -- go list -m -f '{{{{.Version}}' "$wails_mod")"
    mise upgrade --bump --local --exclude "go:$wails_mod/cmd/wails3"
    mise exec -- go get -u ./...
    mise exec -- go get "$wails_mod@$wails_ver"
    mise exec -- go mod tidy
    (cd frontend && mise exec -- bun update)
    mise exec -- pre-commit autoupdate

# Remove build outputs, generated bindings, and test artifacts
[group('setup')]
clean:
    rm -rf bin frontend/dist frontend/bindings frontend/test-results cover.out

# Development

alias r := dev

# Run the app in development mode (native window, Vite hot reload)
[group('dev')]
dev:
    {{ wails }} dev

# Build and run the desktop app
[group('dev')]
run:
    {{ wails }} run

# Build and run the headless server; open http://localhost:8080
[group('dev')]
run-server:
    {{ wails }} run:server

# Regenerate the TypeScript bindings under frontend/bindings
[group('dev')]
bindings:
    {{ wails }} common:generate:bindings

# Print tool versions
[group('dev')]
info:
    @echo "Go:            $({{ mise_exec }} go version 2>/dev/null || echo 'not installed')"
    @echo "Bun:           $({{ mise_exec }} bun --version 2>/dev/null || echo 'not installed')"
    @echo "wails3:        $({{ mise_exec }} wails3 version 2>&1 || echo 'not installed')"
    @echo "golangci-lint: $({{ golangci }} version --short 2>/dev/null || echo 'not installed')"
    @echo "Docker:        $(docker --version 2>/dev/null || echo 'not installed (needed for docker and cross builds)')"

# Code Quality

alias f := format
alias fmt := format
alias tc := typecheck

# Format Go (gofumpt, goimports, gci, golines via golangci-lint), apply modernize fixes, fix markdown
[group('quality')]
format:
    {{ golangci }} fmt ./...
    {{ mise_exec }} modernize -fix -test ./...
    {{ mise_exec }} markdownlint-cli2 --fix

# Check Go formatting without writing
[group('quality')]
format-check:
    {{ golangci }} fmt --diff ./...

# Lint everything: go vet, golangci-lint, modernize, frontend tsc, markdown
[group('quality')]
lint: lint-go typecheck lint-md

# Lint Go: go vet, golangci-lint (no fixes), modernize dry run
[group('quality')]
lint-go: _dist
    {{ mise_exec }} go vet ./...
    {{ golangci }} run --fix=false ./...
    {{ mise_exec }} modernize -test ./...

# Lint Go and apply the fixes golangci-lint offers (writes files)
[group('quality')]
lint-fix:
    {{ golangci }} run --fix ./...

# Type-check the frontend (no emit); needs generated bindings
[group('quality')]
typecheck: bindings
    cd frontend && {{ mise_exec }} bunx tsc --noEmit

# Lint markdown with markdownlint-cli2 (check only; `just format` applies fixes)
[group('quality')]
lint-md:
    {{ mise_exec }} markdownlint-cli2

# Run every pre-commit hook against all files
[group('quality')]
check: _dist
    {{ mise_exec }} pre-commit run --all-files

# Testing

alias t := test

# Run Go (with -race) and frontend unit tests
[group('test')]
test: test-go test-frontend

# Run Go tests under the race detector
[group('test')]
test-go: _dist
    {{ mise_exec }} go test -race ./...

# Run frontend vitest suite; needs generated bindings
[group('test')]
test-frontend: bindings
    cd frontend && {{ mise_exec }} bun run vitest run

# Go coverage for internal/ (the 80% target); prints the total line
[group('test')]
test-coverage: _dist
    {{ mise_exec }} go test -race -covermode=atomic -coverprofile=cover.out ./internal/...
    {{ mise_exec }} go tool cover -func=cover.out | tail -1

# Open the Go coverage report in a browser
[group('test')]
coverage: test-coverage
    {{ mise_exec }} go tool cover -html=cover.out

# Install the Playwright browser (run once; idempotent)
[group('test')]
e2e-install:
    cd frontend && {{ mise_exec }} bunx playwright install chromium

# Run the Playwright suite against a fresh server build
[group('test')]
e2e: build-server e2e-install
    cd frontend && {{ mise_exec }} bunx playwright test

# Open the Playwright UI runner against a fresh server build
[group('test')]
e2e-ui: build-server e2e-install
    cd frontend && {{ mise_exec }} bunx playwright test --ui

# Build

alias b := build

# Build the desktop app for the host OS -> bin/redtracks
[group('build')]
build:
    {{ wails }} build

# Build the headless server binary -> bin/redtracks-server
[group('build')]
build-server:
    {{ wails }} build:server

# Produce the production package for the host OS
[group('build')]
package:
    {{ wails }} package

# Build the server-mode Docker image
[group('build')]
build-docker:
    {{ wails }} build:docker

# Build and run the server-mode Docker image on 127.0.0.1:8080 with ./.redtracks-data mounted
[group('build')]
run-docker:
    {{ wails }} run:docker

# Build the cross-compilation Docker image (one-time, large download)
[group('build')]
setup-docker:
    {{ wails }} setup:docker

# CI

# Final gate before commits: hooks, formatting, lint, unit tests
[group('ci')]
ci-check: check format-check lint test
    @echo "[ci-check] all checks passed"

# Everything in ci-check plus the end-to-end suite
[group('ci')]
ci-full: ci-check e2e
    @echo "[ci-full] all checks passed"
