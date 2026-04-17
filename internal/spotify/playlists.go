package spotify

import (
	"context"
	"net/http"
	"net/url"
)

type Playlist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URI  string `json:"uri"`
}

func (c *Client) ListPlaylists(ctx context.Context) ([]Playlist, error) {
	params := url.Values{
		"limit": {"50"},
	}

	var response struct {
		Items []Playlist `json:"items"`
	}
	if err := c.do(ctx, http.MethodGet, "/me/playlists", params, nil, &response); err != nil {
		return nil, err
	}
	return response.Items, nil
}
