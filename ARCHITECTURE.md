# spotui Architecture

## Overview
`spotui` now has two front ends: a Cobra CLI and a Bubble Tea TUI. The service layer introduced in Phase 2 remains the boundary that keeps both front ends from reaching into Spotify HTTP details directly.

## Package Boundaries

### `cmd/spotui`
- Parses arguments and prints output.
- Starts either CLI commands or the `tui` subcommand.
- Depends on `internal/app` for behavior and `internal/config` for login/config bootstrap.
- Does not call `internal/spotify` directly.

### `internal/app`
- Owns application workflows such as search, device selection, playback commands, and search-result reference resolution.
- Translates raw Spotify responses into app-facing types used by CLI and future TUI code.
- Owns lightweight caching:
  - last search is kept in memory and persisted in `config.json`
  - playlist list is cached in memory for 60 seconds
- Exposes playback-state data in a UI-friendly shape so the TUI can poll one service method instead of assembling state from multiple API calls.

### `internal/ui`
- Contains the Bubble Tea model and view code.
- Renders the status bar, search-result list, and query input.
- Polls playback state on an adaptive interval:
  - faster while music is actively playing
  - slower while paused
  - slowest when no active device is available

### `internal/spotify`
- Thin Spotify Web API client.
- Knows about HTTP paths, request/response payloads, and Spotify-specific error mapping.
- Does not contain CLI formatting or config-persistence decisions.

### `internal/auth`
- Handles OAuth PKCE login, token persistence, and token refresh.

### `internal/config`
- Loads and saves local configuration under `~/.config/spotui/`.
- Stores stable user preferences and the persisted last-search cache.

## Why This Prevents Spaghetti

Before Phase 2, `cmd/spotui` handled argument parsing, device matching, last-search persistence, search-index resolution, and direct Spotify client calls in one file. That works for a small CLI, but it becomes brittle as soon as a second UI exists.

With the new boundary, both the CLI and a future Bubble Tea TUI can call the same `internal/app.PlayerService` methods. That keeps:

- transport logic in `internal/spotify`
- workflow and caching logic in `internal/app`
- presentation logic in `cmd/spotui` or future `internal/ui`

This matters because the TUI needs the same actions as the CLI plus polling and richer state. Reusing the service avoids duplicating device-selection rules, search caching, playback behavior, and reference-resolution logic across two front ends.

## Current Data Flow

1. `cmd/spotui` loads config and starts either a CLI command or `internal/ui`.
2. `internal/app` creates the auth manager and Spotify client, then executes the requested workflow.
3. `internal/spotify` performs HTTP requests against the Spotify Web API.
4. `internal/app` updates caches or persisted config when needed and returns app-facing types.
5. `cmd/spotui` or `internal/ui` renders the result.

## Polling Strategy

The TUI only polls playback state, and it adapts the interval based on the latest known state:

- active playback: about every 1.5 seconds
- paused playback: every 4 seconds
- no active device: every 6 seconds

That keeps the status bar responsive without hammering Spotify when there is nothing changing.
