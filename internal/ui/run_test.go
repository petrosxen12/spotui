package ui

import (
	"context"
	"errors"
	"testing"

	"github.com/petrosxen/spotui/internal/app"
)

type shutdownOnlyService struct {
	stopCalls int
	stopErr   error
	stopCtx   context.Context
}

func (s *shutdownOnlyService) CurrentUser(context.Context) (app.User, error) {
	return app.User{}, nil
}

func (s *shutdownOnlyService) Search(context.Context, string) (app.Results, error) {
	return app.Results{}, nil
}

func (s *shutdownOnlyService) PlayTrack(context.Context, string) error {
	return nil
}

func (s *shutdownOnlyService) PlayPlaylist(context.Context, string) error {
	return nil
}

func (s *shutdownOnlyService) Pause(context.Context) error {
	return nil
}

func (s *shutdownOnlyService) Resume(context.Context) error {
	return nil
}

func (s *shutdownOnlyService) Next(context.Context) error {
	return nil
}

func (s *shutdownOnlyService) Prev(context.Context) error {
	return nil
}

func (s *shutdownOnlyService) GetPlaybackState(context.Context) (app.PlaybackState, error) {
	return app.PlaybackState{}, nil
}

func (s *shutdownOnlyService) GetCurrentTrackDetails(context.Context) (app.TrackDetails, error) {
	return app.TrackDetails{}, nil
}

func (s *shutdownOnlyService) ListDevices(context.Context) ([]app.Device, error) {
	return nil, nil
}

func (s *shutdownOnlyService) ListPlaylists(context.Context) ([]app.Playlist, error) {
	return nil, nil
}

func (s *shutdownOnlyService) SetDeviceByID(context.Context, string) error {
	return nil
}

func (s *shutdownOnlyService) SetDeviceByName(context.Context, string) (app.Device, error) {
	return app.Device{}, nil
}

func (s *shutdownOnlyService) LocalPlayerStatus(context.Context) (app.LocalPlayerStatus, error) {
	return app.LocalPlayerStatus{}, nil
}

func (s *shutdownOnlyService) StartLocalPlayer(context.Context) error {
	return nil
}

func (s *shutdownOnlyService) StopLocalPlayer(ctx context.Context) error {
	s.stopCalls++
	s.stopCtx = ctx
	return s.stopErr
}

func (s *shutdownOnlyService) UseLocalPlayer(context.Context) error {
	return nil
}

func (s *shutdownOnlyService) ResetLocalPlayer(context.Context) error {
	return nil
}

func TestStopManagedLocalPlayerCallsService(t *testing.T) {
	service := &shutdownOnlyService{}

	if err := stopManagedLocalPlayer(service); err != nil {
		t.Fatalf("stopManagedLocalPlayer() error = %v", err)
	}
	if service.stopCalls != 1 {
		t.Fatalf("StopLocalPlayer() calls = %d, want 1", service.stopCalls)
	}
	if service.stopCtx == nil {
		t.Fatal("StopLocalPlayer() context = nil, want timeout context")
	}
	deadline, ok := service.stopCtx.Deadline()
	if !ok {
		t.Fatal("StopLocalPlayer() context missing deadline")
	}
	if deadline.IsZero() {
		t.Fatal("StopLocalPlayer() deadline = zero")
	}
}

func TestStopManagedLocalPlayerPropagatesError(t *testing.T) {
	want := errors.New("stop failed")
	service := &shutdownOnlyService{stopErr: want}

	err := stopManagedLocalPlayer(service)
	if !errors.Is(err, want) {
		t.Fatalf("stopManagedLocalPlayer() error = %v, want %v", err, want)
	}
}

func TestStopManagedLocalPlayerNilService(t *testing.T) {
	if err := stopManagedLocalPlayer(nil); err != nil {
		t.Fatalf("stopManagedLocalPlayer(nil) error = %v", err)
	}
}
