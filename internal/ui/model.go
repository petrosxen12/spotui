package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/petrosxen/spotui/internal/app"
	"github.com/petrosxen/spotui/internal/spoterr"
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
	bannerText       string
	bannerIsError    bool
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
	deviceCache      []app.Device
	deviceCacheReady bool
	deviceCacheBusy  bool
	localPlayer      localPlayerStatus
	viewHistory      []viewState
	pollFailures     int
	lastActionUntil  time.Time
}

type viewState struct {
	listItems       []list.Item
	listMode        listMode
	query           string
	lastResults     app.Results
	resultCount     int
	inputValue      string
	lastAction      string
	lastActionErr   bool
	lastActionUntil time.Time
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		checkConnectionCmd(m.service),
		fetchPlaybackCmd(m.service),
		fetchLocalPlayerStatusCmd(m.service),
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
			if m.inputFocused {
				return m, m.refreshSuggestions()
			}
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
			refreshCmd := m.refreshSuggestions()
			return m, tea.Batch(cmd, refreshCmd)
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
			return m, m.refreshSuggestions()
		}

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case connectionMsg:
		if msg.err != nil {
			m.connectionStatus = msg.err.Error()
			m.showBannerForError(msg.err)
			return m, nil
		}
		if msg.user.DisplayName != "" {
			m.connectionStatus = fmt.Sprintf("Connected as %s (%s)", msg.user.DisplayName, msg.user.ID)
		} else {
			m.connectionStatus = fmt.Sprintf("Connected as %s", msg.user.ID)
		}
		m.clearBanner()
		return m, nil
	case searchMsg:
		if msg.err != nil {
			m.setLastAction(msg.err.Error(), true)
			m.showBannerForError(msg.err)
			return m, nil
		}
		m.pushViewState()
		m.query = msg.query
		m.lastResults = msg.results
		m.listMode = listModeSearch
		m.list.SetItems(itemsFromResults(msg.results))
		m.list.Select(0)
		m.resultCount = len(msg.results.Tracks) + len(msg.results.Playlists)
		actionCmd := m.setLastAction(fmt.Sprintf("Loaded %d results for %q", m.resultCount, msg.query), false)
		m.clearBanner()
		return m, actionCmd
	case devicesMsg:
		m.deviceCacheBusy = false
		if msg.err != nil {
			m.setLastAction(msg.err.Error(), true)
			m.showBannerForError(msg.err)
			return m, nil
		}
		m.storeDeviceCache(msg.devices)
		if msg.pushHistory {
			m.pushViewState()
		}
		m.listMode = listModeDevices
		m.list.SetItems(itemsFromDevices(msg.devices))
		m.list.Select(0)
		m.resultCount = len(msg.devices)
		actionCmd := m.setLastAction(fmt.Sprintf("Loaded %d devices", len(msg.devices)), false)
		m.clearBanner()
		return m, actionCmd
	case deviceSelectedMsg:
		if msg.err != nil {
			m.setLastAction(msg.err.Error(), true)
			m.showBannerForError(msg.err)
			return m, nil
		}
		actionCmd := m.setLastAction(fmt.Sprintf("Selected device: %s", msg.device.Name), false)
		m.clearBanner()
		return m, tea.Batch(fetchDevicesCmd(m.service, false), actionCmd)
	case deviceCacheMsg:
		m.deviceCacheBusy = false
		if msg.err != nil {
			return m, nil
		}
		m.storeDeviceCache(msg.devices)
		if !m.inputFocused {
			return m, nil
		}
		return m, m.refreshSuggestions()
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
		actionCmd := m.setLastAction("Command reference", false)
		m.clearBanner()
		return m, actionCmd
	case playbackMsg:
		if msg.err == nil {
			m.playback = msg.state
			m.pollFailures = 0
			m.pollEvery = nextPollInterval(msg.state)
			m.clearBanner()
			if cmd := m.refreshAccentColor(); cmd != nil {
				return m, tea.Batch(pollPlaybackCmd(m.pollEvery), fetchLocalPlayerStatusCmd(m.service), cmd)
			}
		} else {
			m.pollFailures++
			m.setLastAction(msg.err.Error(), true)
			m.showBannerForError(msg.err)
			m.pollEvery = nextPollIntervalForError(msg.err, m.pollFailures)
		}
		return m, tea.Batch(pollPlaybackCmd(m.pollEvery), fetchLocalPlayerStatusCmd(m.service))
	case localPlayerStatusMsg:
		if msg.err != nil {
			return m, nil
		}
		m.localPlayer = msg.status
		return m, nil
	case localPlayerActionMsg:
		if msg.err != nil {
			m.setLastAction(msg.err.Error(), true)
			m.showBannerForError(msg.err)
			return m, fetchLocalPlayerStatusCmd(m.service)
		}
		if msg.status.supported {
			m.localPlayer = msg.status
		}
		actionCmd := m.setLastAction(msg.text, false)
		m.clearBanner()
		return m, tea.Batch(actionCmd, fetchPlaybackCmd(m.service), fetchLocalPlayerStatusCmd(m.service))
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
			m.setLastAction(msg.err.Error(), true)
			m.showBannerForError(msg.err)
			return m, nil
		}
		actionCmd := m.setLastAction(msg.text, false)
		m.clearBanner()
		return m, tea.Batch(fetchPlaybackCmd(m.service), actionCmd)
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

func (m *model) showBannerForError(err error) {
	if err == nil {
		return
	}
	m.bannerText = spoterr.BannerMessage(err)
	m.bannerIsError = true
}

func (m *model) clearBanner() {
	m.bannerText = ""
	m.bannerIsError = false
}

func (m *model) setLastAction(text string, isErr bool) tea.Cmd {
	m.lastAction = text
	m.lastActionErr = isErr
	if isErr || text == "" {
		m.lastActionUntil = time.Time{}
	} else {
		m.lastActionUntil = time.Now().Add(5 * time.Second)
	}
	return nil
}

func (m model) currentLastAction() string {
	if m.lastAction == "" {
		return ""
	}
	if m.lastActionErr || m.lastActionUntil.IsZero() || time.Now().Before(m.lastActionUntil) {
		return m.lastAction
	}
	return ""
}

func (m *model) toggleFocus() {
	m.inputFocused = !m.inputFocused
	if m.inputFocused {
		m.input.Focus()
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
	actionCmd := m.setLastAction(fmt.Sprintf("Running %q...", value), false)
	if strings.HasPrefix(value, "/") {
		m.closeSuggestions()
		return tea.Batch(actionCmd, m.runSlashCommand(value))
	}
	return tea.Batch(actionCmd, searchCmd(m.service, value))
}

func (m *model) storeDeviceCache(devices []app.Device) {
	m.deviceCache = append(m.deviceCache[:0], devices...)
	m.deviceCacheReady = true
}

func (m *model) pushViewState() {
	snapshot := viewState{
		listItems:       append([]list.Item(nil), m.list.Items()...),
		listMode:        m.listMode,
		query:           m.query,
		lastResults:     m.lastResults,
		resultCount:     m.resultCount,
		inputValue:      m.input.Value(),
		lastAction:      m.lastAction,
		lastActionErr:   m.lastActionErr,
		lastActionUntil: m.lastActionUntil,
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
	m.lastActionUntil = snapshot.lastActionUntil
	m.inputFocused = true
	m.input.Focus()
	m.closeSuggestions()
	return true
}
