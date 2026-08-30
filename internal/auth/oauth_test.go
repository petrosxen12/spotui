package auth

import (
	"net/url"
	"strings"
	"testing"

	"github.com/petrosxen/spotui/internal/spoterr"
)

func TestBuildAuthURLIncludesLibraryScopes(t *testing.T) {
	rawURL := buildAuthURL("client", "http://127.0.0.1/callback", "challenge", "state")
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}

	gotScopes := make(map[string]bool)
	for _, scope := range strings.Fields(parsed.Query().Get("scope")) {
		gotScopes[scope] = true
	}
	for _, want := range []string{"user-library-read", "user-library-modify"} {
		if !gotScopes[want] {
			t.Fatalf("scope %q missing from %q", want, parsed.Query().Get("scope"))
		}
	}
}

func TestClassifyTokenErrorUsesErrorDescriptionAsAuthExpired(t *testing.T) {
	err := classifyTokenError([]byte(`{"error":"invalid_grant","error_description":"Failed to remove token"}`))

	if got := spoterr.KindOf(err); got != spoterr.KindAuthExpired {
		t.Fatalf("KindOf(err) = %q, want %q", got, spoterr.KindAuthExpired)
	}
	if got := err.Error(); got != "Spotify token error: Failed to remove token" {
		t.Fatalf("err.Error() = %q, want %q", got, "Spotify token error: Failed to remove token")
	}
}

func TestClassifyTokenErrorFallsBackToRawBodyAsAuthExpired(t *testing.T) {
	err := classifyTokenError([]byte("bad gateway from upstream"))

	if got := spoterr.KindOf(err); got != spoterr.KindAuthExpired {
		t.Fatalf("KindOf(err) = %q, want %q", got, spoterr.KindAuthExpired)
	}
	if got := err.Error(); got != "Spotify token error: bad gateway from upstream" {
		t.Fatalf("err.Error() = %q, want %q", got, "Spotify token error: bad gateway from upstream")
	}
}
