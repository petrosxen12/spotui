package ui

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
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

func TestBadgeColorsAreDistinctHarmonicsOfDynamicAccent(t *testing.T) {
	m := newModel(nil)
	m.accentColor = "#8c7a69"

	track := m.trackBadgeColor()
	playlist := m.playlistBadgeColor()
	if track == playlist {
		t.Fatalf("badge colors should differ, both were %q", track)
	}
	if track == m.accentColor || playlist == m.accentColor {
		t.Fatalf("badge colors should be derived from the accent, got track=%q playlist=%q", track, playlist)
	}

	baseColor, err := colorful.Hex(m.accentColor)
	if err != nil {
		t.Fatal(err)
	}
	baseHue, _, _ := baseColor.Hcl()
	assertBadgeHue := func(name, raw string, wantShift float64) {
		t.Helper()
		color, err := colorful.Hex(raw)
		if err != nil {
			t.Fatalf("%s badge color %q is invalid: %v", name, raw, err)
		}
		hue, chroma, lightness := color.Hcl()
		wantHue := normalizeHue(baseHue + wantShift)
		hueDelta := math.Abs(hue - wantHue)
		hueDelta = math.Min(hueDelta, 360-hueDelta)
		if hueDelta > 1 {
			t.Fatalf("%s badge hue = %.2f, want %.2f (+/- 1)", name, hue, wantHue)
		}
		if chroma > 0.16 {
			t.Fatalf("%s badge chroma = %.3f, want a subtle color", name, chroma)
		}
		if lightness < 0.47 || lightness > 0.65 {
			t.Fatalf("%s badge lightness = %.3f, want balanced contrast", name, lightness)
		}
	}
	assertBadgeHue("track", track, -28)
	assertBadgeHue("playlist", playlist, 32)
}

