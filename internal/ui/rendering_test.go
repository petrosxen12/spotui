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

func TestEffectiveAccentColorFallsBackToSpotifyGreen(t *testing.T) {
	m := newModel(nil)

	if got := m.effectiveAccentColor(); got != "#1DB954" {
		t.Fatalf("effectiveAccentColor() = %q, want %q", got, "#1DB954")
	}
	if got := m.vividAccentColor(); got != "#1DB954" {
		t.Fatalf("vividAccentColor() = %q, want %q", got, "#1DB954")
	}
}

func TestDerivedAccentColorsSplitTextAndChrome(t *testing.T) {
	m := newModel(nil)
	m.accentColor = "#8c7a69"

	if got := m.textAccentColor(); got == m.accentColor {
		t.Fatalf("textAccentColor() = %q, want derived color different from base", got)
	}
	if got := m.vividAccentColor(); got == m.accentColor {
		t.Fatalf("vividAccentColor() = %q, want boosted color different from base", got)
	}
	if m.textAccentColor() == m.vividAccentColor() {
		t.Fatalf("expected text and vivid accent colors to differ, got %q", m.textAccentColor())
	}
}

func TestRefreshAccentColorUsesCachedAlbumAccentWhilePaused(t *testing.T) {
	m := newModel(nil)
	m.playback = app.PlaybackState{
		IsPlaying:   false,
		AlbumArtURL: "https://example.com/cover.jpg",
	}
	m.accentColorCache[m.playback.AlbumArtURL] = "#abcdef"

	cmd := m.refreshAccentColor()
	if cmd != nil {
		t.Fatal("refreshAccentColor() returned unexpected fetch command for cached art")
	}
	if got := m.accentColor; got != "#abcdef" {
		t.Fatalf("accentColor = %q, want %q", got, "#abcdef")
	}
}

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

	if len(lines) != 3 {
		t.Fatalf("expected very narrow compact playbar to collapse to 3 lines, got %d", len(lines))
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
	popupWidth := 0
	for _, line := range strings.Split(rendered, "\n") {
		popupWidth = maxInt(popupWidth, lipgloss.Width(line))
	}

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

func TestItemsFromResultsDropsEntriesWithoutVisibleTitles(t *testing.T) {
	items := itemsFromResults(app.Results{
		Tracks: []app.SearchItem{
			{Name: "Visible Track", URI: "spotify:track:1"},
			{Name: " \t ", URI: "spotify:track:2"},
		},
		Playlists: []app.SearchItem{
			{Name: "\u200d\ufe0f", URI: "spotify:playlist:1"},
			{Name: "Visible Playlist", URI: "spotify:playlist:2"},
		},
	})

	if len(items) != 2 {
		t.Fatalf("itemsFromResults() len = %d, want 2", len(items))
	}

	track, ok := items[0].(resultItem)
	if !ok {
		t.Fatalf("items[0] type = %T, want resultItem", items[0])
	}
	if track.title != "Visible Track" {
		t.Fatalf("track title = %q, want %q", track.title, "Visible Track")
	}

	playlist, ok := items[1].(resultItem)
	if !ok {
		t.Fatalf("items[1] type = %T, want resultItem", items[1])
	}
	if playlist.title != "Visible Playlist" {
		t.Fatalf("playlist title = %q, want %q", playlist.title, "Visible Playlist")
	}
}

func TestFooterShowsLocalPlayerStatus(t *testing.T) {
	m := newModel(nil)
	m.width = 120
	m.height = 26
	m.connectionStatus = "Connected as tester"
	m.localPlayer = localPlayerStatus{
		supported:       true,
		binaryAvailable: true,
		process:         "running",
		device:          "spotui-speaker",
		message:         "ready",
	}

	layout := m.layoutMetrics()
	footer := m.footerPanel(layout.mainWidth, layout)

	if !strings.Contains(footer, "Local player: running") {
		t.Fatalf("expected footer to include local player status, got %q", footer)
	}
	if !strings.Contains(footer, "spotui-speaker") {
		t.Fatalf("expected footer to include local device name, got %q", footer)
	}
	if strings.Count(footer, "\n") > 1 {
		t.Fatalf("expected compact footer status band, got %q", footer)
	}
	if strings.Contains(footer, m.connectionStatus) {
		t.Fatalf("expected footer to omit connection status, got %q", footer)
	}
}

func TestNewModelDoesNotExposeDefaultSuccessAction(t *testing.T) {
	m := newModel(nil)
	m.width = 120
	m.height = 26
	m.connectionStatus = "Connected as tester"

	layout := m.layoutMetrics()
	footer := m.footerPanel(layout.mainWidth, layout)

	if strings.Contains(footer, "Search for something or use slash commands from the command dock.") {
		t.Fatalf("expected footer to omit the old default action copy, got %q", footer)
	}
}

func TestEmptyResultsHintIsShortAndDirective(t *testing.T) {
	m := newModel(nil)

	if got := m.emptyResultsHint(); got != "" {
		t.Fatalf("emptyResultsHint() = %q, want empty string", got)
	}
}

func TestEmptyResultsViewIsBlankWithoutContext(t *testing.T) {
	m := newModel(nil)

	rendered := m.emptyResultsView(60)
	if rendered != "" {
		t.Fatalf("expected empty results view to stay blank without extra context, got %q", rendered)
	}
}

func TestEmptySearchResultsPanelUsesTransientLoadingHeader(t *testing.T) {
	m := newModel(nil)
	m.width = 120
	m.height = 26
	m.bootFrames = 1

	layout := m.layoutMetrics()
	panel := m.resultsPanel(layout.mainWidth, layout)

	if strings.Contains(panel, "No results yet") {
		t.Fatalf("expected empty search panel to omit the no-results count label, got %q", panel)
	}
	if !strings.Contains(panel, "•") {
		t.Fatalf("expected empty search panel to show the boot loading header, got %q", panel)
	}
}

func TestEmptySearchResultsPanelGoesQuietAfterBoot(t *testing.T) {
	m := newModel(nil)
	m.width = 120
	m.height = 26
	m.bootAnimationDone = true

	layout := m.layoutMetrics()
	panel := m.resultsPanel(layout.mainWidth, layout)

	if strings.Contains(panel, "Search") {
		t.Fatalf("expected empty search panel to omit persistent search copy, got %q", panel)
	}
	if strings.TrimSpace(panel) != "" {
		t.Fatalf("expected empty search panel to be visually quiet after boot, got %q", panel)
	}
}

func TestFooterHidesHintsWhileBannerIsVisible(t *testing.T) {
	m := newModel(nil)
	m.width = 120
	m.height = 26
	m.connectionStatus = "Connected as tester"
	m.bannerText = "Spotify login expired. Run `spotui login` again."
	m.bannerIsError = true

	layout := m.layoutMetrics()
	footer := m.footerPanel(layout.mainWidth, layout)

	if !strings.Contains(footer, m.bannerText) {
		t.Fatalf("expected footer to include banner text, got %q", footer)
	}
	if strings.Contains(footer, "tab focus") {
		t.Fatalf("expected footer hints to be suppressed while a banner is visible, got %q", footer)
	}
}

func TestPlaybarStatusRowShowsConnectedUser(t *testing.T) {
	m := newModel(nil)
	m.connectionStatus = "Connected as Petros Xen (11124806036)"

	row := m.playbarStatusRow(120, "idle", "")
	if !strings.Contains(row, "Petros Xen") {
		t.Fatalf("expected status row to include the connected user, got %q", row)
	}
}

func TestPlaybarStatusRowShowsOfflineStateWhenNotConnected(t *testing.T) {
	m := newModel(nil)
	m.connectionStatus = "not logged in; run `spotui login` first"

	row := m.playbarStatusRow(120, "idle", "")
	if !strings.Contains(row, "offline") {
		t.Fatalf("expected status row to include offline state, got %q", row)
	}
}

func TestIdlePlaybarOmitsIdleSearchSubtitle(t *testing.T) {
	m := newModel(nil)
	m.width = 120
	m.height = 26

	layout := m.layoutMetrics()
	playbar := m.playbarView(layout)

	if strings.Contains(playbar, "Search tracks or playlists to start playback.") {
		t.Fatalf("expected idle playbar to omit the search subtitle, got %q", playbar)
	}
}
