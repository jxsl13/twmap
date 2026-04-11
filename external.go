package twmap

import (
	"image"
	"strings"
	"sync"
)

// imageRegistry is a concurrency-safe registry for named NRGBA images.
// Used for tileset images, particle images, and any future named image sets.
type imageRegistry struct {
	mu     sync.RWMutex
	images map[string]*image.NRGBA
}

func newImageRegistry() *imageRegistry {
	return &imageRegistry{images: make(map[string]*image.NRGBA)}
}

func (r *imageRegistry) register(name string, img *image.NRGBA) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return
	}
	r.mu.Lock()
	r.images[key] = img
	r.mu.Unlock()
}

func (r *imageRegistry) resolve(name string) *image.NRGBA {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return nil
	}
	r.mu.RLock()
	img := r.images[key]
	r.mu.RUnlock()
	return img
}

// externalImages holds all tileset PNGs, keyed by lowercase name
// without extension (e.g. "grass_main"). Populated by
// RegisterExternalImage, typically called from init() functions
// of packages that provide tilesets.
var externalImages = newImageRegistry()

// RegisterExternalImage registers a tileset image under the given name.
// The name is normalized to lowercase and trimmed of whitespace.
// If an image with the same name already exists, it is replaced.
//
// This function is safe for concurrent use and is intended to be called
// from init() functions of packages that provide tilesets,
// following the same pattern as image/png and image/jpeg:
//
//	import _ "github.com/jxsl13/twmap/external/mapres"
func RegisterExternalImage(name string, img *image.NRGBA) {
	externalImages.register(name, img)
}

// resolveExternalImage looks up an external image by name in the tileset
// registry. Returns nil if the name does not match any registered tileset.
func resolveExternalImage(name string) *image.NRGBA {
	return externalImages.resolve(name)
}

// resolveExternalImage07 looks up an external image for a 0.7 map.
// It first tries the 0.7-specific variant (name_0.7), then falls
// back to the base name.
func resolveExternalImage07(name string) *image.NRGBA {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return nil
	}
	externalImages.mu.RLock()
	img := externalImages.images[key+"_0.7"]
	if img == nil {
		img = externalImages.images[key]
	}
	externalImages.mu.RUnlock()
	return img
}

// particleImage holds the registered particle sprite sheet.
// There is a single active particle image; registering a new one replaces the old.
var particleImage *image.NRGBA //nolint: unused // will be read when particle rendering is implemented

// particleMu guards particleImage for concurrent access.
var particleMu sync.RWMutex

// RegisterParticleImage registers a particle sprite sheet image.
// There is a single active particle image; calling this function replaces
// any previously registered one. The default is provided by importing:
//
//	import _ "github.com/jxsl13/twmap/external/particles"
func RegisterParticleImage(img *image.NRGBA) {
	particleMu.Lock()
	particleImage = img
	particleMu.Unlock()
}

// gameSkinImage holds the registered game skin image (default: "game").
// There is a single active game skin; registering a new one replaces the old.
var gameSkinImage *image.NRGBA

// gameSkinMu guards gameSkinImage for concurrent access.
var gameSkinMu sync.RWMutex

// RegisterGameSkin registers a game skin image. There is a single active
// game skin; calling this function replaces any previously registered skin.
// The default game skin is provided by importing:
//
//	import _ "github.com/jxsl13/twmap/external/gameskin"
//
// To use a custom game skin, call this function with your own image after
// the default has been registered (or without importing the default).
func RegisterGameSkin(img *image.NRGBA) {
	gameSkinMu.Lock()
	gameSkinImage = img
	gameSkinMu.Unlock()
}

// resolveGameSkin returns the currently registered game skin, or nil.
func resolveGameSkin() *image.NRGBA {
	gameSkinMu.RLock()
	img := gameSkinImage
	gameSkinMu.RUnlock()
	return img
}

// ToNRGBA converts any [image.Image] to [*image.NRGBA].
// If the source is already *image.NRGBA it is returned as-is.
// This is a convenience helper for preparing images before passing them
// to [RegisterExternalImage], [RegisterGameSkin], or [RegisterParticleImage].
func ToNRGBA(src image.Image) *image.NRGBA {
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
