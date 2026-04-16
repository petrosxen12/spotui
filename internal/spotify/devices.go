package spotify

import (
	"context"
	"net/http"
)

func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	var response struct {
		Devices []Device `json:"devices"`
	}
	if err := c.do(ctx, http.MethodGet, "/me/player/devices", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Devices, nil
}
