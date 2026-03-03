# spotui Architecture

## Overview
spotui is a minimal Spotify controller with a terminal user interface (TUI) built with Bubble Tea. It controls existing Spotify Connect devices via the Spotify Web API.

## Core Assumptions & Constraints

### 1. Playback Model
- **External playback only**: spotui does NOT stream audio. It sends commands to existing Spotify Connect devices (desktop app, mobile, web player, smart speakers).
- **Active device requirement**: User must have an active Spotify Connect device running. If no device is available, playback commands will fail gracefully.
- **Premium requirement**: Most playback endpoints (`/me/player/*`) require Spotify Premium. Non-premium accounts will receive clear error messages.

### 2. Authentication
- **OAuth 2.0 Authorization Code Flow with PKCE**: No client secret required, improving security for native/desktop apps.
- **Scopes required**:
  - `user-read-playback-state` - read current playback
  - `user-modify-playback-state` - control playback (play, pause, skip, volume)
  - `user-read-currently-playing` - get currently playing track
  - `user-library-read` - access saved tracks (future)
- **Token storage**: Access and refresh tokens stored in `~/.config/spotui/tokens.json` with `0600` permissions.
- **Token refresh**: Automatic refresh when access token expires (1 hour lifetime).

### 3. Security Considerations
- No client secret in code (PKCE handles this)
- Token file restricted to user-only read/write
- Redirect to `http://localhost:8888/callback` during auth flow
- State parameter for CSRF protection

### 4. Error Handling
- **403 Forbidden**: Premium required - display clear message
- **404 Not Found**: No active device - prompt user to start Spotify
- **401 Unauthorized**: Token expired - auto-refresh
- **429 Rate Limited**: Exponential backoff
- **Network errors**: Retry with timeout

## Project Structure

```
spotui/
├── cmd/
│   └── spotui/
│       └── main.go              # Entry point, CLI argument parsing
│
├── internal/
│   ├── auth/
│   │   ├── oauth.go            # PKCE flow implementation
│   │   ├── token.go            # Token persistence and refresh
│   │   └── server.go           # Callback server for OAuth
│   │
│   ├── spotify/
│   │   ├── client.go           # API client with auth handling
│   │   ├── player.go           # Playback endpoints (play, pause, skip, volume)
│   │   ├── devices.go          # List/select devices
│   │   ├── tracks.go           # Track/album/artist info
│   │   └── types.go            # API response structs
│   │
│   ├── config/
│   │   ├── config.go           # Load/save app configuration
│   │   └── paths.go            # XDG Base Directory paths
│   │
│   ├── app/
│   │   └── service.go          # Business logic layer (future)
│   │
│   └── ui/
│       └── tui.go              # Bubble Tea UI (Phase 2)
│
├── ARCHITECTURE.md
├── PHASE1.md
├── README.md
├── go.mod
└── go.sum
```

### Package Responsibilities

#### `cmd/spotui`
- Parse CLI flags
- Initialize config and auth
- Launch TUI or handle CLI commands

#### `internal/auth`
- Generate PKCE code verifier and challenge
- Start local HTTP server for callback
- Exchange authorization code for tokens
- Persist and refresh tokens
- Provide valid access tokens to API client

#### `internal/spotify`
- HTTP client with automatic token injection
- Spotify Web API endpoints
- Error parsing and wrapping
- Rate limit handling
- Device management

#### `internal/config`
- Read/write `~/.config/spotui/config.toml`
- XDG Base Directory spec compliance
- Client ID, redirect URI, port configuration
- Default values

#### `internal/app`
- Coordinate between Spotify API and UI (Phase 2)
- State management
- Background updates (polling current playback)

#### `internal/ui`
- Bubble Tea components (Phase 2)
- Key bindings
- Display current track, playback controls
- Device selector

## Configuration

### Environment Variables (optional)
```bash
SPOTUI_CLIENT_ID=<spotify-app-client-id>
SPOTUI_REDIRECT_URI=http://localhost:8888/callback
SPOTUI_CALLBACK_PORT=8888
```

### Configuration File: `~/.config/spotui/config.toml`
```toml
[auth]
client_id = "your-spotify-client-id"
redirect_uri = "http://localhost:8888/callback"
callback_port = 8888

[app]
poll_interval_ms = 1000  # How often to refresh playback state
default_device = ""      # Device ID to prefer (optional)
```

### Token File: `~/.config/spotui/tokens.json` (0600)
```json
{
  "access_token": "BQD...",
  "refresh_token": "AQD...",
  "token_type": "Bearer",
  "expires_at": "2026-03-03T10:30:00Z"
}
```

### Required User Setup
1. Create Spotify app at https://developer.spotify.com/dashboard
2. Add redirect URI: `http://localhost:8888/callback`
3. Copy client ID
4. Run `spotui auth` to authenticate
5. Ensure Spotify is running on a device (desktop, mobile, web)

## Phase 1 Scope
See [PHASE1.md](PHASE1.md) for detailed checklist.

**Phase 1 delivers**:
- OAuth PKCE authentication flow
- Token persistence and refresh
- Basic Spotify API client
- Playback control functions (play/pause, skip, volume)
- Device listing
- CLI testing tool (not full TUI)

**Phase 2 (future)**:
- Bubble Tea TUI
- Live playback display
- Interactive controls
- Search and library browsing
