# spotui Architecture

## Overview
`spotui` has two front ends:

- a Cobra CLI in `cmd/spotui`
- a Bubble Tea TUI launched by `spotui tui`

Both use the same service layer in `internal/app`. That layer owns playback workflows, device selection, search-result resolution, and local-player orchestration.

## Package Boundaries

### `cmd/spotui`
- Defines the CLI surface.
- Loads config, constructs `app.PlayerService`, and prints results.
- Exposes login, search, playback, device-selection, and local-player commands.
- Does not call `internal/spotify` directly.

### `internal/app`
- Defines the `PlayerService` interface used by the CLI and TUI.
- Owns application workflows:
  - search
  - playback commands
  - playback-state aggregation
  - device selection by id or fuzzy name match
  - reference resolution for last-search indices and Spotify ids/uris
  - local-player control through `spotifyd`
- Maps Spotify responses into app-facing types.
- Owns lightweight state and caching:
  - last search is persisted in `config.json`
  - playlist lists are cached in memory for 60 seconds
  - last-used device metadata is persisted after selection or observed playback
- Resolves playback devices in this order:
  - preferred device id
  - last used device id
  - fuzzy match on last used device name
  - currently active device

### `internal/ui`
- Contains the Bubble Tea model, commands, messages, layout, and rendering code.
- Renders playback state, search results, devices, help, and local-player status.
- Uses `app.PlayerService` instead of calling auth or Spotify transport layers directly.
- Polls playback state on an adaptive interval:
  - active playback: about 1.5 seconds
  - paused playback: 4 seconds
  - no active device: 6 seconds

### `internal/spotify`
- Thin Spotify Web API client.
- Owns HTTP transport, request paths, JSON payloads, timeouts, and Spotify-specific error mapping.
- Uses a token source interface provided by `internal/auth`.

### `internal/auth`
- Handles Authorization Code with PKCE login.
- Runs the loopback callback server.
- Loads, saves, validates, and refreshes tokens.

### `internal/config`
- Loads and saves config under `~/.config/spotui/` or `XDG_CONFIG_HOME`.
- Stores Spotify client settings, device preferences, persisted last-search results, and local-player config.
- Provides secure file-writing helpers and runtime/token paths.

### `internal/spotifyd`
- Manages the optional local `spotifyd` process.
- Resolves the binary, generates runtime config, starts and stops the process, and tracks runtime metadata.
- Uses runtime files under `~/.config/spotui/runtime/spotifyd/`.
- Keeps a small package facade in `manager.go`; config, runtime-state, process inspection, status assembly, and startup/log helpers live in focused package-local files.

### `internal/spoterr`
- Defines typed errors for auth expiry, no active device, rate limiting, premium-required, and network failures.
- Keeps error handling consistent across the Spotify client, service layer, and TUI.

## Runtime Flow

1. `cmd/spotui` loads config and dispatches a CLI command or launches the TUI.
2. `internal/app.NewPlayerService` wires together config, auth, the Spotify API client, and the `spotifyd` manager.
3. The CLI and TUI call `PlayerService` methods.
4. `internal/app` applies workflow rules, cache lookups, and persisted-state updates.
5. `internal/spotify` performs Spotify Web API requests using tokens from `internal/auth`.
6. Results are mapped into app-facing types and rendered by the CLI or TUI.

## Local Player Flow

1. `spotui local ...` or TUI slash commands call local-player methods on `PlayerService`.
2. `internal/app` delegates process control to `internal/spotifyd`.
3. After startup, `internal/app` polls Spotify devices until the configured local device appears.
4. Once found, that device becomes the preferred playback target.
