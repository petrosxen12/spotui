package ui

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/petrosxen/spotui/internal/app"
)

type QAReviewScenario struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Focus       string `json:"focus"`
	Description string `json:"description"`
	TextFile    string `json:"text_file"`
	SVGFile     string `json:"svg_file"`
}

type qaReviewBundle struct {
	GeneratedAt string             `json:"generated_at"`
	Scenarios   []QAReviewScenario `json:"scenarios"`
}

type qaScenarioSpec struct {
	id          string
	title       string
	width       int
	height      int
	focus       string
	description string
	build       func() model
}

func WriteQAReviewBundle(outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	specs := qaScenarioSpecs()
	bundle := qaReviewBundle{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Scenarios:   make([]QAReviewScenario, 0, len(specs)),
	}

	for _, spec := range specs {
		m := spec.build()
		m.width = spec.width
		m.height = spec.height
		m.resize()

		view := stripANSI(m.View())
		textName := spec.id + ".txt"
		svgName := spec.id + ".svg"

		if err := os.WriteFile(filepath.Join(outDir, textName), []byte(view+"\n"), 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(outDir, svgName), []byte(renderTerminalSVG(spec.title, spec.width, spec.height, view)), 0o644); err != nil {
			return err
		}

		bundle.Scenarios = append(bundle.Scenarios, QAReviewScenario{
			ID:          spec.id,
			Title:       spec.title,
			Width:       spec.width,
			Height:      spec.height,
			Focus:       spec.focus,
			Description: spec.description,
			TextFile:    textName,
			SVGFile:     svgName,
		})
	}

	manifest, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "manifest.json"), manifest, 0o644); err != nil {
		return err
	}

	brief := qaReviewBrief(bundle)
	if err := os.WriteFile(filepath.Join(outDir, "review-brief.md"), []byte(brief), 0o644); err != nil {
		return err
	}

	return nil
}

func qaScenarioSpecs() []qaScenarioSpec {
	return []qaScenarioSpec{
		{
			id:          "desktop-search",
			title:       "Desktop Search Flow",
			width:       144,
			height:      34,
			focus:       "Assess information hierarchy, whitespace, and right-rail usefulness on a wide layout.",
			description: "Wide layout with context rail, active playback, search results, and a visible next-up state.",
			build: func() model {
				m := baseQAModel()
				m.query = "fka twigs"
				m.listMode = listModeSearch
				m.lastResults = app.Results{
					Tracks: []app.SearchItem{
						{Name: "Eusexua", Subtitle: "FKA twigs", URI: "spotify:track:1", Kind: "track"},
						{Name: "Cellophane", Subtitle: "FKA twigs", URI: "spotify:track:2", Kind: "track"},
						{Name: "Two Weeks", Subtitle: "FKA twigs", URI: "spotify:track:3", Kind: "track"},
					},
					Playlists: []app.SearchItem{
						{Name: "twigs study session", Subtitle: "Petros Xen", URI: "spotify:playlist:1", Kind: "playlist"},
					},
				}
				m.list.SetItems(itemsFromResults(m.lastResults))
				m.list.Select(0)
				m.resultCount = len(m.lastResults.Tracks) + len(m.lastResults.Playlists)
				m.inputFocused = false
				m.lastAction = `Loaded 4 results for "fka twigs"`
				m.playback = app.PlaybackState{
					Device:         app.Device{ID: "desktop-airplay", Name: "Office Speakers", Type: "Speaker", IsActive: true},
					IsPlaying:      true,
					Progress:       97 * time.Second,
					Duration:       271 * time.Second,
					ItemName:       "Perfect Stranger",
					ArtistName:     "FKA twigs",
					NextItemName:   "Drums of Death",
					NextArtistName: "FKA twigs",
					AlbumArtURL:    "https://example.com/fka-twigs.jpg",
				}
				m.accentColor = "#c97f6b"
				return m
			},
		},
		{
			id:          "laptop-devices",
			title:       "Laptop Device Picker",
			width:       108,
			height:      26,
			focus:       "Check scanability of the device list and whether the command/status surfaces feel calm rather than busy.",
			description: "Mid-width device view with an active device, local-player metadata, and the command dock in focus.",
			build: func() model {
				m := baseQAModel()
				m.listMode = listModeDevices
				m.list.SetItems(itemsFromDevices([]app.Device{
					{ID: "mac", Name: "MacBook Pro", Type: "Computer", IsActive: true},
					{ID: "homepod", Name: "Kitchen HomePod", Type: "Speaker"},
					{ID: "headphones", Name: "AirPods Pro", Type: "Headphones"},
				}))
				m.list.Select(1)
				m.resultCount = 3
				m.inputFocused = true
				m.input.SetValue("/device kitchen")
				m.lastAction = "Selected device: MacBook Pro"
				m.playback = app.PlaybackState{
					Device:     app.Device{ID: "mac", Name: "MacBook Pro", Type: "Computer", IsActive: true},
					IsPlaying:  false,
					Progress:   41 * time.Second,
					Duration:   226 * time.Second,
					ItemName:   "Lonely City",
					ArtistName: "Moses Sumney",
				}
				m.localPlayer = localPlayerStatus{
					supported:       true,
					binaryAvailable: true,
					process:         "running",
					device:          "spotui-speaker",
					message:         "ready",
				}
				return m
			},
		},
		{
			id:          "compact-local-player",
			title:       "Compact Local Player Bootstrap",
			width:       72,
			height:      18,
			focus:       "Inspect density and whether the compact layout still feels deliberate instead of cramped.",
			description: "Compact terminal with no active playback device and the local player in a transitional state.",
			build: func() model {
				m := baseQAModel()
				m.inputFocused = true
				m.input.SetValue("/local use")
				m.lastAction = "Started local player. Current state: starting"
				m.localPlayer = localPlayerStatus{
					supported:       true,
					binaryAvailable: true,
					process:         "starting",
					device:          "spotui-speaker",
					message:         "waiting for Spotify Connect",
				}
				m.playback = app.PlaybackState{
					IsPlaying: false,
					ItemName:  "Nothing playing",
				}
				m.bannerText = "Waiting for spotifyd to register as a Spotify device."
				return m
			},
		},
		{
			id:          "narrow-command-suggestions",
			title:       "Narrow Command Suggestions",
			width:       54,
			height:      14,
			focus:       "Check whether autocomplete remains legible and controlled in a narrow terminal without visual spill or confusion.",
			description: "Narrow layout with command suggestions open and long command descriptions under pressure.",
			build: func() model {
				m := baseQAModel()
				m.inputFocused = true
				m.input.SetValue("/local ")
				m.suggestionsOpen = true
				m.suggestions = []suggestion{
					{value: "/local start", insertValue: "/local start", description: "start the lightweight local player"},
					{value: "/local use", insertValue: "/local use", description: "start and select the local player"},
					{value: "/local status", insertValue: "/local status", description: "show local-player status"},
					{value: "/local reset", insertValue: "/local reset", description: "clear runtime metadata"},
				}
				m.suggestionIndex = 1
				m.localPlayer = localPlayerStatus{
					supported:       true,
					binaryAvailable: false,
					message:         "spotifyd binary not found",
				}
				return m
			},
		},
	}
}

