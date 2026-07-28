package render

import (
	"bytes"
	"context"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGolden renders every internal/render/testdata/*.d2 fixture and
// compares the encoded PNG byte-for-byte against a checked-in golden file of
// the same name. Rendering only depends on D2's own (pure Go, deterministic)
// layout/text-measurement and our own pure-software rasterizer, so this
// should be exact-match stable across machines/CI — no perceptual diffing
// needed.
//
// Run with UPDATE_GOLDEN=1 to (re)generate the golden files after an
// intentional rendering change.
func TestGolden(t *testing.T) {
	fixtures, err := filepath.Glob("testdata/*.d2")
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no testdata/*.d2 fixtures found")
	}

	update := os.Getenv("UPDATE_GOLDEN") != ""

	for _, fixture := range fixtures {
		fixture := fixture
		name := strings.TrimSuffix(filepath.Base(fixture), ".d2")

		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(fixture)
			if err != nil {
				t.Fatal(err)
			}

			diagram, err := Compile(context.Background(), string(src))
			if err != nil {
				t.Fatalf("compile: %v", err)
			}

			img, err := Render(diagram, 1)
			if err != nil {
				t.Fatalf("render: %v", err)
			}

			var buf bytes.Buffer
			if err := png.Encode(&buf, img); err != nil {
				t.Fatalf("encode: %v", err)
			}

			goldenPath := filepath.Join("testdata", name+".golden.png")

			if update {
				if err := os.WriteFile(goldenPath, buf.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("missing golden file %s (run `UPDATE_GOLDEN=1 go test ./internal/render/...` to create it): %v", goldenPath, err)
			}
			if !bytes.Equal(want, buf.Bytes()) {
				t.Errorf("rendered PNG for %s differs from %s", fixture, goldenPath)
			}
		})
	}
}
