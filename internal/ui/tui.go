package ui

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/petrosxen/spotui/internal/app"
)

const (
	playbackPollActive   = 1500 * time.Millisecond
	playbackPollIdle     = 4 * time.Second
	playbackPollNoDevice = 6 * time.Second
)

var (
	pageStyle = lipgloss.NewStyle().
			Padding(0, 1)

	heroStyle = lipgloss.NewStyle().
			MarginBottom(1)

	playbarStyle = lipgloss.NewStyle().
			MarginBottom(1)

	panelStyle = lipgloss.NewStyle().
			Padding(0, 0)

	panelFocusedStyle = panelStyle.Copy().
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"})

	dockStyle = lipgloss.NewStyle().
			MarginTop(1)

	dockFocusedStyle = dockStyle.Copy().
				BorderTop(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"})

	eyebrowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#A0A0A0"}).
			Bold(true)

	kickerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#A0A0A0"})

	titleStyle = lipgloss.NewStyle().
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#555555", Dark: "#B8B8B8"})

	metaPillStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#A0A0A0"})

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#555555", Dark: "#B8B8B8"})

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#444444", Dark: "#D0D0D0"})

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#8A3B2E", Dark: "#FF8A7A"})

	commandHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#9A9A9A"})

	inputShellStyle = lipgloss.NewStyle().
			Padding(0, 0)

	suggestionPopupStyle = lipgloss.NewStyle().
				Padding(0, 0).
				MarginBottom(1).
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"})

	suggestionSelectedStyle = lipgloss.NewStyle().
				Bold(true)

	selectedRowStyle = lipgloss.NewStyle().
				Padding(0, 0).
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.AdaptiveColor{Light: "#444444", Dark: "#D0D0D0"})

	selectedTitleStyle = lipgloss.NewStyle().
				Bold(true)

	selectedDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#555555", Dark: "#B8B8B8"})

	rowTitleStyle = lipgloss.NewStyle().
			UnsetForeground()

	rowDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#9A9A9A"})
)

type resultDelegate struct{}

func (d resultDelegate) Height() int  { return 2 }
func (d resultDelegate) Spacing() int { return 1 }
func (d resultDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d resultDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var titleText string
	var descText string
	var kind string

	switch entry := item.(type) {
	case resultItem:
		titleText = entry.title
		descText = entry.description
		kind = entry.kind
	case deviceItem:
		titleText = entry.title
		descText = entry.description
		kind = "device"
	case infoItem:
		titleText = entry.title
		descText = entry.description
		kind = "info"
	default:
		return
	}

	meta := metaPillStyle.Render("[" + strings.ToUpper(kind) + "]")
	title := rowTitleStyle.Render(titleText)
	desc := rowDescStyle.Render(descText)

	line1 := lipgloss.JoinHorizontal(lipgloss.Left, meta, " ", title)
	line2 := "  " + desc

	if index == m.Index() {
		block := strings.Join([]string{
			selectedTitleStyle.Render("> " + line1),
			selectedDescStyle.Render("  " + descText),
		}, "\n")
		fmt.Fprint(w, selectedRowStyle.Render(block))
		return
	}

	fmt.Fprint(w, strings.Join([]string{"  " + line1, line2}, "\n"))
}

type resultItem struct {
	title       string
	description string
	kind        string
	uri         string
}

func (i resultItem) Title() string       { return i.title }
func (i resultItem) Description() string { return i.description }
func (i resultItem) FilterValue() string { return i.title + " " + i.description + " " + i.kind }

type deviceItem struct {
	title       string
	description string
	id          string
}

func (i deviceItem) Title() string       { return i.title }
func (i deviceItem) Description() string { return i.description }
func (i deviceItem) FilterValue() string { return i.title + " " + i.description + " " + i.id }

type infoItem struct {
	title       string
	description string
}

func (i infoItem) Title() string       { return i.title }
func (i infoItem) Description() string { return i.description }
func (i infoItem) FilterValue() string { return i.title + " " + i.description }

type suggestion struct {
	value       string
	insertValue string
	description string
}

type slashCommand struct {
	name        string
	usage       string
	description string
}

type listMode string

const (
	listModeSearch  listMode = "search"
	listModeDevices listMode = "devices"
	listModeHelp    listMode = "help"
)

var slashCommands = []slashCommand{
	{name: "/help", usage: "/help", description: "show command help"},
	{name: "/next", usage: "/next", description: "skip to the next item"},
	{name: "/prev", usage: "/prev", description: "go back to the previous item"},
	{name: "/pause", usage: "/pause", description: "pause playback"},
	{name: "/resume", usage: "/resume", description: "resume playback"},
	{name: "/devices", usage: "/devices", description: "show available devices in the list"},
	{name: "/device", usage: "/device <name>", description: "select a device by name substring"},
	{name: "/play", usage: "/play <query|index>", description: "play from the last search results"},
	{name: "/quit", usage: "/quit", description: "quit the TUI"},
}

