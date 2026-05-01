package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/petrosxen/spotui/internal/app"
	"github.com/petrosxen/spotui/internal/spoterr"
)

func TestClassifyHeightMode(t *testing.T) {
	tests := []struct {
		height int
		want   layoutHeightMode
	}{
		{height: 17, want: heightModeMinimal},
		{height: 18, want: heightModeCompact},
		{height: 23, want: heightModeCompact},
		{height: 24, want: heightModeNormal},
	}

	for _, tt := range tests {
		if got := classifyHeightMode(tt.height); got != tt.want {
			t.Fatalf("classifyHeightMode(%d) = %q, want %q", tt.height, got, tt.want)
		}
	}
}

func TestLayoutMetricsShortHeightModes(t *testing.T) {
	m := newModel(nil)

	t.Run("compact height hides rail without a meaningful selection and hides footer hints", func(t *testing.T) {
		m.width = 140
		m.height = 22

		layout := m.layoutMetrics()

		if layout.heightMode != heightModeCompact {
			t.Fatalf("heightMode = %q, want %q", layout.heightMode, heightModeCompact)
		}
		if layout.railEnabled {
			t.Fatal("expected context rail to stay hidden until a meaningful selection exists")
		}
		if layout.footerShowHints {
			t.Fatal("expected footer hints to be hidden in compact height mode")
		}
		if !layout.playbarCompact {
			t.Fatal("expected compact playbar in compact height mode")
		}
	})

	t.Run("minimal height still disables rail", func(t *testing.T) {
		m.width = 140
		m.height = 19

		layout := m.layoutMetrics()

		if layout.heightMode != heightModeCompact {
			t.Fatalf("heightMode = %q, want %q", layout.heightMode, heightModeCompact)
		}
		if layout.railEnabled {
			t.Fatal("expected context rail to stay disabled on very short compact-height terminals")
		}
	})

	t.Run("minimal height reduces chrome aggressively", func(t *testing.T) {
		m.width = 140
		m.height = 17

		layout := m.layoutMetrics()

		if layout.heightMode != heightModeMinimal {
			t.Fatalf("heightMode = %q, want %q", layout.heightMode, heightModeMinimal)
		}
		if layout.railEnabled {
			t.Fatal("expected context rail to be disabled in minimal height mode")
		}
		if !layout.playbarMinimal {
			t.Fatal("expected minimal playbar in minimal height mode")
		}
		if layout.resultsChromeHeight != 2 {
			t.Fatalf("resultsChromeHeight = %d, want 2", layout.resultsChromeHeight)
		}
		if layout.footerShowStatus {
			t.Fatal("expected status line to be hidden in minimal height mode")
		}
	})
}

func TestLowHeightPanelsCollapseChrome(t *testing.T) {
	m := newModel(nil)
	m.width = 100
	m.height = 17
	m.connectionStatus = "Connected as tester"
	m.lastAction = "Loaded 12 results"
	m.playback = app.PlaybackState{
		IsPlaying:  true,
		ItemName:   "A Track",
		ArtistName: "An Artist",
	}

	layout := m.layoutMetrics()

	footer := m.footerPanel(layout.mainWidth, layout)
	if strings.Contains(footer, m.lastAction) {
		t.Fatal("expected minimal footer to hide the last action line")
	}
	if strings.Contains(footer, "tab focus") {
		t.Fatal("expected minimal footer to hide command hints")
	}
	if strings.Contains(footer, m.connectionStatus) {
		t.Fatal("expected minimal footer to omit connection status after moving it to the top status row")
	}

	results := m.resultsPanel(layout.mainWidth, layout)
	if strings.Contains(results, "\n\n") {
		t.Fatal("expected minimal results panel to omit the blank spacer line")
	}

	playbar := m.playbarView(layout)
	if strings.Contains(playbar, "No active device") {
		t.Fatal("expected minimal playbar to omit the secondary metadata line")
	}
}

func TestWideCompactHeightKeepsRailWhenContextWouldOverflow(t *testing.T) {
	m := newModel(nil)
	m.width = 200
	m.height = 22
	m.query = "very long query that triggers the context rail content"
	m.listMode = listModeSearch
	m.connectionStatus = "Connected as tester"
	m.lastAction = "Loaded 24 results for a long search query"
	m.playback = app.PlaybackState{
		IsPlaying:      true,
		ItemName:       "Current Track",
		ArtistName:     "Current Artist",
		NextItemName:   "Upcoming Track With A Longer Name",
		NextArtistName: "Upcoming Artist",
		Device:         app.Device{Name: "Studio Speakers"},
	}
	m.viewHistory = []viewState{{query: "previous search"}}
	m.list.SetItems([]list.Item{
		resultItem{
			title:       "Selected Result Title",
			description: "Selected Result Description",
			kind:        "track",
		},
	})
	m.list.Select(0)

	layout := m.layoutMetrics()
	if !layout.railEnabled {
		t.Fatal("expected layout to keep the context rail enabled on wide compact-height terminals")
	}

	view := m.View()
	if !strings.Contains(view, "spotui") {
		t.Fatal("expected playbar to remain visible in rendered view")
	}
	if !strings.Contains(view, "Context") {
		t.Fatal("expected rendered view to include the context rail")
	}
	if got := strings.Count(view, "\n") + 1; got > m.height {
		t.Fatalf("rendered view height = %d, want <= %d", got, m.height)
	}
}

func TestWideIdleLayoutHidesContextRailWithoutSelection(t *testing.T) {
	m := newModel(nil)
	m.width = 160
	m.height = 26

	layout := m.layoutMetrics()
	if layout.railEnabled {
		t.Fatal("expected idle layout to hide the context rail without a selection")
	}
}

func TestNextPollIntervalForErrorBacksOffAndCaps(t *testing.T) {
	rateLimited := spoterr.WithRetryAfter(spoterr.New(spoterr.KindRateLimited, "rate limited"), 7*time.Second)
	if got := nextPollIntervalForError(rateLimited, 1); got != 7*time.Second {
		t.Fatalf("first interval = %v, want %v", got, 7*time.Second)
	}
	if got := nextPollIntervalForError(rateLimited, 4); got != 30*time.Second {
		t.Fatalf("capped interval = %v, want %v", got, 30*time.Second)
	}
}
