package app

import "time"

type User struct {
	ID          string
	DisplayName string
}

type SearchItem struct {
	ID       string
	Name     string
	URI      string
	Subtitle string
	Kind     string
}

type Results struct {
	Tracks    []SearchItem
	Playlists []SearchItem
}

type Device struct {
	ID       string
	Name     string
	Type     string
	IsActive bool
}

type LocalPlayerBinary struct {
	Available bool
}

type LocalPlayerProcess struct {
	State string
}

type LocalPlayerMessage struct {
	Text string
}

type LocalPlayerDevice struct {
	ID   string
	Name string
}

type LocalPlayerStatus struct {
	Enabled        bool
	Binary         LocalPlayerBinary
	Process        LocalPlayerProcess
	Device         LocalPlayerDevice
	Message        LocalPlayerMessage
	ConfiguredName string
	LogPath        string
}

type Playlist struct {
	ID   string
	Name string
	URI  string
}

type PlaybackState struct {
	Device           Device
	IsPlaying        bool
	Progress         time.Duration
	Duration         time.Duration
	ItemName         string
	ArtistName       string
	AlbumArtURL      string
	ItemURI          string
	ContextURI       string
	CurrentlyPlaying string
	NextItemName     string
	NextArtistName   string
}
