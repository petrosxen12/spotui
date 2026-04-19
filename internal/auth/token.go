package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/petrosxen/spotui/internal/config"
	"github.com/petrosxen/spotui/internal/spoterr"
)

type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func LoadToken() (*Token, error) {
	path, err := config.TokenPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, spoterr.New(spoterr.KindAuthExpired, "not logged in; run `spotui login` first")
		}
		return nil, fmt.Errorf("read token: %w", err)
	}
	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	return &token, nil
}

func SaveToken(token *Token) error {
	if token == nil {
		return errors.New("token is required")
	}
	path, err := config.TokenPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}
	if err := config.WriteSecureFile(path, append(data, '\n')); err != nil {
		return fmt.Errorf("write token: %w", err)
	}
	return nil
}

func (t *Token) Expired() bool {
	return time.Now().UTC().After(t.ExpiresAt)
}

func (t *Token) ShouldRefresh() bool {
	return time.Now().UTC().Add(1 * time.Minute).After(t.ExpiresAt)
}
