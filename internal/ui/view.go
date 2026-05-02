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

	playbarStyle = lipgloss.NewStyle().
			MarginBottom(1)

	panelStyle = lipgloss.NewStyle().
			Padding(0, 0)

	panelFocusedStyle = panelStyle.Copy().
				Bold(true)

	dockStyle = lipgloss.NewStyle().
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#C4CCC7", Dark: "#4D5551"})

	eyebrowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#66706B", Dark: "#8A948F"}).
			Bold(true)

	statusRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#7A837F", Dark: "#8E9793"})

	kickerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#6E7772", Dark: "#737C78"})

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#111513", Dark: "#F3F6F4"}).
			Bold(true)

	nowPlayingHeroStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#111513", Dark: "#F6F8F7"}).
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

	bannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#C24934", Dark: "#FF9A87"})

	commandHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#707874", Dark: "#838A87"})

	inputShellStyle = lipgloss.NewStyle().
			Padding(0, 0)

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

	mainContent := m.resultsPanel(layout.mainWidth, layout)
	if layout.railEnabled {
		rail := contextRailStyle.Width(layout.railWidth).MaxHeight(lipgloss.Height(mainContent)).Render(m.contextRailView(layout))
		mainContent = lipgloss.JoinHorizontal(
			lipgloss.Top,
			mainContent,
			"   ",
			rail,
		)
	}

	mainSections := []string{
		playbar,
		mainContent,
		dock,
	}
	if layout.footerVisible {
		mainSections = append(mainSections, m.footerPanel(layout.mainWidth, layout))
	}
	return page.Width(m.width).Render(strings.Join(mainSections, "\n"))
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
		style = style.Copy().BorderForeground(lipgloss.Color(m.vividAccentColor()))
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

	artist := ""
	if m.playback.ArtistName != "" {
		artist = m.playback.ArtistName
	}

	device := ""
	if m.playback.Device.Name != "" {
		device = m.playback.Device.Name
	}

	progress := m.progressBar(layout.playbarProgressLen)
	timing := formatDuration(m.playback.Progress) + " / " + formatDuration(m.playback.Duration)
	progressGroup := m.playbarProgressView(layout, progress, timing)
	statusRow := m.playbarStatusRow(layout.bodyWidth, strings.ToLower(status), device)
	if layout.playbarMinimal {
		primary := m.playbarPrimaryLine(layout.bodyWidth, title)
		return strings.Join([]string{
			statusRow,
			primary,
			progressGroup,
		}, "\n")
	}
	if layout.playbarCompact {
		if layout.bodyWidth < 36 {
			return strings.Join([]string{
				statusRow,
				m.playbarPrimaryLine(layout.bodyWidth, title),
				progressGroup,
			}, "\n")
		}
		secondaryParts := make([]string, 0, 3)
		if artist != "" {
			secondaryParts = append(secondaryParts, artist)
		}
		if device != "" {
			secondaryParts = append(secondaryParts, device)
		}
		if m.playback.NextItemName != "" && layout.bodyWidth >= 52 {
			secondaryParts = append(secondaryParts, "next "+m.playback.NextItemName)
		}
		lines := []string{
			statusRow,
			m.playbarPrimaryLine(layout.bodyWidth, title),
			progressGroup,
		}
		if len(secondaryParts) > 0 {
			lines = append(lines[0:2], append([]string{subtitleStyle.Render(joinAndTruncate(layout.bodyWidth, "  ·  ", secondaryParts...))}, lines[2:]...)...)
		}
		return strings.Join(lines, "\n")
	}

	titleWidth := maxInt(1, layout.bodyWidth-lipgloss.Width(progressGroup)-2)
	titleText := m.nowPlayingTitleStyle().Render(truncateText(title, titleWidth))
	if titleWidth >= 18 {
		titleLine := lipgloss.JoinHorizontal(
			lipgloss.Left,
			titleText,
			strings.Repeat(" ", maxInt(2, layout.bodyWidth-lipgloss.Width(titleText)-lipgloss.Width(progressGroup))),
			progressGroup,
		)
		metaParts := make([]string, 0, 3)
		if artist != "" {
			metaParts = append(metaParts, artist)
		}
		if device != "" {
			metaParts = append(metaParts, device)
		}
		if m.playback.NextItemName != "" {
			metaParts = append(metaParts, "next "+m.playback.NextItemName)
		}
		lines := []string{
			statusRow,
			titleLine,
		}
		if len(metaParts) > 0 {
			lines = append(lines, subtitleStyle.Render(joinAndTruncate(layout.bodyWidth, "  ·  ", metaParts...)))
		}
		return strings.Join(lines, "\n")
	}

	lines := []string{
		statusRow,
		m.playbarPrimaryLine(layout.bodyWidth, title),
		progressGroup,
	}
	if artist != "" {
		lines = append(lines[0:2], append([]string{subtitleStyle.Render(truncateText(artist, layout.bodyWidth))}, lines[2:]...)...)
	}
	return strings.Join(lines, "\n")
}

