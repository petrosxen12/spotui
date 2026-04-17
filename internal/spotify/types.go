package spotify

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

type Artist struct {
	Name string `json:"name"`
}

type SearchItem struct {
	ID       string
	Name     string
	URI      string
	Subtitle string
}

type SearchResults struct {
	Tracks    []SearchItem
	Playlists []SearchItem
}