type connectionMsg struct {
	user app.User
	err  error
}

type searchMsg struct {
	query   string
	results app.Results
	err     error
}

type playbackMsg struct {
	state app.PlaybackState
	err   error
}

type actionMsg struct {
	text string
	err  error
}

type devicesMsg struct {
	devices     []app.Device
	err         error
	pushHistory bool
}

type deviceSelectedMsg struct {
	device app.Device
	err    error
}

type helpMsg struct{}

type accentColorMsg struct {
	albumArtURL string
	color       string
	err         error
}

type pollTickMsg struct{}

type model struct {
	service          app.PlayerService
	list             list.Model
	input            textinput.Model
	width            int
	height           int
	inputFocused     bool
	connectionStatus string
	lastAction       string
	lastActionErr    bool
	query            string
	lastResults      app.Results
	listMode         listMode
	playback         app.PlaybackState
	pollEvery        time.Duration
	resultCount      int
	suggestions      []suggestion
	suggestionIndex  int
	suggestionsOpen  bool
	accentColor      string
	accentColorCache map[string]string
	viewHistory      []viewState
}

type viewState struct {
	listItems     []list.Item
	listMode      listMode
	query         string
	lastResults   app.Results
	resultCount   int
	inputValue    string
	lastAction    string
	lastActionErr bool
}

type layoutMetrics struct {
	bodyWidth           int
	compact             bool
	playbarProgressLen  int
	listHeight          int
	resultsChromeHeight int
	inputWidth          int
	pagePaddingX        int
	pagePaddingY        int
}

