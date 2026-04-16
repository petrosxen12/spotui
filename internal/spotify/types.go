package spotify

import "github.com/petrosxen/spotui/internal/config"

type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type Device struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	IsActive bool   `json:"is_active"`
}

type SearchResults struct {
	Tracks    []config.SearchItem
	Playlists []config.SearchItem
}
