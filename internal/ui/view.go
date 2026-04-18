package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	spotifyGreen = lipgloss.Color("#1DB954")

	pageStyle = lipgloss.NewStyle().
			Padding(0, 1)

	heroStyle = lipgloss.NewStyle().
			MarginBottom(1)

	playbarStyle = lipgloss.NewStyle().
			MarginBottom(1)

	panelStyle = lipgloss.NewStyle().
			Padding(0, 0)

	panelFocusedStyle = panelStyle.Copy().
				Bold(true)

	dockStyle = lipgloss.NewStyle().
			MarginTop(1)

	dockFocusedStyle = dockStyle.Copy().
				BorderTop(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.AdaptiveColor{Light: "#5E8F72", Dark: "#2A7E4F"})

	eyebrowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#66706B", Dark: "#8A948F"}).
			Bold(true)

	kickerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#6E7772", Dark: "#737C78"})

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#111513", Dark: "#F3F6F4"}).
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#505854", Dark: "#C7CECA"})

	metaPillStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#7A847F", Dark: "#8B938F"}).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#59615D", Dark: "#BAC1BD"})

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#355B45", Dark: "#8FD0A7"})

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#8A3B2E", Dark: "#FF8A7A"})

	commandHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#707874", Dark: "#838A87"})

	inputShellStyle = lipgloss.NewStyle().
			Padding(0, 0)

	suggestionPopupStyle = lipgloss.NewStyle().
				Padding(0, 0).
				MarginBottom(1).
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.AdaptiveColor{Light: "#5E8F72", Dark: "#2A7E4F"})

	suggestionSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#111513", Dark: "#F3F6F4"}).
				Bold(true)

	selectedRowStyle = lipgloss.NewStyle().
			Padding(0, 0)

	selectedTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#111513", Dark: "#F4F7F5"}).
				Bold(true)

	selectedDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#59615D", Dark: "#C7CECA"})

	rowTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#18201D", Dark: "#E7ECE9"})

	rowDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#6A726E", Dark: "#7F8884"})

	contextRailStyle = lipgloss.NewStyle().
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.AdaptiveColor{Light: "#58605C", Dark: "#626A66"}).
				PaddingLeft(1)
)

func (m model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Loading spotui..."
	}

	layout := m.layoutMetrics()
	m.resizeWithLayout(layout)

	page := pageStyle.Copy().Padding(layout.pagePaddingY, layout.pagePaddingX)
	playbar := m.playbarContainerStyle(layout).Width(layout.bodyWidth).Render(m.playbarView(layout))

	dock := m.dockContainerStyle(layout).Width(layout.bodyWidth).Render(m.commandDockView(layout))

	mainContent := strings.Join([]string{
		m.resultsPanel(layout.mainWidth, layout),
		m.footerPanel(layout.mainWidth, layout),
	}, "\n")
	if layout.railEnabled {
		mainContent = lipgloss.JoinHorizontal(
			lipgloss.Top,
			mainContent,
			"   ",
			contextRailStyle.Width(layout.railWidth).Render(m.contextRailView(layout)),
		)
	}

	return page.Width(m.width).Render(strings.Join([]string{
		playbar,
		mainContent,
		dock,
	}, "\n"))
}

func (m model) playbarContainerStyle(layout layoutMetrics) lipgloss.Style {
	if layout.heightCompact {
		return playbarStyle.Copy().MarginBottom(0)
	}
	return playbarStyle
}

