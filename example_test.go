package twmap_test

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"

	"github.com/jxsl13/twmap"
	_ "github.com/jxsl13/twmap/external" // register default tilesets
)

// ExampleParse demonstrates how to parse a Teeworlds/DDNet map file
// and inspect its structure.
func ExampleParse() {
	f, err := os.Open("testdata/example.map")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	m, err := twmap.Parse(f)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Author:", m.Info.Author)
	fmt.Println("Images:", len(m.Images))
	fmt.Println("Groups:", len(m.Groups))

	for i, g := range m.Groups {
		fmt.Printf("Group %d: %d layer(s), offset=(%d,%d)\n",
			i, len(g.Layers), g.OffsetX, g.OffsetY)
	}
}

// ExampleParseInfo demonstrates how to extract only map metadata
// without decoding images or layers.
func ExampleParseInfo() {
	f, err := os.Open("testdata/example.map")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	info, err := twmap.ParseInfo(f)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Author: ", info.Author)
	fmt.Println("Version:", info.Version)
	fmt.Println("Credits:", info.Credits)
	fmt.Println("License:", info.License)
}

// ExampleValidate demonstrates how to validate the structural
// integrity of a map file.
func ExampleValidate() {
	f, err := os.Open("testdata/example.map")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := twmap.Validate(f); err != nil {
		if errors.Is(err, twmap.ErrNoGameLayer) {
			fmt.Println("map is missing a game layer")
		} else {
			fmt.Println("invalid map:", err)
		}
		return
	}
	fmt.Println("map is valid")
}

// ExampleValidate_multipleErrors shows how to check for specific
// validation errors using errors.Is.
func ExampleValidate_multipleErrors() {
	var buf bytes.Buffer // empty reader simulates a broken file

	err := twmap.Validate(&buf)
	if err == nil {
		fmt.Println("valid")
		return
	}

	switch {
	case errors.Is(err, twmap.ErrMissingVersion):
		fmt.Println("missing version item")
	case errors.Is(err, twmap.ErrNoGameLayer):
		fmt.Println("no game layer")
	case errors.Is(err, twmap.ErrTooManyGameLayers):
		fmt.Println("duplicate game layer")
	case errors.Is(err, twmap.ErrInconsistentGameLayerDimensions):
		fmt.Println("special layer size mismatch")
	default:
		fmt.Println("error:", err)
	}
}

// ExampleRender demonstrates a one-step parse-and-render workflow
// that produces a PNG thumbnail from a map file.
// Tilesets are available automatically via the blank import of the
// external package at the top of this file.
func ExampleRender() {
	f, err := os.Open("testdata/example.map")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Render a 800x600 thumbnail.
	thumb, err := twmap.Render(f, twmap.WithMaxSize(800, 600))
	if err != nil {
		log.Fatal(err)
	}

	out, err := os.Create("thumbnail.png")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	if err := png.Encode(out, thumb); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("thumbnail: %dx%d\n", thumb.Bounds().Dx(), thumb.Bounds().Dy())
}

// ExampleRenderMap demonstrates rendering from an already-parsed Map,
// useful when you need to inspect the map before rendering.
func ExampleRenderMap() {
	f, err := os.Open("testdata/example.map")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	m, err := twmap.Parse(f)
	if err != nil {
		log.Fatal(err)
	}

	// Inspect the map before rendering.
	fmt.Println("Author:", m.Info.Author)

	thumb, err := twmap.RenderMap(m, twmap.WithMaxSize(400, 300))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("thumbnail: %dx%d\n", thumb.Bounds().Dx(), thumb.Bounds().Dy())
}

// ExampleRegisterExternalImage demonstrates how to register a custom
// tileset image manually, without using the external package.
func ExampleRegisterExternalImage() {
	// Create a minimal 1x1 tileset image.
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	twmap.RegisterExternalImage("my_tileset", img)
	fmt.Println("registered")
	// Output: registered
}

// ExampleParse_iterateLayers shows how to walk through all layers
// in a parsed map and classify them by kind.
func ExampleParse_iterateLayers() {
	f, err := os.Open("testdata/example.map")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	m, err := twmap.Parse(f)
	if err != nil {
		log.Fatal(err)
	}

	for _, g := range m.Groups {
		for _, l := range g.Layers {
			switch {
			case l.Kind == twmap.LayerKindGame:
				fmt.Printf("Game layer: %dx%d\n", l.Width, l.Height)
			case l.IsTilemap():
				fmt.Printf("Tile layer %q: %dx%d, image=%d\n",
					l.Name, l.Width, l.Height, l.ImageID)
			case l.Kind == twmap.LayerKindQuads:
				fmt.Printf("Quad layer %q: %d quad(s)\n",
					l.Name, len(l.Quads))
			}
		}
	}
}

// ExampleNewMap demonstrates building a multi-layer map from scratch with
// the builder helpers, then serialising it to the Teeworlds datafile format.
func ExampleNewMap() {
	m := twmap.NewMap(twmap.MapVersion06)
	m.Info.Author = "me"

	// A group holds layers and shares offset/parallax (defaults to 100/100).
	g := m.AddGroup(twmap.NewGroup("Game"))

	// Background tile layer, filled with a solid tile.
	bg := g.AddLayer(twmap.NewTileLayer("bg", 20, 10))
	bg.Fill(twmap.Tile{ID: twmap.TileSolid})

	// Game (physics) layer with a couple of solid tiles.
	game := g.AddLayer(twmap.NewGameLayer(20, 10))
	game.SetTile(0, 9, twmap.Tile{ID: twmap.TileSolid})
	game.SetTile(19, 9, twmap.Tile{ID: twmap.TileSolid})

	var buf bytes.Buffer
	if err := m.Write(&buf); err != nil {
		log.Fatal(err)
	}

	// Re-parse to confirm the built map is well-formed.
	out, err := twmap.Parse(&buf)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("author=%s groups=%d layers=%d\n",
		out.Info.Author, len(out.Groups), len(out.Groups[0].Layers))
	// Output: author=me groups=1 layers=2
}

// ExampleGroup_IsPhysicsGroup shows how to find the physics group.
func ExampleGroup_IsPhysicsGroup() {
	f, err := os.Open("testdata/example.map")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	m, err := twmap.Parse(f)
	if err != nil {
		log.Fatal(err)
	}

	for i, g := range m.Groups {
		if g.IsPhysicsGroup() {
			fmt.Printf("Group %d is the physics group (%d layers)\n",
				i, len(g.Layers))
		}
	}
}
