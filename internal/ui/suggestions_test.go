package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/petrosxen/spotui/internal/app"
	"github.com/petrosxen/spotui/internal/spoterr"
)

type stubPlayerService struct {
	listDevices       func(context.Context) ([]app.Device, error)
	localPlayerStatus app.LocalPlayerStatus
	startLocalPlayer  func(context.Context) error
	resetLocalPlayer  func(context.Context) error
	stopLocalPlayer   func(context.Context) error
	useLocalPlayer    func(context.Context) error
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

func (s stubPlayerService) LocalPlayerStatus(context.Context) (app.LocalPlayerStatus, error) {
	return s.localPlayerStatus, nil
}

func (s stubPlayerService) StartLocalPlayer(ctx context.Context) error {
	if s.startLocalPlayer != nil {
		return s.startLocalPlayer(ctx)
	}
	return nil
}

func (s stubPlayerService) StopLocalPlayer(ctx context.Context) error {
	if s.stopLocalPlayer != nil {
		return s.stopLocalPlayer(ctx)
	}
	return nil
}

func (s stubPlayerService) UseLocalPlayer(ctx context.Context) error {
	if s.useLocalPlayer != nil {
		return s.useLocalPlayer(ctx)
	}
	return nil
}

func (s stubPlayerService) ResetLocalPlayer(ctx context.Context) error {
	if s.resetLocalPlayer != nil {
		return s.resetLocalPlayer(ctx)
	}
	return nil
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

func TestBuildSuggestionsIncludesLocalCommands(t *testing.T) {
	m := newModel(stubPlayerService{})

	suggestions := m.buildSuggestions("/loc")
	if len(suggestions) < 2 {
		t.Fatalf("expected multiple local command suggestions, got %d", len(suggestions))
	}
	if suggestions[0].insertValue != "/local start" {
		t.Fatalf("unexpected insert value %q", suggestions[0].insertValue)
	}
	foundReset := false
	for _, suggestion := range suggestions {
		if suggestion.insertValue == "/local reset" {
			foundReset = true
			break
		}
	}
	if !foundReset {
		t.Fatal("expected /local reset suggestion")
	}
}

func TestBuildSuggestionsIncludesLocalSubcommandsAfterSpace(t *testing.T) {
	m := newModel(stubPlayerService{})

	suggestions := m.buildSuggestions("/local ")
	if len(suggestions) < 4 {
		t.Fatalf("expected full local subcommand list, got %d", len(suggestions))
	}
	if suggestions[0].insertValue != "/local start" {
		t.Fatalf("unexpected first suggestion %q", suggestions[0].insertValue)
	}
}

func TestBuildSuggestionsFiltersLocalSubcommands(t *testing.T) {
	m := newModel(stubPlayerService{})

	suggestions := m.buildSuggestions("/local re")
	if len(suggestions) != 1 {
		t.Fatalf("expected exactly one filtered local suggestion, got %d", len(suggestions))
	}
	if suggestions[0].insertValue != "/local reset" {
		t.Fatalf("unexpected filtered suggestion %q", suggestions[0].insertValue)
	}
}

func TestLocalStartCommandCallsService(t *testing.T) {
	called := 0
	m := newModel(stubPlayerService{
		localPlayerStatus: app.LocalPlayerStatus{
			Binary:  app.LocalPlayerBinary{Available: true},
			Process: app.LocalPlayerProcess{State: "running"},
			Device:  app.LocalPlayerDevice{Name: "spotui-speaker"},
			Message: app.LocalPlayerMessage{Text: "ready"},
		},
		startLocalPlayer: func(context.Context) error {
			called++
			return nil
		},
	})

	cmd := m.runSlashCommand("/local start")
	if cmd == nil {
		t.Fatal("expected start command")
	}
	msg := cmd()
	action, ok := msg.(localPlayerActionMsg)
	if !ok {
		t.Fatalf("expected localPlayerActionMsg, got %T", msg)
	}
	if action.err != nil {
		t.Fatalf("unexpected error: %v", action.err)
	}
	if action.text == "Started local player" {
		t.Fatal("expected action text to include refreshed local-player status")
	}
	if called != 1 {
		t.Fatalf("expected StartLocalPlayer to be called once, got %d", called)
	}
}

func TestLocalResetCommandCallsService(t *testing.T) {
	called := 0
	m := newModel(stubPlayerService{
		localPlayerStatus: app.LocalPlayerStatus{
			Binary:  app.LocalPlayerBinary{Available: true},
			Process: app.LocalPlayerProcess{State: "stopped"},
			Message: app.LocalPlayerMessage{Text: "ready"},
		},
		resetLocalPlayer: func(context.Context) error {
			called++
			return nil
		},
	})

	cmd := m.runSlashCommand("/local reset")
	if cmd == nil {
		t.Fatal("expected reset command")
	}
	msg := cmd()
	action, ok := msg.(localPlayerActionMsg)
	if !ok {
		t.Fatalf("expected localPlayerActionMsg, got %T", msg)
	}
	if action.err != nil {
		t.Fatalf("unexpected error: %v", action.err)
	}
	if action.text == "Reset local player" {
		t.Fatal("expected reset text to include refreshed local-player status")
	}
	if called != 1 {
		t.Fatalf("expected ResetLocalPlayer to be called once, got %d", called)
	}
}

func TestLocalStatusCommandShowsConcreteFeedback(t *testing.T) {
	m := newModel(stubPlayerService{
		localPlayerStatus: app.LocalPlayerStatus{
			Binary:  app.LocalPlayerBinary{Available: true},
			Process: app.LocalPlayerProcess{State: "running"},
			Device:  app.LocalPlayerDevice{Name: "spotui-speaker"},
			Message: app.LocalPlayerMessage{Text: "ready"},
		},
	})

	cmd := m.runSlashCommand("/local status")
	if cmd == nil {
		t.Fatal("expected status command")
	}
	msg := cmd()
	action, ok := msg.(localPlayerActionMsg)
	if !ok {
		t.Fatalf("expected localPlayerActionMsg, got %T", msg)
	}
	if action.err != nil {
		t.Fatalf("unexpected error: %v", action.err)
	}
	if action.text == "" || action.text == "Fetched local player status" {
		t.Fatalf("expected concrete status feedback, got %q", action.text)
	}
	if !strings.Contains(action.text, "Current state: running") {
		t.Fatalf("expected running state in feedback, got %q", action.text)
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

func TestSuccessfulActionExpiresAfterTTL(t *testing.T) {
	m := newModel(stubPlayerService{})
	cmd := m.setLastAction("Resumed playback", false)
	if cmd != nil {
		t.Fatal("did not expect expiry command")
	}
	if m.lastAction != "Resumed playback" {
		t.Fatalf("lastAction = %q", m.lastAction)
	}
	if m.currentLastAction() != "Resumed playback" {
		t.Fatalf("currentLastAction() = %q", m.currentLastAction())
	}
	m.lastActionUntil = time.Now().Add(-time.Second)
	if got := m.currentLastAction(); got != "" {
		t.Fatalf("expected action to expire, got %q", got)
	}
}

func TestNewerActionReplacesOlderAction(t *testing.T) {
	m := newModel(stubPlayerService{})
	first := m.setLastAction("First", false)
	second := m.setLastAction("Second", false)
	if first != nil || second != nil {
		t.Fatal("did not expect expiry commands")
	}
	if m.currentLastAction() != "Second" {
		t.Fatalf("expected newer action to remain, got %q", m.currentLastAction())
	}
}
