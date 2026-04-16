package spotify

import (
	"context"
	"net/http"
	"net/url"

	"github.com/petrosxen/spotui/internal/config"
)

func (c *Client) Me(ctx context.Context) (*User, error) {
	var user User
	if err := c.do(ctx, http.MethodGet, "/me", nil, nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) Search(ctx context.Context, query string) (*SearchResults, error) {
	params := url.Values{
		"q":     {query},
		"type":  {"track,playlist"},
		"limit": {"5"},
	}

	var response struct {
		Tracks struct {
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				URI  string `json:"uri"`
			} `json:"items"`
		} `json:"tracks"`
		Playlists struct {
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				URI  string `json:"uri"`
			} `json:"items"`
		} `json:"playlists"`
	}
	if err := c.do(ctx, http.MethodGet, "/search", params, nil, &response); err != nil {
		return nil, err
	}

	results := &SearchResults{}
	for _, item := range response.Tracks.Items {
		results.Tracks = append(results.Tracks, config.SearchItem{
			ID:   item.ID,
			Name: item.Name,
			URI:  item.URI,
		})
	}
	for _, item := range response.Playlists.Items {
		results.Playlists = append(results.Playlists, config.SearchItem{
			ID:   item.ID,
			Name: item.Name,
			URI:  item.URI,
		})
	}
	return results, nil
}