func Run(service app.PlayerService) error {
	m := newModel(service)
	program := tea.NewProgram(m, tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func newModel(service app.PlayerService) model {
	results := list.New([]list.Item{}, resultDelegate{}, 0, 0)
	results.Title = ""
	results.SetShowStatusBar(false)
	results.SetShowPagination(false)
	results.SetShowHelp(false)
	results.SetFilteringEnabled(false)
	results.DisableQuitKeybindings()

	input := textinput.New()
	input.Placeholder = "Search tracks, playlists, or enter /pause /resume /next /prev"
	input.Prompt = "› "
	input.Focus()
	input.CharLimit = 256

	return model{
		service:          service,
		list:             results,
		input:            input,
		inputFocused:     true,
		connectionStatus: "Connecting to Spotify...",
		lastAction:       "Search for something or use slash commands from the command dock.",
		listMode:         listModeSearch,
		pollEvery:        playbackPollIdle,
		accentColorCache: make(map[string]string),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		checkConnectionCmd(m.service),
		fetchPlaybackCmd(m.service),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			if m.inputFocused && m.suggestionsOpen {
				m.cycleSuggestion()
				return m, nil
			}
			m.toggleFocus()
			return m, nil
		}

		if m.inputFocused {
			switch msg.String() {
			case "esc":
				if m.suggestionsOpen {
					m.closeSuggestions()
					return m, nil
				}
				if m.popViewState() {
					return m, nil
				}
				m.input.SetValue("")
				m.closeSuggestions()
				return m, nil
			case "right", "ctrl+space", "ctrl+@":
				if m.suggestionsOpen {
					m.acceptSuggestion()
					return m, nil
				}
			case "enter":
				return m, m.submitInput()
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.refreshSuggestions()
			return m, cmd
		}

		switch msg.String() {
		case "esc":
			if m.popViewState() {
				return m, nil
			}
			m.toggleFocus()
			m.input.SetValue("")
			return m, nil
		case "enter":
			switch selected := m.list.SelectedItem().(type) {
			case resultItem:
				m.lastAction = fmt.Sprintf("Playing %s: %s", selected.kind, selected.title)
				return m, playSelectionCmd(m.service, selected)
			case deviceItem:
				m.lastAction = fmt.Sprintf("Selecting device: %s", selected.title)
				return m, selectDeviceByIDCmd(m.service, selected.id, selected.title)
			}
			return m, nil
		case "/":
			m.toggleFocus()
			m.input.SetValue("/")
			m.refreshSuggestions()
			return m, nil
		}

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case connectionMsg:
		if msg.err != nil {
			m.connectionStatus = msg.err.Error()
			return m, nil
		}
		if msg.user.DisplayName != "" {
			m.connectionStatus = fmt.Sprintf("Connected as %s (%s)", msg.user.DisplayName, msg.user.ID)
		} else {
			m.connectionStatus = fmt.Sprintf("Connected as %s", msg.user.ID)
		}
		return m, nil
	case searchMsg:
		if msg.err != nil {
			m.lastAction = msg.err.Error()
			m.lastActionErr = true
			return m, nil
		}
		m.pushViewState()
		m.query = msg.query
		m.lastResults = msg.results
		m.listMode = listModeSearch
		m.list.SetItems(itemsFromResults(msg.results))
		m.list.Select(0)
		m.resultCount = len(msg.results.Tracks) + len(msg.results.Playlists)
		m.lastAction = fmt.Sprintf("Loaded %d results for %q", m.resultCount, msg.query)
		m.lastActionErr = false
		return m, nil
	case devicesMsg:
		if msg.err != nil {
			m.lastAction = msg.err.Error()
			m.lastActionErr = true
			return m, nil
		}
		if msg.pushHistory {
			m.pushViewState()
		}
		m.listMode = listModeDevices
		m.list.SetItems(itemsFromDevices(msg.devices))
		m.list.Select(0)
		m.resultCount = len(msg.devices)
		m.lastAction = fmt.Sprintf("Loaded %d devices", len(msg.devices))
		m.lastActionErr = false
		return m, nil
	case deviceSelectedMsg:
		if msg.err != nil {
			m.lastAction = msg.err.Error()
			m.lastActionErr = true
			return m, nil
		}
		m.lastAction = fmt.Sprintf("Selected device: %s", msg.device.Name)
		m.lastActionErr = false
		return m, fetchDevicesCmd(m.service, false)
	case helpMsg:
		m.pushViewState()
		m.listMode = listModeHelp
		helpItems := make([]list.Item, 0, len(slashCommands))
		for _, command := range slashCommands {
			helpItems = append(helpItems, infoItem{
				title:       command.usage,
				description: command.description,
			})
		}
		m.list.SetItems(helpItems)
		m.list.Select(0)
		m.resultCount = len(helpItems)
		m.lastAction = "Command reference"
		m.lastActionErr = false
		return m, nil
	case playbackMsg:
		if msg.err == nil {
			m.playback = msg.state
			m.pollEvery = nextPollInterval(msg.state)
			if cmd := m.refreshAccentColor(); cmd != nil {
				return m, tea.Batch(pollPlaybackCmd(m.pollEvery), cmd)
			}
		} else {
			m.lastAction = msg.err.Error()
			m.lastActionErr = true
			m.pollEvery = playbackPollIdle
		}
		return m, pollPlaybackCmd(m.pollEvery)
	case accentColorMsg:
		if msg.err != nil {
			return m, nil
		}
		if msg.albumArtURL != "" {
			m.accentColorCache[msg.albumArtURL] = msg.color
			if m.playback.AlbumArtURL == msg.albumArtURL {
				m.accentColor = msg.color
			}
		}
		return m, nil
	case actionMsg:
		if msg.err != nil {
			m.lastAction = msg.err.Error()
			m.lastActionErr = true
			return m, nil
		}
		m.lastAction = msg.text
		m.lastActionErr = false
		return m, fetchPlaybackCmd(m.service)
	case pollTickMsg:
		return m, fetchPlaybackCmd(m.service)
	}

	var cmd tea.Cmd
	if m.inputFocused {
		m.input, cmd = m.input.Update(msg)
	} else {
		m.list, cmd = m.list.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Loading spotui..."
	}

	layout := m.layoutMetrics()
	m.resizeWithLayout(layout)

	page := pageStyle.Copy().Padding(layout.pagePaddingY, layout.pagePaddingX)
	playbar := playbarStyle.Width(layout.bodyWidth).Render(m.playbarView(layout))

	dockStyleToUse := dockStyle
	if m.inputFocused {
		dockStyleToUse = dockFocusedStyle
	}
	dock := dockStyleToUse.Width(layout.bodyWidth).Render(m.commandDockView(layout))

	return page.Width(m.width).Render(strings.Join([]string{
		playbar,
		m.resultsPanel(layout.bodyWidth, layout),
		m.footerPanel(layout.bodyWidth, layout),
		dock,
	}, "\n"))
}

func (m *model) resize() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	m.resizeWithLayout(m.layoutMetrics())
}

func (m *model) resizeWithLayout(layout layoutMetrics) {
	m.list.SetSize(maxInt(20, layout.bodyWidth), maxInt(1, layout.listHeight-layout.resultsChromeHeight))
	m.input.Width = layout.inputWidth
}

func (m model) layoutMetrics() layoutMetrics {
	paddingX := 1
	paddingY := 1
	if m.width < 70 {
		paddingX = 0
	}

	bodyWidth := m.width - (paddingX * 2)
	if bodyWidth < 20 {
		bodyWidth = 20
	}

	compact := bodyWidth < 72 || m.height < 26

	playbarProgressLen := clampInt(bodyWidth/4, 12, 32)
	inputWidth := maxInt(10, bodyWidth-2)
	resultsChromeHeight := 3

	tempLayout := layoutMetrics{
		bodyWidth:           bodyWidth,
		compact:             compact,
		playbarProgressLen:  playbarProgressLen,
		inputWidth:          inputWidth,
		pagePaddingX:        paddingX,
		pagePaddingY:        paddingY,
		resultsChromeHeight: resultsChromeHeight,
	}

	pageVertical := paddingY * 2
	playbarHeight := lipgloss.Height(playbarStyle.Width(bodyWidth).Render(m.playbarView(tempLayout)))
	footerHeight := lipgloss.Height(m.footerPanel(bodyWidth, tempLayout))
	dockHeight := lipgloss.Height(dockStyle.Width(bodyWidth).Render(m.commandDockView(tempLayout)))
	sectionGaps := 3
	available := m.height - pageVertical - playbarHeight - footerHeight - dockHeight - sectionGaps
	listHeight := clampInt(available, 6, maxInt(6, available))

	return layoutMetrics{
		bodyWidth:           bodyWidth,
		compact:             compact,
		playbarProgressLen:  playbarProgressLen,
		listHeight:          listHeight,
		resultsChromeHeight: resultsChromeHeight,
		inputWidth:          inputWidth,
		pagePaddingX:        paddingX,
		pagePaddingY:        paddingY,
	}
}

func (m *model) toggleFocus() {
	m.inputFocused = !m.inputFocused
	if m.inputFocused {
		m.input.Focus()
		m.refreshSuggestions()
		return
	}
	m.input.Blur()
	m.closeSuggestions()
}

func (m *model) submitInput() tea.Cmd {
	value := strings.TrimSpace(m.input.Value())
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, "/") && m.suggestionsOpen && !m.hasExactCommand(value) {
		m.acceptSuggestion()
		return nil
	}
	m.input.SetValue("")
	m.lastAction = fmt.Sprintf("Running %q...", value)
	m.lastActionErr = false
	if strings.HasPrefix(value, "/") {
		m.closeSuggestions()
		return m.runSlashCommand(value)
	}
	return searchCmd(m.service, value)
}

