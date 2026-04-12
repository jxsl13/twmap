package main

import (
	"fmt"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/jxsl13/twmap"
	_ "github.com/jxsl13/twmap/external"
)

const (
	tutorialMapPath = "testdata/Tutorial.map"
	outputDir       = "tutorial_renders"
	outputPrefix    = "tutorial_rendered"
)

type renderToggle struct {
	name string
	opt  twmap.RenderOption
}

func main() {
	f, err := os.Open(tutorialMapPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	m, err := twmap.Parse(f)
	if err != nil {
		log.Fatal(err)
	}

	b := gameLayerBounds(m)
	cx := float64(b.MinX+b.MaxX) / 2.0
	cy := float64(b.MinY+b.MaxY) / 2.0

	if err := os.RemoveAll(outputDir); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		log.Fatal(err)
	}

	baseOptions := []twmap.RenderOption{
		twmap.WithRegion(b),
		twmap.WithCamera(cx, cy),
	}

	toggles := availableToggles(m)

	if err := renderVariant(m, filepath.Join(outputDir, outputPrefix+"_incremental_00_base.png"), baseOptions, nil); err != nil {
		log.Fatal(err)
	}
	progressive := make([]twmap.RenderOption, 0, len(toggles))
	for i, toggle := range toggles {
		progressive = append(progressive, toggle.opt)
		name := fmt.Sprintf("%s_incremental_%02d_%s.png", outputPrefix, i+1, toggle.name)
		if err := renderVariant(m, filepath.Join(outputDir, name), baseOptions, progressive); err != nil {
			log.Fatal(err)
		}
	}

	if err := renderVariant(m, filepath.Join(outputDir, outputPrefix+"_option_00_base.png"), baseOptions, nil); err != nil {
		log.Fatal(err)
	}
	for i, toggle := range toggles {
		fileName := fmt.Sprintf("%s_option_%02d_%s.png", outputPrefix, i+1, toggle.name)
		if err := renderVariant(m, filepath.Join(outputDir, fileName), baseOptions, []twmap.RenderOption{toggle.opt}); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("rendered %d incremental frames and %d non-incremental variants into %s\n", len(toggles)+1, len(toggles)+1, outputDir)
}

func availableToggles(m *twmap.Map) []renderToggle {
	toggles := make([]renderToggle, 0, 9)
	if hasDetailLayers(m) {
		toggles = append(toggles, renderToggle{name: "detail", opt: twmap.WithDetail(true)})
	}
	if hasParticleMarkers(m) {
		toggles = append(toggles, renderToggle{name: "particles", opt: twmap.WithParticles(true)})
	}
	if hasSwitchTiles(m) {
		toggles = append(toggles, renderToggle{name: "switch_layer", opt: twmap.WithSwitchLayer(true)})
	}
	if hasTeleTiles(m) {
		toggles = append(toggles, renderToggle{name: "tele_layer", opt: twmap.WithTeleLayer(true)})
	}
	if hasSpeedupTiles(m) {
		toggles = append(toggles, renderToggle{name: "speedup_layer", opt: twmap.WithSpeedupLayer(true)})
	}
	if hasTuneTiles(m) {
		toggles = append(toggles, renderToggle{name: "tune_layer", opt: twmap.WithTuneLayer(true)})
	}
	if hasGameOverlayTiles(m) {
		toggles = append(toggles, renderToggle{name: "game_layer", opt: twmap.WithGameLayer(true)})
	}
	if hasFrontOverlayTiles(m) {
		toggles = append(toggles, renderToggle{name: "front_layer", opt: twmap.WithFrontLayer(true)})
	}
	if hasRenderableEntities(m) {
		toggles = append(toggles, renderToggle{name: "entities", opt: twmap.WithEntities(true)})
	}
	return toggles
}

func hasDetailLayers(m *twmap.Map) bool {
	for _, g := range m.Groups {
		for _, l := range g.Layers {
			if l.Detail {
				return true
			}
		}
	}
	return false
}

func hasParticleMarkers(m *twmap.Map) bool {
	for _, g := range m.Groups {
		for _, l := range g.Layers {
			if l.Kind != twmap.LayerKindGame && l.Kind != twmap.LayerKindFront {
				continue
			}
			for _, tile := range l.Tiles {
				if isParticleMarker(tile.ID) {
					return true
				}
			}
		}
	}
	return false
}

func hasSwitchTiles(m *twmap.Map) bool {
	for _, g := range m.Groups {
		for _, l := range g.Layers {
			if l.Kind != twmap.LayerKindSwitch {
				continue
			}
			for _, tile := range l.SwitchTiles {
				if twmap.IsValidSwitchTile(tile.ID) {
					return true
				}
			}
		}
	}
	return false
}

func hasTeleTiles(m *twmap.Map) bool {
	for _, g := range m.Groups {
		for _, l := range g.Layers {
			if l.Kind != twmap.LayerKindTele {
				continue
			}
			for _, tile := range l.TeleTiles {
				if twmap.IsValidTeleTile(tile.ID) {
					return true
				}
			}
		}
	}
	return false
}

func hasSpeedupTiles(m *twmap.Map) bool {
	for _, g := range m.Groups {
		for _, l := range g.Layers {
			if l.Kind != twmap.LayerKindSpeedup {
				continue
			}
			for _, tile := range l.SpeedupTiles {
				if twmap.IsValidSpeedupTile(tile.ID) && (tile.Force != 0 || tile.MaxSpeed != 0) {
					return true
				}
			}
		}
	}
	return false
}

func hasTuneTiles(m *twmap.Map) bool {
	for _, g := range m.Groups {
		for _, l := range g.Layers {
			if l.Kind != twmap.LayerKindTune {
				continue
			}
			for _, tile := range l.TuneTiles {
				if twmap.IsValidTuneTile(tile.ID) {
					return true
				}
			}
		}
	}
	return false
}

func hasGameOverlayTiles(m *twmap.Map) bool {
	for _, g := range m.Groups {
		for _, l := range g.Layers {
			if l.Kind != twmap.LayerKindGame {
				continue
			}
			for _, tile := range l.Tiles {
				if tile.ID != twmap.TileAir && twmap.IsValidGameTile(tile.ID) {
					return true
				}
			}
		}
	}
	return false
}

func hasFrontOverlayTiles(m *twmap.Map) bool {
	for _, g := range m.Groups {
		for _, l := range g.Layers {
			if l.Kind != twmap.LayerKindFront {
				continue
			}
			for _, tile := range l.Tiles {
				if tile.ID != twmap.TileAir && twmap.IsValidFrontTile(tile.ID) {
					return true
				}
			}
		}
	}
	return false
}

func hasRenderableEntities(m *twmap.Map) bool {
	for _, g := range m.Groups {
		for _, l := range g.Layers {
			if l.Kind != twmap.LayerKindGame {
				continue
			}
			for _, tile := range l.Tiles {
				if isRenderableEntity(tile.ID) {
					return true
				}
			}
		}
	}
	return false
}

func isRenderableEntity(id uint8) bool {
	switch id {
	case twmap.TileFlagstandRed,
		twmap.TileFlagstandBlue,
		twmap.TileArmor,
		twmap.TileHealth,
		twmap.TileWeaponShotgun,
		twmap.TileWeaponGrenade,
		twmap.TilePowerupNinja,
		twmap.TileWeaponLaser,
		twmap.TileEntityArmorShotgun,
		twmap.TileEntityArmorGrenade,
		twmap.TileEntityArmorNinja,
		twmap.TileEntityArmorLaser:
		return true
	default:
		return false
	}
}

func isParticleMarker(id uint8) bool {
	switch id {
	case twmap.TileUnlimitedJumpsOn,
		twmap.TileUnlimitedJumpsOff,
		twmap.TileJetpackOn,
		twmap.TileJetpackOff,
		twmap.TileTeleGunEnable,
		twmap.TileTeleGunDisable,
		twmap.TileTeleGrenadeEnable,
		twmap.TileTeleGrenadeDisable,
		twmap.TileTeleLaserEnable,
		twmap.TileTeleLaserDisable,
		twmap.TileEHookEnable,
		twmap.TileEHookDisable,
		twmap.TileHitEnable,
		twmap.TileHitDisable,
		twmap.TileNPHEnable,
		twmap.TileNPHDisable,
		twmap.TileAllowTeleGun,
		twmap.TileAllowBlueTeleGun:
		return true
	default:
		return false
	}
}

func renderVariant(m *twmap.Map, outputPath string, baseOptions []twmap.RenderOption, toggles []twmap.RenderOption) error {
	options := make([]twmap.RenderOption, 0, len(baseOptions)+len(toggles))
	options = append(options, baseOptions...)
	options = append(options, toggles...)

	thumb, err := twmap.RenderMap(m, options...)
	if err != nil {
		return err
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if err := png.Encode(out, thumb); err != nil {
		return err
	}

	fmt.Printf("rendered: %dx%d -> %s\n", thumb.Bounds().Dx(), thumb.Bounds().Dy(), outputPath)
	return nil
}

func gameLayerBounds(m *twmap.Map) twmap.MapBounds {
	b := twmap.MapBounds{MinX: math.MaxInt, MinY: math.MaxInt, MaxX: math.MinInt, MaxY: math.MinInt}
	hasGame := false

	for _, g := range m.Groups {
		offX := -int(g.OffsetX) / 32
		offY := -int(g.OffsetY) / 32
		for _, l := range g.Layers {
			if l.Kind != twmap.LayerKindGame {
				continue
			}
			hasGame = true
			if offX < b.MinX {
				b.MinX = offX
			}
			if offY < b.MinY {
				b.MinY = offY
			}
			if offX+l.Width > b.MaxX {
				b.MaxX = offX + l.Width
			}
			if offY+l.Height > b.MaxY {
				b.MaxY = offY + l.Height
			}
		}
	}

	if hasGame {
		return b
	}
	return m.Bounds()
}
