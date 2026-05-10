# spotui

Terminal Spotify controller with a CLI and Bubble Tea TUI, plus optional lightweight local playback via `spotifyd`.

## Screenshots

![spotui TUI demo](./docs/screenshots/spotui-demo.gif)
![spotui search results](./docs/screenshots/tui-overview.svg)
![spotui devices view](./docs/screenshots/tui-commands.svg)

Raw terminal recording: [spotui-demo.cast](./docs/screenshots/spotui-demo.cast)

## QA Review Bundle

Generate a deterministic multi-layout TUI review bundle, separate from the main `spotui` binary:

```bash
task qa:tui-review
```

This writes review assets to `docs/qa/tui-review/`, including:

- SVG screenshot-style renders for several terminal sizes
- plain-text captures of each rendered TUI state
- a `review-brief.md` prompt for AI or human QA review
- a `manifest.json` index of scenarios and files

Use this as a quality-control pass for visual polish, density, command clarity, and compact-layout behavior.

## Features

- CLI and TUI in one binary
- Spotify Authorization Code with PKCE
- Search tracks and playlists
- Device listing and preferred-device selection
- Optional managed local playback with `spotifyd`
- Fuzzy matching for `/device` and `/play`
- Inline autocomplete and ghost completion
- Polling backoff for no-device, network, and rate-limit states
- Clear error messages for auth expiry, Premium requirements, and connectivity issues

## Requirements

- Go 1.24.2+
- Spotify app client ID
- Spotify Premium for playback control
- `spotifyd` on Linux for local playback without the full Spotify desktop app

### Recommended `spotifyd` Install

For Linux desktop setups, prefer the upstream `spotifyd` release binary over the Homebrew build.

Why:

- some Homebrew Linux builds expose a reduced backend set
- PipeWire/PulseAudio desktop systems work better with a full upstream build
- `spotui` can point at any `spotifyd` binary via `local_player.spotifyd_path`

Typical Ubuntu/PipeWire setup:

- install the upstream `spotifyd` `linux-x86_64-full` release
- place it somewhere stable such as `~/.local/bin/spotifyd`
- set `local_player.spotifyd_path` in `~/.config/spotui/config.json`

## Quick Start

1. Create a Spotify app at <https://developer.spotify.com/dashboard>
2. Add this redirect URI:

```text
http://127.0.0.1:8888/callback
```

3. Export your client ID:

```bash
export SPOTUI_CLIENT_ID=your_spotify_client_id
```

4. Build and log in:

```bash
go build -o spotui ./cmd/spotui
./spotui login --client-id "$SPOTUI_CLIENT_ID"
./spotui tui
```

## Install

```bash
go build -o spotui ./cmd/spotui
```

Or:

```bash
task build
```

## Usage

Use the built-in help for the full command reference:

```bash
spotui --help
spotui [command] --help
```

Typical flow:

```bash
spotui login --client-id "$SPOTUI_CLIENT_ID"
spotui devices
spotui search "daft punk"
spotui play track 3
spotui tui
```

Use `spotui use <name>` to set a preferred device by substring match, for example `spotui use kitchen`.

### Playing From Search Results

`spotui play` accepts:

- a Spotify ID
- a Spotify URI
- an index from the most recent `spotui search`

```bash
spotui play track 11dFghVXANMlKmJXsNCbNl
spotui play track spotify:track:11dFghVXANMlKmJXsNCbNl
spotui play track 3

spotui play playlist 37i9dQZF1DXcBWIGoYBM5M
spotui play playlist spotify:playlist:37i9dQZF1DXcBWIGoYBM5M
spotui play playlist 1
```

### Local Playback

`spotui` can manage a local `spotifyd` daemon on Linux and select it as the preferred Spotify Connect device.

```bash
spotui local status
spotui local start
spotui local use
spotui local reset
spotui local stop
```

Recommended local-player config shape:

```json
{
  "local_player": {
    "backend": "pulseaudio",
    "spotifyd_path": "/home/you/.local/bin/spotifyd"
  }
}
```

On PipeWire desktops, leave `audio_device` empty unless you specifically need to force a sink.

## TUI

Run `spotui tui` for the interactive interface.

Inside the TUI:

- type a search query and press `Enter`
- press `/help` for the slash-command list
- use `Tab` and inline suggestions to complete commands quickly
- use `q` or `Ctrl+C` to quit

### Autocomplete

- Suggestions appear for `/` commands
- Best match is shown inline as ghost completion
- `/local` expands to local-player subcommands such as `start`, `stop`, `use`, `status`, and `reset`
- `/device` uses known Spotify devices
- `/play` uses the latest TUI search results

## Config

Stored in `~/.config/spotui/`:

- `config.json`: client ID, redirect URI, preferred device, local-player settings, last-used device, last search cache
- `token.json`: Spotify access and refresh tokens
- managed local-player runtime files: generated `spotifyd` config, PID, and log files

Environment variables:

```bash
export SPOTUI_CLIENT_ID=your_spotify_client_id
export SPOTUI_REDIRECT_URI=http://127.0.0.1:8888/callback
```

### Token Storage

- Config dir mode `0700`
- Token file mode `0600`
- Atomic temp-file-and-rename writes

## Troubleshooting

### No active device

Start Spotify on a desktop, mobile, or web player, then run `spotui devices` or use `/devices`.

If `spotifyd` is installed, run:

```bash
spotui local use
```

If local-player runtime state is stuck or pointing at the wrong `spotifyd` process, reset it with:

```bash
spotui local reset
```

### `spotifyd` exits during startup

`spotui` now surfaces the tail of the managed `spotifyd` log on startup failure.

Common causes:

- unsupported backend in the installed `spotifyd` build
- invalid `audio_device`
- local audio stack mismatch

If needed, inspect the full log directly:

```bash
tail -n 100 ~/.config/spotui/runtime/spotifyd/spotifyd.log
```

### Linux audio backend issues

If you are on a PipeWire desktop and local playback is unstable:

- prefer an upstream `spotifyd` full build
- use `backend = "pulseaudio"` when that build supports it
- avoid forcing `audio_device` until the default path works

If you see PortAudio-specific device errors, the installed `spotifyd` build is often the problem rather than `spotui`.

### Login expired

```bash
spotui login --client-id "$SPOTUI_CLIENT_ID"
```

### Premium required

Playback control requires Spotify Premium.

### Rate limited

`spotui` backs off automatically on `429`.

## Development

```bash
go test ./...
```
