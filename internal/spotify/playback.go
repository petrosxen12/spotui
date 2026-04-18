package spotify

import (
	"context"
	"net/http"
)

type PlaybackState struct {
	Device               Device `json:"device"`
	IsPlaying            bool   `json:"is_playing"`
	ProgressMS           int64  `json:"progress_ms"`
	CurrentlyPlayingType string `json:"currently_playing_type"`
	Context              struct {
		URI string `json:"uri"`
	} `json:"context"`
	Item struct {
		Name       string   `json:"name"`
		URI        string   `json:"uri"`
		DurationMS int64    `json:"duration_ms"`
		Artists    []Artist `json:"artists"`
		Album      struct {
			Images []Image `json:"images"`
		} `json:"album"`
	} `json:"item"`
}

type QueueResponse struct {
	CurrentlyPlaying QueueItem   `json:"currently_playing"`
	Queue            []QueueItem `json:"queue"`
}

type QueueItem struct {
	Name    string   `json:"name"`
	URI     string   `json:"uri"`
	Artists []Artist `json:"artists"`
}

func (c *Client) GetPlaybackState(ctx context.Context) (*PlaybackState, error) {
	var state PlaybackState
	if err := c.do(ctx, http.MethodGet, "/me/player", nil, nil, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (c *Client) GetQueue(ctx context.Context) (*QueueResponse, error) {
	var queue QueueResponse
	if err := c.do(ctx, http.MethodGet, "/me/player/queue", nil, nil, &queue); err != nil {
		return nil, err
	}
	return &queue, nil
}
