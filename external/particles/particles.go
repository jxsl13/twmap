// Package particles registers the DDNet particles sprite sheet with the
// twmap package as a side effect of being imported:
//
//	import _ "github.com/jxsl13/twmap/external/particles"
//
// The package embeds particles.png from the DDNet data directory and
// registers it with [twmap.RegisterParticleImage] during init().
// When registered, the [twmap.WithParticles] render option can draw
// static particle/capability markers from this image.
package particles

import (
	"embed"
	"image/png"

	"github.com/jxsl13/twmap"
)

//go:embed particles.png
var particlesFS embed.FS

func init() {
	f, err := particlesFS.Open("particles.png")
	if err != nil {
		return
	}
	defer f.Close()

	decoded, err := png.Decode(f)
	if err != nil {
		return
	}

	twmap.RegisterParticleImage(twmap.ToNRGBA(decoded))
}

// Region represents a rectangular area within the sprite sheet.
type Region struct {
	X, Y, W, H int
}

// Known sprite regions in the 512x512 particles.png, laid out on a
// 8x8 grid of 64x64 cells. These match the DDNet source positions.
var (
	SpriteSlice  = Region{X: 0, Y: 0, W: 64, H: 64}
	SpriteBall   = Region{X: 64, Y: 0, W: 64, H: 64}
	SpriteSplat0 = Region{X: 128, Y: 0, W: 64, H: 64}
	SpriteSplat1 = Region{X: 192, Y: 0, W: 64, H: 64}
	SpriteSplat2 = Region{X: 256, Y: 0, W: 64, H: 64}

	SpriteSmoke   = Region{X: 0, Y: 64, W: 64, H: 64}
	SpriteShell   = Region{X: 64, Y: 64, W: 64, H: 64}
	SpriteExpl    = Region{X: 128, Y: 64, W: 128, H: 128}
	SpriteAirJump = Region{X: 320, Y: 64, W: 64, H: 64}

	SpriteSpiral = Region{X: 0, Y: 128, W: 128, H: 128}
	SpriteSpark  = Region{X: 128, Y: 128, W: 128, H: 128}

	SpriteExplBig = Region{X: 0, Y: 256, W: 256, H: 256}
)
