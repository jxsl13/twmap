// Package speeduparrow registers the default DDNet speedup arrow image
// with the twmap package as a side effect of being imported:
//
//	import _ "github.com/jxsl13/twmap/external/speeduparrow"
package speeduparrow

import (
	"embed"
	"image"
	"image/png"

	"github.com/jxsl13/twmap"
)

//go:embed speed_arrow.png speed_arrow_array.png
var speedupArrowFS embed.FS

func init() {
	register("speed_arrow.png", twmap.RegisterSpeedupArrowImage)
	register("speed_arrow_array.png", twmap.RegisterSpeedupArrowArrayImage)
}

// register decodes an embedded PNG and hands it to the given twmap registrar.
// Missing or undecodable assets are skipped silently, matching the prior
// best-effort init behaviour.
func register(name string, registrar func(*image.NRGBA)) {
	f, err := speedupArrowFS.Open(name)
	if err != nil {
		return
	}
	defer f.Close()

	decoded, err := png.Decode(f)
	if err != nil {
		return
	}

	registrar(twmap.ToNRGBA(decoded))
}
