# Project Structure

```text
redtracks/
  main.go                 Wails app entry point; embeds frontend/dist, opens the window
  mise.toml               pinned toolchain and settings (bun as package manager)
  Taskfile.yml            root task runner, dispatches to build/<os>/Taskfile.yml
  go.mod / go.sum         Go module: github.com/unclesp1d3r/redtracks
  .golangci.yml           linter and formatter config
  checklists/             embedded sample checklist definitions + embed.go, _readme.md
  internal/               all tested Go behavior (not importable outside the module)
    model/                domain types, ID generation, default flag sets
    store/                engagement load, atomic write, hash-check mutations, poller
    checklist/            definition load, match, merge, safe command substitution, watch
    importers/            one file per tool + testdata/ fixtures; detect -> parse -> merge
    progress/             what's-left summaries, board ordering, Mermaid attack graph
    app/                  Wails services (thin), logging, binding surface
  frontend/               React + TypeScript app
    src/
      lib/                api.ts, query.ts (TanStack Query + Wails events)
      components/         AppShell, Mermaid, CommandPalette, QueryStates, shared UI
        ui/               shadcn/ui components
      features/           one folder per screen: engagements, board, host, tables, import, left
    bindings/             GENERATED Go->TS bindings (gitignored, do not hand-edit)
    e2e/                  Playwright specs
  build/                  per-platform Taskfiles, icons, Docker, packaging config
  bin/                    build output
  docs/                   plans and specs (see docs/plans, docs/superpowers)
```

## Architecture layering

Data flows **Frontend -> Wails services (`internal/app`) -> core packages (`internal/*`) -> disk**, with services emitting events back to the frontend for cache invalidation.

- `internal/app` services are **thin**: they bind to the frontend and delegate to core packages. Keep logic and all tested behavior in the core packages (`model`, `store`, `checklist`, `importers`, `progress`), not in services.
- Everything persistent goes through `internal/store`'s single atomic write path. Callers never hold pointers into store state; mutations are applied via closures against a freshly re-read document.
- Importers are plain functions, not plugins: one parser file, one sniff rule, and one fixture per tool under `importers/testdata/`.
- Progress/graph functions are pure and golden-tested where output must be stable.

## Conventions

- **Storage layout:** one folder per engagement under `~/RedTracks/engagements/` (override with `REDTRACKS_HOME`). Host folders are named only from IPs that pass `netip.ParseAddr`. Checklist definitions live under `~/RedTracks/checklists/`.
- **JSON:** camelCase field tags, two-space indentation, struct-ordered fields to keep diffs small. Checklist items use stable IDs so edits never orphan history.
- **Go style:** small, focused packages; types-only `model` package; errors wrapped with context; follow `.golangci.yml` (gofumpt, 120-char lines, named-return and magic-number rules enabled). Tests live in the same package as the code under test.
- **Frontend:** feature-first folders under `src/features/`; shared UI in `src/components`; every screen defines loading, empty, and error states, and color is never the only signal.
- Never hand-edit generated bindings in `frontend/bindings/` — regenerate them.
