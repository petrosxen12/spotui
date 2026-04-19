package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/petrosxen/spotui/internal/app"
	"github.com/sahilm/fuzzy"
)

type suggestion struct {
	value       string
	insertValue string
	description string
}

func (m *model) refreshSuggestions() tea.Cmd {
	value := m.input.Value()
	suggestions := m.buildSuggestions(value)
	if len(suggestions) == 0 {
		m.closeSuggestions()
		return m.maybeRefreshDeviceCache(value)
	}
	m.suggestions = suggestions
	if m.suggestionIndex >= len(m.suggestions) {
		m.suggestionIndex = 0
	}
	m.suggestionsOpen = true
	return m.maybeRefreshDeviceCache(value)
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
		if !m.deviceCacheReady {
			return nil
		}
		return deviceSuggestions(m.deviceCache, strings.TrimSpace(arg))
	case "play":
		return m.playSuggestions(strings.TrimSpace(arg))
	default:
		return nil
	}
}

func (m *model) maybeRefreshDeviceCache(raw string) tea.Cmd {
	if m.deviceCacheReady || m.deviceCacheBusy {
		return nil
	}
	if !isDeviceSuggestionInput(raw) {
		return nil
	}
	m.deviceCacheBusy = true
	return refreshDeviceCacheCmd(m.service)
}

func isDeviceSuggestionInput(raw string) bool {
	if !strings.HasPrefix(raw, "/") {
		return false
	}
	command, _, hasSpace := parseSlashInput(raw)
	return command == "device" && hasSpace
}

func (m model) playSuggestions(prefix string) []suggestion {
	playables := m.lastPlayableItems()
	if len(playables) == 0 {
		return nil
	}
	return playableSuggestions(playables, prefix)
}

func deviceSuggestions(devices []app.Device, prefix string) []suggestion {
	if len(devices) == 0 {
		return nil
	}
	if strings.TrimSpace(prefix) == "" {
		suggestions := make([]suggestion, 0, len(devices))
		for _, device := range devices {
			suggestions = append(suggestions, suggestion{
				value:       device.Name,
				insertValue: "/device " + device.Name,
				description: strings.ToLower(device.Type),
			})
		}
		return suggestions
	}

	targets := make([]string, 0, len(devices))
	for _, device := range devices {
		targets = append(targets, device.Name)
	}
	matches := fuzzy.Find(prefix, targets)
	suggestions := make([]suggestion, 0, len(matches))
	for _, match := range matches {
		device := devices[match.Index]
		suggestions = append(suggestions, suggestion{
			value:       device.Name,
			insertValue: "/device " + device.Name,
			description: strings.ToLower(device.Type),
		})
	}
	return suggestions
}

func playableSuggestions(playables []resultItem, prefix string) []suggestion {
	if len(playables) == 0 {
		return nil
	}
	if strings.TrimSpace(prefix) == "" {
		suggestions := make([]suggestion, 0, len(playables))
		for i, item := range playables {
			suggestions = append(suggestions, suggestion{
				value:       fmt.Sprintf("%d. %s", i+1, item.title),
				insertValue: "/play " + item.title,
				description: item.kind,
			})
		}
		return suggestions
	}

	targets := make([]string, 0, len(playables))
	for _, item := range playables {
		targets = append(targets, item.title)
	}
	matches := fuzzy.Find(prefix, targets)
	suggestions := make([]suggestion, 0, len(matches))
	for _, match := range matches {
		item := playables[match.Index]
		suggestions = append(suggestions, suggestion{
			value:       fmt.Sprintf("%d. %s", match.Index+1, item.title),
			insertValue: "/play " + item.title,
			description: item.kind,
		})
	}
	return suggestions
}
