package ui

import (
	"context"
	"testing"

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
