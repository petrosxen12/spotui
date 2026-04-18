package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/petrosxen/spotui/internal/app"
)

type slashCommand struct {
	name        string
	usage       string
	description string
}

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

func refreshDeviceCacheCmd(service app.PlayerService) tea.Cmd {
	return func() tea.Msg {
		devices, err := service.ListDevices(context.Background())
		return deviceCacheMsg{devices: devices, err: err}
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
