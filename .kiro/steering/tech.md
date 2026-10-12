# Tech Stack

RedTracks is a [Wails v3](https://v3.wails.io/) desktop app: a Go backend with an embedded React web frontend, plus a headless server build used as a browser fallback and for e2e tests.

## Backend (Go)

- Go 1.26 (toolchain pinned in `mise.toml`), module `github.com/unclesp1d3r/redtracks`.
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

- [mise](https://mise.jdx.dev/) manages all tools; run commands through `mise exec -- <cmd>` (or in an activated mise shell). Install with `mise install`.
- [Task](https://taskfile.dev/) (`wails3 task ...`) is the task runner; the root `Taskfile.yml` dispatches to per-platform files under `build/`.
- Lint/format: `golangci-lint` (config in `.golangci.yml`), `gofumpt`, `goimports`, `gci`, `golines` (max line length 120); frontend type-checks with `tsc`.
- pre-commit hooks and markdownlint/mdformat are configured at the repo root.

## Common commands

Run via mise, e.g. `mise exec -- wails3 task dev`.

```bash
mise install                      # install the pinned toolchain
wails3 task dev                   # run the app in development (native window)
wails3 task run:server            # server mode; open http://localhost:8080 fallback
wails3 task build                 # build the desktop app -> bin/redtracks
wails3 task package               # production package for the host OS
wails3 task test                  # go test ./... + frontend vitest run
wails3 task lint                  # go vet, golangci-lint, frontend tsc --noEmit
wails3 task e2e                   # Playwright happy-path against the server build
wails3 task common:generate:bindings   # regenerate TypeScript bindings
```

Frontend-only (from `frontend/`): `bun run dev`, `bun run build`, `bun run vitest run`.

## Verification expectations

- Go packages under `internal/` target 80% line coverage.
- Run `test` and `lint` before considering a change done; Go store/importer tests are expected to pass under `-race`.
- The headless server build binds to `localhost` only and carries no auth.
