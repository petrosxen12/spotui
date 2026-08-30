package spotify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestSaveTrack(t *testing.T) {
	client := newSavedTrackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/v1/me/tracks" {
			t.Fatalf("path = %q, want /v1/me/tracks", r.URL.Path)
		}
		assertTrackIDsBody(t, r, "track-1")
		w.WriteHeader(http.StatusOK)
	})

	if err := client.SaveTrack(context.Background(), "track-1"); err != nil {
		t.Fatalf("SaveTrack() error = %v", err)
	}
}

func TestRemoveSavedTrack(t *testing.T) {
	client := newSavedTrackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %q, want DELETE", r.Method)
		}
		if r.URL.Path != "/v1/me/tracks" {
			t.Fatalf("path = %q, want /v1/me/tracks", r.URL.Path)
		}
		assertTrackIDsBody(t, r, "track-1")
		w.WriteHeader(http.StatusOK)
	})

	if err := client.RemoveSavedTrack(context.Background(), "track-1"); err != nil {
		t.Fatalf("RemoveSavedTrack() error = %v", err)
	}
}

func TestIsTrackSaved(t *testing.T) {
	client := newSavedTrackTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/v1/me/tracks/contains" {
			t.Fatalf("path = %q, want /v1/me/tracks/contains", r.URL.Path)
		}
		if got := r.URL.Query().Get("ids"); got != "track-1" {
			t.Fatalf("ids = %q, want track-1", got)
		}
		writeSavedTrackJSON(t, w, []bool{true})
	})

	saved, err := client.IsTrackSaved(context.Background(), "track-1")
	if err != nil {
		t.Fatalf("IsTrackSaved() error = %v", err)
	}
	if !saved {
		t.Fatal("IsTrackSaved() = false, want true")
	}
}

func TestSavedTrackMethodsRejectEmptyTrackID(t *testing.T) {
	client := &Client{}

	if err := client.SaveTrack(context.Background(), " "); err == nil {
		t.Fatal("SaveTrack() error = nil, want validation error")
	}
	if err := client.RemoveSavedTrack(context.Background(), " "); err == nil {
		t.Fatal("RemoveSavedTrack() error = nil, want validation error")
	}
	if _, err := client.IsTrackSaved(context.Background(), " "); err == nil {
		t.Fatal("IsTrackSaved() error = nil, want validation error")
	}
}

func newSavedTrackTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	transport := server.Client().Transport

	client := NewClient(nil, savedTrackTokenSource{})
	client.httpClient = &http.Client{Transport: savedTrackRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		clone := req.Clone(req.Context())
		clone.URL.Scheme = target.Scheme
		clone.URL.Host = target.Host
		return transport.RoundTrip(clone)
	})}
	return client
}

func assertTrackIDsBody(t *testing.T, r *http.Request, want string) {
	t.Helper()

	var body struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if len(body.IDs) != 1 || body.IDs[0] != want {
		t.Fatalf("ids = %#v, want [%q]", body.IDs, want)
	}
}

func writeSavedTrackJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

type savedTrackTokenSource struct{}

func (savedTrackTokenSource) ValidAccessToken(context.Context) (string, error) {
	return "test-token", nil
}

type savedTrackRoundTripFunc func(*http.Request) (*http.Response, error)

func (f savedTrackRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
