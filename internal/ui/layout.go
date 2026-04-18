package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/petrosxen/spotui/internal/app"
)

type layoutMetrics struct {
	bodyWidth           int
	mainWidth           int
	compact             bool
	widthCompact        bool
	playbarCompact      bool
	playbarProgressLen  int
	listHeight          int
	resultsChromeHeight int
	inputWidth          int
	pagePaddingX        int
	pagePaddingY        int
	railEnabled         bool
	railWidth           int
}

func (m *model) resize() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	m.resizeWithLayout(m.layoutMetrics())
}

func (m *model) resizeWithLayout(layout layoutMetrics) {
	m.list.SetDelegate(resultDelegate{
		width:      layout.mainWidth,
		wideLayout: !layout.widthCompact,
	})
	m.list.SetSize(maxInt(20, layout.mainWidth), maxInt(1, layout.listHeight-layout.resultsChromeHeight))
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

	mainWidth := bodyWidth
	railEnabled := false
	railWidth := 0
	if bodyWidth >= 118 {
		candidateRailWidth := clampInt(bodyWidth/5, 22, 28)
		candidateMainWidth := bodyWidth - candidateRailWidth - 3
		if candidateMainWidth >= 72 {
			mainWidth = candidateMainWidth
			railEnabled = true
			railWidth = candidateRailWidth
		}
	}

	widthCompact := mainWidth < 72
	compact := widthCompact || m.height < 26
	playbarCompact := mainWidth < 72

	playbarProgressLen := clampInt(mainWidth/4, 12, 32)
	inputWidth := maxInt(10, bodyWidth-2)
	resultsChromeHeight := 3

	tempLayout := layoutMetrics{
		bodyWidth:           bodyWidth,
		mainWidth:           mainWidth,
		compact:             compact,
		widthCompact:        widthCompact,
		playbarCompact:      playbarCompact,
		playbarProgressLen:  playbarProgressLen,
		inputWidth:          inputWidth,
		pagePaddingX:        paddingX,
		pagePaddingY:        paddingY,
		resultsChromeHeight: resultsChromeHeight,
		railEnabled:         railEnabled,
		railWidth:           railWidth,
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
		mainWidth:           mainWidth,
		compact:             compact,
		widthCompact:        widthCompact,
		playbarCompact:      playbarCompact,
		playbarProgressLen:  playbarProgressLen,
		listHeight:          listHeight,
		resultsChromeHeight: resultsChromeHeight,
		inputWidth:          inputWidth,
		pagePaddingX:        paddingX,
		pagePaddingY:        paddingY,
		railEnabled:         railEnabled,
		railWidth:           railWidth,
	}
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

func truncateText(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}

	runes := []rune(value)
	truncated := make([]rune, 0, len(runes))
	for _, r := range runes {
		next := string(append(truncated, r))
		if lipgloss.Width(next+"…") > width {
			break
		}
		truncated = append(truncated, r)
	}
	if len(truncated) == 0 {
		return "…"
	}
	return string(truncated) + "…"
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
