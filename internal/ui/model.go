package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/petrosxen/spotui/internal/app"
)

type listMode string

const (
	listModeSearch  listMode = "search"
	listModeDevices listMode = "devices"
	listModeHelp    listMode = "help"
)

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
