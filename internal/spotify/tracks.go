package spotify

import (
	"context"
	"net/http"
	"net/url"
	"strings"
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
				ID      string `json:"id"`
				Name    string `json:"name"`
				URI     string `json:"uri"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"items"`
		} `json:"tracks"`
		Playlists struct {
			Items []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				URI   string `json:"uri"`
				Owner struct {
					DisplayName string `json:"display_name"`
				} `json:"owner"`
			} `json:"items"`
		} `json:"playlists"`
	}
	if err := c.do(ctx, http.MethodGet, "/search", params, nil, &response); err != nil {
		return nil, err
	}

	results := &SearchResults{}
	for _, item := range response.Tracks.Items {
		artistNames := make([]string, 0, len(item.Artists))
		for _, artist := range item.Artists {
			if artist.Name != "" {
				artistNames = append(artistNames, artist.Name)
			}
		}
		results.Tracks = append(results.Tracks, SearchItem{
			ID:       item.ID,
			Name:     item.Name,
			URI:      item.URI,
			Subtitle: strings.Join(artistNames, ", "),
		})
	}
	for _, item := range response.Playlists.Items {
		results.Playlists = append(results.Playlists, SearchItem{
			ID:       item.ID,
			Name:     item.Name,
			URI:      item.URI,
			Subtitle: item.Owner.DisplayName,
		})
	}
	return results, nil
}
