# AGENTS.md

Guidance for AI coding agents working in the RedTracks repo. (Kiro reads the same material from `.kiro/steering/`; this file mirrors it for other tools.)

## Project context

The canonical, maintained guidance lives in `.kiro/steering/`. Read these first:

- [`.kiro/steering/product.md`](.kiro/steering/product.md) — what RedTracks is, core principles, key features, and scope boundaries.
- [`.kiro/steering/tech.md`](.kiro/steering/tech.md) — tech stack, toolchain, and the common build/test/lint commands.
- [`.kiro/steering/structure.md`](.kiro/steering/structure.md) — folder layout, architecture layering, and conventions.

If anything below conflicts with those files, the steering docs win — update this file to match.

## TL;DR

RedTracks is an **offline, single-user** Wails v3 desktop app (Go backend, embedded React + TypeScript frontend) for tracking pentest/CTF engagements. State is plain JSON files under `~/RedTracks/`, hand-editable without the app. No network calls, no database, no multi-user.

## Setup

```bash
mise install    # installs the pinned toolchain (Go, bun, wails3, linters, ...)
```

Run tools through `mise exec -- <cmd>` (or activate a mise shell). The package manager is **bun**, not npm/pnpm.

## Common commands

```bash
mise exec -- wails3 task dev          # run the app in development (native window)
mise exec -- wails3 task run:server   # server mode; open http://localhost:8080
mise exec -- wails3 task build        # build the desktop app -> bin/redtracks
mise exec -- wails3 task test         # go test ./... + frontend vitest run
mise exec -- wails3 task lint         # go vet, golangci-lint, frontend tsc --noEmit
mise exec -- wails3 task e2e          # Playwright happy-path against the server build
mise exec -- wails3 task common:generate:bindings   # regenerate TS bindings
```

Run `test` and `lint` before considering a change done. Go store/importer tests are expected to pass under `-race`.

## Conventions that matter

- **Keep logic in `internal/` core packages** (`model`, `store`, `checklist`, `importers`, `progress`), not in the thin `internal/app` Wails services.
- **All persistence goes through `internal/store`'s single atomic write path.** Never hold pointers into store state; mutate via closures against a freshly re-read document.
- **JSON on disk** uses camelCase tags, two-space indentation, and struct-ordered fields; checklist items use stable IDs. The files are the interface — keep them readable and keep faith with external edits.
- **Target-controlled strings are hostile.** Validate hostnames, domains, and OS strings from scan output before they become file paths, shell substitutions, or Mermaid labels.
- **Go style:** small focused packages; errors wrapped with context; follow `.golangci.yml` (gofumpt, 120-char lines). Tests live in the same package as the code under test.
- **Frontend:** feature-first folders under `src/features/`; shared UI in `src/components`; dark theme only; every screen defines loading, empty, and error states, and color is never the only signal.
- **Do not hand-edit** generated bindings in `frontend/bindings/` — regenerate them.
- **Dependency discipline:** standard library first; Wails is pinned to one beta tag, bumped only in a dedicated commit, never mid-feature.