func (m *model) refreshAccentColor() tea.Cmd {
	if !m.playback.IsPlaying || m.playback.AlbumArtURL == "" {
		m.accentColor = ""
		return nil
	}
	if cached, ok := m.accentColorCache[m.playback.AlbumArtURL]; ok {
		m.accentColor = cached
		return nil
	}
	return fetchAccentColorCmd(m.playback.AlbumArtURL)
}

func (m *model) pushViewState() {
	snapshot := viewState{
		listItems:     append([]list.Item(nil), m.list.Items()...),
		listMode:      m.listMode,
		query:         m.query,
		lastResults:   m.lastResults,
		resultCount:   m.resultCount,
		inputValue:    m.input.Value(),
		lastAction:    m.lastAction,
		lastActionErr: m.lastActionErr,
	}
	m.viewHistory = append(m.viewHistory, snapshot)
}

func (m *model) popViewState() bool {
	if len(m.viewHistory) == 0 {
		m.listMode = listModeSearch
		m.query = ""
		m.lastResults = app.Results{}
		m.resultCount = 0
		m.list.SetItems(nil)
		m.list.Select(0)
		m.input.SetValue("")
		m.inputFocused = true
		m.input.Focus()
		m.closeSuggestions()
		return false
	}

	idx := len(m.viewHistory) - 1
	snapshot := m.viewHistory[idx]
	m.viewHistory = m.viewHistory[:idx]

	m.listMode = snapshot.listMode
	m.query = snapshot.query
	m.lastResults = snapshot.lastResults
	m.resultCount = snapshot.resultCount
	m.list.SetItems(snapshot.listItems)
	m.list.Select(0)
	m.input.SetValue(snapshot.inputValue)
	m.lastAction = snapshot.lastAction
	m.lastActionErr = snapshot.lastActionErr
	m.inputFocused = true
	m.input.Focus()
	m.closeSuggestions()
	return true
}

