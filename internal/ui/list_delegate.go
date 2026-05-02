package ui

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/petrosxen/spotui/internal/app"
)

type resultDelegate struct {
	width       int
	wideLayout  bool
	focused     bool
	accentColor string
}

func (d resultDelegate) Height() int  { return 2 }
func (d resultDelegate) Spacing() int { return 1 }

func (d resultDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d resultDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var titleText string
	var descText string
	var metaText string
	selected := index == m.Index()

	switch entry := item.(type) {
	case resultItem:
		titleText = entry.title
		descText = entry.description
		metaText = strings.ToUpper(entry.kind)
	case deviceItem:
		titleText = entry.title
		descText = entry.description
		metaText = "DEVICE"
	case infoItem:
		titleText = entry.title
		descText = entry.description
		metaText = "HELP"
	default:
		return
	}

	line1 := d.renderPrimaryLine(titleText, metaText, selected)
	descWidth := maxInt(1, d.contentWidth()-2)
	line2 := "  " + d.descriptionStyle(selected).Render(truncateText(descText, descWidth))

	if selected {
		block := strings.Join([]string{line1, line2}, "\n")
		fmt.Fprint(w, selectedRowStyle.Render(block))
		return
	}

	fmt.Fprint(w, strings.Join([]string{line1, line2}, "\n"))
}

func (d resultDelegate) renderPrimaryLine(titleText string, metaText string, selected bool) string {
	prefix := "  "
	prefixStyle := lipgloss.NewStyle()
	titleStyleToUse := rowTitleStyle
	metaStyleToUse := metaPillStyle
	if selected {
		prefix = "· "
		titleStyleToUse = rowTitleStyle.Copy().Bold(true)
		metaStyleToUse = rowDescStyle.Copy().Bold(true)
		if d.focused {
			prefix = "› "
			prefixStyle = prefixStyle.Foreground(lipgloss.Color(d.accentColor)).Bold(true)
			titleStyleToUse = selectedTitleStyle
			metaStyleToUse = selectedDescStyle.Copy().Bold(true)
		}
	}

	if !d.wideLayout || d.contentWidth() < 36 {
		titleWidth := maxInt(1, d.contentWidth()-lipgloss.Width(metaText)-1)
		left := lipgloss.JoinHorizontal(
			lipgloss.Left,
			metaStyleToUse.Render(metaText),
			" ",
			titleStyleToUse.Render(truncateText(titleText, titleWidth)),
		)
		return prefixStyle.Render(prefix) + lipgloss.NewStyle().MaxWidth(maxInt(1, d.contentWidth())).Render(left)
	}

	meta := metaStyleToUse.Render(metaText)
	leftWidth := maxInt(1, d.contentWidth()-lipgloss.Width(meta)-2)
	left := titleStyleToUse.Render(truncateText(titleText, leftWidth))
	gap := d.contentWidth() - lipgloss.Width(left) - lipgloss.Width(meta)
	if gap < 2 {
		gap = 2
	}
	return prefixStyle.Render(prefix) + left + strings.Repeat(" ", gap) + meta
}

func (d resultDelegate) contentWidth() int {
	if d.width <= 0 {
		return 40
	}
	return maxInt(1, d.width-2)
}

func (d resultDelegate) descriptionStyle(selected bool) lipgloss.Style {
	if selected {
		if d.focused {
			return selectedDescStyle
		}
		return rowDescStyle
	}
	return rowDescStyle
}

type resultItem struct {
	title       string
	description string
	kind        string
	uri         string
}

func (i resultItem) Title() string       { return i.title }
func (i resultItem) Description() string { return i.description }
func (i resultItem) FilterValue() string { return i.title + " " + i.description + " " + i.kind }

type deviceItem struct {
	title       string
	description string
	id          string
}

func (i deviceItem) Title() string       { return i.title }
func (i deviceItem) Description() string { return i.description }
func (i deviceItem) FilterValue() string { return i.title + " " + i.description + " " + i.id }

type infoItem struct {
	title       string
	description string
}

func (i infoItem) Title() string       { return i.title }
func (i infoItem) Description() string { return i.description }
func (i infoItem) FilterValue() string { return i.title + " " + i.description }

func itemsFromResults(results app.Results) []list.Item {
	items := make([]list.Item, 0, len(results.Tracks)+len(results.Playlists))
	for _, track := range results.Tracks {
		title, ok := visibleResultTitle(track.Name)
		if !ok {
			continue
		}
		description := "track"
		if track.Subtitle != "" {
			description = track.Subtitle
		}
		items = append(items, resultItem{
			title:       title,
			description: description,
			kind:        "track",
			uri:         track.URI,
		})
	}
	for _, playlist := range results.Playlists {
		title, ok := visibleResultTitle(playlist.Name)
		if !ok {
			continue
		}
		description := "playlist"
		if playlist.Subtitle != "" {
			description = "playlist by " + playlist.Subtitle
		}
		items = append(items, resultItem{
			title:       title,
			description: description,
			kind:        "playlist",
			uri:         playlist.URI,
		})
	}
	return items
}

func visibleResultTitle(name string) (string, bool) {
	trimmed := strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			return -1
		}
		return r
	}, name))
	if trimmed == "" || lipgloss.Width(trimmed) == 0 {
		return "", false
	}
	return trimmed, true
}

func itemsFromDevices(devices []app.Device) []list.Item {
	items := make([]list.Item, 0, len(devices))
	for _, device := range devices {
		state := device.Type
		if device.IsActive {
			state += " • active"
		}
		items = append(items, deviceItem{
			title:       device.Name,
			description: state,
			id:          device.ID,
		})
	}
	return items
}
