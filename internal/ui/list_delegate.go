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
		metaText = entry.badge
		if metaText == "" {
			metaText = "HELP"
		}
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
	badge       string
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

func itemsFromTrackDetails(details app.TrackDetails) []list.Item {
	rows := []infoItem{
		{badge: "DETAIL", title: "Track", description: details.Title},
		{badge: "DETAIL", title: "Artist", description: details.Artists},
		{badge: "DETAIL", title: "Album", description: details.Album},
		{badge: "DETAIL", title: "Status", description: trackStatusSummary(details)},
		{badge: "DETAIL", title: "Explicit", description: yesNo(details.Explicit)},
		{badge: "DETAIL", title: "Popularity", description: fmt.Sprintf("%d/100", details.Popularity)},
		{badge: "DETAIL", title: "Device", description: details.DeviceName},
		{badge: "DETAIL", title: "Track URI", description: details.TrackURI},
		{badge: "DETAIL", title: "Context URI", description: details.ContextURI},
	}

	if details.AudioFeaturesAvailable {
		rows = append(rows,
			infoItem{badge: "FEATURE", title: "Danceability", description: formatScore(details.Danceability)},
			infoItem{badge: "FEATURE", title: "Energy", description: formatScore(details.Energy)},
			infoItem{badge: "FEATURE", title: "Valence", description: formatScore(details.Valence)},
			infoItem{badge: "FEATURE", title: "Acousticness", description: formatScore(details.Acousticness)},
			infoItem{badge: "FEATURE", title: "Instrumentalness", description: formatScore(details.Instrumentalness)},
			infoItem{badge: "FEATURE", title: "Liveness", description: formatScore(details.Liveness)},
			infoItem{badge: "FEATURE", title: "Speechiness", description: formatScore(details.Speechiness)},
			infoItem{badge: "FEATURE", title: "Tempo", description: fmt.Sprintf("%.0f BPM", details.Tempo)},
			infoItem{badge: "FEATURE", title: "Key", description: musicalKey(details.Key, details.Mode)},
			infoItem{badge: "FEATURE", title: "Time Signature", description: timeSignatureLabel(details.TimeSignature)},
		)
	} else if details.AudioFeaturesNote != "" {
		rows = append(rows, infoItem{badge: "NOTICE", title: "Audio Features", description: details.AudioFeaturesNote})
	}

	items := make([]list.Item, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.description) == "" {
			continue
		}
		items = append(items, row)
	}
	return items
}

func trackStatusSummary(details app.TrackDetails) string {
	status := "paused"
	if details.IsPlaying {
		status = "playing"
	}

	parts := []string{status}
	if details.Duration > 0 {
		parts = append(parts, formatDuration(details.Progress)+" / "+formatDuration(details.Duration))
	}
	return strings.Join(parts, "  ·  ")
}

func formatScore(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func musicalKey(key int, mode int) string {
	keys := []string{"C", "C#/Db", "D", "D#/Eb", "E", "F", "F#/Gb", "G", "G#/Ab", "A", "A#/Bb", "B"}
	if key < 0 || key >= len(keys) {
		return "Unknown"
	}
	tonality := "major"
	if mode == 0 {
		tonality = "minor"
	}
	return fmt.Sprintf("%s %s", keys[key], tonality)
}

func timeSignatureLabel(signature int) string {
	if signature <= 0 {
		return ""
	}
	return fmt.Sprintf("%d beats/bar", signature)
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
