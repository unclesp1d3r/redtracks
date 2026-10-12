# Requirements Document

## Introduction

RedTracks slice one is an offline, single-user macOS desktop application that tracks the state of a penetration test — hosts, services, per-service checklists, credentials, users, shares, loot, findings, and flags — during OffSec labs and exams (OSCP+, OSAI) and CTFs. It replaces the templated-note workflow with structure as data, surfacing which ports still have untouched checklists, which credentials were never tried against a host, which flags are missing, and how each host is reached.

The product is built on three principles that these requirements enforce. First, it is offline and local only: no network call, no sync, no cloud, no telemetry, no multi-user support. Second, the files are the interface: all state is plain JSON files (plus `notes.md`) under `~/RedTracks`, readable and hand-editable in an external editor, and the app keeps faith with external edits through a single atomic write path, per-file hash-check-and-reapply, and a two-second mtime poll. Third, target-controlled strings are hostile: hostnames, domains, and OS strings from scan output are validated before they ever become file paths, shell command substitutions, or Mermaid graph labels.

These requirements are derived from the approved design document (`design.md`) and align one-to-one with the implementation plan's canonical requirement list R1–R24 (`docs/plans/2026-10-12-001-feat-redtracks-slice-one-plan.md`). Requirement numbering is held stable so that the design's "Validates: Requirements X.Y" property references remain valid.

## Glossary

- **RedTracks**: The desktop application described by this document; the overall system.
- **Store**: The backend component (`internal/store`) that owns every byte written under an engagement folder, performing load, atomic write, hash-check-and-reapply mutations, parse-error reporting, and mtime polling.
- **Checklist_Engine**: The backend component (`internal/checklist`) that loads definitions, matches them to services and hosts, merges them with recorded state, substitutes commands, hot-reloads, and installs the shipped set.
- **Importer**: The backend component (`internal/importers`) that detects a file's tool from content, parses it into a normalized result, builds a preview, and merges it into the Store.
- **Progress_Engine**: The backend component (`internal/progress`) that computes per-host "what's left" summaries, board ordering, and the Mermaid attack graph.
- **UI**: The embedded React + TypeScript + shadcn/ui frontend presenting six screens and a command palette.
- **Engagement**: A single tracked penetration test, stored as one folder under `~/RedTracks/engagements/`.
- **Engagement_Type**: One of `lab`, `oscp`, `osai`, or `custom`, which sets the default flag types and points for an Engagement.
- **Host**: A tracked target within an Engagement, stored as `hosts/<ip>/host.json`.
- **Service**: An open network service on a Host, with a port, protocol, and open-string type.
- **Checklist_Definition**: A JSON file under `~/RedTracks/checklists/` describing an ordered set of checklist items matched to services by type or hosts by OS, with stable item IDs.
- **Checklist_State**: The recorded status (`todo`, `done`, `skipped`, `na`) and note for each checklist item, keyed by stable item ID.
- **Flag_Type**: A kind of flag (such as `local`, `proof`, `ai_vector`) with an ID, label, and point value.
- **Access_Path**: A recorded way a Host is reached, with a method, optional via-host, optional credential, command, and notes.
- **Credential**: A username/secret pair in the engagement-wide credentials table, tracking the host it was found on (`foundOnHostId`), hosts it works on (`worksOn`), and hosts it was tried on (`triedOn`).
- **Atomic_Write**: The single write operation (temp file in the same directory, fsync, rename over the original) through which the Store performs all persistence.
- **Parse_Error**: An error reporting a failed JSON parse with path, line, and column; the offending file is never overwritten.
- **Round_Trip**: Writing a value to disk and reading it back such that the result is equal to the original modulo in-memory-only fields.
- **Attack_Graph**: A Mermaid flowchart of Hosts and Access_Paths rendered on the Board screen.

## Requirements

### Requirement 1: Hand-editable JSON storage, one folder per engagement

**User Story:** As a pentest operator, I want all engagement state stored as plain JSON files in one folder per engagement, so that I can read and hand-edit my data in an external editor if the app ever breaks.

#### Acceptance Criteria

