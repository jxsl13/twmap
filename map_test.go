package twmap

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// testMapFiles returns all .map files in testdata/.
func testMapFiles(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob("testdata/*.map")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Skip("no .map files in testdata/")
	}
	return matches
}

func TestParseAllMaps(t *testing.T) {
	for _, path := range testMapFiles(t) {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			m, err := Parse(bytes.NewReader(data), WithRequireInfo(false))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Basic structural sanity checks
			if len(m.Groups) == 0 {
				t.Error("no groups")
			}

			var hasGame bool
			for _, g := range m.Groups {
				for _, l := range g.Layers {
					if l.Kind == LayerKindGame {
						hasGame = true
						if l.Width <= 0 || l.Height <= 0 {
							t.Errorf("game layer has invalid dimensions: %dx%d", l.Width, l.Height)
						}
						if len(l.Tiles) != l.Width*l.Height {
							t.Errorf("game layer tiles: got %d, want %d", len(l.Tiles), l.Width*l.Height)
						}
					}
				}
			}
			if !hasGame {
				t.Error("no game layer found")
			}

			t.Logf("version=%d images=%d groups=%d envelopes=%d sounds=%d",
				m.Version, len(m.Images), len(m.Groups), len(m.Envelopes), len(m.Sounds))
			for gi, g := range m.Groups {
				t.Logf("  group %d %q: offset=(%d,%d) parallax=(%d,%d) layers=%d",
					gi, g.Name, g.OffsetX, g.OffsetY, g.ParallaxX, g.ParallaxY, len(g.Layers))
				for li, l := range g.Layers {
					t.Logf("    layer %d %q: kind=%s %s",
						li, l.Name, layerKindStr(l.Kind), layerDetail(l))
				}
			}
		})
	}
}

func TestParseAllMaps_Validate(t *testing.T) {
	for _, path := range testMapFiles(t) {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			if err := Validate(bytes.NewReader(data), WithRequireInfo(false)); err != nil {
				t.Fatalf("Validate failed: %v", err)
			}
		})
	}
}

func TestDatafileRoundTrip(t *testing.T) {
	for _, path := range testMapFiles(t) {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			// Parse the datafile
			df, err := parseDatafile(bytes.NewReader(original))
			if err != nil {
				t.Fatalf("parseDatafile failed: %v", err)
			}

			// Write it back
			var buf bytes.Buffer
			if err := df.writeTo(&buf); err != nil {
				t.Fatalf("writeTo failed: %v", err)
			}

			// Compare
			got := buf.Bytes()
			if !bytes.Equal(original, got) {
				t.Errorf("round-trip mismatch: original=%d bytes, got=%d bytes", len(original), len(got))

				// Find first difference
				minLen := min(len(got), len(original))
				for i := 0; i < minLen; i++ {
					if original[i] != got[i] {
						t.Errorf("first difference at byte %d: original=0x%02x, got=0x%02x", i, original[i], got[i])
						break
					}
				}
			}
		})
	}
}

func TestMapWriteRoundTrip(t *testing.T) {
	for _, path := range testMapFiles(t) {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			// Parse
			m, err := Parse(bytes.NewReader(data), WithRequireInfo(false))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Write
			var buf bytes.Buffer
			if err := m.Write(&buf); err != nil {
				t.Fatalf("Write failed: %v", err)
			}

			// Re-parse the written output
			m2, err := Parse(bytes.NewReader(buf.Bytes()), WithRequireInfo(false))
			if err != nil {
				t.Fatalf("re-Parse failed: %v", err)
			}

			// Compare structural properties
			if len(m2.Groups) != len(m.Groups) {
				t.Errorf("groups: got %d, want %d", len(m2.Groups), len(m.Groups))
			}
			if len(m2.Images) != len(m.Images) {
				t.Errorf("images: got %d, want %d", len(m2.Images), len(m.Images))
			}
			if len(m2.Envelopes) != len(m.Envelopes) {
				t.Errorf("envelopes: got %d, want %d", len(m2.Envelopes), len(m.Envelopes))
			}
			if len(m2.Sounds) != len(m.Sounds) {
				t.Errorf("sounds: got %d, want %d", len(m2.Sounds), len(m.Sounds))
			}
			if m2.Info.Author != m.Info.Author {
				t.Errorf("author: got %q, want %q", m2.Info.Author, m.Info.Author)
			}

			// Compare layers
			for gi, g := range m.Groups {
				if gi >= len(m2.Groups) {
					break
				}
				g2 := m2.Groups[gi]
				if len(g2.Layers) != len(g.Layers) {
					t.Errorf("group %d layers: got %d, want %d", gi, len(g2.Layers), len(g.Layers))
					continue
				}
				for li, l := range g.Layers {
					l2 := g2.Layers[li]
					if l2.Kind != l.Kind {
						t.Errorf("group %d layer %d: kind got %d, want %d", gi, li, l2.Kind, l.Kind)
					}
					if l2.Width != l.Width || l2.Height != l.Height {
						t.Errorf("group %d layer %d: dims got %dx%d, want %dx%d", gi, li, l2.Width, l2.Height, l.Width, l.Height)
					}
					if len(l2.Tiles) != len(l.Tiles) {
						t.Errorf("group %d layer %d: tiles got %d, want %d", gi, li, len(l2.Tiles), len(l.Tiles))
					} else {
						for ti, tile := range l.Tiles {
							if l2.Tiles[ti] != tile {
								t.Errorf("group %d layer %d tile %d: got %+v, want %+v", gi, li, ti, l2.Tiles[ti], tile)
								break
							}
						}
					}
					if len(l2.Quads) != len(l.Quads) {
						t.Errorf("group %d layer %d: quads got %d, want %d", gi, li, len(l2.Quads), len(l.Quads))
					}
				}
			}
		})
	}
}

func layerKindStr(k LayerKind) string {
	switch k {
	case LayerKindTiles:
		return "Tiles"
	case LayerKindGame:
		return "Game"
	case LayerKindFront:
		return "Front"
	case LayerKindTele:
		return "Tele"
	case LayerKindSpeedup:
		return "Speedup"
	case LayerKindSwitch:
		return "Switch"
	case LayerKindTune:
		return "Tune"
	case LayerKindQuads:
		return "Quads"
	case LayerKindSounds:
		return "Sounds"
	case LayerKindInvalid:
		return "Invalid"
	default:
		return fmt.Sprintf("Unknown(%d)", k)
	}
}

func layerDetail(l Layer) string {
	switch l.Kind {
	case LayerKindTiles, LayerKindGame, LayerKindFront:
		return fmt.Sprintf("%dx%d tiles=%d img=%d", l.Width, l.Height, len(l.Tiles), l.ImageID)
	case LayerKindTele:
		return fmt.Sprintf("%dx%d tele=%d", l.Width, l.Height, len(l.TeleTiles))
	case LayerKindSpeedup:
		return fmt.Sprintf("%dx%d speedup=%d", l.Width, l.Height, len(l.SpeedupTiles))
	case LayerKindSwitch:
		return fmt.Sprintf("%dx%d switch=%d", l.Width, l.Height, len(l.SwitchTiles))
	case LayerKindTune:
		return fmt.Sprintf("%dx%d tune=%d", l.Width, l.Height, len(l.TuneTiles))
	case LayerKindQuads:
		return fmt.Sprintf("quads=%d img=%d", len(l.Quads), l.QuadImageID)
	case LayerKindSounds:
		return fmt.Sprintf("sources=%d sound=%d", len(l.SoundSources), l.SoundID)
	default:
		return ""
	}
}
