package ui

import (
	"strings"
	"testing"

	"github.com/petrosxen/spotui/internal/app"
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

	t.Run("compact height disables rail and footer hints", func(t *testing.T) {
		m.width = 140
		m.height = 22

		layout := m.layoutMetrics()

		if layout.heightMode != heightModeCompact {
			t.Fatalf("heightMode = %q, want %q", layout.heightMode, heightModeCompact)
		}
		if layout.railEnabled {
			t.Fatal("expected context rail to be disabled in compact height mode")
		}
		if layout.footerShowHints {
			t.Fatal("expected footer hints to be hidden in compact height mode")
		}
		if !layout.playbarCompact {
			t.Fatal("expected compact playbar in compact height mode")
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
	if !strings.Contains(footer, m.connectionStatus) {
		t.Fatal("expected minimal footer to keep connection status")
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
