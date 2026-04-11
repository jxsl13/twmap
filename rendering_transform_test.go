package twmap

import "testing"

func rotateLeft4(a [4]uint8, n int) [4]uint8 {
	n %= 4
	if n < 0 {
		n += 4
	}
	if n == 0 {
		return a
	}
	return [4]uint8{a[n], a[(n+1)%4], a[(n+2)%4], a[(n+3)%4]}
}

func TestTransformTileCoordMatchesDDNetTable(t *testing.T) {
	const last = uint32(7)
	corners := [4]struct {
		x uint32
		y uint32
	}{
		{0, 0},       // TL
		{last, 0},    // TR
		{last, last}, // BR
		{0, last},    // BL
	}

	for tableFlag := uint8(0); tableFlag < 8; tableFlag++ {
		flags := tableFlag & 0x3
		if tableFlag&0x4 != 0 {
			flags |= TileFlagRotate
		}

		// DDNet reference: start with TL/TR/BR/BL texcoords.
		x := [4]uint8{0, 1, 1, 0}
		y := [4]uint8{0, 0, 1, 1}

		if flags&TileFlagVFlip != 0 {
			x = rotateLeft4(x, 2)
		}
		if flags&TileFlagHFlip != 0 {
			y = rotateLeft4(y, 2)
		}
		if flags&TileFlagRotate != 0 {
			x = rotateLeft4(x, 3)
			y = rotateLeft4(y, 3)
		}

		for i, c := range corners {
			tx, ty := transformTileCoord(tableFlag, c.x, c.y, last)
			ex := uint32(x[i]) * last
			ey := uint32(y[i]) * last
			if tx != ex || ty != ey {
				t.Fatalf("tableFlag=%d corner=%d got=(%d,%d) want=(%d,%d)", tableFlag, i, tx, ty, ex, ey)
			}
		}
	}
}