func baseQAModel() model {
	m := newModel(nil)
	m.bootAnimationDone = true
	m.connectionStatus = "Connected as Petros Xen (11124806036)"
	m.accentColor = "#8ab6a2"
	m.list.SetItems([]list.Item{})
	return m
}

func renderTerminalSVG(title string, cols int, rows int, content string) string {
	const (
		cellWidth    = 9
		lineHeight   = 20
		paddingX     = 28
		paddingY     = 34
		headerHeight = 36
	)

	lines := strings.Split(content, "\n")
	maxWidth := cols
	for _, line := range lines {
		if width := lipgloss.Width(line); width > maxWidth {
			maxWidth = width
		}
	}

	canvasWidth := paddingX*2 + (maxWidth * cellWidth)
	canvasHeight := paddingY*2 + headerHeight + (maxInt(rows, len(lines)) * lineHeight)

	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="%s">`, canvasWidth, canvasHeight, canvasWidth, canvasHeight, html.EscapeString(title)))
	b.WriteString(`<rect width="100%" height="100%" fill="#0f1110"/>`)
	b.WriteString(fmt.Sprintf(`<rect x="14" y="14" width="%d" height="%d" rx="18" fill="#161918" stroke="#2b312e"/>`, canvasWidth-28, canvasHeight-28))
	b.WriteString(`<circle cx="42" cy="37" r="6" fill="#ff5f57"/>`)
	b.WriteString(`<circle cx="62" cy="37" r="6" fill="#febc2e"/>`)
	b.WriteString(`<circle cx="82" cy="37" r="6" fill="#28c840"/>`)
	b.WriteString(fmt.Sprintf(`<text x="%d" y="42" fill="#8e9793" font-family="SFMono-Regular, Menlo, Consolas, monospace" font-size="14">%s</text>`, paddingX+72, html.EscapeString(fmt.Sprintf("%s  ·  %dx%d", title, cols, rows))))

	baseY := paddingY + headerHeight
	for i, line := range lines {
		y := baseY + (i * lineHeight)
		b.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="#d6ddd8" font-family="SFMono-Regular, Menlo, Consolas, monospace" font-size="15" xml:space="preserve">%s</text>`, paddingX, y, html.EscapeString(line)))
	}

	b.WriteString(`</svg>`)
	return b.String()
}

func qaReviewBrief(bundle qaReviewBundle) string {
	var b strings.Builder
	b.WriteString("# spotui TUI QA Review Brief\n\n")
	b.WriteString("Review this bundle as a UI quality-control pass, not as a code audit. Judge whether the TUI feels minimal, functional, and polished, with the restraint and smoothness expected from an Apple-quality production surface.\n\n")
	b.WriteString("## What To Review\n\n")
	b.WriteString("- Visual hierarchy: is the most important information obvious within a second?\n")
	b.WriteString("- Density: does each layout breathe, or do panels feel cramped or noisy?\n")
	b.WriteString("- Interaction clarity: is focus, selection, and command intent easy to parse?\n")
	b.WriteString("- Compact behavior: do narrow and short layouts degrade cleanly without awkward wrapping?\n")
	b.WriteString("- Finish quality: does the experience feel deliberate and calm rather than merely functional?\n\n")
	b.WriteString("## Output Format\n\n")
	b.WriteString("1. List blockers first. Only include issues that materially hurt quality or confidence.\n")
	b.WriteString("2. Then list polish improvements.\n")
	b.WriteString("3. End with a short verdict: `ship`, `ship with fixes`, or `needs another pass`.\n\n")
	b.WriteString("## Scenarios\n\n")
	for _, scenario := range bundle.Scenarios {
		b.WriteString(fmt.Sprintf("- `%s`: `%s` (`%dx%d`) - %s\n", scenario.ID, scenario.Title, scenario.Width, scenario.Height, scenario.Focus))
	}
	return b.String()
}

func stripANSI(raw string) string {
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		if raw[i] == 0x1b {
			i++
			for i < len(raw) && raw[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(raw[i])
	}
	return b.String()
}
