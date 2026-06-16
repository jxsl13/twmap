// Package external is a convenience package that registers the default
// tileset images, entity-layer sprites, speedup-arrow sprite, game skin, and particle sprites with the twmap package
// as a side effect of being imported:
//
//	import _ "github.com/jxsl13/twmap/external"
//
// This is equivalent to importing all five sub-packages individually:
//
//	import _ "github.com/jxsl13/twmap/external/mapres"
//	import _ "github.com/jxsl13/twmap/external/entities"
//	import _ "github.com/jxsl13/twmap/external/gameskin"
//	import _ "github.com/jxsl13/twmap/external/particles"
//	import _ "github.com/jxsl13/twmap/external/speeduparrow"
//
// For finer control, import only the sub-packages you need.
package external

import (
	_ "github.com/jxsl13/twmap/external/entities"
	_ "github.com/jxsl13/twmap/external/gameskin"
	_ "github.com/jxsl13/twmap/external/mapres"
	_ "github.com/jxsl13/twmap/external/particles"
	_ "github.com/jxsl13/twmap/external/speeduparrow"
)
