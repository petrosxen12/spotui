package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"

	"github.com/petrosxen/spotui/internal/config"
	spotifyapi "github.com/petrosxen/spotui/internal/spotify"
	"github.com/petrosxen/spotui/internal/spotifyd"
)

func TestBestDeviceMatchPrefersFuzzyFallback(t *testing.T) {
	devices := []Device{
		{Name: "Kitchen Speaker", ID: "kitchen"},
		{Name: "Desk Headphones", ID: "desk"},
	}

	match, err := bestDeviceMatch(devices, "kithn")
	if err != nil {
		t.Fatalf("bestDeviceMatch returned error: %v", err)
	}
	if match.ID != "kitchen" {
		t.Fatalf("expected kitchen device, got %q", match.ID)
	}
}

func TestBestDeviceMatchRejectsAmbiguousSubstring(t *testing.T) {
	devices := []Device{
		{Name: "Living Room", ID: "one"},
		{Name: "Living Room TV", ID: "two"},
	}

	_, err := bestDeviceMatch(devices, "living")
	if err == nil {
		t.Fatal("expected ambiguous match error")
	}
}

func TestFindDeviceByID(t *testing.T) {
	devices := []Device{{ID: "abc", Name: "Office"}}
	match, ok := findDeviceByID(devices, "abc")
	if !ok {
		t.Fatal("expected device to be found")
	}
	if match.Name != "Office" {
		t.Fatalf("unexpected device %+v", match)
	}
}

