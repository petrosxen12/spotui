package ui

import (
	"context"
	"testing"
	"time"

	"github.com/petrosxen/spotui/internal/app"
	"github.com/petrosxen/spotui/internal/spoterr"
)

type stubPlayerService struct {
	listDevices func(context.Context) ([]app.Device, error)
}

func (s stubPlayerService) CurrentUser(context.Context) (app.User, error) {
	return app.User{}, nil
}

func (s stubPlayerService) Search(context.Context, string) (app.Results, error) {
	return app.Results{}, nil
}

func (s stubPlayerService) PlayTrack(context.Context, string) error {
	return nil
}

func (s stubPlayerService) PlayPlaylist(context.Context, string) error {
	return nil
}

func (s stubPlayerService) Pause(context.Context) error {
	return nil
}

func (s stubPlayerService) Resume(context.Context) error {
	return nil
}

func (s stubPlayerService) Next(context.Context) error {
	return nil
}

func (s stubPlayerService) Prev(context.Context) error {
	return nil
}

func (s stubPlayerService) GetPlaybackState(context.Context) (app.PlaybackState, error) {
	return app.PlaybackState{}, nil
}

func (s stubPlayerService) ListDevices(ctx context.Context) ([]app.Device, error) {
	if s.listDevices != nil {
		return s.listDevices(ctx)
	}
	return nil, nil
}

func (s stubPlayerService) ListPlaylists(context.Context) ([]app.Playlist, error) {
	return nil, nil
}

func (s stubPlayerService) SetDeviceByID(context.Context, string) error {
	return nil
}

func (s stubPlayerService) SetDeviceByName(context.Context, string) (app.Device, error) {
	return app.Device{}, nil
}

func TestBuildSuggestionsUsesCachedDevices(t *testing.T) {
	m := model{
		deviceCache: []app.Device{
			{Name: "Kitchen Speaker", Type: "Speaker"},
			{Name: "Desk Headphones", Type: "Computer"},
		},
		deviceCacheReady: true,
	}

	suggestions := m.buildSuggestions("/device ki")
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}
	if suggestions[0].insertValue != "/device Kitchen Speaker" {
		t.Fatalf("unexpected insert value %q", suggestions[0].insertValue)
	}
	if suggestions[0].description != "speaker" {
		t.Fatalf("unexpected description %q", suggestions[0].description)
	}
}

func TestRefreshSuggestionsSchedulesDeviceCacheFetchOnce(t *testing.T) {
	calls := 0
	m := newModel(stubPlayerService{
		listDevices: func(context.Context) ([]app.Device, error) {
			calls++
			return []app.Device{{Name: "Kitchen Speaker", Type: "Speaker"}}, nil
		},
	})
	m.input.SetValue("/device ki")

	cmd := m.refreshSuggestions()
	if cmd == nil {
		t.Fatal("expected device cache refresh command")
	}
	if !m.deviceCacheBusy {
		t.Fatal("expected device cache busy flag to be set")
	}
	if m.suggestionsOpen {
		t.Fatal("did not expect suggestions to open before cache is loaded")
	}

	if next := m.refreshSuggestions(); next != nil {
		t.Fatal("expected refresh to be suppressed while cache is in flight")
	}

	msg := cmd()
	cacheMsg, ok := msg.(deviceCacheMsg)
	if !ok {
		t.Fatalf("expected deviceCacheMsg, got %T", msg)
	}
	if len(cacheMsg.devices) != 1 || cacheMsg.devices[0].Name != "Kitchen Speaker" {
		t.Fatalf("unexpected cache contents: %+v", cacheMsg.devices)
	}
	if calls != 1 {
		t.Fatalf("expected 1 device fetch, got %d", calls)
	}
}

func TestDeviceCacheMsgRebuildsSuggestionsFromCache(t *testing.T) {
	m := newModel(stubPlayerService{})
	m.input.SetValue("/device ki")

	updated, cmd := m.Update(deviceCacheMsg{
		devices: []app.Device{{Name: "Kitchen Speaker", Type: "Speaker"}},
	})
	nextModel, ok := updated.(model)
	if !ok {
		t.Fatalf("expected model, got %T", updated)
	}
	if cmd != nil {
		t.Fatal("did not expect a follow-up command after rebuilding suggestions from cache")
	}
	if !nextModel.suggestionsOpen {
		t.Fatal("expected suggestions to open after cache update")
	}
	if len(nextModel.suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(nextModel.suggestions))
	}
	if nextModel.suggestions[0].insertValue != "/device Kitchen Speaker" {
		t.Fatalf("unexpected insert value %q", nextModel.suggestions[0].insertValue)
	}
}

func TestBuildSuggestionsUsesFuzzyDeviceMatch(t *testing.T) {
	m := model{
		deviceCache: []app.Device{
			{Name: "Kitchen Speaker", Type: "Speaker"},
			{Name: "Desk Headphones", Type: "Computer"},
		},
		deviceCacheReady: true,
	}

	suggestions := m.buildSuggestions("/device kithn")
	if len(suggestions) == 0 {
		t.Fatal("expected fuzzy device suggestion")
	}
	if suggestions[0].insertValue != "/device Kitchen Speaker" {
		t.Fatalf("unexpected insert value %q", suggestions[0].insertValue)
	}
}

func TestPlaySuggestionsUsesFuzzyTitleMatch(t *testing.T) {
	m := newModel(stubPlayerService{})
	m.lastResults = app.Results{
		Tracks: []app.SearchItem{{Name: "One More Time", URI: "spotify:track:1"}},
	}

	suggestions := m.buildSuggestions("/play onemr")
	if len(suggestions) == 0 {
		t.Fatal("expected fuzzy play suggestion")
	}
	if suggestions[0].insertValue != "/play One More Time" {
		t.Fatalf("unexpected insert value %q", suggestions[0].insertValue)
	}
}

func TestGhostCompletionUsesSelectedSuggestionSuffix(t *testing.T) {
	m := newModel(stubPlayerService{})
	m.input.SetValue("/de")
	m.suggestionsOpen = true
	m.suggestions = []suggestion{{insertValue: "/device", value: "/device"}}

	if got := m.ghostCompletion(); got != "vice" {
		t.Fatalf("ghostCompletion() = %q, want %q", got, "vice")
	}
}

func TestPlaybackErrorShowsBannerAndBackoff(t *testing.T) {
	m := newModel(stubPlayerService{})

	updated, cmd := m.Update(playbackMsg{err: spoterr.New(spoterr.KindRateLimited, "rate limited")})
	nextModel := updated.(model)
	if cmd == nil {
		t.Fatal("expected polling command")
	}
	if nextModel.pollFailures != 1 {
		t.Fatalf("pollFailures = %d, want 1", nextModel.pollFailures)
	}
	if nextModel.bannerText == "" {
		t.Fatal("expected banner text")
	}
	if nextModel.pollEvery < 5*time.Second {
		t.Fatalf("pollEvery = %v, want backed off interval", nextModel.pollEvery)
	}
}
