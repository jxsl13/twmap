// Package gameskin registers the default DDNet game skin sprite sheet
// (game.png) with the twmap package as a side effect of being imported:
//
//	import _ "github.com/jxsl13/twmap/external/gameskin"
//
// The game skin is used by the [twmap.WithEntities] render option to draw
// pickup, flag, and spawn sprites from the sprite sheet. Without this
// import (or a custom game skin registered via [twmap.RegisterGameSkin]),
// entity rendering falls back to simple colored shapes.
//
// To use a custom game skin, call [twmap.RegisterGameSkin] with your own
// image. The image must follow the same 1024×512 layout (32×16 grid of
// 32×32 cells) as the DDNet game.png.
package gameskin

import (
	"embed"
	"image"
	"image/png"

	"github.com/jxsl13/twmap"
)

//go:embed game.png
var gameskinFS embed.FS

func init() {
	f, err := gameskinFS.Open("game.png")
	if err != nil {
		return
	}
	defer f.Close()

	decoded, err := png.Decode(f)
	if err != nil {
		return
	}

	twmap.RegisterGameSkin(twmap.ToNRGBA(decoded))
}

// GridCell is the pixel size of one cell in the game.png grid (32×16 grid).
const GridCell = 32

// SpriteRegion defines a rectangle in the game.png sprite sheet.
// X, Y, W, H are in grid cells (multiply by GridCell for pixels).
type SpriteRegion struct {
	X, Y, W, H int
}

// Pixel-level bounds from a SpriteRegion.
func (r SpriteRegion) Bounds() image.Rectangle {
	return image.Rect(r.X*GridCell, r.Y*GridCell, (r.X+r.W)*GridCell, (r.Y+r.H)*GridCell)
}

// Sprite regions for entities, matching DDNet content.py definitions.
// Grid coordinates: set_game = SpriteSet("game", image_game, 32, 16)
var (
	SpritePickupHealth  = SpriteRegion{X: 10, Y: 2, W: 2, H: 2} // heart
	SpritePickupArmor   = SpriteRegion{X: 12, Y: 2, W: 2, H: 2} // shield
	SpritePickupShotgun = SpriteRegion{X: 2, Y: 6, W: 8, H: 2}
	SpritePickupGrenade = SpriteRegion{X: 2, Y: 8, W: 7, H: 2}
	SpritePickupNinja   = SpriteRegion{X: 2, Y: 10, W: 8, H: 2}
	SpritePickupLaser   = SpriteRegion{X: 2, Y: 12, W: 7, H: 3}

	SpriteFlagBlue = SpriteRegion{X: 12, Y: 8, W: 4, H: 8}
	SpriteFlagRed  = SpriteRegion{X: 16, Y: 8, W: 4, H: 8}

	SpriteHealthFull = SpriteRegion{X: 21, Y: 0, W: 2, H: 2} // HUD heart
	SpriteArmorFull  = SpriteRegion{X: 21, Y: 2, W: 2, H: 2} // HUD shield

	SpriteStar1 = SpriteRegion{X: 15, Y: 0, W: 2, H: 2} // spawn indicator
	SpriteStar2 = SpriteRegion{X: 17, Y: 0, W: 2, H: 2}
	SpriteStar3 = SpriteRegion{X: 19, Y: 0, W: 2, H: 2}

	SpritePickupArmorShotgun = SpriteRegion{X: 15, Y: 2, W: 2, H: 2}
	SpritePickupArmorGrenade = SpriteRegion{X: 17, Y: 2, W: 2, H: 2}
	SpritePickupArmorNinja   = SpriteRegion{X: 10, Y: 10, W: 2, H: 2}
	SpritePickupArmorLaser   = SpriteRegion{X: 19, Y: 2, W: 2, H: 2}
)
