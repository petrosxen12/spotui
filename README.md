# spotui Phase 1

`spotui` is a CLI-only Spotify controller for Phase 1. It authenticates with Spotify using Authorization Code with PKCE, stores tokens locally, searches tracks and playlists, lists devices, selects a preferred device, and sends playback commands through the Spotify Web API.

## Build

Requirements:

- Go 1.22+
- A Spotify Premium account for playback control commands
- A Spotify app client ID from the Spotify developer dashboard

Build the CLI:

```bash
go build -o spotui ./cmd/spotui
```

## Spotify App Setup

1. Open the Spotify Developer Dashboard: <https://developer.spotify.com/dashboard>
2. Create an app.
3. Copy the app's Client ID.
4. In the app settings, add this redirect URI:

```text
http://127.0.0.1:8888/callback
```

5. Save the settings.

You can keep the default redirect URI above, or pass a different one to `spotui login --redirect-uri ...` as long as it points to `127.0.0.1` or `localhost` and is also registered in the Spotify app settings.

## Configuration

Config files are stored in `~/.config/spotui/`:

- `config.json`: client ID, redirect URI, preferred device, and last search cache
- `token.json`: access and refresh tokens, written with file mode `0600`

Environment variables:

```bash
export SPOTUI_CLIENT_ID=your_spotify_client_id
export SPOTUI_REDIRECT_URI=http://127.0.0.1:8888/callback
```

## Commands

Authenticate:

```bash
spotui login --client-id "$SPOTUI_CLIENT_ID"
```

Inspect the authenticated account:

```bash
spotui me
```

List devices:

```bash
spotui devices
```

Set the preferred device by substring match:

```bash
spotui use kitchen
```

Search tracks and playlists:

```bash
spotui search "daft punk"
```

For `spotui play`, the final argument can be one of:

- a raw Spotify ID
- a full Spotify URI
- a numeric index from the most recent `spotui search` results for that same item type

Track forms:

```text
spotui play track <spotify-track-id>
spotui play track <spotify-track-uri>
spotui play track <last-search-track-index>
```

Track examples:

```bash
spotui play track 11dFghVXANMlKmJXsNCbNl
spotui play track spotify:track:11dFghVXANMlKmJXsNCbNl
spotui play track 3
```

Playlist forms:

```text
spotui play playlist <spotify-playlist-id>
spotui play playlist <spotify-playlist-uri>
spotui play playlist <last-search-playlist-index>
```

Playlist examples:

```bash
spotui play playlist 37i9dQZF1DXcBWIGoYBM5M
spotui play playlist spotify:playlist:37i9dQZF1DXcBWIGoYBM5M
spotui play playlist 1
```

Playback controls:

```bash
spotui pause
spotui resume
spotui next
spotui prev
```

## Example Output

`spotui devices`:

```text
74c1c2f4b9d9f55f9f3d2f6a9d4c2b1e4f5a6789	MacBook Pro Speakers	Computer	active
98f18ab7b6f21d0013a987c5fe5a0a3ff0bcb123	Kitchen Nest Hub	Speaker	inactive
```

`spotui search "daft punk"`:

```text
Tracks for "daft punk":
  [1] spotify:track:2cGxRwrMyEAp8dEbuZaVv6 | 2cGxRwrMyEAp8dEbuZaVv6 | One More Time
  [2] spotify:track:0DiWol3AO6WpXZgp0goxAV | 0DiWol3AO6WpXZgp0goxAV | Get Lucky
Playlists for "daft punk":
  [1] spotify:playlist:37i9dQZF1DX1g0iEXLFycr | 37i9dQZF1DX1g0iEXLFycr | This Is Daft Punk
  [2] spotify:playlist:3C4j8T2L8s3vk0S2lQkR4n | 3C4j8T2L8s3vk0S2lQkR4n | Daft Punk Radio
```

In the examples above, `spotui play track 3` means "play the third track from the most recent search results", while `spotui play playlist 1` means "play the first playlist from the most recent search results".

## Notes

- Playback commands require Spotify Premium. Spotify typically returns HTTP 403 otherwise.
- If no active or preferred device exists, playback commands fail with a clear error instead of guessing.
- Phase 1 does not poll for current playback state.