func (m model) playbarView(layout layoutMetrics) string {
	status := "Idle"
	if m.playback.Device.ID != "" && !m.playback.IsPlaying {
		status = "Paused"
	}
	if m.playback.IsPlaying {
		status = "Playing"
	}

	title := "Nothing playing"
	if m.playback.ItemName != "" {
		title = m.playback.ItemName
	}

	artist := "Search for a track or choose a result to start playback."
	if m.playback.ArtistName != "" {
		artist = m.playback.ArtistName
	}

	device := "No active device"
	if m.playback.Device.Name != "" {
		device = m.playback.Device.Name
	}

	progress := m.progressBar(layout.playbarProgressLen)
	timing := formatDuration(m.playback.Progress) + " / " + formatDuration(m.playback.Duration)

	left := lipgloss.JoinHorizontal(
		lipgloss.Left,
		eyebrowStyle.Render("spotui"),
		"  ",
		m.playingStatusStyle().Render(strings.ToLower(status)),
		"  ",
		m.nowPlayingTitleStyle().Render(title),
	)
	right := lipgloss.JoinHorizontal(
		lipgloss.Left,
		subtitleStyle.Render(artist),
		"  ·  ",
		metaPillStyle.Render(device),
		"  ",
		kickerStyle.Render(progress+"  "+timing),
	)
	if layout.compact {
		return strings.Join([]string{
			left,
			subtitleStyle.Render(artist),
			kickerStyle.Render(progress + "  " + timing),
		}, "\n")
	}
	return heroStyle.Render(strings.Join([]string{left, right}, "\n"))
}

func (m model) resultsPanel(width int, layout layoutMetrics) string {
	style := panelStyle
	if !m.inputFocused {
		style = panelFocusedStyle
	}

	header := "Results"
	switch m.listMode {
	case listModeDevices:
		header = "Devices"
	case listModeHelp:
		header = "Commands"
	case listModeSearch:
		if m.query != "" {
			header = fmt.Sprintf("Results for %q", m.query)
		}
	}

	countLabel := "No items yet"
	switch m.listMode {
	case listModeDevices:
		countLabel = fmt.Sprintf("%d devices", m.resultCount)
	case listModeHelp:
		countLabel = fmt.Sprintf("%d commands", m.resultCount)
	case listModeSearch:
		countLabel = "No results yet"
		if m.resultCount > 0 {
			countLabel = fmt.Sprintf("%d items", m.resultCount)
		}
	}

	body := m.list.View()
	if m.resultCount == 0 {
		switch m.listMode {
		case listModeDevices:
			body = subtitleStyle.Render("Run /devices to refresh the available playback devices.")
		case listModeHelp:
			body = subtitleStyle.Render("Type /help to load the command list.")
		default:
			body = subtitleStyle.Render("Type a query below and let the list settle into view.")
		}
	}

	lines := []string{eyebrowStyle.Render(header)}
	if layout.compact {
		lines = append(lines, kickerStyle.Render(countLabel+"  "+m.listProgressText()))
	} else {
		lines = append(lines, kickerStyle.Render(countLabel+"  "+m.listProgressText()))
	}
	lines = append(lines, "", body)
	return style.Width(width).Render(strings.Join(lines, "\n"))
}