func TestGetPlaybackStateReusesQueueWhileTrackAndDeviceAreUnchanged(t *testing.T) {
	cfg := testConfig(t)

	var mu sync.Mutex
	playerCalls := 0
	queueCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch r.URL.Path {
		case "/v1/me/player":
			playerCalls++
			writeJSON(t, w, map[string]any{
				"device": map[string]any{
					"id":        "device-1",
					"name":      "Desk Speakers",
					"type":      "Computer",
					"is_active": true,
				},
				"is_playing":             true,
				"progress_ms":            1000,
				"currently_playing_type": "track",
				"context":                map[string]any{"uri": "spotify:playlist:mix"},
				"item": map[string]any{
					"name":        "Track One",
					"uri":         "spotify:track:one",
					"duration_ms": 200000,
					"artists":     []map[string]any{{"name": "Artist One"}},
					"album":       map[string]any{"images": []map[string]any{{"url": "https://img/one"}}},
				},
			})
		case "/v1/me/player/queue":
			queueCalls++
			writeJSON(t, w, map[string]any{
				"currently_playing": map[string]any{"name": "Track One", "uri": "spotify:track:one"},
				"queue": []map[string]any{{
					"name":    "Track Two",
					"uri":     "spotify:track:two",
					"artists": []map[string]any{{"name": "Artist Two"}},
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := &Service{
		cfg:    cfg,
		client: spotifyapi.NewClient(cfg, staticTokenSource{}),
		local:  stubLocalPlayerManager{},
	}

	restoreTransport := rewriteSpotifyAPI(t, server.URL)
	defer restoreTransport()

	first, err := service.GetPlaybackState(context.Background())
	if err != nil {
		t.Fatalf("first GetPlaybackState() error = %v", err)
	}
	second, err := service.GetPlaybackState(context.Background())
	if err != nil {
		t.Fatalf("second GetPlaybackState() error = %v", err)
	}

	if first.NextItemName != "Track Two" || first.NextArtistName != "Artist Two" {
		t.Fatalf("first playback next item = %q / %q, want Track Two / Artist Two", first.NextItemName, first.NextArtistName)
	}
	if second.NextItemName != "Track Two" || second.NextArtistName != "Artist Two" {
		t.Fatalf("second playback next item = %q / %q, want cached Track Two / Artist Two", second.NextItemName, second.NextArtistName)
	}

	mu.Lock()
	defer mu.Unlock()
	if playerCalls != 2 {
		t.Fatalf("/me/player calls = %d, want 2", playerCalls)
	}
	if queueCalls != 1 {
		t.Fatalf("/me/player/queue calls = %d, want 1", queueCalls)
	}
}

func TestGetPlaybackStateRefetchesQueueWhenTrackChanges(t *testing.T) {
	cfg := testConfig(t)

	var mu sync.Mutex
	playerCalls := 0
	queueCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch r.URL.Path {
		case "/v1/me/player":
			playerCalls++
			trackURI := "spotify:track:one"
			trackName := "Track One"
			nextName := "Track Two"
			nextArtist := "Artist Two"
			if playerCalls > 1 {
				trackURI = "spotify:track:three"
				trackName = "Track Three"
				nextName = "Track Four"
				nextArtist = "Artist Four"
			}
			writeJSON(t, w, map[string]any{
				"device": map[string]any{
					"id":        "device-1",
					"name":      "Desk Speakers",
					"type":      "Computer",
					"is_active": true,
				},
				"is_playing":             true,
				"progress_ms":            1000,
				"currently_playing_type": "track",
				"context":                map[string]any{"uri": "spotify:playlist:mix"},
				"item": map[string]any{
					"name":        trackName,
					"uri":         trackURI,
					"duration_ms": 200000,
					"artists":     []map[string]any{{"name": "Artist One"}},
					"album":       map[string]any{"images": []map[string]any{{"url": "https://img/one"}}},
				},
			})
			_ = nextName
			_ = nextArtist
		case "/v1/me/player/queue":
			queueCalls++
			nextName := "Track Two"
			nextArtist := "Artist Two"
			if queueCalls > 1 {
				nextName = "Track Four"
				nextArtist = "Artist Four"
			}
			writeJSON(t, w, map[string]any{
				"currently_playing": map[string]any{"name": "Current", "uri": "spotify:track:current"},
				"queue": []map[string]any{{
					"name":    nextName,
					"uri":     "spotify:track:next",
					"artists": []map[string]any{{"name": nextArtist}},
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := &Service{
		cfg:    cfg,
		client: spotifyapi.NewClient(cfg, staticTokenSource{}),
		local:  stubLocalPlayerManager{},
	}

	restoreTransport := rewriteSpotifyAPI(t, server.URL)
	defer restoreTransport()

	first, err := service.GetPlaybackState(context.Background())
	if err != nil {
		t.Fatalf("first GetPlaybackState() error = %v", err)
	}
	second, err := service.GetPlaybackState(context.Background())
	if err != nil {
		t.Fatalf("second GetPlaybackState() error = %v", err)
	}

	if first.NextItemName != "Track Two" {
		t.Fatalf("first NextItemName = %q, want Track Two", first.NextItemName)
	}
	if second.NextItemName != "Track Four" {
		t.Fatalf("second NextItemName = %q, want Track Four", second.NextItemName)
	}

	mu.Lock()
	defer mu.Unlock()
	if playerCalls != 2 {
		t.Fatalf("/me/player calls = %d, want 2", playerCalls)
	}
	if queueCalls != 2 {
		t.Fatalf("/me/player/queue calls = %d, want 2", queueCalls)
	}
}

func TestGetCurrentTrackDetailsFetchesAudioFeatures(t *testing.T) {
	cfg := testConfig(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/me/player":
			writeJSON(t, w, map[string]any{
				"device": map[string]any{
					"id":        "device-1",
					"name":      "Desk Speakers",
					"type":      "Computer",
					"is_active": true,
				},
				"is_playing":             true,
				"progress_ms":            42000,
				"currently_playing_type": "track",
				"context":                map[string]any{"uri": "spotify:playlist:mix"},
				"item": map[string]any{
					"name":        "Track One",
					"uri":         "spotify:track:one",
					"duration_ms": 200000,
					"explicit":    true,
					"popularity":  87,
					"artists":     []map[string]any{{"name": "Artist One"}},
					"album": map[string]any{
						"name":   "Album One",
						"images": []map[string]any{{"url": "https://img/one"}},
					},
				},
			})
		case "/v1/audio-features/one":
			writeJSON(t, w, map[string]any{
				"danceability":     0.81,
				"energy":           0.72,
				"valence":          0.64,
				"acousticness":     0.12,
				"instrumentalness": 0.01,
				"liveness":         0.09,
				"speechiness":      0.05,
				"tempo":            123.9,
				"key":              9,
				"mode":             1,
				"time_signature":   4,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := &Service{
		cfg:    cfg,
		client: spotifyapi.NewClient(cfg, staticTokenSource{}),
		local:  stubLocalPlayerManager{},
	}

	restoreTransport := rewriteSpotifyAPI(t, server.URL)
	defer restoreTransport()

	details, err := service.GetCurrentTrackDetails(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentTrackDetails() error = %v", err)
	}
	if details.Title != "Track One" {
		t.Fatalf("Title = %q, want Track One", details.Title)
	}
	if details.Album != "Album One" {
		t.Fatalf("Album = %q, want Album One", details.Album)
	}
	if details.Artists != "Artist One" {
		t.Fatalf("Artists = %q, want Artist One", details.Artists)
	}
	if details.DeviceName != "Desk Speakers" {
		t.Fatalf("DeviceName = %q, want Desk Speakers", details.DeviceName)
	}
	if details.Danceability != 0.81 {
		t.Fatalf("Danceability = %v, want 0.81", details.Danceability)
	}
	if details.Popularity != 87 {
		t.Fatalf("Popularity = %d, want 87", details.Popularity)
	}
	if !details.Explicit {
		t.Fatal("expected Explicit to be true")
	}
	if !details.AudioFeaturesAvailable {
		t.Fatal("expected AudioFeaturesAvailable to be true")
	}
}

func TestGetCurrentTrackDetailsFallsBackWhenAudioFeaturesForbidden(t *testing.T) {
	cfg := testConfig(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/me/player":
			writeJSON(t, w, map[string]any{
				"device": map[string]any{
					"id":        "device-1",
					"name":      "Desk Speakers",
					"type":      "Computer",
					"is_active": true,
				},
				"is_playing":             true,
				"progress_ms":            42000,
				"currently_playing_type": "track",
				"context":                map[string]any{"uri": "spotify:playlist:mix"},
				"item": map[string]any{
					"name":        "Track One",
					"uri":         "spotify:track:one",
					"duration_ms": 200000,
					"explicit":    true,
					"popularity":  87,
					"artists":     []map[string]any{{"name": "Artist One"}},
					"album": map[string]any{
						"name":   "Album One",
						"images": []map[string]any{{"url": "https://img/one"}},
					},
				},
			})
		case "/v1/audio-features/one":
			w.WriteHeader(http.StatusForbidden)
			writeJSON(t, w, map[string]any{
				"error": map[string]any{
					"status":  http.StatusForbidden,
					"message": "audio features access restricted",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := &Service{
		cfg:    cfg,
		client: spotifyapi.NewClient(cfg, staticTokenSource{}),
		local:  stubLocalPlayerManager{},
	}

	restoreTransport := rewriteSpotifyAPI(t, server.URL)
	defer restoreTransport()

	details, err := service.GetCurrentTrackDetails(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentTrackDetails() error = %v", err)
	}
	if details.Title != "Track One" {
		t.Fatalf("Title = %q, want Track One", details.Title)
	}
	if details.AudioFeaturesAvailable {
		t.Fatal("expected AudioFeaturesAvailable to be false")
	}
	if details.AudioFeaturesNote == "" {
		t.Fatal("expected AudioFeaturesNote to be set")
	}
}

type staticTokenSource struct{}

func (staticTokenSource) ValidAccessToken(context.Context) (string, error) {
	return "test-token", nil
}

type stubLocalPlayerManager struct{}

func (stubLocalPlayerManager) Status(context.Context) (spotifyd.Status, error) {
	panic("unexpected Status call")
}

func (stubLocalPlayerManager) Start(context.Context) (spotifyd.Status, error) {
	panic("unexpected Start call")
}

func (stubLocalPlayerManager) Stop(context.Context) error {
	panic("unexpected Stop call")
}

func (stubLocalPlayerManager) Reset(context.Context) error {
	panic("unexpected Reset call")
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()

	dir := t.TempDir()
	if err := os.Setenv("XDG_CONFIG_HOME", dir); err != nil {
		t.Fatalf("set XDG_CONFIG_HOME: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("XDG_CONFIG_HOME")
	})

	return &config.Config{
		ClientID:    "test-client",
		RedirectURI: config.DefaultRedirectURI,
		LocalPlayer: config.DefaultLocalPlayerConfig(),
	}
}

func rewriteSpotifyAPI(t *testing.T, rawURL string) func() {
	t.Helper()

	serverURL, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}

	previous := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		clone := req.Clone(req.Context())
		if clone.URL.Host == "api.spotify.com" && clone.URL.Scheme == "https" {
			clone.URL.Scheme = serverURL.Scheme
			clone.URL.Host = serverURL.Host
		}
		return previous.RoundTrip(clone)
	})

	return func() {
		http.DefaultTransport = previous
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode JSON response: %v", err)
	}
}
