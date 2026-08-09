---
name: update-architecture
description: Sync ARCHITECTURE.md's prose and Mermaid diagram with the code after a change that adds/removes/renames a package, changes which packages depend on each other, or changes the runtime/local-player flow. Use when a package boundary, dependency direction, or top-level workflow described in ARCHITECTURE.md is now stale relative to the code.
---

# Update architecture

Keeps `ARCHITECTURE.md` (package boundaries, runtime flow, and the Mermaid diagram under `## Diagram`)
truthful after structural changes. Do not run this for behavior-only changes inside a package that don't
touch its boundary or dependencies — check the diff first.

## 1. Decide if this run is warranted

Skip the rest of this skill if the diff only changes logic *inside* an existing package without changing
what it imports, what imports it, or the workflows in `## Runtime Flow` / `## Local Player Flow`. Proceed if
the diff adds/removes/renames a top-level package under `cmd/` or `internal/`, changes an import between
packages listed in `ARCHITECTURE.md`, or changes one of the documented flows.

## 2. Update the prose sections

- Add/remove/rename the affected `### package` subsection under `## Package Boundaries`.
- Update `## Runtime Flow` and/or `## Local Player Flow` if the sequence of calls changed.
- Keep descriptions at the same level of detail as neighboring sections — responsibilities and boundaries,
  not implementation detail.

## 3. Update the Mermaid diagram

Edit the ` ```mermaid ` block under `## Diagram` to match:

- Node per package that has its own `###` subsection (plus external systems it talks to: Spotify Web API,
  the `spotifyd` process).
- Solid arrow (`-->`) for a direct call/import dependency; dotted arrow (`-.->`) for a cross-cutting
  dependency like shared error types (`internal/spoterr`).
- Arrow direction follows the caller — the importing package points at the package it imports, matching the
  "Does not call X directly" / "Uses X" statements in the prose.

## 4. Verify

Render the diagram to catch syntax errors before committing — either paste the block into a Mermaid live
editor, or if `mmdc` (mermaid-cli) is installed locally:

```bash
mmdc -i ARCHITECTURE.md -o /tmp/architecture-check.svg 2>&1 | head -20
```

Re-read `ARCHITECTURE.md` end to end and confirm every package subsection has a corresponding diagram node,
and every diagram edge matches a stated dependency in the prose.