1. WHEN RedTracks writes an Engagement to disk and later reads the same Engagement folder with no intervening external modification, THE Store SHALL produce an Engagement equal to the written Engagement for every persisted field, ignoring only the in-memory-only engagement directory field (Round_Trip equality).
2. THE Store SHALL persist all Engagement state as JSON files within a single folder, one folder per Engagement, directly under `~/RedTracks/engagements/`.
3. THE Store SHALL write every app-owned JSON file with two-space indentation and with JSON fields emitted in struct-declaration order.
4. WHEN RedTracks opens an Engagement folder, THE Store SHALL load the Engagement by reading its files directly from disk, without consulting or populating any intermediate cache or index.
5. WHERE a referenced app-owned file does not exist on read, THE Store SHALL treat that file as absent state and SHALL NOT return an error for its absence.

### Requirement 2: Single atomic write path and model integrity

**User Story:** As a pentest operator, I want every write to be atomic and every record to carry a unique identity, so that a crash or interruption never leaves my engagement files corrupted or partially written.

#### Acceptance Criteria

1. WHEN RedTracks persists any app-owned file, THE Store SHALL perform the write as an Atomic_Write such that, after the operation, the file contains either the complete new content or the unchanged previous content, and never a partial mixture of the two.
2. WHEN an Atomic_Write completes successfully, THE Store SHALL leave no temporary file remaining in the target directory and SHALL record a stored SHA-256 hash equal to the SHA-256 hash of the exact bytes written.
3. IF an Atomic_Write fails or is interrupted at any point before the final rename over the target path, THEN THE Store SHALL leave the original file content byte-for-byte unchanged.
4. THE Store SHALL route every app-owned file write through a single Atomic_Write function, with the append-only application log as the only documented exemption.
5. WHEN RedTracks creates a record, THE Store SHALL assign the record an identifier that is non-empty, opaque, and generated such that two independently created records do not receive the same identifier, and THE System SHALL NOT use that identifier as the record's primary human-facing label.

### Requirement 3: Hash-check-and-reapply mutations

**User Story:** As a pentest operator, I want my in-app edits and my external file edits to both survive, so that changing a field in the app never silently discards an edit I made in my editor.

#### Acceptance Criteria

1. WHEN RedTracks applies a mutation to an app-owned file whose on-disk content changed since the last read and still parses successfully, THE Store SHALL reparse the fresh document and reapply the mutation closure to that fresh document, so that both the external edit and the app's edit are present in the written file.
2. WHEN the Store applies a mutation, THE Store SHALL re-read the target file and compare its SHA-256 hash to the stored SHA-256 hash before writing.
3. WHEN a mutation completes successfully, THE Store SHALL record a stored SHA-256 hash equal to the SHA-256 hash of the exact bytes written.
4. THE Store SHALL apply each mutation against a deep copy of its in-memory value so that callers never receive or retain a pointer into Store state.
5. IF the fresh on-disk document fails to parse while re-reading before a mutation, THEN THE Store SHALL abort the mutation, return a Parse_Error naming the path, line, and column, and leave the file content unchanged.

### Requirement 4: Positioned parse-error reporting without overwrite

**User Story:** As a pentest operator, I want files that fail to parse reported with their exact location and never overwritten, so that a hand-edit mistake is pointed out to me instead of silently destroyed.

#### Acceptance Criteria

1. IF an app-owned JSON file fails to parse, THEN THE Store SHALL report a Parse_Error naming the file path, the 1-based line number, and the 1-based column number of the error, and SHALL leave the file content unchanged.
2. THE Store SHALL NOT rewrite or reformat a file that failed to parse for the purpose of correcting it.
3. IF a `host.json` file declares an IP that does not equal its containing folder name, THEN THE Store SHALL report a Parse_Error naming the declared IP value, the folder name value, and the file path, and SHALL leave the file content unchanged.

### Requirement 5: Notes conflict detection on external change

**User Story:** As a pentest operator, I want saving notes over an externally changed `notes.md` to be refused with a resolvable conflict, so that my editor edits to notes are never silently clobbered by the app.

#### Acceptance Criteria