func (m model) dockContainerStyle(layout layoutMetrics) lipgloss.Style {
	style := dockStyle
	if m.inputFocused {
		style = dockFocusedStyle
	}
	if layout.heightCompact {
		return style.Copy().MarginTop(0)
	}
	return style
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

	leftPrefix := lipgloss.JoinHorizontal(
		lipgloss.Left,
		eyebrowStyle.Render("spotui"),
		"  ",
		m.playingStatusStyle().Render("● "+strings.ToLower(status)),
	)
	progressGroup := lipgloss.JoinHorizontal(
		lipgloss.Left,
		kickerStyle.Render(progress),
		"  ",
		infoStyle.Render(timing),
	)
	if layout.playbarCompact {
		secondary := artist + "  ·  " + device
		if m.playback.NextItemName != "" {
			nextFragment := "  ·  next " + m.playback.NextItemName
			if available := layout.bodyWidth - lipgloss.Width(secondary); available > 8 {
				secondary += truncateText(nextFragment, available)
			}
		}
		left := lipgloss.JoinHorizontal(
			lipgloss.Left,
			leftPrefix,
			"  ",
			m.nowPlayingTitleStyle().Render(title),
		)
		return strings.Join([]string{
			left,
			subtitleStyle.Render(secondary),
			progressGroup,
		}, "\n")
	}

	minTitleWidth := 12
	gapWidth := 2
	leftMeta := lipgloss.JoinHorizontal(
		lipgloss.Left,
		subtitleStyle.Render(artist),
		"  ·  ",
		metaPillStyle.Render(device),
	)
	availableTitleWidth := layout.bodyWidth - lipgloss.Width(leftPrefix) - lipgloss.Width(leftMeta) - lipgloss.Width(progressGroup) - (gapWidth * 3)
	if availableTitleWidth >= minTitleWidth {
		left := lipgloss.JoinHorizontal(
			lipgloss.Left,
			leftPrefix,
			"  ",
			m.nowPlayingTitleStyle().Render(truncateText(title, availableTitleWidth)),
			"  ·  ",
			leftMeta,
		)
		gap := layout.bodyWidth - lipgloss.Width(left) - lipgloss.Width(progressGroup)
		if gap >= gapWidth {
			return lipgloss.JoinHorizontal(lipgloss.Left, left, strings.Repeat(" ", gap), progressGroup)
		}
	}

	left := lipgloss.JoinHorizontal(
		lipgloss.Left,
		leftPrefix,
		"  ",
		m.nowPlayingTitleStyle().Render(title),
	)
	right := lipgloss.JoinHorizontal(
		lipgloss.Left,
		subtitleStyle.Render(artist),
		"  ·  ",
		metaPillStyle.Render(device),
		"  ",
		progressGroup,
	)
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
	progressText := strings.TrimSpace(m.listProgressText())
	if progressText != "" {
		lines = append(lines, kickerStyle.Render(countLabel+"  ·  "+progressText))
	} else {
		lines = append(lines, kickerStyle.Render(countLabel))
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
	if !layout.widthCompact {
		lines = append(lines, commandHintStyle.Render("tab focus  ·  enter play  ·  / commands  ·  q quit"))
	}
	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func (m model) contextRailView(layout layoutMetrics) string {
	lines := []string{eyebrowStyle.Render("Context")}

	switch m.listMode {
	case listModeDevices:
		lines = append(lines, subtitleStyle.Render("Browsing devices"))
	case listModeHelp:
		lines = append(lines, subtitleStyle.Render("Command reference"))
	default:
		if m.query != "" {
			lines = append(lines, subtitleStyle.Render("Search"))
			lines = append(lines, titleStyle.Render(truncateText(m.query, layout.railWidth)))
		} else {
			lines = append(lines, subtitleStyle.Render("Ready for search"))
		}
	}

	if selected := m.selectedContextLines(layout.railWidth); len(selected) > 0 {
		lines = append(lines, "")
		lines = append(lines, subtitleStyle.Render("Selection"))
		lines = append(lines, selected...)
	}

	if total := len(m.list.Items()); total > 0 {
		index := clampInt(m.list.Index()+1, 1, total)
		remaining := maxInt(0, total-index)
		lines = append(lines, "")
		lines = append(lines, subtitleStyle.Render("List"))
		lines = append(lines, infoStyle.Render(fmt.Sprintf("%d of %d", index, total)))
		lines = append(lines, infoStyle.Render(fmt.Sprintf("%d left", remaining)))
	}

	if m.playback.Device.Name != "" {
		lines = append(lines, "")
		lines = append(lines, subtitleStyle.Render("Output"))
		lines = append(lines, infoStyle.Render(truncateText(m.playback.Device.Name, layout.railWidth)))
	}

	if m.playback.NextItemName != "" {
		lines = append(lines, "")
		lines = append(lines, subtitleStyle.Render("Next"))
		lines = append(lines, titleStyle.Render(truncateText(m.playback.NextItemName, layout.railWidth)))
		if m.playback.NextArtistName != "" {
			lines = append(lines, infoStyle.Render(truncateText(m.playback.NextArtistName, layout.railWidth)))
		}
	}

	if depth := len(m.viewHistory); depth > 0 {
		lines = append(lines, "")
		lines = append(lines, subtitleStyle.Render("Back"))
		lines = append(lines, infoStyle.Render(fmt.Sprintf("esc × %d", depth)))
	}

	return lipgloss.NewStyle().Width(layout.railWidth).Render(strings.Join(lines, "\n"))
}

func (m model) commandDockView(layout layoutMetrics) string {
	inputView := inputShellStyle.Width(maxInt(10, layout.bodyWidth-4)).Render(m.input.View())
	lines := []string{}
	if popup := m.suggestionsView(layout); popup != "" {
		lines = append(lines, popup)
	}
	lines = append(lines, inputView)
	return strings.Join(lines, "\n")
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
			line += "  " + commandHintStyle.Render(suggestion.description)
		}
		if i == m.suggestionIndex {
			lines = append(lines, suggestionSelectedStyle.Render("› ")+line)
		} else {
			lines = append(lines, commandHintStyle.Render("  "+line))
		}
	}
	return suggestionPopupStyle.Width(maxInt(20, layout.bodyWidth-4)).Render(strings.Join(lines, "\n"))
}

func (m model) selectedContextLines(width int) []string {
	switch item := m.list.SelectedItem().(type) {
	case resultItem:
		lines := []string{titleStyle.Render(truncateText(item.title, width))}
		if item.description != "" {
			lines = append(lines, infoStyle.Render(truncateText(item.description, width)))
		}
		lines = append(lines, infoStyle.Render(strings.ToUpper(item.kind)))
		return lines
	case deviceItem:
		lines := []string{titleStyle.Render(truncateText(item.title, width))}
		if item.description != "" {
			lines = append(lines, infoStyle.Render(truncateText(item.description, width)))
		}
		return lines
	case infoItem:
		lines := []string{titleStyle.Render(truncateText(item.title, width))}
		if item.description != "" {
			lines = append(lines, infoStyle.Render(truncateText(item.description, width)))
		}
		return lines
	default:
		return nil
	}
}

func (m model) playingStatusStyle() lipgloss.Style {
	if m.playback.IsPlaying {
		if m.accentColor != "" {
			return metaPillStyle.Copy().Foreground(lipgloss.Color(m.accentColor)).Bold(true)
		}
		return metaPillStyle.Copy().Foreground(spotifyGreen).Bold(true)
	}
	return metaPillStyle
}

func (m model) nowPlayingTitleStyle() lipgloss.Style {
	if m.playback.IsPlaying && m.accentColor != "" {
		return titleStyle.Copy().Foreground(lipgloss.Color(m.accentColor))
	}
	return titleStyle
}
