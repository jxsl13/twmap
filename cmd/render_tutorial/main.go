package main

import (
	"fmt"
	"image/png"
	"log"
	"math"
	"os"

	"github.com/jxsl13/twmap"
	_ "github.com/jxsl13/twmap/external"
)

func main() {
	f, err := os.Open("testdata/Tutorial.map")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	m, err := twmap.Parse(f)
	if err != nil {
		log.Fatal(err)
	}

	b := gameLayerBounds(m)
	cx := float64(b.MinX+b.MaxX) / 2.0
	cy := float64(b.MinY+b.MaxY) / 2.0

	thumb, err := twmap.RenderMap(m,
		twmap.WithDetail(true),
		twmap.WithGameLayer(true),
		twmap.WithEntities(true),
		twmap.WithRegion(b),
		twmap.WithCamera(cx, cy),
	)
	if err != nil {
		log.Fatal(err)
	}

	out, err := os.Create("tutorial_rendered.png")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	if err := png.Encode(out, thumb); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("rendered: %dx%d -> tutorial_rendered.png\n", thumb.Bounds().Dx(), thumb.Bounds().Dy())
}

func gameLayerBounds(m *twmap.Map) twmap.MapBounds {
	b := twmap.MapBounds{MinX: math.MaxInt, MinY: math.MaxInt, MaxX: math.MinInt, MaxY: math.MinInt}
	hasGame := false

	for _, g := range m.Groups {
		offX := -int(g.OffsetX) / 32
		offY := -int(g.OffsetY) / 32
		for _, l := range g.Layers {
			if l.Kind != twmap.LayerKindGame {
				continue
			}
			hasGame = true
			if offX < b.MinX {
				b.MinX = offX
			}
			if offY < b.MinY {
				b.MinY = offY
			}
			if offX+l.Width > b.MaxX {
				b.MaxX = offX + l.Width
			}
			if offY+l.Height > b.MaxY {
				b.MaxY = offY + l.Height
			}
		}
	}

	if hasGame {
		return b
	}
	return m.Bounds()
}