func (m model) footerPanel(width int, layout layoutMetrics) string {
	statusTone := infoStyle
	if m.lastActionErr {
		statusTone = errorStyle
	} else if m.lastAction != "" {
		statusTone = successStyle
	}

	lines := []string{
		infoStyle.Render(m.connectionStatus),
		statusTone.Render(m.lastAction),
	}
	if !layout.compact {
		lines = append(lines, commandHintStyle.Render("tab focus  ·  enter select  ·  / commands  ·  q quit"))
	}
	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func (m model) commandDockView(layout layoutMetrics) string {
	inputView := inputShellStyle.Width(maxInt(10, layout.bodyWidth-4)).Render(m.input.View())
	lines := []string{}
	if popup := m.suggestionsView(layout); popup != "" {
		lines = append(lines, popup)
	}
	if layout.compact {
		lines = append(lines, inputView)
	} else {
		lines = append(lines, inputView)
	}
	return strings.Join(lines, "\n")
}

func itemsFromResults(results app.Results) []list.Item {
	items := make([]list.Item, 0, len(results.Tracks)+len(results.Playlists))
	for _, track := range results.Tracks {
		description := "track"
		if track.Subtitle != "" {
			description = track.Subtitle
		}
		items = append(items, resultItem{
			title:       track.Name,
			description: description,
			kind:        "track",
			uri:         track.URI,
		})
	}
	for _, playlist := range results.Playlists {
		description := "playlist"
		if playlist.Subtitle != "" {
			description = "playlist by " + playlist.Subtitle
		}
		items = append(items, resultItem{
			title:       playlist.Name,
			description: description,
			kind:        "playlist",
			uri:         playlist.URI,
		})
	}
	return items
}

func itemsFromDevices(devices []app.Device) []list.Item {
	items := make([]list.Item, 0, len(devices))
	for _, device := range devices {
		state := device.Type
		if device.IsActive {
			state += " • active"
		}
		items = append(items, deviceItem{
			title:       device.Name,
			description: state,
			id:          device.ID,
		})
	}
	return items
}

func (m model) suggestionsView(layout layoutMetrics) string {
	if !m.suggestionsOpen || len(m.suggestions) == 0 {
		return ""
	}
	visible := m.suggestions
	if len(visible) > 5 {
		visible = visible[:5]
	}
	lines := make([]string, 0, len(visible))
	for i, suggestion := range visible {
		line := suggestion.value
		if suggestion.description != "" {
			line += "  " + suggestion.description
		}
		if i == m.suggestionIndex {
			lines = append(lines, suggestionSelectedStyle.Render(line))
		} else {
			lines = append(lines, commandHintStyle.Render(line))
		}
	}
	return suggestionPopupStyle.Width(maxInt(20, layout.bodyWidth-4)).Render(strings.Join(lines, "\n"))
}

func (m model) listProgressText() string {
	total := len(m.list.Items())
	if total <= 1 {
		return ""
	}

	index := m.list.Index()
	if index < 0 {
		index = 0
	}
	if index >= total {
		index = total - 1
	}

	percent := int(float64(index+1) / float64(total) * 100)
	return renderListProgress(index, total) + fmt.Sprintf(" %d%%", percent)
}

func renderListProgress(index, total int) string {
	if total <= 1 {
		return ""
	}

	const width = 8
	filled := int(math.Round(float64(index+1) / float64(total) * float64(width)))
	if filled < 1 {
		filled = 1
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("•", filled) + strings.Repeat("·", width-filled)
}

func (m model) playingStatusStyle() lipgloss.Style {
	if m.playback.IsPlaying && m.accentColor != "" {
		return metaPillStyle.Copy().Foreground(lipgloss.Color(m.accentColor)).Bold(true)
	}
	return metaPillStyle
}

func (m model) nowPlayingTitleStyle() lipgloss.Style {
	if m.playback.IsPlaying && m.accentColor != "" {
		return titleStyle.Copy().Foreground(lipgloss.Color(m.accentColor))
	}
	return titleStyle
}

func (m *model) refreshSuggestions() {
	value := m.input.Value()
	suggestions := m.buildSuggestions(value)
	if len(suggestions) == 0 {
		m.closeSuggestions()
		return
	}
	m.suggestions = suggestions
	if m.suggestionIndex >= len(m.suggestions) {
		m.suggestionIndex = 0
	}
	m.suggestionsOpen = true
}

func (m *model) closeSuggestions() {
	m.suggestions = nil
	m.suggestionIndex = 0
	m.suggestionsOpen = false
}

func (m *model) cycleSuggestion() {
	if len(m.suggestions) == 0 {
		return
	}
	m.suggestionIndex = (m.suggestionIndex + 1) % len(m.suggestions)
}

func (m *model) acceptSuggestion() {
	if !m.suggestionsOpen || len(m.suggestions) == 0 {
		return
	}
	m.input.SetValue(m.suggestions[m.suggestionIndex].insertValue)
	m.input.CursorEnd()
	m.refreshSuggestions()
}

func (m model) buildSuggestions(raw string) []suggestion {
	if !strings.HasPrefix(raw, "/") {
		return nil
	}

	command, arg, hasSpace := parseSlashInput(raw)
	if !hasSpace {
		prefix := "/" + command
		matches := make([]suggestion, 0)
		for _, candidate := range slashCommands {
			if strings.HasPrefix(candidate.name, prefix) {
				matches = append(matches, suggestion{
					value:       candidate.name,
					insertValue: candidate.name,
					description: candidate.description,
				})
			}
		}
		return matches
	}

	switch command {
	case "device":
		devices, err := m.service.ListDevices(context.Background())
		if err != nil {
			return nil
		}
		prefix := strings.ToLower(strings.TrimSpace(arg))
		matches := make([]suggestion, 0)
		for _, device := range devices {
			name := device.Name
			if prefix == "" || strings.HasPrefix(strings.ToLower(name), prefix) {
				matches = append(matches, suggestion{
					value:       name,
					insertValue: "/device " + name,
					description: strings.ToLower(device.Type),
				})
			}
		}
		return matches
	case "play":
		return m.playSuggestions(strings.TrimSpace(arg))
	default:
		return nil
	}
}

func (m model) playSuggestions(prefix string) []suggestion {
	playables := m.lastPlayableItems()
	if len(playables) == 0 {
		return nil
	}
	prefix = strings.ToLower(prefix)
	matches := make([]suggestion, 0)
	for i, item := range playables {
		label := fmt.Sprintf("%d. %s", i+1, item.title)
		if prefix == "" || strings.HasPrefix(strings.ToLower(item.title), prefix) {
			matches = append(matches, suggestion{
				value:       label,
				insertValue: "/play " + item.title,
				description: item.kind,
			})
		}
	}
	return matches
}

func parseSlashInput(raw string) (command string, arg string, hasSpace bool) {
	trimmed := strings.TrimPrefix(raw, "/")
	if idx := strings.Index(trimmed, " "); idx >= 0 {
		return trimmed[:idx], strings.TrimLeft(trimmed[idx+1:], " "), true
	}
	return trimmed, "", false
}

func (m model) hasExactCommand(raw string) bool {
	command, _, _ := parseSlashInput(raw)
	for _, candidate := range slashCommands {
		if strings.TrimPrefix(candidate.name, "/") == command {
			return true
		}
	}
	return false
}

func (m model) runSlashCommand(raw string) tea.Cmd {
	command, arg, hasSpace := parseSlashInput(raw)
	if !hasSpace && !m.hasExactCommand(raw) && len(m.suggestions) > 0 {
		m.acceptSuggestion()
		return nil
	}

	switch command {
	case "help":
		return func() tea.Msg { return helpMsg{} }
	case "next":
		return func() tea.Msg {
			return actionMsg{text: "Skipped to next item", err: m.service.Next(context.Background())}
		}
	case "prev":
		return func() tea.Msg {
			return actionMsg{text: "Returned to previous item", err: m.service.Prev(context.Background())}
		}
	case "pause":
		return func() tea.Msg { return actionMsg{text: "Paused playback", err: m.service.Pause(context.Background())} }
	case "resume":
		return func() tea.Msg {
			return actionMsg{text: "Resumed playback", err: m.service.Resume(context.Background())}
		}
	case "devices":
		return fetchDevicesCmd(m.service, true)
	case "device":
		needle := strings.TrimSpace(arg)
		if needle == "" {
			return func() tea.Msg { return actionMsg{text: "Usage: /device <name>", err: nil} }
		}
		return selectDeviceByNameCmd(m.service, needle)
	case "play":
		ref := strings.TrimSpace(arg)
		if ref == "" {
			return func() tea.Msg { return actionMsg{text: "Usage: /play <query|index>", err: nil} }
		}
		return m.playFromLastResultsCmd(ref)
	case "quit":
		return tea.Quit
	default:
		return func() tea.Msg { return actionMsg{text: fmt.Sprintf("Unknown command /%s", command), err: nil} }
	}
}

func (m model) playFromLastResultsCmd(ref string) tea.Cmd {
	return func() tea.Msg {
		item, err := m.resolvePlayable(ref)
		if err != nil {
			return actionMsg{text: err.Error(), err: err}
		}
		switch item.kind {
		case "track":
			return actionMsg{text: "Playing track: " + item.title, err: m.service.PlayTrack(context.Background(), item.uri)}
		case "playlist":
			return actionMsg{text: "Playing playlist: " + item.title, err: m.service.PlayPlaylist(context.Background(), item.uri)}
		default:
			err := fmt.Errorf("unsupported item type %q", item.kind)
			return actionMsg{text: err.Error(), err: err}
		}
	}
}

func (m model) resolvePlayable(ref string) (resultItem, error) {
	playables := m.lastPlayableItems()
	if len(playables) == 0 {
		return resultItem{}, fmt.Errorf("no search results available; run a search first")
	}
	if idx, err := strconv.Atoi(ref); err == nil {
		if idx <= 0 || idx > len(playables) {
			return resultItem{}, fmt.Errorf("play index %d is out of range", idx)
		}
		return playables[idx-1], nil
	}

	ref = strings.ToLower(strings.TrimSpace(ref))
	matches := make([]resultItem, 0)
	for _, item := range playables {
		if strings.Contains(strings.ToLower(item.title), ref) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return resultItem{}, fmt.Errorf("no search result matched %q", ref)
	case 1:
		return matches[0], nil
	default:
		return resultItem{}, fmt.Errorf("multiple search results matched %q; use /play <index>", ref)
	}
}

func (m model) lastPlayableItems() []resultItem {
	items := itemsFromResults(m.lastResults)
	playables := make([]resultItem, 0, len(items))
	for _, item := range items {
		if playable, ok := item.(resultItem); ok {
			playables = append(playables, playable)
		}
	}
	return playables
}

func (m model) progressBar(width int) string {
	if m.playback.Duration <= 0 || width <= 0 {
		return strings.Repeat("─", maxInt(8, width))
	}

	filled := int((m.playback.Progress * time.Duration(width)) / m.playback.Duration)
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	if filled == 0 {
		return "○" + strings.Repeat("─", width-1)
	}
	if filled >= width {
		return m.accent(strings.Repeat("━", width))
	}

	active := strings.Repeat("━", maxInt(0, filled-1)) + "●"
	return m.accent(active) + strings.Repeat("─", width-filled)
}

func formatDuration(value time.Duration) string {
	if value <= 0 {
		return "0:00"
	}
	totalSeconds := int(value.Round(time.Second).Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func nextPollInterval(state app.PlaybackState) time.Duration {
	if state.Device.ID == "" {
		return playbackPollNoDevice
	}
	if !state.IsPlaying {
		return playbackPollIdle
	}
	return playbackPollActive
}

func checkConnectionCmd(service app.PlayerService) tea.Cmd {
	return func() tea.Msg {
		user, err := service.CurrentUser(context.Background())
		return connectionMsg{user: user, err: err}
	}
}

func searchCmd(service app.PlayerService, query string) tea.Cmd {
	return func() tea.Msg {
		results, err := service.Search(context.Background(), query)
		return searchMsg{query: query, results: results, err: err}
	}
}

func fetchDevicesCmd(service app.PlayerService, pushHistory bool) tea.Cmd {
	return func() tea.Msg {
		devices, err := service.ListDevices(context.Background())
		return devicesMsg{devices: devices, err: err, pushHistory: pushHistory}
	}
}

func selectDeviceByNameCmd(service app.PlayerService, name string) tea.Cmd {
	return func() tea.Msg {
		device, err := service.SetDeviceByName(context.Background(), name)
		return deviceSelectedMsg{device: device, err: err}
	}
}

func selectDeviceByIDCmd(service app.PlayerService, id string, name string) tea.Cmd {
	return func() tea.Msg {
		err := service.SetDeviceByID(context.Background(), id)
		return deviceSelectedMsg{device: app.Device{ID: id, Name: name}, err: err}
	}
}

func fetchPlaybackCmd(service app.PlayerService) tea.Cmd {
	return func() tea.Msg {
		state, err := service.GetPlaybackState(context.Background())
		return playbackMsg{state: state, err: err}
	}
}

func fetchAccentColorCmd(albumArtURL string) tea.Cmd {
	return func() tea.Msg {
		color, err := dominantColorFromImageURL(albumArtURL)
		return accentColorMsg{albumArtURL: albumArtURL, color: color, err: err}
	}
}

func pollPlaybackCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return pollTickMsg{}
	})
}

func playSelectionCmd(service app.PlayerService, item resultItem) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch item.kind {
		case "track":
			err = service.PlayTrack(context.Background(), item.uri)
		case "playlist":
			err = service.PlayPlaylist(context.Background(), item.uri)
		default:
			err = fmt.Errorf("unsupported selection type %q", item.kind)
		}
		return actionMsg{text: fmt.Sprintf("Playing %s: %s", item.kind, item.title), err: err}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (m model) accent(text string) string {
	if m.accentColor == "" || text == "" {
		return text
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(m.accentColor)).Render(text)
}

func dominantColorFromImageURL(rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("fetch album art: HTTP %d", resp.StatusCode)
	}
	img, _, err := image.Decode(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return "", err
	}
	return extractAccentColor(img), nil
}

