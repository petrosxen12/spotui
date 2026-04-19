package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/petrosxen/spotui/internal/app"
)

const (
	playbackPollActive   = 1500 * time.Millisecond
	playbackPollIdle     = 4 * time.Second
	playbackPollNoDevice = 6 * time.Second
)

func Run(service app.PlayerService) error {
	m := newModel(service)
	program := tea.NewProgram(m, tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func newModel(service app.PlayerService) model {
	results := list.New([]list.Item{}, resultDelegate{}, 0, 0)
	results.Title = ""
	results.SetShowTitle(false)
	results.SetShowStatusBar(false)
	results.SetShowPagination(false)
	results.SetShowHelp(false)
	results.SetFilteringEnabled(false)
	results.DisableQuitKeybindings()

	input := textinput.New()
	input.Placeholder = "Search tracks, playlists, or enter /pause /resume /next /prev /local start"
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