1. IF the current on-disk SHA-256 hash of a `notes.md` file differs from the base hash the editor loaded, THEN THE Store SHALL refuse the save, return a conflict error whose message begins with the text `conflict`, and leave the file content unchanged.
2. WHEN RedTracks reads a `notes.md` file, THE Store SHALL return the note text together with the current hex-encoded SHA-256 hash of its on-disk bytes to serve as the conflict baseline.
3. WHEN the base hash the editor loaded equals the current on-disk SHA-256 hash, THE Store SHALL write the note text successfully and SHALL record the SHA-256 hash of the written bytes as the new baseline.
4. WHEN RedTracks reads a `notes.md` file that does not exist, THE Store SHALL return empty note text with the baseline hash of empty content, so that the first save of a new note is not reported as a conflict.

### Requirement 6: Two-second mtime poll surfaces external changes

**User Story:** As a pentest operator, I want external file changes picked up automatically, so that edits I make in my editor show up in the app without me reopening the engagement.

#### Acceptance Criteria

1. WHILE an Engagement is open, THE Store SHALL poll file modification times every two seconds and reload each file whose modification time changed since the previous poll.
2. WHEN a poll detects one or more changed files that reload successfully, THE Store SHALL emit a single change notification identifying the changed file paths.
3. IF a file fails to parse during polling, THEN THE UI SHALL surface the Parse_Error as a persistent banner naming the file path and the 1-based line and column, and SHALL NOT fail silently.
4. WHEN a file fails to parse during a reload, THE Store SHALL retain the last successfully loaded copy of that file, continue reloading the remaining changed files, and return the Parse_Error after processing all remaining changed files.

### Requirement 7: Host folder names restricted to valid IPs

**User Story:** As a pentest operator, I want host folders named only from validated IP addresses, so that hostile scan output can never create a path-traversal folder on my disk.

#### Acceptance Criteria

1. IF an IP string passed to an IP-taking path helper fails `netip.ParseAddr`, THEN THE Store SHALL reject the string, return a validation error that identifies the rejected IP string, and SHALL NOT create, rename, or write any file or folder for that string.
2. IF the IP supplied to `AddHost` or `SaveImport` fails `netip.ParseAddr`, THEN THE Store SHALL perform no write, leave all existing engagement files unchanged, and return the validation error to the caller.
3. WHEN RedTracks creates a Host folder, THE Store SHALL derive the folder name only from an IP that passes `netip.ParseAddr`, and the resulting folder name SHALL be exactly the parsed IP string with no path separator, parent-directory segment, or whitespace.
4. WHERE an engagement-level notes path is requested with an empty IP string, THE Store SHALL accept the empty IP and resolve the path to the engagement-level `notes.md` rather than a host-level file.

### Requirement 8: Engagement type sets default flag types and points

**User Story:** As a pentest operator, I want the engagement type to set the right flag types and points, so that an OSAI exam and a lab each start with the scoring model that matches them.

#### Acceptance Criteria

1. WHEN RedTracks creates an Engagement whose Engagement_Type is `lab` or `oscp`, THE System SHALL assign exactly two Flag_Types in order: `local` (10 points) then `proof` (10 points).
2. WHEN RedTracks creates an Engagement whose Engagement_Type is `osai`, THE System SHALL assign exactly three Flag_Types in order: `ai_vector` (15 points), `trad_vector` (10 points), then `proof` (5 points).
3. WHEN RedTracks creates an Engagement whose Engagement_Type is `custom`, THE System SHALL assign an empty set of Flag_Types (zero entries).
4. WHEN RedTracks assigns default Flag_Types for an Engagement_Type, THE System SHALL use the exact IDs, labels, point values, and ordering stated for that type and SHALL NOT add, omit, or reorder any Flag_Type.
5. WHERE an operator has edited the Flag_Types in `engagement.json`, THE System SHALL honor the edited Flag_Types as written and SHALL NOT reset them to the type defaults.

### Requirement 9: Host record fields

**User Story:** As a pentest operator, I want each host to record its identity, status, access paths, services, flags, and a host-level checklist, so that I can see the full state of a target in one place.

#### Acceptance Criteria

