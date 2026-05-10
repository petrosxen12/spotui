package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteQAReviewBundle(t *testing.T) {
	dir := t.TempDir()

	if err := WriteQAReviewBundle(dir); err != nil {
		t.Fatalf("WriteQAReviewBundle() error = %v", err)
	}

	manifestPath := filepath.Join(dir, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", manifestPath, err)
	}

	var bundle qaReviewBundle
	if err := json.Unmarshal(manifestData, &bundle); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(bundle.Scenarios) != 4 {
		t.Fatalf("len(bundle.Scenarios) = %d, want 4", len(bundle.Scenarios))
	}

	for _, scenario := range bundle.Scenarios {
		textPath := filepath.Join(dir, scenario.TextFile)
		svgPath := filepath.Join(dir, scenario.SVGFile)

		textData, err := os.ReadFile(textPath)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", textPath, err)
		}
		if len(textData) == 0 {
			t.Fatalf("expected %q to contain rendered TUI output", textPath)
		}

		svgData, err := os.ReadFile(svgPath)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", svgPath, err)
		}
		if len(svgData) == 0 || string(svgData[:4]) != "<svg" {
			t.Fatalf("expected %q to contain an svg document", svgPath)
		}
	}

	briefPath := filepath.Join(dir, "review-brief.md")
	briefData, err := os.ReadFile(briefPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", briefPath, err)
	}
	if len(briefData) == 0 {
		t.Fatalf("expected %q to contain a review brief", briefPath)
	}
}
