package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/petrosxen/spotui/internal/config"
	"github.com/petrosxen/spotui/internal/spoterr"
)

const (
	authURL  = "https://accounts.spotify.com/authorize"
	tokenURL = "https://accounts.spotify.com/api/token"
)

var scopes = []string{
	"user-read-playback-state",
	"user-modify-playback-state",
	"user-read-currently-playing",
}

type Manager struct {
	config *config.Config
}

func NewManager(cfg *config.Config) (*Manager, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if cfg.RedirectURI == "" {
		cfg.RedirectURI = config.DefaultRedirectURI
	}
	if cfg.ClientID == "" {
		return &Manager{config: cfg}, nil
	}
	return &Manager{config: cfg}, nil
}

func (m *Manager) Login(ctx context.Context) (*Token, error) {
	if m.config.ClientID == "" {
		return nil, errors.New("missing Spotify client ID")
	}

	redirect, err := url.Parse(m.config.RedirectURI)
	if err != nil {
		return nil, fmt.Errorf("parse redirect URI: %w", err)
	}
	if redirect.Hostname() != "127.0.0.1" && redirect.Hostname() != "localhost" {
		return nil, errors.New("redirect URI host must be 127.0.0.1 or localhost")
	}
	if redirect.Path == "" {
		return nil, errors.New("redirect URI must include a callback path")
	}

	codeVerifier, err := randomURLString(64)
	if err != nil {
		return nil, err
	}
	state, err := randomURLString(32)
	if err != nil {
		return nil, err
	}

	codeChallenge := challenge(codeVerifier)
	codeCh, errCh, err := waitForCode(ctx, redirect, state)
	if err != nil {
		return nil, err
	}

	authPage := buildAuthURL(m.config.ClientID, m.config.RedirectURI, codeChallenge, state)
	if err := openBrowser(authPage); err != nil {
		fmt.Printf("Open this URL in your browser to continue:\n%s\n", authPage)
	} else {
		fmt.Printf("Opening browser for Spotify login...\nIf nothing appears, open:\n%s\n", authPage)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		if err == nil {
			return nil, errors.New("authorization callback ended unexpectedly")
		}
		return nil, err
	case code := <-codeCh:
		token, err := exchangeCode(ctx, m.config.ClientID, m.config.RedirectURI, codeVerifier, code)
		if err != nil {
			return nil, err
		}
		if err := SaveToken(token); err != nil {
			return nil, err
		}
		return token, nil
	}
}

func (m *Manager) ValidAccessToken(ctx context.Context) (string, error) {
	token, err := LoadToken()
	if err != nil {
		return "", err
	}
	if token.RefreshToken == "" && token.Expired() {
		return "", spoterr.New(spoterr.KindAuthExpired, "token expired and no refresh token is available; run `spotui login` again")
	}
	if !token.ShouldRefresh() {
		return token.AccessToken, nil
	}
	refreshed, err := m.Refresh(ctx, token)
	if err != nil {
		return "", err
	}
	return refreshed.AccessToken, nil
}

func (m *Manager) Refresh(ctx context.Context, token *Token) (*Token, error) {
	if m.config.ClientID == "" {
		return nil, errors.New("missing Spotify client ID; run `spotui login --client-id ...` first")
	}
	if token == nil {
		return nil, errors.New("token is required")
	}
	if token.RefreshToken == "" {
		return nil, spoterr.New(spoterr.KindAuthExpired, "token refresh failed: refresh token missing; run `spotui login` again")
	}

	form := url.Values{
		"client_id":     {m.config.ClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
	}
	refreshed, err := doTokenRequest(ctx, form)
	if err != nil {
		return nil, err
	}
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = token.RefreshToken
	}
	if err := SaveToken(refreshed); err != nil {
		return nil, err
	}
	return refreshed, nil
}

func waitForCode(ctx context.Context, redirect *url.URL, expectedState string) (<-chan string, <-chan error, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:              redirect.Host,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	mux.HandleFunc(redirect.Path, func(w http.ResponseWriter, r *http.Request) {
		if state := r.URL.Query().Get("state"); state != expectedState {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			select {
			case errCh <- errors.New("state mismatch during OAuth callback"):
			default:
			}
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing authorization code", http.StatusBadRequest)
			select {
			case errCh <- errors.New("authorization code missing from callback"):
			default:
			}
			return
		}

		_, _ = io.WriteString(w, "spotui login complete. You can close this window.")
		select {
		case codeCh <- code:
		default:
		}

		go func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
		}()
	})

	ln, err := net.Listen("tcp", redirect.Host)
	if err != nil {
		return nil, nil, fmt.Errorf("start callback server on %s: %w", redirect.Host, err)
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	go func() {
		defer close(errCh)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("callback server failed: %w", err)
		}
	}()

	return codeCh, errCh, nil
}

func exchangeCode(ctx context.Context, clientID, redirectURI, codeVerifier, code string) (*Token, error) {
	form := url.Values{
		"client_id":     {clientID},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
	}
	return doTokenRequest(ctx, form)
}

func doTokenRequest(ctx context.Context, form url.Values) (*Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request Spotify token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, classifyTokenError(body)
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if payload.AccessToken == "" {
		return nil, errors.New("Spotify token response did not include an access token")
	}

	return &Token{
		AccessToken:  payload.AccessToken,
		TokenType:    payload.TokenType,
		RefreshToken: payload.RefreshToken,
		Scope:        payload.Scope,
		ExpiresAt:    time.Now().UTC().Add(time.Duration(payload.ExpiresIn) * time.Second),
	}, nil
}

func classifyTokenError(body []byte) error {
	var apiErr struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &apiErr)

	message := strings.TrimSpace(string(body))
	switch {
	case apiErr.ErrorDescription != "":
		message = apiErr.ErrorDescription
	case apiErr.Error != "":
		message = apiErr.Error
	}

	return spoterr.New(spoterr.KindAuthExpired, "Spotify token error: "+message)
}

func buildAuthURL(clientID, redirectURI, codeChallenge, state string) string {
	values := url.Values{
		"client_id":             {clientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"code_challenge_method": {"S256"},
		"code_challenge":        {codeChallenge},
		"state":                 {state},
		"scope":                 {strings.Join(scopes, " ")},
	}
	return authURL + "?" + values.Encode()
}

func openBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}

func challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomURLString(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random value: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