func (m model) playbarStatusRow(width int, status string, device string) string {
	parts := []string{
		eyebrowStyle.Render("spotui"),
		m.playingStatusStyle().Render("● " + status),
	}
	if connection := m.connectionBadge(width); connection != "" {
		parts = append(parts, connection)
	}
	if device != "" {
		parts = append(parts, truncateText(device, maxInt(10, width/4)))
	}
	return statusRowStyle.Render(joinAndTruncate(width, "  ·  ", parts...))
}

func (m model) playbarPrimaryLine(width int, title string) string {
	return nowPlayingHeroStyle.Copy().Width(width).Render(truncateText(title, width))
}

func (m model) playbarProgressView(layout layoutMetrics, progress string, timing string) string {
	if layout.bodyWidth <= 0 {
		return ""
	}
	timingText := truncateText(timing, layout.bodyWidth)
	if layout.bodyWidth < 20 || progress == "" {
		return infoStyle.Render(timingText)
	}

	barWidth := layout.bodyWidth - lipgloss.Width(timingText) - 2
	if barWidth < 4 {
		return infoStyle.Render(timingText)
	}
	barWidth = minInt(barWidth, layout.playbarProgressLen)

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		kickerStyle.Render(m.progressBar(barWidth)),
		"  ",
		infoStyle.Render(timingText),
	)
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
		} else if m.resultCount == 0 && !m.bootAnimationDone {
			header = m.bootLoadingText()
		} else if m.resultCount == 0 && m.bootAnimationDone {
			header = ""
		}
	}

	countLabel := ""
	switch m.listMode {
	case listModeDevices:
		countLabel = fmt.Sprintf("%d devices", m.resultCount)
	case listModeHelp:
		countLabel = fmt.Sprintf("%d commands", m.resultCount)
	case listModeSearch:
		if m.resultCount > 0 {
			countLabel = fmt.Sprintf("%d items", m.resultCount)
		} else if m.query != "" {
			countLabel = "No results yet"
		}
	}
	focusLabel := m.resultsFocusLabel()

	body := m.list.View()
	if m.resultCount == 0 {
		switch m.listMode {
		case listModeDevices:
			body = subtitleStyle.Render("Run /devices to refresh the available playback devices.")
		case listModeHelp:
			body = subtitleStyle.Render("Type /help to load the command list.")
		default:
			body = m.emptyResultsView(width)
		}
	}

	header = truncateText(header, width)
	countLabel = truncateText(countLabel, width)

	lines := make([]string, 0, 4)
	if header != "" {
		lines = append(lines, eyebrowStyle.Render(header))
	}
	progressText := strings.TrimSpace(m.listProgressText())
	metaParts := make([]string, 0, 3)
	if countLabel != "" {
		metaParts = append(metaParts, countLabel)
	}
	if progressText != "" {
		metaParts = append(metaParts, progressText)
	}
	if focusLabel != "" && (len(metaParts) > 0 || body != "") {
		metaParts = append(metaParts, focusLabel)
	}
	if len(metaParts) > 0 {
		lines = append(lines, kickerStyle.Render(joinAndTruncate(width, "  ·  ", metaParts...)))
	}
	if body == "" {
		return style.Width(width).Render(strings.Join(lines, "\n"))
	}
	if layout.heightMode == heightModeMinimal {
		lines = append(lines, body)
	} else {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, body)
	}
	return style.Width(width).Render(strings.Join(lines, "\n"))
}

func (m model) resultsFocusLabel() string {
	if m.inputFocused {
		return "command active"
	}
	return "list active"
}

