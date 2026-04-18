package ui

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

func (m *model) refreshAccentColor() tea.Cmd {
	if !m.playback.IsPlaying || m.playback.AlbumArtURL == "" {
		m.accentColor = ""
		return nil
	}
	if cached, ok := m.accentColorCache[m.playback.AlbumArtURL]; ok {
		m.accentColor = cached
		return nil
	}
	return fetchAccentColorCmd(m.playback.AlbumArtURL)
}

func fetchAccentColorCmd(albumArtURL string) tea.Cmd {
	return func() tea.Msg {
		color, err := dominantColorFromImageURL(albumArtURL)
		return accentColorMsg{albumArtURL: albumArtURL, color: color, err: err}
	}
}

func (m model) accent(text string) string {
	if m.accentColor == "" || text == "" {
		return text
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(m.accentColor)).Render(text)
}

func dominantColorFromImageURL(rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("fetch album art: HTTP %d", resp.StatusCode)
	}
	img, _, err := image.Decode(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return "", err
	}
	return extractAccentColor(img), nil
}

func extractAccentColor(img image.Image) string {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return ""
	}

	stepX := maxInt(1, width/24)
	stepY := maxInt(1, height/24)
	var sumR, sumG, sumB, total float64
	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r, g, b, _ := img.At(x, y).RGBA()
			color := colorful.Color{
				R: float64(r) / 65535.0,
				G: float64(g) / 65535.0,
				B: float64(b) / 65535.0,
			}
			h, c, l := color.Hcl()
			if math.IsNaN(h) {
				continue
			}
			weight := 0.35 + c + (0.5 - math.Abs(l-0.5))
			if weight < 0.1 {
				weight = 0.1
			}
			sumR += color.R * weight
			sumG += color.G * weight
			sumB += color.B * weight
			total += weight
		}
	}
	if total == 0 {
		return ""
	}

	base := colorful.Color{
		R: sumR / total,
		G: sumG / total,
		B: sumB / total,
	}
	h, c, l := base.Hcl()
	if math.IsNaN(h) {
		return ""
	}

	targetChroma := clampFloat(c*0.85+0.06, 0.07, 0.16)
	targetLightness := clampFloat(0.56-(l-0.5)*0.35, 0.42, 0.62)

	accent := colorful.Hcl(h, targetChroma, targetLightness)
	if !accent.IsValid() {
		accent = base
	}
	return accent.Clamped().Hex()
}
