package twmap

import (
	"image"
	"strings"
	"sync"
)

// externalImages holds all tileset PNGs, keyed by lowercase name
// without extension (e.g. "grass_main"). Populated by
// RegisterExternalImage, typically called from init() functions
// of packages that provide tilesets.
var externalImages = make(map[string]*image.NRGBA)

// externalMu guards externalImages for concurrent registration.
var externalMu sync.RWMutex

// RegisterExternalImage registers a tileset image under the given name.
// The name is normalized to lowercase and trimmed of whitespace.
// If an image with the same name already exists, it is replaced.
//
// This function is safe for concurrent use and is intended to be called
// from init() functions of packages that provide tilesets,
// following the same pattern as image/png and image/jpeg:
//
//	import _ "github.com/jxsl13/twmap/external"
func RegisterExternalImage(name string, img *image.NRGBA) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return
	}
	externalMu.Lock()
	externalImages[key] = img
	externalMu.Unlock()
}

// resolveExternalImage looks up an external image by name in the tileset
// registry. Returns nil if the name does not match any registered tileset.
func resolveExternalImage(name string) *image.NRGBA {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return nil
	}
	externalMu.RLock()
	img := externalImages[key]
	externalMu.RUnlock()
	return img
}

// toNRGBA converts any image.Image to *image.NRGBA.
func toNRGBA(src image.Image) *image.NRGBA {
	if nrgba, ok := src.(*image.NRGBA); ok {
		return nrgba
	}
	bounds := src.Bounds()
	nrgba := image.NewNRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			nrgba.Set(x, y, src.At(x, y))
		}
	}
	return nrgba
}