func (m model) footerPanel(width int, layout layoutMetrics) string {
	if !layout.footerVisible {
		return ""
	}

	currentAction := m.currentLastAction()
	statusTone := infoStyle
	if m.lastActionErr {
		statusTone = errorStyle
	} else if currentAction != "" {
		statusTone = successStyle.Copy().Foreground(lipgloss.Color(m.vividAccentColor()))
	}

	statusParts := make([]string, 0, 1)
	if line := m.localPlayer.statusLine(); line != "" && (layout.footerShowStatus || m.playback.Device.ID == "") {
		statusParts = append(statusParts, line)
	}
	lines := make([]string, 0, 3)
	if len(statusParts) > 0 {
		lines = append(lines, commandHintStyle.Render(joinAndTruncate(width, "  ·  ", statusParts...)))
	}
	if m.bannerText != "" {
		tone := commandHintStyle
		if m.bannerIsError {
			tone = bannerStyle
		}
		lines = append(lines, tone.Render(truncateText(m.bannerText, width)))
	}
	if layout.footerShowStatus && currentAction != "" {
		lines = append(lines, statusTone.Render(truncateText(currentAction, width)))
	}
	if layout.footerShowHints && m.bannerText == "" {
		lines = append(lines, commandHintStyle.Render(truncateText("tab focus  ·  enter play  ·  / commands  ·  q quit", width)))
	}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
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
	} else if line := m.localPlayer.statusLine(); line != "" {
		lines = append(lines, "")
		lines = append(lines, subtitleStyle.Render("Local player"))
		lines = append(lines, infoStyle.Render(truncateText(line, layout.railWidth)))
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

func (m model) emptyResultsHint() string {
	if m.playback.Device.ID == "" && m.localPlayer.supported && (m.localPlayer.process == "running" || m.localPlayer.process == "starting") {
		if m.localPlayer.device != "" {
			return "Waiting for " + m.localPlayer.device + " to become the active output."
		}
		return "Waiting for the local player to become the active output."
	}
	if m.playback.Device.ID == "" && m.localPlayer.supported && m.localPlayer.binaryAvailable {
		return "Run /local start to use the local player."
	}
	if m.playback.Device.ID == "" && m.localPlayer.supported && !m.localPlayer.binaryAvailable && m.localPlayer.message != "" {
		return "Local player unavailable: " + m.localPlayer.message
	}
	return ""
}

func (m model) emptyResultsView(width int) string {
	secondary := m.emptyResultsHint()
	if secondary == "" {
		return ""
	}
	return subtitleStyle.Render(truncateText(secondary, width))
}

func (m model) connectionBadge(width int) string {
	connected, label := parseConnectionStatus(m.connectionStatus)
	if label == "" {
		return ""
	}
	if connected {
		return successStyle.Copy().
			Foreground(spotifyGreen).
			Render("● " + truncateText(label, maxInt(12, width/5)))
	}
	return errorStyle.Render("● " + truncateText(label, maxInt(12, width/6)))
}

func parseConnectionStatus(raw string) (bool, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, ""
	}
	const prefix = "Connected as "
	if strings.HasPrefix(raw, prefix) {
		label := strings.TrimPrefix(raw, prefix)
		if idx := strings.Index(label, " ("); idx > 0 {
			label = label[:idx]
		}
		return true, strings.TrimSpace(label)
	}
	return false, "offline"
}

func (m model) bootLoadingText() string {
	frames := []string{"·  ·  ·", "•  ·  ·", "•  •  ·", "•  •  •"}
	return frames[m.bootFrames%len(frames)]
}

func (m model) commandDockView(layout layoutMetrics) string {
	inputLine := m.input.View()
	inputView := inputShellStyle.Width(maxInt(10, layout.bodyWidth-4)).Render(inputLine)
	lines := []string{eyebrowStyle.Render(m.commandDockTitle())}
	if popup := m.suggestionsView(layout); popup != "" {
		lines = append(lines, popup)
	}
	lines = append(lines, inputView)
	return strings.Join(lines, "\n")
}

func (m model) commandDockTitle() string {
	if m.inputFocused {
		return "Command dock"
	}
	return "Press tab for command dock"
}

func (m model) suggestionsView(layout layoutMetrics) string {
	if !m.suggestionsOpen || len(m.suggestions) == 0 {
		return ""
	}
	visible := m.suggestions
	if len(visible) > 5 {
		visible = visible[:5]
	}
	contentWidth := maxInt(1, layout.bodyWidth-6)
	lines := make([]string, 0, len(visible))
	for i, suggestion := range visible {
		line := suggestion.value
		if suggestion.description != "" {
			line = joinAndTruncate(contentWidth, "  ", suggestion.value, suggestion.description)
		} else {
			line = truncateText(line, contentWidth)
		}
		if i == m.suggestionIndex {
			lines = append(lines, suggestionSelectedStyle.Render("› "+line))
		} else {
			lines = append(lines, commandHintStyle.Render("  "+line))
		}
	}
	return lipgloss.NewStyle().
		Padding(0, 1).
		BorderTop(true).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(m.vividAccentColor())).
		Width(maxInt(20, layout.bodyWidth-4)).
		Render(strings.Join(lines, "\n"))
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
		return nowPlayingHeroStyle.Copy().Foreground(lipgloss.Color(m.accentColor))
	}
	return nowPlayingHeroStyle
}
