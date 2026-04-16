package spotify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/petrosxen/spotui/internal/config"
)

const baseURL = "https://api.spotify.com/v1"

type tokenSource interface {
	ValidAccessToken(ctx context.Context) (string, error)
}

type Client struct {
	httpClient  *http.Client
	config      *config.Config
	tokenSource tokenSource
}

func NewClient(cfg *config.Config, source tokenSource) *Client {
	return &Client{
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		config:      cfg,
		tokenSource: source,
	}
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	requestURL := baseURL + path
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}

	token, err := c.tokenSource.ValidAccessToken(ctx)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request Spotify API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return fmt.Errorf("read Spotify response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return classifyError(resp.StatusCode, respBody)
	}
	if out == nil || len(strings.TrimSpace(string(respBody))) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode Spotify response: %w", err)
	}
	return nil
}

func classifyError(status int, body []byte) error {
	var payload struct {
		Error struct {
			Status  int    `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &payload)

	message := payload.Error.Message
	if message == "" {
		message = strings.TrimSpace(string(body))
	}

	switch status {
	case http.StatusUnauthorized:
		return errors.New("Spotify authentication failed or token refresh did not succeed; run `spotui login` again")
	case http.StatusForbidden:
		if strings.Contains(strings.ToLower(message), "premium") {
			return fmt.Errorf("Spotify Premium is required for playback control: %s", message)
		}
		return fmt.Errorf("Spotify denied the request: %s", message)
	case http.StatusNotFound:
		return errors.New("no active playback device found; start Spotify on a device or select one with `spotui use`")
	case http.StatusTooManyRequests:
		return fmt.Errorf("Spotify rate limited the request: %s", message)
	default:
		if message == "" {
			return fmt.Errorf("Spotify API returned HTTP %d", status)
		}
		return fmt.Errorf("Spotify API returned HTTP %d: %s", status, message)
	}
}
