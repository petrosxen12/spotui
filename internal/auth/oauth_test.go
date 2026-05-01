package auth

import (
	"testing"

	"github.com/petrosxen/spotui/internal/spoterr"
)

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
