# Tech Stack

RedTracks is a [Wails v3](https://v3.wails.io/) desktop app: a Go backend with an embedded React web frontend, plus a headless server build used as a browser fallback and for e2e tests.

## Backend (Go)

- Go 1.27 (toolchain pinned in `mise.toml`), module `github.com/unclesp1d3r/redtracks`.
- Wails `v3.0.0-beta.28` — pinned to one beta tag; version bumps are their own commit, never mid-feature.
- Standard library first; current dependencies are only Wails and its transitive deps.
- No database. State is plain JSON files written through one atomic write path (temp file, fsync, rename) with per-file SHA-256 hash-check-and-reapply.

## Frontend

- React 18 + TypeScript, built with Vite 8.
- [shadcn/ui](https://ui.shadcn.com/) components over Tailwind CSS v4; dark theme only.
- TanStack Query for data, driven by Wails events (`engagement:changed`, `engagement:error`) for cache invalidation.
- react-router for routing, Mermaid for the attack graph (strict security mode + DOMPurify on output), sonner for toasts.
- Package manager is **bun** (configured in `mise.toml`).
- TypeScript bindings to Go services are **generated** into `frontend/bindings/` (gitignored) — do not hand-edit them; regenerate instead.

## Toolchain

- [mise](https://mise.jdx.dev/) manages all tools; install with `mise install`. Ad-hoc commands run through `mise exec -- <cmd>` (or in an activated mise shell).
- [just](https://just.systems/) is the front door for every workflow (`justfile` at the repo root; `just` alone lists recipes). Recipes invoke tools through `mise exec`.
- [Task](https://taskfile.dev/) (`wails3 task ...`) owns the Wails build pipeline; the root `Taskfile.yml` dispatches to per-platform files under `build/`. The just build, package, run, dev, server, docker, and bindings recipes delegate to it.
- Lint/format: `golangci-lint` (config in `.golangci.yml`), `gofumpt`, `goimports`, `gci`, `golines` (max line length 120); `modernize` (gopls analyzers, dry run in `just lint`, `-fix` in `just format`); frontend type-checks with `tsc`; markdown via `markdownlint-cli2`.
- pre-commit hooks and markdownlint/mdformat are configured at the repo root.

## Common commands

```bash
mise install                      # install the pinned toolchain
just install                      # pinned tools, git hooks, go mod tidy, bun install
just dev                          # run the app in development (native window)
just run-server                   # server mode; open http://localhost:8080 fallback
just build                        # build the desktop app -> bin/redtracks
just package                      # production package for the host OS
just test                         # go test -race ./... + frontend vitest run
just lint                         # go vet, golangci-lint, modernize, tsc --noEmit, markdownlint
just format                       # golangci-lint fmt + modernize -fix
just e2e                          # Playwright happy-path against a fresh server build
just bindings                     # regenerate TypeScript bindings
just ci-check                     # pre-commit hooks, format check, lint, test
```

Frontend-only (from `frontend/`): `bun run dev`, `bun run build`, `bun run vitest run`.

## Verification expectations

- Go packages under `internal/` target 80% line coverage.
- Run `just test` and `just lint` before considering a change done; Go store/importer tests are expected to pass under `-race`.
- The headless server build binds to `localhost` only and carries no auth.
