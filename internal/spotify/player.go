package spotify

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) Pause(ctx context.Context, deviceID string) error {
	return c.do(ctx, http.MethodPut, "/me/player/pause", url.Values{"device_id": {deviceID}}, nil, nil)
}

func (c *Client) Resume(ctx context.Context, deviceID string) error {
	return c.do(ctx, http.MethodPut, "/me/player/play", url.Values{"device_id": {deviceID}}, map[string]any{}, nil)
}

func (c *Client) Next(ctx context.Context, deviceID string) error {
	return c.do(ctx, http.MethodPost, "/me/player/next", url.Values{"device_id": {deviceID}}, nil, nil)
}

func (c *Client) Previous(ctx context.Context, deviceID string) error {
	return c.do(ctx, http.MethodPost, "/me/player/previous", url.Values{"device_id": {deviceID}}, nil, nil)
}

func (c *Client) PlayTrack(ctx context.Context, deviceID string, trackURI string) error {
	body := map[string]any{
		"uris": []string{trackURI},
	}
	return c.do(ctx, http.MethodPut, "/me/player/play", url.Values{"device_id": {deviceID}}, body, nil)
}

func (c *Client) PlayPlaylist(ctx context.Context, deviceID string, playlistURI string) error {
	body := map[string]any{
		"context_uri": playlistURI,
	}
	return c.do(ctx, http.MethodPut, "/me/player/play", url.Values{"device_id": {deviceID}}, body, nil)
}

func (c *Client) ResolvePlaybackDevice(ctx context.Context, preferredID string) (string, error) {
	devices, err := c.Devices(ctx)
	if err != nil {
		return "", err
	}
	if len(devices) == 0 {
		return "", fmt.Errorf("no available Spotify devices; open Spotify on desktop, mobile, or web first")
	}
	if preferredID != "" {
		for _, device := range devices {
			if device.ID == preferredID {
				return device.ID, nil
			}
		}
		return "", fmt.Errorf("preferred device %s was not found; run `spotui devices` or `spotui use` to pick another device", preferredID)
	}
	for _, device := range devices {
		if device.IsActive {
			return device.ID, nil
		}
	}
	return "", fmt.Errorf("no active device found and no preferred device configured; run `spotui use <substring>` after starting Spotify on a device")
}
