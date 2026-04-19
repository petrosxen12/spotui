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
	"github.com/petrosxen/spotui/internal/spoterr"
	spotifyapi "github.com/petrosxen/spotui/internal/spotify"
	"github.com/sahilm/fuzzy"
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
	playback := PlaybackState{
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
	}

	queue, err := s.client.GetQueue(ctx)
	if err == nil && len(queue.Queue) > 0 {
		playback.NextItemName = queue.Queue[0].Name
		playback.NextArtistName = strings.Join(artistNames(queue.Queue[0].Artists), ", ")
	}
	if playback.Device.ID != "" {
		s.rememberLastDevice(playback.Device, false)
	}

	return playback, nil
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
			s.rememberLastDevice(device, true)
			return nil
		}
	}

	return fmt.Errorf("device %s was not found; run `spotui devices` or `spotui use` to pick another device", id)
}

func (s *Service) SetDeviceByName(ctx context.Context, substring string) (Device, error) {
	devices, err := s.ListDevices(ctx)
	if err != nil {
		return Device{}, err
	}

	match, err := bestDeviceMatch(devices, substring)
	if err != nil {
		return Device{}, err
	}

	s.cfg.PreferredDeviceID = match.ID
	s.rememberLastDevice(match, true)
	return match, nil
}

func (s *Service) resolvePlaybackDevice(ctx context.Context) (string, error) {
	devices, err := s.ListDevices(ctx)
	if err != nil {
		return "", err
	}
	if len(devices) == 0 {
		return "", spoterr.New(spoterr.KindNoActiveDevice, "no available Spotify devices; open Spotify on desktop, mobile, or web first")
	}
	if device, ok := findDeviceByID(devices, s.cfg.PreferredDeviceID); ok {
		return device.ID, nil
	}
	if device, ok := findDeviceByID(devices, s.cfg.LastUsedDevice.ID); ok {
		return device.ID, nil
	}
	if s.cfg.LastUsedDevice.Name != "" {
		if device, err := bestDeviceMatch(devices, s.cfg.LastUsedDevice.Name); err == nil {
			return device.ID, nil
		}
	}
	for _, device := range devices {
		if device.IsActive {
			return device.ID, nil
		}
	}
	return "", spoterr.New(spoterr.KindNoActiveDevice, "no active device found and no saved device is currently available; run `spotui use <name>` after starting Spotify on a device")
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

func (s *Service) rememberLastDevice(device Device, selected bool) {
	if device.ID == "" {
		return
	}
	s.cfg.LastUsedDevice = config.LastDevice{
		ID:       device.ID,
		Name:     device.Name,
		Type:     device.Type,
		SeenAt:   time.Now().UTC(),
		Selected: selected,
	}
	_ = config.Save(s.cfg)
}

func findDeviceByID(devices []Device, id string) (Device, bool) {
	if id == "" {
		return Device{}, false
	}
	for _, device := range devices {
		if device.ID == id {
			return device, true
		}
	}
	return Device{}, false
}

func bestDeviceMatch(devices []Device, needle string) (Device, error) {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return Device{}, fmt.Errorf("device name is required")
	}

	lowerNeedle := strings.ToLower(needle)
	exact := make([]Device, 0, 1)
	substrings := make([]Device, 0)
	for _, device := range devices {
		lowerName := strings.ToLower(device.Name)
		switch {
		case lowerName == lowerNeedle:
			exact = append(exact, device)
		case strings.Contains(lowerName, lowerNeedle):
			substrings = append(substrings, device)
		}
	}

	if len(exact) == 1 {
		return exact[0], nil
	}
	if len(exact) > 1 {
		return Device{}, fmt.Errorf("multiple devices matched %q: %s", needle, joinDeviceNames(exact))
	}
	if len(substrings) == 1 {
		return substrings[0], nil
	}
	if len(substrings) > 1 {
		return Device{}, fmt.Errorf("multiple devices matched %q: %s", needle, joinDeviceNames(substrings))
	}

	targets := make([]string, 0, len(devices))
	for _, device := range devices {
		targets = append(targets, device.Name)
	}
	matches := fuzzy.Find(needle, targets)
	if len(matches) == 0 {
		return Device{}, fmt.Errorf("no device matched %q", needle)
	}
	if len(matches) > 1 && matches[0].Score == matches[1].Score {
		return Device{}, fmt.Errorf("multiple devices matched %q: %s", needle, joinDeviceNames([]Device{
			devices[matches[0].Index],
			devices[matches[1].Index],
		}))
	}
	return devices[matches[0].Index], nil
}

func joinDeviceNames(devices []Device) string {
	names := make([]string, 0, len(devices))
	for _, device := range devices {
		names = append(names, fmt.Sprintf("%s (%s)", device.Name, device.ID))
	}
	return strings.Join(names, ", ")
}
