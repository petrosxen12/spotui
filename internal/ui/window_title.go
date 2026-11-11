package ui

import (
	"strings"
	"unicode"

	"github.com/petrosxen/spotui/internal/app"
)

func playbackWindowTitle(state app.PlaybackState) string {
	// Metadata must not introduce terminal control sequences into the title.
	clean := func(value string) string {
		return strings.TrimSpace(strings.Map(func(r rune) rune {
			if unicode.IsControl(r) {
				return -1
			}
			return r
		}, value))
	}
	title := clean(state.ItemName)
	if title == "" {
		return "spotui"
	}
	if artist := clean(state.ArtistName); artist != "" {
		title += " — " + artist
	}
	return title + " | spotui"
}