1. THE System SHALL store each Host with an IP, a hostname, an OS string, a status, a set of tags, a list of access paths, a list of services, a list of flags, and a host-level Checklist_State.
2. THE System SHALL constrain Host status to exactly one of `untouched`, `enumerating`, `foothold`, `user`, `root`, or `done`.
3. WHEN RedTracks serializes a Host, THE System SHALL order the JSON fields with the identifier field first, and SHALL place the access paths field before the services field and the services field before the flags field.
4. WHEN an operator sets a Host status to one of the six allowed values, THE HostService SHALL persist the new status through the Store and the persisted `host.json` SHALL reflect the new status on the next read.
5. IF an operator sets a Host status to a value that is not one of the six allowed values, THEN THE HostService SHALL reject the change, leave the stored status unchanged, and return a validation error identifying the invalid value.

### Requirement 10: Service record fields

**User Story:** As a pentest operator, I want each service to record its port, protocol, type, product, version, web paths, and a per-service checklist keyed by stable item IDs, so that progress on each service survives edits to the checklist definition.

#### Acceptance Criteria

1. THE System SHALL store each Service with a port, a protocol, an open-string type, a product, a version, a list of web paths, and a Checklist_State keyed by stable item IDs.
2. THE System SHALL constrain Service port to an integer from 1 to 65535 inclusive and Service protocol to exactly one of `tcp` or `udp`.
3. WHEN an Importer provides a Service type, THE System SHALL store the provided value verbatim as an open string and SHALL NOT infer, normalize, or substitute an AI service type.
4. WHEN an operator updates a Service, THE HostService SHALL replace only the scalar fields (port, protocol, type, product, version) and SHALL leave the existing web paths list and Checklist_State unchanged.
5. IF an operator updates a Service with a port outside 1 to 65535 or a protocol other than `tcp` or `udp`, THEN THE HostService SHALL reject the update, leave the stored Service unchanged, and return a validation error identifying the invalid field.

### Requirement 11: Engagement-wide tables with host references

**User Story:** As a pentest operator, I want credentials, users, shares, loot, and findings tracked in engagement-wide tables with host references, so that I know which credentials I found where and which I have not yet tried on a host.

#### Acceptance Criteria

1. THE System SHALL store credentials, users, shares, loot, and findings as engagement-wide tables whose rows reference hosts by host identifier.
2. THE System SHALL store each Credential with a found-on host identifier (`foundOnHostId`), a list of hosts it works on (`worksOn`), and a list of hosts it was tried on (`triedOn`).
3. WHEN an operator marks a Credential tried on a Host, THE EngagementService SHALL append that Host identifier to `triedOn` only if it is not already present, so that marking the same Host tried more than once leaves `triedOn` containing that Host identifier exactly once.
4. WHEN an operator marks a Credential tried on a Host and indicates the credential works, THE EngagementService SHALL append that Host identifier to `worksOn` only if it is not already present.
5. WHEN an operator edits a table row, THE EngagementService SHALL replace only the row's scalar fields and SHALL leave `worksOn`, `triedOn`, and web paths unchanged.
6. WHEN an operator deletes a table row that exists, THE EngagementService SHALL remove that row from its table and the removed row SHALL be absent on the next read.
7. IF an operator references a host identifier that does not exist in the Engagement when marking a Credential tried or working, THEN THE EngagementService SHALL reject the operation, leave the Credential unchanged, and return an error identifying the unknown host identifier.

### Requirement 12: Checklist definitions matched by type and OS with stable IDs

**User Story:** As a pentest operator, I want checklist definitions stored as JSON matched by service type and host OS with stable item IDs, so that editing a definition never orphans the progress I already recorded.

#### Acceptance Criteria

1. WHEN the Checklist_Engine merges a Checklist_Definition with a Checklist_State, THE Checklist_Engine SHALL preserve the recorded status and note for every item whose stable ID is present in both the definition and the recorded state.
2. WHEN the Checklist_Engine merges a Checklist_Definition with a Checklist_State, THE Checklist_Engine SHALL read any definition item that has no matching recorded state as status `todo`.
3. THE Checklist_Engine SHALL load Checklist_Definitions from JSON files located under `~/RedTracks/checklists/`.
4. THE Checklist_Engine SHALL match a Checklist_Definition to a Service by service type and to a Host by host OS.
5. IF two loaded Checklist_Definition files declare the same item ID, THEN THE Checklist_Engine SHALL return an error naming both file paths and the duplicated item ID, and SHALL NOT merge either definition.
6. WHEN an operator changes an item's stable ID in a Checklist_Definition, THE Checklist_Engine SHALL treat the item as a new item with no recorded state and read its status as `todo`.

