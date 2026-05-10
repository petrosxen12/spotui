package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/petrosxen/spotui/internal/app"
	"github.com/petrosxen/spotui/internal/spoterr"
)

type layoutMetrics struct {
	bodyWidth           int
	mainWidth           int
	compact             bool
	widthCompact        bool
	heightCompact       bool
	heightMode          layoutHeightMode
	playbarCompact      bool
	playbarMinimal      bool
	playbarProgressLen  int
	listHeight          int
	resultsChromeHeight int
	inputWidth          int
	pagePaddingX        int
	pagePaddingY        int
	railEnabled         bool
	railWidth           int
	footerVisible       bool
	footerShowStatus    bool
	footerShowHints     bool
}

type layoutHeightMode string

const (
	heightModeNormal  layoutHeightMode = "normal"
	heightModeCompact layoutHeightMode = "compact"
	heightModeMinimal layoutHeightMode = "minimal"
)

func (m *model) resize() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	m.resizeWithLayout(m.layoutMetrics())
}

func (m *model) resizeWithLayout(layout layoutMetrics) {
	m.list.SetDelegate(resultDelegate{
		width:       layout.mainWidth,
		wideLayout:  !layout.widthCompact,
		focused:     !m.inputFocused,
		accentColor: m.vividAccentColor(),
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
	heightMode := classifyHeightMode(m.height)
	if heightMode != heightModeNormal {
		paddingY = 0
	}

	bodyWidth := m.width - (paddingX * 2)
	if bodyWidth < 20 {
		bodyWidth = 20
	}

	layoutForRail := m.layoutMetricsForWidth(bodyWidth, paddingX, paddingY, heightMode, true)
	return layoutForRail
}

func (m model) layoutMetricsForWidth(bodyWidth int, paddingX int, paddingY int, heightMode layoutHeightMode, allowRail bool) layoutMetrics {

	mainWidth := bodyWidth
	railEnabled := false
	railWidth := 0
	if allowRail && m.shouldShowContextRail() && bodyWidth >= 118 && heightMode != heightModeMinimal && m.height >= 20 {
		candidateRailWidth := clampInt(bodyWidth/5, 22, 28)
		candidateMainWidth := bodyWidth - candidateRailWidth - 3
		if candidateMainWidth >= 72 {
			mainWidth = candidateMainWidth
			railEnabled = true
			railWidth = candidateRailWidth
		}
	}

	widthCompact := mainWidth < 72
	heightCompact := heightMode != heightModeNormal
	compact := widthCompact || heightCompact
	playbarCompact := widthCompact || heightCompact
	playbarMinimal := heightMode == heightModeMinimal

	playbarProgressLen := clampInt(mainWidth/4, 12, 32)
	if playbarMinimal {
		playbarProgressLen = clampInt(mainWidth/5, 8, 18)
	}
	inputWidth := maxInt(10, bodyWidth-2)
	resultsChromeHeight := 3
	if heightMode == heightModeMinimal {
		resultsChromeHeight = 2
	}
	footerVisible := true
	footerShowStatus := true
	footerShowHints := !widthCompact && heightMode == heightModeNormal
	if heightMode == heightModeMinimal {
		footerShowStatus = false
	}

	tempLayout := layoutMetrics{
		bodyWidth:           bodyWidth,
		mainWidth:           mainWidth,
		compact:             compact,
		widthCompact:        widthCompact,
		heightCompact:       heightCompact,
		heightMode:          heightMode,
		playbarCompact:      playbarCompact,
		playbarMinimal:      playbarMinimal,
		playbarProgressLen:  playbarProgressLen,
		inputWidth:          inputWidth,
		pagePaddingX:        paddingX,
		pagePaddingY:        paddingY,
		resultsChromeHeight: resultsChromeHeight,
		railEnabled:         railEnabled,
		railWidth:           railWidth,
		footerVisible:       footerVisible,
		footerShowStatus:    footerShowStatus,
		footerShowHints:     footerShowHints,
	}

	pageVertical := paddingY * 2
	playbarHeight := lipgloss.Height(m.playbarContainerStyle(tempLayout).Width(bodyWidth).Render(m.playbarView(tempLayout)))
	commandBarHeight := lipgloss.Height(m.commandBarContainerStyle().Width(bodyWidth).Render(m.commandBarView(tempLayout)))
	footerHeight := 0
	if footerVisible {
		footerHeight = lipgloss.Height(m.footerPanel(mainWidth, tempLayout))
	}
	sectionGaps := 1
	if footerVisible {
		sectionGaps++
	}
	available := m.height - pageVertical - playbarHeight - footerHeight - commandBarHeight - sectionGaps
	minListHeight := 1
	if m.height >= 24 {
		minListHeight = 6
	} else if m.height >= 20 {
		minListHeight = 4
	} else if m.height >= 16 {
		minListHeight = 2
	}
	listHeight := clampInt(available, minListHeight, maxInt(minListHeight, available))

	return layoutMetrics{
		bodyWidth:           bodyWidth,
		mainWidth:           mainWidth,
		compact:             compact,
		widthCompact:        widthCompact,
		heightCompact:       heightCompact,
		heightMode:          heightMode,
		playbarCompact:      playbarCompact,
		playbarMinimal:      playbarMinimal,
		playbarProgressLen:  playbarProgressLen,
		listHeight:          listHeight,
		resultsChromeHeight: resultsChromeHeight,
		inputWidth:          inputWidth,
		pagePaddingX:        paddingX,
		pagePaddingY:        paddingY,
		railEnabled:         railEnabled,
		railWidth:           railWidth,
		footerVisible:       footerVisible,
		footerShowStatus:    footerShowStatus,
		footerShowHints:     footerShowHints,
	}
}

func (m model) shouldShowContextRail() bool {
	return len(m.selectedContextLines(28)) > 0
}

func classifyHeightMode(height int) layoutHeightMode {
	switch {
	case height < 10:
		return heightModeMinimal
	case height < 24:
		return heightModeCompact
	default:
		return heightModeNormal
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
	if width <= 0 {
		return ""
	}
	if m.playback.Duration <= 0 {
		return strings.Repeat("─", width)
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

func nextPollIntervalForError(err error, failures int) time.Duration {
	base := playbackPollIdle
	switch spoterr.KindOf(err) {
	case spoterr.KindNoActiveDevice:
		base = playbackPollNoDevice
	case spoterr.KindRateLimited:
		base = 5 * time.Second
	case spoterr.KindNetworkFailure:
		base = 3 * time.Second
	case spoterr.KindAuthExpired:
		base = 10 * time.Second
	}

	if retryAfter := spoterr.RetryAfter(err); retryAfter > base {
		base = retryAfter
	}

	if failures < 1 {
		failures = 1
	}
	backoff := base * time.Duration(1<<minInt(failures-1, 3))
	return minDuration(backoff, 30*time.Second)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
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

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
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

func joinAndTruncate(width int, separator string, parts ...string) string {
	if width <= 0 {
		return ""
	}

	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) == 0 {
		return ""
	}

	line := filtered[0]
	if lipgloss.Width(line) >= width {
		return truncateText(line, width)
	}
	for _, part := range filtered[1:] {
		candidate := line + separator + part
		if lipgloss.Width(candidate) <= width {
			line = candidate
			continue
		}
		remaining := width - lipgloss.Width(line) - lipgloss.Width(separator)
		if remaining <= 0 {
			return truncateText(line, width)
		}
		return line + separator + truncateText(part, remaining)
	}
	return line
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
