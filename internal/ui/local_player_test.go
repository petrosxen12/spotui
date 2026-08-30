package ui

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/petrosxen/spotui/internal/app"
)

type reflectedLocalPlayerService struct{}

func (reflectedLocalPlayerService) CurrentUser(context.Context) (app.User, error) {
	return app.User{}, nil
}

func (reflectedLocalPlayerService) Search(context.Context, string) (app.Results, error) {
	return app.Results{}, nil
}

func (reflectedLocalPlayerService) PlayTrack(context.Context, string) error {
	return nil
}

func (reflectedLocalPlayerService) PlayPlaylist(context.Context, string) error {
	return nil
}

func (reflectedLocalPlayerService) Pause(context.Context) error {
	return nil
}

func (reflectedLocalPlayerService) Resume(context.Context) error {
	return nil
}

func (reflectedLocalPlayerService) Next(context.Context) error {
	return nil
}

func (reflectedLocalPlayerService) Prev(context.Context) error {
	return nil
}

func (reflectedLocalPlayerService) GetPlaybackState(context.Context) (app.PlaybackState, error) {
	return app.PlaybackState{}, nil
}

func (reflectedLocalPlayerService) GetCurrentTrackDetails(context.Context) (app.TrackDetails, error) {
	return app.TrackDetails{}, nil
}

func (reflectedLocalPlayerService) ListDevices(context.Context) ([]app.Device, error) {
	return nil, nil
}

func (reflectedLocalPlayerService) ListPlaylists(context.Context) ([]app.Playlist, error) {
	return nil, nil
}

func (reflectedLocalPlayerService) SetDeviceByID(context.Context, string) error {
	return nil
}

func (reflectedLocalPlayerService) SetDeviceByName(context.Context, string) (app.Device, error) {
	return app.Device{}, nil
}

func (reflectedLocalPlayerService) LocalPlayerStatus(context.Context) (app.LocalPlayerStatus, error) {
	return app.LocalPlayerStatus{
		Binary:  app.LocalPlayerBinary{Available: true},
		Process: app.LocalPlayerProcess{State: "running"},
		Device:  app.LocalPlayerDevice{Name: "spotui-speaker"},
		Message: app.LocalPlayerMessage{Text: "ready"},
	}, nil
}

func (reflectedLocalPlayerService) StartLocalPlayer(context.Context) error {
	return nil
}

func (reflectedLocalPlayerService) StopLocalPlayer(context.Context) error {
	return nil
}

func (reflectedLocalPlayerService) UseLocalPlayer(context.Context) error {
	return nil
}

func (reflectedLocalPlayerService) ResetLocalPlayer(context.Context) error {
	return nil
}

func TestGetLocalPlayerStatusReflectsServiceShape(t *testing.T) {
	status, err := getLocalPlayerStatus(reflectedLocalPlayerService{})
	if err != nil {
		t.Fatalf("getLocalPlayerStatus() error = %v", err)
	}
	if !status.supported {
		t.Fatal("expected local player support to be detected")
	}
	if !status.binaryAvailable {
		t.Fatal("expected binary availability to be true")
	}
	if status.process != "running" {
		t.Fatalf("process = %q, want running", status.process)
	}
	if status.device != "spotui-speaker" {
		t.Fatalf("device = %q, want spotui-speaker", status.device)
	}
	if status.message != "ready" {
		t.Fatalf("message = %q, want ready", status.message)
	}
}

type sleepRecoveryService struct {
	reflectedLocalPlayerService
	calls    []string
	resetErr error
	startErr error
}

func (s *sleepRecoveryService) ResetLocalPlayer(context.Context) error {
	s.calls = append(s.calls, "reset")
	return s.resetErr
}

func (s *sleepRecoveryService) StartLocalPlayer(context.Context) error {
	s.calls = append(s.calls, "start")
	return s.startErr
}

