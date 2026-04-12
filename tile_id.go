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

	TileFreeze           uint8 = 9
	TileTeleInEvil       uint8 = 10
	TileUnfreeze         uint8 = 11
	TileDeepFreeze       uint8 = 12
	TileDeepUnfreeze     uint8 = 13
	TileTeleInWeapon     uint8 = 14
	TileTeleInHook       uint8 = 15
	TileWallJump         uint8 = 16
	TileEHookEnable      uint8 = 17
	TileEHookDisable     uint8 = 18
	TileHitEnable        uint8 = 19
	TileHitDisable       uint8 = 20
	TileSoloEnable       uint8 = 21
	TileSoloDisable      uint8 = 22
	TileSwitchTimedOpen  uint8 = 22
	TileSwitchTimedClose uint8 = 23
	TileSwitchOpen       uint8 = 24
	TileSwitchClose      uint8 = 25
	TileTeleIn           uint8 = 26
	TileTeleOut          uint8 = 27
	TileSpeedBoostOld    uint8 = 28
	TileSpeedBoost       uint8 = 29
	TileTeleCheck        uint8 = 29
	TileTeleCheckOut     uint8 = 30
	TileTeleCheckIn      uint8 = 31
	TileRefillJumps      uint8 = 32

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
	TileAllowTeleGun      uint8 = 98
	TileAllowBlueTeleGun  uint8 = 99

	TileNPCEnable          uint8 = 104
	TileUnlimitedJumpsOn   uint8 = 105
	TileJetpackOn          uint8 = 106
	TileNPHEnable          uint8 = 107
	TileTeleGrenadeEnable  uint8 = 112
	TileTeleGrenadeDisable uint8 = 113
	TileTeleLaserEnable    uint8 = 128
	TileTeleLaserDisable   uint8 = 129

	TileLiveFreeze   uint8 = 144
	TileLiveUnfreeze uint8 = 145

	// Entity tiles in the game layer (tile ID = 192 + entity index).
	TileEntityOffset       uint8 = 192
	TileSpawn              uint8 = 192 // generic (DM) spawn
	TileSpawnRed           uint8 = 193 // red team spawn
	TileSpawnBlue          uint8 = 194 // blue team spawn
	TileFlagstandRed       uint8 = 195
	TileFlagstandBlue      uint8 = 196
	TileArmor              uint8 = 197
	TileHealth             uint8 = 198
	TileWeaponShotgun      uint8 = 199
	TileWeaponGrenade      uint8 = 200
	TilePowerupNinja       uint8 = 201
	TileWeaponLaser        uint8 = 202
	TileEntityArmorShotgun uint8 = 226
	TileEntityArmorGrenade uint8 = 227
	TileEntityArmorNinja   uint8 = 228
	TileEntityArmorLaser   uint8 = 229

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

// IsSpawn reports whether the tile ID is any kind of spawn point
// (generic, red, or blue).
func IsSpawn(id uint8) bool {
	return id == TileSpawn || id == TileSpawnRed || id == TileSpawnBlue
}

// IsValidEntity reports whether the tile ID addresses the DDNet entity area.
func IsValidEntity(id uint8) bool {
	return id >= TileEntityOffset
}

// IsValidGameTile reports whether the tile ID is valid in the game layer.
func IsValidGameTile(id uint8) bool {
	return id == TileAir ||
		(id >= TileSolid && id <= TileThrough) ||
		id == TileFreeze ||
		(id >= TileUnfreeze && id <= TileDeepUnfreeze) ||
		(id >= TileWallJump && id <= TileSoloDisable) ||
		(id >= TileRefillJumps && id <= TileStopA) ||
		(id >= TileCP && id <= TileThroughDir) ||
		(id >= TileOldLaser && id <= TileUnlockTeam) ||
		(id >= TileNPCDisable && id <= TileNPHDisable) ||
		(id >= TileTeleGunEnable && id <= TileTeleGunDisable) ||
		(id >= TileTeleGrenadeEnable && id <= TileTeleGrenadeDisable) ||
		(id >= TileTeleLaserEnable && id <= TileTeleLaserDisable) ||
		(id >= TileNPCEnable && id <= TileNPHEnable) ||
		id == TileAllowTeleGun ||
		id == TileAllowBlueTeleGun ||
		IsValidEntity(id)
}

