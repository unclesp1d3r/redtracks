# RedTracks

Offline engagement tracker for pentests and CTFs: hosts, services, flags, credentials, and what is left to try. It runs on one machine, keeps everything in plain files, and needs no network.

## Install

The toolchain (Go, Wails, bun, golangci-lint, just) is pinned in `mise.toml`. Install mise, then:

```sh
mise install
just install    # git hooks, Go modules, frontend deps
```

Run `just` with no arguments to list every recipe.

## Run

```sh
just dev
```

If the native window is unavailable, run the same app as a local HTTP server and open <http://localhost:8080>:

```sh
just run-server
```

The server listens on localhost only and has no authentication.

## Test

```sh
just test       # Go (-race) and frontend unit tests
just lint       # go vet, golangci-lint, modernize, tsc, markdownlint
just e2e        # Playwright against a fresh server build
just ci-check   # pre-commit hooks, format check, lint, test
```

## Where data lives

Under `$REDTRACKS_HOME`, or `~/RedTracks` when it is unset:

```text
checklists/             your checklist definitions
engagements/<name>/
  engagement.json       engagement metadata
  creds.json            credentials, users, shares, loot
  findings.json
  notes.md
  hosts/<ip>/
    host.json           services, flags, access paths, checklist state
    notes.md
    imports/            raw scan output, copied unchanged
redtracks.log
```

The app writes the JSON files atomically and reloads them when you edit them by hand. A file that fails to parse is reported with its path, line, and column and is never overwritten. `notes.md` files and everything under `checklists/` belong to you.

## Checklists

A checklist is one JSON file in `checklists/` with a `match` rule (service types or host OS) and a list of items, each with a title, a hint, and commands. On first run the bundled set is copied there once and never overwritten. Edit, add, or delete files freely; the app reloads them while it runs.

On a host page, focus an item and press space to mark it done. Skip and N/A are buttons on the row. Commands substitute the host IP and engagement domain when copied.

## Add an importer

1. Write one parser file in `internal/importers/` with a `func(data []byte) (Result, error)`, and register it in `var parsers` in `parse.go`.
2. Add one sniff rule in `detect.go` so `Detect` recognises the file from its first 8 KB.
3. Add one fixture under `internal/importers/testdata/` and a row for it in `TestDetectAllFixtures`.

## License

Apache-2.0. See `LICENSE`.
