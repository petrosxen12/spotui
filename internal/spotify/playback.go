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
		Explicit   bool     `json:"explicit"`
		Popularity int      `json:"popularity"`
		Artists    []Artist `json:"artists"`
		Album      struct {
			Name   string  `json:"name"`
			Images []Image `json:"images"`
		} `json:"album"`
	} `json:"item"`
}

type AudioFeatures struct {
	Danceability     float64 `json:"danceability"`
	Energy           float64 `json:"energy"`
	Key              int     `json:"key"`
	Mode             int     `json:"mode"`
	Speechiness      float64 `json:"speechiness"`
	Acousticness     float64 `json:"acousticness"`
	Instrumentalness float64 `json:"instrumentalness"`
	Liveness         float64 `json:"liveness"`
	Valence          float64 `json:"valence"`
	Tempo            float64 `json:"tempo"`
	TimeSignature    int     `json:"time_signature"`
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

func (c *Client) GetAudioFeatures(ctx context.Context, trackID string) (*AudioFeatures, error) {
	var features AudioFeatures
	if err := c.do(ctx, http.MethodGet, "/audio-features/"+trackID, nil, nil, &features); err != nil {
		return nil, err
	}
	return &features, nil
}