func extractAccentColor(img image.Image) string {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return ""
	}

	stepX := maxInt(1, width/24)
	stepY := maxInt(1, height/24)
	var sumR, sumG, sumB, total float64
	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r, g, b, _ := img.At(x, y).RGBA()
			color := colorful.Color{
				R: float64(r) / 65535.0,
				G: float64(g) / 65535.0,
				B: float64(b) / 65535.0,
			}
			h, c, l := color.Hcl()
			if math.IsNaN(h) {
				continue
			}
			weight := 0.35 + c + (0.5 - math.Abs(l-0.5))
			if weight < 0.1 {
				weight = 0.1
			}
			sumR += color.R * weight
			sumG += color.G * weight
			sumB += color.B * weight
			total += weight
		}
	}
	if total == 0 {
		return ""
	}

	base := colorful.Color{
		R: sumR / total,
		G: sumG / total,
		B: sumB / total,
	}
	h, c, l := base.Hcl()
	if math.IsNaN(h) {
		return ""
	}

	// Keep the album hue, but bias the accent toward a deeper and steadier tone
	// so bright covers do not wash out in light terminals and still read in dark ones.
	targetChroma := clampFloat(c*0.85+0.06, 0.07, 0.16)
	targetLightness := clampFloat(0.56-(l-0.5)*0.35, 0.42, 0.62)

	accent := colorful.Hcl(h, targetChroma, targetLightness)
	if !accent.IsValid() {
		accent = base
	}
	return accent.Clamped().Hex()
}

func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
