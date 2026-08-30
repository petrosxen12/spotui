package spotify

import (
	"context"
	"errors"
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

func (c *Client) SaveTrack(ctx context.Context, trackID string) error {
	trackID = strings.TrimSpace(trackID)
	if trackID == "" {
		return errors.New("track ID is required")
	}
	return c.do(ctx, http.MethodPut, "/me/tracks", nil, map[string][]string{"ids": {trackID}}, nil)
}

func (c *Client) RemoveSavedTrack(ctx context.Context, trackID string) error {
	trackID = strings.TrimSpace(trackID)
	if trackID == "" {
		return errors.New("track ID is required")
	}
	return c.do(ctx, http.MethodDelete, "/me/tracks", nil, map[string][]string{"ids": {trackID}}, nil)
}

func (c *Client) IsTrackSaved(ctx context.Context, trackID string) (bool, error) {
	trackID = strings.TrimSpace(trackID)
	if trackID == "" {
		return false, errors.New("track ID is required")
	}

	var saved []bool
	if err := c.do(ctx, http.MethodGet, "/me/tracks/contains", url.Values{"ids": {trackID}}, nil, &saved); err != nil {
		return false, err
	}
	if len(saved) != 1 {
		return false, errors.New("Spotify returned an invalid saved-track response")
	}
	return saved[0], nil
}