### Requirement 13: Ten sample definitions ship embedded, copied once

**User Story:** As a pentest operator, I want a useful set of checklist definitions out of the box that I can then own and edit, so that I start with coverage but my customizations are never overwritten.

#### Acceptance Criteria

1. WHEN RedTracks launches and no `~/RedTracks/checklists/` directory exists, THE Checklist_Engine SHALL create the directory and copy all ten embedded sample Checklist_Definitions into it before loading any definition.
2. WHERE a `~/RedTracks/checklists/` directory already exists, THE Checklist_Engine SHALL leave every existing file byte-for-byte unchanged and SHALL add no file, including when the existing directory is empty.
3. IF copying the embedded sample Checklist_Definitions fails partway, THEN THE Checklist_Engine SHALL leave the `~/RedTracks/checklists/` directory with none of the ten sample files rather than a partial set, and SHALL emit an error indicating the copy failed.
4. THE System SHALL ship exactly ten embedded sample Checklist_Definitions, each containing at least five and at most ten items.

### Requirement 14: Definitions hot-reload without blocking startup

**User Story:** As a pentest operator, I want checklist definitions to reload while the app runs and a bad definition to never block startup, so that I can refine my checklists mid-engagement without restarting or risking a broken launch.

#### Acceptance Criteria

1. WHILE RedTracks is running, WHEN a definition file under `~/RedTracks/checklists/` is created, modified, or deleted, THE Checklist_Engine SHALL reload the Checklist_Definitions within two seconds of the change.
2. IF a Checklist_Definition file fails to load during a reload, THEN THE Checklist_Engine SHALL retain the last successfully loaded definition set unchanged and SHALL emit an error naming the definition file and, when the failure is a JSON parse error, its line and column.
3. IF a Checklist_Definition file is invalid when RedTracks starts, THEN THE System SHALL complete startup and present the UI normally using the remaining valid definitions, and SHALL log an error naming the invalid file rather than aborting launch.

### Requirement 15: Safe command substitution of $T and $D only

**User Story:** As a pentest operator, I want command substitution limited to validated target and domain tokens, so that a hostile hostname or domain from scan output can never inject shell metacharacters into a command I copy.

#### Acceptance Criteria

1. WHEN the Checklist_Engine substitutes a command, THE Checklist_Engine SHALL replace every occurrence of the literal token `$T` with the target IP and every occurrence of the literal token `$D` with the domain, SHALL leave all other characters of the command unchanged, and SHALL insert each validated value verbatim without adding, escaping, or removing any character.
2. IF the target IP fails `netip.ParseAddr`, THEN THE Checklist_Engine SHALL return an error naming the invalid IP and SHALL NOT return a substituted command.
3. IF a non-empty domain does not match `^[A-Za-z0-9._-]+$`, THEN THE Checklist_Engine SHALL return an error naming the invalid domain and SHALL NOT return a substituted command.
4. WHERE the domain is empty and the command contains the token `$D`, THE Checklist_Engine SHALL return an error indicating a required domain is missing and SHALL NOT return a substituted command.

### Requirement 16: Import detection by content for nine formats

**User Story:** As a pentest operator, I want RedTracks to recognize a dropped scan file by its content, so that I can import output from my tools without choosing a format by hand.

#### Acceptance Criteria

1. WHEN an operator provides a scan file, THE Importer SHALL detect the tool from the file content across nmap XML (including rustscan output), masscan JSON, ffuf JSON, feroxbuster JSONL, gobuster text, nikto JSON, nuclei JSONL, enum4linux-ng JSON, and netexec text.
2. WHEN the Importer parses netexec text, THE Importer SHALL strip ANSI escape sequences from the content before detection and parsing so that colored and uncolored output of the same scan produce the identical parsed result.
3. WHEN the Importer parses a port-scan result, THE Importer SHALL create a Service only for a port whose state is open and SHALL skip every host whose state is down.
4. IF a parsed host result carries a non-empty IP that fails `netip.ParseAddr`, THEN THE Importer SHALL drop only that host result, count it as dropped, and continue parsing the remaining host results in the file.
5. IF the file content matches no known format, THEN THE Importer SHALL return an unknown-format error listing the formats tried.
6. IF the provided file is empty, THEN THE Importer SHALL return an unknown-format error and SHALL write nothing.

