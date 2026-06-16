package twmap

import (
	"bytes"
	"math/bits"
	"testing"

	"image/color"
)

func testFixedPointMap() *Map {
	return &Map{
		Version: MapVersion06,
		Groups: []Group{{
			ParallaxX: 100,
			ParallaxY: 100,
			Layers: []Layer{
				{
					Kind:     LayerKindGame,
					Width:    1,
					Height:   1,
					ImageID:  -1,
					ColorEnv: -1,
					Tiles:    []Tile{{ID: TileAir}},
				},
				{
					Kind:        LayerKindQuads,
					QuadImageID: -1,
					Quads: []Quad{{
						Points: [5]Point{{X: 12345, Y: -23456}, {X: 34567, Y: -45678}, {X: 56789, Y: 67890}, {X: -13579, Y: 24680}, {X: 11111, Y: -22222}},
						Colors: [4]color.NRGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}, {R: 255, G: 255, A: 255}},
						TexCoords: [4]Point{{X: 100, Y: 200}, {X: 300, Y: 400}, {X: 500, Y: 600}, {X: 700, Y: 800}},
						PosEnv:    -1,
						ColorEnv:  -1,
					}},
				},
				{
					Kind:    LayerKindSounds,
					SoundID: -1,
					SoundSources: []SoundSource{{
						Position:  Point{X: 22222, Y: -33333},
						ShapeType: ShapeTypeCircle,
					}},
				},
			},
		}},
	}
}

func TestWriteRoundTripPreservesRawFixedPointValues(t *testing.T) {
	m := testFixedPointMap()
	quadWant := m.Groups[0].Layers[1].Quads[0]
	soundWant := m.Groups[0].Layers[2].SoundSources[0]

	var buf bytes.Buffer
	if err := m.Write(&buf); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	m2, err := Parse(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	quadGot := m2.Groups[0].Layers[1].Quads[0]
	if quadGot.Points != quadWant.Points {
		t.Fatalf("quad points changed: got=%v want=%v", quadGot.Points, quadWant.Points)
	}
	if quadGot.TexCoords != quadWant.TexCoords {
		t.Fatalf("quad texcoords changed: got=%v want=%v", quadGot.TexCoords, quadWant.TexCoords)
	}

	soundGot := m2.Groups[0].Layers[2].SoundSources[0]
	if soundGot.Position != soundWant.Position {
		t.Fatalf("sound position changed: got=%v want=%v", soundGot.Position, soundWant.Position)
	}
}

func TestWriteFailsOnOutOfRangeFixedPointRawInts(t *testing.T) {
	if bits.UintSize < 64 {
		t.Skip("requires 64-bit int to construct out-of-range raw fixed-point values")
	}

	overMax := int(int64(1<<31))
	underMin := int(int64(-1<<31) - 1)

	tests := []struct {
		name string
		mutate func(*Map)
	}{
		{
			name: "quad point overflow",
			mutate: func(m *Map) {
				m.Groups[0].Layers[1].Quads[0].Points[0].X = overMax
			},
		},
		{
			name: "quad texcoord underflow",
			mutate: func(m *Map) {
				m.Groups[0].Layers[1].Quads[0].TexCoords[0].Y = underMin
			},
		},
		{
			name: "sound position overflow",
			mutate: func(m *Map) {
				m.Groups[0].Layers[2].SoundSources[0].Position.X = overMax
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := testFixedPointMap()
			tt.mutate(m)

			var buf bytes.Buffer
			if err := m.Write(&buf); err == nil {
				t.Fatalf("expected Write to fail for out-of-range raw fixed-point value")
			}
		})
	}
}