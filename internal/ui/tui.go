package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/petrosxen/spotui/internal/app"
)

const (
	playbackPollActive   = 1500 * time.Millisecond
	playbackPollIdle     = 4 * time.Second
	playbackPollNoDevice = 6 * time.Second
)

var (
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F5F5F5")).
			Background(lipgloss.Color("#1D6F5F")).
			Padding(0, 1)
	activeInputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F6C453"))
	mutedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D8597"))
	errorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
)

type resultItem struct {
	title       string
	description string
	kind        string
	uri         string
}

func (i resultItem) Title() string       { return i.title }
func (i resultItem) Description() string { return i.description }
func (i resultItem) FilterValue() string { return i.title + " " + i.description + " " + i.kind }

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
	playback         app.PlaybackState
	pollEvery        time.Duration
}

func Run(service app.PlayerService) error {
	m := newModel(service)
	program := tea.NewProgram(m, tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func newModel(service app.PlayerService) model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#F6C453")).
		BorderForeground(lipgloss.Color("#F6C453"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("#DDE7C7"))
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(lipgloss.Color("#EDEDED"))
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(lipgloss.Color("#93A1A1"))

	results := list.New([]list.Item{}, delegate, 0, 0)
	results.Title = "Search Results"
	results.SetShowStatusBar(false)
	results.SetShowHelp(false)
	results.SetFilteringEnabled(false)
	results.DisableQuitKeybindings()

	input := textinput.New()
	input.Placeholder = "Type a search query or /pause /resume /next /prev"
	input.Prompt = "search> "
	input.Focus()
	input.CharLimit = 256

	return model{
		service:          service,
		list:             results,
		input:            input,
		inputFocused:     true,
		connectionStatus: "Connecting to Spotify...",
		lastAction:       "Enter a query to search. Tab switches focus between input and results.",
		pollEvery:        playbackPollIdle,
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
			m.toggleFocus()
			return m, nil
		}

		if m.inputFocused {
			switch msg.String() {
			case "enter":
				return m, m.submitInput()
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "enter":
			selected, ok := m.list.SelectedItem().(resultItem)
			if !ok {
				return m, nil
			}
			m.lastAction = fmt.Sprintf("Playing %s: %s", selected.kind, selected.title)
			return m, playSelectionCmd(m.service, selected)
		case "/":
			m.toggleFocus()
			m.input.SetValue("/")
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
		m.query = msg.query
		m.list.SetItems(itemsFromResults(msg.results))
		m.list.Select(0)
		count := len(msg.results.Tracks) + len(msg.results.Playlists)
		m.lastAction = fmt.Sprintf("Loaded %d results for %q", count, msg.query)
		m.lastActionErr = false
		m.list.Title = fmt.Sprintf("Search Results: %q", msg.query)
		return m, nil
	case playbackMsg:
		if msg.err == nil {
			m.playback = msg.state
			m.pollEvery = nextPollInterval(msg.state)
		} else {
			m.lastAction = msg.err.Error()
			m.lastActionErr = true
			m.pollEvery = playbackPollIdle
		}
		return m, pollPlaybackCmd(m.pollEvery)
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
	top := statusBarStyle.Width(m.width).Render(m.statusLine())
	m.resize()

	focus := "input"
	if !m.inputFocused {
		focus = "results"
	}

	infoStyle := mutedStyle
	if m.lastActionErr {
		infoStyle = errorStyle
	}
	info := infoStyle.Render(m.connectionStatus + "  |  " + m.lastAction + "  |  focus: " + focus)

	inputView := m.input.View()
	if m.inputFocused {
		inputView = activeInputStyle.Render(inputView)
	}

	return strings.Join([]string{
		top,
		m.list.View(),
		info,
		inputView,
	}, "\n")
}

func (m *model) resize() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	listHeight := m.height - 4
	if listHeight < 6 {
		listHeight = 6
	}
	m.list.SetSize(m.width, listHeight)
	m.input.Width = m.width - 2
}

func (m *model) toggleFocus() {
	m.inputFocused = !m.inputFocused
	if m.inputFocused {
		m.input.Focus()
		return
	}
	m.input.Blur()
}

func (m *model) submitInput() tea.Cmd {
	value := strings.TrimSpace(m.input.Value())
	if value == "" {
		return nil
	}
	m.input.SetValue("")
	m.lastAction = fmt.Sprintf("Running %q...", value)
	m.lastActionErr = false
	if strings.HasPrefix(value, "/") {
		return runSlashCommandCmd(m.service, value)
	}
	return searchCmd(m.service, value)
}

func (m model) statusLine() string {
	device := "No device"
	if m.playback.Device.Name != "" {
		device = m.playback.Device.Name
	}

	state := "paused"
	if m.playback.IsPlaying {
		state = "playing"
	}
	if m.playback.Device.ID == "" {
		state = "idle"
	}

	title := "Nothing playing"
	if m.playback.ItemName != "" {
		title = m.playback.ItemName
		if m.playback.ArtistName != "" {
			title += " - " + m.playback.ArtistName
		}
	}

	progress := ""
	if m.playback.Duration > 0 {
		progress = " " + renderProgress(m.playback.Progress, m.playback.Duration)
	}

	return fmt.Sprintf("device %s  |  %s  |  %s%s", device, state, title, progress)
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

func renderProgress(progress time.Duration, total time.Duration) string {
	if total <= 0 {
		return ""
	}
	const width = 10
	filled := int((progress * width) / total)
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("=", filled) + strings.Repeat("-", width-filled) + "]"
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

func fetchPlaybackCmd(service app.PlayerService) tea.Cmd {
	return func() tea.Msg {
		state, err := service.GetPlaybackState(context.Background())
		return playbackMsg{state: state, err: err}
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

func runSlashCommandCmd(service app.PlayerService, raw string) tea.Cmd {
	command := strings.TrimSpace(strings.TrimPrefix(raw, "/"))
	return func() tea.Msg {
		switch command {
		case "pause":
			return actionMsg{text: "Paused playback", err: service.Pause(context.Background())}
		case "resume":
			return actionMsg{text: "Resumed playback", err: service.Resume(context.Background())}
		case "next":
			return actionMsg{text: "Skipped to next item", err: service.Next(context.Background())}
		case "prev":
			return actionMsg{text: "Returned to previous item", err: service.Prev(context.Background())}
		case "quit", "q":
			return actionMsg{text: "Use q or Ctrl+C to quit", err: nil}
		default:
			return actionMsg{text: fmt.Sprintf("Unknown command /%s", command), err: nil}
		}
	}
}
