package app

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/petrosxen/spotui/internal/auth"
	"github.com/petrosxen/spotui/internal/config"
	spotifyapi "github.com/petrosxen/spotui/internal/spotify"
)

const playlistCacheTTL = 60 * time.Second

type PlayerService interface {
	CurrentUser(ctx context.Context) (User, error)
	Search(ctx context.Context, query string) (Results, error)
	PlayTrack(ctx context.Context, ref string) error
	PlayPlaylist(ctx context.Context, ref string) error
	Pause(ctx context.Context) error
	Resume(ctx context.Context) error
	Next(ctx context.Context) error
	Prev(ctx context.Context) error
	GetPlaybackState(ctx context.Context) (PlaybackState, error)
	ListDevices(ctx context.Context) ([]Device, error)
	ListPlaylists(ctx context.Context) ([]Playlist, error)
	SetDeviceByID(ctx context.Context, id string) error
	SetDeviceByName(ctx context.Context, substring string) (Device, error)
}

type Service struct {
	cfg     *config.Config
	client  *spotifyapi.Client
	manager *auth.Manager

	mu                sync.RWMutex
	lastSearch        Results
	lastSearchLoaded  bool
	playlistCache     []Playlist
	playlistCacheTime time.Time
}

func NewPlayerService(cfg *config.Config) (*Service, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	manager, err := auth.NewManager(cfg)
	if err != nil {
		return nil, err
	}

	svc := &Service{
		cfg:     cfg,
		manager: manager,
		client:  spotifyapi.NewClient(cfg, manager),
	}
	if len(cfg.LastSearch.Tracks) > 0 || len(cfg.LastSearch.Playlists) > 0 {
		svc.lastSearch = resultsFromConfig(cfg.LastSearch)
		svc.lastSearchLoaded = true
	}

	return svc, nil
}

func (s *Service) CurrentUser(ctx context.Context) (User, error) {
	me, err := s.client.Me(ctx)
	if err != nil {
		return User{}, err
	}
	return User{
		ID:          me.ID,
		DisplayName: me.DisplayName,
	}, nil
}

func (s *Service) Search(ctx context.Context, query string) (Results, error) {
	results, err := s.client.Search(ctx, query)
	if err != nil {
		return Results{}, err
	}

	mapped := mapResults(results)

	s.mu.Lock()
	s.lastSearch = mapped
	s.lastSearchLoaded = true
	s.mu.Unlock()

	s.cfg.LastSearch = config.LastSearch{
		Query:      query,
		Tracks:     searchItemsToConfig(mapped.Tracks),
		Playlists:  searchItemsToConfig(mapped.Playlists),
		SearchedAt: time.Now().UTC(),
	}
	if err := config.Save(s.cfg); err != nil {
		return Results{}, err
	}

	return mapped, nil
}

func (s *Service) PlayTrack(ctx context.Context, ref string) error {
	deviceID, err := s.resolvePlaybackDevice(ctx)
	if err != nil {
		return err
	}

	uri, err := s.resolveTrackRef(ref)
	if err != nil {
		return err
	}
	return s.client.PlayTrack(ctx, deviceID, uri)
}

func (s *Service) PlayPlaylist(ctx context.Context, ref string) error {
	deviceID, err := s.resolvePlaybackDevice(ctx)
	if err != nil {
		return err
	}

	uri, err := s.resolvePlaylistRef(ref)
	if err != nil {
		return err
	}
	return s.client.PlayPlaylist(ctx, deviceID, uri)
}

func (s *Service) Pause(ctx context.Context) error {
	deviceID, err := s.resolvePlaybackDevice(ctx)
	if err != nil {
		return err
	}
	return s.client.Pause(ctx, deviceID)
}

func (s *Service) Resume(ctx context.Context) error {
	deviceID, err := s.resolvePlaybackDevice(ctx)
	if err != nil {
		return err
	}
	return s.client.Resume(ctx, deviceID)
}

func (s *Service) Next(ctx context.Context) error {
	deviceID, err := s.resolvePlaybackDevice(ctx)
	if err != nil {
		return err
	}
	return s.client.Next(ctx, deviceID)
}

func (s *Service) Prev(ctx context.Context) error {
	deviceID, err := s.resolvePlaybackDevice(ctx)
	if err != nil {
		return err
	}
	return s.client.Previous(ctx, deviceID)
}

func (s *Service) GetPlaybackState(ctx context.Context) (PlaybackState, error) {
	state, err := s.client.GetPlaybackState(ctx)
	if err != nil {
		return PlaybackState{}, err
	}
	return PlaybackState{
		Device: Device{
			ID:       state.Device.ID,
			Name:     state.Device.Name,
			Type:     state.Device.Type,
			IsActive: state.Device.IsActive,
		},
		IsPlaying:        state.IsPlaying,
		Progress:         time.Duration(state.ProgressMS) * time.Millisecond,
		Duration:         time.Duration(state.Item.DurationMS) * time.Millisecond,
		ItemName:         state.Item.Name,
		ArtistName:       strings.Join(artistNames(state.Item.Artists), ", "),
		AlbumArtURL:      firstImageURL(state.Item.Album.Images),
		ItemURI:          state.Item.URI,
		ContextURI:       state.Context.URI,
		CurrentlyPlaying: state.CurrentlyPlayingType,
	}, nil
}

func (s *Service) ListDevices(ctx context.Context) ([]Device, error) {
	devices, err := s.client.Devices(ctx)
	if err != nil {
		return nil, err
	}
	return mapDevices(devices), nil
}

