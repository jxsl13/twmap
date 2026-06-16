// Package mapres registers the default set of DDNet/Teeworlds tileset
// images with the twmap package as a side effect of being imported:
//
//	import _ "github.com/jxsl13/twmap/external/mapres"
//
// The package embeds all PNG files in its directory and registers them
// with [twmap.RegisterExternalImage] during init(). Images are keyed by
// filename without extension, lowercased (e.g. "grass_main.png" becomes
// "grass_main").
package mapres

import (
	"embed"
	"image/png"
	"strings"

	"github.com/jxsl13/twmap"
)

//go:embed *.png
var mapresFS embed.FS

func init() {
	entries, err := mapresFS.ReadDir(".")
	if err != nil {
		return
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".png") {
			continue
		}
		name := e.Name()
		key := strings.TrimSuffix(name, ".png")

		f, err := mapresFS.Open(name)
		if err != nil {
			continue
		}

		decoded, err := png.Decode(f)
		f.Close()
		if err != nil {
			continue
		}

		twmap.RegisterExternalImage(key, twmap.ToNRGBA(decoded))
	}
}
