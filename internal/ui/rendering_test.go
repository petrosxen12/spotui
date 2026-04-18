package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/petrosxen/spotui/internal/app"
)

func TestCompactPlaybarVeryNarrowWidthStaysBounded(t *testing.T) {
	m := newModel(nil)
	m.width = 34
	m.height = 22
	m.playback = app.PlaybackState{
		IsPlaying:      true,
		ItemName:       "An Extremely Long Track Title That Should Never Wrap Unbounded",
		ArtistName:     "A Very Long Artist Name With Guests And More Guests",
		NextItemName:   "Another Long Upcoming Track",
		Duration:       4*time.Minute + 32*time.Second,
		Progress:       83 * time.Second,
		Device:         app.Device{Name: "Bedroom Speaker With A Long Name"},
		NextArtistName: "Another Artist",
	}

	layout := m.layoutMetrics()
	playbar := m.playbarView(layout)
	lines := strings.Split(playbar, "\n")

	if len(lines) != 2 {
		t.Fatalf("expected very narrow compact playbar to collapse to 2 lines, got %d", len(lines))
	}
	for _, line := range lines {
		if lipgloss.Width(line) > layout.bodyWidth {
			t.Fatalf("playbar line width %d exceeds body width %d: %q", lipgloss.Width(line), layout.bodyWidth, line)
		}
	}
}

func TestSuggestionsViewTruncatesLongEntries(t *testing.T) {
	m := newModel(nil)
	m.width = 32
	m.height = 24
	m.suggestionsOpen = true
	m.suggestions = []suggestion{
		{
			value:       "/device Living Room Television Output",
			insertValue: "/device Living Room Television Output",
			description: "speaker target with long description",
		},
	}

	layout := m.layoutMetrics()
	rendered := m.suggestionsView(layout)
	popupWidth := maxInt(20, layout.bodyWidth-4) + 1

	for _, line := range strings.Split(rendered, "\n") {
		if lipgloss.Width(line) > popupWidth {
			t.Fatalf("suggestion line width %d exceeds popup width %d: %q", lipgloss.Width(line), popupWidth, line)
		}
	}
}

func TestResultDelegateTruncatesLongRows(t *testing.T) {
	delegate := resultDelegate{width: 24, wideLayout: false}
	items := []list.Item{
		resultItem{
			title:       "A Track With A Very Long Title That Should Truncate",
			description: "Description text that should also truncate instead of wrapping badly",
			kind:        "track",
		},
	}
	model := list.New(items, delegate, 24, 3)
	model.Select(0)

	var buf bytes.Buffer
	delegate.Render(&buf, model, 0, items[0])

	for _, line := range strings.Split(buf.String(), "\n") {
		if lipgloss.Width(line) > delegate.contentWidth()+2 {
			t.Fatalf("row line width %d exceeds expected bound: %q", lipgloss.Width(line), line)
		}
	}
}
