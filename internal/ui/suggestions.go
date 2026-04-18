package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
		prefix := strings.ToLower(strings.TrimSpace(arg))
		matches := make([]suggestion, 0)
		for _, device := range m.deviceCache {
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
