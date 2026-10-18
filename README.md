# RedTracks

Offline engagement tracker for pentests and CTFs: hosts, services, flags, credentials, and what is left to try. It runs on one machine, keeps everything in plain files, and needs no network.

## Install

The toolchain (Go, Wails, bun, golangci-lint, just) is pinned in `mise.toml`. Install mise, then:

```sh
mise install
just install    # pinned tools, git hooks, Go modules, frontend deps
```

`just` is itself one of the pinned tools, so until mise is activated in your shell run it as `mise exec -- just <recipe>`. Run `just` with no arguments to list every recipe.

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
  logs/                 session logs you imported, copied unchanged
  hosts/<ip>/
    host.json           services, flags, access paths, checklist state
    notes.md
    evidence.json       the evidence index: what each file proves
    evidence/           screenshots and text captures, copied in
    imports/            raw scan output, copied unchanged
inbox/                  drop screenshots here; the app offers to file them
redtracks.log
```

The app writes the JSON files atomically and reloads them when you edit them by hand. A file that fails to parse is reported with its path, line, and column and is never overwritten. `notes.md` files and everything under `checklists/` belong to you.

## Checklists

A checklist is one JSON file in `checklists/` with a `match` rule (service types or host OS) and a list of items, each with a title, a hint, and commands. A file may also declare `slots`: the proof a matching host or service owes, such as a shell screenshot. See `checklists/_readme.md`. On first run the bundled set is copied there once and never overwritten. Edit, add, or delete files freely; the app reloads them while it runs.

On a host page, focus an item and press space to mark it done. Skip and N/A are buttons on the row. Commands substitute the host IP and engagement domain when copied.

## Evidence

Evidence is a screenshot or a text capture copied into `hosts/<ip>/evidence/` and indexed in `evidence.json` with the flags, findings, access paths, and proof slots it proves. Capture it three ways: drop or paste onto a host, a flag, a finding, an access path, or a slot; drop files into `~/RedTracks/inbox` and assign them from the Inbox screen; or import a terminal session log by path, search it, and clip a line range. Text is stored with terminal control sequences removed; images are stored byte for byte, 20 MB at most. Nothing is ever moved out of the engagement folder, and a file you drop into `evidence/` by hand shows up as "not indexed" until you adopt it.

The board sorts hosts by owed proof first: missing flags and missing slots together. What's left names each missing slot beside the open checklist items.

## Add an importer

1. Write one parser file in `internal/importers/` with a `func(data []byte) (Result, error)`, and register it in `var parsers` in `parse.go`.
2. Add one sniff rule in `detect.go` so `Detect` recognises the file from its first 8 KB.
3. Add one fixture under `internal/importers/testdata/` and a row for it in `TestDetectAllFixtures`.

## License

Apache-2.0. See `LICENSE`.