func TestPlaybackPollRecoversRunningLocalPlayerAfterWallClockGap(t *testing.T) {
	service := &sleepRecoveryService{}
	base := time.Now()
	m := newModel(service)
	m.pollEvery = 4 * time.Second
	m.lastPlaybackPollAt = base.Round(0)
	m.localPlayer = localPlayerStatus{
		supported:       true,
		binaryAvailable: true,
		process:         "running",
		device:          "spotui-speaker",
	}

	updated, cmd := m.Update(pollTickMsg{at: base.Add(13 * time.Second)})
	next := updated.(model)
	if cmd == nil {
		t.Fatal("expected local-player recovery command")
	}
	if !next.localPlayerRecovering {
		t.Fatal("expected local-player recovery to be marked in progress")
	}
	if !next.lastPlaybackPollAt.Equal(base.Add(13 * time.Second)) {
		t.Fatalf("lastPlaybackPollAt = %v, want %v", next.lastPlaybackPollAt, base.Add(13*time.Second))
	}

	msg := cmd()
	action, ok := msg.(localPlayerActionMsg)
	if !ok {
		t.Fatalf("expected localPlayerActionMsg, got %T", msg)
	}
	if action.err != nil {
		t.Fatalf("unexpected recovery error: %v", action.err)
	}
	if action.text != "Reconnected local player after sleep" {
		t.Fatalf("recovery text = %q", action.text)
	}
	if len(service.calls) != 2 || service.calls[0] != "reset" || service.calls[1] != "start" {
		t.Fatalf("recovery calls = %v, want [reset start]", service.calls)
	}

	recovered, _ := next.Update(action)
	recoveredModel := recovered.(model)
	if recoveredModel.localPlayerRecovering {
		t.Fatal("expected recovery-in-progress state to clear")
	}
	if recoveredModel.currentLastAction() != "Reconnected local player after sleep" {
		t.Fatalf("action banner = %q", recoveredModel.currentLastAction())
	}
}

func TestPlaybackPollDoesNotRecoverWithoutAnOversizedGap(t *testing.T) {
	service := &sleepRecoveryService{}
	base := time.Now().Round(0)
	m := newModel(service)
	m.pollEvery = 4 * time.Second
	m.lastPlaybackPollAt = base
	m.localPlayer = localPlayerStatus{supported: true, process: "running"}

	updated, cmd := m.Update(pollTickMsg{at: base.Add(12 * time.Second)})
	next := updated.(model)
	if cmd == nil {
		t.Fatal("expected normal playback fetch command")
	}
	if next.localPlayerRecovering {
		t.Fatal("did not expect recovery at exactly three times the poll interval")
	}
	if _, ok := cmd().(playbackMsg); !ok {
		t.Fatal("expected the normal playback fetch to continue")
	}
	if len(service.calls) != 0 {
		t.Fatalf("unexpected local-player calls: %v", service.calls)
	}
}

func TestPlaybackPollDoesNotRecoverDeliberatelyStoppedLocalPlayer(t *testing.T) {
	service := &sleepRecoveryService{}
	base := time.Now().Round(0)
	m := newModel(service)
	m.pollEvery = 4 * time.Second
	m.lastPlaybackPollAt = base
	m.localPlayer = localPlayerStatus{supported: true, process: "running"}

	stopped, _ := m.Update(localPlayerActionMsg{
		text:   "Stopped local player",
		action: localPlayerActionStop,
	})
	stoppedModel := stopped.(model)
	if !stoppedModel.localPlayerStoppedByUser {
		t.Fatal("expected deliberate stop to disable automatic recovery")
	}

	updated, cmd := stoppedModel.Update(pollTickMsg{at: base.Add(13 * time.Second)})
	next := updated.(model)
	if next.localPlayerRecovering {
		t.Fatal("did not expect a deliberately stopped player to recover")
	}
	if _, ok := cmd().(playbackMsg); !ok {
		t.Fatal("expected the normal playback fetch to continue")
	}
	if len(service.calls) != 0 {
		t.Fatalf("unexpected local-player calls: %v", service.calls)
	}
}

func TestRecoverLocalPlayerDoesNotStartWhenResetFails(t *testing.T) {
	wantErr := errors.New("reset failed")
	service := &sleepRecoveryService{resetErr: wantErr}

	msg := recoverLocalPlayerCmd(service)()
	action, ok := msg.(localPlayerActionMsg)
	if !ok {
		t.Fatalf("expected localPlayerActionMsg, got %T", msg)
	}
	if !errors.Is(action.err, wantErr) {
		t.Fatalf("recovery error = %v, want %v", action.err, wantErr)
	}
	if len(service.calls) != 1 || service.calls[0] != "reset" {
		t.Fatalf("recovery calls = %v, want [reset]", service.calls)
	}
}
