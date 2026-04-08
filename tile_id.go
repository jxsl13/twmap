package twmap

// Game layer tile IDs from DDNet mapitems.h.
// These identify the gameplay function of each tile in the game layer.
const (
	TileAir        uint8 = 0
	TileSolid      uint8 = 1
	TileDeath      uint8 = 2
	TileUnhookable uint8 = 3 // TILE_NOHOOK
	TileNoLaser    uint8 = 4
	TileThroughCut uint8 = 5
	TileThrough    uint8 = 6
	TileJump       uint8 = 7

	TileFreeze       uint8 = 9
	TileTeleInEvil   uint8 = 10
	TileUnfreeze     uint8 = 11
	TileDeepFreeze   uint8 = 12
	TileDeepUnfreeze uint8 = 13
	TileTeleInWeapon uint8 = 14
	TileTeleInHook   uint8 = 15
	TileWallJump     uint8 = 16
	TileEHookEnable  uint8 = 17
	TileEHookDisable uint8 = 18
	TileHitEnable    uint8 = 19
	TileHitDisable   uint8 = 20
	TileSoloEnable   uint8 = 21
	TileSoloDisable  uint8 = 22
	TileSwitchOpen   uint8 = 24
	TileSwitchClose  uint8 = 25
	TileTeleIn       uint8 = 26
	TileTeleOut      uint8 = 27
	TileSpeedBoost   uint8 = 29
	TileTeleCheckOut uint8 = 30
	TileTeleCheckIn  uint8 = 31
	TileRefillJumps  uint8 = 32

	// DDRace race tiles.
	TileStart  uint8 = 33
	TileFinish uint8 = 34

	// Time checkpoints: 35-59.
	TileTimeCheckFirst uint8 = 35
	TileTimeCheckLast  uint8 = 59

	TileStop          uint8 = 60
	TileStopS         uint8 = 61
	TileStopA         uint8 = 62
	TileTeleCheckEvil uint8 = 63
	TileCP            uint8 = 64
	TileCPF           uint8 = 65
	TileThroughAll    uint8 = 66
	TileThroughDir    uint8 = 67
	TileTune          uint8 = 68
	TileOldLaser      uint8 = 71
	TileNPC           uint8 = 72
	TileEHook         uint8 = 73
	TileNoHit         uint8 = 74
	TileNPH           uint8 = 75
	TileUnlockTeam    uint8 = 76
	TileAddTime       uint8 = 79

	TileNPCDisable        uint8 = 88
	TileUnlimitedJumpsOff uint8 = 89
	TileJetpackOff        uint8 = 90
	TileNPHDisable        uint8 = 91
	TileSubtractTime      uint8 = 95
	TileTeleGunEnable     uint8 = 96
	TileTeleGunDisable    uint8 = 97

	TileNPCEnable        uint8 = 104
	TileUnlimitedJumpsOn uint8 = 105
	TileJetpackOn        uint8 = 106
	TileNPHEnable        uint8 = 107

	TileLiveFreeze   uint8 = 144
	TileLiveUnfreeze uint8 = 145

	// TileSize is the side length of a tile in world coordinate units.
	TileSize = 32
)

// IsSolid reports whether the tile ID blocks player movement.
func IsSolid(id uint8) bool {
	return id == TileSolid || id == TileUnhookable
}

// IsPassable reports whether a player can move through a tile
// (not solid, not death, not freeze).
func IsPassable(id uint8) bool {
	return !IsSolid(id) && id != TileDeath && id != TileFreeze && id != TileDeepFreeze && id != TileLiveFreeze
}
