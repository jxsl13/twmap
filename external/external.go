// Package external registers the default set of DDNet/Teeworlds tileset
// images with the twmap package as a side effect of being imported.
// This follows the same convention as standard library image format
// packages (image/png, image/jpeg) where a blank import triggers
// registration:
//
//	import _ "github.com/jxsl13/twmap/external"
//
// After this import, all embedded tileset PNGs are available when
// rendering map thumbnails — no explicit Init() call is required.
//
// The package embeds all PNG files in its mapres/ directory and
// registers them with [twmap.RegisterExternalImage] during init().
// Images are keyed by filename without extension, lowercased
// (e.g. "mapres/grass_main.png" becomes "grass_main").
//
// # Creating your own tileset package
//
// To ship additional tilesets, follow the same pattern:
//
//  1. Create a Go package with a mapres/ directory containing PNG files.
//  2. Use //go:embed to embed the directory.
//  3. In init(), decode each PNG and call [twmap.RegisterExternalImage].
package external

import (
	"embed"
	"image"
	"image/png"
	"strings"

	"github.com/jxsl13/twmap"
)

//go:embed mapres/*.png
var mapresFS embed.FS

func init() {
	entries, err := mapresFS.ReadDir("mapres")
	if err != nil {
		return
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".png") {
			continue
		}
		name := e.Name()
		key := strings.TrimSuffix(name, ".png")

		f, err := mapresFS.Open("mapres/" + name)
		if err != nil {
			continue
		}

		decoded, err := png.Decode(f)
		f.Close()
		if err != nil {
			continue
		}

		nrgba := toNRGBA(decoded)
		twmap.RegisterExternalImage(key, nrgba)
	}
}

// toNRGBA converts any image.Image to *image.NRGBA.
func toNRGBA(src image.Image) *image.NRGBA {
	if n, ok := src.(*image.NRGBA); ok {
		return n
	}
	bounds := src.Bounds()
	n := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			n.Set(x, y, src.At(x, y))
		}
	}
	return n
}