func (s *Service) ListPlaylists(ctx context.Context) ([]Playlist, error) {
	s.mu.RLock()
	if time.Since(s.playlistCacheTime) < playlistCacheTTL && s.playlistCache != nil {
		cached := append([]Playlist(nil), s.playlistCache...)
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	playlists, err := s.client.ListPlaylists(ctx)
	if err != nil {
		return nil, err
	}

	mapped := make([]Playlist, 0, len(playlists))
	for _, playlist := range playlists {
		mapped = append(mapped, Playlist{
			ID:   playlist.ID,
			Name: playlist.Name,
			URI:  playlist.URI,
		})
	}

	s.mu.Lock()
	s.playlistCache = append([]Playlist(nil), mapped...)
	s.playlistCacheTime = time.Now()
	s.mu.Unlock()

	return mapped, nil
}

func (s *Service) SetDeviceByID(ctx context.Context, id string) error {
	devices, err := s.ListDevices(ctx)
	if err != nil {
		return err
	}

	for _, device := range devices {
		if device.ID == id {
			s.cfg.PreferredDeviceID = id
			return config.Save(s.cfg)
		}
	}

	return fmt.Errorf("device %s was not found; run `spotui devices` or `spotui use` to pick another device", id)
}

func (s *Service) SetDeviceByName(ctx context.Context, substring string) (Device, error) {
	devices, err := s.ListDevices(ctx)
	if err != nil {
		return Device{}, err
	}

	var matches []Device
	for _, device := range devices {
		if strings.Contains(strings.ToLower(device.Name), strings.ToLower(substring)) {
			matches = append(matches, device)
		}
	}

	switch len(matches) {
	case 0:
		return Device{}, fmt.Errorf("no device matched %q", substring)
	case 1:
		s.cfg.PreferredDeviceID = matches[0].ID
		if err := config.Save(s.cfg); err != nil {
			return Device{}, err
		}
		return matches[0], nil
	default:
		names := make([]string, 0, len(matches))
		for _, match := range matches {
			names = append(names, fmt.Sprintf("%s (%s)", match.Name, match.ID))
		}
		return Device{}, fmt.Errorf("multiple devices matched %q: %s", substring, strings.Join(names, ", "))
	}
}

func (s *Service) resolvePlaybackDevice(ctx context.Context) (string, error) {
	return s.client.ResolvePlaybackDevice(ctx, s.cfg.PreferredDeviceID)
}

func (s *Service) resolveTrackRef(ref string) (string, error) {
	if idx, err := strconv.Atoi(ref); err == nil {
		results := s.lastSearchResults()
		if idx <= 0 || idx > len(results.Tracks) {
			return "", fmt.Errorf("track index %d is out of range for the last search", idx)
		}
		return results.Tracks[idx-1].URI, nil
	}
	return normalizePlayableRef(ref, "track"), nil
}

func (s *Service) resolvePlaylistRef(ref string) (string, error) {
	if idx, err := strconv.Atoi(ref); err == nil {
		results := s.lastSearchResults()
		if idx <= 0 || idx > len(results.Playlists) {
			return "", fmt.Errorf("playlist index %d is out of range for the last search", idx)
		}
		return results.Playlists[idx-1].URI, nil
	}
	return normalizePlayableRef(ref, "playlist"), nil
}

func (s *Service) lastSearchResults() Results {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.lastSearchLoaded {
		return s.lastSearch
	}
	return resultsFromConfig(s.cfg.LastSearch)
}

func normalizePlayableRef(ref string, kind string) string {
	if strings.HasPrefix(ref, "spotify:") {
		return ref
	}
	return fmt.Sprintf("spotify:%s:%s", kind, ref)
}

func mapResults(results *spotifyapi.SearchResults) Results {
	if results == nil {
		return Results{}
	}
	return Results{
		Tracks:    searchItemsFromSpotify(results.Tracks, "track"),
		Playlists: searchItemsFromSpotify(results.Playlists, "playlist"),
	}
}

func mapDevices(devices []spotifyapi.Device) []Device {
	mapped := make([]Device, 0, len(devices))
	for _, device := range devices {
		mapped = append(mapped, Device{
			ID:       device.ID,
			Name:     device.Name,
			Type:     device.Type,
			IsActive: device.IsActive,
		})
	}
	return mapped
}

func resultsFromConfig(last config.LastSearch) Results {
	return Results{
		Tracks:    searchItemsFromConfig(last.Tracks),
		Playlists: searchItemsFromConfig(last.Playlists),
	}
}

func searchItemsFromConfig(items []config.SearchItem) []SearchItem {
	mapped := make([]SearchItem, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, SearchItem{
			ID:   item.ID,
			Name: item.Name,
			URI:  item.URI,
		})
	}
	return mapped
}

func searchItemsFromSpotify(items []spotifyapi.SearchItem, kind string) []SearchItem {
	mapped := make([]SearchItem, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, SearchItem{
			ID:       item.ID,
			Name:     item.Name,
			URI:      item.URI,
			Subtitle: item.Subtitle,
			Kind:     kind,
		})
	}
	return mapped
}

func searchItemsToConfig(items []SearchItem) []config.SearchItem {
	mapped := make([]config.SearchItem, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, config.SearchItem{
			ID:   item.ID,
			Name: item.Name,
			URI:  item.URI,
		})
	}
	return mapped
}

func artistNames(artists []spotifyapi.Artist) []string {
	names := make([]string, 0, len(artists))
	for _, artist := range artists {
		if artist.Name != "" {
			names = append(names, artist.Name)
		}
	}
	return names
}

func firstImageURL(images []spotifyapi.Image) string {
	for _, image := range images {
		if image.URL != "" {
			return image.URL
		}
	}
	return ""
}
