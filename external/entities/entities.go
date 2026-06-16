// Package entities registers the default DDNet entity-layer sprite sheet
// (entities.png) with the twmap package as a side effect of being imported:
//
//	import _ "github.com/jxsl13/twmap/external/entities"
//
// DDNet treats this sheet separately from mapres tilesets. It is used for the
// game/front/tele/speedup/switch/tune overlay rendering paths enabled by the
// corresponding RenderOptions.
package entities

import (
	"embed"
	"image/png"

	"github.com/jxsl13/twmap"
)

//go:embed entities.png
var entitiesFS embed.FS

func init() {
	f, err := entitiesFS.Open("entities.png")
	if err != nil {
		return
	}
	defer f.Close()

	decoded, err := png.Decode(f)
	if err != nil {
		return
	}

	twmap.RegisterEntitiesImage(twmap.ToNRGBA(decoded))
}