### Requirement 17: Import preview with resolution before any write

**User Story:** As a pentest operator, I want every import to show a preview with new, fills, and duplicate badges and resolve hosts before anything is written, so that I always see and confirm what an import will change.

#### Acceptance Criteria

1. IF a host result cannot be resolved by IP, by URL host matching an engagement host IP or hostname, or by an operator-supplied override, THEN THE Importer SHALL report it as unresolved in the preview and SHALL write nothing for it.
2. WHEN the Importer builds a preview, THE Importer SHALL badge every resolved row as exactly one of `new`, `fills`, or `duplicate`.
3. THE Importer SHALL write nothing to the Store until the operator confirms the preview, including when the preview contains at least one unresolved host result.
4. WHEN an operator supplies an override for an unresolved host whose override IP passes `netip.ParseAddr`, THE Importer SHALL resolve the host result to the override IP and port before writing.
5. IF an operator supplies an override whose IP fails `netip.ParseAddr`, THEN THE Importer SHALL keep the host result unresolved, return an error naming the invalid override IP, and write nothing for it.

### Requirement 18: Merge preserves operator intent and deduplicates

**User Story:** As a pentest operator, I want merges to never overwrite the service types I set, to deduplicate repeated findings, and to keep the raw file, so that importing twice is safe and my manual classifications are respected.

#### Acceptance Criteria

1. WHEN the Importer applies the same parsed result twice, THE Importer SHALL produce an Engagement state after the second application byte-for-byte equal to the state after the first application (idempotent merge).
2. THE Importer SHALL leave an operator-set Service type unchanged and SHALL fill only the type, product, version, and notes fields that are empty at merge time.
3. THE Importer SHALL deduplicate web paths by the combination of path and status, findings by the combination of source and title per Service, and users and shares by name per Host, matching dedup keys case-insensitively.
4. WHEN the Importer resolves at least one Host for a parsed result, THE Importer SHALL copy the raw file byte-for-byte unchanged under the first resolved Host's `imports/` directory.
5. IF the Importer resolves no Host for a parsed result, THEN THE Importer SHALL copy no raw file and SHALL write nothing to the Store.

### Requirement 19: Per-host what's-left and most-neglected board ordering

**User Story:** As a pentest operator, I want each host's missing flags, untouched services, open items, untried credentials, and missing access path computed and the board sorted most-neglected-first, so that at hour nineteen of an exam I immediately see what to try next.

#### Acceptance Criteria

1. WHEN the Progress_Engine summarizes a Host, THE Progress_Engine SHALL compute and return, in a single summary result, that Host's missing flags, untouched services, open checklist items, untried credentials, and a boolean indicating whether it lacks an access path.
2. THE Progress_Engine SHALL list a Service as untouched only when a matching Checklist_Definition exists for that Service and every item of that definition has Checklist_State status `todo`.
3. THE Progress_Engine SHALL exclude from a Host's untried credentials every Credential whose `triedOn` list includes the Host, whose `worksOn` list includes the Host, or whose `foundOnHostId` equals the Host.
4. IF a Host has no Access_Path and its status is any value beyond `untouched`, THEN THE Progress_Engine SHALL report that Host as missing an access path; otherwise THE Progress_Engine SHALL NOT report it as missing an access path.
5. WHEN the Progress_Engine orders the board, THE Progress_Engine SHALL sort hosts in ascending order by the ratio of captured flag points to total flag points (treating a total of 0 points as a ratio of 1.0), then break ties in ascending order by checklist completion ratio (completed items over total items, treating 0 items as 1.0), then break remaining ties by Host identifier in ascending lexicographic order, so that the ordering is deterministic and the most neglected Host is first.

### Requirement 20: Mermaid attack graph with escaped labels in strict mode

**User Story:** As a pentest operator, I want the board to render an attack graph of hosts and access paths with escaped labels in strict security mode, so that a hostile hostname can never break out of a graph label.

