---
name: update-demo
description: Re-record the spotui asciinema TUI demo and sync README.md (Screenshots, Features) after a change that meaningfully alters the TUI's look or behavior. Use when a change adds/renames a slash command, changes a keybinding, changes TUI layout, or adds a new view.
---

# Update demo

Keeps `docs/screenshots/spotui-demo.cast`, `spotui-demo.gif`, and `README.md` truthful after TUI-affecting
changes. Do not run this for backend-only, CLI-only, or non-UI changes — check the diff first.

## 1. Decide if this run is warranted

Skip the rest of this skill if the diff only touches: Spotify API/auth logic, `spotifyd` process management,
non-UI CLI commands, tests, or docs. Proceed if it touches `internal/ui/`, adds/renames a `/` command, or the
README's Features/Usage sections are now stale relative to the code.

## 2. Edit the recording script for whatever changed

The whole recording happens inside a Docker container (Go, tmux, asciinema, and `agg` baked into the image at
`docs/screenshots/docker/Dockerfile`), so there is no dependency on what's installed locally.

Before running it, edit the "drive the TUI" block in `docs/screenshots/docker/record-demo.sh` so the recording
actually exercises whatever changed (new command, new view, etc.) — don't just replay the old script unchanged.

## 3. Record + convert in one step

```bash
task demo:record
```

This builds the `spotui-demo-record` image (cached after the first run), then runs
`docs/screenshots/docker/record-demo.sh` inside it. That script builds `spotui`, scripts the TUI through `tmux` +
`asciinema rec`, and converts the result with `agg`. Both outputs land directly in `docs/screenshots/` on the host
(the repo root is bind-mounted at `/workspace`):

- `docs/screenshots/spotui-demo.cast`
- `docs/screenshots/spotui-demo.gif`

It mounts `~/.config/spotui` read-only so the demo shows a real logged-in session — make sure you're logged in
locally first (`spotui login --client-id "$SPOTUI_CLIENT_ID"`) if the cast should show live Spotify data.

Verify the cast looks reasonable:

```bash
asciinema cat docs/screenshots/spotui-demo.cast | tail -40
```

## 4. Update README.md

- If new commands, views, or keybindings were added/changed, update the `## Features`, `## TUI`, and
  `### Autocomplete` sections to match reality — don't leave stale descriptions.
- Screenshots section stays pointing at `spotui-demo.gif` / `spotui-demo.cast`; only the underlying files change.
- Do not touch `docs/screenshots/tui-overview.svg` / `tui-commands.svg` here — those are hand-curated with live
  Spotify data (see git history), not part of this flow.

## Adaptive-layout (resize) demo

`task demo:resize-record` runs `docs/screenshots/docker/record-demo-resize-showcase.sh` and produces
`docs/screenshots/spotui-resize-demo.cast` / `.gif`, showing spotui reflow live inside a shrinking/growing tmux
split pane alongside another program (`top`). Re-run this specifically when layout/reflow behavior changes
(not for every UI change — most changes only warrant re-running `task demo:record`).

That script uses a different technique from `record-demo.sh`: an inner tmux session holds the actual
split-pane layout, and a separate session records `tmux attach` to it, so the recording captures the whole
pane layout rather than just spotui's own pty. Only ever use `tmux resize-pane` (redistributes columns between
panes within a fixed window size) inside it, never `tmux resize-window` (changes the overall terminal size) —
the installed asciinema version doesn't record live terminal-size changes into the cast, so a whole-window
resize desyncs the recording from what `agg` renders into silently misaligned frames. See the comments in
`record-demo-resize-showcase.sh` for the full explanation.

Also watch out for `Escape` after a search: it closes an open suggestion panel on the first press, but a
second press falls through to `popViewState()` and silently pops back to the pre-search view if one was
already pushed — losing the results for the rest of the recording. Use `Tab` to move focus back onto the
results list instead of a second `Escape`.

## 5. Verify

```bash
go build ./...
go test ./...
```

Confirm the new `.cast` file is non-trivially different from the previous one (`git diff --stat
docs/screenshots/`) before reporting done.