// IsValidFrontTile reports whether the tile ID is valid in the front layer.
func IsValidFrontTile(id uint8) bool {
	return id == TileAir ||
		id == TileDeath ||
		(id >= TileNoLaser && id <= TileThrough) ||
		id == TileFreeze ||
		(id >= TileUnfreeze && id <= TileDeepUnfreeze) ||
		(id >= TileWallJump && id <= TileSoloDisable) ||
		(id >= TileRefillJumps && id <= TileStopA) ||
		(id >= TileCP && id <= TileThroughDir) ||
		(id >= TileOldLaser && id <= TileUnlockTeam) ||
		(id >= TileNPCDisable && id <= TileNPHDisable) ||
		(id >= TileTeleGunEnable && id <= TileAllowBlueTeleGun) ||
		(id >= TileTeleGrenadeEnable && id <= TileTeleGrenadeDisable) ||
		(id >= TileTeleLaserEnable && id <= TileTeleLaserDisable) ||
		(id >= TileNPCEnable && id <= TileNPHEnable) ||
		IsValidEntity(id)
}

// IsValidTeleTile reports whether the tile ID is valid in the tele layer.
func IsValidTeleTile(id uint8) bool {
	return id == TileTeleInEvil ||
		id == TileTeleInWeapon ||
		id == TileTeleInHook ||
		id == TileTeleIn ||
		id == TileTeleOut ||
		id == TileTeleCheck ||
		id == TileTeleCheckOut ||
		id == TileTeleCheckIn ||
		id == TileTeleCheckEvil
}

// IsTeleTileCheckpoint reports whether the tele tile uses checkpoint numbering semantics.
func IsTeleTileCheckpoint(id uint8) bool {
	return id == TileTeleCheck || id == TileTeleCheckOut
}

// IsTeleTileNumberUsed reports whether the tele tile consumes a tele number.
func IsTeleTileNumberUsed(id uint8, checkpoint bool) bool {
	if checkpoint {
		return IsTeleTileCheckpoint(id)
	}
	return !IsTeleTileCheckpoint(id) && id != TileTeleCheckIn && id != TileTeleCheckEvil
}

// IsTeleTileNumberUsedAny reports whether the tele tile uses any number overlay.
func IsTeleTileNumberUsedAny(id uint8) bool {
	return id != TileTeleCheckIn && id != TileTeleCheckEvil
}

// IsValidSpeedupTile reports whether the tile ID is valid in the speedup layer.
func IsValidSpeedupTile(id uint8) bool {
	return id == TileSpeedBoostOld || id == TileSpeedBoost
}

// IsValidSwitchTile reports whether the tile ID is valid in the switch layer.
func IsValidSwitchTile(id uint8) bool {
	return id == TileJump ||
		id == TileFreeze ||
		id == TileDeepFreeze ||
		id == TileDeepUnfreeze ||
		id == TileLiveFreeze ||
		id == TileLiveUnfreeze ||
		id == TileHitEnable ||
		id == TileHitDisable ||
		(id >= TileSwitchTimedOpen && id <= TileSwitchClose) ||
		id == TileAddTime ||
		id == TileSubtractTime ||
		id == TileAllowTeleGun ||
		id == TileAllowBlueTeleGun ||
		id >= TileArmor
}

// IsSwitchTileFlagsUsed reports whether the switch tile consumes flip/rotate flags.
func IsSwitchTileFlagsUsed(id uint8) bool {
	return id != TileFreeze && id != TileDeepFreeze && id != TileDeepUnfreeze
}

// IsSwitchTileNumberUsed reports whether the switch tile uses its number field.
func IsSwitchTileNumberUsed(id uint8) bool {
	return id != TileJump &&
		id != TileHitEnable &&
		id != TileHitDisable &&
		id != TileAllowTeleGun &&
		id != TileAllowBlueTeleGun
}

// IsSwitchTileDelayUsed reports whether the switch tile uses its delay field.
func IsSwitchTileDelayUsed(id uint8) bool {
	return id != TileDeepFreeze && id != TileDeepUnfreeze
}

// IsValidTuneTile reports whether the tile ID is valid in the tune layer.
func IsValidTuneTile(id uint8) bool {
	return id == TileTune
}