#### Acceptance Criteria

1. WHEN the Progress_Engine renders the Attack_Graph, THE Progress_Engine SHALL escape every label so that no double quote, single quote, left or right angle bracket, carriage return, or newline character remains unescaped inside any label, and SHALL truncate each label to at most 60 characters before escaping.
2. WHEN the Progress_Engine renders the Attack_Graph, THE Progress_Engine SHALL emit Mermaid flowchart text whose first non-whitespace line is `flowchart LR`, containing exactly one node per Host and exactly one edge per Access_Path.
3. WHEN the UI renders the Attack_Graph, THE UI SHALL initialize Mermaid in strict security mode and SHALL sanitize the rendered output through DOMPurify before inserting it into the document.
4. IF rendering or sanitizing the Attack_Graph fails, THEN THE UI SHALL display an error state identifying that the attack graph could not be rendered and SHALL insert no unsanitized output into the document.

### Requirement 21: Six screens, command palette, and space-to-toggle

**User Story:** As a pentest operator, I want six focused screens plus a command palette and a keyboard toggle for checklist items, so that I can move through my engagement quickly during an exam.

#### Acceptance Criteria

1. THE UI SHALL provide exactly six screens: Engagements, Board, Host, Tables, Import, and What's left.
2. WHEN an operator presses `cmd+k`, THE UI SHALL open a command palette that offers a jump entry per Host, a jump entry per Service, a "Go to" entry for each of the six screens, and an Add host entry.
3. WHEN an operator presses the space key while a checklist row has keyboard focus and focus is not inside a text input, THE UI SHALL toggle that item's Checklist_State between `todo` and `done`.
4. WHILE keyboard focus is inside a text input, THE UI SHALL ignore the space key as a checklist toggle and SHALL insert a space character into that input.
5. IF the operator enters a command palette query that matches no Host, Service, screen, or action, THEN THE UI SHALL display an empty-result indication within the open palette and SHALL perform no navigation.

### Requirement 22: Every screen defines loading, empty, and error states

**User Story:** As a pentest operator, I want every screen to define loading, empty, and error states and never rely on color alone, so that the app is readable and accessible under exam pressure.

#### Acceptance Criteria

1. WHILE a screen is awaiting its initial data, THE UI SHALL present a loading state for that screen.
2. WHILE a screen has successfully loaded but has no records to display, THE UI SHALL present an empty state for that screen.
3. IF a screen fails to load its data, THEN THE UI SHALL present an error state for that screen that conveys, in text, that loading failed.
4. WHERE state is conveyed by color, THE UI SHALL additionally convey that same state with text or an icon so that color is never the only signal.
5. THE UI SHALL give every icon-only control a non-empty accessible name and SHALL expose every toggle control's pressed or unpressed state to assistive technology.

### Requirement 23: Pinned Go and Wails toolchain

**User Story:** As a pentest operator, I want the Go and Wails versions pinned, so that builds are reproducible and a dependency change is a deliberate, isolated event.

#### Acceptance Criteria

1. THE System SHALL pin Go to version 1.26 and Wails to version `v3.0.0-beta.28` in both `mise.toml` and `go.mod`.
2. THE System SHALL declare bun as the frontend package manager in `mise.toml`.
3. WHERE the Wails version is changed, THE System SHALL record that change in a commit that modifies no application feature code.

### Requirement 24: Coverage gate and end-to-end happy path

**User Story:** As a pentest operator, I want the tested backend to meet a coverage bar and one end-to-end path to run against the server build, so that I can trust the app before I depend on it in an exam.

#### Acceptance Criteria

1. THE System SHALL achieve at least 80% line coverage across the packages under `internal/`.
2. IF line coverage for the packages under `internal/` falls below 80%, THEN THE System SHALL fail the test run and report the measured coverage percentage.
3. WHEN the Playwright happy-path test runs against the Wails server build, THE System SHALL create a `lab` Engagement, import the nmap fixture, open the Host, tick one checklist item from `todo` to `done`, add a Credential, and assert that the persisted `host.json` and `creds.json` reflect each of those changes.
4. THE Store and Importer test suites SHALL pass with the Go race detector enabled.