func TestBadgeColorsRemainDistinctWithDefaultAccent(t *testing.T) {
	m := newModel(nil)
	if m.trackBadgeColor() == m.playlistBadgeColor() {
		t.Fatalf("default badge colors should differ, both were %q", m.trackBadgeColor())
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

	if len(items) != 4 {
		t.Fatalf("itemsFromResults() len = %d, want 4", len(items))
	}

	tracksHeader, ok := items[0].(sectionHeaderItem)
	if !ok {
		t.Fatalf("items[0] type = %T, want sectionHeaderItem", items[0])
	}
	if tracksHeader.title != "Tracks" {
		t.Fatalf("tracks header = %q, want %q", tracksHeader.title, "Tracks")
	}

	track, ok := items[1].(resultItem)
	if !ok {
		t.Fatalf("items[1] type = %T, want resultItem", items[1])
	}
	if track.title != "Visible Track" {
		t.Fatalf("track title = %q, want %q", track.title, "Visible Track")
	}

	playlistsHeader, ok := items[2].(sectionHeaderItem)
	if !ok {
		t.Fatalf("items[2] type = %T, want sectionHeaderItem", items[2])
	}
	if playlistsHeader.title != "Playlists" {
		t.Fatalf("playlists header = %q, want %q", playlistsHeader.title, "Playlists")
	}

	playlist, ok := items[3].(resultItem)
	if !ok {
		t.Fatalf("items[3] type = %T, want resultItem", items[3])
	}
	if playlist.title != "Visible Playlist" {
		t.Fatalf("playlist title = %q, want %q", playlist.title, "Visible Playlist")
	}
}

func TestItemsFromResultsOmitsEmptySection(t *testing.T) {
	items := itemsFromResults(app.Results{
		Playlists: []app.SearchItem{{Name: "Focus Mix", URI: "spotify:playlist:1"}},
	})

	if len(items) != 2 {
		t.Fatalf("itemsFromResults() len = %d, want 2", len(items))
	}
	header, ok := items[0].(sectionHeaderItem)
	if !ok || header.title != "Playlists" {
		t.Fatalf("first item = %#v, want Playlists section header", items[0])
	}
}

func TestSearchNavigationSkipsSectionHeaders(t *testing.T) {
	items := itemsFromResults(app.Results{
		Tracks: []app.SearchItem{
			{Name: "Track One", URI: "spotify:track:1"},
			{Name: "Track Two", URI: "spotify:track:2"},
		},
		Playlists: []app.SearchItem{{Name: "Playlist One", URI: "spotify:playlist:1"}},
	})
	delegate := resultDelegate{}
	listModel := list.New(items, delegate, 80, 20)
	selectFirstResult(&listModel)

	assertSelected := func(wantTitle string) {
		t.Helper()
		selected, ok := listModel.SelectedItem().(resultItem)
		if !ok {
			t.Fatalf("selected item type = %T, want resultItem", listModel.SelectedItem())
		}
		if selected.title != wantTitle {
			t.Fatalf("selected title = %q, want %q", selected.title, wantTitle)
		}
	}
	assertSelected("Track One")

	for _, want := range []string{"Track Two", "Playlist One"} {
		listModel, _ = listModel.Update(tea.KeyMsg{Type: tea.KeyDown})
		assertSelected(want)
	}

	listModel, _ = listModel.Update(tea.KeyMsg{Type: tea.KeyUp})
	assertSelected("Track Two")
}

func TestSearchListRendersHeadedSectionsInOrder(t *testing.T) {
	items := itemsFromResults(app.Results{
		Tracks:    []app.SearchItem{{Name: "Track One", URI: "spotify:track:1"}},
		Playlists: []app.SearchItem{{Name: "Playlist One", URI: "spotify:playlist:1"}},
	})
	delegate := resultDelegate{
		width:              80,
		wideLayout:         true,
		trackBadgeColor:    "#6688aa",
		playlistBadgeColor: "#aa8866",
	}
	listModel := list.New(items, delegate, 80, 20)
	listModel.SetShowTitle(false)
	listModel.SetShowStatusBar(false)
	listModel.SetShowPagination(false)
	listModel.SetShowHelp(false)
	selectFirstResult(&listModel)

	rendered := listModel.View()
	positions := []int{
		strings.Index(rendered, "Tracks"),
		strings.Index(rendered, "Track One"),
		strings.Index(rendered, "Playlists"),
		strings.Index(rendered, "Playlist One"),
	}
	for index, position := range positions {
		if position < 0 {
			t.Fatalf("expected section content at position %d in %q", index, rendered)
		}
		if index > 0 && position <= positions[index-1] {
			t.Fatalf("section content rendered out of order: %v", positions)
		}
	}
}

func TestSearchProgressIgnoresSectionHeaders(t *testing.T) {
	m := newModel(nil)
	m.listMode = listModeSearch
	m.list.SetItems(itemsFromResults(app.Results{
		Tracks:    []app.SearchItem{{Name: "Track One"}, {Name: "Track Two"}},
		Playlists: []app.SearchItem{{Name: "Playlist One"}},
	}))
	m.list.Select(2)

	progress := m.listProgressText()
	want := renderListProgress(1, 3) + " 66%"
	if progress != want {
		t.Fatalf("search progress = %q, want result-only progress %q", progress, want)
	}
}

func TestResultDelegateUsesDifferentBadgeColors(t *testing.T) {
	m := newModel(nil)
	m.accentColor = "#8c7a69"
	delegate := resultDelegate{
		trackBadgeColor:    m.trackBadgeColor(),
		playlistBadgeColor: m.playlistBadgeColor(),
	}

	trackForeground := fmt.Sprint(delegate.badgeStyle("track", metaPillStyle).GetForeground())
	playlistForeground := fmt.Sprint(delegate.badgeStyle("playlist", metaPillStyle).GetForeground())
	if trackForeground == playlistForeground {
		t.Fatalf("badge foregrounds should differ, both were %q", trackForeground)
	}
	if trackForeground != m.trackBadgeColor() {
		t.Fatalf("track badge foreground = %q, want %q", trackForeground, m.trackBadgeColor())
	}
	if playlistForeground != m.playlistBadgeColor() {
		t.Fatalf("playlist badge foreground = %q, want %q", playlistForeground, m.playlistBadgeColor())
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

func TestResultsPanelOmitsExplicitFocusStateCopy(t *testing.T) {
	m := newModel(nil)
	m.width = 120
	m.height = 26
	m.query = "boards of canada"
	m.resultCount = 3
	m.list.SetItems([]list.Item{
		resultItem{title: "Dayvan Cowboy", description: "track", kind: "track"},
	})

	layout := m.layoutMetrics()

	panel := m.resultsPanel(layout.mainWidth, layout)
	if strings.Contains(panel, "command active") {
		t.Fatalf("expected input-focused results panel to omit command focus copy, got %q", panel)
	}

	m.inputFocused = false
	m.resize()
	panel = m.resultsPanel(layout.mainWidth, layout)
	if strings.Contains(panel, "list active") {
		t.Fatalf("expected list-focused results panel to omit list focus copy, got %q", panel)
	}
}

func TestViewRendersCommandBarBeforeResults(t *testing.T) {
	m := newModel(nil)
	m.width = 120
	m.height = 26
	m.localPlayer = localPlayerStatus{
		supported:       true,
		binaryAvailable: true,
		process:         "running",
		device:          "spotui-speaker",
	}

	view := m.View()

	commandIndex := strings.Index(view, "search or /")
	resultsIndex := strings.Index(view, "Waiting for spotui-speaker to become the active output.")
	if commandIndex == -1 || resultsIndex == -1 {
		t.Fatalf("expected view to include command bar and results panel, got %q", view)
	}
	if commandIndex > resultsIndex {
		t.Fatalf("expected command bar to render before results, got %q", view)
	}
}
